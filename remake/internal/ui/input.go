package ui

import (
	"strconv"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/schowave/bmp/remake/internal/game"
)

// InputBox asks for a number or a text.
type InputBox struct {
	Title   string
	Prompt  string
	Value   string
	Numeric bool
	MaxLen  int
	Done    func(app *App, value string, ok bool)

	ok, cancel Button
}

// NewNumberInput asks for a number.
func NewNumberInput(title, prompt string, value int64, done func(app *App, v int64, ok bool)) *InputBox {
	b := &InputBox{Title: title, Prompt: prompt, Numeric: true, MaxLen: 9}
	if value != 0 {
		b.Value = strconv.FormatInt(value, 10)
	}
	b.Done = func(app *App, s string, ok bool) {
		v, err := strconv.ParseInt(s, 10, 64)
		if err != nil {
			v = 0
		}
		done(app, v, ok)
	}
	return b
}

// NewTextInput asks for a text.
func NewTextInput(title, prompt, value string, maxLen int, done func(app *App, v string, ok bool)) *InputBox {
	return &InputBox{Title: title, Prompt: prompt, Value: value, MaxLen: maxLen, Done: done}
}

func (b *InputBox) Update(app *App) error {
	in := &app.Input
	b.ok = Button{X: 95, Y: 128, W: 60, H: 12, Label: "OKAY"}
	b.cancel = Button{X: 165, Y: 128, W: 60, H: 12, Label: "ABBRUCH"}
	for _, c := range in.Chars {
		if len(b.Value) >= b.MaxLen {
			break
		}
		if b.Numeric && (c < '0' || c > '9') {
			continue
		}
		if c < 0x20 {
			continue
		}
		b.Value += string(c)
	}
	if in.Backspace && len(b.Value) > 0 {
		r := []rune(b.Value)
		b.Value = string(r[:len(r)-1])
	}
	switch {
	case in.Enter || b.ok.Hit(in):
		app.Pop()
		b.Done(app, b.Value, true)
	case in.Escape || in.Right || b.cancel.Hit(in):
		app.Pop()
		b.Done(app, b.Value, false)
	}
	return nil
}

func (b *InputBox) Draw(app *App, dst *ebiten.Image) {
	g := app.Gfx
	x, y, w, h := 70, 84, 180, 62
	FillRect(dst, x, y, w, h, g.PalColor(ColDarkRed))
	FrameRect(dst, x, y, w, h, g.PalColor(ColRed))
	g.TextCentered(dst, FontNormal, x, y+3, w, b.Title, g.PalColor(ColBeige))
	g.TextCentered(dst, FontSmall, x, y+16, w, b.Prompt, g.PalColor(ColBeige))
	FillRect(dst, x+20, y+27, w-40, 11, g.PalColor(ColBlack))
	v := b.Value
	if b.Numeric && v != "" {
		if n, err := strconv.ParseInt(v, 10, 64); err == nil {
			v = game.FormatMoney(n)
		}
	}
	cursor := ""
	if (app.frame/20)%2 == 0 {
		cursor = "_"
	}
	g.TextCentered(dst, FontNormal, x+20, y+29, w-40, v+cursor, g.PalColor(ColYellow))
	b.ok.Draw(g, dst, app.Input.In(b.ok.X, b.ok.Y, b.ok.W, b.ok.H))
	b.cancel.Draw(g, dst, app.Input.In(b.cancel.X, b.cancel.Y, b.cancel.W, b.cancel.H))
}
