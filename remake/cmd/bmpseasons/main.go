// Command bmpseasons writes real seasons of the remake from Transfermarkt
// season files (tools/transfermarkt), run in remake/ by
// mise run remake:seasons-update:
//
//	bmpseasons update -tm DIR [-from Y] [-to Y]
//
// For each DIR/YYYY-YY.json between the seasons from and to it writes
// internal/seasons/data/YYYY-YY.json and the logos of new clubs, the season
// before first (see internal/seasons/update).
package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
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
	fmt.Fprintln(os.Stderr, "usage: bmpseasons update -tm DIR [-from Y] [-to Y]")
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
	out, err := update.Season(update.Input{Year: s.Year, Clubs: clubs, Previous: prev, Known: known, NextLogo: next})
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
	for n, pic := range out.Logos {
		b, err := assets.EncodePic(pic, &pals[0])
		if err != nil {
			return err
		}
		if err := os.WriteFile(filepath.Join(seasonsDir, "pic", strconv.Itoa(n)+".png"), b, 0o644); err != nil {
			return err
		}
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
