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
