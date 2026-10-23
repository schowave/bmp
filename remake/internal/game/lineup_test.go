package game

import (
	"errors"
	"testing"
)

func lineupState(t *testing.T) *State {
	t.Helper()
	s, err := NewGame(mana(t), Options{Managers: []NewManager{{Name: "TEST", Team: 9}}, Level: 2, Seed: 5})
	if err != nil {
		t.Fatal(err)
	}
	s.Start()
	for i := range s.Squads[0] {
		s.Squads[0][i].Status &^= 3
	}
	s.SetFormation(0, 0)
	s.ManualLineup(0)
	return s
}

func findLineup(s *State, mi int, f func(*Slot) bool) int {
	for i := 0; i < SquadMax; i++ {
		if sl := &s.Squads[mi][i]; !sl.Empty() && f(sl) {
			return i
		}
	}
	return -1
}

func TestSetFormationBoard(t *testing.T) {
	s := lineupState(t)
	if got := s.countLineup(0, 1); got != 11 {
		t.Fatalf("team has %d players", got)
	}
	cells := map[[2]int]bool{}
	for i := 0; i < SquadMax; i++ {
		sl := &s.Squads[0][i]
		if sl.Empty() || sl.Lineup < 1 || sl.Lineup > 11 {
			continue
		}
		c := [2]int{sl.Col, sl.Row}
		if cells[c] {
			t.Fatalf("two players on cell %v", c)
		}
		cells[c] = true
		if s.PlayerAt(0, sl.Col, sl.Row) != i {
			t.Fatalf("PlayerAt(%v) != %d", c, i)
		}
	}
	// 4-4-2: the goalkeeper in the goal, the outer defenders one row up.
	for _, c := range [][2]int{{3, 7}, {1, 5}, {2, 6}, {4, 6}, {5, 5}} {
		if !cells[c] {
			t.Errorf("no player on %v", c)
		}
	}
}

func TestLineupClicks(t *testing.T) {
	s := lineupState(t)
	var vac BoardCell
	in := findLineup(s, 0, func(sl *Slot) bool { return sl.Lineup >= 1 && sl.Lineup <= 11 && sl.Row == 6 })
	out := findLineup(s, 0, func(sl *Slot) bool { return sl.Lineup == 0 })
	if in < 0 || out < 0 {
		t.Fatal("no players")
	}
	if err := s.LineupLeft(0, out, &vac); !errors.Is(err, ErrNotMoreThan) {
		t.Fatalf("12th player: %v", err)
	}
	col, row := s.Squads[0][in].Col, s.Squads[0][in].Row
	if err := s.LineupLeft(0, in, &vac); err != nil {
		t.Fatal(err)
	}
	if s.Squads[0][in].Lineup != 0 || !vac.Set {
		t.Fatal("player not taken out")
	}
	other := findLineup(s, 0, func(sl *Slot) bool { return sl.Lineup >= 1 && sl.Lineup <= 11 })
	if err := s.LineupLeft(0, other, &vac); !errors.Is(err, ErrNotLessThan) {
		t.Fatalf("9 players: %v", err)
	}
	if err := s.LineupLeft(0, out, &vac); err != nil {
		t.Fatal(err)
	}
	if sl := &s.Squads[0][out]; sl.Lineup < 1 || sl.Lineup > 11 || sl.Col != col || sl.Row != row || vac.Set {
		t.Fatalf("new player not on the free cell: %+v", *sl)
	}
	// Numbers 1..11 in slot order.
	n := 1
	for i := 0; i < SquadMax; i++ {
		if sl := &s.Squads[0][i]; !sl.Empty() && sl.Lineup >= 1 && sl.Lineup <= 11 {
			if sl.Lineup != n {
				t.Fatalf("slot %d has number %d, want %d", i, sl.Lineup, n)
			}
			n++
		}
	}
	// Right click: reserves up to 4, a left click takes a reserve out.
	for i := 0; i < SquadMax; i++ {
		if sl := &s.Squads[0][i]; sl.Lineup >= 12 {
			sl.Lineup = 0
		}
	}
	added := 0
	var err error
	for i := 0; i < SquadMax && err == nil; i++ {
		if sl := &s.Squads[0][i]; !sl.Empty() && sl.Lineup == 0 {
			if err = s.LineupRight(0, i); err == nil {
				added++
				if sl.Lineup != 11+added {
					t.Fatalf("reserve number %d, want %d", sl.Lineup, 11+added)
				}
			}
		}
	}
	if added != MaxReserves || !errors.Is(err, ErrMaxReserves) {
		t.Fatalf("added %d reserves, err %v", added, err)
	}
	r := findLineup(s, 0, func(sl *Slot) bool { return sl.Lineup == 12 })
	if err := s.LineupLeft(0, r, &vac); err != nil || s.Squads[0][r].Lineup != 0 {
		t.Fatalf("reserve not taken out: %v", err)
	}
	// Injured players and the automatic lineup refuse clicks.
	s.Squads[0][r].Status |= 2
	if err := s.LineupRight(0, r); !errors.Is(err, ErrInjuredShort) {
		t.Fatalf("injured: %v", err)
	}
	s.SetFormation(0, 1)
	if err := s.LineupLeft(0, out, &vac); !errors.Is(err, ErrAutoActive) {
		t.Fatalf("auto: %v", err)
	}
}

func TestMoveAndPlace(t *testing.T) {
	s := lineupState(t)
	a := s.PlayerAt(0, 3, 7)
	b := s.PlayerAt(0, 2, 6)
	if a < 0 || b < 0 {
		t.Fatal("no players")
	}
	s.MoveOnBoard(0, a, 2, 6)
	if s.PlayerAt(0, 2, 6) != a || s.PlayerAt(0, 3, 7) != b {
		t.Fatal("not swapped")
	}
	s.MoveOnBoard(0, a, 0, 0)
	if s.PlayerAt(0, 0, 0) != a || s.PlayerAt(0, 2, 6) >= 0 {
		t.Fatal("not moved")
	}
	sl := &s.Squads[0][a]
	sl.Col = -1
	s.PlaceOnBoard(0, a)
	want := 7 - 7*s.Persons[sl.Person].Pos/100
	if want == 7 {
		want = 6
	}
	if sl.Col < 0 || s.PlayerAt(0, sl.Col, sl.Row) != a {
		t.Fatalf("not placed: %+v", *sl)
	}
	n := 0
	for i := 0; i < SquadMax; i++ {
		if o := &s.Squads[0][i]; !o.Empty() && o.Lineup >= 1 && o.Lineup <= 11 && o.Col == sl.Col && o.Row == sl.Row {
			n++
		}
	}
	if n != 1 || sl.Row < want {
		t.Fatalf("placed on %d,%d (%d players there), row of the position %d", sl.Col, sl.Row, n, want)
	}
}
