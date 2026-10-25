// Package update makes a real season of the remake (internal/seasons/data)
// from the Transfermarkt file of the season (tools/transfermarkt) and the
// season before: the clubs of the tables on the places of the game with
// strengths from their table, their squads by market value; clubs the file
// lacks, the logos and the foreign clubs come from the season before.
package update

import (
	"cmp"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"maps"
	"math"
	"os"
	"slices"
	"sort"
	"strconv"
	"strings"

	"github.com/schowave/bmp/remake/internal/assets"
)

// Club is a club of a Transfermarkt season file (tools/transfermarkt):
// its league, its place in the table and its players by position, the most
// valuable first, surnames in capitals.
type Club struct {
	Name    string `json:"name"`
	League  string `json:"league"` // Transfermarkt: L1, L2, L3, or a group of the third or fourth tier
	Tier    int    `json:"tier"`   // 1..4; 0 in files before the Regionalliga was fetched
	Rank    int    `json:"rank"`
	Games   int    `json:"games"`
	Goals   string `json:"goals"` // "59:23"
	Points  int    `json:"points"`
	Players Squad  `json:"players"`
}

// Squad are the players of a club by position, the first the most
// valuable (or, in a squads file, the most often played), surnames in
// capitals.
type Squad struct {
	Goalkeepers []string `json:"goalkeepers"`
	Defenders   []string `json:"defenders"`
	Midfielders []string `json:"midfielders"`
	Forwards    []string `json:"forwards"`
}

// ReadClubs reads a Transfermarkt season file.
func ReadClubs(file string) ([]Club, error) {
	b, err := os.ReadFile(file)
	if err != nil {
		return nil, err
	}
	var clubs []Club
	if err := json.Unmarshal(b, &clubs); err != nil {
		return nil, fmt.Errorf("%s: %w", file, err)
	}
	for _, c := range clubs {
		if c.Rank == 0 {
			return nil, fmt.Errorf("%s: %s has no table place; fetch the season again (files before the table was kept lack it)", file, c.Name)
		}
	}
	return clubs, nil
}

// tier is the tier of a club of a Transfermarkt file: as the file says,
// else by its league: 1, 2, else 3 (the 3. Liga or a group of the Oberliga
// or Regionalliga).
func tier(c Club) int {
	if c.Tier > 0 {
		return c.Tier
	}
	switch c.League {
	case "L1":
		return 1
	case "L2":
		return 2
	}
	return 3
}

// goals reads "59:23".
func goals(s string) (forGoals, against int) {
	a, b, _ := strings.Cut(s, ":")
	forGoals, _ = strconv.Atoi(strings.TrimSpace(a))
	against, _ = strconv.Atoi(strings.TrimSpace(b))
	return forGoals, against
}

// Foreign is a foreign club of the European cups of a season
// (YYYY-YY.europe.json): the main rounds of CL, EL, UCOL first, then their
// qualifications, each in the order of Transfermarkt.
type Foreign struct {
	Name    string `json:"name"`
	Country string `json:"country"`
	Cup     string `json:"cup"`
}

// ReadForeign reads the European cup file of a season; none when there is
// no file.
func ReadForeign(file string) ([]Foreign, error) {
	b, err := os.ReadFile(file)
	if errors.Is(err, fs.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var out []Foreign
	if err := json.Unmarshal(b, &out); err != nil {
		return nil, fmt.Errorf("%s: %w", file, err)
	}
	return out, nil
}

// Input is what a season is made of.
type Input struct {
	Year     int
	Clubs    []Club           // the Transfermarkt file of the season
	Foreign  []Foreign        // the foreign clubs of its European cups, if fetched
	Previous *assets.Mana     // the season before (or the clubs of the original)
	Before   *assets.Mana     // the season as it was before the update, if any
	Earlier  []*assets.Mana   // the seasons before Previous, the latest first
	Known    []*assets.Mana   // every season, for the logos of clubs by name
	NextLogo int              // the number of the next new logo
	Squads   map[string]Squad // squads by games for clubs by name (tm_games.py), if fetched
}

// Output is a season and the logos it adds.
type Output struct {
	Mana  *assets.Mana
	Logos map[int]*assets.Pic // new logos by number
}

// Season makes the season of in.
func Season(in Input) (*Output, error) {
	// A club keeps the name it has in the season as it was, else in the
	// season before, else in the latest season before that has it, else
	// in any season, so that it keeps its name and logo. The earlier
	// seasons match a name only strictly.
	var tmNames []string
	logos := knownLogos(in.Known)
	for _, c := range in.Clubs {
		tmNames = append(tmNames, clubName(c.Name))
	}
	bySeasons := map[string]string{} // name of the file -> name in the seasons
	taken := map[string]bool{}
	others := slices.Clone(tmNames)
	for i, m := range append([]*assets.Mana{in.Before, in.Previous}, in.Earlier...) {
		if m == nil {
			continue
		}
		strict := i > 1 // an earlier season: only the same club
		var names []string
		for _, c := range m.Clubs[:numGerman] {
			if n := assets.FromCP437(c.Name); !taken[n] {
				names = append(names, n)
			}
		}
		for n, k := range matchNames(others, names, strict) {
			bySeasons[n], taken[k] = k, true
		}
		others = slices.DeleteFunc(others, func(n string) bool { return bySeasons[n] != "" })
	}
	var knownNames []string
	for n := range logos {
		if !taken[n] {
			knownNames = append(knownNames, n)
		}
	}
	sort.Strings(knownNames)
	byKnown := matchNames(others, knownNames, false)
	// The places and strengths of the German clubs in the season before,
	// for clubs that have not played yet.
	prev := map[string]assets.ClubRecord{}
	prevLeague := map[string]int{}
	var prevNames []string
	for i, c := range in.Previous.Clubs[:numGerman] {
		name := assets.FromCP437(c.Name)
		prev[name], prevLeague[name] = c, leagueOf(i)
		prevNames = append(prevNames, name)
	}
	// The club of the season before that a club of the file is, also when
	// the season has kept another name of it.
	asPrev := matchNames(tmNames, prevNames, false)
	inFile := map[string]bool{} // clubs of the season before
	var rows []row
	tm := map[string]Club{}
	for _, c := range in.Clubs {
		name := clubName(c.Name)
		prevName, ok := asPrev[name]
		if !ok {
			prevName = name
		}
		if m, ok := bySeasons[name]; ok {
			name = m
		} else if m, ok := byKnown[name]; ok {
			name = m
		}
		if _, ok := tm[name]; ok || name == "" {
			continue // the same club twice
		}
		tm[name], inFile[prevName] = c, true
		gf, ga := goals(c.Goals)
		r := row{name: name, tier: tier(c), group: c.League, rank: c.Rank,
			games: c.Games, goalsFor: gf, goalsAgainst: ga, points: c.Points, prevLeague: newLeague,
			second: isSecond(c.Name) || isSecond(name)}
		if p, ok := prev[prevName]; ok {
			r.prevLeague, r.prevStrength = prevLeague[prevName], p.Strength
		}
		rows = append(rows, r)
	}
	// The German clubs of the season before that the file lacks fill what
	// is left, in their order there.
	for i, c := range in.Previous.Clubs[:numGerman] {
		name := assets.FromCP437(c.Name)
		if _, ok := tm[name]; !ok && !inFile[name] {
			rows = append(rows, row{name: name, tier: fillTier, rank: i, prevLeague: leagueOf(i), prevStrength: c.Strength, second: isSecond(name)})
		}
	}
	places, err := placeClubs(rows, in.Year)
	if err != nil {
		return nil, err
	}
	str := strengths(places)
	out := &Output{Mana: &assets.Mana{Clubs: make([]assets.ClubRecord, assets.NumClubs)}, Logos: map[int]*assets.Pic{}}
	next := in.NextLogo
	for i, p := range places {
		c := &out.Mana.Clubs[i]
		name, err := assets.ToCP437(p.row.name)
		if err != nil {
			return nil, err
		}
		c.Name, c.Strength = name, str[p.row.name]
		if t, ok := tm[p.row.name]; ok {
			if c.Players, err = squadRecord(t.Players); err != nil {
				return nil, fmt.Errorf("%s: %w", t.Name, err)
			}
		} else {
			c.Players = prev[p.row.name].Players
		}
		logo, ok := logos[p.row.name]
		if !ok {
			// A second team without a logo of its own wears its first team's.
			if f := firstTeam(p.row.name, slices.Sorted(maps.Keys(logos))); f != "" {
				logo, ok = logos[f], true
			}
		}
		if !ok {
			logo = next
			out.Logos[next] = crest("")
			next++
		}
		c.Logo, logos[p.row.name] = logo, logo
	}
	// Seasons without market values: the squads by games instead.
	if _, err := SetSquads(out.Mana, in.Squads); err != nil {
		return nil, err
	}
	copy(out.Mana.Clubs[numGerman:], foreignClubs(in.Foreign, in.Previous, out.Mana.Clubs[:numGerman]))
	return out, nil
}

// foreignStrength is the strength range of the clubs of a European cup.
var foreignStrength = map[string][2]int{"CL": {72, 99}, "EL": {52, 84}, "UCOL": {40, 70},
	"CLQ": {50, 76}, "ELQ": {38, 64}, "ECLQ": {34, 56}}

// foreignClubs are the 136 foreign clubs of a season: those of its
// European cups, by cup and in the order of Transfermarkt, the best of a
// cup the strongest; the foreign clubs of the season before fill the rest
// (and all of them without a European cup file).
func foreignClubs(cups []Foreign, prev *assets.Mana, german []assets.ClubRecord) []assets.ClubRecord {
	n := assets.NumClubs - numGerman
	taken := map[string]bool{}
	for _, c := range german {
		taken[c.Name] = true
	}
	perCup := map[string]int{}
	for _, f := range cups {
		perCup[f.Cup]++
	}
	rank := map[string]int{}
	var out []assets.ClubRecord
	for _, f := range cups {
		r, ok := foreignStrength[f.Cup]
		name, err := assets.ToCP437(clubName(f.Name))
		if !ok || err != nil || name == "" || taken[name] || len(out) == n {
			continue
		}
		taken[name] = true
		v := spread(rank[f.Cup], perCup[f.Cup], r[0], r[1])
		rank[f.Cup]++
		out = append(out, assets.ClubRecord{Name: name, Strength: [3]int{v, v, v}})
	}
	for _, c := range prev.Clubs[numGerman:] {
		if len(out) == n {
			break
		}
		if !taken[c.Name] {
			taken[c.Name] = true
			out = append(out, c)
		}
	}
	return out
}

// knownLogos maps the names of the German clubs of every season to their
// logos; a later season wins.
func knownLogos(seasons []*assets.Mana) map[string]int {
	out := map[string]int{}
	for _, m := range seasons {
		for _, c := range m.Clubs[:numGerman] {
			if c.Logo > 0 {
				out[assets.FromCP437(c.Name)] = c.Logo
			}
		}
	}
	return out
}

// Per club and position the players of a squad: goalkeepers, defenders,
// midfielders, forwards (D:0x9432).
var rosterSizes = [4]int{2, 6, 7, 5}

// pickPlayers chooses the 20 player names of a club in the slots of
// MANA.DAT (2 goalkeepers, 6 defenders, 7 midfielders, 5 forwards): per
// position the first of the file, the most valuable. Missing ones stay
// empty; the game gives those slots random names.
func pickPlayers(s Squad) []string {
	var out []string
	for k, names := range [4][]string{s.Goalkeepers, s.Defenders, s.Midfielders, s.Forwards} {
		for i := 0; i < rosterSizes[k]; i++ {
			n := ""
			if i < len(names) {
				n = playerName(names[i])
			}
			out = append(out, n)
		}
	}
	return out
}

// squadRecord are the players of s as MANA.DAT keeps them.
func squadRecord(s Squad) ([]string, error) {
	var out []string
	for _, n := range pickPlayers(s) {
		cp, err := assets.ToCP437(n)
		if err != nil {
			return nil, err
		}
		out = append(out, cp)
	}
	return out, nil
}

// ReadSquads reads a squads file of tools/transfermarkt/tm_games.py: per
// German club of a season, by its name in the season, its squad; nil
// without the file.
func ReadSquads(file string) (map[string]Squad, error) {
	b, err := os.ReadFile(file)
	if errors.Is(err, fs.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var out map[string]Squad
	if err := json.Unmarshal(b, &out); err != nil {
		return nil, fmt.Errorf("%s: %w", file, err)
	}
	return out, nil
}

// SetSquads gives the German clubs of m that squads names their players
// from it and leaves everything else of m as it is; it returns the number
// of clubs changed.
func SetSquads(m *assets.Mana, squads map[string]Squad) (int, error) {
	n := 0
	for i := range m.Clubs[:numGerman] {
		c := &m.Clubs[i]
		s, ok := squads[assets.FromCP437(c.Name)]
		if !ok {
			continue
		}
		players, err := squadRecord(s)
		if err != nil {
			return n, fmt.Errorf("%s: %w", assets.FromCP437(c.Name), err)
		}
		if !slices.Equal(players, c.Players) {
			c.Players = players
			n++
		}
	}
	return n, nil
}

// Places of the game (internal/game: TeamsPerLeague, NumGerman).
const (
	league1   = 18
	league2   = 20
	league3   = 20
	outside   = 6
	numGerman = league1 + league2 + league3 + outside
	fillTier  = 5 // a club of the season before that the file lacks
	newLeague = 4 // the league of a club that was not in the season before
)

// secondTeamsFrom is the first year whose seasons take the second teams of
// professional clubs like any club; before, they only fill places that stay
// empty without them (README: Deviations).
const secondTeamsFrom = 2008

// leagueOf is the league of place i of a season: 0..2, 3 = outside.
func leagueOf(i int) int {
	switch {
	case i < league1:
		return 0
	case i < league1+league2:
		return 1
	case i < league1+league2+league3:
		return 2
	}
	return 3
}

// Strength ranges of the original per place, measured in its clubs.json.
var strengthRange = [4][2]int{{73, 95}, {46, 74}, {20, 49}, {26, 44}}

// row is a club of a table.
type row struct {
	name                   string
	tier                   int
	group                  string
	rank                   int
	games                  int
	goalsFor, goalsAgainst int
	points                 int
	prevLeague             int    // its league in the season before, newLeague if none
	prevStrength           [3]int // its strengths there
	second                 bool   // a second team, also when its name is cut ("…FÜRTH I")
}

// unplayed compares two clubs without games by the season before, the
// better first: by the league they played in, then by their strength k
// there (-1: the sum of the three); clubs new to the game come last.
func unplayed(x, y row, k int) int {
	if x.prevLeague != y.prevLeague {
		return x.prevLeague - y.prevLeague
	}
	sx, sy := x.prevStrength[0]+x.prevStrength[1]+x.prevStrength[2], y.prevStrength[0]+y.prevStrength[1]+y.prevStrength[2]
	if k >= 0 {
		sx, sy = x.prevStrength[k], y.prevStrength[k]
	}
	return sy - sx
}

// comparePerGame compares a per ga games with b per gb games by
// multiplying across, so that no float decides; a club without games
// counts 0.
func comparePerGame(a, ga, b, gb int) int {
	if ga == 0 {
		a, ga = 0, 1
	}
	if gb == 0 {
		b, gb = 0, 1
	}
	return cmp.Compare(a*gb, b*ga)
}

// spread is the strength of place pos of n, the best first, spread from
// hi down to lo. Its float maths has no multiply-add to fuse, so it writes
// the same on arm64 and amd64; exact integer rounding would round some
// halves up that it rounds down and change the seasons.
func spread(pos, n, lo, hi int) int {
	p := 1.0
	if n > 1 {
		p = 1 - float64(pos)/float64(n-1)
	}
	return lo + int(math.Round(p*float64(hi-lo)))
}

// ordered sorts the rows of a tier best first: by place, so that the
// groups of a tier alternate, then points and goal difference per game.
// Before the first game the places say nothing: clubs without games go by
// the season before (unplayed); those the file lacks keep their order.
func ordered(rows []row) []row {
	out := append([]row(nil), rows...)
	sort.SliceStable(out, func(a, b int) bool {
		x, y := out[a], out[b]
		if x.games == 0 && y.games == 0 && x.tier != fillTier {
			if c := unplayed(x, y, -1); c != 0 {
				return c < 0
			}
			return x.name < y.name
		}
		if x.rank != y.rank {
			return x.rank < y.rank
		}
		if c := comparePerGame(x.points, x.games, y.points, y.games); c != 0 {
			return c > 0
		}
		if c := comparePerGame(x.goalsFor-x.goalsAgainst, x.games, y.goalsFor-y.goalsAgainst, y.games); c != 0 {
			return c > 0
		}
		return x.name < y.name
	})
	return out
}

// placed is a German club on a place of the game.
type placed struct {
	row    row
	league int // 0..2, 3 = outside the leagues
}

// placeClubs puts the clubs onto the 64 German places: 18 in the first
// league, 20 in the second and third, 6 outside. A league with more clubs
// than places passes its last ones down, one with fewer takes the best of
// the tier below; the clubs of the season before come last, before
// secondTeamsFrom only the second teams after them.
func placeClubs(rows []row, year int) ([]placed, error) {
	tiers := map[int][]row{}
	for _, r := range rows {
		tiers[r.tier] = append(tiers[r.tier], r)
	}
	lower := append(ordered(tiers[3]), ordered(tiers[4])...)
	lower = append(lower, ordered(tiers[fillTier])...)
	if year < secondTeamsFrom {
		var firsts, seconds []row
		for _, r := range lower {
			if r.second {
				seconds = append(seconds, r)
			} else {
				firsts = append(firsts, r)
			}
		}
		lower = append(firsts, seconds...)
	}
	cands := [][]row{ordered(tiers[1]), ordered(tiers[2]), lower}
	sizes := []int{league1, league2, league3 + outside}
	var out []placed
	used := map[string]bool{}
	var down []row // passed down from the league above
	for l := 0; l < 3; l++ {
		list := append(down, cands[l]...)
		down = nil
		var mine []row
		for _, r := range list {
			if used[r.name] {
				continue
			}
			if len(mine) < sizes[l] {
				mine = append(mine, r)
				used[r.name] = true
			} else if l < 2 {
				down = append(down, r)
			}
		}
		// Too few: the best of the tiers below move up, but no second
		// team, which may not play in the league of its first team's.
		for k := l + 1; len(mine) < sizes[l] && k < 3; k++ {
			for _, r := range cands[k] {
				if len(mine) == sizes[l] {
					break
				}
				if !used[r.name] && !r.second {
					mine = append(mine, r)
					used[r.name] = true
				}
			}
		}
		if len(mine) < sizes[l] {
			return nil, fmt.Errorf("league %d: %d clubs, want %d", l+1, len(mine), sizes[l])
		}
		for i, r := range mine {
			lg := l
			if l == 2 && i >= league3 {
				lg = 3
			}
			out = append(out, placed{row: r, league: lg})
		}
	}
	return out, nil
}

// strengths gives the clubs of a place their three strengths (defence,
// midfield, attack): their goals against, points and goals per game,
// ranked within the place and spread over the range of the original. A
// club from a higher tier ranks before those of the place's own, one from
// a lower tier after them. Two clubs without games (before the first game
// of the season) rank by the season before (unplayed); else without games
// the order of the place counts.
func strengths(clubs []placed) map[string][3]int {
	out := map[string][3]int{}
	for lg := 0; lg < 4; lg++ {
		var rows []row
		for _, c := range clubs {
			if c.league == lg {
				rows = append(rows, c.row)
			}
		}
		lo, hi := strengthRange[lg][0], strengthRange[lg][1]
		metric := [3]func(r row) int{
			func(r row) int { return -r.goalsAgainst },
			func(r row) int { return r.points },
			func(r row) int { return r.goalsFor },
		}
		for k := 0; k < 3; k++ {
			idx := make([]int, len(rows))
			for i := range idx {
				idx[i] = i
			}
			sort.SliceStable(idx, func(a, b int) bool {
				ra, rb := rows[idx[a]], rows[idx[b]]
				if ra.tier != rb.tier {
					return ra.tier < rb.tier
				}
				if ra.games == 0 && rb.games == 0 {
					if c := unplayed(ra, rb, k); c != 0 {
						return c < 0
					}
					return idx[a] < idx[b]
				}
				if ra.games == 0 || rb.games == 0 {
					return idx[a] < idx[b]
				}
				return comparePerGame(metric[k](ra), ra.games, metric[k](rb), rb.games) > 0
			})
			for pos, i := range idx {
				v := out[rows[i].name]
				v[k] = spread(pos, len(rows), lo, hi)
				out[rows[i].name] = v
			}
		}
	}
	return out
}
