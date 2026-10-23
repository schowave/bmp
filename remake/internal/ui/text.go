package ui

import (
	"image"
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/schowave/bmp/remake/internal/assets"
	"github.com/schowave/bmp/remake/internal/game"
)

// Font is the file name of a font of the original.
type Font string

// The fonts of the original.
const (
	FontNormal Font = "flow.font"    // 8 pixels, most text
	FontSmall  Font = "micro4.font"  // 6 pixels, small print
	FontBig    Font = "newbold.font" // 16 pixels, headlines
)

// fontAtlas keeps the glyphs of a font as white pixels on one image, so text
// can be drawn in any colour with a colour scale.
type fontAtlas struct {
	font   *assets.Font
	atlas  *ebiten.Image
	glyphs map[byte]image.Rectangle
}

func newFontAtlas(f *assets.Font) *fontAtlas {
	width := 0
	for c := 0x20; c < 0x7f; c++ {
		width += f.Glyphs[c-0x20].W + 1
	}
	rgba := image.NewRGBA(image.Rect(0, 0, width, f.Height))
	a := &fontAtlas{font: f, glyphs: map[byte]image.Rectangle{}}
	x := 0
	// One cell per glyph of the font, keyed by its own character.
	for c := 0x20; c < 0x7f; c++ {
		g := &f.Glyphs[c-0x20]
		for y, row := range g.Rows {
			for gx, on := range row {
				if on {
					rgba.Set(x+gx, y, color.White)
				}
			}
		}
		a.glyphs[byte(c)] = image.Rect(x, 0, x+g.W, f.Height)
		x += g.W + 1
	}
	a.atlas = ebiten.NewImageFromImage(rgba)
	return a
}

// TextWidth returns the width of s in pixels.
func (g *Gfx) TextWidth(font Font, s string) int {
	a := g.fonts[font]
	w := 0
	for _, c := range game.ToCP437(s) {
		w += a.font.Glyph(c).W
	}
	return w
}

// FontHeight returns the height of a font.
func (g *Gfx) FontHeight(font Font) int {
	return g.fonts[font].font.Height
}

// Text draws s with its top-left corner at x, y.
func (g *Gfx) Text(dst *ebiten.Image, font Font, x, y int, s string, c color.Color) {
	a := g.fonts[font]
	r, gr, b, al := c.RGBA()
	op := &ebiten.DrawImageOptions{}
	op.ColorScale.Scale(float32(r)/0xffff, float32(gr)/0xffff, float32(b)/0xffff, float32(al)/0xffff)
	for _, ch := range game.ToCP437(s) {
		cell := a.cache(ch)
		op.GeoM.Reset()
		op.GeoM.Translate(float64(x), float64(y))
		dst.DrawImage(a.atlas.SubImage(cell).(*ebiten.Image), op)
		x += cell.Dx()
	}
}

// TextCentered draws s centred horizontally in [x, x+w), with y the top of
// the text. It rounds (w - width)/2 down, one pixel left of TextMid when w
// is even and the width odd, and has no clamp at the screen edge: the
// screens it draws were matched with the original that way. Text placed
// like f_06c7_087c uses TextMid.
func (g *Gfx) TextCentered(dst *ebiten.Image, font Font, x, y, w int, s string, c color.Color) {
	g.Text(dst, font, x+(w-g.TextWidth(font, s))/2, y, s, c)
}

// TextCenteredShadow draws s centred like TextCentered, over a shadow one
// pixel to the right, like the calendar sheet of the original.
func (g *Gfx) TextCenteredShadow(dst *ebiten.Image, font Font, x, y, w int, s string, c, shadow color.Color) {
	g.TextCentered(dst, font, x+1, y, w, s, shadow)
	g.TextCentered(dst, font, x, y, w, s, c)
}

// TextRight draws s right-aligned so that it ends at x.
func (g *Gfx) TextRight(dst *ebiten.Image, font Font, x, y int, s string, c color.Color) {
	g.Text(dst, font, x-g.TextWidth(font, s), y, s, c)
}

func (a *fontAtlas) cache(c byte) image.Rectangle {
	return a.glyphs[a.font.Cell(c)]
}

// Text drawing like the original (f_06c7_087c, f_06c7_07d5): the y given is
// the baseline, the row of the bottom of the capitals, and a shadow in
// another colour lies one pixel to the right. Colours are palette indices;
// NoShadow leaves the shadow out.

// NoShadow is the shadow colour for text without shadow.
const NoShadow = -1

// baseline returns the row of the bottom of the capitals of a font.
func (g *Gfx) baseline(font Font) int {
	a := g.fonts[font]
	gl := a.font.Glyph('A')
	for r := len(gl.Rows) - 1; r >= 0; r-- {
		for _, on := range gl.Rows[r] {
			if on {
				return r
			}
		}
	}
	return a.font.Height - 1
}

// TextAt draws s left-aligned at x with its baseline at y.
func (g *Gfx) TextAt(dst *ebiten.Image, font Font, x, y int, s string, c, shadow int) {
	top := y - g.baseline(font)
	if shadow >= 0 {
		g.Text(dst, font, x+1, top, s, g.PalColor(shadow))
	}
	g.Text(dst, font, x, top, s, g.PalColor(c))
}

// TextMid draws s centred between x0 and x1 with its baseline at y.
func (g *Gfx) TextMid(dst *ebiten.Image, font Font, x0, x1, y int, s string, c, shadow int) {
	g.TextAt(dst, font, midX(x0, x1, g.TextWidth(font, s)), y, s, c, shadow)
}

// midX is where f_06c7_087c starts a text of width w centred between x0
// and x1: x0 + (x1-x0)/2 - w/2 with C division, and at the left edge when
// the text, one pixel further right, would leave the screen.
func midX(x0, x1, w int) int {
	x := x0 + (x1-x0)/2 - w/2
	if x+1+w > 0x13f || x+1 < 0 {
		return 0
	}
	return x
}

// TextEnd draws s right-aligned so that it ends before x, baseline at y.
func (g *Gfx) TextEnd(dst *ebiten.Image, font Font, x, y int, s string, c, shadow int) {
	g.TextAt(dst, font, x-g.TextWidth(font, s), y, s, c, shadow)
}
