package game

// Lineup errors of the lineup screen (f_1ecd_1560).
var (
	ErrAutoOff     = CreditError("Auto-Aufstellung ist deaktiviert !")
	ErrTooMany     = CreditError("Nicht mehr als 11 Spieler einsetzen !")
	ErrTooManyRes  = CreditError("Maximal 4 Reservespieler !")
	ErrTooFew      = CreditError("Nicht weniger als 10 Spieler einsetzen !")
	ErrUnavailable = CreditError("Dieser Spieler kann nicht spielen.")
)

// ToggleLineup moves a player through "in the team" -> "reserve" -> "not
// nominated". A manual change switches the automatic lineup off.
func (s *State) ToggleLineup(mi, i int) error {
	m := s.Managers[mi]
	var warn error
	if m.AutoLineup > 1 {
		m.AutoLineup = 1
		warn = ErrAutoOff
	}
	sl := &s.Squads[mi][i]
	switch {
	case sl.Lineup == 0:
		if !s.available(sl) {
			return ErrUnavailable
		}
		if s.countLineup(mi, 1) >= 11 {
			if s.countLineup(mi, 2) >= MaxReserves {
				return ErrTooMany
			}
			sl.Lineup = s.freeReserveNumber(mi)
			break
		}
		sl.Lineup = 11
		s.placeManually(mi, sl)
	case sl.Lineup <= 11:
		if s.countLineup(mi, 1) < 11 {
			return ErrTooFew
		}
		if s.countLineup(mi, 2) >= MaxReserves {
			sl.Lineup = 0
			break
		}
		sl.Lineup = s.freeReserveNumber(mi)
	default:
		sl.Lineup = 0
	}
	s.renumber(mi)
	s.LineupStrength(mi, false)
	return warn
}

// freeReserveNumber returns the lowest reserve number not in use.
func (s *State) freeReserveNumber(mi int) int {
	for n := 12; ; n++ {
		used := false
		for i := 0; i < SquadMax; i++ {
			if sl := &s.Squads[mi][i]; !sl.Empty() && sl.Lineup == n {
				used = true
			}
		}
		if !used {
			return n
		}
	}
}

// placeManually puts a player on the tactics board by his position.
func (s *State) placeManually(mi int, sl *Slot) {
	pos := s.Persons[sl.Person].Position()
	sl.Row = formationRow[pos]
	used := map[int]bool{}
	for i := 0; i < SquadMax; i++ {
		o := &s.Squads[mi][i]
		if o != sl && !o.Empty() && o.Lineup >= 1 && o.Lineup <= 11 && o.Row == sl.Row {
			used[o.Col] = true
		}
	}
	for _, c := range []int{3, 1, 5, 2, 4, 0, 6} {
		if !used[c] {
			sl.Col = c
			return
		}
	}
	sl.Col = 3
}
