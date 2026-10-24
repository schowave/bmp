// Command bmp is a re-implementation of Bundesliga Manager Professional. The
// pictures, fonts and club data of the original are built into the binary;
// -data or $BMP_DATA reads them from a directory of the original instead.
package main

import (
	"bytes"
	_ "embed"
	"flag"
	"fmt"
	"image"
	"image/png"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/schowave/bmp/remake/internal/assets"
	"github.com/schowave/bmp/remake/internal/embedded"
	"github.com/schowave/bmp/remake/internal/game"
	"github.com/schowave/bmp/remake/internal/ui"
)

func main() {
	data := flag.String("data", "", "directory of an extract of the game (bmpextract) instead of the files built into the binary (default: $BMP_DATA)")
	scale := flag.Int("scale", 3, "window scale factor")
	shot := flag.String("shot", "", "save a screenshot of the canvas to this PNG file and quit")
	shotFrame := flag.Int("shot-frame", 10, "frame to take the screenshot at")
	demo := flag.String("demo", "", "show a development demo scene (see -demos)")
	demos := flag.Bool("demos", false, "list the demo scenes and quit")
	script := flag.String("script", "", "play the input of this development script (see ui.ParseScript) and quit")
	control := flag.String("control", "", "development: play the scripts written to DIR/cmd while running, at -tps ticks a second")
	load := flag.String("load", "", "development: start with this save game")
	tps := flag.Int("tps", ui.TPS, "ticks a second with -control")
	saves := flag.String("saves", defaultSaveDir(), "directory for save games and high scores")
	flag.Parse()
	if *demos {
		fmt.Println(strings.Join(ui.DemoNames(), "\n"))
		return
	}
	if *demo != "" && !slices.Contains(ui.DemoNames(), *demo) {
		fmt.Fprintln(os.Stderr, "bmp: unknown demo", *demo)
		os.Exit(2)
	}

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
	if ic, err := png.Decode(bytes.NewReader(iconPNG)); err == nil {
		ebiten.SetWindowIcon([]image.Image{ic})
	}
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
	if *load != "" {
		data, err := os.ReadFile(*load)
		var st *game.State
		if err == nil {
			st, err = game.Load(data, mana)
		}
		if err != nil {
			fmt.Fprintln(os.Stderr, "bmp:", err)
			os.Exit(1)
		}
		first = ui.NewDriver(&ui.Session{S: st, Mana: mana, SaveDir: *saves})
	}
	app := ui.NewApp(gfx, first)
	app.ShotPath, app.ShotFrame = *shot, *shotFrame
	if *control != "" {
		startControl(app, *control, *tps)
	} else {
		pace(app)
	}
	if *script != "" {
		f, err := os.Open(*script)
		if err == nil {
			app.Script, err = ui.ParseScript(f)
			_ = f.Close()
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
	if d := try(embedded.FS(), "embedded files"); d != nil {
		return d, nil
	}
	return nil, firstErr
}

// iconPNG is the icon of the remake, drawn by cmd/bmpicon (mise run
// remake:icon); macOS takes the icon of the app instead.
//
//go:embed icon.png
var iconPNG []byte

func defaultSaveDir() string {
	if d, err := os.UserConfigDir(); err == nil {
		return filepath.Join(d, "bmp")
	}
	return "saves"
}
