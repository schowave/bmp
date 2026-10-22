package game

// ClubStats is the season record of a German club (F:0x0ecc + id*0x36).
// Every statistic has a home [0] and an away [1] part. The snapshot copies
// are taken before a matchday so that the table can be recomputed from live
// scores at any minute.
type ClubStats struct {
	Points, PointsSnap [2]int
	Form, FormSnap     string // last 8 results, '@' = none
	Goals, GoalsSnap   [2]int
	Against, AgainstSn [2]int
	Games              [2]int
	Wins, WinsSnap     [2]int
	Losses, LossesSnap [2]int
	Pos                int   // current table position, 0-based
	Eternal            int64 // eternal-table points, never reset
}

const emptyForm = "@@@@@@@@"

// Total helpers.
func (c *ClubStats) TotalPoints() int  { return c.Points[0] + c.Points[1] }
func (c *ClubStats) TotalGames() int   { return c.Games[0] + c.Games[1] }
func (c *ClubStats) TotalGoals() int   { return c.Goals[0] + c.Goals[1] }
func (c *ClubStats) TotalAgainst() int { return c.Against[0] + c.Against[1] }
func (c *ClubStats) TotalWins() int    { return c.Wins[0] + c.Wins[1] }
func (c *ClubStats) TotalLosses() int  { return c.Losses[0] + c.Losses[1] }
func (c *ClubStats) TotalDraws() int {
	return c.TotalGames() - c.TotalWins() - c.TotalLosses()
}

// MinusPoints is shown next to the points under the 2-point rule.
func (c *ClubStats) MinusPoints() int { return 2*c.TotalGames() - c.TotalPoints() }

func (c *ClubStats) resetSeason() {
	stats := ClubStats{Eternal: c.Eternal, Pos: c.Pos, Form: emptyForm, FormSnap: emptyForm}
	*c = stats
}

func (c *ClubStats) snapshot() {
	c.PointsSnap, c.GoalsSnap, c.AgainstSn = c.Points, c.Goals, c.Against
	c.WinsSnap, c.LossesSnap, c.FormSnap = c.Wins, c.Losses, c.Form
}

func (c *ClubStats) restore() {
	c.Points, c.Goals, c.Against = c.PointsSnap, c.GoalsSnap, c.AgainstSn
	c.Wins, c.Losses, c.Form = c.WinsSnap, c.LossesSnap, c.FormSnap
}

// addForm appends a result letter (f_2a41_1fb0): 's'/'S' win, 'u'/'U' draw,
// 'n'/'N' loss, lower case at home.
func (c *ClubStats) addForm(ch byte) {
	f := []byte(c.Form)
	if len(f) != 8 {
		f = []byte(emptyForm)
	}
	copy(f, f[1:])
	f[7] = ch
	c.Form = string(f)
}

// Postponed marks a game that has been postponed (home goals 0x1e).
const Postponed = 0x1e

// Score is the result of a match. Home == Postponed marks a postponed game.
type Score struct {
	Home, Away int
}

// snapshotMatchday saves the table fields of all clubs playing on the
// current matchday of league l (f_2a41_2d33 mode 1).
func (s *State) snapshotMatchday(pairs []Pair, results []Score) {
	for i, p := range pairs {
		if results[i].Home == Postponed {
			continue
		}
		s.Clubs[p.Home].snapshot()
		s.Clubs[p.Away].snapshot()
	}
}

// applyResults adds results to the table (f_2a41_2d33 mode 0). It restores
// the snapshot first, so calling it again with newer scores is safe; only
// the games counters are not snapshotted and must be counted once.
func (s *State) applyResults(pairs []Pair, results []Score, countGames bool) {
	for i, p := range pairs {
		r := results[i]
		if r.Home == Postponed {
			continue
		}
		h, a := &s.Clubs[p.Home], &s.Clubs[p.Away]
		h.restore()
		a.restore()
		h.Goals[0] += r.Home
		h.Against[0] += r.Away
		a.Goals[1] += r.Away
		a.Against[1] += r.Home
		if countGames {
			h.Games[0]++
			a.Games[1]++
		}
		switch {
		case r.Home > r.Away:
			h.Points[0] += PointsPerWin
			h.Wins[0]++
			a.Losses[1]++
			h.addForm('s')
			a.addForm('N')
		case r.Home == r.Away:
			h.Points[0]++
			a.Points[1]++
			h.addForm('u')
			a.addForm('U')
		default:
			a.Points[1] += PointsPerWin
			a.Wins[1]++
			h.Losses[0]++
			h.addForm('n')
			a.addForm('S')
		}
	}
}

// sortTable sorts the table of league l (exchange sort of f_2a41_2d33):
// points, fewer games, goal difference, goals. Ties keep their order.
func (s *State) sortTable(l int) {
	t := s.Tables[l][:TeamsPerLeague[l]]
	n := len(t)
	for i := 0; i < n-1; i++ {
		for j := i + 1; j < n; j++ {
			a, b := &s.Clubs[t[i]], &s.Clubs[t[j]]
			pa, pb := a.TotalPoints(), b.TotalPoints()
			swap := false
			switch {
			case pb > pa:
				swap = true
			case pb == pa:
				ga, gb := a.TotalGames(), b.TotalGames()
				if gb < ga {
					swap = true
				} else if ga == gb {
					da := a.TotalGoals() - a.TotalAgainst()
					db := b.TotalGoals() - b.TotalAgainst()
					if da < db || (da == db && a.TotalGoals() < b.TotalGoals()) {
						swap = true
					}
				}
			}
			if swap {
				t[i], t[j] = t[j], t[i]
			}
		}
	}
	for i, id := range t {
		s.Clubs[id].Pos = i
	}
	if md := s.Matchday[l] - 1; md >= 0 && md < len(Manager{}.PosAfter) {
		for _, m := range s.Managers {
			if m.League == l {
				m.PosAfter[md] = s.Clubs[m.Team].Pos
			}
		}
	}
}
