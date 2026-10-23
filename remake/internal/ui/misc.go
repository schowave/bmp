package ui

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sort"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/schowave/bmp/remake/internal/game"
)

// highscores reads the high-score table of a game kind (HIGH.xy of the
// original; the remake keeps it as JSON in the save directory).
func highscores(dir, kind string) []game.HighscoreEntry {
	var hs []game.HighscoreEntry
	data, err := os.ReadFile(filepath.Join(dir, "high."+kind+".json"))
	if err == nil {
		_ = json.Unmarshal(data, &hs)
	}
	return hs
}

func saveHighscores(dir, kind string, hs []game.HighscoreEntry) {
	data, err := json.MarshalIndent(hs, "", " ")
	if err != nil {
		return
	}
	_ = os.MkdirAll(dir, 0o755)
	_ = os.WriteFile(filepath.Join(dir, "high."+kind+".json"), data, 0o644)
}

// enterHighscore puts e into the table like f_322b_2366(2): it replaces
// the entry of the same manager and club, or else the last of 20 when e
// has more points, and sorts the table. It returns the table and the row
// of e (-1 when it did not get in).
func enterHighscore(hs []game.HighscoreEntry, e game.HighscoreEntry) ([]game.HighscoreEntry, int) {
	hs = append([]game.HighscoreEntry(nil), hs...)
	at := -1
	for i := range hs {
		if hs[i].Manager == e.Manager && hs[i].Club == e.Club {
			at = i
			break
		}
	}
	switch {
	case at >= 0:
		hs[at] = e
	case len(hs) < 20:
		hs = append(hs, e)
	case hs[19].Score < e.Score:
		hs[19] = e
	default:
		return hs, -1
	}
	// The original swaps an entry down only when the other has more
	// points: a stable sort by descending points.
	sort.SliceStable(hs, func(a, b int) bool { return hs[a].Score > hs[b].Score })
	if len(hs) > 20 {
		hs = hs[:20]
	}
	for i := range hs {
		if hs[i] == e {
			return hs, i
		}
	}
	return hs, -1
}

// recordHighscores enters every manager into the table and saves it (the
// game end, @1a58:4161).
func recordHighscores(ss *Session) []game.HighscoreEntry {
	kind := ss.S.HighscoreKind()
	hs := highscores(ss.SaveDir, kind)
	for mi := range ss.S.Managers {
		hs, _ = enterHighscore(hs, ss.S.HighscoreEntry(mi))
	}
	saveHighscores(ss.SaveDir, kind, hs)
	return hs
}

// HighscoreScreen is the high-score table on the full-screen picture
// PIC/45.CP (f_322b_2366): 20 rows "NAME VEREIN PUNKTE: n(m/d/e)". A left
// click anywhere closes it.
type HighscoreScreen struct {
	hs []game.HighscoreEntry
	// mark tells the rows drawn in the highlight colour.
	mark func(row int, e game.HighscoreEntry) bool
	done func(app *App)
}

// NewHighscoreScreen shows the table from the main menu (f_322b_2366(1)):
// the current manager's entry as it would be now is put into it. In a
// timed game that row is highlighted. In an endless game the table is
// saved like that, and the original highlights the rows whose game id
// equals the current one (D:0x668); that id is never set, so every row is
// highlighted.
func NewHighscoreScreen(ss *Session) Scene {
	kind := ss.S.HighscoreKind()
	hs, row := enterHighscore(highscores(ss.SaveDir, kind), ss.S.HighscoreEntry(ss.Mgr()))
	mark := func(r int, _ game.HighscoreEntry) bool { return r == row }
	if ss.S.EndYear >= 2000 {
		saveHighscores(ss.SaveDir, kind, hs)
		mark = func(int, game.HighscoreEntry) bool { return true }
	}
	return &HighscoreScreen{hs: hs, mark: mark, done: func(app *App) { app.Pop() }}
}

func (h *HighscoreScreen) Update(app *App) error {
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
	for mi := range ss.S.Managers {
		e := ss.S.HighscoreEntry(mi)
		mine[[2]string{e.Manager, e.Club}] = true
	}
	o.list = &HighscoreScreen{hs: recordHighscores(ss),
		mark: func(_ int, e game.HighscoreEntry) bool { return mine[[2]string{e.Manager, e.Club}] }}
	return o
}

func (o *GameOver) Update(app *App) error {
	if app.Input.Left || app.Input.Right || app.Input.Enter || app.Input.Escape {
		app.Reset(NewTitle().OnDone(func() Scene { return NewSetup(o.ss.Mana, o.ss.SaveDir) }))
	}
	return nil
}

func (o *GameOver) Draw(app *App, dst *ebiten.Image) {
	g := app.Gfx
	o.list.Draw(app, dst)
	FillRect(dst, 100, 200, 120, 20, g.C(ColDarkRed))
	g.TextCentered(dst, FontBig, 100, 202, 120, "ENDE", g.C(ColBeige))
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
