package ui

import (
	"sort"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/schowave/bmp/remake/internal/game"
)

// slider is a value with - and + buttons.
type slider struct {
	label    string
	value    *int
	min, max int
	canUp    func() bool
}

// TrainingScreen sets training points, intensity and youth work
// (f_112a_274c).
type TrainingScreen struct {
	ss      *Session
	sliders []slider
}

// NewTrainingScreen creates the training screen.
func NewTrainingScreen(ss *Session) *TrainingScreen {
	m := ss.M()
	sum := func(v *[4]int) int { return v[0] + v[1] + v[2] + v[3] }
	t := &TrainingScreen{ss: ss}
	for i, n := range []string{"Kondition", "Spiel", "Schuss", "Taktik"} {
		t.sliders = append(t.sliders, slider{n, &m.Training[i], 0, 20, func() bool { return sum(&m.Training) < 20 }})
	}
	for i, n := range []string{"Tor", "Abwehr", "Mittelfeld", "Angriff"} {
		t.sliders = append(t.sliders, slider{n, &m.PosTraining[i], 0, 10, func() bool { return sum(&m.PosTraining) < 10 }})
	}
	t.sliders = append(t.sliders,
		slider{"Intensität", &m.Intensity, 1, 10, nil},
		slider{"Jugendarbeit", &m.Youth, 0, 34, nil})
	return t
}

func (t *TrainingScreen) rowY(i int) int {
	y := 26 + i*15
	if i >= 4 {
		y += 6
	}
	if i >= 8 {
		y += 6
	}
	return y
}

func (t *TrainingScreen) Update(app *App) error {
	in := &app.Input
	for i, s := range t.sliders {
		y := t.rowY(i)
		if in.Clicked(180, y, 14, 12) && *s.value > s.min {
			*s.value--
		}
		if in.Clicked(290, y, 14, 12) && *s.value < s.max && (s.canUp == nil || s.canUp()) {
			*s.value++
		}
	}
	if MenuButtonHit(in) {
		app.Pop()
	}
	return nil
}

func (t *TrainingScreen) Draw(app *App, dst *ebiten.Image) {
	g := app.Gfx
	s := t.ss.S
	g.Background(dst)
	g.Panel(dst, 10, 6, 300, 186)
	g.Panel(dst, 10, 196, 250, 14)
	g.Heading(dst, 16, 10, "TRAINING")
	g.Text(dst, FontSmall, 100, 12, "PUNKTE: "+itoa(20-sumOf(t.ss.M().Training))+" / "+itoa(10-sumOf(t.ss.M().PosTraining)), g.C(ColLight))
	for i, sl := range t.sliders {
		y := t.rowY(i)
		g.Text(dst, FontNormal, 20, y+2, sl.label, g.C(ColWhite))
		g.Text(dst, FontNormal, 182, y+2, "-", g.C(ColYellow))
		FillRect(dst, 196, y+2, 92, 8, g.C(ColBlack))
		FillRect(dst, 197, y+3, (*sl.value-sl.min)*90/(sl.max-sl.min), 6, g.C(ColRed))
		g.Text(dst, FontNormal, 292, y+2, "+", g.C(ColYellow))
		g.TextRight(dst, FontSmall, 176, y+3, itoa(*sl.value), g.C(ColLight))
	}
	g.Text(dst, FontSmall, 16, 200, "JUGENDARBEIT KOSTET "+Money(int64((t.ss.M().Youth+1)*625))+" PRO MONAT", g.C(ColLight))
	_ = s
	g.DrawMenuButton(dst)
}

func sumOf(v [4]int) int { return v[0] + v[1] + v[2] + v[3] }

// CampScreen books a training camp (f_112a_0145, PIC 5).
type CampScreen struct {
	ss    *Session
	hover int
}

// NewCampScreen creates the training camp screen.
func NewCampScreen(ss *Session) *CampScreen { return &CampScreen{ss: ss, hover: -1} }

// Camp picture cells in PIC/5.VGA: 2 columns of buildings, 4 rows.
func campRect(i int) (x, y, w, h int) {
	col, row := i/4, i%4
	return 30 + 47 + col*85, 20 + row*39, 82, 37
}

func (c *CampScreen) Update(app *App) error {
	in := &app.Input
	c.hover = -1
	for i := 0; i < 8; i++ {
		x, y, w, h := campRect(i)
		if in.In(x, y, w, h) {
			c.hover = i
			if in.Left {
				mi := c.ss.Mgr()
				if err := c.ss.S.BookCamp(mi, i); err != nil {
					msg := err.Error()
					if err == game.ErrCampClosed {
						msg = game.CampNames[i] + " " + msg
					}
					app.Push(NewMessage(c.ss.M().Name+":", msg))
				} else {
					app.Pop()
					app.Push(NewMessage(c.ss.M().Name+":", "Ihre Mannschaft fährt ins Trainingslager "+game.CampNames[i]+"."))
				}
				return nil
			}
		}
	}
	if MenuButtonHit(in) {
		app.Pop()
	}
	return nil
}

func (c *CampScreen) Draw(app *App, dst *ebiten.Image) {
	g := app.Gfx
	s := c.ss.S
	g.Background(dst)
	Draw(dst, g.Pic(5, false), 30, 20)
	for i := 0; i < 8; i++ {
		x, y, w, h := campRect(i)
		if !s.CampOpen(i) {
			FillRect(dst, x+30, y+14, 22, 9, g.C(ColBlack))
			g.TextCentered(dst, FontSmall, x, y+15, w, "ZU", g.C(ColRed))
		}
		if i == c.hover {
			FrameRect(dst, x, y, w, h, g.C(ColYellow))
		}
	}
	g.Panel(dst, 4, 182, 260, 54)
	if c.hover >= 0 {
		i := c.hover
		g.Text(dst, FontNormal, 8, 185, game.CampNames[i]+": "+Money(s.CampPrice(c.ss.Mgr(), i))+"/Woche", g.C(ColWhite))
		for k, n := range game.CampRatingNames {
			g.Text(dst, FontSmall, 8+(k%3)*85, 197+(k/3)*9, n+" "+itoa(game.CampRatings[i][k]), g.C(ColLight))
		}
		if !s.CampOpen(i) {
			g.Text(dst, FontSmall, 8, 216, game.CampNames[i]+" hat nicht geöffnet...", g.C(ColRed))
		}
	} else {
		g.Text(dst, FontSmall, 8, 186, "Wählen Sie ein Trainingslager.", g.C(ColLight))
		g.Text(dst, FontSmall, 8, 196, "Kontostand: "+Money(c.ss.M().Money), g.C(ColLight))
	}
	g.DrawMenuButton(dst)
}

// ContractScreen negotiates contracts (f_2277_2a8f). At the season end it
// handles the expiring contracts and continues the game when closed.
type ContractScreen struct {
	ss        *Session
	seasonEnd bool
	table     *TableView
	rows      []int
	done      Button
}

// NewContractScreen creates the contract screen.
func NewContractScreen(ss *Session, seasonEnd bool) *ContractScreen {
	cols := []Column{{"NAME", 2, 80, Left}, {"ALTER", 84, 24, Right}, {"STÄRKE", 110, 30, Right},
		{"LIGASP.", 142, 30, Right}, {"VERTRAG", 174, 34, Right}, {"GEHALT", 210, 54, Right}, {"STATUS", 266, 46, Left}}
	c := &ContractScreen{ss: ss, seasonEnd: seasonEnd, table: NewTableView(2, 24, 316, 168, cols)}
	c.done = Button{X: 200, Y: 210, W: 60, H: 14, Label: "FERTIG"}
	return c
}

func (c *ContractScreen) refresh() {
	s := c.ss.S
	mi := c.ss.Mgr()
	c.table.Rows = c.table.Rows[:0]
	c.rows = c.rows[:0]
	if c.seasonEnd {
		c.rows = append(c.rows, s.ExpiringContracts()...)
	} else {
		for i := 0; i < s.SquadSize(mi); i++ {
			c.rows = append(c.rows, i)
		}
	}
	for _, i := range c.rows {
		sl := &s.Squads[mi][i]
		p := &s.Persons[sl.Person]
		st := ""
		switch {
		case sl.Loan != 0:
			st = "LEIHE"
		case sl.Nego&0x80 != 0:
			st = "HÖRT AUF"
		case sl.Nego >= 100:
			st = "WILL " + itoa(sl.Nego-100) + " J."
		case sl.Nego > 0:
			st = "WARTEN"
		}
		c.table.Rows = append(c.table.Rows, Row{Cells: []string{p.Name, itoa(p.Age), itoa(sl.Strength()),
			itoa(sl.Games[0]), itoa(sl.Years) + " J.", formatMoney(sl.Salary), st}})
	}
}

func (c *ContractScreen) Update(app *App) error {
	s := c.ss.S
	mi := c.ss.Mgr()
	in := &app.Input
	c.refresh()
	left, right := c.table.Update(in)
	if right >= 0 {
		app.Push(NewPlayerInfo(c.ss, mi, c.rows[right]))
		return nil
	}
	if left >= 0 {
		c.negotiate(app, c.rows[left])
		return nil
	}
	if c.seasonEnd {
		if c.done.Hit(in) || in.Escape {
			s.ContractsDone()
			app.Pop()
		}
		return nil
	}
	if MenuButtonHit(in) {
		app.Pop()
	}
	return nil
}

// negotiate runs the dialog for one player: years, then salary.
func (c *ContractScreen) negotiate(app *App, si int) {
	s := c.ss.S
	mi := c.ss.Mgr()
	name := c.ss.M().Name + ":"
	forced := 0
	if !c.seasonEnd {
		f, err := s.NegotiateCheck(mi, si)
		if err != nil {
			app.Push(NewMessage(name, err.Error()))
			return
		}
		forced = f
	}
	sl := &s.Squads[mi][si]
	pname := s.Persons[sl.Person].Name
	years := int64(max(1, sl.Years))
	if forced != 0 {
		years = int64(forced)
	}
	app.Push(NewNumberInput(pname, "VERTRAGSDAUER (IN SAISONS)", years, func(app *App, y int64, ok bool) {
		if !ok {
			return
		}
		hint := s.ContractHint(mi, si, int(y))
		app.Push(NewNumberInput(pname, "GEHALT PRO MONAT (IN DM), WUNSCH ca. "+formatMoney(hint), hint,
			func(app *App, sal int64, ok bool) {
				if !ok {
					return
				}
				if err := s.NegotiateContract(mi, si, int(y), sal, forced); err != nil {
					app.Push(NewMessage(name, err.Error()))
					return
				}
				app.Push(NewMessage(name, "Ihr Angebot wurde angenommen !"))
			}))
	}))
}

func (c *ContractScreen) Draw(app *App, dst *ebiten.Image) {
	g := app.Gfx
	g.Background(dst)
	g.Panel(dst, 4, 3, 312, 17)
	title := "VERTRAGSVERHANDLUNGEN"
	if c.seasonEnd {
		title = "AUSLAUFENDE VERTRÄGE - " + c.ss.M().Name
	}
	g.TextCentered(dst, FontNormal, 4, 7, 312, title, g.C(ColBeige))
	c.refresh()
	c.table.Draw(g, dst)
	if c.seasonEnd {
		g.Text(dst, FontSmall, 4, 196, "Spieler ohne neuen Vertrag verlassen den Verein (Ablöse: halber Marktwert).", g.C(ColLight))
		c.done.Draw(g, dst, app.Input.In(c.done.X, c.done.Y, c.done.W, c.done.H))
		return
	}
	g.Text(dst, FontSmall, 4, 196, "LINKS: VERHANDELN   RECHTS: INFO", g.C(ColLight))
	g.DrawMenuButton(dst)
}

// ScorerScreen shows the best scorers of the manager's league
// (f_14a4_1ad5, "Die Besten der Liga").
func NewScorerScreen(ss *Session) Scene {
	s := ss.S
	l := ss.M().League
	cols := []Column{{"PL.", 4, 14, Right}, {"SPIELER", 24, 100, Left}, {"VEREIN", 130, 110, Left},
		{"TORE", 244, 26, Right}, {"T/SP", 276, 30, Right}}
	t := NewTableView(4, 24, 312, 168, cols)
	t.RowH = 8
	var ps []int
	for p := 1; p < game.NumPersons; p++ {
		pp := &s.Persons[p]
		if pp.Team >= game.NumGerman || pp.Team == 0xff || game.LeagueOf(pp.Team) != l || pp.LeagueGoals == 0 {
			continue
		}
		ps = append(ps, p)
	}
	sort.SliceStable(ps, func(a, b int) bool { return s.Persons[ps[a]].LeagueGoals > s.Persons[ps[b]].LeagueGoals })
	for i, p := range ps {
		if i >= 20 {
			break
		}
		pp := &s.Persons[p]
		ratio := "-"
		if pp.LeagueGames > 0 {
			r := pp.LeagueGoals * 100 / pp.LeagueGames
			ratio = itoa(r/100) + "." + pad2(r%100)
		}
		row := Row{Cells: []string{itoa(i+1) + ".", pp.Name, s.Teams[pp.Team].Name, itoa(pp.LeagueGoals), ratio}}
		if s.ManagerOf(pp.Team) != nil {
			row.Color = ColYellow
		}
		t.Rows = append(t.Rows, row)
	}
	if len(t.Rows) == 0 {
		t.Rows = append(t.Rows, Row{Cells: []string{"", "NOCH KEINE EINTRÄGE VORHANDEN..."}})
	}
	return &ListScreen{Title: "Die Besten der " + game.LeagueNames[l], Table: t}
}

func pad2(n int) string {
	if n < 10 {
		return "0" + itoa(n)
	}
	return itoa(n)
}
