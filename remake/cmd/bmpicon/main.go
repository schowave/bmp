// Command bmpicon draws the icon of the remake from the embedded pictures:
// the letters of the title banner PIC/4.VGA in three lines, BUNDES, LIGA and
// MANAGER, with the red "professional" over the foot of MANAGER, on the turf
// of the banner in a frame of the marble of PIC/22.CP. Three short lines
// make the letters larger than the banner's two, so the icon reads at small
// sizes. It writes a 1024x1024 PNG, which cmd/bmp embeds as its window icon
// and mise run remake:dist turns into the icons of the Windows program and
// the Mac app.
package main

import (
	"flag"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"os"

	"github.com/schowave/bmp/remake/internal/embedded"
)

const (
	size   = 1024
	inset  = 100 // transparent margin around the rounded square, as macOS icons have
	radius = 185

	frame   = 60 // width of the marble frame
	edge    = 6  // dark line between marble and turf
	padding = 20 // turf around the letters

	turfScale = 6 // screen pixels per pixel of turf

	// The colours of the game are made for a dark room; the icon lifts them
	// towards white, the marble most, the script least.
	lightenMarble  = 0.30
	lightenLetters = 0.12
	lightenScript  = 0.05
)

func main() {
	out := flag.String("out", "cmd/bmp/icon.png", "PNG file to write")
	flag.Parse()
	if err := run(*out); err != nil {
		fmt.Fprintln(os.Stderr, "bmpicon:", err)
		os.Exit(1)
	}
}

func run(out string) error {
	banner, err := loadPNG("pic/4.png")
	if err != nil {
		return err
	}
	marble, err := loadPNG("pic/full/22.png")
	if err != nil {
		return err
	}
	f, err := os.Create(out)
	if err != nil {
		return err
	}
	if err := png.Encode(f, icon(banner, marble)); err != nil {
		_ = f.Close()
		return err
	}
	return f.Close()
}

func loadPNG(name string) (image.Image, error) {
	f, err := embedded.FS().Open(name)
	if err != nil {
		return nil, err
	}
	defer func() { _ = f.Close() }()
	m, err := png.Decode(f)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", name, err)
	}
	return m, nil
}

// icon puts the lettering on turf in a marble frame on a rounded square.
func icon(banner, marble image.Image) *image.RGBA {
	dst := image.NewRGBA(image.Rect(0, 0, size, size))
	inner := size - 2*inset
	bg := image.NewRGBA(image.Rect(0, 0, inner, inner))
	scale(bg, bg.Bounds(), marble, image.Rect(50, 20, 256, 226))
	dark := color.RGBA{40, 40, 60, 255}
	lineW, lineR := inner-2*(frame-edge), radius-frame+edge
	turfW, turfR := inner-2*frame, radius-frame
	for y := range inner {
		for x := range inner {
			switch {
			case inRoundedSquare(x-frame, y-frame, turfW, turfR):
				c := turf(banner, x/turfScale, y/turfScale)
				dst.Set(inset+x, inset+y, lighten(c, lightenLetters))
			case inRoundedSquare(x-frame+edge, y-frame+edge, lineW, lineR):
				dst.Set(inset+x, inset+y, dark)
			case inRoundedSquare(x, y, inner, radius):
				dst.Set(inset+x, inset+y, lighten(bg.At(x, y), lightenMarble))
			}
		}
	}
	l := lettering(banner)
	avail := float64(turfW - 2*padding)
	s := min(avail/float64(l.Bounds().Dx()), avail/float64(l.Bounds().Dy()))
	w, h := int(float64(l.Bounds().Dx())*s), int(float64(l.Bounds().Dy())*s)
	scale(dst, image.Rect((size-w)/2, (size-h)/2, (size+w)/2, (size+h)/2), l, l.Bounds())
	return dst
}

// The banner PIC/4.VGA is 320x44: stone around, turf between. The faces of
// the letters of BUNDESLIGA and MANAGER fill rows 9-28; each letter casts a
// black shadow to the lower right. The red script crosses rows 22-43 under
// MANAGER, over the foot of its M, A, G, E and R, and the goal post stands
// in columns 166-180 between the words.
const glyphT, glyphB = 9, 29

// columns of the letters of BUNDESLIGA
var (
	letB, letU, letN = [2]int{10, 24}, [2]int{26, 40}, [2]int{43, 58}
	letD, letE, letS = [2]int{60, 76}, [2]int{78, 91}, [2]int{93, 108}
	letL, letI, letG = [2]int{110, 122}, [2]int{124, 128}, [2]int{130, 151}
	letA             = [2]int{151, 166}
)

// lettering is BUNDES, LIGA and MANAGER, each letter with a fresh shadow,
// and the script over the foot of MANAGER, on a transparent ground.
func lettering(b image.Image) *image.RGBA {
	word := func(cols ...[2]int) *image.RGBA {
		gs := make([]*image.RGBA, len(cols))
		for i, c := range cols {
			gs[i] = glyph(b, c[0], c[1])
		}
		return shadowed(hcat(gs, 3), 2)
	}
	lines := []*image.RGBA{
		word(letB, letU, letN, letD, letE, letS),
		word(letL, letI, letG, letA),
		// MANAGER takes A, N, G and E from BUNDESLIGA, where the script
		// does not cover them
		shadowed(hcat([]*image.RGBA{
			letterM(b), glyph(b, letA[0], letA[1]), glyph(b, letN[0], letN[1]),
			glyph(b, letA[0], letA[1]), glyph(b, letG[0], letG[1]),
			glyph(b, letE[0], letE[1]), letterR(b),
		}, 3), 2),
	}
	sc := script(b)
	const gap, overlap = 1, 9
	w, h := sc.Bounds().Dx(), sc.Bounds().Dy()-overlap-gap
	for _, l := range lines {
		w = max(w, l.Bounds().Dx())
		h += l.Bounds().Dy() + gap
	}
	dst := image.NewRGBA(image.Rect(0, 0, w, h))
	y := 0
	for _, l := range append(lines, sc) {
		if l == sc {
			y -= gap + overlap
		}
		x := (w - l.Bounds().Dx()) / 2
		draw.Draw(dst, l.Bounds().Add(image.Pt(x, y)), l, image.Point{}, draw.Over)
		y += l.Bounds().Dy() + gap
	}
	return dst
}

// glyph is the face of the letter in columns x0 to x1 (exclusive).
func glyph(b image.Image, x0, x1 int) *image.RGBA {
	g := image.NewRGBA(image.Rect(0, 0, x1-x0, glyphB-glyphT))
	for y := glyphT; y < glyphB; y++ {
		for x := x0; x < x1; x++ {
			if c := b.At(x, y); isFace(c) {
				g.Set(x-x0, y-glyphT, lighten(c, lightenLetters))
			}
		}
	}
	return g
}

// letterM rebuilds the M of MANAGER (columns 181-195), whose right foot the
// script covers, from its left half: the M is symmetric about column 188.
func letterM(b image.Image) *image.RGBA {
	left := glyph(b, 181, 189)
	g := image.NewRGBA(image.Rect(0, 0, 15, glyphB-glyphT))
	draw.Draw(g, left.Bounds(), left, image.Point{}, draw.Src)
	for y := range g.Bounds().Dy() {
		for x := range 7 {
			g.Set(14-x, y, left.At(x, y))
		}
	}
	return g
}

// letterR is the R of MANAGER (columns 286-301), whose foot in rows 26-28
// the script covers: the stem goes on straight, the leg one column to the
// right per row, as above.
func letterR(b image.Image) *image.RGBA {
	g := image.NewRGBA(image.Rect(0, 0, 16, glyphB-glyphT))
	top := glyph(b, 286, 302)
	draw.Draw(g, image.Rect(0, 0, 16, 26-glyphT), top, image.Point{}, draw.Src)
	c := lighten(b.At(287, 20), lightenLetters)
	for i, y := range []int{26, 27, 28} {
		for x := range 16 {
			if x < 4 || (x >= 9+i && x < 14+i) {
				g.Set(x, y-glyphT, c)
			}
		}
	}
	return g
}

// script is the red "professional" (columns 176-315, rows 20-43) with a
// shadow of one pixel.
func script(b image.Image) *image.RGBA {
	g := image.NewRGBA(image.Rect(0, 0, 140, 24))
	for y := 20; y < 44; y++ {
		for x := 176; x < 316; x++ {
			if c := b.At(x, y); isRed(c) {
				g.Set(x-176, y-20, lighten(c, lightenScript))
			}
		}
	}
	return shadowed(g, 1)
}

// hcat sets glyphs side by side, gap pixels apart.
func hcat(gs []*image.RGBA, gap int) *image.RGBA {
	w, h := -gap, 0
	for _, g := range gs {
		w += g.Bounds().Dx() + gap
		h = max(h, g.Bounds().Dy())
	}
	dst := image.NewRGBA(image.Rect(0, 0, w, h))
	x := 0
	for _, g := range gs {
		r := g.Bounds().Add(image.Pt(x, h-g.Bounds().Dy()))
		draw.Draw(dst, r, g, image.Point{}, draw.Over)
		x += g.Bounds().Dx() + gap
	}
	return dst
}

// shadowed adds a solid black shadow off pixels to the lower right of g.
func shadowed(g *image.RGBA, off int) *image.RGBA {
	r := g.Bounds()
	dst := image.NewRGBA(image.Rect(0, 0, r.Dx()+off, r.Dy()+off))
	black := image.NewUniform(color.Black)
	for dy := 1; dy <= off; dy++ {
		for dx := 1; dx <= off; dx++ {
			draw.DrawMask(dst, r.Add(image.Pt(dx, dy)), black, image.Point{}, g, image.Point{}, draw.Over)
		}
	}
	draw.Draw(dst, r, g, image.Point{}, draw.Over)
	return dst
}

// turf repeats a strip of the banner below the letters of BUNDESLIGA.
func turf(b image.Image, x, y int) color.Color { return b.At(30+x%90, 32+y%7) }

func rgb(c color.Color) (r, g, b uint32) {
	r, g, b, _ = c.RGBA()
	return r >> 8, g >> 8, b >> 8
}

func isFace(c color.Color) bool { r, g, b := rgb(c); return r > 150 && g > 150 && b > 120 }
func isRed(c color.Color) bool  { r, g, b := rgb(c); return r > 100 && g < 80 && b < 80 }

// lighten moves c by f of the way to white.
func lighten(c color.Color, f float64) color.Color {
	r, g, b := rgb(c)
	up := func(v uint32) uint8 { return uint8(float64(v) + (255-float64(v))*f + 0.5) }
	return color.RGBA{up(r), up(g), up(b), 255}
}

// scale draws sr of src into r of dst, nearest neighbour, skipping pixels
// that are not opaque.
func scale(dst draw.Image, r image.Rectangle, src image.Image, sr image.Rectangle) {
	for y := range r.Dy() {
		for x := range r.Dx() {
			c := src.At(sr.Min.X+x*sr.Dx()/r.Dx(), sr.Min.Y+y*sr.Dy()/r.Dy())
			if _, _, _, a := c.RGBA(); a == 0xffff {
				dst.Set(r.Min.X+x, r.Min.Y+y, c)
			}
		}
	}
}

// inRoundedSquare reports whether x, y lies in a square of side w with
// corners of radius rad.
func inRoundedSquare(x, y, w, rad int) bool {
	if x < 0 || y < 0 || x >= w || y >= w {
		return false
	}
	cx := min(max(x, rad), w-rad-1)
	cy := min(max(y, rad), w-rad-1)
	dx, dy := x-cx, y-cy
	return dx*dx+dy*dy <= rad*rad
}
