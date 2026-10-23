package ui

import (
	"testing"

	"github.com/schowave/bmp/remake/internal/game"
)

func lineupApp(t *testing.T) (*App, *game.State, *SquadScreen) {
	t.Helper()
	app, mana := testApp(t)
	s, err := game.NewGame(mana, game.Options{Managers: []game.NewManager{{Name: "TEST", Team: 9}}, Level: 2, Seed: 5})
	if err != nil {
		t.Fatal(err)
	}
	s.Start()
	for i := range s.Squads[0] {
		s.Squads[0][i].Status &^= 3
	}
	ss := &Session{S: s, Mana: mana, SaveDir: t.TempDir()}
	app.Reset(NewDriver(ss))
	q := NewSquadScreen(ss)
	app.Push(q)
	return app, s, q
}

// cellXY is the middle of a cell of the tactics board.
func cellXY(col, row int) (int, int) { return col*16 + lnCellX + 6, row*14 + lnCellY + 6 }

func TestLineupBoard(t *testing.T) {
	app, s, q := lineupApp(t)
	step(t, app, click(lnIconX+10, lnIconY[1]+10))
	if q.view != lineupPitch {
		t.Fatal("pitch icon did not open the board")
	}
	// A formation button sets the automatic lineup.
	step(t, app, click(60+4+10, lnFormY+10))
	if s.Managers[0].AutoLineup != 3 {
		t.Fatalf("AutoLineup %d, want 3 (3-5-2)", s.Managers[0].AutoLineup)
	}
	gk := s.PlayerAt(0, 3, 7)
	if gk < 0 {
		t.Fatal("no goalkeeper on the board")
	}
	// Picking a player switches the automatic lineup off with a message.
	x, y := cellXY(3, 7)
	step(t, app, click(x, y))
	if _, ok := top(app).(*lineupToast); !ok || s.Managers[0].AutoLineup != 1 || q.picked != gk {
		t.Fatalf("pick: toast %T, auto %d, picked %d", top(app), s.Managers[0].AutoLineup, q.picked)
	}
	for i := 0; i < 100 && top(app) != Scene(q); i++ {
		step(t, app, Input{})
	}
	if top(app) != Scene(q) {
		t.Fatal("message did not close")
	}
	// Moving him to an empty cell, then swapping with another player.
	x, y = cellXY(0, 0)
	step(t, app, click(x, y))
	if s.PlayerAt(0, 0, 0) != gk || q.picked != -1 {
		t.Fatal("not moved to the empty cell")
	}
	other := s.PlayerAt(0, 1, 6)
	step(t, app, click(x, y))
	x, y = cellXY(1, 6)
	step(t, app, click(x, y))
	if s.PlayerAt(0, 1, 6) != gk || s.PlayerAt(0, 0, 0) != other {
		t.Fatal("not swapped")
	}
	// The list icon closes the board, HAUPT MENU the screen.
	step(t, app, click(lnIconX+10, lnIconY[0]+10))
	if q.view != lineupList {
		t.Fatal("board not closed")
	}
	step(t, app, click(lnIconX+10, lnIconY[3]+10))
	if top(app) == Scene(q) {
		t.Fatal("HAUPT MENU did not close the screen")
	}
}

func TestLineupListClicks(t *testing.T) {
	app, s, q := lineupApp(t)
	s.ManualLineup(0)
	in := -1
	for i := 0; i < game.SquadMax; i++ {
		if l := s.Squads[0][i].Lineup; l >= 1 && l <= 11 {
			in = i
			break
		}
	}
	rowY := func(i int) int { return lnRow0Y + 6*i - 2 }
	// A click on the name does nothing, one on the number takes him out.
	step(t, app, click(60, rowY(in)))
	if s.Squads[0][in].Lineup == 0 {
		t.Fatal("click on the name changed the lineup")
	}
	step(t, app, click(10, rowY(in)))
	if s.Squads[0][in].Lineup != 0 || !q.vac.Set {
		t.Fatal("click on the number did not take the player out")
	}
	// A right click makes him a reserve player (if there is room).
	for i := 0; i < game.SquadMax; i++ {
		if s.Squads[0][i].Lineup >= 12 {
			s.Squads[0][i].Lineup = 0
		}
	}
	step(t, app, rclick2(10, rowY(in)))
	if s.Squads[0][in].Lineup != 12 {
		t.Fatalf("right click: number %d, want 12", s.Squads[0][in].Lineup)
	}
	// The EINSATZ meter follows the mouse.
	step(t, app, click(0x113+10, 20))
	if s.Managers[0].EffortSetting != 10 {
		t.Fatalf("EffortSetting %d, want 10", s.Managers[0].EffortSetting)
	}
	// The contracts icon shows the contract list.
	step(t, app, click(lnIconX+10, lnIconY[2]+10))
	if q.view != lineupContracts || q.pressed() != 2 {
		t.Fatal("contracts icon did not switch the view")
	}
}

func rclick2(x, y int) Input { return Input{X: x, Y: y, Right: true} }
