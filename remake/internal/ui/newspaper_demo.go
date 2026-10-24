package ui

import (
	"github.com/schowave/bmp/remake/internal/assets"
	"github.com/schowave/bmp/remake/internal/game"
)

// NewspaperDemo starts the demo game of Demo, plays until the n-th league
// match day of the manager (n >= 1) and shows his newspaper; closing it
// goes on with the game.
func NewspaperDemo(mana *assets.Mana, saveDir string, n int) Scene {
	s, err := game.NewGame(mana, game.Options{Managers: []game.NewManager{{Name: "TEST", Team: 9}}, Level: 2, Seed: 5})
	if err != nil {
		panic(err)
	}
	s.Start()
	ss := &Session{S: s, Mana: mana, SaveDir: saveDir}
	for i := 0; i < 400; i++ {
		switch s.Loop.Phase {
		case game.PhaseTurn:
			s.EndTurn()
		case game.PhaseMatches:
			for md := s.NextMatchDay(); md != nil; md = s.NextMatchDay() {
				md.Run()
				if md.HasNewspaper(0) {
					if n--; n <= 0 {
						// The flags and dialogs of the days played through
						// are not shown after the newspaper.
						s.TakeNotices()
						return NewNewspaper(ss, md, 0, func(app *App) { app.Reset(NewDriver(ss)) })
					}
				}
			}
		}
	}
	return NewDriver(ss)
}
