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
// written after its last step was drawn, with what the game shows
// (ui.App.Describe). tools/playtest.sh is such a driver.
func startControl(app *ui.App, dir string, tps int) {
	ebiten.SetTPS(tps)
	ch := make(chan []ui.ScriptStep)
	app.Control = ch
	done := func(text string) {
		// Renamed into place: the driver never reads half a file.
		tmp := filepath.Join(dir, "done.tmp")
		if os.WriteFile(tmp, []byte(text), 0o644) == nil {
			_ = os.Rename(tmp, filepath.Join(dir, "done"))
		}
	}
	app.ControlDone = func() { done(app.Describe()) }
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
			if err == nil && len(steps) == 0 {
				err = fmt.Errorf("empty")
			}
			if err != nil {
				done(fmt.Sprintf("error: script: %v\n", err))
				continue
			}
			ch <- steps
		}
	}()
}
