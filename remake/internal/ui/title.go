package ui

import (
	"github.com/hajimehoshi/ebiten/v2"
)

// Title shows the title picture until a button is pressed.
type Title struct {
	next func() Scene
}

// NewTitle creates the title screen.
func NewTitle() *Title { return &Title{} }

// OnDone sets the scene that follows the title.
func (t *Title) OnDone(next func() Scene) *Title {
	t.next = next
	return t
}

func (t *Title) Update(app *App) error {
	in := &app.Input
	if in.Left || in.Right || in.Enter || in.Escape {
		if t.next != nil {
			app.Replace(t.next())
		} else {
			app.Quit()
		}
	}
	return nil
}

func (t *Title) Draw(app *App, dst *ebiten.Image) {
	// 47.CP uses the third palette of PIC/1.PAL.
	Draw(dst, app.Gfx.PicPal(47, true, 2, -1), 0, 0)
}
