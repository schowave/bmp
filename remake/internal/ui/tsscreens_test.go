package ui

import (
	"testing"

	"github.com/schowave/bmp/remake/internal/game"
)

func tsTestSession(t *testing.T) (*App, *Session) {
	t.Helper()
	app, mana := testApp(t)
	s, err := game.NewGame(mana, game.Options{Managers: []game.NewManager{{Name: "TEST", Team: 9}}, Level: 2, Seed: 5})
	if err != nil {
		t.Fatal(err)
	}
	s.Start()
	ss := &Session{S: s, Mana: mana, SaveDir: t.TempDir()}
	app.Reset(NewDriver(ss))
	return app, ss
}

func TestTrainingBalls(t *testing.T) {
	app, ss := tsTestSession(t)
	m := ss.M()
	m.Training = [4]int{2, 2, 2, 2}
	m.PosTraining = [4]int{2, 2, 2, 2}
	tr := NewTrainingScreen(ss)
	app.Push(tr)
	// The fourth ball of Kondition.
	step(t, app, click(0x1f+3*16+5, 0x22+8))
	if m.Training[0] != 4 {
		t.Fatalf("Kondition %d", m.Training[0])
	}
	// More than the 20 points are refused.
	m.Training = [4]int{4, 6, 5, 5}
	step(t, app, click(0x1f+9*16+5, 0x22+0x25+8))
	if m.Training[1] != 6 {
		t.Fatalf("Spiel %d", m.Training[1])
	}
	// Intensity and position training.
	step(t, app, click(0x1f+6*16+5, 0xc4))
	if m.Intensity != 7 {
		t.Fatalf("Intensität %d", m.Intensity)
	}
	step(t, app, click(0xd2+3*16+5, 0x22+8))
	if m.PosTraining[0] != 4 {
		t.Fatalf("Tor %d", m.PosTraining[0])
	}
	step(t, app, click(0xd9+10, 0xd0))
	if m.Youth != 10 {
		t.Fatalf("Jugend %d", m.Youth)
	}
	step(t, app, click(MenuButton.X+5, MenuButton.Y+5))
	if top(app) == Scene(tr) {
		t.Fatal("HAUPT MENU did not close")
	}
}

func TestScorerToggle(t *testing.T) {
	app, ss := tsTestSession(t)
	sc := NewScorerScreen(ss).(*ScorerScreen)
	app.Push(sc)
	step(t, app, click(0xd4+5, 0xaf+5))
	if !sc.players {
		t.Fatal("SPIELER not selected")
	}
	step(t, app, click(0x36+5, 0xaf+5))
	if sc.players {
		t.Fatal("LIGA not selected")
	}
}

func TestTransferListAndTakeBack(t *testing.T) {
	app, ss := tsTestSession(t)
	s := ss.S
	mi := ss.Mgr()
	tr := NewTransferScreen(ss)
	app.Push(tr)
	n, nm := s.SquadSize(mi), s.SquadSize(game.MarketIndex)
	// A click on the first own player lists him.
	step(t, app, click(30, 0x24+2))
	if s.SquadSize(mi) != n-1 || s.SquadSize(game.MarketIndex) != nm+1 {
		t.Fatalf("not listed: %d %d", s.SquadSize(mi), s.SquadSize(game.MarketIndex))
	}
	// Find him on the market and take him back.
	for i := 0; i < s.SquadSize(game.MarketIndex); i++ {
		if s.Persons[s.Squads[game.MarketIndex][i].Person].Owner == mi {
			step(t, app, click(200, 0x1d+(i+1)*6+2))
			break
		}
	}
	if s.SquadSize(mi) != n {
		t.Fatalf("not taken back: %d", s.SquadSize(mi))
	}
	// LEIHEN switches to loans.
	step(t, app, click(0xaf+5, 0x6e+5))
	if !tr.loan {
		t.Fatal("LEIHEN not selected")
	}
}

func TestContractPanel(t *testing.T) {
	app, ss := tsTestSession(t)
	s := ss.S
	mi := ss.Mgr()
	c := NewContractScreen(ss, false)
	app.Push(c)
	c.negotiate(app, 0)
	p, ok := top(app).(*tsPanel)
	if !ok {
		t.Fatalf("no panel but %T", top(app))
	}
	// ABBRUCH makes the player unwilling to talk.
	step(t, app, click(0xb8+5, 0xd8+5))
	if top(app) == Scene(p) {
		t.Fatal("ABBRUCH did not close")
	}
	if n := s.Squads[mi][0].Nego; n < 10 || n > 18 {
		t.Fatalf("nego %d", n)
	}
	c.negotiate(app, 0)
	if _, ok := top(app).(*tsMessage); !ok {
		t.Fatalf("no message but %T", top(app))
	}
	// Typing a contract length proposes the wanted salary.
	c.negotiate(app, 1)
	p = top(app).(*tsPanel)
	step(t, app, click(20, 0xd6))
	step(t, app, Input{Chars: []rune{'3'}})
	step(t, app, Input{Enter: true})
	if p.years != 3 || p.salary != s.ContractHint(mi, 1, 3) {
		t.Fatalf("years %d salary %d", p.years, p.salary)
	}
}
