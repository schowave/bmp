package ui

import "testing"

// After booking a camp the camp icon shows the cobweb at once, not only
// from the next turn.
func TestCampShutAfterBooking(t *testing.T) {
	_, s, _ := lineupApp(t)
	ss := &Session{S: s, SaveDir: t.TempDir()}
	m := NewTurnMenu(ss)
	if !m.Left[2].available(4) {
		t.Fatal("camp shut before booking")
	}
	s.Managers[0].Money = 1 << 30
	booked := false
	for c := range 8 {
		if s.BookCamp(0, c) == nil {
			booked = true
			break
		}
	}
	if !booked {
		t.Skip("no camp open")
	}
	if m.Left[2].available(4) {
		t.Fatal("camp still open after booking")
	}
}
