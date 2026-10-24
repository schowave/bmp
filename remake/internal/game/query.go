package game

// EventLabel describes today's event for manager mi like the header of the
// main menu ("12. Spieltag", "DFB-Pokal", ...). A cup day or the relegation
// shows only for a club that takes part in it, otherwise the day is
// "Spielfrei" (f_08bc_0b5f @0f28).
func (s *State) EventLabel(mi int) string {
	if s.Loop.SeasonEnd {
		return "SAISONENDE"
	}
	ev := s.eventValue()
	m := s.Managers[mi]
	l := m.League
	inCup := func(c Cup) bool { return m.CupStatus[c] == s.Cups.Round[c] }
	switch {
	case ev&(1<<l) != 0:
		return itoa(s.Matchday[l]) + ". Spieltag"
	case ev == EvDFB && inCup(CupDFB):
		return "DFB-Pokal"
	case ev == EvRelegation && (m.Team == s.Tables[0][15] || m.Team == s.Tables[1][2]):
		return "Relegation"
	case ev&EvEurope != 0 && ev != EvRelegation &&
		(inCup(CupLandesmeister) || inCup(CupPokalsieger) || inCup(CupUEFA)):
		return "Europapokal"
	case ev&EvReplay != 0 && s.hasReplays():
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
