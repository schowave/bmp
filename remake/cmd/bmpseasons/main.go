// Command bmpseasons writes real seasons of the remake from Transfermarkt
// season files (tools/transfermarkt), run in remake/ by
// mise run remake:seasons-update:
//
//	bmpseasons update -tm DIR [-from Y] [-to Y]
//	bmpseasons logo -club NAME -image FILE [-source TEXT]
//	bmpseasons squads -tm DIR [-from Y] [-to Y]
//
// update writes internal/seasons/data/YYYY-YY.json for each DIR/YYYY-YY.json
// between the seasons from and to, and the logos of new clubs, the season
// before first (see internal/seasons/update); a DIR/YYYY-YY.squads.json
// gives its clubs their squads by games. logo gives the club NAME (as
// the game shows it; several spellings separated by "|") the logo of an
// image, drawn in the colours of the logos of the original, in every
// season where it has a plain shield, and notes its source in
// internal/seasons/credits/logos.csv. squads gives the clubs of the seasons
// between from and to their players from DIR/YYYY-YY.squads.json
// (tools/transfermarkt/tm_games.py, mise run remake:seasons-games) and
// changes nothing else.
package main

import (
	"encoding/csv"
	"flag"
	"fmt"
	"image"
	"maps"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"

	"github.com/schowave/bmp/remake/internal/assets"
	"github.com/schowave/bmp/remake/internal/seasons"
	"github.com/schowave/bmp/remake/internal/seasons/update"
)

const (
	seasonsDir = "internal/seasons/data"
	origDir    = "internal/embedded/data"
)

func main() {
	if len(os.Args) >= 2 && os.Args[1] == "logo" {
		logoMain()
		return
	}
	if len(os.Args) >= 2 && os.Args[1] == "squads" {
		squadsMain()
		return
	}
	if len(os.Args) < 2 || os.Args[1] != "update" {
		usage()
	}
	fl := flag.NewFlagSet("update", flag.ExitOnError)
	tm := fl.String("tm", "", "directory of the Transfermarkt season files (YYYY-YY.json)")
	from := fl.Int("from", 0, "first season to write (its first year), 0 for every file")
	to := fl.Int("to", 9999, "last season to write")
	_ = fl.Parse(os.Args[2:])
	if fl.NArg() > 0 || *tm == "" {
		usage()
	}
	if _, err := os.Stat(seasonsDir); err != nil {
		fmt.Fprintln(os.Stderr, "bmpseasons: run it in remake/ (mise run remake:seasons-update)")
		os.Exit(2)
	}
	if err := run(*tm, *from, *to); err != nil {
		fmt.Fprintln(os.Stderr, "bmpseasons:", err)
		os.Exit(1)
	}
}

func usage() {
	fmt.Fprintln(os.Stderr, "usage: bmpseasons update -tm DIR [-from Y] [-to Y] | logo -club NAME -image FILE [-source TEXT] | squads -tm DIR [-from Y] [-to Y]")
	os.Exit(2)
}

func run(dir string, from, to int) error {
	files, err := filepath.Glob(filepath.Join(dir, "*.json"))
	if err != nil {
		return err
	}
	var todo []seasons.Season
	for _, f := range files {
		s, err := seasons.ParseID(strings.TrimSuffix(filepath.Base(f), ".json"))
		if err == nil && s.Year >= from && s.Year <= to {
			todo = append(todo, s)
		}
	}
	if len(todo) == 0 {
		return fmt.Errorf("no season files YYYY-YY.json in %s for these years", dir)
	}
	// Oldest first: each season takes over from the one before.
	for i := 1; i < len(todo); i++ {
		for j := i; j > 0 && todo[j].Year < todo[j-1].Year; j-- {
			todo[j], todo[j-1] = todo[j-1], todo[j]
		}
	}
	for _, s := range todo {
		if err := write(dir, s); err != nil {
			return fmt.Errorf("%s: %w", s.ID, err)
		}
	}
	return nil
}

// write makes season s from its file in dir and the seasons written so far.
func write(dir string, s seasons.Season) error {
	clubs, err := update.ReadClubs(filepath.Join(dir, s.ID+".json"))
	if err != nil {
		return err
	}
	known, next, err := current()
	if err != nil {
		return err
	}
	prev, err := previous(s.Year)
	if err != nil {
		return err
	}
	earlier, err := earlier(s.Year)
	if err != nil {
		return err
	}
	// The season as it was keeps the names of its clubs.
	var before *assets.Mana
	if b, err := os.ReadFile(filepath.Join(seasonsDir, s.ID+".json")); err == nil {
		if before, err = assets.DecodeClubs(b); err != nil {
			return fmt.Errorf("%s: %w", s.ID, err)
		}
	}
	foreign, err := update.ReadForeign(filepath.Join(dir, s.ID+".europe.json"))
	if err != nil {
		return err
	}
	squads, err := update.ReadSquads(filepath.Join(dir, s.ID+".squads.json"))
	if err != nil {
		return err
	}
	out, err := update.Season(update.Input{Year: s.Year, Clubs: clubs, Foreign: foreign, Previous: prev, Before: before, Earlier: earlier, Known: known, NextLogo: next, Squads: squads})
	if err != nil {
		return err
	}
	dir0, err := assets.OpenDir(os.DirFS(origDir))
	if err != nil {
		return err
	}
	pals, err := dir0.Palettes()
	if err != nil {
		return err
	}
	var shields [][]string
	for _, n := range slices.Sorted(maps.Keys(out.Logos)) {
		b, err := assets.EncodePic(out.Logos[n], &pals[0])
		if err != nil {
			return err
		}
		if err := os.WriteFile(filepath.Join(seasonsDir, "pic", strconv.Itoa(n)+".png"), b, 0o644); err != nil {
			return err
		}
		shields = append(shields, []string{strconv.Itoa(n), "shield in the colours ", "", "", ""})
	}
	// The new logos are plain shields: in the credits as such, so that
	// bmpseasons logo may replace them.
	if err := addCredits(shields...); err != nil {
		return err
	}
	data, err := assets.EncodeClubs(out.Mana)
	if err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(seasonsDir, s.ID+".json"), data, 0o644); err != nil {
		return err
	}
	fmt.Fprintf(os.Stderr, "season %s: written, %d new logos\n", s.Label(), len(out.Logos))
	return nil
}

// current reads the seasons in seasonsDir, the oldest first, and the
// number of the next free logo.
func current() ([]*assets.Mana, int, error) {
	files, err := filepath.Glob(filepath.Join(seasonsDir, "*.json"))
	if err != nil {
		return nil, 0, err
	}
	var out []*assets.Mana
	for _, f := range files { // Glob sorts: the oldest first
		b, err := os.ReadFile(f)
		if err != nil {
			return nil, 0, err
		}
		m, err := assets.DecodeClubs(b)
		if err != nil {
			return nil, 0, fmt.Errorf("%s: %w", f, err)
		}
		out = append(out, m)
	}
	pics, err := filepath.Glob(filepath.Join(seasonsDir, "pic", "*.png"))
	if err != nil {
		return nil, 0, err
	}
	next := 1000
	for _, p := range pics {
		if n, err := strconv.Atoi(strings.TrimSuffix(filepath.Base(p), ".png")); err == nil && n >= next {
			next = n + 1
		}
	}
	return out, next, nil
}

// previous is the season before year: from seasonsDir, else the clubs of
// the original.
func previous(year int) (*assets.Mana, error) {
	if b, err := os.ReadFile(filepath.Join(seasonsDir, seasons.ForYear(year-1).ID+".json")); err == nil {
		return assets.DecodeClubs(b)
	}
	dir, err := assets.OpenDir(os.DirFS(origDir))
	if err != nil {
		return nil, err
	}
	return dir.Mana()
}

// earlier are the seasons in seasonsDir before the season before year, the
// latest first.
func earlier(year int) ([]*assets.Mana, error) {
	files, err := filepath.Glob(filepath.Join(seasonsDir, "*.json"))
	if err != nil {
		return nil, err
	}
	var out []*assets.Mana
	for _, f := range slices.Backward(files) { // Glob sorts: the latest last
		s, err := seasons.ParseID(strings.TrimSuffix(filepath.Base(f), ".json"))
		if err != nil || s.Year >= year-1 {
			continue
		}
		b, err := os.ReadFile(f)
		if err != nil {
			return nil, err
		}
		m, err := assets.DecodeClubs(b)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", f, err)
		}
		out = append(out, m)
	}
	return out, nil
}

func squadsMain() {
	fl := flag.NewFlagSet("squads", flag.ExitOnError)
	tm := fl.String("tm", "", "directory of the squads files (YYYY-YY.squads.json)")
	from := fl.Int("from", 0, "first season to change (its first year)")
	to := fl.Int("to", 9999, "last season to change")
	_ = fl.Parse(os.Args[2:])
	if fl.NArg() > 0 || *tm == "" {
		usage()
	}
	if _, err := os.Stat(seasonsDir); err != nil {
		fmt.Fprintln(os.Stderr, "bmpseasons: run it in remake/")
		os.Exit(2)
	}
	if err := setSquads(*tm, *from, *to); err != nil {
		fmt.Fprintln(os.Stderr, "bmpseasons:", err)
		os.Exit(1)
	}
}

// setSquads gives the seasons between from and to the squads of their
// files in dir.
func setSquads(dir string, from, to int) error {
	files, err := filepath.Glob(filepath.Join(dir, "*.squads.json"))
	if err != nil {
		return err
	}
	done := 0
	for _, f := range files {
		s, err := seasons.ParseID(strings.TrimSuffix(filepath.Base(f), ".squads.json"))
		if err != nil || s.Year < from || s.Year > to {
			continue
		}
		squads, err := update.ReadSquads(f)
		if err != nil {
			return err
		}
		file := filepath.Join(seasonsDir, s.ID+".json")
		b, err := os.ReadFile(file)
		if err != nil {
			return err
		}
		m, err := assets.DecodeClubs(b)
		if err != nil {
			return fmt.Errorf("%s: %w", file, err)
		}
		n, err := update.SetSquads(m, squads)
		if err != nil {
			return fmt.Errorf("%s: %w", s.ID, err)
		}
		if b, err = assets.EncodeClubs(m); err != nil {
			return err
		}
		if err := os.WriteFile(file, b, 0o644); err != nil {
			return err
		}
		fmt.Fprintf(os.Stderr, "season %s: %d of %d squads changed\n", s.Label(), n, len(squads))
		done++
	}
	if done == 0 {
		return fmt.Errorf("no squads files YYYY-YY.squads.json in %s for these years", dir)
	}
	return nil
}

const creditsFile = "internal/seasons/credits/logos.csv"

func logoMain() {
	fl := flag.NewFlagSet("logo", flag.ExitOnError)
	club := fl.String("club", "", "the club as the game shows its name, e.g. \"SPVGG WEIDEN\"; spellings separated by |")
	file := fl.String("image", "", "the logo, PNG, JPEG or GIF")
	src := fl.String("source", "", "where the logo comes from, for the credits (default: the file name)")
	_ = fl.Parse(os.Args[2:])
	if fl.NArg() > 0 || *club == "" || *file == "" {
		usage()
	}
	if _, err := os.Stat(seasonsDir); err != nil {
		fmt.Fprintln(os.Stderr, "bmpseasons: run it in remake/")
		os.Exit(2)
	}
	if *src == "" {
		*src = filepath.Base(*file)
	}
	if err := setLogo(*club, *file, *src); err != nil {
		fmt.Fprintln(os.Stderr, "bmpseasons:", err)
		os.Exit(1)
	}
}

// setLogo draws the image file as a logo with the next free number and
// gives it to the German clubs named club (spellings separated by "|") in
// the seasons where they have a plain shield; a club of the same name with
// a logo is another club (FC Gütersloh of 1978 and of 2000).
func setLogo(club, file, src string) error {
	names := map[string]bool{}
	for _, n := range strings.Split(club, "|") {
		names[strings.TrimSpace(n)] = true
	}
	shields, err := shieldLogos()
	if err != nil {
		return err
	}
	f, err := os.Open(file)
	if err != nil {
		return err
	}
	img, _, err := image.Decode(f)
	_ = f.Close()
	if err != nil {
		return fmt.Errorf("%s: %w", file, err)
	}
	dir, err := assets.OpenDir(os.DirFS(origDir))
	if err != nil {
		return err
	}
	pals, err := dir.Palettes()
	if err != nil {
		return err
	}
	_, n, err := current()
	if err != nil {
		return err
	}
	files, err := filepath.Glob(filepath.Join(seasonsDir, "*.json"))
	if err != nil {
		return err
	}
	var changed []string
	for _, sf := range files {
		b, err := os.ReadFile(sf)
		if err != nil {
			return err
		}
		m, err := assets.DecodeClubs(b)
		if err != nil {
			return fmt.Errorf("%s: %w", sf, err)
		}
		hit := false
		for i := range m.Clubs[:assets.NumGermanClubs] {
			if names[assets.FromCP437(m.Clubs[i].Name)] && shields[m.Clubs[i].Logo] {
				m.Clubs[i].Logo, hit = n, true
			}
		}
		if !hit {
			continue
		}
		out, err := assets.EncodeClubs(m)
		if err != nil {
			return err
		}
		if err := os.WriteFile(sf, out, 0o644); err != nil {
			return err
		}
		changed = append(changed, strings.TrimSuffix(filepath.Base(sf), ".json"))
	}
	if len(changed) == 0 {
		return fmt.Errorf("no season has a club named %q", club)
	}
	pic, err := assets.EncodePic(update.ConvertLogo(img, &pals[0]), &pals[0])
	if err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(seasonsDir, "pic", strconv.Itoa(n)+".png"), pic, 0o644); err != nil {
		return err
	}
	if err := addCredits([]string{strconv.Itoa(n), src, "", "", ""}); err != nil {
		return err
	}
	fmt.Fprintf(os.Stderr, "logo %d for %s in %s\n", n, club, strings.Join(changed, ", "))
	return nil
}

// addCredits appends rows (logo, source, license, author, page) to the
// credits of the logos.
func addCredits(rows ...[]string) error {
	if len(rows) == 0 {
		return nil
	}
	cf, err := os.OpenFile(creditsFile, os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	w := csv.NewWriter(cf)
	if err := w.WriteAll(rows); err != nil {
		_ = cf.Close()
		return err
	}
	return cf.Close()
}

// shieldLogos are the numbers of the plain shields, from the credits.
func shieldLogos() (map[int]bool, error) {
	f, err := os.Open(creditsFile)
	if err != nil {
		return nil, err
	}
	defer func() { _ = f.Close() }()
	rows, err := csv.NewReader(f).ReadAll()
	if err != nil {
		return nil, err
	}
	out := map[int]bool{}
	for _, r := range rows[1:] {
		if n, err := strconv.Atoi(r[0]); err == nil && strings.HasPrefix(r[1], "shield") {
			out[n] = true
		}
	}
	return out, nil
}
