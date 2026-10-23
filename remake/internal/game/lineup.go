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

// Tactics board of the lineup screen (f_1ecd_1560). The board has the
// columns 0..6 (left..right) and the rows 0 (forward line)..7 (goal); a
// player of the team stands on one cell (Slot.Col, Slot.Row).
const (
	BoardCols = 7
	BoardRows = 8
)

// LineupError is a message of the lineup screen: the original shows it
// in two lines (f_076b_0e57).
type LineupError [2]string

func (e LineupError) Error() string { return e[0] + " " + e[1] }

// Messages of the lineup screen (D:0x4d00.., D:0x27b4, D:0x4d70, D:0x4d24).
var (
	ErrAutoDisabled   = LineupError{"Auto-Aufstellung", "ist deaktiviert !"}
	ErrAutoActive     = LineupError{"Auto-Aufstellung", "ist aktiviert !"}
	ErrNotMoreThan    = LineupError{"Nicht mehr als 11", "Spieler einsetzen !"}
	ErrNotLessThan    = LineupError{"Nicht weniger als 10", "Spieler einsetzen !"}
	ErrMaxReserves    = LineupError{"Maximal 4", "Reservespieler !"}
	ErrSuspendedShort = LineupError{"Spieler ist", "GESPERRT."}
	ErrInjuredShort   = LineupError{"Spieler ist", "VERLETZT "}
)

// BoardCell is a cell of the tactics board. The lineup screen keeps the
// cell a player left when he was taken out of the team (f_1ecd_1560,
// registers SI and uVar21): the next player put into the team takes it.
type BoardCell struct {
	Col, Row int
	Set      bool
}

// PlayerAt returns the slot of the player of the team of manager mi on
// the cell col, row, or -1 (f_1ecd_11f9).
func (s *State) PlayerAt(mi, col, row int) int {
	for i := 0; i < SquadMax; i++ {
		sl := &s.Squads[mi][i]
		if !sl.Empty() && sl.Lineup >= 1 && sl.Lineup <= 11 && sl.Col == col && sl.Row == row {
			return i
		}
	}
	return -1
}

// PlaceOnBoard puts a player of the team without a cell (Col < 0) on the
// first free cell of the row of his position (f_1ecd_1266): row
// 7 - 7*Pos/100, but never the goal row; the columns 0..7 are tried, then
// the next rows.
func (s *State) PlaceOnBoard(mi, i int) {
	sl := &s.Squads[mi][i]
	if sl.Col >= 0 {
		return
	}
	row := 7 - 7*s.Persons[sl.Person].Pos/100
	if row == 7 {
		row = 6
	}
	for {
		for col := 0; col < 8; col++ {
			if s.PlayerAt(mi, col, row) < 0 {
				sl.Col, sl.Row = col, row
				return
			}
		}
		row++
		if row == 8 {
			row = 0
		}
	}
}

// MoveOnBoard moves the player of slot i to the cell col, row; a player
// standing there takes the old cell of i (f_1ecd_29c6).
func (s *State) MoveOnBoard(mi, i, col, row int) {
	sl := &s.Squads[mi][i]
	if o := s.PlayerAt(mi, col, row); o >= 0 {
		other := &s.Squads[mi][o]
		other.Col, other.Row = sl.Col, sl.Row
	}
	sl.Col, sl.Row = col, row
}

// lineupCheck returns the message shown before a click in the list is
// handled: the automatic lineup is on, or the player cannot play.
func (s *State) lineupCheck(mi int, sl *Slot) error {
	if s.Managers[mi].AutoLineup >= 2 {
		return ErrAutoActive
	}
	switch sl.Status & 3 {
	case 1:
		if !s.CupDay {
			return ErrSuspendedShort
		}
	case 2:
		return ErrInjuredShort
	}
	return nil
}

// LineupLeft handles a left click on the number of slot i in the lineup
// list (f_1ecd_1560): a player of the team or a reserve player is taken
// out (the team keeps at least 10 players; a player of the team leaves his
// cell in vac), a player not nominated goes into the team (at most 11) on
// the cell in vac or on a free cell of his position.
func (s *State) LineupLeft(mi, i int, vac *BoardCell) error {
	sl := &s.Squads[mi][i]
	if sl.Empty() {
		return nil
	}
	if err := s.lineupCheck(mi, sl); err != nil {
		return err
	}
	inTeam := s.countLineup(mi, 1)
	if sl.Lineup == 0 {
		if inTeam >= 11 {
			return ErrNotMoreThan
		}
		sl.Lineup = 11
		sl.Col = -1
		if vac.Set {
			sl.Col, sl.Row = vac.Col, vac.Row
			vac.Set = false
		}
		s.PlaceOnBoard(mi, i)
	} else {
		if sl.Lineup <= 11 {
			if inTeam < 11 {
				return ErrNotLessThan
			}
			*vac = BoardCell{Col: sl.Col, Row: sl.Row, Set: true}
		}
		sl.Lineup = 0
	}
	s.renumber(mi)
	return nil
}

// LineupRight handles a right click on the number of slot i in the lineup
// list (f_1ecd_1560): a player not nominated becomes a reserve player
// with the lowest free number.
func (s *State) LineupRight(mi, i int) error {
	sl := &s.Squads[mi][i]
	if sl.Empty() {
		return nil
	}
	if err := s.lineupCheck(mi, sl); err != nil {
		return err
	}
	if sl.Lineup != 0 {
		return nil
	}
	n := s.freeReserveNumber(mi)
	if n > 11+MaxReserves {
		return ErrMaxReserves
	}
	sl.Lineup = n
	sl.Col = -1
	return nil
}

// SetFormation switches the automatic lineup to formation f (0 4-4-2,
// 1 3-5-2, 2 3-4-3) and sets the lineup (formation buttons, f_1ecd_29c6).
func (s *State) SetFormation(mi, f int) {
	s.Managers[mi].AutoLineup = f + 2
	s.AutoLineup(mi)
}

// ManualLineup switches the automatic lineup off; it reports whether it
// was on (f_1ecd_14c7).
func (s *State) ManualLineup(mi int) bool {
	if s.Managers[mi].AutoLineup > 1 {
		s.Managers[mi].AutoLineup = 1
		return true
	}
	return false
}
