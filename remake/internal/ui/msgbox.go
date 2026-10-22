package ui

import (
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

const (
	msgX = 75
	msgW = 170
)

func (b *MessageBox) geometry(g *Gfx) (y, h int, lines []string) {
	for _, l := range b.Lines {
		lines = append(lines, g.Wrap(FontSmall, l, msgW-12)...)
	}
	h = 14 + len(lines)*8 + 22
	y = (ScreenH - h) / 2
	if !b.layout {
		b.layout = true
		n := len(b.Buttons)
		bw := 60
		gap := (msgW - n*bw) / (n + 1)
		for i, s := range b.Buttons {
			b.buttons = append(b.buttons, Button{X: msgX + gap + i*(bw+gap), Y: y + h - 16, W: bw, H: 11, Label: s})
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
	FillRect(dst, msgX, y, msgW, h, g.C(ColDarkRed))
	FrameRect(dst, msgX, y, msgW, h, g.C(ColRed))
	FrameRect(dst, msgX+1, y+1, msgW-2, h-2, g.C(ColBlack))
	g.TextCentered(dst, FontNormal, msgX, y+3, msgW, b.Title, g.C(ColBeige))
	FillRect(dst, msgX+3, y+12, msgW-6, 1, g.C(ColBeige))
	for i, l := range lines {
		g.TextCentered(dst, FontSmall, msgX, y+16+i*8, msgW, l, g.C(ColBeige))
	}
	for i := range b.buttons {
		bt := &b.buttons[i]
		FillRect(dst, bt.X, bt.Y, bt.W, bt.H, g.C(ColBlack))
		FrameRect(dst, bt.X, bt.Y, bt.W, bt.H, g.C(ColGrey3))
		c := g.C(ColRed)
		if app.Input.In(bt.X, bt.Y, bt.W, bt.H) {
			c = g.C(ColYellow)
		}
		g.TextCentered(dst, FontSmall, bt.X, bt.Y+3, bt.W, bt.Label, c)
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
