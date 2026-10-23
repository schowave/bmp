package ui

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/schowave/bmp/remake/internal/game"
)

func TestScriptNamedTargetAndDump(t *testing.T) {
	steps, err := ParseScript(strings.NewReader("click menu.calendar\ndump x.MAN\nkey space\n"))
	if err != nil {
		t.Fatal(err)
	}
	if len(steps) != 5 || string(steps[3].In.Chars) != " " || steps[0].Target != "menu.calendar" || !steps[0].In.Left ||
		steps[1].Target != "menu.calendar" || steps[1].In.Left || steps[2].Dump != "x.MAN" {
		t.Fatalf("steps %+v", steps)
	}
}

// Every named target hits what its name says: the calendar ends the turn,
// a dialog button answers the dialog.
func TestControlTargets(t *testing.T) {
	app, mana := testApp(t)
	s, err := game.NewGame(mana, game.Options{Managers: []game.NewManager{{Name: "TEST", Team: 9}}, Level: 2, Seed: 1})
	if err != nil {
		t.Fatal(err)
	}
	s.Start()
	ss := &Session{S: s, Mana: mana, SaveDir: t.TempDir()}
	menu := NewTurnMenu(ss)
	app.stack = []Scene{NewDriver(ss), menu}
	app.Control = make(chan []ScriptStep) // more scripts may follow
	for _, name := range []string{"menu.desk", "menu.quit", "menu.item.1", "menu.item.9", "menu.messages",
		"messages.close", "button.menu", "lineup.exit"} {
		if _, _, ok := app.target(name); !ok {
			t.Errorf("no target %s", name)
		}
	}
	if _, _, ok := app.target("box.0"); ok {
		t.Error("box.0 without a dialog")
	}

	var ended bool
	menu.Calendar = func(*App) { ended = true }
	app.Script = []ScriptStep{{In: Input{Left: true, LeftHeld: true}, Target: "menu.calendar"}}
	if err := app.update(Input{}, app.next); err != nil {
		t.Fatal(err)
	}
	if !ended {
		t.Error("menu.calendar did not end the turn")
	}

	choice := -1
	box := &MessageBox{Title: "An Alle:", Lines: []string{"Frage"}, Buttons: []string{"JA", "NEIN"},
		Done: func(_ *App, c int) { choice = c }}
	app.Push(box)
	app.Script = []ScriptStep{{In: Input{Left: true, LeftHeld: true}, Target: "box.1"}}
	if err := app.update(Input{}, app.next); err != nil {
		t.Fatal(err)
	}
	if choice != 1 {
		t.Errorf("box.1 chose %d", choice)
	}

	if d := app.Describe(); !strings.Contains(d, "manager: TEST") || !strings.Contains(d, "money: ") {
		t.Errorf("describe:\n%s", d)
	}
	path := filepath.Join(t.TempDir(), "d.MAN")
	if err := app.dump(path); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := game.Load(data, mana); err != nil {
		t.Errorf("dump does not load: %v", err)
	}
}
