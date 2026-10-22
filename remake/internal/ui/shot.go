package ui

import (
	"image"
	"image/png"
	"os"

	"github.com/hajimehoshi/ebiten/v2"
)

func savePNG(screen *ebiten.Image, path string) error {
	b := screen.Bounds()
	rgba := image.NewRGBA(b)
	screen.ReadPixels(rgba.Pix)
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	return png.Encode(f, rgba)
}
