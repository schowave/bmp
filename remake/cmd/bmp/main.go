// Command bmp is a re-implementation of Bundesliga Manager Professional. It
// needs the files of the original game (BMMAIN.EXE, MANA.DAT, PIC/...).
package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/schowave/bmp/remake/internal/assets"
	"github.com/schowave/bmp/remake/internal/ui"
)

func main() {
	data := flag.String("data", "", "directory of the original game (default: $BMP_DATA, ./bmp or ../bmp)")
	scale := flag.Int("scale", 3, "window scale factor")
	shot := flag.String("shot", "", "save a screenshot of the canvas to this PNG file and quit")
	shotFrame := flag.Int("shot-frame", 10, "frame to take the screenshot at")
	demo := flag.String("demo", "", "show a development demo scene")
	saves := flag.String("saves", defaultSaveDir(), "directory for save games and high scores")
	flag.Parse()

	dir, err := openGameDir(*data)
	if err != nil {
		fmt.Fprintln(os.Stderr, "bmp:", err)
		os.Exit(1)
	}
	gfx, err := ui.NewGfx(dir)
	if err != nil {
		fmt.Fprintln(os.Stderr, "bmp:", err)
		os.Exit(1)
	}
	ebiten.SetWindowSize(ui.ScreenW**scale, ui.ScreenH**scale)
	ebiten.SetWindowTitle("Bundesliga Manager Professional")
	ebiten.SetWindowResizingMode(ebiten.WindowResizingModeEnabled)
	mana, err := dir.Mana()
	if err != nil {
		fmt.Fprintln(os.Stderr, "bmp:", err)
		os.Exit(1)
	}
	var first ui.Scene = ui.NewTitle().OnDone(func() ui.Scene { return ui.NewSetup(mana, *saves) })
	if *demo != "" {
		first = ui.Demo(*demo, mana, *saves)
	}
	app := ui.NewApp(gfx, first)
	app.ShotPath, app.ShotFrame = *shot, *shotFrame
	if err := ebiten.RunGame(app); err != nil {
		fmt.Fprintln(os.Stderr, "bmp:", err)
		os.Exit(1)
	}
}

func openGameDir(flagDir string) (*assets.Dir, error) {
	candidates := []string{flagDir, os.Getenv("BMP_DATA"), "bmp", filepath.Join("..", "bmp")}
	var firstErr error
	for _, c := range candidates {
		if c == "" {
			continue
		}
		d, err := assets.OpenDir(os.DirFS(c))
		if err == nil {
			return d, nil
		}
		if firstErr == nil {
			firstErr = fmt.Errorf("%s: %w", c, err)
		}
	}
	if firstErr == nil {
		firstErr = fmt.Errorf("game directory not found, pass -data")
	}
	return nil, firstErr
}

func defaultSaveDir() string {
	if d, err := os.UserConfigDir(); err == nil {
		return filepath.Join(d, "bmp")
	}
	return "saves"
}
