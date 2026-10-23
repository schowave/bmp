// Package build writes the real seasons of the remake (internal/seasons/
// data) from the raw data that cmd/bmpseasons fetches into remake/data. It
// works offline and gives the same files for the same data.
package build

import (
	"encoding/csv"
	"errors"
	"fmt"
	"image"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"github.com/schowave/bmp/remake/internal/assets"
	"github.com/schowave/bmp/remake/internal/seasons/source"
)

// FirstLogo is the number of the first logo of the seasons; the pictures
// of the original end at 229.
const FirstLogo = 1000

// Per club and position the players of a squad: goalkeepers, defenders,
// midfielders, forwards (D:0x9432).
var rosterSizes = [4]int{2, 6, 7, 5}

// curated is a line of remake/data/clubs.csv: a club of the original with
// its logo, and optionally a name that replaces the one of the tables.
type curated struct {
	name string
	logo int
}

func readCSV(fsys fs.FS, p string) ([][]string, error) {
	f, err := fsys.Open(p)
	if err != nil {
		return nil, err
	}
	defer func() { _ = f.Close() }()
	rows, err := csv.NewReader(f).ReadAll()
	if err != nil {
		return nil, fmt.Errorf("%s: %w", p, err)
	}
	if len(rows) == 0 {
		return nil, nil
	}
	return rows[1:], nil
}

// All writes the seasons from..to that data holds into out. orig is the
// extract of the original (palettes and foreign clubs).
func All(data, orig fs.FS, out string, from, to int, logf func(string, ...any)) error {
	dir, err := assets.OpenDir(orig)
	if err != nil {
		return err
	}
	pals, err := dir.Palettes()
	if err != nil {
		return err
	}
	origMana, err := dir.Mana()
	if err != nil {
		return err
	}
	cur := map[string]curated{}
	rows, err := readCSV(data, "clubs.csv")
	if err != nil {
		return err
	}
	for _, r := range rows {
		logo, _ := strconv.Atoi(r[2])
		cur[r[0]] = curated{name: r[1], logo: logo}
	}
	images := map[string]string{} // logo file -> image in data/logos
	if rows, err = readCSV(data, "logos/index.csv"); err != nil && !errors.Is(err, fs.ErrNotExist) {
		return err
	}
	for _, r := range rows {
		images[r[0]] = r[1]
	}

	type built struct {
		id    string
		clubs []assets.ClubRecord
		logos []string // per German club: logo key, "" for a logo of the original
	}
	var all []built
	keys := map[string]bool{}
	for y := from; y <= to; y++ {
		id := fmt.Sprintf("%d-%02d", y, (y+1)%100)
		sub, err := fs.Sub(data, path.Join("seasons", id))
		if err != nil {
			return err
		}
		s, err := source.ReadSeason(sub, y)
		if errors.Is(err, fs.ErrNotExist) {
			continue
		}
		if err != nil {
			return fmt.Errorf("season %s: %w", id, err)
		}
		clubs, logos, err := buildSeason(s, cur, images, origMana, logf)
		if err != nil {
			return fmt.Errorf("season %s: %w", id, err)
		}
		for _, k := range logos {
			if k != "" {
				keys[k] = true
			}
		}
		all = append(all, built{id, clubs, logos})
	}

	// One number per logo, in the order of the keys.
	sorted := make([]string, 0, len(keys))
	for k := range keys {
		sorted = append(sorted, k)
	}
	sort.Strings(sorted)
	number := map[string]int{}
	if err := clean(out); err != nil {
		return err
	}
	for i, k := range sorted {
		n := FirstLogo + i
		number[k] = n
		pic, err := logoPic(data, k, &pals[0])
		if err != nil {
			return err
		}
		b, err := assets.EncodePic(pic, &pals[0])
		if err != nil {
			return err
		}
		if err := os.WriteFile(filepath.Join(out, "pic", strconv.Itoa(n)+".png"), b, 0o644); err != nil {
			return err
		}
	}
	for _, b := range all {
		for i, k := range b.logos {
			if k != "" {
				b.clubs[i].Logo = number[k]
			}
		}
		data, err := assets.EncodeClubs(&assets.Mana{Clubs: b.clubs})
		if err != nil {
			return fmt.Errorf("season %s: %w", b.id, err)
		}
		if err := os.WriteFile(filepath.Join(out, b.id+".json"), data, 0o644); err != nil {
			return err
		}
	}
	logf("%d seasons, %d logos", len(all), len(sorted))
	return nil
}

// clean empties out except its README.
func clean(out string) error {
	for _, pat := range []string{"*.json", "pic/*.png"} {
		old, err := filepath.Glob(filepath.Join(out, pat))
		if err != nil {
			return err
		}
		for _, f := range old {
			if err := os.Remove(f); err != nil {
				return err
			}
		}
	}
	return os.MkdirAll(filepath.Join(out, "pic"), 0o755)
}

// logoPic draws the logo of key: "file:<image in data/logos>" or
// "crest:<colours>".
func logoPic(data fs.FS, key string, pal *assets.Palette) (*assets.Pic, error) {
	kind, v, _ := strings.Cut(key, ":")
	if kind == "crest" {
		return crest(v), nil
	}
	f, err := data.Open(path.Join("logos", v))
	if err != nil {
		return nil, err
	}
	defer func() { _ = f.Close() }()
	img, _, err := image.Decode(f)
	if err != nil {
		return nil, fmt.Errorf("logo %s: %w", v, err)
	}
	return convertLogo(img, pal), nil
}

// buildSeason makes the 200 clubs of a season: the 64 German ones on their
// places, then the foreign ones. logos holds per German club the key of its
// logo, or "" when it keeps a logo of the original.
func buildSeason(s *source.Season, cur map[string]curated, images map[string]string, orig *assets.Mana,
	logf func(string, ...any)) ([]assets.ClubRecord, []string, error) {
	places, err := placeClubs(s.Year, s.Tables)
	if err != nil {
		return nil, nil, err
	}
	str := strengths(places)
	clubs := make([]assets.ClubRecord, assets.NumClubs)
	logos := make([]string, numGerman)
	names := map[string]bool{}
	for i, p := range places {
		c := &clubs[i]
		name := clubName(p.row.Label)
		if cu, ok := cur[p.row.Club]; ok && cu.name != "" {
			name = cu.name
		}
		if names[name] {
			logf("  %d: two clubs named %s", s.Year, name)
		}
		names[name] = true
		var err error
		if c.Name, err = assets.ToCP437(name); err != nil {
			return nil, nil, err
		}
		c.Strength = str[p.row.Club]
		switch cu, ok := cur[p.row.Club]; {
		case ok && cu.logo > 0:
			c.Logo = cu.logo
		case images[s.Logos[p.article]] != "":
			logos[i] = "file:" + images[s.Logos[p.article]]
		default:
			logos[i] = "crest:" + strings.ToLower(s.Colors[p.article])
		}
		players := pickPlayers(s.Squads[p.row.Club])
		for _, n := range players {
			cp, err := assets.ToCP437(n)
			if err != nil {
				return nil, nil, err
			}
			c.Players = append(c.Players, cp)
		}
	}
	foreign := pickForeign(s.Europe, orig, names)
	for i, f := range foreign {
		clubs[numGerman+i] = f
	}
	return clubs, logos, nil
}

// pickPlayers chooses the 20 player names of a club: per position the
// players with the most games, else those who joined last, in the slots of MANA.DAT (2 goalkeepers,
// 6 defenders, 7 midfielders, 5 forwards). Missing ones stay empty; the
// game gives those slots random names.
func pickPlayers(ps []source.Player) []string {
	var byPos [4][]source.Player
	seen := map[string]bool{}
	for _, p := range ps {
		if p.Pos == source.PosNone || seen[p.Name] {
			continue
		}
		seen[p.Name] = true
		byPos[p.Pos-1] = append(byPos[p.Pos-1], p)
	}
	var out []string
	surnames := map[string]int{}
	var chosen []source.Player
	for k, n := range rosterSizes {
		list := byPos[k]
		sort.SliceStable(list, func(a, b int) bool {
			if list[a].Games != list[b].Games {
				return list[a].Games > list[b].Games
			}
			if list[a].Since != list[b].Since {
				return list[a].Since > list[b].Since
			}
			return list[a].Name < list[b].Name
		})
		for i := 0; i < n; i++ {
			if i < len(list) {
				chosen = append(chosen, list[i])
				surnames[playerName(list[i].Surname)]++
			} else {
				chosen = append(chosen, source.Player{})
			}
		}
	}
	for _, p := range chosen {
		name := playerName(p.Surname)
		if name != "" && surnames[name] > 1 {
			// Two players of one name: the initial tells them apart.
			if f := []rune(upper(p.Name)); len(f) > 0 {
				name = cut(string(f[0])+"."+name, playerNameMax)
			}
		}
		out = append(out, name)
	}
	return out
}

// foreignStrength is the strength range per European cup: Champions
// League, Cup Winners' Cup, UEFA Cup/Europa League, Conference League,
// Intertoto Cup.
var foreignStrength = [5][2]int{{72, 99}, {58, 86}, {48, 84}, {34, 66}, {34, 60}}

// pickForeign chooses the 136 foreign clubs: the clubs of the European
// cups, the highest cup and the most mentioned first; the foreign clubs of
// the original fill what is left.
func pickForeign(europe []source.EuroClub, orig *assets.Mana, german map[string]bool) []assets.ClubRecord {
	n := assets.NumClubs - numGerman
	var out []assets.ClubRecord
	names := map[string]bool{}
	perCup := map[int]int{}
	for _, e := range europe {
		perCup[e.Cup]++
	}
	rank := map[int]int{}
	for _, e := range europe {
		if len(out) == n {
			break
		}
		name := clubName(e.Label)
		if name == "" || names[name] || german[name] {
			continue
		}
		cp, err := assets.ToCP437(name)
		if err != nil {
			continue
		}
		names[name] = true
		lo, hi := foreignStrength[e.Cup][0], foreignStrength[e.Cup][1]
		p := 1.0
		if perCup[e.Cup] > 1 {
			p = 1 - float64(rank[e.Cup])/float64(perCup[e.Cup]-1)
		}
		rank[e.Cup]++
		v := lo + int(p*float64(hi-lo)+0.5)
		out = append(out, assets.ClubRecord{Name: cp, Strength: [3]int{v, v, v}})
	}
	for _, c := range orig.Clubs[numGerman:] {
		if len(out) == n {
			break
		}
		if !names[c.Name] {
			names[c.Name] = true
			out = append(out, assets.ClubRecord{Name: c.Name, Strength: c.Strength})
		}
	}
	return out
}
