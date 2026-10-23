package ui

import (
	"strconv"

	"github.com/hajimehoshi/ebiten/v2"
)

// Dialogs of the team screens: the four-line message with OKAY
// (f_3091_0e3a), the two-line notice that pops up at the top right and
// vanishes by itself (f_076b_0e57), the black dialog buttons
// (f_3091_123d) and the number field typed into a panel (f_076b_07d7).

// tsButton draws a dialog button (f_3091_123d): 57x12 pixels, black with
// a bevel, the label centred in colour c.
func tsButton(g *Gfx, dst *ebiten.Image, x, y int, label string, c int) {
	g.BevelButton(dst, x, y, 57, 12, ColBlack)
	g.TextMid(dst, FontSmall, x+1, x+57, y+8, label, c, NoShadow)
}

// tsButtonHit reports a left click on the dialog button at x, y.
func tsButtonHit(in *Input, x, y int) bool { return in.Clicked(x, y, 57, 12) }

// tsMessage is the dark red message box of f_3091_0e3a: four lines in the
// normal font and an OKAY button; only the button closes it.
type tsMessage struct {
	lines [4]string
	done  func(app *App)
}

// newTsMessage creates the message box.
func newTsMessage(l0, l1, l2, l3 string, done func(app *App)) *tsMessage {
	return &tsMessage{lines: [4]string{l0, l1, l2, l3}, done: done}
}

func (m *tsMessage) Update(app *App) error {
	if tsButtonHit(&app.Input, 0xb2, 0xa0) || app.Input.Enter {
		app.Pop()
		if m.done != nil {
			m.done(app)
		}
	}
	return nil
}

func (m *tsMessage) Draw(app *App, dst *ebiten.Image) {
	g := app.Gfx
	g.Window(dst, 0x3c, 0x6e, 0x3c+0xb9, 0x6e+0x41, ColDarkRed)
	for i, l := range m.lines {
		g.TextMid(dst, FontNormal, 0x3c, 0xf5, 0x7b+i*10, l, ColSand, NoShadow)
	}
	tsButton(g, dst, 0xb2, 0xa0, "OKAY", 26)
}

// tsToast is the notice of f_076b_0e57: a dark red box grows at the top
// right, shows two lines for a moment and disappears again.
type tsToast struct {
	l0, l1 string
	frame  int
	hold   int
	done   func(app *App)
}

const tsToastGrow = 16

// newTsToast creates the notice; done runs when it is gone.
func newTsToast(l0, l1 string, done func(app *App)) *tsToast {
	hold := 60
	if len(l0) >= 5 && l0[:5] == "Auto-" {
		hold = 30
	}
	return &tsToast{l0: l0, l1: l1, hold: hold, done: done}
}

func (t *tsToast) Update(app *App) error {
	t.frame++
	if t.frame > tsToastGrow+t.hold {
		app.Pop()
		if t.done != nil {
			t.done(app)
		}
	}
	return nil
}

func (t *tsToast) Draw(app *App, dst *ebiten.Image) {
	g := app.Gfx
	// The final box: an orange and a black frame around dark red.
	x0, y0, x1, y1 := 165, 13, 319, 44
	k := min(t.frame, tsToastGrow)
	if k < tsToastGrow {
		cx, cy := (x0+x1)/2, (y0+y1)/2
		x0 = cx - (cx-x0)*k/tsToastGrow
		x1 = cx + (x1-cx)*k/tsToastGrow
		y0 = cy - (cy-y0)*k/tsToastGrow
		y1 = cy + (y1-cy)*k/tsToastGrow
		FillRect(dst, x0, y0, x1-x0+1, y1-y0+1, g.C(ColDarkRed))
		return
	}
	FrameRect(dst, x0, y0, x1-x0+1, y1-y0+1, g.C(ColOrange))
	FrameRect(dst, x0+1, y0+1, x1-x0-1, y1-y0-1, g.C(ColBlack))
	FillRect(dst, x0+2, y0+2, x1-x0-3, y1-y0-3, g.C(ColDarkRed))
	g.TextMid(dst, FontNormal, 0xb9, 0x12f, 0x1a, t.l0, ColWhite, ColBlack)
	g.TextMid(dst, FontNormal, 0xb9, 0x12f, 0x27, t.l1, ColWhite, ColBlack)
}

// tsNumField is a number typed into a panel (f_076b_07d7): digits only,
// a block cursor after the text, Enter ends, Escape leaves it empty.
type tsNumField struct {
	x, y   int
	font   string
	color  int
	maxLen int
	text   string
	active bool
}

// start begins the input.
func (f *tsNumField) start(x, y, maxLen int, font string, color int) {
	*f = tsNumField{x: x, y: y, font: font, color: color, maxLen: maxLen, active: true}
}

// update handles the keys; it reports true when the input ended.
func (f *tsNumField) update(in *Input) bool {
	if !f.active {
		return false
	}
	for _, c := range in.Chars {
		if c >= '0' && c <= '9' && len(f.text) < f.maxLen {
			f.text += string(c)
		}
	}
	if in.Backspace && len(f.text) > 0 {
		f.text = f.text[:len(f.text)-1]
	}
	switch {
	case in.Escape:
		f.text = ""
		f.active = false
		return true
	case in.Enter:
		f.active = false
		return true
	}
	return false
}

// value returns the number typed, 0 for none.
func (f *tsNumField) value() int64 {
	v, _ := strconv.ParseInt(f.text, 10, 64)
	return v
}

func (f *tsNumField) draw(g *Gfx, dst *ebiten.Image) {
	g.TextAt(dst, f.font, f.x, f.y, f.text, f.color, NoShadow)
	if f.active {
		x := f.x + g.TextWidth(f.font, f.text)
		FillRect(dst, x+2, f.y-4, 3, 5, g.C(f.color))
	}
}
