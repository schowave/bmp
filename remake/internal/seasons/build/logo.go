package build

import (
	"crypto/sha256"
	"encoding/hex"
	"image"
	"image/color"
	_ "image/gif"  // logos of Wikipedia
	_ "image/jpeg" // logos of Wikipedia
	_ "image/png"  // logos of Wikipedia
	"math"
	"regexp"
	"strings"

	"github.com/schowave/bmp/remake/internal/assets"
)

// Logos are 38×38 like those of the original (PIC/80..143.VGA).
const logoSize = 38

// logoBackground is the palette index around the original logos.
const logoBackground = 29

// logoColors are the palette indices the original logos use.
var logoColors = []uint8{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20, 22, 23, 24, 25, 28, 29, 30}

var fileChars = regexp.MustCompile(`[^A-Za-z0-9._-]+`)

// LogoFileName is the base name under which the fetcher stores the logo
// file title ("Datei:Logo 1 FC Kaiserslautern.svg"): in lower case with a
// hash of the title, since titles that differ only in case are different
// files, but one file on a case-insensitive file system.
func LogoFileName(file string) string {
	_, name, ok := strings.Cut(file, ":")
	if !ok {
		name = file
	}
	sum := sha256.Sum256([]byte(file))
	return strings.ToLower(strings.Trim(fileChars.ReplaceAllString(name, "_"), "_")) + "-" + hex.EncodeToString(sum[:4])
}

// convertLogo scales img into a 38×38 picture in the colours of the
// original logos, centred on their background.
func convertLogo(img image.Image, pal *assets.Palette) *assets.Pic {
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

// nearest is the logo colour closest to c.
func nearest(c color.RGBA, pal *assets.Palette) uint8 {
	best, bd := logoColors[0], math.MaxFloat64
	for _, i := range logoColors {
		p := pal[i]
		dr, dg, db := float64(c.R)-float64(p.R), float64(c.G)-float64(p.G), float64(c.B)-float64(p.B)
		// Weighted like the eye: green counts most.
		d := 0.3*dr*dr + 0.59*dg*dg + 0.11*db*db
		if d < bd {
			best, bd = i, d
		}
	}
	return best
}

// colorWords map colour names of the infoboxes to palette indices.
var colorWords = []struct {
	word string
	idx  uint8
}{
	{"schwarz", 0}, {"weiß", 29}, {"weiss", 29}, {"dunkelblau", 8}, {"hellblau", 1}, {"himmelblau", 1},
	{"blau", 22}, {"weinrot", 16}, {"bordeaux", 16}, {"rot", 17}, {"gelb", 23}, {"gold", 24},
	{"dunkelgrün", 12}, {"grün", 14}, {"orange", 19}, {"violett", 30}, {"lila", 30}, {"braun", 21},
	{"silber", 2}, {"grau", 4},
}

// crestColors reads up to two colours from "Schwarz-Gelb", "Rot und Weiß".
func crestColors(colors string) []uint8 {
	var out []uint8
	for _, word := range strings.FieldsFunc(strings.ToLower(colors), func(r rune) bool {
		return r == '-' || r == '/' || r == ',' || r == ' '
	}) {
		for _, cw := range colorWords {
			if strings.HasSuffix(word, cw.word) || word == cw.word {
				out = append(out, cw.idx)
				break
			}
		}
		if len(out) == 2 {
			break
		}
	}
	switch len(out) {
	case 0:
		return []uint8{22, 29}
	case 1:
		return []uint8{out[0], 29}
	}
	return out
}

// crest draws a plain shield in the colours of a club, for a club without
// a free logo: split vertically, with a black outline.
func crest(colors string) *assets.Pic {
	cs := crestColors(colors)
	p := &assets.Pic{W: logoSize, H: logoSize, Pix: make([]byte, logoSize*logoSize)}
	for i := range p.Pix {
		p.Pix[i] = logoBackground
	}
	const top, bottom, left, right = 4, 34, 7, 31
	inside := func(x, y int) bool {
		if y < top || y > bottom || x < left || x > right {
			return false
		}
		// The lower third narrows to a point.
		if y > 24 {
			d := (y - 24) * 12 / (bottom - 24)
			return x >= left+d && x <= right-d
		}
		return true
	}
	for y := 0; y < logoSize; y++ {
		for x := 0; x < logoSize; x++ {
			if !inside(x, y) {
				continue
			}
			c := cs[0]
			if x > (left+right)/2 {
				c = cs[1]
			}
			if !inside(x-1, y) || !inside(x+1, y) || !inside(x, y-1) || !inside(x, y+1) {
				c = 0
			}
			p.Pix[y*logoSize+x] = c
		}
	}
	return p
}
