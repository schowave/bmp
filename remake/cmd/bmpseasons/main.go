// Command bmpseasons builds the real seasons of the remake in two steps:
//
//	bmpseasons fetch   fetches tables, squads, European cups and logos from
//	                   Wikipedia and Wikidata into remake/data (network)
//	bmpseasons build   writes internal/seasons/data from remake/data
//	                   (offline, deterministic)
package main

import (
	"encoding/csv"
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

	"github.com/schowave/bmp/remake/internal/seasons/build"
	"github.com/schowave/bmp/remake/internal/seasons/source"
)

func main() {
	if len(os.Args) < 2 {
		usage()
	}
	cmd := os.Args[1]
	fl := flag.NewFlagSet(cmd, flag.ExitOnError)
	data := fl.String("data", "data", "directory of the raw data")
	from := fl.Int("from", 1990, "first season (its first year)")
	to := fl.Int("to", 2026, "last season")
	cache := fl.String("cache", "data/.cache", "cache of the fetched pages (fetch)")
	delay := fl.Duration("delay", 300*time.Millisecond, "pause between two requests (fetch)")
	orig := fl.String("orig", "internal/embedded/data", "extract of the original (build)")
	out := fl.String("out", "internal/seasons/data", "output directory (build)")
	_ = fl.Parse(os.Args[2:])
	var err error
	switch cmd {
	case "fetch":
		c := &source.Client{Dir: *cache, Delay: *delay, Log: logf}
		err = fetch(c, *data, *from, *to)
	case "build":
		err = build.All(os.DirFS(*data), os.DirFS(*orig), *out, *from, *to, logf)
	default:
		usage()
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, "bmpseasons:", err)
		os.Exit(1)
	}
}

func usage() {
	fmt.Fprintln(os.Stderr, "usage: bmpseasons fetch|build [flags]")
	os.Exit(2)
}

func logf(format string, args ...any) { fmt.Fprintf(os.Stderr, format+"\n", args...) }

// seasonDir is the directory of the raw data of the season of year.
func seasonDir(data string, year int) string {
	return filepath.Join(data, "seasons", fmt.Sprintf("%d-%02d", year, (year+1)%100))
}

func fetch(c *source.Client, data string, from, to int) error {
	files := map[string]bool{}
	for y := from; y <= to; y++ {
		logf("season %s", source.SeasonTitle(y))
		s, err := c.FetchSeason(y)
		if err != nil {
			return fmt.Errorf("season %d: %w", y, err)
		}
		for _, m := range s.Missing {
			logf("  %s", m)
		}
		if err := source.WriteSeason(seasonDir(data, y), s); err != nil {
			return err
		}
		for _, f := range s.Logos {
			if f != "" {
				files[f] = true
			}
		}
	}
	// Logos of all seasons fetched so far.
	all, err := fs.Glob(os.DirFS(data), "seasons/*/logos.csv")
	if err != nil {
		return err
	}
	for _, p := range all {
		rows, err := readRows(filepath.Join(data, p))
		if err != nil {
			return err
		}
		for _, r := range rows {
			if len(r) > 1 && r[1] != "" {
				files[r[1]] = true
			}
		}
	}
	return fetchLogos(c, filepath.Join(data, "logos"), files)
}

func readRows(p string) ([][]string, error) {
	f, err := os.Open(p)
	if err != nil {
		return nil, err
	}
	defer func() { _ = f.Close() }()
	rows, err := csv.NewReader(f).ReadAll()
	if err != nil || len(rows) == 0 {
		return nil, err
	}
	return rows[1:], nil
}

// fetchLogos downloads a PNG rendering of each logo file, 76 pixels wide,
// with its licence, into dir. Files that are not free are left out.
func fetchLogos(c *source.Client, dir string, files map[string]bool) error {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	names := make([]string, 0, len(files))
	for f := range files {
		names = append(names, f)
	}
	sort.Strings(names)
	rows := [][]string{{"file", "image", "license", "artist", "page"}}
	for _, f := range names {
		info, err := c.File(f, 76)
		if err != nil {
			logf("  logo %s: %v", f, err)
			continue
		}
		if info.NonFree || info.Thumb == "" {
			logf("  logo %s: not free (%s)", f, info.License)
			continue
		}
		b, err := c.Download(info.Thumb)
		if err != nil {
			logf("  logo %s: %v", f, err)
			continue
		}
		u, err := url.Parse(info.Thumb)
		if err != nil {
			return err
		}
		img := build.LogoFileName(f) + strings.ToLower(path.Ext(u.Path))
		if err := os.WriteFile(filepath.Join(dir, img), b, 0o644); err != nil {
			return err
		}
		rows = append(rows, []string{f, img, info.License, info.Artist, info.Page})
	}
	fo, err := os.Create(filepath.Join(dir, "index.csv"))
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
