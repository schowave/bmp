package update

import (
	"strings"

	"github.com/schowave/bmp/remake/internal/assets"
)

// Logos are 38×38 like those of the original (PIC/80..143.VGA).
const logoSize = 38

// logoBackground is the palette index around the original logos.
const logoBackground = 29

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
