package ui

import (
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
)

// Scene is one screen of the game. Scenes are kept on a stack: a dialog is
// pushed on top of the screen it belongs to and popped when it closes.
type Scene interface {
	Update(app *App) error
	Draw(app *App, dst *ebiten.Image)
}

// passer is a scene that only decides what comes next and has nothing of
// its own to show (Passing true): when it comes on top it runs again in the
// same tick, so that it is never drawn for a frame between two screens.
type passer interface {
	Scene
	Passing() bool
}

// maxPasses bounds the passes of one tick, for a passer that opens nothing.
const maxPasses = 8

// Input is the state of mouse and keyboard for one frame, in canvas
// coordinates. The original is played with the mouse: the left button
// selects, the right button confirms in some places but does not leave a
// screen.
type Input struct {
	X, Y       int
	Left       bool // left button pressed this frame
	Right      bool // right button pressed this frame
	LeftHeld   bool
	Chars      []rune
	Enter      bool
	Escape     bool
	Backspace  bool
	WheelDelta float64
}

// In reports whether the pointer is inside the rectangle.
func (in *Input) In(x, y, w, h int) bool {
	return in.X >= x && in.X < x+w && in.Y >= y && in.Y < y+h
}

// Clicked reports a left click inside the rectangle.
func (in *Input) Clicked(x, y, w, h int) bool {
	return in.Left && in.In(x, y, w, h)
}

// App is the Ebitengine game: a stack of scenes on a 320x240 canvas.
type App struct {
	Gfx   *Gfx
	Input Input
	stack []Scene
	quit  bool
	frame int

	// ShotPath, when set, makes the app save the canvas as PNG after
	// ShotFrame frames and quit (for tests and documentation).
	ShotPath  string
	ShotFrame int
	// Script feeds synthetic input, one entry per frame, instead of real
	// input; a step with a Shot saves the canvas after that frame.
	Script    []ScriptStep
	shot      string
	scriptEnd bool  // the last step of the script ran: quit after drawing
	err       error // a screenshot could not be saved; Update returns it

	// Paced runs the scenes at TPS ticks a second by the clock, for an
	// Ebitengine that calls Update on every input event as well (see
	// cmd/bmp). The input between two ticks is gathered in pending.
	Paced   bool
	next    time.Time
	pending Input
	pressed bool // pending holds a click: its place is kept

	// Control, when set, delivers further scripts while the app runs (for
	// -control); ControlDone is called after the last step of each was drawn.
	Control     <-chan []ScriptStep
	ControlDone func()
	ctlDone     bool
	ctlErr      string // a step of the last control script failed
}

// TPS is the number of ticks a second the scenes are updated.
const TPS = 60

// ScriptStep is the input of one frame of a script. Target names a click
// target (see App.target) that replaces the place of the input; Dump
// writes the running game to that file.
type ScriptStep struct {
	In     Input
	Shot   string
	Target string
	Dump   string
}

// NewApp creates the application with a first scene.
func NewApp(gfx *Gfx, first Scene) *App {
	return &App{Gfx: gfx, stack: []Scene{first}}
}

// Push shows a scene on top of the current one.
func (a *App) Push(s Scene) { a.stack = append(a.stack, s) }

// Pop closes the topmost scene.
func (a *App) Pop() {
	if len(a.stack) > 0 {
		a.stack = a.stack[:len(a.stack)-1]
	}
}

// Replace swaps the topmost scene for another.
func (a *App) Replace(s Scene) {
	a.Pop()
	a.Push(s)
}

// Reset replaces the whole stack with one scene.
func (a *App) Reset(s Scene) { a.stack = []Scene{s} }

// Top returns the topmost scene, or nil.
func (a *App) Top() Scene {
	if len(a.stack) == 0 {
		return nil
	}
	return a.stack[len(a.stack)-1]
}

// Quit ends the program after this frame.
func (a *App) Quit() { a.quit = true }

// Update implements ebiten.Game.
func (a *App) Update() error {
	if a.err != nil {
		return a.err
	}
	if err := a.Gfx.Err(); err != nil {
		return err
	}
	return a.update(readInput(), time.Now())
}

// update runs a tick of the top scene with the input read from Ebitengine
// at now; a paced app gathers the input until the next tick is due.
func (a *App) update(raw Input, now time.Time) error {
	if a.quit || len(a.stack) == 0 {
		return ebiten.Termination
	}
	if a.shot != "" || (a.scriptEnd && a.ShotPath == "") {
		// Ebitengine may run several updates per frame: wait for the
		// screenshot of the script to be drawn.
		return nil
	}
	a.gather(raw)
	if a.Paced && !a.tickDue(now) {
		return nil
	}
	a.Input, a.pending, a.pressed = a.pending, Input{}, false
	a.frame++
	if len(a.Script) == 0 && a.Control != nil {
		select {
		case a.Script = <-a.Control:
			a.ctlErr = ""
		default:
		}
	}
	if len(a.Script) > 0 {
		st := a.Script[0]
		a.Input = st.In
		a.shot = st.Shot
		a.Script = a.Script[1:]
		if st.Target != "" {
			// The release after a click may find the target gone (a
			// dialog closed): only the click itself needs it.
			if x, y, ok := a.target(st.Target); ok {
				a.Input.X, a.Input.Y = x, y
			} else if st.In.Left || st.In.Right {
				a.ctlErr = "no target " + st.Target
				a.Input = Input{}
			}
		}
		if st.Dump != "" {
			if err := a.dump(st.Dump); err != nil {
				a.ctlErr = err.Error()
			}
		}
		if a.Control != nil {
			a.ctlDone = len(a.Script) == 0
		} else {
			a.scriptEnd = len(a.Script) == 0
		}
	}
	return a.updateTop()
}

// updateTop runs a tick of the top scene, then of every passer that comes
// on top, without input.
func (a *App) updateTop() error {
	if err := a.stack[len(a.stack)-1].Update(a); err != nil {
		return err
	}
	for range maxPasses {
		p, ok := a.Top().(passer)
		if !ok || !p.Passing() {
			break
		}
		a.Input = Input{}
		if err := p.Update(a); err != nil {
			return err
		}
	}
	return nil
}

// tickDue reports whether the next tick is due at now. A tick may come up
// to half a tick early, so that a frame ticker at TPS does not beat against
// the clock; the ticks still average TPS a second.
func (a *App) tickDue(now time.Time) bool {
	const tick = time.Second / TPS
	if a.next.IsZero() || now.Sub(a.next) > 4*tick {
		// The first tick, or the game was stalled: start the clock anew.
		a.next = now
	}
	if now.Before(a.next.Add(-tick / 2)) {
		return false
	}
	a.next = a.next.Add(tick)
	return true
}

// gather adds the input of one Update to the input of the next tick: the
// presses and the typed keys add up, the pointer is the latest one unless a
// click is pending, which keeps the place it was made at.
func (a *App) gather(in Input) {
	p := &a.pending
	if !a.pressed {
		p.X, p.Y = in.X, in.Y
		a.pressed = in.Left || in.Right
	}
	p.Left = p.Left || in.Left
	p.Right = p.Right || in.Right
	p.LeftHeld = in.LeftHeld
	p.Chars = append(p.Chars, in.Chars...)
	p.Enter = p.Enter || in.Enter
	p.Escape = p.Escape || in.Escape
	p.Backspace = p.Backspace || in.Backspace
	p.WheelDelta += in.WheelDelta
}

func readInput() Input {
	x, y := ebiten.CursorPosition()
	in := Input{
		X:         x,
		Y:         y,
		Left:      inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft),
		Right:     inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonRight),
		LeftHeld:  ebiten.IsMouseButtonPressed(ebiten.MouseButtonLeft),
		Chars:     ebiten.AppendInputChars(nil),
		Enter:     inpututil.IsKeyJustPressed(ebiten.KeyEnter) || inpututil.IsKeyJustPressed(ebiten.KeyNumpadEnter),
		Escape:    inpututil.IsKeyJustPressed(ebiten.KeyEscape),
		Backspace: inpututil.IsKeyJustPressed(ebiten.KeyBackspace),
	}
	_, in.WheelDelta = ebiten.Wheel()
	return in
}

// Draw implements ebiten.Game. Every scene of the stack is drawn, bottom
// first, so dialogs appear over their screen.
func (a *App) Draw(screen *ebiten.Image) {
	for _, s := range a.stack {
		s.Draw(a, screen)
	}
	if a.shot != "" {
		if err := savePNG(screen, a.shot); err != nil && a.err == nil {
			a.err = err
		}
		a.shot = ""
	}
	if a.scriptEnd && a.ShotPath == "" {
		a.quit = true
	}
	if a.ctlDone {
		a.ctlDone = false
		a.ControlDone()
	}
	if a.ShotPath != "" && a.frame >= a.ShotFrame {
		if err := savePNG(screen, a.ShotPath); err != nil && a.err == nil {
			a.err = err
		}
		a.ShotPath = ""
		a.quit = true
	}
}

// Layout implements ebiten.Game: the canvas always has the original size and
// Ebitengine scales it to the window.
func (a *App) Layout(int, int) (int, int) { return ScreenW, ScreenH }
