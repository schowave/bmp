package ui

import (
	"github.com/schowave/bmp/remake/internal/assets"
	"github.com/schowave/bmp/remake/internal/game"
)

// NewspaperDemo starts the demo game of Demo at club team, plays until the n-th league
// match day of the manager (n >= 1) and shows his newspaper; closing it
// goes on with the game.
func NewspaperDemo(mana *assets.Mana, saveDir string, team, n int) Scene {
	s, err := game.NewGame(mana, game.Options{Managers: []game.NewManager{{Name: "TEST", Team: team}}, Level: 2, Seed: 5})
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
						return NewNewspaper(ss, md, 0, func(app *App) { app.Reset(NewDriver(ss)) })
					}
				}
			}
		}
	}
	return NewDriver(ss)
}
