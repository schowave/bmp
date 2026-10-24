package update

import (
	"image"
	"image/color"
	_ "image/gif"  // logos the owner provides
	_ "image/jpeg" // logos the owner provides
	_ "image/png"  // logos the owner provides
	"math"

	"github.com/schowave/bmp/remake/internal/assets"
)

// logoColors are the palette indices the original logos use.
var logoColors = []uint8{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20, 22, 23, 24, 25, 28, 29, 30}

// ConvertLogo scales img into a 38×38 picture in the colours of the
// original logos, centred on their background.
func ConvertLogo(img image.Image, pal *assets.Palette) *assets.Pic {
	b := img.Bounds()
	w, h := b.Dx(), b.Dy()
	box := float64(logoSize - 2)
	scale := math.Min(box/float64(w), box/float64(h))
	tw, th := int(math.Round(float64(w)*scale)), int(math.Round(float64(h)*scale))
	ox, oy := (logoSize-tw)/2, (logoSize-th)/2
	p := &assets.Pic{W: logoSize, H: logoSize, Pix: make([]byte, logoSize*logoSize)}
	for i := range p.Pix {
		p.Pix[i] = logoBackground
	}
	for y := 0; y < th; y++ {
		for x := 0; x < tw; x++ {
			// The source pixels under the target pixel, weighted by alpha.
			x0, x1 := float64(x)/scale, float64(x+1)/scale
			y0, y1 := float64(y)/scale, float64(y+1)/scale
			var r, g, bl, a, n float64
			for sy := int(y0); sy < int(math.Ceil(y1)) && sy < h; sy++ {
				for sx := int(x0); sx < int(math.Ceil(x1)) && sx < w; sx++ {
					cr, cg, cb, ca := img.At(b.Min.X+sx, b.Min.Y+sy).RGBA()
					r += float64(cr)
					g += float64(cg)
					bl += float64(cb)
					a += float64(ca)
					n++
				}
			}
			if n == 0 || a/n < 0x8000 {
				continue
			}
			// Colours are premultiplied: divide by the alpha.
			c := color.RGBA{uint8(r / a * 255), uint8(g / a * 255), uint8(bl / a * 255), 255}
			p.Pix[(oy+y)*logoSize+ox+x] = nearest(c, pal)
		}
	}
	return p
}

// nearest is the logo colour closest to c. It counts in integers: floats
// may fuse a*b+c on one CPU and not on another, and a tie would then pick
// another colour.
func nearest(c color.RGBA, pal *assets.Palette) uint8 {
	best, bd := logoColors[0], math.MaxInt
	for _, i := range logoColors {
		p := pal[i]
		dr, dg, db := int(c.R)-int(p.R), int(c.G)-int(p.G), int(c.B)-int(p.B)
		// Plain distance: weighted by brightness a saturated colour like
		// light blue would turn grey.
		d := dr*dr + dg*dg + db*db
		if d < bd {
			best, bd = i, d
		}
	}
	return best
}
