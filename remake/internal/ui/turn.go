package ui

import (
	"github.com/schowave/bmp/remake/internal/game"
)

// NewTurnMenu creates the main menu of the current manager's turn.
func NewTurnMenu(ss *Session) *MainMenu {
	m := &MainMenu{}
	m.Header = func() Header { return turnHeader(ss) }
	push := func(f func() Scene) func(*App) { return func(app *App) { app.Push(f()) } }
	m.Left[0] = MenuGroup{Icon: IconDesk, Items: [9]MenuItem{
		{IconFinanceChart, push(func() Scene { return NewFinanceScreen(ss) })},
		{IconHistory, push(func() Scene { return NewEternalScreen(ss) })},
		{IconTrophies, push(func() Scene { return NewHistoryScreen(ss) })},
		{IconStadium, push(func() Scene { return NewStadiumScreen(ss) })},
		{IconBank, push(func() Scene { return NewBankScreen(ss) })},
		{IconSponsors, push(func() Scene { return NewSponsorScreen(ss) })},
	}}
	m.Left[1] = MenuGroup{Icon: IconClub}
	for l := 0; l < game.NumLeagues; l++ {
		l := l
		m.Left[1].Items[l] = MenuItem{IconTable1 + Icon(l), push(func() Scene { return NewTableScreen(ss, l) })}
		m.Left[1].Items[3+l] = MenuItem{IconResults1 + Icon(l), push(func() Scene { return NewResultsScreen(ss, l) })}
		m.Left[1].Items[6+l] = MenuItem{IconStrength1 + Icon(l), push(func() Scene { return NewStrengthScreen(ss, l) })}
	}
	m.Left[2] = MenuGroup{Icon: IconTeam, Items: [9]MenuItem{
		{IconLineup, push(func() Scene { return NewSquadScreen(ss) })},
		{IconContracts, push(func() Scene { return NewContractScreen(ss, false) })},
		{IconTransfers, push(func() Scene { return NewTransferScreen(ss) })},
		{IconTraining, push(func() Scene { return NewTrainingScreen(ss) })},
		{IconCamp, nil},
		{IconPlayerInfo, push(func() Scene { return NewScorerScreen(ss) })},
	}}
	if ss.M().CampDays == 0 {
		m.Left[2].Items[4].Open = push(func() Scene { return NewCampScreen(ss) })
	}
	m.Right[0] = MenuGroup{Icon: IconCompetitions, Items: [9]MenuItem{
		{IconDFBPokal, push(func() Scene { return NewCupScreen(ss, game.CupDFB) })},
		{IconLandesmeister, push(func() Scene { return NewCupScreen(ss, game.CupLandesmeister) })},
		{IconPokalsieger, push(func() Scene { return NewCupScreen(ss, game.CupPokalsieger) })},
		{IconUEFACup, push(func() Scene { return NewCupScreen(ss, game.CupUEFA) })},
	}}
	m.Right[1] = MenuGroup{Icon: IconDisk, Items: [9]MenuItem{
		{IconSave, push(func() Scene { return NewFileScreen(ss, true) })},
		{IconLoad, push(func() Scene { return NewFileScreen(ss, false) })},
		{IconHighscore, push(func() Scene { return NewHighscoreScreen(ss) })},
		{IconOptions, push(func() Scene { return NewOptionsScreen(ss) })},
	}}
	endTurn := func(app *App) {
		ss.S.EndTurn()
		app.Pop()
	}
	m.Right[2] = MenuGroup{Icon: IconContinue, Action: endTurn}
	m.Calendar = endTurn
	return m
}

func turnHeader(ss *Session) Header {
	s := ss.S
	mg := ss.M()
	d := s.Date()
	h := Header{
		Face:     mg.Face,
		Matchday: s.EventLabel(ss.Mgr()),
		Manager:  mg.Name,
		Logo:     s.Teams[mg.Team].Logo,
		Weekday:  game.WeekdayNames[game.Weekday(s.Day)],
		Day:      d.Day,
		Month:    game.MonthNames[d.Month],
		Year:     d.Year,
	}
	if opp, home, ok := s.NextOpponent(ss.Mgr()); ok {
		where := "(A)"
		if home {
			where = "(H)"
		}
		h.Message[0] = "GEGEN " + s.Teams[opp].Name + " " + where
	}
	h.Message[1] = "KONTO: " + Money(mg.Money)
	return h
}

func posText(s *game.State, t int) string {
	if t >= game.NumGerman {
		return "-"
	}
	return itoa(s.Clubs[t].Pos + 1)
}
