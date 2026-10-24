// Command bmpseasons builds the real seasons of the remake in two steps,
// run in remake/ (mise run remake:seasons-fetch, mise run remake:seasons):
//
//	bmpseasons fetch   fetches tables, squads, European cups and logos from
//	                   Wikipedia and Wikidata into data/ (network; the running
//	                   season always anew, the others from data/.cache)
//	bmpseasons import-tm -tm DIR
//	                   takes the Transfermarkt squads of DIR (YYYY-YY.json)
//	                   into data/seasons/*/transfermarkt.csv
//	bmpseasons build   writes internal/seasons/data from every season in
//	                   data/ (offline, deterministic)
package main

import (
	"encoding/csv"
	"errors"
	"flag"
	"fmt"
	"io/fs"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/schowave/bmp/remake/internal/seasons"
	"github.com/schowave/bmp/remake/internal/seasons/build"
	"github.com/schowave/bmp/remake/internal/seasons/source"
)

func main() {
	if len(os.Args) < 2 || strings.HasPrefix(os.Args[1], "-") {
		usage()
	}
	cmd := os.Args[1]
	fl := flag.NewFlagSet(cmd, flag.ExitOnError)
	data := fl.String("data", "data", "directory of the raw data")
	var err error
	switch cmd {
	case "fetch":
		from := fl.Int("from", 1990, "first season to fetch (its first year)")
		to := fl.Int("to", source.RunningSeason(), "last season to fetch")
		refresh := fl.Bool("refresh", false, "fetch every season anew instead of from the cache")
		delay := fl.Duration("delay", 300*time.Millisecond, "pause between two requests")
		parse(fl)
		c := &source.Client{Dir: filepath.Join(*data, ".cache"), Delay: *delay, Log: logf}
		err = fetch(c, *data, *from, *to, *refresh)
	case "import-tm":
		tm := fl.String("tm", "", "directory of the Transfermarkt season files (YYYY-YY.json)")
		parse(fl)
		err = importTM(*data, *tm)
	case "build":
		orig := fl.String("orig", "internal/embedded/data", "extract of the original")
		out := fl.String("out", "internal/seasons/data", "output directory")
		parse(fl)
		err = build.All(build.Config{Data: os.DirFS(*data), Orig: os.DirFS(*orig), Out: *out, Log: logf})
	default:
		usage()
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, "bmpseasons:", err)
		os.Exit(1)
	}
}

// parse reads the flags of the subcommand and refuses other arguments and
// a working directory other than remake/, where the default paths lie.
func parse(fl *flag.FlagSet) {
	_ = fl.Parse(os.Args[2:])
	if fl.NArg() > 0 {
		fmt.Fprintf(os.Stderr, "bmpseasons %s: unexpected %q (seasons are chosen with -from and -to)\n", fl.Name(), fl.Arg(0))
		os.Exit(2)
	}
	if _, err := os.Stat(filepath.Join("internal", "seasons")); err != nil {
		fmt.Fprintln(os.Stderr, "bmpseasons: run it in remake/ (mise run remake:seasons-fetch, mise run remake:seasons)")
		os.Exit(2)
	}
}

func usage() {
	fmt.Fprintln(os.Stderr, "usage: bmpseasons fetch [-from Y] [-to Y] [-refresh] | import-tm -tm DIR | build (flags after the subcommand)")
	os.Exit(2)
}

func logf(format string, args ...any) { fmt.Fprintf(os.Stderr, format+"\n", args...) }

// fetch fetches the seasons from..to, then the logos of every season in
// data. A season that fails is reported and skipped; the error comes at the
// end.
func fetch(c *source.Client, data string, from, to int, refresh bool) error {
	var failed []error
	for y := from; y <= to; y++ {
		logf("season %s", source.SeasonTitle(y))
		c.Refresh = refresh || y >= source.RunningSeason()
		s, err := c.FetchSeason(y)
		if err == nil {
			for _, m := range s.Missing {
				logf("  %s", m)
			}
			err = source.WriteSeason(filepath.Join(data, "seasons", seasons.ForYear(y).ID), s)
		}
		if err != nil {
			logf("  failed: %v", err)
			failed = append(failed, fmt.Errorf("season %d: %w", y, err))
		}
	}
	c.Refresh = refresh
	if err := fetchLogos(c, data); err != nil {
		failed = append(failed, err)
	}
	return errors.Join(failed...)
}

// logoIndex is data/logos/index.csv: per logo file its image and licence.
const logoIndex = "logos/index.csv"

// fetchLogos downloads a PNG rendering, 76 pixels wide, of every logo the
// seasons in data name, with its licence, into data/logos. Logos that are
// not free are left out. A logo that cannot be fetched keeps the image of
// an earlier fetch; images no season names any more are deleted.
func fetchLogos(c *source.Client, data string) error {
	dir := filepath.Join(data, "logos")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	files, err := logoFiles(data)
	if err != nil {
		return err
	}
	old := map[string][]string{}
	if rows, err := source.ReadCSV(os.DirFS(data), logoIndex); err == nil {
		for _, r := range rows {
			if _, err := os.Stat(filepath.Join(dir, r[1])); err == nil {
				old[r[0]] = r
			}
		}
	}
	rows := [][]string{{"file", "image", "license", "artist", "page"}}
	keep := map[string]bool{"index.csv": true}
	for _, f := range files {
		row, err := fetchLogo(c, dir, f)
		if err != nil {
			logf("  logo %s: %v", f, err)
			if o, ok := old[f]; ok {
				row = o
			}
		}
		if row != nil {
			rows = append(rows, row)
			keep[row[1]] = true
		}
	}
	if err := writeCSV(filepath.Join(dir, "index.csv"), rows); err != nil {
		return err
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		return err
	}
	for _, e := range entries {
		if !keep[e.Name()] {
			if err := os.Remove(filepath.Join(dir, e.Name())); err != nil {
				return err
			}
		}
	}
	return nil
}

// fetchLogo downloads logo file f into dir and returns its index row; a
// logo that is not free gives no row.
func fetchLogo(c *source.Client, dir, f string) ([]string, error) {
	info, err := c.File(f, 76)
	if err != nil {
		return nil, err
	}
	if info.NonFree || info.Thumb == "" {
		logf("  logo %s: not free (%s)", f, info.License)
		return nil, nil
	}
	u, err := url.Parse(info.Thumb)
	if err != nil {
		return nil, err
	}
	b, err := c.Download(info.Thumb)
	if err != nil {
		return nil, err
	}
	img := build.LogoFileName(f) + strings.ToLower(path.Ext(u.Path))
	if err := os.WriteFile(filepath.Join(dir, img), b, 0o644); err != nil {
		return nil, err
	}
	return []string{f, img, info.License, info.Artist, info.Page}, nil
}

// logoFiles are the logo files the seasons in data name, sorted.
func logoFiles(data string) ([]string, error) {
	all, err := fs.Glob(os.DirFS(data), "seasons/*/logos.csv")
	if err != nil {
		return nil, err
	}
	set := map[string]bool{}
	for _, p := range all {
		rows, err := source.ReadCSV(os.DirFS(data), p)
		if err != nil {
			return nil, err
		}
		for _, r := range rows {
			if len(r) > 1 && r[1] != "" {
				set[r[1]] = true
			}
		}
	}
	files := make([]string, 0, len(set))
	for f := range set {
		files = append(files, f)
	}
	sort.Strings(files)
	return files, nil
}

func writeCSV(file string, rows [][]string) error {
	fo, err := os.Create(file)
	if err != nil {
		return err
	}
	w := csv.NewWriter(fo)
	if err := w.WriteAll(rows); err != nil {
		_ = fo.Close()
		return err
	}
	return fo.Close()
}

// importTM matches the clubs of the Transfermarkt season files in dir with
// the clubs of the tables in data and writes their squads next to them.
func importTM(data, dir string) error {
	if dir == "" {
		return errors.New("import-tm: -tm DIR missing")
	}
	files, err := filepath.Glob(filepath.Join(dir, "*.json"))
	if err != nil {
		return err
	}
	for _, f := range files {
		sea, err := seasons.ParseID(strings.TrimSuffix(filepath.Base(f), ".json"))
		if err != nil {
			continue
		}
		sdir := filepath.Join(data, "seasons", sea.ID)
		s, err := source.ReadSeason(os.DirFS(sdir), sea.Year)
		if err != nil {
			return fmt.Errorf("%s: %w", sea.ID, err)
		}
		clubs, err := source.ReadTM(f)
		if err != nil {
			return fmt.Errorf("%s: %w", f, err)
		}
		squads, names := map[string][]source.Player{}, map[string]string{}
		match := source.MatchTM(clubs, s.Tables)
		unmatched := 0
		for i, c := range clubs {
			ps := c.Squad()
			if len(ps) == 0 || squads[match[i]] != nil {
				continue // no players, or the same club twice in the file
			}
			if match[i] == "" {
				unmatched++
				logf("  %s: no club for %s (%s)", sea.ID, c.Name, c.League)
				continue
			}
			squads[match[i]], names[match[i]] = ps, c.Name
		}
		if err := source.WriteTM(sdir, squads, names); err != nil {
			return err
		}
		logf("season %s: %d clubs, %d without a match", sea.Label(), len(squads), unmatched)
	}
	return nil
}
