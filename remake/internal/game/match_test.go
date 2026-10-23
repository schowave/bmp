package game

import "testing"

// An AI shoot-out (f_0310_356d) draws who shoots first before it looks at
// the teams, then the goals of both sides until they differ.
func TestAIShootoutDraws(t *testing.T) {
	for seed := uint32(1); seed < 50; seed++ {
		s := &State{Rand: Rand{Seed: seed}}
		f := &Fixture{League: -1, Cup: 0, Pair: 4, Home: 1, Away: 2}
		md := &MatchDay{s: s, Fixtures: []*Fixture{f}}
		md.shootout(0, 4)

		want := Rand{Seed: seed}
		want.Between(0, 1)
		var h, a int
		for h == a {
			h, a = want.Between(2, 5), want.Between(2, 5)
		}
		if got := s.Cups.Result[0][4]; got != h+markPenalties || s.Cups.Result[0][5] != a {
			t.Fatalf("seed %d: %d:%d, want %d:%d", seed, got, s.Cups.Result[0][5], h+markPenalties, a)
		}
		if s.Rand.Seed != want.Seed {
			t.Fatalf("seed %d: generator at %d, want %d", seed, s.Rand.Seed, want.Seed)
		}
	}
}

// f_14a4_14d4 does nothing for a team without goals: no draws, and its
// players get no league game.
func TestAIScorersWithoutGoals(t *testing.T) {
	s := &State{Rand: Rand{Seed: 7}}
	for p := 1; p < 12; p++ {
		s.Persons[p].Team = 3
	}
	s.aiScorersOf(3, 0)
	if s.Rand.Seed != 7 {
		t.Errorf("generator moved to %d", s.Rand.Seed)
	}
	for p := 1; p < 12; p++ {
		if s.Persons[p].LeagueGames != 0 {
			t.Fatalf("person %d: %d league games", p, s.Persons[p].LeagueGames)
		}
	}

	s.aiScorersOf(3, 1)
	if s.Rand.Seed == 7 {
		t.Error("a team with a goal draws nothing")
	}
}

// On a day of the 2. Bundesliga alone, f_0310_2303 still plans the chances
// of the other two leagues: their draws are made, their results stay.
func TestIdleLeaguesArePlanned(t *testing.T) {
	s, err := NewGame(mana(t), Options{Managers: []NewManager{{Name: "TEST", Team: 9}}, Level: 2, Seed: 3})
	if err != nil {
		t.Fatal(err)
	}
	md := s.newMatchDay(2, 0, false)
	for _, f := range md.Fixtures {
		if f.League != 1 {
			t.Fatalf("fixture of league %d", f.League)
		}
	}
	before := s.Rand
	results := s.Results
	md.startPeriod(1, 45)
	all := s.Rand

	s.Rand = before
	md.planFixtures(1, 45, func(*Fixture) bool { return true })
	if s.Rand == all {
		t.Error("the idle leagues draw nothing")
	}
	if s.Results[0] != results[0] || s.Results[2] != results[2] {
		t.Error("planning changes the results of an idle league")
	}

	// Planning the three leagues in their order gives the same draws.
	s.Rand = before
	md.planIdle(0, 1, 45)
	md.planFixtures(1, 45, func(*Fixture) bool { return true })
	md.planIdle(2, 1, 45)
	if s.Rand != all {
		t.Error("the leagues are planned out of order")
	}
}

// twoManagerTie is a DFB-Pokal tie of two human managers; the away
// manager has the lower index.
func twoManagerTie(t *testing.T) (*State, *MatchDay, *Fixture) {
	t.Helper()
	s, err := NewGame(mana(t), Options{Managers: []NewManager{{Name: "GAST", Team: 9}, {Name: "HEIM", Team: 4}}, Level: 2, Seed: 11})
	if err != nil {
		t.Fatal(err)
	}
	s.Cups.Round[CupDFB] = 1
	f := &Fixture{League: -1, Cup: CupDFB, Home: s.Managers[1].Team, Away: s.Managers[0].Team}
	md := &MatchDay{s: s, Fixtures: []*Fixture{f}, humans: make([]humanMatch, 2)}
	return s, md, f
}

// f_1a58_20b2 goes through the managers by index: the away manager who
// comes first gets the estimate for an AI home team as his share.
func TestTieManagersInIndexOrder(t *testing.T) {
	s, md, f := twoManagerTie(t)
	r0, money := s.Rand, s.Managers[0].Money
	price := AITicket[bucket(f.Home)] + s.Rand.Between(0, 1)
	want := money + int64(price)*s.Attendance(0, f.Home, f.Away, 1, true, price)/2
	s.Rand = r0
	md.prepare(0, f)
	if got := s.Managers[0].Money; got != want {
		t.Errorf("away share %d, want %d", got-money, want-money)
	}
	if !md.humans[0].plays || !md.humans[1].plays || md.humans[0].side != 1 {
		t.Errorf("humans %+v", md.humans)
	}
}

// When the away manager forfeits, the home manager still gets his gate.
func TestAwayForfeitKeepsHomeGate(t *testing.T) {
	s, md, f := twoManagerTie(t)
	s.Managers[0].Effort = ForfeitEffort
	home := s.Managers[1].Money
	md.prepare(0, f)
	if f.Forfeit != 2 || f.Watched || md.humans[0].plays || md.humans[1].plays {
		t.Fatalf("forfeit %d watched %v humans %+v", f.Forfeit, f.Watched, md.humans)
	}
	if s.Managers[1].Money <= home || f.Attendance == 0 {
		t.Errorf("home manager got %d for %d spectators", s.Managers[1].Money-home, f.Attendance)
	}
}
