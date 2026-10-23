package ui

import (
	"strings"

	"github.com/hajimehoshi/ebiten/v2"
)

// MessageBox is the dark red dialog of the original ("An Alle:" with a
// text and one or two buttons). It is pushed on top of the current scene.
type MessageBox struct {
	Title   string
	Lines   []string
	Buttons []string
	// Done is called with the index of the chosen button.
	Done func(app *App, choice int)

	buttons []Button
	layout  bool
}

// NewMessage creates a message box with an OK button.
func NewMessage(title, text string) *MessageBox {
	return &MessageBox{Title: title, Lines: []string{text}, Buttons: []string{"OKAY"}}
}

// NewQuestion creates a message box with two answers.
func NewQuestion(title, text, yes, no string, done func(app *App, yes bool)) *MessageBox {
	return &MessageBox{Title: title, Lines: []string{text}, Buttons: []string{yes, no},
		Done: func(app *App, c int) { done(app, c == 0) }}
}

// Geometry of the dialog of the original, measured on the DFB-Pokal
// question: 156 pixels wide, centred on y=143, a black shadow of two pixels
// to the right and below.
const (
	msgX = 75
	msgW = 156
	// msgCenterY is the vertical centre of the dialog.
	msgCenterY = 143
)

func (b *MessageBox) geometry(g *Gfx) (y, h int, lines []string) {
	for _, l := range b.Lines {
		for _, part := range strings.Split(l, "\n") {
			lines = append(lines, g.Wrap(FontSmall, part, msgW-8)...)
		}
	}
	h = 42 + len(lines)*8
	y = msgCenterY - h/2
	if !b.layout {
		b.layout = true
		// Two buttons sit 14 pixels from the left and right edge.
		const bw = 57
		xs := []int{msgX + (msgW-bw)/2}
		if len(b.Buttons) == 2 {
			xs = []int{msgX + 14, msgX + 84}
		}
		for i, s := range b.Buttons {
			b.buttons = append(b.buttons, Button{X: xs[i], Y: y + h - 16, W: bw, H: 12, Label: s})
		}
	}
	return y, h, lines
}

func (b *MessageBox) Update(app *App) error {
	b.geometry(app.Gfx)
	in := &app.Input
	choice := -1
	for i := range b.buttons {
		if b.buttons[i].Hit(in) {
			choice = i
		}
	}
	if choice < 0 && (in.Enter || (len(b.buttons) == 1 && (in.Right || in.Left))) {
		choice = 0
	}
	if choice < 0 && in.Escape {
		choice = len(b.buttons) - 1
	}
	if choice >= 0 {
		app.Pop()
		if b.Done != nil {
			b.Done(app, choice)
		}
	}
	return nil
}

func (b *MessageBox) Draw(app *App, dst *ebiten.Image) {
	g := app.Gfx
	y, h, lines := b.geometry(g)
	g.Bevel(dst, msgX, y, msgW, h, ColDarkRed)
	FillRect(dst, msgX+msgW, y, 2, h+2, g.PalColor(ColBlack))
	FillRect(dst, msgX+2, y+h, msgW, 2, g.PalColor(ColBlack))
	// Texts are centred over the dialog and its shadow.
	x0, x1 := msgX+2, msgX+msgW+1
	g.TextMid(dst, FontNormal, x0, x1, y+9, b.Title, ColBeige, NoShadow)
	FillRect(dst, msgX+2, y+11, msgW-4, 1, g.PalColor(ColBeige))
	for i, l := range lines {
		g.TextMid(dst, FontSmall, x0, x1, y+21+i*8, l, ColSand, NoShadow)
	}
	FillRect(dst, msgX+2, y+h-21, msgW-4, 1, g.PalColor(ColBeige))
	for i := range b.buttons {
		bt := &b.buttons[i]
		g.BevelButton(dst, bt.X, bt.Y, bt.W, bt.H, ColBlack)
		c := ColRedDark
		if app.Input.In(bt.X, bt.Y, bt.W, bt.H) {
			c = ColLight
		}
		g.TextMid(dst, FontSmall, bt.X, bt.X+bt.W, bt.Y+8, bt.Label, c, NoShadow)
	}
}

// ShowMessages shows a queue of messages one after another, then calls done.
func ShowMessages(app *App, title string, texts []string, done func(app *App)) {
	if len(texts) == 0 {
		if done != nil {
			done(app)
		}
		return
	}
	mb := NewMessage(title, texts[0])
	mb.Done = func(app *App, _ int) { ShowMessages(app, title, texts[1:], done) }
	app.Push(mb)
}
