package main

import (
	"time"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/schowave/bmp/remake/internal/ui"
)

// pace makes the frames come from a ticker instead of vsync. With vsync the
// main thread of macOS waits for the display outside the run loop, and
// AppKit then hands a mouse click to the game about 230 ms late, when the
// pointer may have left what was clicked. FPSModeVsyncOffMinimum waits for
// events inside the run loop; it also runs a frame on every input event, so
// the app ticks by the clock. The mode is deprecated but has no successor
// that waits in the run loop.
func pace(app *ui.App) {
	ebiten.SetFPSMode(ebiten.FPSModeVsyncOffMinimum) //nolint:staticcheck // deprecated without a successor, see above
	go func() {
		for range time.Tick(time.Second / ui.TPS) {
			ebiten.ScheduleFrame() //nolint:staticcheck // belongs to FPSModeVsyncOffMinimum
		}
	}()
	app.Paced = true
}
