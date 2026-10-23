package ui

import (
	"image/color"
	"strings"

	"github.com/hajimehoshi/ebiten/v2"
)

// DeskToast is the short message of f_076b_0e57 at the top right: a dark red
// box grows from the middle, shows two lines for about a second and is
// wiped away from the middle again. It blocks the input while it runs.
type DeskToast struct {
	Line1, Line2 string
	Done         func(app *App)
	// Pal is the palette of the screen below (1 on the sponsor screen).
	Pal int

	frame int
	auto  bool // texts starting with "Auto-" appear at once and stay shorter
	img   *ebiten.Image
}

// NewDeskToast creates the message.
func NewDeskToast(line1, line2 string) *DeskToast {
	return &DeskToast{Line1: line1, Line2: line2, auto: strings.HasPrefix(line1, "Auto-")}
}

// deskToastSteps is the number of growing rectangles: the height goes from 3
// to 35 in steps of 2.
const deskToastSteps = 16

func (t *DeskToast) steps() int {
	if t.auto {
		return 1
	}
	return deskToastSteps
}

// hold is the time the message stays: f_3930_01e7((2-auto)*30), a third
// of that in ticks of the 18.2 Hz timer, at 60 frames per second.
func (t *DeskToast) hold() int { return (2 - deskB2I(t.auto)) * 10 * 60 / 18 }

func deskB2I(b bool) int {
	if b {
		return 1
	}
	return 0
}

// rect returns the rectangle of step i: x, y, w, h.
func (t *DeskToast) rect(i int) (x, y, w, h int) {
	x, y, w, h = 0xb3, 0x1b, 7, 3
	if t.auto {
		x, y, w, h = 0xa4, 0xc, 0x9d, 0x21
	}
	return x - i, y - i, w + 10*i, h + 2*i
}

func (t *DeskToast) Update(app *App) error {
	t.frame++
	if t.frame >= 2*t.steps()+t.hold() {
		app.Pop()
		if t.Done != nil {
			t.Done(app)
		}
	}
	return nil
}

func (t *DeskToast) Draw(app *App, dst *ebiten.Image) {
	g := app.Gfx
	c := func(i int) color.RGBA { return g.Palette(t.Pal)[i] }
	if t.img == nil {
		t.img = ebiten.NewImage(ScreenW, ScreenH)
	}
	t.img.Clear()
	n := t.steps()
	open := min(t.frame+1, n)
	for i := 0; i < open; i++ {
		x, y, w, h := t.rect(i)
		fill6(t.img, x+1, y+1, x+w-1, y+h-1, c(ColDarkRed))
	}
	if t.frame >= n {
		x, y, w, h := t.rect(n)
		rect6(t.img, x+3, y+3, x+w-0xc, y+h-3, c(ColBlack))
		rect6(t.img, x+2, y+2, x+w-0xb, y+h-2, c(ColOrange))
		for i, l := range []string{t.Line1, t.Line2} {
			w := g.TextWidth(FontNormal, l)
			x := 0xb9 + (0x12f-0xb9)/2 - w/2
			top := 0x1a + i*0xd - g.baseline(FontNormal)
			g.Text(t.img, FontNormal, x+1, top, l, c(ColBlack))
			g.Text(t.img, FontNormal, x, top, l, c(ColWhite))
		}
	}
	// wiping: the saved background comes back in the same rectangles
	if wipe := t.frame - n - t.hold(); wipe >= 0 {
		for i := 0; i <= min(wipe, n-1); i++ {
			x, y, w, h := t.rect(i)
			deskClear(t.img, x, y, w, h)
		}
	}
	Draw(dst, t.img, 0, 0)
}

// deskClear makes a rectangle of img transparent.
func deskClear(img *ebiten.Image, x, y, w, h int) {
	if w <= 0 || h <= 0 {
		return
	}
	sub := img.SubImage(rectWH(x, y, w, h)).(*ebiten.Image)
	sub.Clear()
}
