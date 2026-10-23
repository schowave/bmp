package ui

import (
	"strings"

	"github.com/schowave/bmp/remake/internal/game"
)

// NewTurnMenu creates the main menu of the current manager's turn.
func NewTurnMenu(ss *Session) *MainMenu {
	m := &MainMenu{}
	m.Header = func() Header { return turnHeader(ss) }
	m.Messages = &MessagePanel{Messages: func() [][]string {
		var out [][]string
		for _, msg := range ss.S.Messages(ss.Mgr()) {
			d := game.DateOf(ss.S.StartYear, msg.Day)
			date := itoa(d.Day) + ". " + game.MonthNames[d.Month] + " " + itoa(d.Year)
			out = append(out, append([]string{date}, strings.Split(msg.Text, "\n")...))
		}
		return out
	}}
	m.NewMessages = func() bool { return ss.S.TakeNewMessages(ss.Mgr()) }
	m.Key = func(app *App, c rune) bool {
		if c != 'o' {
			return false
		}
		m.Note = optimizeTurn(ss)
		return true
	}
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
		{IconContracts, push(func() Scene { return NewSquadContracts(ss) })},
		{IconTransfers, push(func() Scene { return NewTransferScreen(ss) })},
		{IconTraining, push(func() Scene { return NewTrainingScreen(ss) })},
		// The camp shows the cobweb from the booking until CampDays runs out.
		{IconCamp, push(func() Scene { return NewCampScreen(ss) })},
		{IconPlayerInfo, push(func() Scene { return NewScorerScreen(ss) })},
	}}
	m.Left[2].Shut = func(k int) bool { return k == 4 && ss.M().CampDays != 0 }
	m.Right[0] = MenuGroup{Icon: IconCompetitions, Items: [9]MenuItem{
		{IconDFBPokal, push(func() Scene { return NewCupScreen(ss, game.CupDFB) })},
		{IconLandesmeister, push(func() Scene { return NewCupScreen(ss, game.CupLandesmeister) })},
		{IconPokalsieger, push(func() Scene { return NewCupScreen(ss, game.CupPokalsieger) })},
		{IconUEFACup, push(func() Scene { return NewCupScreen(ss, game.CupUEFA) })},
	}}
	m.Right[1] = MenuGroup{Icon: IconDisk, Items: [9]MenuItem{
		{IconLoad, push(func() Scene { return NewFileScreen(ss, false) })},
		{IconSave, push(func() Scene { return NewFileScreen(ss, true) })},
		{IconHighscore, push(func() Scene { return NewHighscoreScreen(ss) })},
		{IconOptions, push(func() Scene { return NewOptionsScreen(ss) })},
		{IconNew, push(func() Scene { return NewAddManagers(ss) })},
	}}
	// A right click on the manager tearing his hair asks whether to quit,
	// a left click does nothing; the turn ends on the calendar sheet.
	m.Right[2] = MenuGroup{Icon: IconContinue, RightAction: func(app *App, done func()) {
		app.Push(NewQuestion(ss.M().Name, "MöCHTEN SIE\nDAS SPIEL WIRKLICH\nBEENDEN ?", "LEIDER JA", "KEIN GEDANKE",
			func(app *App, yes bool) {
				done()
				if yes {
					app.Quit()
				}
			}))
	}}
	m.Calendar = func(app *App) {
		ss.S.EndTurn()
		app.Pop()
	}
	return m
}

func turnHeader(ss *Session) Header {
	s := ss.S
	mg := ss.M()
	d := s.Date()
	played := s.Matchday[mg.League] - 1
	if played > len(mg.PosAfter) {
		played = len(mg.PosAfter)
	}
	if s.Day <= 1 {
		played = 0 // the curve starts on the second day, before it a dark line
	}
	h := Header{
		Places:   mg.PosAfter[:max(played, 0)],
		Chart:    true,
		Face:     mg.Face,
		Matchday: s.EventLabel(ss.Mgr()),
		Manager:  mg.Name,
		Logo:     s.Teams[mg.Team].Logo,
		Weekday:  game.WeekdayNames[game.Weekday(s.Day)],
		Day:      d.Day,
		Month:    game.MonthNames[d.Month],
		Year:     d.Year,
	}
	return h
}

// optimizeTurn is the hidden key o of the main menu (the remake's): it sets
// the optimal training (game.State.OptimizeTraining), the optimal lineup
// (game.State.OptimizeLineup) and the optimal ticket price
// (game.State.OptimizeTicketPrice), and returns the note of what each is
// worth and gained against the one before, and the opponent of the lineup.
func optimizeTurn(ss *Session) []string {
	s, mi := ss.S, ss.Mgr()
	tr := s.OptimizeTraining(mi)
	ln := s.OptimizeLineup(mi)
	tp := s.OptimizeTicketPrice(mi)
	note := []string{
		"AUFSTELLUNG " + pointsText(ln.After) + " " + signedDecimal(ln.After-ln.Before, 2),
		"TRAINING " + tenthsText(tr.After) + " " + signedDecimal(tr.After-tr.Before, 1),
	}
	if tp.Opponent >= 0 {
		note = append(note, priceLine(tp))
	}
	if ln.Opponent >= 0 {
		note = append(note, "GEGEN "+s.Teams[ln.Opponent].Name)
	}
	return note
}
