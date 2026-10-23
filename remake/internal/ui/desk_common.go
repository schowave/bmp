package ui

import (
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"
)

// numText is f_076b_0681: v in decimal with a dot every three digits
// (unless nodots, D:0x7b2), padded on the left to width characters (D:0x79c)
// with '^' (a blank of the width of a digit) and a '0' for the last digit.
// Negative numbers get a '-' and no padding.
func numText(v int64, width int, nodots bool) string {
	neg := v < 0
	if neg {
		v = -v
	}
	var r []byte // reversed
	for v != 0 {
		if n := len(r); !nodots && (n == 3 || n == 7 || n == 11) {
			r = append(r, '.')
		}
		r = append(r, byte('0'+v%10))
		v /= 10
	}
	if neg {
		r = append(r, '-')
	} else {
		for len(r) < width {
			if len(r) == 0 {
				r = append(r, '0')
			} else {
				r = append(r, '^')
			}
		}
	}
	for i, j := 0, len(r)-1; i < j; i, j = i+1, j-1 {
		r[i], r[j] = r[j], r[i]
	}
	return string(r)
}

// frameIncl draws the outline of the rectangle x0,y0 .. x1,y1 inclusive
// (f_06c7_0753).
func frameIncl(dst *ebiten.Image, x0, y0, x1, y1 int, c color.Color) {
	FrameRect(dst, x0, y0, x1-x0+1, y1-y0+1, c)
}

// fillIncl fills the rectangle x0,y0 .. x1,y1 inclusive (f_06c7_083c).
func fillIncl(dst *ebiten.Image, x0, y0, x1, y1 int, c color.Color) {
	if x1 < x0 || y1 < y0 {
		return
	}
	FillRect(dst, x0, y0, x1-x0+1, y1-y0+1, c)
}

// lineIncl draws a horizontal or vertical line inclusive (f_06c7_09b0).
func lineIncl(dst *ebiten.Image, x0, y0, x1, y1 int, c color.Color) {
	fillIncl(dst, min(x0, x1), min(y0, y1), max(x0, x1), max(y0, y1), c)
}

// deskVGA converts a colour of 6-bit VGA components like the palette loader.
func deskVGA(r, g, b int) color.RGBA {
	f := func(v int) uint8 { return uint8(v<<2 | v>>4) }
	return color.RGBA{f(r), f(g), f(b), 0xff}
}

// deskHeading draws a heading of the desk screens (f_262f_0af7 and
// others): beige text of the normal font with its baseline at y and a line
// under it at y+2 as long as the text.
func (g *Gfx) deskHeading(dst *ebiten.Image, x, y int, s string) {
	g.TextAt(dst, FontNormal, x, y, s, ColBeige, -1)
	lineIncl(dst, x, y+2, x+g.TextWidth(FontNormal, s), y+2, g.PalColor(ColBeige))
}
