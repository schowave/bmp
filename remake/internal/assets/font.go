package assets

import (
	"encoding/binary"
	"fmt"
)

// The graphics library of BMMAIN.EXE keeps its bitmap fonts in its own code
// segment: a count at fontTable, followed by near pointers to font headers.
// A header is {name pointer, height, spacing, 95 glyph pointers for the
// characters 0x20..0x7e}. A glyph is a width byte followed by height rows of
// (width+7)/8 bytes, most significant bit = leftmost pixel.
const (
	gfxSegment  = 0x35ac
	fontTable   = 0x3fc
	charRemap   = 0x3eb
	firstGlyph  = 0x20
	glyphCount  = 0x7f - firstGlyph
	headerWords = 3
)

// Font is one of the bitmap fonts of the game.
type Font struct {
	Name    string
	Height  int
	Spacing int
	Glyphs  [glyphCount]Glyph
	remap   map[byte]byte
}

// Glyph is the bitmap of one character.
type Glyph struct {
	W    int
	Rows [][]bool
}

// Fonts extracts the fonts embedded in the unpacked BMMAIN.EXE image, keyed
// by name ("flow.font", "micro4.font", "newbold.font"). The font named
// "normal.font" has no glyphs of its own and is left out.
func Fonts(img *Image) (map[string]*Font, error) {
	base := gfxSegment * 16
	if base+0x10000 > len(img.Data) {
		return nil, fmt.Errorf("image too small for the font segment")
	}
	seg := img.Data[base : base+0x10000]
	w := func(off int) int { return int(binary.LittleEndian.Uint16(seg[off:])) }
	remap := map[byte]byte{}
	for i := 0; i < int(seg[charRemap]); i++ {
		remap[seg[charRemap+1+2*i]] = seg[charRemap+2+2*i]
	}
	n := w(fontTable)
	if n == 0 || n > 16 {
		return nil, fmt.Errorf("unexpected font count %d", n)
	}
	fonts := map[string]*Font{}
	for i := 0; i < n; i++ {
		h := w(fontTable + 2 + 2*i)
		name := cString(seg, w(h))
		if name == "normal.font" {
			continue
		}
		f := &Font{Name: name, Height: w(h + 2), Spacing: w(h + 4), remap: remap}
		if f.Height <= 0 || f.Height > 32 {
			return nil, fmt.Errorf("font %q: bad height %d", name, f.Height)
		}
		for c := 0; c < glyphCount; c++ {
			g := w(h + 2*headerWords + 2*c)
			width := int(seg[g])
			bpr := (width + 7) / 8
			rows := make([][]bool, f.Height)
			for r := range rows {
				rows[r] = make([]bool, width)
				for x := 0; x < width; x++ {
					b := seg[g+1+r*bpr+x/8]
					rows[r][x] = b&(0x80>>(x%8)) != 0
				}
			}
			f.Glyphs[c] = Glyph{W: width, Rows: rows}
		}
		fonts[name] = f
	}
	return fonts, nil
}

// Glyph returns the glyph for a character of the game's 8-bit text. CP437
// umlauts are mapped like the original does; unknown characters render as
// a space.
func (f *Font) Glyph(c byte) *Glyph {
	if c < firstGlyph {
		c = ' '
	}
	if r, ok := f.remap[c]; ok {
		c = r
	} else if c >= 0x7f {
		c = ' '
	}
	return &f.Glyphs[c-firstGlyph]
}

// Width returns the width of a string in pixels.
func (f *Font) Width(s []byte) int {
	n := 0
	for _, c := range s {
		n += f.Glyph(c).W
	}
	return n
}
