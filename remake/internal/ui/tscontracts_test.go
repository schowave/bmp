package ui

import (
	"testing"

	"github.com/schowave/bmp/remake/internal/game"
)

// ABBRUCH with nothing typed changes nothing during the season; with a
// typed length the player refuses to talk for 10..18 weeks
// (f_2277_2a8f).
func TestCancelNegotiation(t *testing.T) {
	app, mana := testApp(t)
	s, err := game.NewGame(mana, game.Options{Managers: []game.NewManager{{Name: "TEST", Team: 9}}, Level: 2, Seed: 5})
	if err != nil {
		t.Fatal(err)
	}
	s.Start()
	ss := &Session{S: s, Mana: mana, SaveDir: t.TempDir()}
	app.Reset(NewDriver(ss))
	sl := &s.Squads[0][0]
	sl.Nego = 0

	p := newTsPanel(ss, 0, 0, tsPanelNormal, nil)
	app.Push(p)
	before := s.Rand
	p.cancel(app)
	if s.Rand != before || sl.Nego != 0 || app.stack[len(app.stack)-1] == Scene(p) {
		t.Fatalf("empty ABBRUCH: nego %d, panel open %v", sl.Nego, app.stack[len(app.stack)-1] == Scene(p))
	}

	p = newTsPanel(ss, 0, 0, tsPanelNormal, nil)
	app.Push(p)
	p.years = 2
	p.cancel(app)
	if sl.Nego < 10 || sl.Nego > 18 {
		t.Errorf("ABBRUCH after typing: nego %d", sl.Nego)
	}

	// After the season the empty ABBRUCH is a refusal too.
	sl.Nego = 0
	s.Day = 330
	p = newTsPanel(ss, 0, 0, tsPanelSeason, nil)
	app.Push(p)
	p.cancel(app)
	if sl.Nego < 10 || sl.Nego > 18 {
		t.Errorf("ABBRUCH at the season end: nego %d", sl.Nego)
	}
}
