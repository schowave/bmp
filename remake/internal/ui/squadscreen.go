package ui

import (
	"github.com/hajimehoshi/ebiten/v2"

	"github.com/schowave/bmp/remake/internal/game"
)

var posShort = [4]string{"TOR", "ABW", "MIT", "ANG"}

// squadColumns are the columns of the squad list (f_1ecd_06af, reduced).
func squadColumns() []Column {
	return []Column{
		{"NR", 2, 10, Right}, {"POS", 15, 16, Left}, {"NAME", 33, 66, Left},
		{"SP", 101, 14, Right}, {"K", 118, 12, Right}, {"T", 132, 12, Right}, {"F", 146, 12, Right},
		{"ST", 160, 14, Right}, {"TO", 176, 12, Right}, {"KA", 190, 16, Center},
		{"STATUS", 208, 30, Left}, {"TD", 240, 8, Center}, {"VT", 250, 8, Right}, {"GEHALT", 260, 48, Right},
	}
}

// squadRow builds the row of a slot.
func squadRow(s *game.State, sl *game.Slot) Row {
	p := &s.Persons[sl.Person]
	nr := ""
	switch {
	case sl.Lineup >= 1 && sl.Lineup <= 11:
		nr = itoa(sl.Lineup)
	case sl.Lineup >= 12:
		nr = "R"
	}
	status := ""
	switch {
	case sl.Status&2 != 0:
		status = "VERL." + itoa(sl.Out)
	case sl.Status&1 != 0:
		status = "GESP." + itoa(sl.Out)
	case sl.Loan != 0:
		status = "LEIH"
	case sl.Nego&0x80 != 0:
		status = "HÖRT AUF"
	}
	td := [3]string{"-", "O", "+"}[game.TendencyClass(sl.Tendency)]
	games := sl.Games[0] + sl.Games[1] + sl.Games[2]
	goals := sl.Goals[0] + sl.Goals[1] + sl.Goals[2]
	row := Row{Cells: []string{nr, posShort[p.Position()], p.Name, itoa(games),
		itoa(sl.Skill[0]), itoa(sl.Skill[1]), itoa(sl.Skill[2]), itoa(sl.Strength()), itoa(goals),
		itoa(sl.Cards[1]) + " " + itoa(sl.Cards[0]), status, td, itoa(sl.Years), formatMoney(sl.Salary)}}
	switch {
	case sl.Lineup >= 1 && sl.Lineup <= 11:
		row.Color = ColWhite
	case sl.Lineup >= 12:
		row.Color = ColBeige
	}
	if sl.Status&3 != 0 {
		row.Color = ColRed
	}
	return row
}

// PlayerInfo shows the details of a player (f_14a4_0906).
type PlayerInfo struct {
	ss     *Session
	m, i   int
	market bool
}

// NewPlayerInfo creates the info screen of slot i of squad m.
func NewPlayerInfo(ss *Session, m, i int) *PlayerInfo { return &PlayerInfo{ss: ss, m: m, i: i} }

func (p *PlayerInfo) Update(app *App) error {
	if app.Input.Left || app.Input.Right || app.Input.Escape || app.Input.Enter {
		app.Pop()
	}
	return nil
}

func (p *PlayerInfo) Draw(app *App, dst *ebiten.Image) {
	g := app.Gfx
	s := p.ss.S
	sl := &s.Squads[p.m][p.i]
	ps := &s.Persons[sl.Person]
	x, y, w, h := 30, 20, 260, 200
	g.Frame(dst, x, y, w, h)
	g.Panel(dst, x, y, w, h)
	g.TextCentered(dst, FontNormal, x, y+4, w, "Info über "+ps.Name, g.C(ColBeige))
	lines := []string{
		ps.Name + " IST " + itoa(ps.Age) + " JAHRE ALT " + game.AgeLabel(ps.Age),
		game.PositionNames[ps.Position()] + game.SideText(ps.Side),
		"VEREIN: " + teamName(s, ps.Team),
		"",
		"KONDITION " + itoa(sl.Skill[0]) + "   TECHNIK " + itoa(sl.Skill[1]) + "   FORM " + itoa(sl.Skill[2]),
		"DURCHSCHNITTS-STÄRKE " + itoa(sl.Strength()),
		"TENDENZ: " + game.TendencyNames[game.TendencyClass(sl.Tendency)] + "   ERSCHÖPFUNG: " + itoa(sl.Fatigue),
		"",
		"SPIELE: LIGA " + itoa(sl.Games[0]) + "  POKAL " + itoa(sl.Games[1]) + "  EUROPA " + itoa(sl.Games[2]),
		"TORE:   LIGA " + itoa(sl.Goals[0]) + "  POKAL " + itoa(sl.Goals[1]) + "  EUROPA " + itoa(sl.Goals[2]),
		"KARRIERE: " + itoa(sl.Career[0]+sl.Career[1]+sl.Career[2]) + " SPIELE, " +
			itoa(sl.CareerGoals[0]+sl.CareerGoals[1]+sl.CareerGoals[2]) + " TORE",
		"KARTEN: GELB " + itoa(sl.Cards[1]) + "  ROT " + itoa(sl.Cards[0]) + "  GELB-ROT " + itoa(sl.Cards[2]),
		"",
		"VERTRAG: " + itoa(sl.Years) + " JAHR(E), GEHALT " + Money(sl.Salary) + "/MONAT",
		"MARKTWERT: " + Money(s.PlayerValue(p.m, p.i, 0)),
	}
	switch {
	case sl.Status&2 != 0:
		lines = append(lines, "STATUS: VERLETZT ("+game.InjuryNames[sl.Injury]+"), NOCH "+itoa(sl.Out)+" WOCHEN")
	case sl.Status&1 != 0:
		lines = append(lines, "STATUS: GESPERRT, NOCH "+itoa(sl.Out)+" SPIELE")
	case sl.Lineup >= 1 && sl.Lineup <= 11:
		lines = append(lines, "STATUS: IST FÜR'S NÄCHSTE SPIEL EINGEPLANT (NR. "+itoa(sl.Lineup)+")")
	case sl.Lineup >= 12:
		lines = append(lines, "STATUS: HÜTET DIE ERSATZBANK")
	default:
		lines = append(lines, "STATUS: HÄNGT HALT SO RUM...")
	}
	for i, l := range lines {
		g.Text(dst, FontSmall, x+8, y+18+i*10, l, g.C(ColLight))
	}
}

func teamName(s *game.State, t int) string {
	if t < 0 || t >= game.NumTeams {
		return "-"
	}
	return s.Teams[t].Name
}

// SubstitutionScreen lets a manager substitute players during a match.
type SubstitutionScreen struct {
	ss    *Session
	md    *game.MatchDay
	mi    int
	table *TableView
	out   int
	msg   string
}

// NewSubstitutionScreen creates the substitution screen.
func NewSubstitutionScreen(ss *Session, md *game.MatchDay, mi int) *SubstitutionScreen {
	t := NewTableView(2, 14, 316, 176, squadColumns())
	return &SubstitutionScreen{ss: ss, md: md, mi: mi, table: t, out: -1}
}

func (u *SubstitutionScreen) Update(app *App) error {
	s := u.ss.S
	in := &app.Input
	u.table.Rows = u.table.Rows[:0]
	for i := 0; i < s.SquadSize(u.mi); i++ {
		u.table.Rows = append(u.table.Rows, squadRow(s, &s.Squads[u.mi][i]))
	}
	u.table.Selected = u.out
	left, _ := u.table.Update(in)
	if left >= 0 {
		sl := &s.Squads[u.mi][left]
		if u.out < 0 {
			if sl.Lineup >= 1 && sl.Lineup <= 11 {
				u.out = left
				u.msg = "Wer kommt für " + s.Persons[sl.Person].Name + "?"
			} else {
				u.msg = "Spieler ist nicht aufgestellt"
			}
			return nil
		}
		if err := u.md.Substitute(u.mi, u.out, left); err != nil {
			u.msg = err.Error()
		} else {
			u.msg = "Auswechslung durchgeführt."
		}
		u.out = -1
		return nil
	}
	if MenuButtonHit(in) {
		app.Pop()
	}
	return nil
}

func (u *SubstitutionScreen) Draw(app *App, dst *ebiten.Image) {
	g := app.Gfx
	g.Background(dst)
	g.Text(dst, FontNormal, 4, 3, "AUSWECHSELN - "+u.ss.S.Managers[u.mi].Name, g.C(ColBeige))
	u.table.Draw(g, dst)
	msg := u.msg
	if msg == "" {
		msg = "Spieler auswählen, der ausgewechselt werden soll."
	}
	g.Text(dst, FontSmall, 4, 196, msg, g.C(ColWhite))
	g.DrawMenuButton(dst)
}
