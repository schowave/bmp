package ui

import (
	"image/color"
	"testing"
)

// TestPointerColours checks the arrow against the colours it has in the
// shots of the original (testdata/original, e.g. e6).
func TestPointerColours(t *testing.T) {
	app, _ := testApp(t)
	img := app.Gfx.pointerRGBA()
	for _, c := range []struct {
		x, y int
		want color.RGBA
	}{
		{0, 0, color.RGBA{146, 0, 16, 255}},   // fill, 17
		{0, 1, color.RGBA{178, 0, 32, 255}},   // left edge, 18
		{1, 0, color.RGBA{195, 113, 32, 255}}, // right edge, 19
		{10, 0, color.RGBA{}},                 // transparent
	} {
		if got := img.RGBAAt(c.x, c.y); got != c.want {
			t.Errorf("pixel %d,%d = %v, want %v", c.x, c.y, got, c.want)
		}
	}
}
