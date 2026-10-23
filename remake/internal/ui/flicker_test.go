package ui

import "testing"

// A scene on top for a single frame that goes without any input flashes,
// and so does a passer drawn while it still passes: TestNoFlashingScenes
// plays two seasons with idle frames after every input and records the top
// scene of every frame, like the player would see it.
func TestNoFlashingScenes(t *testing.T) {
	app, mana := testApp(t)
	setup := NewSetup(mana, t.TempDir())
	app.Reset(setup)
	step(t, app, click(30, slotY+120))
	step(t, app, Input{Chars: []rune("TEST")})
	step(t, app, Input{Enter: true})
	step(t, app, click(30, slotY+20))
	step(t, app, rclick())
	step(t, app, rclick())
	ss := app.stack[0].(*Driver).ss
	type rec struct {
		sc    Scene
		input bool
	}
	var log []rec
	frame := func(in Input) {
		step(t, app, in)
		log = append(log, rec{top(app), in.Left || in.Right || in.Enter || in.Escape || len(in.Chars) > 0})
		if p, ok := top(app).(passer); ok && p.Passing() {
			t.Errorf("frame %d: %T drawn before it has passed", len(log), p)
		}
	}
	idle := func() {
		for range 4 {
			frame(Input{})
		}
	}
	opened := map[*MainMenu]bool{}
	endYear := ss.S.StartYear + 2
	for i := 0; i < 100000 && ss.S.StartYear < endYear; i++ {
		switch sc := top(app).(type) {
		case *MainMenu:
			if !opened[sc] && len(opened) < 3 {
				opened[sc] = true
				visitMenus(t, app, sc, frame, idle)
			}
			if sc.msgOpen {
				frame(click(msgButtonX+5, msgPanelY+msgFieldPitch+5))
				idle()
			}
			frame(click(260, headerPanelY+30))
		case *SquadScreen:
			frame(click(lnIconX+10, lnIconY[3]+10))
		case *MessageBox, *tsMessage, *AlertBox:
			frame(Input{Enter: true})
		case *tsPanel:
			frame(click(0xb8+5, 0xd8+5))
		case *ResultsScreen, *TableScreen, *CupScreen:
			frame(click(MenuButton.X+10, MenuButton.Y+10))
		default:
			frame(click(160, 120))
		}
		idle()
	}
	if ss.S.StartYear < endYear {
		t.Fatalf("stuck on %T in year %d", top(app), ss.S.StartYear)
	}
	for i := 1; i+1 < len(log); i++ {
		if log[i].sc != log[i-1].sc && log[i].sc != log[i+1].sc && !log[i+1].input {
			t.Errorf("frame %d: %T flashes between %T and %T", i, log[i].sc, log[i-1].sc, log[i+1].sc)
		}
	}
}

// visitMenus opens the submenu items of the turn menu m (without load,
// save and new managers) and leaves each again.
func visitMenus(t *testing.T, app *App, m *MainMenu, frame func(Input), idle func()) {
	t.Helper()
	groups := []struct {
		x int
		g *MenuGroup
	}{{menuLeftX, &m.Left[0]}, {menuLeftX, &m.Left[1]}, {menuLeftX, &m.Left[2]}, {menuRightX, &m.Right[0]}, {menuRightX, &m.Right[1]}}
	rows := []int{0, 1, 2, 0, 1}
	leave := []Input{click(MenuButton.X+10, MenuButton.Y+10), {Escape: true}, {Enter: true}, rclick()}
	for gi, gr := range groups {
		for k := range gr.g.Items {
			if !gr.g.available(k) || gi == 4 && k != 2 && k != 3 {
				continue
			}
			frame(click(gr.x+10, menuRowsY[rows[gi]]+10))
			idle()
			frame(click(submenuCols[k/3]+10, submenuRowsY[k%3]+10))
			idle()
			for n := 0; top(app) != Scene(m) && n < 20; n++ {
				frame(leave[n%len(leave)])
				idle()
			}
			if top(app) != Scene(m) {
				t.Fatalf("cannot leave %T (group %d item %d)", top(app), gi, k)
			}
		}
	}
}
