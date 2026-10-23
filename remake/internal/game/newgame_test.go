package game

import "testing"

// A manager added with NEU gets 2/5/8/5 free players of the person ranges
// of the positions (f_08bc_2184), and only a new 1-year game puts a club
// into the UEFA-Pokal.
func TestAddManagerGetsSquad(t *testing.T) {
	s, err := NewGame(mana(t), Options{Managers: []NewManager{{Name: "ERST", Team: 9}}, Level: 2, Seed: 4, Length: OneYear})
	if err != nil {
		t.Fatal(err)
	}
	s.Start()
	if s.Managers[0].CupStatus[CupUEFA] != 1 {
		t.Fatalf("first manager not in the UEFA-Pokal: %v", s.Managers[0].CupStatus)
	}
	free := 0
	for p := 1; p < NumPersons; p++ {
		if s.Persons[p].Owner == OwnerFree {
			free++
		}
	}
	team := -1
	for id := 0; id < NumGerman && team < 0; id++ {
		if s.ManagerOf(id) == nil && LeagueOf(id) == 0 {
			team = id
		}
	}
	if err := s.AddManager("SPÄT", team, 0); err != nil {
		t.Fatal(err)
	}
	m := s.Managers[1]
	if m.CupStatus[CupUEFA] != NotInCup {
		t.Errorf("late manager in the UEFA-Pokal: %v", m.CupStatus)
	}
	n := 0
	for i := 0; i < SquadMax; i++ {
		sl := &s.Squads[1][i]
		if sl.Empty() {
			continue
		}
		p := sl.Person
		if s.Persons[p].Owner != 1 {
			t.Errorf("person %d owned by %d", p, s.Persons[p].Owner)
		}
		n++
		if sl.Salary == 0 || sl.Years != 1 {
			t.Errorf("slot %d: salary %d, %d years", i, sl.Salary, sl.Years)
		}
	}
	if free >= 40 && n != 20 {
		t.Errorf("%d players, want 20", n)
	}
}
