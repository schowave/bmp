// Package ui draws the game with Ebitengine on a 320x240 canvas, the
// resolution of the original (VGA mode X), using the original pictures and
// fonts.
package ui

import (
	"fmt"
	"image"
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/schowave/bmp/remake/internal/assets"
)

// Screen size of the original game.
const (
	ScreenW = 320
	ScreenH = 240
)

// Gfx holds the decoded pictures and fonts as Ebitengine images.
type Gfx struct {
	dir   *assets.Dir
	pals  []assets.Palette
	pics  map[picKey]*ebiten.Image
	fonts map[string]*fontAtlas
}

type picKey struct {
	n    int
	full bool
	pal  int
	key  int
}

// NewGfx loads palettes and fonts from the game directory.
func NewGfx(dir *assets.Dir) (*Gfx, error) {
	pals, err := dir.Palettes()
	if err != nil {
		return nil, err
	}
	img, err := dir.MainImage()
	if err != nil {
		return nil, err
	}
	fonts, err := assets.Fonts(img)
	if err != nil {
		return nil, err
	}
	g := &Gfx{dir: dir, pals: pals, pics: map[picKey]*ebiten.Image{}, fonts: map[string]*fontAtlas{}}
	for name, f := range fonts {
		g.fonts[name] = newFontAtlas(f)
	}
	return g, nil
}

// Palette returns palette i of PIC/1.PAL.
func (g *Gfx) Palette(i int) *assets.Palette {
	if i < 0 || i >= len(g.pals) {
		i = 0
	}
	return &g.pals[i]
}

// Pic returns picture n (N.VGA, or N.CP when full) rendered with palette 0.
// It panics when the picture is missing: the game directory was checked at
// start-up and a missing picture is a programming error.
func (g *Gfx) Pic(n int, full bool) *ebiten.Image {
	return g.PicPal(n, full, 0, -1)
}

// PicKeyed returns picture n with palette index key transparent.
func (g *Gfx) PicKeyed(n int, key int) *ebiten.Image {
	return g.PicPal(n, false, 0, key)
}

// PicPal returns picture n rendered with palette pal; key < 0 means opaque.
func (g *Gfx) PicPal(n int, full bool, pal int, key int) *ebiten.Image {
	k := picKey{n, full, pal, key}
	if img, ok := g.pics[k]; ok {
		return img
	}
	p, err := g.dir.Pic(n, full)
	if err != nil {
		panic(fmt.Sprintf("picture %d: %v", n, err))
	}
	var rgba *image.RGBA
	if key >= 0 {
		rgba = p.RGBAKeyed(g.Palette(pal), key)
	} else {
		rgba = p.RGBA(g.Palette(pal))
	}
	img := ebiten.NewImageFromImage(rgba)
	g.pics[k] = img
	return img
}

// Sub returns the part r of picture n (sheets like 3.VGA hold many icons).
func (g *Gfx) Sub(n int, r image.Rectangle) *ebiten.Image {
	return g.Pic(n, false).SubImage(r).(*ebiten.Image)
}

// SubKeyed returns the part r of picture n with palette index key transparent.
func (g *Gfx) SubKeyed(n int, key int, r image.Rectangle) *ebiten.Image {
	return g.PicKeyed(n, key).SubImage(r).(*ebiten.Image)
}

// Draw draws img with its top-left corner at x, y.
func Draw(dst, img *ebiten.Image, x, y int) {
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Translate(float64(x), float64(y))
	dst.DrawImage(img, op)
}

// FillRect fills a rectangle with a solid colour.
func FillRect(dst *ebiten.Image, x, y, w, h int, c color.Color) {
	if w <= 0 || h <= 0 {
		return
	}
	dst.SubImage(image.Rect(x, y, x+w, y+h)).(*ebiten.Image).Fill(c)
}

// FrameRect draws a one-pixel rectangle outline.
func FrameRect(dst *ebiten.Image, x, y, w, h int, c color.Color) {
	FillRect(dst, x, y, w, 1, c)
	FillRect(dst, x, y+h-1, w, 1, c)
	FillRect(dst, x, y, 1, h, c)
	FillRect(dst, x+w-1, y, 1, h, c)
}

// PalColor returns colour index i of palette 0, the palette of the UI.
func (g *Gfx) PalColor(i int) color.RGBA {
	return g.pals[0][i&0xff]
}
