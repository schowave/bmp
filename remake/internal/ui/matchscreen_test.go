package ui

import (
	"testing"

	"github.com/schowave/bmp/remake/internal/game"
)

// A goal counts on the scoreboard only when its show reaches the goal
// picture, not while "Chance für X !" blinks before it.
func TestScoreboardWaitsForGoalPicture(t *testing.T) {
	app, s, _, md, m := matchApp(t)
	s.Display.GoalScenes = false
	for range 20000 {
		if len(m.shows) > 0 {
			sh := m.shows[0]
			k := sh.ev.Kind
			if sh.ticker == "" && (k == game.EvGoal || k == game.EvPenaltyGoal) {
				fi := sh.ev.Fixture
				h, a := md.Score(fi)
				sh2, sa := m.shownScore(fi)
				if sh2+sa != h+a-1 {
					t.Fatalf("during the chance: shown %d:%d, score %d:%d", sh2, sa, h, a)
				}
				for !sh.resultShown() {
					step(t, app, Input{})
				}
				if sh2, sa = m.shownScore(fi); sh2 != h || sa != a {
					t.Fatalf("with the goal picture: shown %d:%d, score %d:%d", sh2, sa, h, a)
				}
				return
			}
		}
		if md.Done && len(m.shows) == 0 {
			break
		}
		step(t, app, Input{})
	}
	t.Skip("no goal of a human match")
}

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
