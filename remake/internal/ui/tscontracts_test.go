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

// The key o fills in the offer of game.State.OptimalContract, or notes that
// there is none.
func TestNegotiationKey(t *testing.T) {
	app, mana := testApp(t)
	s, err := game.NewGame(mana, game.Options{Managers: []game.NewManager{{Name: "TEST", Team: 9}}, Level: 2, Seed: 5})
	if err != nil {
		t.Fatal(err)
	}
	s.Start()
	ss := &Session{S: s, Mana: mana, SaveDir: t.TempDir()}
	app.Reset(NewDriver(ss))
	sl := &s.Squads[0][0]
	sl.Nego, sl.Years = 0, 1

	p := newTsPanel(ss, 0, 0, tsPanelNormal, nil)
	app.Push(p)
	want := s.OptimalContract(0, 0, 0)
	step(t, app, Input{Chars: []rune("o")})
	if !want.OK || p.years != want.Years || p.salary != want.Salary || p.shownS != itoa64(want.Salary) {
		t.Fatalf("key o: years %d salary %d, want %+v", p.years, p.salary, want)
	}

	sl.Years = 4
	p = newTsPanel(ss, 0, 0, tsPanelNormal, nil)
	app.Push(p)
	step(t, app, Input{Chars: []rune("o")})
	if p.none == 0 || p.years != 0 || p.salary != 0 {
		t.Fatalf("key o without a sure offer: none %d years %d salary %d", p.none, p.years, p.salary)
	}
}

// A row of the contract list is blue grey while the player does not talk
// and dark red when he retires (f_1ecd_06af, mode 3).
func TestContractColor(t *testing.T) {
	for _, c := range []struct{ nego, want int }{
		{0, 21}, {1, ColGrey3}, {99, ColGrey3}, {102, 21}, {0x80, ColRedDark}, {0x85, ColRedDark},
	} {
		if got := contractColor(&game.Slot{Nego: c.nego}, 21); got != c.want {
			t.Errorf("nego %d: colour %d, want %d", c.nego, got, c.want)
		}
	}
}
