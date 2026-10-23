package game

// CancelNegotiation ends the contract talks with squad slot si of squad m
// without a new contract (ABBRUCH in the negotiation panel, f_2277_2a8f).
// Like an offer, it makes the player unwilling to talk again for a while.
func (s *State) CancelNegotiation(m, si int) {
	sl := &s.Squads[m][si]
	if sl.Nego > 99 {
		sl.Message = false
	}
	sl.Nego = s.Rand.Between(10, 18)
}

// Scorer is a line of the list of the best scorers of a league.
type Scorer struct {
	Person int
	Goals  int
	Games  int
	Rank   int // place in the league list, also when humans filters
}

// TopScorers returns the best scorers of league l like f_14a4_1ad5: all
// players with at least two league goals, sorted by goals and then by
// fewer games. With humans set, only players of clubs of human managers
// are listed.
func (s *State) TopScorers(l int, humans bool) []Scorer {
	var all []Scorer
	for p := 1; p < NumPersons; p++ {
		ps := &s.Persons[p]
		all = append(all, Scorer{Person: p, Goals: ps.LeagueGoals, Games: ps.LeagueGames})
	}
	// The original sorts with an exchange sort.
	for i := 0; i < len(all); i++ {
		for j := i + 1; j < len(all); j++ {
			a, b := all[i], all[j]
			if a.Goals < b.Goals || (a.Goals == b.Goals && b.Games < a.Games) {
				all[i], all[j] = b, a
			}
		}
	}
	lo, hi := 0, 17
	switch l {
	case 1:
		lo, hi = 18, 37
	case 2:
		lo, hi = 38, 57
	}
	var out []Scorer
	rank := 0
	for _, sc := range all {
		t := s.Persons[sc.Person].Team
		if t < lo || t > hi {
			continue
		}
		if sc.Goals < 2 {
			break
		}
		rank++
		if humans && s.ManagerOf(t) == nil {
			continue
		}
		sc.Rank = rank
		out = append(out, sc)
	}
	return out
}
