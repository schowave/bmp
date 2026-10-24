package game

import "testing"

func TestEventLabelOnlyForClubsTakingPart(t *testing.T) {
	s, err := NewGame(mana(t), Options{Managers: []NewManager{{Name: "TEST", Team: 9}}, Level: 2, Seed: 1})
	if err != nil {
		t.Fatal(err)
	}
	m := s.Managers[0]
	set := func(ev byte) { s.Events[s.Event] = ev }

	set(EvDFB)
	s.Cups.Round[CupDFB] = 2
	m.CupStatus[CupDFB] = 2
	if got := s.EventLabel(0); got != "DFB-Pokal" {
		t.Errorf("DFB day in the cup: %q", got)
	}
	m.CupStatus[CupDFB] = 1 // knocked out in round 1
	if got := s.EventLabel(0); got != "Spielfrei" {
		t.Errorf("DFB day knocked out: %q", got)
	}

	set(0x20)
	s.Cups.Round = [NumCups]int{2, 2, 2, 2}
	m.CupStatus = [4]int{NotInCup, NotInCup, NotInCup, NotInCup}
	if got := s.EventLabel(0); got != "Spielfrei" {
		t.Errorf("Europe day not taking part: %q", got)
	}
	m.CupStatus[CupUEFA] = 2
	if got := s.EventLabel(0); got != "Europapokal" {
		t.Errorf("Europe day in the UEFA-Pokal: %q", got)
	}

	set(EvRelegation)
	s.Tables[0][15], s.Tables[1][2] = m.Team+1, m.Team+2
	if got := s.EventLabel(0); got != "Spielfrei" {
		t.Errorf("relegation without the club: %q", got)
	}
	s.Tables[0][15] = m.Team
	if got := s.EventLabel(0); got != "Relegation" {
		t.Errorf("relegation with the club: %q", got)
	}

	set(EvReplay)
	l := m.League
	other := 0
	for i, p := range Fixtures(l, 1) {
		if p.Home != m.Team && p.Away != m.Team {
			other = i
			break
		}
	}
	s.Postponed[0] = PostponedGame{Slot: s.Event, Matchday: 1, League: l, Pair: other}
	if got := s.EventLabel(0); got != "Spielfrei" {
		t.Errorf("Nachholspiele without the club: %q", got)
	}
	for i, p := range Fixtures(l, 1) {
		if p.Home == m.Team || p.Away == m.Team {
			s.Postponed[1] = PostponedGame{Slot: s.Event, Matchday: 1, League: l, Pair: i}
		}
	}
	if got := s.EventLabel(0); got != "Nachholspiel" {
		t.Errorf("Nachholspiel of the club: %q", got)
	}
}
