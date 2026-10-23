package main

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/schowave/bmp/remake/internal/ui"
)

// startControl lets a test driver play the game: a script written to
// dir/cmd (see ui.ParseScript) is played and removed, and dir/done is
// written after its last step was drawn.
func startControl(app *ui.App, dir string, tps int) {
	ebiten.SetTPS(tps)
	ch := make(chan []ui.ScriptStep)
	app.Control = ch
	app.ControlDone = func() {
		_ = os.WriteFile(filepath.Join(dir, "done"), []byte(fmt.Sprintf("%T", app.Top())), 0o644)
	}
	cmd := filepath.Join(dir, "cmd")
	go func() {
		for range time.Tick(20 * time.Millisecond) {
			f, err := os.Open(cmd)
			if err != nil {
				continue
			}
			steps, err := ui.ParseScript(f)
			_ = f.Close()
			_ = os.Remove(cmd)
			if err != nil || len(steps) == 0 {
				fmt.Fprintln(os.Stderr, "bmp: control:", err)
				_ = os.WriteFile(filepath.Join(dir, "done"), nil, 0o644)
				continue
			}
			ch <- steps
		}
	}()
}
