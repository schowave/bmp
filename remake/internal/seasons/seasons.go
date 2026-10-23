// Package seasons holds the clubs of real seasons from 1990/91 on, built
// into every binary: for each season a club database in the format of
// clubs.json, and the logos of the clubs the original does not have.
// cmd/bmpseasons writes them from remake/data (mise run remake:seasons).
package seasons

import (
	"embed"
	"fmt"
	"io/fs"
	"slices"
	"strconv"
	"strings"

	"github.com/schowave/bmp/remake/internal/assets"
)

//go:embed data
var data embed.FS

var files = mustSub(data)

func mustSub(f embed.FS) fs.FS {
	sub, err := fs.Sub(f, "data")
	if err != nil {
		panic(err)
	}
	return sub
}

// Season is a real season, named by its years ("1995-96").
type Season struct {
	ID   string
	Year int // the year it starts in
}

// Label is the season as the game shows it: "1995/96".
func (s Season) Label() string { return strings.Replace(s.ID, "-", "/", 1) }

// ParseID reads a season id "YYYY-YY".
func ParseID(id string) (Season, error) {
	y, rest, ok := strings.Cut(id, "-")
	year, err := strconv.Atoi(y)
	if !ok || err != nil || len(y) != 4 || rest != fmt.Sprintf("%02d", (year+1)%100) {
		return Season{}, fmt.Errorf("season %q: want YYYY-YY", id)
	}
	return Season{ID: id, Year: year}, nil
}

// List returns the built-in seasons, the oldest first.
func List() []Season { return list(files) }

func list(fsys fs.FS) []Season {
	names, _ := fs.Glob(fsys, "*.json")
	var out []Season
	for _, n := range names {
		if s, err := ParseID(strings.TrimSuffix(n, ".json")); err == nil {
			out = append(out, s)
		}
	}
	slices.SortFunc(out, func(a, b Season) int { return a.Year - b.Year })
	return out
}

// Mana returns the clubs of season id.
func Mana(id string) (*assets.Mana, error) { return mana(files, id) }

func mana(fsys fs.FS, id string) (*assets.Mana, error) {
	if _, err := ParseID(id); err != nil {
		return nil, err
	}
	b, err := fs.ReadFile(fsys, id+".json")
	if err != nil {
		return nil, err
	}
	m, err := assets.DecodeClubs(b)
	if err != nil {
		return nil, fmt.Errorf("season %s: %w", id, err)
	}
	return m, nil
}

// Files returns the built-in files: the seasons and pic/.
func Files() fs.FS { return files }

// Pics returns the logos of the clubs of the seasons, as pic/N.png for
// assets.Dir.AddPics.
func Pics() fs.FS { return files }
