package game

// Cheat applies a hidden key of the options screen (f_262f_01b8) for
// manager m and reports whether the key is one of them:
//
//	'0'..'5' Spiel-Level (Level = 5 - digit)
//	's'      every player of the squad gets tendency 68 and fatigue 50
//	'a'      like 's', plus all three skills 99 and 99,999,999 DM
//	'g'      200,000 DM more
//	'd', 'p' known keys whose effects the remake has no state for: 'd'
//	         clears the table of loaded save games (the original refuses to
//	         load a game saved the same day), 'p' sets D:0x5f5
func (s *State) Cheat(m int, key rune) bool {
	switch {
	case key >= '0' && key <= '5':
		s.Level = 5 - int(key-'0')
	case key == 's' || key == 'a':
		// The original walks the first 24 of the 25 slots.
		for i := 0; i < SquadMax; i++ {
			sl := &s.Squads[m][i]
			if sl.Empty() {
				continue
			}
			sl.Tendency = 0x44
			sl.Fatigue = 0x32
			if key == 'a' {
				sl.Skill = [3]int{99, 99, 99}
				s.Managers[m].Money = 99999999
			}
		}
	case key == 'g':
		s.Managers[m].Money += 200000
	case key == 'd' || key == 'p':
	default:
		return false
	}
	return true
}
