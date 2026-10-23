package game

// EventLabel describes today's event for manager mi like the header of the
// main menu ("12. Spieltag", "DFB-Pokal", ...).
func (s *State) EventLabel(mi int) string {
	if s.Loop.SeasonEnd {
		return "SAISONENDE"
	}
	ev := s.eventValue()
	l := s.Managers[mi].League
	switch {
	case ev&(1<<l) != 0:
		return itoa(s.Matchday[l]) + ". Spieltag"
	case ev&EvDFB != 0:
		return "DFB-Pokal"
	case ev&EvEurope == EvRelegation:
		return "Relegation"
	case ev&EvEurope != 0:
		return "Europapokal"
	case ev&EvReplay != 0:
		return "Nachholspiel"
	}
	return "Spielfrei"
}

// NextOpponent returns the opponent of manager mi in today's event.
func (s *State) NextOpponent(mi int) (opp int, home bool, ok bool) {
	ev := s.eventValue()
	m := s.Managers[mi]
	if l := m.League; ev&(1<<l) != 0 && s.Matchday[l] <= MatchdaysPerLeague[l] {
		for _, p := range Fixtures(l, s.Matchday[l]) {
			if p.Home == m.Team {
				return p.Away, true, true
			}
			if p.Away == m.Team {
				return p.Home, false, true
			}
		}
	}
	cups := []Cup{}
	if ev&EvDFB != 0 {
		cups = append(cups, CupDFB)
	}
	if ev&EvEurope != 0 && ev&EvEurope != EvRelegation {
		cups = append(cups, CupLandesmeister, CupPokalsieger, CupUEFA)
	}
	for _, c := range cups {
		for _, p := range s.CupPairs(c) {
			if p.Home == m.Team {
				return p.Away, true, true
			}
			if p.Away == m.Team {
				return p.Home, false, true
			}
		}
	}
	return 0, false, false
}
