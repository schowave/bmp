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

// The offer of the contract key is accepted with every draw, and 100 DM
// less is refused with some; it does not use the generator.
func TestOptimalContract(t *testing.T) {
	s, err := NewGame(mana(t), Options{Managers: []NewManager{{Name: "TEST", Team: 9}}, Level: 2, Seed: 5})
	if err != nil {
		t.Fatal(err)
	}
	s.Start()
	seeds := uint32(300)
	check := func(si, y int) {
		t.Helper()
		before := s.Rand
		oc := s.OptimalContract(0, si, y)
		if s.Rand != before {
			t.Fatal("OptimalContract used the generator")
		}
		if !oc.OK {
			return
		}
		sl := &s.Squads[0][si]
		saved := *sl
		refused := false
		for seed := uint32(1); seed <= seeds; seed++ {
			s.Rand.Seed = seed
			if err := s.NegotiateContract(0, si, oc.Years, oc.Salary, 0); err != nil {
				t.Fatalf("slot %d: %d years for %d refused with seed %d: %v", si, oc.Years, oc.Salary, seed, err)
			}
			*sl = saved
			s.Rand.Seed = seed
			if s.NegotiateContract(0, si, oc.Years, oc.Salary-100, 0) != nil {
				refused = true
			}
			*sl = saved
		}
		if !refused && oc.Salary-100 >= saved.Salary {
			t.Errorf("slot %d: %d years for %d DM less also accepted", si, oc.Years, oc.Salary-100)
		}
	}
	n := 0
	for si := 0; si < SquadSlots; si++ {
		sl := &s.Squads[0][si]
		if sl.Empty() || sl.Loan != 0 {
			continue
		}
		sl.Nego, sl.Years = 0, 1
		if s.OptimalContract(0, si, 0).OK {
			n++
		}
		check(si, 0)
		check(si, 2)
	}
	if n == 0 {
		t.Fatal("no player of the squad has a sure offer")
	}

	// At the season end the age counts, so the highest threshold is drawn
	// less often, and an expired contract has 0 years.
	s.Day = 330
	seeds = 5000
	for si := 0; si < SquadSlots; si++ {
		if sl := &s.Squads[0][si]; !sl.Empty() && sl.Loan == 0 {
			sl.Years = 0
			check(si, 0)
			check(si, 1)
		}
	}

	// A star asks too much for 4 years at the season end and gets the
	// longest length below it.
	sl := &s.Squads[0][0]
	sl.Skill, sl.Nego = [3]int{99, 99, 99}, 0
	if s.OptimalContract(0, 0, 4).OK {
		t.Error("a player asking 100,000 DM or more has a sure offer")
	}
	if oc := s.OptimalContract(0, 0, 0); !oc.OK || s.OptimalContract(0, 0, oc.Years+1).OK {
		t.Errorf("star at the season end: %+v", oc)
	}
	s.Day = 100

	sl.Years, sl.Nego = 1, 103
	if oc := s.OptimalContract(0, 0, 0); !oc.OK || oc.Years != 3 || oc.Salary != sl.Salary {
		t.Errorf("player asking for 3 years: %+v, salary %d", oc, sl.Salary)
	}
	if s.OptimalContract(0, 0, 2).OK {
		t.Error("player asking for 3 years takes 2")
	}
	sl.Nego = 0
	if s.OptimalContract(0, 0, 1).OK {
		t.Error("the current length is no extension")
	}
}
