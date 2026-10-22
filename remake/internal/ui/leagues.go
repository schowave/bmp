package ui

import (
	"github.com/hajimehoshi/ebiten/v2"

	"github.com/schowave/bmp/remake/internal/game"
)

// ListScreen is a screen with a title, one table and the HAUPTMENÜ button.
// Arrows (if Prev/Next are set) page through the content.
type ListScreen struct {
	Title     string
	Sub       string
	Table     *TableView
	Prev      func()
	Next      func()
	OnClick   func(app *App, row int)
	OnRight   func(app *App, row int)
	ExtraDraw func(app *App, dst *ebiten.Image)
}

var (
	arrowLeft  = Button{X: 214, Y: 196, W: ButtonW, H: ButtonH}
	arrowRight = Button{X: 160, Y: 196, W: ButtonW, H: ButtonH}
)

func (l *ListScreen) Update(app *App) error {
	in := &app.Input
	left, right := l.Table.Update(in)
	if left >= 0 && l.OnClick != nil {
		l.OnClick(app, left)
		return nil
	}
	if right >= 0 && l.OnRight != nil {
		l.OnRight(app, right)
		return nil
	}
	if l.Prev != nil && arrowRight.Hit(in) {
		l.Prev()
		return nil
	}
	if l.Next != nil && arrowLeft.Hit(in) {
		l.Next()
		return nil
	}
	if MenuButtonHit(in) {
		app.Pop()
	}
	return nil
}

func (l *ListScreen) Draw(app *App, dst *ebiten.Image) {
	g := app.Gfx
	g.Background(dst)
	g.Panel(dst, 4, 3, 312, 17)
	g.TextCentered(dst, FontNormal, 4, 7, 312, l.Title, g.C(ColBeige))
	if l.Sub != "" {
		g.TextRight(dst, FontSmall, 310, 8, l.Sub, g.C(ColLight))
	}
	l.Table.Draw(g, dst)
	if l.Prev != nil {
		g.DrawIconButton(dst, IconLeft, arrowRight.X, arrowRight.Y)
	}
	if l.Next != nil {
		g.DrawIconButton(dst, IconRight, arrowLeft.X, arrowLeft.Y)
	}
	if l.ExtraDraw != nil {
		l.ExtraDraw(app, dst)
	}
	g.DrawMenuButton(dst)
}

// NewTableScreen shows the table of league l (f_2a41_214c).
func NewTableScreen(ss *Session, l int) Scene {
	s := ss.S
	cols := []Column{
		{"PL.", 4, 12, Right}, {"VEREIN", 22, 110, Left}, {"BILANZ", 134, 40, Left},
		{"SP", 176, 14, Right}, {"S-U-N", 194, 40, Center}, {"PUNKTE", 236, 30, Center},
		{"TORE", 268, 34, Center},
	}
	t := NewTableView(4, 24, 312, 168, cols)
	t.RowH = 8
	for i := 0; i < game.TeamsPerLeague[l]; i++ {
		id := s.Tables[l][i]
		c := &s.Clubs[id]
		row := Row{Cells: []string{
			itoa(i+1) + ".", s.Teams[id].Name, formText(c.Form), itoa(c.TotalGames()),
			itoa(c.TotalWins()) + "-" + itoa(c.TotalDraws()) + "-" + itoa(c.TotalLosses()),
			itoa(c.TotalPoints()) + ":" + itoa(c.MinusPoints()),
			itoa(c.TotalGoals()) + ":" + itoa(c.TotalAgainst()),
		}}
		row.Bar = tableZone(l, i)
		if s.ManagerOf(id) != nil {
			row.Color = ColYellow
		}
		t.Rows = append(t.Rows, row)
	}
	md := s.Matchday[l] - 1
	return &ListScreen{Title: "Tabelle " + game.LeagueNames[l], Sub: itoa(md) + ". SPIELTAG", Table: t}
}

// formText turns the form string into the letters shown (s/u/n, upper
// case for away games).
func formText(f string) string {
	out := []byte{}
	for i := 0; i < len(f); i++ {
		if f[i] != '@' {
			out = append(out, f[i])
		}
	}
	return string(out)
}

// tableZone colours champion/UEFA/promotion and relegation places.
func tableZone(l, i int) int {
	n := game.TeamsPerLeague[l]
	switch {
	case l == 0 && i == 0:
		return ColGrey5
	case l == 0 && i <= game.UEFAPlaces:
		return ColGrey6
	case l > 0 && i < game.Relegated[l-1]:
		return ColGrey6
	case i >= n-game.Relegated[l]:
		return ColDarkRed
	}
	return 0
}

// NewResultsScreen shows the fixtures and results of league l, one
// matchday at a time (f_2a41_120a).
func NewResultsScreen(ss *Session, l int) Scene {
	s := ss.S
	md := max(1, s.Matchday[l]-1)
	if s.Matchday[l] > game.MatchdaysPerLeague[l] {
		md = game.MatchdaysPerLeague[l]
	}
	cols := []Column{{"", 4, 130, Right}, {"", 136, 10, Center}, {"", 148, 130, Left}, {"", 280, 28, Center}}
	t := NewTableView(4, 24, 312, 168, cols)
	t.RowH = 10
	t.NoHeader = true
	ls := &ListScreen{Table: t}
	fill := func() {
		t.Rows = nil
		ls.Title = itoa(md) + ". Spieltag " + game.LeagueNames[l]
		d := game.DateOf(s.StartYear, s.MatchdayDay(l, md))
		ls.Sub = itoa(d.Day) + "." + itoa(d.Month+1) + "."
		for i, p := range game.Fixtures(l, md) {
			r := s.Results[l][md-1][i]
			res := "-:-"
			if md < s.Matchday[l] {
				if r.Home == game.Postponed {
					res = "(N)"
				} else {
					res = itoa(r.Home) + ":" + itoa(r.Away)
				}
			}
			row := Row{Cells: []string{s.Teams[p.Home].Name, "-", s.Teams[p.Away].Name, res}}
			if s.ManagerOf(p.Home) != nil || s.ManagerOf(p.Away) != nil {
				row.Color = ColYellow
			}
			t.Rows = append(t.Rows, row)
		}
	}
	fill()
	ls.Prev = func() {
		if md > 1 {
			md--
			fill()
		}
	}
	ls.Next = func() {
		if md < game.MatchdaysPerLeague[l] {
			md++
			fill()
		}
	}
	return ls
}

// NewStrengthScreen shows the strengths of the clubs of league l in table
// order (f_2a41_3891).
func NewStrengthScreen(ss *Session, l int) Scene {
	s := ss.S
	cols := []Column{{"PL.", 4, 12, Right}, {"VEREIN", 22, 120, Left}, {"KONDITION", 150, 50, Center},
		{"TECHNIK", 204, 50, Center}, {"FORM", 258, 40, Center}}
	t := NewTableView(4, 24, 312, 168, cols)
	t.RowH = 8
	for i := 0; i < game.TeamsPerLeague[l]; i++ {
		id := s.Tables[l][i]
		tm := &s.Teams[id]
		row := Row{Cells: []string{itoa(i+1) + ".", tm.Name,
			itoa(avg3(tm.Strength)), itoa(avg3(tm.Base)), itoa(avg3(tm.Extra))}}
		if s.ManagerOf(id) != nil {
			row.Color = ColYellow
		}
		t.Rows = append(t.Rows, row)
	}
	return &ListScreen{Title: "Stärken " + game.LeagueNames[l], Table: t}
}

func avg3(v [3]int) int { return (v[0] + v[1] + v[2]) / 3 }
