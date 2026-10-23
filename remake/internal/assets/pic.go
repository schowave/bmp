package assets

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"strings"
)

// Pic is an 8-bit indexed picture from PIC\N.VGA or PIC\N.CP.
type Pic struct {
	W, H int
	Pix  []byte // W*H palette indices, row by row
}

// ParsePic decodes the picture format of the game: a 6-byte header (width,
// height, size of the packed data, all little-endian words) followed by a
// byte-oriented run-length stream. A control byte c < 0x80 copies the next
// c+1 bytes, c > 0x80 repeats the next byte (c&0x7f)+1 times, 0x80 ends the
// stream. Full-screen pictures (.CP, 320x240) use the same format; the game
// only writes them to the four VGA planes of mode X instead of linear memory.
func ParsePic(data []byte) (*Pic, error) {
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

// ParsePalettes reads PIC\1.PAL: consecutive blocks of 256 RGB triples with
// 6-bit components.
func ParsePalettes(data []byte) []Palette {
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

// EncodePic writes the picture as an indexed PNG: its pixels are the
// palette indices, its colour table is pal (only to view the file; the
// game draws it with the palette of the screen).
func EncodePic(p *Pic, pal *Palette) ([]byte, error) {
	cp := make(color.Palette, len(pal))
	for i, c := range pal {
		cp[i] = c
	}
	img := &image.Paletted{Pix: p.Pix, Stride: p.W, Rect: image.Rect(0, 0, p.W, p.H), Palette: cp}
	var b bytes.Buffer
	if err := png.Encode(&b, img); err != nil {
		return nil, err
	}
	return b.Bytes(), nil
}

// DecodePic reads a picture written by EncodePic. The PNG must be indexed:
// the game needs the palette indices, not the colours.
func DecodePic(data []byte) (*Pic, error) {
	img, err := png.Decode(bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	pi, ok := img.(*image.Paletted)
	if !ok {
		return nil, errors.New("not an indexed PNG (the game needs palette indices)")
	}
	r := pi.Bounds()
	p := &Pic{W: r.Dx(), H: r.Dy(), Pix: make([]byte, 0, r.Dx()*r.Dy())}
	for y := r.Min.Y; y < r.Max.Y; y++ {
		off := pi.PixOffset(r.Min.X, y)
		p.Pix = append(p.Pix, pi.Pix[off:off+p.W]...)
	}
	return p, nil
}

// EncodePalettes writes the palettes as JSON: for each palette its 256
// colours as "#rrggbb", eight to a line.
func EncodePalettes(pals []Palette) []byte {
	var b strings.Builder
	b.WriteString("[\n")
	for i, pal := range pals {
		b.WriteString("\t[\n")
		for k, c := range pal {
			if k%8 == 0 {
				b.WriteString("\t\t")
			}
			fmt.Fprintf(&b, "\"#%02x%02x%02x\"", c.R, c.G, c.B)
			switch {
			case k == len(pal)-1:
				b.WriteString("\n")
			case k%8 == 7:
				b.WriteString(",\n")
			default:
				b.WriteString(", ")
			}
		}
		b.WriteString("\t]")
		if i < len(pals)-1 {
			b.WriteString(",")
		}
		b.WriteString("\n")
	}
	b.WriteString("]\n")
	return []byte(b.String())
}

// DecodePalettes reads palettes written by EncodePalettes.
func DecodePalettes(data []byte) ([]Palette, error) {
	var in [][]string
	if err := json.Unmarshal(data, &in); err != nil {
		return nil, err
	}
	if len(in) == 0 {
		return nil, errors.New("no palette")
	}
	pals := make([]Palette, len(in))
	for i, cols := range in {
		if len(cols) != len(pals[i]) {
			return nil, fmt.Errorf("palette %d: %d colours, want %d", i, len(cols), len(pals[i]))
		}
		for k, s := range cols {
			var r, g, b uint8
			if n, err := fmt.Sscanf(s, "#%02x%02x%02x", &r, &g, &b); err != nil || n != 3 || len(s) != 7 {
				return nil, fmt.Errorf("palette %d colour %d: %q is not #rrggbb", i, k, s)
			}
			pals[i][k] = color.RGBA{r, g, b, 0xff}
		}
	}
	return pals, nil
}
