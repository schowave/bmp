package ui

import (
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
)

// Scene is one screen of the game. Scenes are kept on a stack: a dialog is
// pushed on top of the screen it belongs to and popped when it closes.
type Scene interface {
	Update(app *App) error
	Draw(app *App, dst *ebiten.Image)
}

// Input is the state of mouse and keyboard for one frame, in canvas
// coordinates. The original is played with the mouse: left button selects,
// right button confirms or goes back.
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
	scriptEnd bool // the last step of the script ran: quit after drawing
}

// ScriptStep is the input of one frame of a script.
type ScriptStep struct {
	In   Input
	Shot string
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

// Quit ends the program after this frame.
func (a *App) Quit() { a.quit = true }

// Update implements ebiten.Game.
func (a *App) Update() error {
	if a.quit || len(a.stack) == 0 {
		return ebiten.Termination
	}
	if a.shot != "" || (a.scriptEnd && a.ShotPath == "") {
		// Ebitengine may run several updates per frame: wait for the
		// screenshot of the script to be drawn.
		return nil
	}
	a.frame++
	a.readInput()
	if len(a.Script) > 0 {
		a.Input = a.Script[0].In
		a.shot = a.Script[0].Shot
		a.Script = a.Script[1:]
		a.scriptEnd = len(a.Script) == 0
	}
	return a.stack[len(a.stack)-1].Update(a)
}

func (a *App) readInput() {
	x, y := ebiten.CursorPosition()
	a.Input = Input{
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
	_, a.Input.WheelDelta = ebiten.Wheel()
}

// Draw implements ebiten.Game. Every scene of the stack is drawn, bottom
// first, so dialogs appear over their screen.
func (a *App) Draw(screen *ebiten.Image) {
	for _, s := range a.stack {
		s.Draw(a, screen)
	}
	if a.shot != "" {
		if err := savePNG(screen, a.shot); err != nil {
			panic(err)
		}
		a.shot = ""
	}
	if a.scriptEnd && a.ShotPath == "" {
		a.quit = true
	}
	if a.ShotPath != "" && a.frame >= a.ShotFrame {
		if err := savePNG(screen, a.ShotPath); err != nil {
			panic(err)
		}
		a.ShotPath = ""
		a.quit = true
	}
}

// Layout implements ebiten.Game: the canvas always has the original size and
// Ebitengine scales it to the window.
func (a *App) Layout(int, int) (int, int) { return ScreenW, ScreenH }
