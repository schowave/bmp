package assets

import (
	"encoding/binary"
	"errors"
	"image"
	"image/color"
)

// Pic is an 8-bit indexed picture from PIC\N.VGA or PIC\N.CP.
type Pic struct {
	W, H int
	Pix  []byte // W*H palette indices, row by row
}

// DecodePic decodes the picture format of the game: a 6-byte header (width,
// height, size of the packed data, all little-endian words) followed by a
// byte-oriented run-length stream. A control byte c < 0x80 copies the next
// c+1 bytes, c > 0x80 repeats the next byte (c&0x7f)+1 times, 0x80 ends the
// stream. Full-screen pictures (.CP, 320x240) use the same format; the game
// only writes them to the four VGA planes of mode X instead of linear memory.
func DecodePic(data []byte) (*Pic, error) {
	if len(data) < 6 {
		return nil, errors.New("picture too short")
	}
	w := int(binary.LittleEndian.Uint16(data[0:]))
	h := int(binary.LittleEndian.Uint16(data[2:]))
	pix := make([]byte, 0, w*h)
	p := 6
	for p < len(data) && len(pix) < w*h {
		c := data[p]
		p++
		if c < 0x80 {
			n := int(c) + 1
			if p+n > len(data) {
				return nil, errors.New("picture literal out of range")
			}
			pix = append(pix, data[p:p+n]...)
			p += n
			continue
		}
		n := int(c & 0x7f)
		if n == 0 {
			break
		}
		if p >= len(data) {
			return nil, errors.New("picture run out of range")
		}
		for i := 0; i <= n; i++ {
			pix = append(pix, data[p])
		}
		p++
	}
	if len(pix) < w*h {
		pix = append(pix, make([]byte, w*h-len(pix))...)
	}
	return &Pic{W: w, H: h, Pix: pix[:w*h]}, nil
}

// Palette is a 256-colour VGA palette.
type Palette [256]color.RGBA

// DecodePalettes reads PIC\1.PAL: consecutive blocks of 256 RGB triples with
// 6-bit components.
func DecodePalettes(data []byte) []Palette {
	var pals []Palette
	for off := 0; off+768 <= len(data); off += 768 {
		var p Palette
		for i := 0; i < 256; i++ {
			c := data[off+i*3 : off+i*3+3]
			p[i] = color.RGBA{scale6(c[0]), scale6(c[1]), scale6(c[2]), 0xff}
		}
		pals = append(pals, p)
	}
	return pals
}

func scale6(v byte) uint8 {
	v &= 0x3f
	return v<<2 | v>>4
}

// RGBA renders the picture with a palette. Index 0 stays opaque; callers that
// need transparency use RGBAKeyed.
func (p *Pic) RGBA(pal *Palette) *image.RGBA {
	return p.render(pal, -1)
}

// RGBAKeyed renders the picture with palette index key made transparent.
func (p *Pic) RGBAKeyed(pal *Palette, key int) *image.RGBA {
	return p.render(pal, key)
}

func (p *Pic) render(pal *Palette, key int) *image.RGBA {
	img := image.NewRGBA(image.Rect(0, 0, p.W, p.H))
	for i, c := range p.Pix {
		if int(c) == key {
			continue
		}
		col := pal[c]
		img.Pix[i*4+0] = col.R
		img.Pix[i*4+1] = col.G
		img.Pix[i*4+2] = col.B
		img.Pix[i*4+3] = 0xff
	}
	return img
}
