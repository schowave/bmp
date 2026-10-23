package ui

import (
	"github.com/hajimehoshi/ebiten/v2"

	"github.com/schowave/bmp/remake/internal/game"
)

var posShort = [4]string{"TOR", "ABW", "MIT", "ANG"}

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
	g.TextCentered(dst, FontNormal, x, y+4, w, "Info über "+ps.Name, g.PalColor(ColBeige))
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
		g.Text(dst, FontSmall, x+8, y+18+i*10, l, g.PalColor(ColLight))
	}
}

func teamName(s *game.State, t int) string {
	if t < 0 || t >= game.NumTeams {
		return "-"
	}
	return s.Teams[t].Name
}
