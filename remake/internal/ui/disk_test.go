package ui

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/schowave/bmp/remake/internal/game"
)

func diskSession(t *testing.T) (*App, *Session) {
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

func TestSaveAndLoadScreens(t *testing.T) {
	app, ss := diskSession(t)
	lastFileName = ""
	f := NewFileScreen(ss, true)
	app.Push(f)
	// A click on FILENAME starts typing; Enter saves NAME.MAN and closes.
	step(t, app, click(100, 178))
	if !f.typing {
		t.Fatal("not typing")
	}
	step(t, app, Input{Chars: []rune("abc d")})
	step(t, app, Input{Enter: true})
	if top(app) == Scene(f) {
		t.Fatal("save screen still open")
	}
	if _, err := os.Stat(filepath.Join(ss.SaveDir, "ABC_D.MAN")); err != nil {
		t.Fatal(err)
	}
	// Saving again under the name asks before it overwrites.
	f = NewFileScreen(ss, true)
	app.Push(f)
	if f.name != "ABC_D.MAN" {
		t.Fatalf("name %q", f.name)
	}
	step(t, app, Input{X: 100, Y: 178, Right: true})
	q, ok := top(app).(*MessageBox)
	if !ok || q.Buttons[0] != "ACH BITTE !" {
		t.Fatalf("no overwrite question: %T", top(app))
	}
	step(t, app, Input{Escape: true})
	if top(app) == Scene(f) {
		t.Fatal("save screen still open after BLOß NICHT")
	}

	// Load: the list shows ".", "..", the file; two clicks on it load it.
	l := NewFileScreen(ss, false)
	app.Push(l)
	if len(l.entries) != 3 || l.entries[2].show != "ABC_D.MAN" {
		t.Fatalf("entries %+v", l.entries)
	}
	old := ss.S
	l.name = ""
	y := fileListY + 2*fileRowH - 2
	step(t, app, click(20, y))
	if l.name != "ABC_D.MAN" {
		t.Fatalf("name %q", l.name)
	}
	step(t, app, click(20, y))
	if ss.S == old {
		t.Fatal("game not loaded")
	}

	// LÖSCHEN deletes after the question.
	l = NewFileScreen(ss, false)
	app.Push(l)
	step(t, app, click(l.buttons[7].X+5, l.buttons[7].Y+5))
	q, ok = top(app).(*MessageBox)
	if !ok || q.Buttons[0] != "MACH SCHON!" {
		t.Fatalf("no delete question: %T", top(app))
	}
	step(t, app, Input{Enter: true})
	if _, err := os.Stat(filepath.Join(ss.SaveDir, "ABC_D.MAN")); !os.IsNotExist(err) {
		t.Fatal("file not deleted")
	}

	// A file that is no save game gives the alert and closes the screen.
	_ = os.WriteFile(filepath.Join(ss.SaveDir, "X.TXT"), []byte("x"), 0o644)
	l = NewFileScreen(ss, false)
	app.Push(l)
	l.name = "X.TXT"
	step(t, app, Input{X: 100, Y: 178, Right: true})
	a, ok := top(app).(*AlertBox)
	if !ok || a.Lines[0] != "Kein" {
		t.Fatalf("no alert: %T", top(app))
	}
	step(t, app, Input{Enter: true})
	if top(app) == Scene(l) {
		t.Fatal("load screen still open")
	}
}

func TestOptionsScreen(t *testing.T) {
	app, ss := diskSession(t)
	o := NewOptionsScreen(ss)
	app.Push(o)
	d := &ss.S.Display
	// AUS of Ergebnisse 2.Liga, AN of Blenden.
	step(t, app, click(180+30, 54+5))
	step(t, app, click(238+5, 119+5))
	if d.Results[1] || !d.Results[0] || !d.Fades {
		t.Fatalf("display %+v", *d)
	}
	step(t, app, click(500, 160))
	step(t, app, click(230, 160))
	if d.Speed != 28 {
		t.Fatalf("speed %d", d.Speed)
	}
	step(t, app, click(285, 160))
	if d.Speed != maxSpeed {
		t.Fatalf("speed %d", d.Speed)
	}
	step(t, app, Input{Chars: []rune{'5'}})
	if ss.S.Level != 0 {
		t.Fatalf("level %d", ss.S.Level)
	}
	step(t, app, rclick())
	if top(app) != Scene(o) {
		t.Fatal("right click left the options")
	}
	step(t, app, click(MenuButton.X+5, MenuButton.Y+5))
	if top(app) == Scene(o) {
		t.Fatal("HAUPT MENU did not leave")
	}
}

func TestEnterHighscore(t *testing.T) {
	var hs []game.HighscoreEntry
	for i := 0; i < 20; i++ {
		hs = append(hs, game.HighscoreEntry{Manager: string(rune('A' + i)), Club: "X", Score: 100 - i})
	}
	// The same manager and club is replaced, even with fewer points.
	got, row := enterHighscore(hs, game.HighscoreEntry{Manager: "B", Club: "X", Score: 50})
	if row != 19 || got[19].Manager != "B" || got[1].Manager != "C" || hs[1].Manager != "B" {
		t.Fatalf("row %d %+v", row, got)
	}
	// A new one only replaces the last when it has more points.
	if _, row := enterHighscore(hs, game.HighscoreEntry{Manager: "Z", Club: "Y", Score: 81}); row != -1 {
		t.Fatalf("row %d", row)
	}
	got, row = enterHighscore(hs, game.HighscoreEntry{Manager: "Z", Club: "Y", Score: 95})
	if row != 6 || len(got) != 20 || got[19].Manager != "S" {
		t.Fatalf("row %d %+v", row, got)
	}
}
