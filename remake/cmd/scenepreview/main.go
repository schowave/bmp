// Command scenepreview plays a goal scene (TORE/*.T, *.V) of the original in
// a window, for development: the scene window of the match screen at
// (0x42, 0x73) on a black 320x240 canvas. With -out it saves every new
// picture as a PNG (f000.png, ...), with -replays it also plays the slow
// motion replays, and with -quit it closes at the end.
package main

import (
	"flag"
	"fmt"
	"image"
	"image/png"
	"os"
	"path/filepath"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/schowave/bmp/remake/internal/assets"
	"github.com/schowave/bmp/remake/internal/embedded"
	"github.com/schowave/bmp/remake/internal/ui"
)

type preview struct {
	g      *ui.Gfx
	sc     *ui.GoalScene
	out    string
	quit   bool
	reps   int
	n      int
	last   int
	over   bool
	canvas *ebiten.Image
}

func (p *preview) Update() error {
	if p.over {
		if p.quit {
			return ebiten.Termination
		}
		return nil
	}
	if p.sc.Update() {
		if p.sc.Plays() <= p.reps && p.sc.Replay() {
			p.last = -1
			return nil
		}
		p.over = true
	}
	return nil
}

func (p *preview) Draw(screen *ebiten.Image) {
	p.canvas.Fill(p.g.PalColor(0))
	p.sc.Draw(p.canvas, 0x42, 0x71+2)
	screen.DrawImage(p.canvas, nil)
	if p.out != "" && p.sc.Frame() != p.last {
		p.last = p.sc.Frame()
		rgba := image.NewRGBA(p.canvas.Bounds())
		p.canvas.ReadPixels(rgba.Pix)
		f, err := os.Create(filepath.Join(p.out, fmt.Sprintf("p%d_f%03d.png", p.sc.Plays(), p.last)))
		if err == nil {
			err = png.Encode(f, rgba)
			if cerr := f.Close(); err == nil {
				err = cerr
			}
		}
		if err != nil {
			fmt.Fprintln(os.Stderr, "scenepreview:", err)
		}
		p.n++
	}
}

func (p *preview) Layout(w, h int) (int, int) { return ui.ScreenW, ui.ScreenH }

func main() {
	data := flag.String("data", "", "directory of the original game instead of the files built into the binary")
	name := flag.String("scene", "10.T", "scene file in TORE/")
	mirror := flag.Bool("mirror", false, "mirrored (chance of the away team)")
	out := flag.String("out", "", "save every picture as PNG into this directory")
	reps := flag.Int("replays", 0, "number of replays to play (0..2)")
	quit := flag.Bool("quit", false, "quit at the end")
	scale := flag.Int("scale", 2, "window scale")
	flag.Parse()
	die := func(err error) {
		fmt.Fprintln(os.Stderr, "scenepreview:", err)
		os.Exit(1)
	}
	fsys := embedded.FS()
	if *data != "" {
		fsys = os.DirFS(*data)
	}
	dir, err := assets.OpenDir(fsys)
	if err != nil {
		die(err)
	}
	g, err := ui.NewGfx(dir)
	if err != nil {
		die(err)
	}
	sc, err := dir.Scene(*name)
	if err != nil {
		die(err)
	}
	if *out != "" {
		if err := os.MkdirAll(*out, 0o755); err != nil {
			die(err)
		}
	}
	fmt.Printf("%s: %d frames, sounds %v, builder %q\n", *name, len(sc.Frames), sc.Sounds, sc.Builder)
	p := &preview{g: g, out: *out, quit: *quit, reps: *reps, last: -1,
		sc: ui.NewGoalScene(g, sc, *mirror), canvas: ebiten.NewImage(ui.ScreenW, ui.ScreenH)}
	ebiten.SetWindowSize(ui.ScreenW**scale, ui.ScreenH**scale)
	ebiten.SetWindowTitle("Torszene " + *name)
	if err := ebiten.RunGame(p); err != nil {
		die(err)
	}
}
