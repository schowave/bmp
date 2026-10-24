package build

import (
	"fmt"
	"math"
	"sort"

	"github.com/schowave/bmp/remake/internal/seasons/source"
)

// Places of the game (internal/game: TeamsPerLeague, NumGerman).
const (
	league1   = 18
	league2   = 20
	league3   = 20
	outside   = 6
	numGerman = league1 + league2 + league3 + outside
)

// Strength ranges of the original per place, measured in its clubs.json.
var strengthRange = [4][2]int{{73, 95}, {46, 74}, {20, 49}, {26, 44}}

// ordered sorts the rows of a tier best first: by place, so that the
// groups of a tier alternate, then points and goal difference per game.
func ordered(rows []source.TierRow) []source.TierRow {
	out := append([]source.TierRow(nil), rows...)
	sort.SliceStable(out, func(a, b int) bool {
		x, y := out[a], out[b]
		if x.Rank != y.Rank {
			return x.Rank < y.Rank
		}
		px, py := perGame(x.Points(), x.Games()), perGame(y.Points(), y.Games())
		if px != py {
			return px > py
		}
		dx, dy := perGame(x.GoalsFor-x.GoalsAgainst, x.Games()), perGame(y.GoalsFor-y.GoalsAgainst, y.Games())
		if dx != dy {
			return dx > dy
		}
		return x.Club < y.Club
	})
	return out
}

func perGame(v, games int) float64 {
	if games == 0 {
		return 0
	}
	return float64(v) / float64(games)
}

// placed is a German club on a place of the game.
type placed struct {
	row     source.TierRow
	league  int    // 0..2, 3 = outside the leagues
	article string // the article for logo and colours: a second team has its club's
}

// placeClubs puts the clubs of the tables of a season onto the 64 German
// places: 18 in the first league, 20 in the second and third, 6 outside.
// A league with more teams than places passes its last ones down, one with
// fewer takes the best of the tier below.
func placeClubs(year int, tables []source.TierRow) ([]placed, error) {
	tier := map[int][]source.TierRow{}
	article := map[string]string{}
	for _, r := range tables {
		key := r.Team()
		article[key] = r.Club
		r.Club = key
		tier[r.Tier] = append(tier[r.Tier], r)
	}
	for k := range tier {
		tier[k] = ordered(tier[k])
	}
	// Candidates per tier, best first; reserve teams leave the amateur
	// tiers before the 3. Liga, and the Regionalliga under the 3. Liga.
	keep := func(rows []source.TierRow) []source.TierRow {
		var out []source.TierRow
		for _, r := range rows {
			if !source.IsSecondTeam(r.Club) && !source.IsSecondTeam(r.Label) {
				out = append(out, r)
			}
		}
		return out
	}
	t3 := tier[3]
	if year < 2008 {
		t3 = keep(t3)
	}
	// Second teams only when the places cannot be filled without them.
	third := append(append(t3, keep(tier[4])...), tier[3]...)
	third = append(third, tier[4]...)
	cands := [][]source.TierRow{tier[1], tier[2], third}
	sizes := []int{league1, league2, league3 + outside}
	var out []placed
	used := map[string]bool{}
	var down []source.TierRow // passed down from the league above
	for l := 0; l < 3; l++ {
		list := append(down, cands[l]...)
		down = nil
		var mine []source.TierRow
		for _, r := range list {
			if used[r.Club] {
				continue
			}
			if len(mine) < sizes[l] {
				mine = append(mine, r)
				used[r.Club] = true
			} else if l < 2 {
				down = append(down, r)
			}
		}
		// Too few: the best of the tier below move up.
		for k := l + 1; len(mine) < sizes[l] && k < 3; k++ {
			for _, r := range cands[k] {
				if len(mine) == sizes[l] {
					break
				}
				if !used[r.Club] {
					mine = append(mine, r)
					used[r.Club] = true
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
			out = append(out, placed{row: r, league: lg, article: article[r.Club]})
		}
	}
	return out, nil
}

// strengths gives the clubs of a place their three strengths (defence,
// midfield, attack): their goals against, points and goals per game,
// ranked within the place and spread over the range of the original. A
// club from a higher tier ranks before those of the place's own, one from
// a lower tier after them.
// Without games (a season that has not started) the order of the place
// counts for all three.
func strengths(clubs []placed) map[string][3]int {
	out := map[string][3]int{}
	for lg := 0; lg < 4; lg++ {
		var rows []source.TierRow
		for _, c := range clubs {
			if c.league == lg {
				rows = append(rows, c.row)
			}
		}
		lo, hi := strengthRange[lg][0], strengthRange[lg][1]
		metric := [3]func(r source.TierRow) float64{
			func(r source.TierRow) float64 { return -perGame(r.GoalsAgainst, r.Games()) },
			func(r source.TierRow) float64 { return perGame(r.Points(), r.Games()) },
			func(r source.TierRow) float64 { return perGame(r.GoalsFor, r.Games()) },
		}
		for k := 0; k < 3; k++ {
			idx := make([]int, len(rows))
			for i := range idx {
				idx[i] = i
			}
			sort.SliceStable(idx, func(a, b int) bool {
				ra, rb := rows[idx[a]], rows[idx[b]]
				if ra.Tier != rb.Tier {
					return ra.Tier < rb.Tier
				}
				if ra.Games() == 0 || rb.Games() == 0 {
					return idx[a] < idx[b]
				}
				return metric[k](ra) > metric[k](rb)
			})
			for pos, i := range idx {
				p := 1.0
				if len(rows) > 1 {
					p = 1 - float64(pos)/float64(len(rows)-1)
				}
				v := out[rows[i].Club]
				v[k] = lo + int(math.Round(p*float64(hi-lo)))
				out[rows[i].Club] = v
			}
		}
	}
	return out
}
