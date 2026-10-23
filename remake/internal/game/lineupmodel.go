package game

import (
	"math"
	"sync"
)

// The model rates a lineup by what the match engine makes of it: the expected
// points of a match (2 for a win, 1 for a draw) against the next opponent, or
// against the clubs of the league, each at home and away. The strengths come
// from lineupValues, the chances of the two sides from chanceCounts and the
// goals from the odds of chance (f_0f9d_0c3b), so the rating weighs the goals
// scored against the goals let in as the result of a match does. What it
// leaves out: cards, injuries and substitutions of the match, the 9 goals a
// side stops at and the second leg of cups.
//
// Against the league one opponent would not do. The odds of a goal are not
// smooth in the engine:
// with the attack 0 to 3 points of strength (the difference divided by 3)
// behind the defence a chance is a goal for sure, one point ahead it is only
// two in three. A search against a single average club would shift its own
// strengths onto that step and gain nothing against any other club.

// With the next opponent known the step is no trap but a fact: the strengths
// of the clubs change before the turn of the managers and not again before
// the match, so the opponent is rated as he will play.

// lineupRater rates the lineup of manager m against the other clubs of his
// league, or against his next opponent. The chances of the engine count by
// their expected number (expectedChances). The dice of the lineup strength
// are replaced by their mean (meanDice) against the league; against one
// opponent the odds of a goal are not smooth, so the rating is the mean of a
// few fixed series of dice (ratingSamples).
type lineupRater struct {
	s      *State
	m      int
	mood   int    // the Mood of the club of m
	refs   []Team // the clubs of the league without a human manager, or the opponent
	venues []bool // home matches, away matches or both
	opp    int    // the club id of the opponent, or -1 for the league
	// samples is the number of series of dice, 0 for the mean.
	samples int
}

// ratingSamples is the number of series of dice that rate a lineup against
// one opponent.
const ratingSamples = 8

// newRater returns the rater of manager m: against his next opponent if he
// has one, else against the league.
func (s *State) newRater(m int) *lineupRater {
	if opp, home, ok := s.NextOpponent(m); ok && !s.humanTeam(opp) {
		// An opponent with a human manager plays with a lineup not known yet.
		lr := s.newLeagueRater(m)
		lr.refs, lr.venues, lr.opp, lr.samples = []Team{s.Teams[opp]}, []bool{home}, opp, ratingSamples
		return lr
	}
	return s.newLeagueRater(m)
}

func (s *State) humanTeam(t int) bool {
	for _, mg := range s.Managers {
		if mg.Team == t {
			return true
		}
	}
	return false
}

// newLeagueRater returns the rater of manager m against the clubs of his
// league.
func (s *State) newLeagueRater(m int) *lineupRater {
	lr := &lineupRater{s: s, m: m, mood: s.Teams[s.Managers[m].Team].Mood, venues: []bool{true, false}, opp: -1}
	l := LeagueOf(s.Managers[m].Team)
	first, end := FirstID[l], 58
	if l < NumLeagues-1 {
		end = FirstID[l+1]
	}
	for t := first; t < end; t++ {
		if !s.humanTeam(t) {
			lr.refs = append(lr.refs, s.Teams[t])
		}
	}
	if len(lr.refs) == 0 {
		// No other club in the league: his own club stands for them.
		lr.refs = []Team{s.Teams[s.Managers[m].Team]}
	}
	return lr
}

// rate is the rating of the lineup as it is.
func (lr *lineupRater) rate() int {
	if lr.samples == 0 {
		return lr.ofValues(lr.s.lineupValues(lr.m, true, meanDice{}))
	}
	sum := 0.0
	for k := 0; k < lr.samples; k++ {
		r := Rand{Seed: ratingSeed + uint32(k)}
		sum += lr.points(lr.s.lineupValues(lr.m, true, &r), lr.refs)
	}
	return int(math.Round(sum / float64(lr.samples) * 100))
}

// ratingSeed starts the series of dice of a rating.
const ratingSeed = 20240301

// ofValues is the rating of the strength values v of a lineup: the expected
// points per match in hundredths (2 for a win, 1 for a draw), the mean over
// the clubs and venues of the rater.
func (lr *lineupRater) ofValues(v lineupValues) int {
	return lr.ofValuesAgainst(v, lr.refs)
}

func (lr *lineupRater) ofValuesAgainst(v lineupValues, refs []Team) int {
	return int(math.Round(lr.points(v, refs) * 100))
}

// points is the expected points per match of the lineup with the strength
// values v against the clubs refs.
func (lr *lineupRater) points(v lineupValues, refs []Team) float64 {
	us := Team{Mood: lr.mood, Strength: v.strength, Base: v.base, Extra: v.extra}
	sum := 0.0
	for i := range refs {
		for _, home := range lr.venues {
			h, a := &us, &refs[i]
			if !home {
				h, a = a, h
			}
			win, draw, loss := matchOdds(h, a)
			if !home {
				win = loss
			}
			sum += PointsPerWin*win + draw
		}
	}
	return sum / float64(len(lr.venues)*len(refs))
}

// maxHalfChances is the most chances a side gets in a half of 44 minutes.
const maxHalfChances = 8

// matchOdds returns the probabilities that the home club h wins, draws or
// loses against the away club a. A half gives each side a number of chances
// (chanceCounts, a joint distribution) and each chance is a goal with the
// odds of its minute; as the minutes of the chances are drawn alike, a side
// with n chances scores like n draws with the mean odds of the half. The two
// halves add up. The 9 goals a side stops at are left out.
func matchOdds(h, a *Team) (win, draw, loss float64) {
	const n = 2*maxHalfChances + 1
	var halves [2][n]float64 // goal difference home - away + 8 of each half
	for k, half := range [2]int{1, 46} {
		d := teamStrengthAt(a, 1, half)/(-3) + teamStrengthAt(h, 1, half)/3
		joint := chanceJoint(d)
		// The chances of a half are spread over its minutes; four minutes
		// stand for them.
		var oh, oa float64
		for _, m := range [4]int{5, 16, 27, 38} {
			mid := half + m
			dH := (teamStrengthAt(h, 2, mid) - teamStrengthAt(a, 0, mid)) / 3
			dA := (teamStrengthAt(a, 2, mid) - teamStrengthAt(h, 0, mid)) / 3
			oh += goalOdds(0, dH) / 4
			oa += goalOdds(1, dA) / 4
		}
		bh, ba := binomials(oh), binomials(oa)
		for ch := 0; ch <= maxHalfChances; ch++ {
			for ca := 0; ca <= maxHalfChances; ca++ {
				p := joint[ch][ca]
				if p == 0 {
					continue
				}
				for gh := 0; gh <= ch; gh++ {
					for ga := 0; ga <= ca; ga++ {
						halves[k][gh-ga+maxHalfChances] += p * bh[ch][gh] * ba[ca][ga]
					}
				}
			}
		}
	}
	for i, p := range halves[0] {
		if p == 0 {
			continue
		}
		for j, q := range halves[1] {
			switch diff := i + j - 2*maxHalfChances; {
			case diff > 0:
				win += p * q
			case diff == 0:
				draw += p * q
			default:
				loss += p * q
			}
		}
	}
	return win, draw, loss
}

// binomials returns b[n][k], the probability of k goals from n chances that
// are each a goal with probability p.
func binomials(p float64) (b [maxHalfChances + 1][maxHalfChances + 1]float64) {
	b[0][0] = 1
	for n := 1; n <= maxHalfChances; n++ {
		for k := 0; k <= n; k++ {
			b[n][k] = b[n-1][k] * (1 - p)
			if k > 0 {
				b[n][k] += b[n-1][k-1] * p
			}
		}
	}
	return b
}

// goalOdds is the probability that a chance of side (0 home, 1 away) is a
// goal when the attack is dd ahead of the defence (f_0f9d_0c3b).
func goalOdds(side, dd int) float64 {
	pass := 0.5
	if side == 0 {
		pass = 0.6
	}
	if dd > 0 {
		n := float64(dd>>2 + 2)
		return pass * n / (n + 1)
	}
	k := float64((-dd) >> 2)
	p := 1 / (k + 1)
	return pass * (p + (1-p)/7)
}

const (
	chanceTableMax   = 80   // d is kept within -80..80
	chanceTableDraws = 3000 // draws per entry
)

var chanceTable struct {
	once  sync.Once
	joint [2*chanceTableMax + 1][maxHalfChances + 1][maxHalfChances + 1]float64
}

func buildChanceTable() {
	t := &chanceTable
	t.once.Do(func() {
		for i := range t.joint {
			d := i - chanceTableMax
			r := Rand{Seed: 20240229}
			for n := 0; n < chanceTableDraws; n++ {
				x, y := chanceCounts(&r, 3*(100+d), 300, 1, 45)
				t.joint[i][x][y] += 1.0 / chanceTableDraws
			}
		}
	})
}

// chanceJoint returns the distribution of the chances of home and away in a
// half of 44 minutes when the midfield difference of the two is d
// (chanceCounts): [home][away]. The table is made once with a fixed series of
// random numbers.
func chanceJoint(d int) *[maxHalfChances + 1][maxHalfChances + 1]float64 {
	buildChanceTable()
	return &chanceTable.joint[clamp(d, -chanceTableMax, chanceTableMax)+chanceTableMax]
}

// expectedChances returns the mean chances of home and away in a half of 44
// minutes when the midfield difference of the two is d.
func expectedChances(d int) (float64, float64) {
	var h, a float64
	for x, row := range chanceJoint(d) {
		for y, p := range row {
			h += float64(x) * p
			a += float64(y) * p
		}
	}
	return h, a
}
