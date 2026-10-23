package ui

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/schowave/bmp/remake/internal/game"
)

// loadHighscores reads the high-score table of the game from the save
// directory; without the file the table is empty, like the original
// starts a zeroed one.
func loadHighscores(ss *Session) ([]game.HighscoreEntry, error) {
	data, err := os.ReadFile(filepath.Join(ss.SaveDir, ss.S.HighscoreFile()))
	if errors.Is(err, fs.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return game.DecodeHighscores(data)
}

func saveHighscores(ss *Session, hs []game.HighscoreEntry) error {
	if err := os.MkdirAll(ss.SaveDir, 0o755); err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(ss.SaveDir, ss.S.HighscoreFile()), game.EncodeHighscores(hs), 0o644)
}

// recordHighscores enters es into the table and saves it, at every season
// end (f_322b_2366(2)) and at the game end (@1a58:4161). It returns the
// table; a table that cannot be read is not overwritten. Entries that
// could not be saved are kept and entered first the next time.
func recordHighscores(ss *Session, es []game.HighscoreEntry) ([]game.HighscoreEntry, error) {
	es = append(ss.unsaved, es...)
	ss.unsaved = nil
	hs, err := loadHighscores(ss)
	for _, e := range es {
		hs, _ = game.EnterHighscore(hs, e)
	}
	if err == nil {
		err = saveHighscores(ss, hs)
	}
	if err != nil {
		ss.unsaved = es
	}
	return hs, err
}

// HighscoreScreen is the high-score table on the full-screen picture
// PIC/45.CP (f_322b_2366): 20 rows "NAME VEREIN PUNKTE: n(m/d/e)". A left
// click anywhere closes it.
type HighscoreScreen struct {
	hs []game.HighscoreEntry
	// mark tells the rows drawn in the highlight colour.
	mark func(row int, e game.HighscoreEntry) bool
	done func(app *App)
	// err is shown over the table: the file could not be read or saved.
	err error
}

// NewHighscoreScreen shows the table from the main menu (f_322b_2366(1)):
// the current manager's entry as it would be now is put into it. In a
// timed game that row is highlighted. In an endless game the table is
// saved like that, and the original highlights the rows whose game id
// equals the current one (D:0x668); that id is never set, so every row is
// highlighted.
func NewHighscoreScreen(ss *Session) Scene {
	hs, err := loadHighscores(ss)
	hs, row := game.EnterHighscore(hs, ss.S.HighscoreEntry(ss.Mgr()))
	mark := func(r int, _ game.HighscoreEntry) bool { return r == row }
	if ss.S.EndYear >= 2000 {
		if err == nil {
			err = saveHighscores(ss, hs)
		}
		mark = func(int, game.HighscoreEntry) bool { return true }
	}
	return &HighscoreScreen{hs: hs, mark: mark, err: err, done: func(app *App) { app.Pop() }}
}

func (h *HighscoreScreen) Update(app *App) error {
	if h.err != nil {
		h.err = nil
		app.Push(diskError())
		return nil
	}
	if app.Input.Left || app.Input.Escape || app.Input.Enter {
		h.done(app)
	}
	return nil
}

func (h *HighscoreScreen) Draw(app *App, dst *ebiten.Image) {
	g := app.Gfx
	Draw(dst, g.Pic(45, true), 0, 0)
	for r, e := range h.hs {
		if r >= 20 {
			break
		}
		if e.Score == 0 {
			continue
		}
		y := r*10 + 32
		c := ColLight
		if h.mark != nil && h.mark(r, e) {
			c = ColGrey3
		}
		g.TextAt(dst, FontSmall, 56, y, e.Manager, c, NoShadow)
		g.TextAt(dst, FontSmall, 104, y, e.Club, c, NoShadow)
		// The number is padded to five places (D:0x79c = 5): it ends where
		// the titles begin.
		g.TextAt(dst, FontSmall, 187, y, "PUNKTE:", c, NoShadow)
		g.TextEnd(dst, FontSmall, 243, y, itoa(e.Score), c, NoShadow)
		g.TextAt(dst, FontSmall, 243, y, "("+itoa(e.Titles[0])+"/"+itoa(e.Titles[1])+"/"+itoa(e.Titles[2])+")", c, NoShadow)
	}
}

// GameOver is shown when the chosen game length is over: the high-score
// table with the managers of this game highlighted, then "ENDE".
type GameOver struct {
	ss   *Session
	list *HighscoreScreen
}

// NewGameOver records the high scores and shows them.
func NewGameOver(ss *Session) Scene {
	o := &GameOver{ss: ss}
	mine := map[[2]string]bool{}
	var es []game.HighscoreEntry
	for mi := range ss.S.Managers {
		e := ss.S.HighscoreEntry(mi)
		mine[[2]string{e.Manager, e.Club}] = true
		es = append(es, e)
	}
	hs, err := recordHighscores(ss, es)
	o.list = &HighscoreScreen{hs: hs, err: err,
		mark: func(_ int, e game.HighscoreEntry) bool { return mine[[2]string{e.Manager, e.Club}] }}
	return o
}

func (o *GameOver) Update(app *App) error {
	if o.list.err != nil {
		o.list.err = nil
		app.Push(diskError())
		return nil
	}
	if app.Input.Left || app.Input.Right || app.Input.Enter || app.Input.Escape {
		app.Reset(NewTitle().OnDone(func() Scene { return NewSetup(o.ss.Mana, o.ss.SaveDir) }))
	}
	return nil
}

func (o *GameOver) Draw(app *App, dst *ebiten.Image) {
	g := app.Gfx
	o.list.Draw(app, dst)
	FillRect(dst, 100, 200, 120, 20, g.PalColor(ColDarkRed))
	g.TextCentered(dst, FontBig, 100, 202, 120, "ENDE", g.PalColor(ColBeige))
}

// AlertBox is the small dark red box of f_3091_0e3a: four centred lines
// and an OKAY button that alone closes it.
type AlertBox struct {
	Lines [4]string
	// Done, when set, is called after the box closed.
	Done func(app *App)
	ok   Button
}

// NewAlert creates an alert box with up to four lines.
func NewAlert(lines ...string) *AlertBox {
	a := &AlertBox{ok: Button{X: 178, Y: 160, W: 57, H: 12, Label: "OKAY"}}
	copy(a.Lines[:], lines)
	return a
}

func (a *AlertBox) Update(app *App) error {
	if a.ok.Hit(&app.Input) || app.Input.Enter || app.Input.Escape {
		app.Pop()
		if a.Done != nil {
			a.Done(app)
		}
	}
	return nil
}

func (a *AlertBox) Draw(app *App, dst *ebiten.Image) {
	g := app.Gfx
	g.Window(dst, 60, 110, 245, 175, ColDarkRed)
	for i, l := range a.Lines {
		g.TextMid(dst, FontNormal, 60, 245, 123+i*10, l, ColSand, NoShadow)
	}
	drawDialogButton(app, dst, &a.ok)
}

// drawDialogButton draws a button of f_3091_123d: black with a bevel and
// the label centred between x+1 and x+57 in colour 26.
func drawDialogButton(app *App, dst *ebiten.Image, b *Button) {
	g := app.Gfx
	g.BevelButton(dst, b.X, b.Y, b.W, b.H, ColBlack)
	g.TextMid(dst, FontSmall, b.X+1, b.X+b.W, b.Y+8, b.Label, colButtonText, NoShadow)
}

// colButtonText is the label colour of the buttons of f_3091_123d.
const colButtonText = 26
