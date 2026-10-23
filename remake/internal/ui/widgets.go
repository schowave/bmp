package ui

import (
	"github.com/hajimehoshi/ebiten/v2"

	"github.com/schowave/bmp/remake/internal/game"
)

// Colour indexes of the UI palette (PIC/1.PAL, palette 0).
const (
	ColBlack     = 0
	ColLight     = 1 // light grey-blue: normal text on blue
	ColGrey2     = 2
	ColGrey3     = 3
	ColGrey4     = 4
	ColGrey5     = 5
	ColGrey6     = 6
	ColGrey7     = 7
	ColBlue      = 8  // panel background
	ColSand      = 10 // text of the dialogs
	ColBeige     = 11
	ColDarkRed   = 16
	ColRedDark   = 17 // text of a button that is not pointed at
	ColRed       = 18
	ColOrange    = 19
	ColBlueLight = 22
	ColYellow    = 23
	ColWhite     = 29
)

// Panel draws a blue panel with the bevelled border of the original.
func (g *Gfx) Panel(dst *ebiten.Image, x, y, w, h int) {
	FillRect(dst, x, y, w, h, g.PalColor(ColBlue))
	FillRect(dst, x, y, w, 1, g.PalColor(ColGrey2))
	FillRect(dst, x, y, 1, h, g.PalColor(ColLight))
	FillRect(dst, x, y+h-1, w, 1, g.PalColor(ColGrey5))
	FillRect(dst, x+w-1, y, 1, h, g.PalColor(ColGrey5))
}

// Bevel fills a rectangle with colour fill and draws the bevel of the
// dialogs of the original: light left side, grey top, dark right side and
// bottom.
func (g *Gfx) Bevel(dst *ebiten.Image, x, y, w, h int, fill int) {
	FillRect(dst, x, y, w, h, g.PalColor(fill))
	FillRect(dst, x+1, y, w-2, 1, g.PalColor(ColGrey2))
	FillRect(dst, x, y, 1, h-1, g.PalColor(ColLight))
	FillRect(dst, x+w-1, y, 1, h-1, g.PalColor(ColGrey7))
	FillRect(dst, x, y+h-1, w, 1, g.PalColor(ColGrey6))
}

// Window draws a window of the original (f_06c7_00a7): a bevelled box
// from x0, y0 to x1, y1 inclusive with a black shadow of two pixels to the
// right and below.
func (g *Gfx) Window(dst *ebiten.Image, x0, y0, x1, y1 int, fill int) {
	w, h := x1-x0+1, y1-y0+1
	g.Bevel(dst, x0, y0, w, h, fill)
	FillRect(dst, x1+1, y0, 2, h+2, g.PalColor(ColBlack))
	FillRect(dst, x0+2, y1+1, w, 2, g.PalColor(ColBlack))
}

// BevelButton is the bevel of the buttons inside dialogs: like Bevel, but
// the four corners keep the colour of what lies below.
func (g *Gfx) BevelButton(dst *ebiten.Image, x, y, w, h int, fill int) {
	FillRect(dst, x+1, y+1, w-2, h-2, g.PalColor(fill))
	FillRect(dst, x+1, y, w-2, 1, g.PalColor(ColGrey2))
	FillRect(dst, x, y+1, 1, h-2, g.PalColor(ColLight))
	FillRect(dst, x+w-1, y+1, 1, h-2, g.PalColor(ColGrey7))
	FillRect(dst, x+1, y+h-1, w-2, 1, g.PalColor(ColGrey6))
}

// Frame draws the grey double frame around a panel.
func (g *Gfx) Frame(dst *ebiten.Image, x, y, w, h int) {
	FrameRect(dst, x-2, y-2, w+4, h+4, g.PalColor(ColGrey3))
	FrameRect(dst, x-1, y-1, w+2, h+2, g.PalColor(ColGrey6))
}

// Background draws the marble background.
func (g *Gfx) Background(dst *ebiten.Image) {
	Draw(dst, g.Pic(22, true), 0, 0)
}

// Heading draws a beige underlined heading.
func (g *Gfx) Heading(dst *ebiten.Image, x, y int, s string) {
	g.Text(dst, FontNormal, x, y, s, g.PalColor(ColBeige))
	FillRect(dst, x, y+9, g.TextWidth(FontNormal, s), 1, g.PalColor(ColBeige))
}

// Button is a clickable text button.
type Button struct {
	X, Y, W, H int
	Label      string
	Disabled   bool
}

// Draw draws the button: grey with a bevel, like the buttons of the dialogs.
func (b *Button) Draw(g *Gfx, dst *ebiten.Image, hot bool) {
	FillRect(dst, b.X, b.Y, b.W, b.H, g.PalColor(ColGrey4))
	FillRect(dst, b.X, b.Y, b.W, 1, g.PalColor(ColLight))
	FillRect(dst, b.X, b.Y, 1, b.H, g.PalColor(ColLight))
	FillRect(dst, b.X, b.Y+b.H-1, b.W, 1, g.PalColor(ColGrey7))
	FillRect(dst, b.X+b.W-1, b.Y, 1, b.H, g.PalColor(ColGrey7))
	c := g.PalColor(ColWhite)
	if hot {
		c = g.PalColor(ColYellow)
	}
	if b.Disabled {
		c = g.PalColor(ColGrey6)
	}
	g.TextCentered(dst, FontSmall, b.X, b.Y+(b.H-6)/2, b.W, b.Label, c)
}

// Hit reports a left click on the button.
func (b *Button) Hit(in *Input) bool {
	return !b.Disabled && in.Clicked(b.X, b.Y, b.W, b.H)
}

// MenuButton is the "HAUPTMENÜ" button at the bottom right of most screens.
var MenuButton = Button{X: 268, Y: 193, W: ButtonW, H: ButtonH}

// DrawMenuButton draws the HAUPTMENÜ icon button.
func (g *Gfx) DrawMenuButton(dst *ebiten.Image) {
	g.DrawIconButton(dst, IconMainMenu, MenuButton.X, MenuButton.Y)
}

// MenuButtonHit reports a click on the HAUPTMENÜ button (or Escape). Like
// in the original the right button does not leave a screen.
func MenuButtonHit(in *Input) bool {
	return MenuButton.Hit(in) || in.Escape
}

// Money formats an amount like the original: "1.500.000 DM".
func Money(v int64) string { return game.FormatMoney(v) + " DM" }

// Fit cuts s so that it fits into w pixels.
func (g *Gfx) Fit(font Font, s string, w int) string {
	r := []rune(s)
	for len(r) > 0 && g.TextWidth(font, string(r)) > w {
		r = r[:len(r)-1]
	}
	return string(r)
}

// Wrap breaks a text into lines of at most w pixels.
func (g *Gfx) Wrap(font Font, s string, w int) []string {
	var lines []string
	line := ""
	word := ""
	flush := func() {
		if word == "" {
			return
		}
		if line == "" {
			line = word
		} else if g.TextWidth(font, line+" "+word) <= w {
			line += " " + word
		} else {
			lines = append(lines, line)
			line = word
		}
		word = ""
	}
	for _, c := range s {
		switch c {
		case ' ':
			flush()
		case '\n':
			flush()
			lines = append(lines, line)
			line = ""
		default:
			word += string(c)
		}
	}
	flush()
	if line != "" {
		lines = append(lines, line)
	}
	return lines
}
