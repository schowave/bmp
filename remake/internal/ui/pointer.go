package ui

import (
	"image"

	"github.com/hajimehoshi/ebiten/v2"
)

// pointerW and pointerH are the size of the mouse pointer; its hot spot is
// the top-left pixel.
const pointerW, pointerH = 11, 11

// pointerPixels is the arrow of the original (D:0x7e4, chosen by
// f_076b_02a3 with flag 1), palette indices with 0 transparent.
var pointerPixels = [pointerW * pointerH]byte{
	17, 19, 0, 0, 0, 0, 0, 0, 0, 0, 0,
	18, 17, 19, 19, 0, 0, 0, 0, 0, 0, 0,
	0, 18, 17, 17, 19, 19, 0, 0, 0, 0, 0,
	0, 18, 17, 17, 17, 17, 19, 19, 0, 0, 0,
	0, 0, 18, 17, 17, 17, 17, 17, 19, 19, 0,
	0, 0, 18, 17, 17, 17, 17, 17, 17, 17, 0,
	0, 0, 0, 18, 17, 17, 17, 19, 0, 0, 0,
	0, 0, 0, 18, 17, 17, 18, 17, 19, 0, 0,
	0, 0, 0, 0, 18, 17, 0, 18, 17, 19, 0,
	0, 0, 0, 0, 18, 17, 0, 0, 18, 17, 19,
	0, 0, 0, 0, 0, 0, 0, 0, 0, 18, 17,
}

// pointerRGBA returns the arrow in the colours of the UI palette.
func (g *Gfx) pointerRGBA() *image.RGBA {
	img := image.NewRGBA(image.Rect(0, 0, pointerW, pointerH))
	for i, c := range pointerPixels {
		if c != 0 {
			img.Set(i%pointerW, i/pointerW, g.PalColor(int(c)))
		}
	}
	return img
}

// drawPointer draws the mouse pointer with its hot spot at x, y, as the
// original's driver does (f_35ac_17a0, keyed blit f_35ac_20c8). Outside the
// canvas, where the system shows its own pointer, it draws nothing.
func (a *App) drawPointer(dst *ebiten.Image, x, y int) {
	if x < 0 || y < 0 || x >= ScreenW || y >= ScreenH {
		return
	}
	if a.pointer == nil {
		a.pointer = ebiten.NewImageFromImage(a.Gfx.pointerRGBA())
	}
	Draw(dst, a.pointer, x, y)
}
