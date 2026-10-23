package ui

import (
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
	l := ss.M().League
	switch name {
	case "table":
		return NewTableScreen(ss, l)
	case "results":
		return NewResultsScreen(ss, l)
	case "strength":
		return NewStrengthScreen(ss, l)
	case "squad":
		return NewSquadScreen(ss)
	case "info":
		return NewPlayerInfo(ss, 0, 3)
	case "transfer":
		return NewTransferScreen(ss)
	case "training":
		return NewTrainingScreen(ss)
	case "camp":
		return NewCampScreen(ss)
	case "contracts":
		return NewContractScreen(ss, false)
	case "scorers":
		return NewScorerScreen(ss)
	case "finance":
		return NewFinanceScreen(ss)
	case "eternal":
		return NewEternalScreen(ss)
	case "history":
		return NewHistoryScreen(ss)
	case "stadium":
		return NewStadiumScreen(ss)
	case "bank":
		return NewBankScreen(ss)
	case "sponsor":
		return NewSponsorScreen(ss)
	case "dfb":
		return NewCupScreen(ss, game.CupDFB)
	case "uefa":
		return NewCupScreen(ss, game.CupUEFA)
	case "highscore":
		return NewHighscoreScreen(ss)
	case "notices":
		return NoticeDemo(ss)
	case "options":
		return NewOptionsScreen(ss)
	case "load":
		return NewFileScreen(ss, false)
	case "save":
		return NewFileScreen(ss, true)
	}
	return NewTurnMenu(ss)
}
