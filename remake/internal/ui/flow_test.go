package ui

import (
	"fmt"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/schowave/bmp/remake/internal/assets"
	"github.com/schowave/bmp/remake/internal/embedded"
	"github.com/schowave/bmp/remake/internal/game"
)

// testApp creates an app with the original assets built into the remake.
func testApp(t *testing.T) (*App, *assets.Mana) {
	t.Helper()
	dir, err := assets.OpenDir(embedded.FS())
	if err != nil {
		t.Fatal(err)
	}
	gfx, err := NewGfx(dir)
	if err != nil {
		t.Fatal(err)
	}
	mana, err := dir.Mana()
	if err != nil {
		t.Fatal(err)
	}
	return &App{Gfx: gfx}, mana
}

// step runs one frame of the top scene with the given input.
func step(t *testing.T, app *App, in Input) {
	t.Helper()
	app.Input = in
	app.frame++
	if len(app.stack) == 0 {
		t.Fatal("empty scene stack")
	}
	if err := app.stack[len(app.stack)-1].Update(app); err != nil {
		t.Fatal(err)
	}
}

func click(x, y int) Input { return Input{X: x, Y: y, Left: true} }
func rclick() Input        { return Input{X: 160, Y: 120, Right: true} }

func top(app *App) Scene { return app.stack[len(app.stack)-1] }

func TestSetupAndFirstTurns(t *testing.T) {
	app, mana := testApp(t)
	setup := NewSetup(mana, t.TempDir())
	app.Reset(setup)
	// Name of manager 1.
	step(t, app, click(30, slotY+120))
	if setup.edit != 0 {
		t.Fatalf("name field not active")
	}
	step(t, app, Input{Chars: []rune("TEST")})
	step(t, app, Input{Enter: true})
	if setup.names[0] != "TEST" {
		t.Fatalf("name %q", setup.names[0])
	}
	// Put the club in the middle of the strip into slot 1.
	step(t, app, click(30, slotY+20))
	if setup.teams[0] != setup.center {
		t.Fatalf("team %d", setup.teams[0])
	}
	step(t, app, rclick())
	if _, ok := top(app).(*setupOptions); !ok {
		t.Fatalf("no options: %T", top(app))
	}
	step(t, app, rclick())
	d, ok := top(app).(*Driver)
	if !ok {
		t.Fatalf("game did not start: %T", top(app))
	}
	ss := d.ss
	// Play a few events through the UI: close message boxes, end turns,
	// run match days.
	var last string
	same := 0
	for i := 0; i < 20000 && ss.S.Event < 12; i++ {
		// A scene that stays on top without anything moving is a screen
		// the test cannot leave: fail instead of hanging until the timeout.
		state := fmt.Sprintf("%p %d %d %d", top(app), len(app.stack), ss.S.Event, ss.S.Day)
		if ms, ok := top(app).(*MatchScreen); ok {
			state += fmt.Sprintf(" %d %d", ms.md.Minute, len(ms.shows))
		}
		if state == last {
			if same++; same > 500 {
				t.Fatalf("stuck on %T at event %d", top(app), ss.S.Event)
			}
		} else {
			last, same = state, 0
		}
		switch sc := top(app).(type) {
		case *Driver:
			step(t, app, Input{})
		case *MessageBox:
			step(t, app, Input{Enter: true})
		case *MainMenu:
			// open every submenu item once on the first turn
			if ss.S.Event == 0 && i < 300 {
				openAllMenus(t, app, sc)
			}
			if sc.msgOpen {
				// close the message panel with X
				step(t, app, click(msgButtonX+5, msgPanelY+msgFieldPitch+5))
			}
			// the calendar sheet goes on to the next date
			step(t, app, click(260, headerPanelY+30))
		case *MatchScreen:
			step(t, app, click(160, 120))
		case *SubstitutionScreen:
			step(t, app, click(MenuButton.X+10, MenuButton.Y+10))
		case *DrawShow:
			step(t, app, click(160, 120))
		case *NewspaperScene:
			step(t, app, click(160, 120))
		case *ResultsScreen, *TableScreen:
			step(t, app, click(MenuButton.X+10, MenuButton.Y+10))
		case *CupScreen:
			step(t, app, click(MenuButton.X+10, MenuButton.Y+10))
		default:
			t.Fatalf("unexpected scene %T", sc)
		}
	}
	if ss.S.Event < 12 {
		t.Fatalf("stuck at event %d phase %d", ss.S.Event, ss.S.Loop.Phase)
	}
	t.Logf("event %d, games %d, money %d", ss.S.Event, ss.S.Clubs[ss.S.Managers[0].Team].TotalGames(), ss.S.Managers[0].Money)
	if ss.S.Clubs[ss.S.Managers[0].Team].TotalGames() == 0 {
		t.Fatal("no league games played")
	}
}

// openAllMenus opens every submenu item and closes it again.
func openAllMenus(t *testing.T, app *App, m *MainMenu) {
	t.Helper()
	groups := []struct {
		x int
		g *MenuGroup
	}{{menuLeftX, &m.Left[0]}, {menuLeftX, &m.Left[1]}, {menuLeftX, &m.Left[2]}, {menuRightX, &m.Right[0]}, {menuRightX, &m.Right[1]}}
	rows := []int{0, 1, 2, 0, 1}
	for gi, gr := range groups {
		step(t, app, click(gr.x+10, menuRowsY[rows[gi]]+10))
		for k, it := range gr.g.Items {
			if it.Open == nil {
				continue
			}
			step(t, app, click(submenuCols[k/3]+10, submenuRowsY[k%3]+10))
			sc := top(app)
			if sc == Scene(m) {
				t.Fatalf("item %d of group %d did not open", k, gi)
			}
			img := newTestImage()
			sc.Draw(app, img)
			for n := 0; top(app) != Scene(m); n++ {
				if n > 100 {
					t.Fatalf("item %d of group %d does not close: %T", k, gi, top(app))
				}
				// screens close with the HAUPTMENÜ button, dialogs with Escape
				step(t, app, Input{Escape: true, Right: true, Left: true, X: MenuButton.X + 10, Y: MenuButton.Y + 10})
			}
		}
	}
}

var _ = game.PhaseTurn

func newTestImage() *ebiten.Image { return ebiten.NewImage(ScreenW, ScreenH) }
