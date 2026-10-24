package game

// CancelNegotiation ends the contract talks with squad slot si of squad m
// without a new contract (ABBRUCH in the negotiation panel, f_2277_2a8f).
// Like an offer, it makes the player unwilling to talk again for a while.
func (s *State) CancelNegotiation(m, si int) {
	sl := &s.Squads[m][si]
	if sl.Nego > 99 {
		s.endMessage(sl)
	}
	sl.Nego = s.Rand.Between(10, 18)
}

// OptimalContract is the offer of the contract key o: the cheapest one the
// player accepts whatever the generator draws.
type OptimalContract struct {
	Years  int
	Salary int64
	OK     bool // false: no sure offer for the asked length
}

// OptimalContract returns the cheapest sure offer for squad slot si of
// squad m and y years, or with y == 0 for the longest length that has one.
// A player who asks for a length himself takes it at his salary
// (NegotiateContract); otherwise only a longer contract extends it. The
// generator is not used.
func (s *State) OptimalContract(m, si, y int) OptimalContract {
	sl := &s.Squads[m][si]
	if sl.Loan != 0 {
		return OptimalContract{}
	}
	if sl.Nego >= 100 && sl.Nego&0x80 == 0 {
		forced := sl.Nego - 100
		if y != 0 && y != forced {
			return OptimalContract{}
		}
		return OptimalContract{Years: forced, Salary: sl.Salary, OK: true}
	}
	if y != 0 {
		sal, ok := s.sureSalary(m, si, y)
		return OptimalContract{Years: y, Salary: sal, OK: ok}
	}
	for y := 4; y > sl.Years; y-- {
		if sal, ok := s.sureSalary(m, si, y); ok {
			return OptimalContract{Years: y, Salary: sal, OK: true}
		}
	}
	return OptimalContract{}
}

// sureSalary returns the lowest salary, in full 100 DM, that playerAccepts
// (f_2277_2270) takes for a contract of y years longer than the current
// one with every draw: above the highest threshold d it can draw, which
// must stay below 100,000 DM, and not below the current salary.
func (s *State) sureSalary(m, si, y int) (int64, bool) {
	sl := &s.Squads[m][si]
	if y <= sl.Years || y > 4 {
		return 0, false
	}
	yp := s.contractYp(m, y)
	k := (yp/6 + 110) * 8 / 10
	d := s.PlayerValue(m, si, 5) * int64(k+4) / 100
	if s.Day > 321 {
		d = d * int64(152-s.Persons[sl.Person].Age) / 100
	}
	if d >= 100000 {
		return 0, false
	}
	sal := max(d+1, sl.Salary)
	return (sal + 99) / 100 * 100, true
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
