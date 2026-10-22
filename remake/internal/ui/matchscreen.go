package ui

import (
	"github.com/hajimehoshi/ebiten/v2"

	"github.com/schowave/bmp/remake/internal/game"
)

// MatchScreen shows a match day live: minute, scores of all matches and
// the events of the human teams' matches.
type MatchScreen struct {
	ss        *Session
	md        *game.MatchDay
	table     *TableView
	speed     int // frames per minute
	tick      int
	seen      int // events already shown
	banner    []string
	bannerT   int
	log       []string
	paused    bool
	rowsOrder []int

	speedBtn Button
	subBtn   Button
	nextBtn  Button
	tableBtn Button
}

var speeds = []int{12, 6, 3, 1}

// NewMatchScreen creates the screen for a match day.
func NewMatchScreen(ss *Session, md *game.MatchDay) *MatchScreen {
	cols := []Column{{"", 4, 116, Right}, {"", 122, 8, Center}, {"", 132, 116, Left}, {"", 252, 26, Center}, {"", 280, 26, Left}}
	t := NewTableView(4, 24, 312, 112, cols)
	t.NoHeader = true
	t.RowH = 7
	m := &MatchScreen{ss: ss, md: md, table: t, speed: speeds[1]}
	m.speedBtn = Button{X: 4, Y: 222, W: 60, H: 13}
	m.subBtn = Button{X: 70, Y: 222, W: 76, H: 13, Label: "AUSWECHSELN"}
	m.tableBtn = Button{X: 152, Y: 222, W: 60, H: 13, Label: "TABELLE"}
	m.nextBtn = Button{X: 252, Y: 222, W: 64, H: 13, Label: "WEITER"}
	m.order()
	return m
}

// order sorts the fixtures: matches of human teams first, then by league.
func (m *MatchScreen) order() {
	s := m.ss.S
	var rows []int
	for pass := 0; pass < 2; pass++ {
		for i, f := range m.md.Fixtures {
			human := s.ManagerOf(f.Home) != nil || s.ManagerOf(f.Away) != nil
			if (pass == 0) == human {
				rows = append(rows, i)
			}
		}
	}
	m.rowsOrder = rows
}

func (m *MatchScreen) humanManagerPlaying() int {
	s := m.ss.S
	for _, f := range m.md.Fixtures {
		if mg := s.ManagerOf(f.Home); mg != nil {
			return indexOfManager(s, mg)
		}
		if mg := s.ManagerOf(f.Away); mg != nil {
			return indexOfManager(s, mg)
		}
	}
	return -1
}

func indexOfManager(s *game.State, m *game.Manager) int {
	for i, x := range s.Managers {
		if x == m {
			return i
		}
	}
	return -1
}

func (m *MatchScreen) Update(app *App) error {
	in := &app.Input
	m.table.Update(in)
	if m.speedBtn.Hit(in) {
		for i, sp := range speeds {
			if sp == m.speed {
				m.speed = speeds[(i+1)%len(speeds)]
				break
			}
		}
		return nil
	}
	if m.md.Done {
		if m.tableBtn.Hit(in) {
			if l := m.league(); l >= 0 {
				app.Push(NewTableScreen(m.ss, l))
			}
			return nil
		}
		if m.nextBtn.Hit(in) || in.Right || in.Enter {
			app.Pop()
		}
		return nil
	}
	if m.subBtn.Hit(in) {
		if mi := m.humanManagerPlaying(); mi >= 0 {
			app.Push(NewSubstitutionScreen(m.ss, m.md, mi))
		}
		return nil
	}
	if m.bannerT > 0 {
		m.bannerT--
		if in.Left || in.Right {
			m.bannerT = 0
		}
		return nil
	}
	if in.Right {
		// fast forward to the end
		m.md.Run()
		m.collect(app)
		return nil
	}
	m.tick++
	if m.tick < m.speed {
		return nil
	}
	m.tick = 0
	m.md.Step()
	m.collect(app)
	return nil
}

// collect turns new events into the log and banners.
func (m *MatchScreen) collect(app *App) {
	s := m.ss.S
	for ; m.seen < len(m.md.Events); m.seen++ {
		e := m.md.Events[m.seen]
		f := m.md.Fixtures[e.Fixture]
		att := f.Home
		if e.Side == 1 {
			att = f.Away
		}
		if !f.Watched {
			continue
		}
		var text string
		switch e.Kind {
		case game.EvGoal, game.EvMiss:
			text = itoa(e.Minute) + ". Chance für " + s.Teams[att].Name + " !"
			if e.Player != "" {
				text += " " + m.md.EventText(e)
			} else if e.Kind == game.EvGoal {
				text += " TOR !"
			}
		case game.EvPenaltyGoal, game.EvPenaltyMiss:
			text = "Elfmeter für " + s.Teams[att].Name + ": " + map[bool]string{true: "TOR", false: "verschossen"}[e.Kind == game.EvPenaltyGoal]
		default:
			text = itoa(e.Minute) + ". " + m.md.EventText(e)
		}
		m.log = append(m.log, text)
		if e.Manager >= 0 || e.Kind == game.EvGoal {
			m.banner = []string{text}
			m.bannerT = 50
		}
		if e.Kind == game.EvInjury && e.Manager >= 0 && !m.md.Done {
			app.Push(NewSubstitutionScreen(m.ss, m.md, e.Manager))
		}
	}
}

// league returns the league shown in the table button, or -1.
func (m *MatchScreen) league() int {
	if mi := m.humanManagerPlaying(); mi >= 0 {
		l := m.ss.S.Managers[mi].League
		for _, f := range m.md.Fixtures {
			if f.League == l {
				return l
			}
		}
	}
	for _, f := range m.md.Fixtures {
		if f.League >= 0 {
			return f.League
		}
	}
	return -1
}

func (m *MatchScreen) Draw(app *App, dst *ebiten.Image) {
	g := app.Gfx
	s := m.ss.S
	g.Background(dst)
	g.Panel(dst, 4, 3, 312, 17)
	g.Text(dst, FontNormal, 10, 7, m.md.Title, g.C(ColBeige))
	minute := m.md.Minute
	if m.md.Done {
		g.TextRight(dst, FontNormal, 310, 7, "ENDE", g.C(ColYellow))
	} else if minute > 0 {
		g.TextRight(dst, FontNormal, 310, 7, "MINUTE "+itoa(minute-1), g.C(ColWhite))
	}
	m.table.Rows = m.table.Rows[:0]
	for _, i := range m.rowsOrder {
		f := m.md.Fixtures[i]
		h, a := m.md.Score(i)
		res := itoa(h) + ":" + itoa(a)
		extra := ""
		if m.md.Penalties(i) {
			extra = "n.E."
		} else if m.md.ExtraTime(i) {
			extra = "n.V."
		}
		if f.Forfeit != 0 {
			extra = "(W)"
		}
		row := Row{Cells: []string{s.Teams[f.Home].Name, "-", s.Teams[f.Away].Name, res, extra}}
		if s.ManagerOf(f.Home) != nil || s.ManagerOf(f.Away) != nil {
			row.Color = ColYellow
		}
		m.table.Rows = append(m.table.Rows, row)
	}
	m.table.Draw(g, dst)
	g.Panel(dst, 4, 140, 312, 78)
	g.Text(dst, FontSmall, 8, 142, "SPIELBERICHT", g.C(ColBeige))
	lines := []string{}
	for _, l := range m.log {
		lines = append(lines, g.Wrap(FontSmall, l, 300)...)
	}
	if len(lines) > 9 {
		lines = lines[len(lines)-9:]
	}
	for i, l := range lines {
		g.Text(dst, FontSmall, 8, 150+i*7, l, g.C(ColLight))
	}
	m.speedBtn.Label = "TEMPO " + itoa(indexOf(speeds, m.speed)+1)
	m.speedBtn.Draw(g, dst, app.Input.In(m.speedBtn.X, m.speedBtn.Y, m.speedBtn.W, m.speedBtn.H))
	if m.md.Done {
		if m.league() >= 0 {
			m.tableBtn.Draw(g, dst, app.Input.In(m.tableBtn.X, m.tableBtn.Y, m.tableBtn.W, m.tableBtn.H))
		}
		m.nextBtn.Draw(g, dst, app.Input.In(m.nextBtn.X, m.nextBtn.Y, m.nextBtn.W, m.nextBtn.H))
	} else if m.humanManagerPlaying() >= 0 {
		m.subBtn.Draw(g, dst, app.Input.In(m.subBtn.X, m.subBtn.Y, m.subBtn.W, m.subBtn.H))
	}
	if m.bannerT > 0 && len(m.banner) > 0 {
		lines := g.Wrap(FontNormal, m.banner[0], 240)
		h := len(lines)*10 + 10
		y := 80 - h/2
		FillRect(dst, 30, y, 260, h, g.C(ColDarkRed))
		FrameRect(dst, 30, y, 260, h, g.C(ColRed))
		for i, l := range lines {
			g.TextCentered(dst, FontNormal, 30, y+5+i*10, 260, l, g.C(ColBeige))
		}
	}
}

func indexOf(xs []int, v int) int {
	for i, x := range xs {
		if x == v {
			return i
		}
	}
	return -1
}
