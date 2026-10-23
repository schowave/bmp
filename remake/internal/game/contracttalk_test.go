package game

import "testing"

func TestCancelNegotiation(t *testing.T) {
	s := &State{}
	s.Squads[0][0].Nego = 102
	s.Squads[0][0].Message = true
	s.CancelNegotiation(0, 0)
	sl := s.Squads[0][0]
	if sl.Nego < 10 || sl.Nego > 18 || sl.Message {
		t.Fatalf("nego %d message %v", sl.Nego, sl.Message)
	}
}

func TestTopScorers(t *testing.T) {
	s := &State{}
	for p := 1; p < NumPersons; p++ {
		s.Persons[p].Team = 0xff
	}
	set := func(p, team, goals, games int) {
		s.Persons[p] = Person{Team: team, LeagueGoals: goals, LeagueGames: games}
	}
	set(1, 3, 5, 10)
	set(2, 4, 7, 10)
	set(3, 5, 5, 8)
	set(4, 20, 9, 9) // other league
	set(5, 6, 1, 3)  // fewer than two goals
	got := s.TopScorers(0, false)
	want := []int{2, 3, 1}
	if len(got) != len(want) {
		t.Fatalf("got %v", got)
	}
	for i, p := range want {
		if got[i].Person != p || got[i].Rank != i+1 {
			t.Fatalf("got %v, want persons %v", got, want)
		}
	}
}
