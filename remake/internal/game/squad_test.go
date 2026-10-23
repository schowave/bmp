package game

import "testing"

// squadOf puts persons with the given position codes and strength into the
// squad of manager 0.
func squadOf(pos []int, strength int) *State {
	s := &State{Managers: []*Manager{{AutoLineup: 2}}}
	for i, p := range pos {
		s.Persons[i+1].Pos = p
		sl := &s.Squads[0][i]
		sl.Person = i + 1
		sl.Skill = [3]int{strength, strength, strength}
	}
	return s
}

// f_1ecd_3635 starts with a best value of 0: the first spare goalkeeper
// offered for a field position scores -1 and is never taken, of more
// goalkeepers the last one.
func TestPickBestSpareGoalkeeper(t *testing.T) {
	s := squadOf([]int{0, 0}, 50)
	s.Squads[0][0].Lineup = 1
	if i := s.pickBest(0, PosMidfield); i != -1 {
		t.Errorf("lone spare goalkeeper taken: slot %d", i)
	}

	s = squadOf([]int{0, 0, 0, 0}, 50)
	s.Squads[0][0].Lineup = 1
	if i := s.pickBest(0, PosMidfield); i != 3 {
		t.Errorf("slot %d, want the last goalkeeper 3", i)
	}

	// A forward is taken before the goalkeeper.
	s = squadOf([]int{0, 0, 80}, 0)
	s.Squads[0][0].Lineup = 1
	if i := s.pickBest(0, PosMidfield); i != 2 {
		t.Errorf("slot %d, want the forward 2", i)
	}
}

// f_0cb5_25d5 swaps the columns of two players in a row when the other
// player's side fits the column better.
func TestSwapSides(t *testing.T) {
	s := squadOf([]int{30, 30, 30}, 50)
	sides := []int{6, 0, 3}
	cols := []int{0, 6, 3}
	for i := range sides {
		s.Persons[i+1].Side = sides[i]
		sl := &s.Squads[0][i]
		sl.Lineup, sl.Row, sl.Col = i+1, 6, cols[i]
	}
	s.swapSides(0)
	for i, want := range []int{6, 0, 3} {
		if got := s.Squads[0][i].Col; got != want {
			t.Errorf("player %d in column %d, want %d", i, got, want)
		}
	}

	// Players of another row or on the bench stay.
	s.Squads[0][1].Row = 3
	s.Squads[0][0].Col, s.Squads[0][1].Col = 0, 6
	s.Squads[0][2].Lineup = 12
	s.swapSides(0)
	if s.Squads[0][0].Col != 0 || s.Squads[0][1].Col != 6 {
		t.Errorf("columns %d %d", s.Squads[0][0].Col, s.Squads[0][1].Col)
	}
}
