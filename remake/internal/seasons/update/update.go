// Package update makes a real season of the remake (internal/seasons/data)
// from the Transfermarkt file of the season (tools/transfermarkt) and the
// season before: the clubs of the tables on the places of the game with
// strengths from their table, their squads by market value; clubs the file
// lacks, the logos and the foreign clubs come from the season before.
package update

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"math"
	"os"
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
	Players struct {
		Goalkeepers []string `json:"goalkeepers"`
		Defenders   []string `json:"defenders"`
		Midfielders []string `json:"midfielders"`
		Forwards    []string `json:"forwards"`
	} `json:"players"`
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
	Clubs    []Club         // the Transfermarkt file of the season
	Foreign  []Foreign      // the foreign clubs of its European cups, if fetched
	Previous *assets.Mana   // the season before (or the clubs of the original)
	Known    []*assets.Mana // every season, for the logos of clubs by name
	NextLogo int            // the number of the next new logo
}

// Output is a season and the logos it adds.
type Output struct {
	Mana  *assets.Mana
	Logos map[int]*assets.Pic // new logos by number
}

// Season makes the season of in.
func Season(in Input) (*Output, error) {
	// A club keeps the name it had in the season before, else in any
	// season, so that it keeps its logo.
	var prevNames, tmNames []string
	for _, c := range in.Previous.Clubs[:numGerman] {
		prevNames = append(prevNames, assets.FromCP437(c.Name))
	}
	logos := knownLogos(in.Known)
	for _, c := range in.Clubs {
		tmNames = append(tmNames, clubName(c.Name))
	}
	byPrev := matchNames(tmNames, prevNames)
	var others []string
	for _, n := range tmNames {
		if _, ok := byPrev[n]; !ok {
			others = append(others, n)
		}
	}
	var knownNames []string
	for n := range logos {
		knownNames = append(knownNames, n)
	}
	sort.Strings(knownNames)
	byKnown := matchNames(others, knownNames)
	var rows []row
	tm := map[string]Club{}
	for _, c := range in.Clubs {
		name := clubName(c.Name)
		if m, ok := byPrev[name]; ok {
			name = m
		} else if m, ok := byKnown[name]; ok {
			name = m
		}
		if _, ok := tm[name]; ok || name == "" {
			continue // the same club twice
		}
		tm[name] = c
		gf, ga := goals(c.Goals)
		rows = append(rows, row{name: name, tier: tier(c), group: c.League, rank: c.Rank,
			games: c.Games, goalsFor: gf, goalsAgainst: ga, points: c.Points})
	}
	// The German clubs of the season before that the file lacks fill what
	// is left, in their order there.
	prev := map[string]assets.ClubRecord{}
	for i, c := range in.Previous.Clubs[:numGerman] {
		name := assets.FromCP437(c.Name)
		prev[name] = c
		if _, ok := tm[name]; !ok {
			rows = append(rows, row{name: name, tier: fillTier, rank: i})
		}
	}
	places, err := placeClubs(rows)
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
			for _, n := range pickPlayers(t) {
				cp, err := assets.ToCP437(n)
				if err != nil {
					return nil, fmt.Errorf("%s: %w", t.Name, err)
				}
				c.Players = append(c.Players, cp)
			}
		} else {
			c.Players = prev[p.row.name].Players
		}
		switch logo, ok := logos[p.row.name]; {
		case ok:
			c.Logo = logo
		default:
			c.Logo = next
			out.Logos[next] = crest("")
			logos[p.row.name] = next
			next++
		}
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
		p := 1.0
		if perCup[f.Cup] > 1 {
			p = 1 - float64(rank[f.Cup])/float64(perCup[f.Cup]-1)
		}
		rank[f.Cup]++
		v := r[0] + int(math.Round(float64(p*float64(r[1]-r[0]))))
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
func pickPlayers(c Club) []string {
	var out []string
	for k, names := range [4][]string{c.Players.Goalkeepers, c.Players.Defenders, c.Players.Midfielders, c.Players.Forwards} {
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

// Places of the game (internal/game: TeamsPerLeague, NumGerman).
const (
	league1   = 18
	league2   = 20
	league3   = 20
	outside   = 6
	numGerman = league1 + league2 + league3 + outside
	fillTier  = 5 // a club of the season before that the file lacks
)

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
}

func perGame(v, games int) float64 {
	if games == 0 {
		return 0
	}
	return float64(v) / float64(games)
}

// ordered sorts the rows of a tier best first: by place, so that the
// groups of a tier alternate, then points and goal difference per game.
func ordered(rows []row) []row {
	out := append([]row(nil), rows...)
	sort.SliceStable(out, func(a, b int) bool {
		x, y := out[a], out[b]
		if x.rank != y.rank {
			return x.rank < y.rank
		}
		if px, py := perGame(x.points, x.games), perGame(y.points, y.games); px != py {
			return px > py
		}
		dx, dy := perGame(x.goalsFor-x.goalsAgainst, x.games), perGame(y.goalsFor-y.goalsAgainst, y.games)
		if dx != dy {
			return dx > dy
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
// the tier below; the clubs of the season before come last.
func placeClubs(rows []row) ([]placed, error) {
	tiers := map[int][]row{}
	for _, r := range rows {
		tiers[r.tier] = append(tiers[r.tier], r)
	}
	lower := append(ordered(tiers[3]), ordered(tiers[4])...)
	cands := [][]row{ordered(tiers[1]), ordered(tiers[2]), append(lower, ordered(tiers[fillTier])...)}
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
		// Too few: the best of the tiers below move up.
		for k := l + 1; len(mine) < sizes[l] && k < 3; k++ {
			for _, r := range cands[k] {
				if len(mine) == sizes[l] {
					break
				}
				if !used[r.name] {
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
// a lower tier after them; without games the order of the place counts.
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
		metric := [3]func(r row) float64{
			func(r row) float64 { return -perGame(r.goalsAgainst, r.games) },
			func(r row) float64 { return perGame(r.points, r.games) },
			func(r row) float64 { return perGame(r.goalsFor, r.games) },
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
				if ra.games == 0 || rb.games == 0 {
					return idx[a] < idx[b]
				}
				return metric[k](ra) > metric[k](rb)
			})
			for pos, i := range idx {
				p := 1.0
				if len(rows) > 1 {
					p = 1 - float64(pos)/float64(len(rows)-1)
				}
				v := out[rows[i].name]
				v[k] = lo + int(math.Round(p*float64(hi-lo)))
				out[rows[i].name] = v
			}
		}
	}
	return out
}
