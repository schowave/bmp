package ui

import (
	"testing"

	"github.com/schowave/bmp/remake/internal/game"
)

func TestTeamInfoButtons(t *testing.T) {
	app, mana := testApp(t)
	s, err := game.NewGame(mana, game.Options{Managers: []game.NewManager{{Name: "TEST", Team: 9}}, Level: 2, Seed: 5})
	if err != nil {
		t.Fatal(err)
	}
	s.Start()
	ss := &Session{S: s, Mana: mana, SaveDir: t.TempDir()}
	app.Reset(NewDriver(ss))
	ti := NewTeamInfoMode(ss, 3, 2)
	app.Push(ti)
	// ANZEIGEN toggles the ticker flag.
	step(t, app, click(tiButtonX[2]+10, tiBtnY+5))
	if !s.Ticker(3) {
		t.Fatal("ANZEIGEN did not set the flag")
	}
	step(t, app, click(tiButtonX[2]+10, tiBtnY+5))
	if s.Ticker(3) {
		t.Fatal("ANZEIGEN did not clear the flag")
	}
	// RESTPROGRAMM, then INFO back, then REICHT MIR closes.
	step(t, app, click(tiButtonX[1]+10, tiBtnY+5))
	if !ti.rest {
		t.Fatal("RESTPROGRAMM not shown")
	}
	step(t, app, click(tiButtonX[2]+10, tiBtnY+5))
	if !ti.second {
		t.Fatal("RÜCKRUNDE not shown")
	}
	step(t, app, click(tiButtonX[1]+10, tiBtnY+5))
	if ti.rest {
		t.Fatal("INFO did not return")
	}
	step(t, app, click(tiButtonX[0]+10, tiBtnY+5))
	if top(app) == Scene(ti) {
		t.Fatal("REICHT MIR did not close")
	}
}
