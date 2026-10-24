// Command bmpicon draws the icon of the remake from the embedded pictures:
// the title banner PIC/4.VGA broken into two lines, BUNDESLIGA and MANAGER
// with the red "professional", on the marble of PIC/22.CP. It writes a
// 1024x1024 PNG, which cmd/bmp embeds as its window icon and mise run
// remake:dist turns into the icons of the Windows program and the Mac app.
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

	// The colours of the game are made for a dark room; the icon lifts them
	// towards white, the marble more than the plate.
	lightenMarble = 0.30
	lightenPlate  = 0.12
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

// icon puts the plate, scaled up by whole pixels and with a shadow, on a
// rounded square of marble.
func icon(banner, marble image.Image) *image.RGBA {
	dst := image.NewRGBA(image.Rect(0, 0, size, size))
	inner := size - 2*inset
	bg := image.NewRGBA(image.Rect(0, 0, inner, inner))
	scale(bg, bg.Bounds(), marble, image.Rect(50, 20, 256, 226))
	for y := range inner {
		for x := range inner {
			if inRoundedSquare(x, y, inner, radius) {
				dst.Set(inset+x, inset+y, lighten(bg.At(x, y), lightenMarble))
			}
		}
	}
	p := plate(banner)
	for y := range p.Bounds().Dy() {
		for x := range p.Bounds().Dx() {
			p.Set(x, y, lighten(p.At(x, y), lightenPlate))
		}
	}
	s := (inner - 40) / p.Bounds().Dx()
	w, h := p.Bounds().Dx()*s, p.Bounds().Dy()*s
	r := image.Rect((size-w)/2, (size-h)/2, (size+w)/2, (size+h)/2)
	shadow := image.NewUniform(color.RGBA{A: 110})
	draw.Draw(dst, r.Add(image.Pt(2*s, 2*s)), shadow, image.Point{}, draw.Over)
	scale(dst, r, p, p.Bounds())
	return dst
}

// The banner PIC/4.VGA is 320x44: stone in rows 0-3 and 39-43 and in
// columns 0-3 and 315-319, turf between. BUNDESLIGA ends at column 167, the
// goal post stands in 168-180, MANAGER and the script take 176-314.
const (
	stoneL, stoneR, stoneT, stoneB = 4, 5, 4, 5
	plateW                         = 168 // turf inside the stone
	line1H                         = 28  // rows 4-31: BUNDESLIGA and its shadow
	line2H                         = 31  // rows 8-38: MANAGER and the script
	line2X, line2W                 = 176, 139
)

// plate is the banner in two lines inside its stone frame.
func plate(b image.Image) *image.RGBA {
	w, h := stoneL+plateW+stoneR, stoneT+line1H+line2H+stoneB
	p := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := range h {
		for x := range w {
			p.Set(x, y, turf(b, x, y))
		}
	}
	// stone above and below, from a part of the banner without letters
	for x := range w {
		for y := range stoneT {
			p.Set(x, y, b.At(20+x%140, y))
		}
		for y := range stoneB {
			p.Set(x, h-stoneB+y, b.At(20+x%140, 39+y))
		}
	}
	// stone at the sides, with the corners of the banner
	side := func(sx, sw, dx int) {
		for y := range h {
			sy := 4 + (y-stoneT)%35
			switch {
			case y < stoneT:
				sy = y
			case y >= h-stoneB:
				sy = 39 + y - (h - stoneB)
			}
			for x := range sw {
				p.Set(dx+x, y, b.At(sx+x, sy))
			}
		}
	}
	side(0, stoneL, 0)
	side(315, stoneR, w-stoneR)

	draw.Draw(p, image.Rect(stoneL, stoneT, stoneL+164, stoneT+line1H), b, image.Pt(4, 4), draw.Src)
	x2, y2 := stoneL+(plateW-line2W)/2, stoneT+line1H
	draw.Draw(p, image.Rect(x2, y2, x2+line2W, y2+line2H), b, image.Pt(line2X, 8), draw.Src)
	// the cross bar of the goal post reaches into the crop before the M
	for y := y2; y < y2+line2H; y++ {
		for x := x2; x < x2+5; x++ {
			if c := p.At(x, y); !isTurf(c) && !isRed(c) {
				p.Set(x, y, turf(b, x, y))
			}
		}
	}
	// the tail of the script over the lower stone
	for y := 39; y < 44; y++ {
		for x := line2X; x < line2X+line2W; x++ {
			if c := b.At(x, y); isRed(c) || isBlack(c) {
				p.Set(x2+x-line2X, y2+line2H+y-39, c)
			}
		}
	}
	return p
}

// turf repeats a strip of the banner below the letters of BUNDESLIGA.
func turf(b image.Image, x, y int) color.Color { return b.At(30+x%90, 32+y%7) }

func rgb(c color.Color) (r, g, b uint32) {
	r, g, b, _ = c.RGBA()
	return r >> 8, g >> 8, b >> 8
}

func isTurf(c color.Color) bool  { r, g, b := rgb(c); return g > r && g > b }
func isRed(c color.Color) bool   { r, g, b := rgb(c); return r > 100 && g < 80 && b < 80 }
func isBlack(c color.Color) bool { r, g, b := rgb(c); return r < 40 && g < 40 && b < 40 }

// lighten moves c by f of the way to white.
func lighten(c color.Color, f float64) color.Color {
	r, g, b := rgb(c)
	up := func(v uint32) uint8 { return uint8(float64(v) + (255-float64(v))*f + 0.5) }
	return color.RGBA{up(r), up(g), up(b), 255}
}

// scale draws sr of src into r of dst, nearest neighbour.
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
	cx := min(max(x, rad), w-rad-1)
	cy := min(max(y, rad), w-rad-1)
	dx, dy := x-cx, y-cy
	return dx*dx+dy*dy <= rad*rad
}
