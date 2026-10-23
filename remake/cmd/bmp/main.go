// Command bmp is a re-implementation of Bundesliga Manager Professional. It
// needs the files of the original game (BMMAIN.EXE, MANA.DAT, PIC/...).
package main

import (
	"flag"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/schowave/bmp/remake/internal/assets"
	"github.com/schowave/bmp/remake/internal/embedded"
	"github.com/schowave/bmp/remake/internal/ui"
)

func main() {
	data := flag.String("data", "", "directory of the original game (default: $BMP_DATA, the files built into the binary, ./bmp or ../bmp)")
	scale := flag.Int("scale", 3, "window scale factor")
	shot := flag.String("shot", "", "save a screenshot of the canvas to this PNG file and quit")
	shotFrame := flag.Int("shot-frame", 10, "frame to take the screenshot at")
	demo := flag.String("demo", "", "show a development demo scene")
	script := flag.String("script", "", "play the input of this development script (see ui.ParseScript) and quit")
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
	if *script != "" {
		f, err := os.Open(*script)
		if err == nil {
			app.Script, err = ui.ParseScript(f)
			f.Close()
		}
		if err != nil {
			fmt.Fprintln(os.Stderr, "bmp:", err)
			os.Exit(1)
		}
	}
	if err := ebiten.RunGame(app); err != nil {
		fmt.Fprintln(os.Stderr, "bmp:", err)
		os.Exit(1)
	}
}

func openGameDir(flagDir string) (*assets.Dir, error) {
	var firstErr error
	try := func(fsys fs.FS, name string) *assets.Dir {
		d, err := assets.OpenDir(fsys)
		if err != nil && firstErr == nil {
			firstErr = fmt.Errorf("%s: %w", name, err)
		}
		return d
	}
	for _, c := range []string{flagDir, os.Getenv("BMP_DATA")} {
		if c != "" {
			if d := try(os.DirFS(c), c); d != nil {
				return d, nil
			}
			return nil, firstErr
		}
	}
	if fsys := embedded.FS(); fsys != nil {
		if d := try(fsys, "embedded files"); d != nil {
			return d, nil
		}
	}
	for _, c := range []string{"bmp", filepath.Join("..", "bmp")} {
		if d := try(os.DirFS(c), c); d != nil {
			return d, nil
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
