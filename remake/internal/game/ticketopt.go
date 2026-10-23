package game

// The optimal ticket price is not a rule of the original: it rates every
// price of the stadium screen by the gate money the attendance of the
// original (Attendance, f_0f9d_11e0) gives at the next home match, and keeps
// the best. It changes nothing but the price; the generator of the game
// stays where it is.

// OptimalPrice is the result of OptimizeTicketPrice: the price before and
// the price chosen with the expected gate money of the next home match for
// each, against Opponent (a club id, -1 when no home match is left).
type OptimalPrice struct {
	Before, After         int
	BeforeGate, AfterGate int64
	Opponent              int
}

// ticketRuns is the number of fixed series of dice the gate money is the
// mean of.
const ticketRuns = 32

// OptimizeTicketPrice sets the ticket price (5..25 DM) with the most
// expected gate money at the next home match of manager m: today's match if
// it is at home, else the next home match of his league. Without one the
// price stays.
func (s *State) OptimizeTicketPrice(m int) OptimalPrice {
	mg := s.Managers[m]
	res := OptimalPrice{Before: mg.TicketPrice, After: mg.TicketPrice, Opponent: -1}
	opp, imp, ok := s.nextHomeMatch(m)
	if !ok {
		return res
	}
	res.Opponent = opp
	res.BeforeGate = s.expectedGate(m, opp, imp, mg.TicketPrice)
	res.AfterGate = res.BeforeGate
	for p := 5; p <= 25; p++ {
		if g := s.expectedGate(m, opp, imp, p); g > res.AfterGate {
			res.After, res.AfterGate = p, g
		}
	}
	s.SetTicketPrice(m, res.After)
	return res
}

// expectedGate is the mean gate money attendance × price of a home match of
// manager m against opp at price, over ticketRuns fixed series of dice.
func (s *State) expectedGate(m, opp, imp, price int) int64 {
	saved := s.Rand
	defer func() { s.Rand = saved }()
	var sum int64
	for r := 0; r < ticketRuns; r++ {
		s.Rand = Rand{Seed: trainingSeed + uint32(r)}
		sum += s.Attendance(m, s.Managers[m].Team, opp, imp, false, price) * int64(price)
	}
	return sum / ticketRuns
}

// nextHomeMatch returns the opponent and the importance of the next home
// match of manager m (imp as Attendance takes it): today's match if he plays
// it at home, else the next one at home in his league.
func (s *State) nextHomeMatch(m int) (opp, imp int, ok bool) {
	mg := s.Managers[m]
	if o, home, today := s.NextOpponent(m); today && home {
		imp = 1
		if ev := s.eventValue(); ev&(1<<mg.League) == 0 && ev&EvDFB == 0 {
			imp = 2 // a European cup
		}
		return o, imp, true
	}
	l := mg.League
	for md := s.Matchday[l]; md <= MatchdaysPerLeague[l]; md++ {
		for _, p := range Fixtures(l, md) {
			if p.Home == mg.Team {
				return p.Away, 1, true
			}
		}
	}
	return 0, 0, false
}
