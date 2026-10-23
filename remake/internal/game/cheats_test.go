package game

import "testing"

func TestCheat(t *testing.T) {
	s := &State{Managers: []*Manager{{Money: 1000}}}
	s.Squads[0][3] = Slot{Person: 7, Tendency: 40, Fatigue: 90, Skill: [3]int{50, 60, 70}}
	if !s.Cheat(0, '1') || s.Level != 4 {
		t.Fatalf("level %d", s.Level)
	}
	if !s.Cheat(0, 'g') || s.Managers[0].Money != 201000 {
		t.Fatalf("money %d", s.Managers[0].Money)
	}
	s.Cheat(0, 's')
	sl := s.Squads[0][3]
	if sl.Tendency != 0x44 || sl.Fatigue != 0x32 || sl.Skill != [3]int{50, 60, 70} {
		t.Fatalf("s: %+v", sl)
	}
	if !s.Squads[0][0].Empty() || s.Squads[0][0].Tendency != 0 {
		t.Fatal("empty slot changed")
	}
	s.Cheat(0, 'a')
	if s.Squads[0][3].Skill != [3]int{99, 99, 99} || s.Managers[0].Money != 99999999 {
		t.Fatalf("a: %+v %d", s.Squads[0][3], s.Managers[0].Money)
	}
	if s.Cheat(0, 'x') {
		t.Fatal("x is no cheat")
	}
}
