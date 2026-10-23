package ui

import (
	"slices"

	"github.com/schowave/bmp/remake/internal/assets"
	"github.com/schowave/bmp/remake/internal/game"
)

// Demo returns a scene for screenshots during development
// (bmp -demo NAME -shot FILE). Most demos start a game with one manager
// at Borussia Dortmund and play matchdays first.
func Demo(name string, mana *assets.Mana, saveDir string) Scene {
	if name == "setup" {
		return NewSetup(mana, saveDir)
	}
	s, err := game.NewGame(mana, game.Options{Managers: []game.NewManager{{Name: "TEST", Team: 9}}, Level: 2, Seed: 5})
	if err != nil {
		panic(err)
	}
	s.Start()
	ss := &Session{S: s, Mana: mana, SaveDir: saveDir}
	switch name {
	case "dfb0":
		return NewCupScreen(ss, game.CupDFB)
	case "uefa0":
		return NewCupScreen(ss, game.CupUEFA)
	case "start":
		return NewDriver(ss)
	case "newspaper":
		return NewspaperDemo(mana, saveDir, 1)
	}
	// Play some events so that the screens have content.
	for i := 0; i < 30 && name != "turn"; i++ {
		switch s.Loop.Phase {
		case game.PhaseTurn:
			s.EndTurn()
		case game.PhaseMatches:
			if name == "match" && i > 20 {
				md := s.NextMatchDay()
				if md != nil {
					for k := 0; k < 70; k++ {
						md.Step()
					}
					return NewMatchScreen(ss, md)
				}
			}
			for md := s.NextMatchDay(); md != nil; md = s.NextMatchDay() {
				md.Run()
			}
		}
	}
	for s.Loop.Phase != game.PhaseTurn {
		if s.Loop.Phase == game.PhaseMatches {
			for md := s.NextMatchDay(); md != nil; md = s.NextMatchDay() {
				md.Run()
			}
		}
	}
	if d, ok := playedDemos[name]; ok {
		return d(ss)
	}
	return NewTurnMenu(ss)
}

// playedDemos are the demos of screens shown after some matchdays.
var playedDemos = map[string]func(ss *Session) Scene{
	"menu":      func(ss *Session) Scene { return NewTurnMenu(ss) },
	"table":     func(ss *Session) Scene { return NewTableScreen(ss, ss.M().League) },
	"results":   func(ss *Session) Scene { return NewResultsScreen(ss, ss.M().League) },
	"strength":  func(ss *Session) Scene { return NewStrengthScreen(ss, ss.M().League) },
	"squad":     func(ss *Session) Scene { return NewSquadScreen(ss) },
	"info":      func(ss *Session) Scene { return NewPlayerInfo(ss, 0, 3) },
	"transfer":  func(ss *Session) Scene { return NewTransferScreen(ss) },
	"training":  func(ss *Session) Scene { return NewTrainingScreen(ss) },
	"camp":      func(ss *Session) Scene { return NewCampScreen(ss) },
	"contracts": func(ss *Session) Scene { return NewContractScreen(ss, false) },
	"scorers":   func(ss *Session) Scene { return NewScorerScreen(ss) },
	"finance":   func(ss *Session) Scene { return NewFinanceScreen(ss) },
	"eternal":   func(ss *Session) Scene { return NewEternalScreen(ss) },
	"history":   func(ss *Session) Scene { return NewHistoryScreen(ss) },
	"stadium":   func(ss *Session) Scene { return NewStadiumScreen(ss) },
	"bank":      func(ss *Session) Scene { return NewBankScreen(ss) },
	"sponsor":   func(ss *Session) Scene { return NewSponsorScreen(ss) },
	"dfb":       func(ss *Session) Scene { return NewCupScreen(ss, game.CupDFB) },
	"uefa":      func(ss *Session) Scene { return NewCupScreen(ss, game.CupUEFA) },
	"highscore": func(ss *Session) Scene { return NewHighscoreScreen(ss) },
	"notices":   func(ss *Session) Scene { return NoticeDemo(ss) },
	"options":   func(ss *Session) Scene { return NewOptionsScreen(ss) },
	"load":      func(ss *Session) Scene { return NewFileScreen(ss, false) },
	"save":      func(ss *Session) Scene { return NewFileScreen(ss, true) },
}

// DemoNames returns the names Demo knows, sorted.
func DemoNames() []string {
	names := []string{"setup", "dfb0", "uefa0", "start", "newspaper", "turn", "match"}
	for n := range playedDemos {
		names = append(names, n)
	}
	slices.Sort(names)
	return names
}
