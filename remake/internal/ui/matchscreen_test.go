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

// WEITER on the results after the final whistle goes straight on to the
// table: the match screen does not come back in between.
func TestNoMatchScreenAfterFinalResults(t *testing.T) {
	app, s, _, _, m := matchApp(t)
	s.Display = game.DefaultDisplay()
	s.Display.GoalScenes = false
	// The results and tables are off in a new game; this test needs them.
	s.Display.Results = [game.NumLeagues]bool{true, true, true}
	s.Display.Table = [game.NumLeagues]bool{true, true, true}
	final := false
	for range 20000 {
		switch sc := top(app).(type) {
		case *MatchScreen:
			if final {
				t.Fatal("match screen on top after the final results")
			}
			step(t, app, click(160, 120))
		case *ResultsScreen:
			final = final || sc.minute >= 90
			step(t, app, click(MenuButton.X+10, MenuButton.Y+10))
		case *TableScreen, *CupScreen:
			step(t, app, click(MenuButton.X+10, MenuButton.Y+10))
		case *SquadScreen:
			step(t, app, click(lnIconX+10, lnIconY[3]+10))
		default:
			step(t, app, click(160, 120))
		}
		gone := true
		for _, sc := range app.stack {
			gone = gone && sc != Scene(m)
		}
		if gone {
			if !final {
				t.Fatal("no final results shown")
			}
			return
		}
	}
	t.Fatalf("stuck on %T", top(app))
}

// With the option Nachholspiele the results of the Nachholspiele follow
// each half (f_0310_2303 → f_2a41_120a(0, 2, minute, 0)).
func TestReplayResultsAfterEachHalf(t *testing.T) {
	app, s, _ := lineupApp(t)
	app.Pop()
	var md *game.MatchDay
	for i := 0; i < 2000 && md == nil; i++ {
		switch s.Loop.Phase {
		case game.PhaseTurn:
			s.TakeNotices()
			s.EndTurn()
		case game.PhaseMatches:
			if d := s.NextMatchDay(); d != nil && d.Replays() {
				md = d
			} else if d != nil {
				d.Run()
			}
		default:
			t.Fatalf("phase %v", s.Loop.Phase)
		}
	}
	if md == nil {
		t.Fatal("no day of Nachholspiele")
	}
	s.TakeNotices()
	s.Display.GoalScenes = false
	s.Display.Replays = true
	ss := &Session{S: s, SaveDir: t.TempDir()}
	m := NewMatchScreen(ss, md)
	app.Push(m)
	var minutes []int
	for range 20000 {
		switch sc := top(app).(type) {
		case *ResultsScreen:
			if sc.replay != md {
				t.Fatal("results of a league on a day of Nachholspiele")
			}
			if want := "NACHHOLSPIELE    " + itoa(sc.minute) + ".MINUTE"; sc.title() != want {
				t.Fatalf("title %q, want %q", sc.title(), want)
			}
			if len(sc.pairs()) != len(md.Fixtures) {
				t.Fatalf("%d pairs, want %d", len(sc.pairs()), len(md.Fixtures))
			}
			minutes = append(minutes, sc.minute)
			step(t, app, click(MenuButton.X+10, MenuButton.Y+10))
		case *SquadScreen:
			step(t, app, click(lnIconX+10, lnIconY[3]+10))
		case *MatchScreen:
			step(t, app, Input{})
		default:
			if len(minutes) == 0 {
				t.Fatalf("%T before the results", sc)
			}
			if len(minutes) != 2 || minutes[0] != 45 || minutes[1] != 90 {
				t.Fatalf("results at %v, want [45 90]", minutes)
			}
			return
		}
	}
	t.Fatalf("stuck on %T", top(app))
}
