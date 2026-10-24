package ui

import (
	"github.com/hajimehoshi/ebiten/v2"

	"github.com/schowave/bmp/remake/internal/assets"
	"github.com/schowave/bmp/remake/internal/game"
)

// Session is a running game with what the screens need besides the state.
type Session struct {
	S       *game.State
	Mana    *assets.Mana
	SaveDir string
	// unsaved are the high-score entries a season end could not save;
	// they are entered again with the next ones.
	unsaved []game.HighscoreEntry
}

// Mgr returns the index of the manager whose turn it is.
func (ss *Session) Mgr() int { return ss.S.Loop.Current }

// M returns the manager whose turn it is.
func (ss *Session) M() *game.Manager { return ss.S.Managers[ss.Mgr()] }

// Driver is the bottom scene of a running game. Whenever it is on top it
// moves the game loop on and opens the screen the loop is waiting for.
type Driver struct {
	ss *Session
}

// NewDriver creates the driver of a session.
func NewDriver(ss *Session) *Driver { return &Driver{ss: ss} }

func (d *Driver) Update(app *App) error {
	s := d.ss.S
	if es := s.TakeHighscores(); len(es) > 0 {
		if _, err := recordHighscores(d.ss, es); err != nil {
			app.Push(diskError())
			return nil
		}
	}
	// The dialogs and screens the game queued (flags, cup draws, credit
	// due, cards, trophies, the season report ...) come first, in the order
	// of the original.
	if s.HasNotices() {
		ShowNotices(app, d.ss, nil)
		return nil
	}
	switch s.Loop.Phase {
	case game.PhaseTurn:
		app.Push(NewTurnMenu(d.ss))
	case game.PhaseMatches:
		// NextMatchDay queues the flag with the competition and a forfeit;
		// the matches follow them.
		if md := s.NextMatchDay(); md != nil {
			ShowNotices(app, d.ss, func(app *App) { app.Push(NewMatchScreen(d.ss, md)) })
		}
	case game.PhaseContracts:
		app.Push(NewContractScreen(d.ss, true))
	case game.PhaseGameOver:
		app.Push(NewGameOver(d.ss))
	}
	return nil
}

// Passing implements passer: the driver only opens the next screen.
func (d *Driver) Passing() bool { return true }

func (d *Driver) Draw(app *App, dst *ebiten.Image) {
	app.Gfx.Background(dst)
}
