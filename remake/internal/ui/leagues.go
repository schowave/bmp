package ui

import (
	"strings"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/schowave/bmp/remake/internal/game"
)

// The league screens of the main menu (Tabelle, Spiele, Stärken) and the
// same table and results layouts used during a match day.

// leagueIconW and leagueIconH are the size of an icon button as a click
// region.
const (
	leagueIconW = ButtonW
	leagueIconH = ButtonH
)

// lgFigSpace is the blank f_076b_0681 pads numbers with: as wide as a
// digit, wider than a space.
const lgFigSpace = '\u2007'

// lgPad formats n right-aligned in w digits, like f_076b_0681 with the width
// D:0x79c.
func lgPad(n, w int) string {
	s := itoa(n)
	if len(s) < w {
		s = strings.Repeat(string(lgFigSpace), w-len(s)) + s
	}
	return s
}

// lgTextFig draws s like TextAt; a lgFigSpace leaves the width of a digit
// blank.
func (g *Gfx) lgTextFig(dst *ebiten.Image, font Font, x, y int, s string, c int) {
	for i, part := range strings.Split(s, string(lgFigSpace)) {
		if i > 0 {
			x += g.TextWidth(font, "0")
		}
		g.TextAt(dst, font, x, y, part, c, NoShadow)
		x += g.TextWidth(font, part)
	}
}

// lgInside reports a left click strictly inside the region x, y, w, h
// (f_06c7_0002).
func lgInside(in *Input, x, y, w, h int) bool {
	return in.Left && in.X > x && in.X < x+w && in.Y > y && in.Y < y+h
}

// lgIconHit reports a left click on the icon button at x, y.
func lgIconHit(in *Input, x, y int) bool { return in.Clicked(x, y, leagueIconW, leagueIconH) }

// lgColor is the colour of the rows of clubs: 11 when a human manager runs
// one of them, else 1.
func lgColor(s *game.State, ids ...int) int {
	for _, id := range ids {
		if s.ManagerOf(id) != nil {
			return ColBeige
		}
	}
	return ColLight
}

// lgLeave closes a league screen: from the menu it goes back, during
// the match day it calls done.
func lgLeave(app *App, done func(app *App)) {
	app.Pop()
	if done != nil {
		done(app)
	}
}

// ---------------------------------------------------------------------
// Tabelle (f_2a41_214c)

// TableScreen is the league table with the buttons HEIM, GESAMT and
// AUSWÄRTS.
type TableScreen struct {
	ss     *Session
	league int
	mode   int // game.TableHome, TableTotal or TableAway
	round  bool
	done   func(app *App)
}

// Geometry of f_2a41_214c.
const (
	tblX0, tblY0, tblX1, tblY1 = 5, 8, 5 + 0x134, 8 + 0xb4
	tblBtnY                    = 0xae
	tblRowY                    = 0x21 // baseline of the first row
	tblLineTop, tblLineBottom  = 0x19, 0xa9
)

var (
	tblModeNames = [3]string{"HEIM", "GESAMT", "AUSWÄRTS"} // D:0x4b94
	// tblInfoMode is D:0x5538: the mode of the club info for each table.
	tblInfoMode = [3]int{0, 2, 1}
)

const tblHeader = "PL.     VEREIN                           BILANZ         SP       S-U-N      PUNKTE     TORE      TORDIFF."

// NewTableScreen shows the table of league l as opened from the main menu,
// with the HAUPT MENU button.
func NewTableScreen(ss *Session, l int) Scene {
	return &TableScreen{ss: ss, league: l, mode: game.TableTotal}
}

// NewTableRound shows the table of league l after the matches of a match
// day, with WEITER instead of HAUPT MENU. Like the original it shows the
// matchday s.Matchday[l], so it has to be shown before the matchday
// counters advance (game.State.afterEvent). WEITER pops the screen and
// calls done.
func NewTableRound(ss *Session, l int, done func(app *App)) Scene {
	return &TableScreen{ss: ss, league: l, mode: game.TableTotal, round: true, done: done}
}

func (t *TableScreen) rows() []game.TableRow { return t.ss.S.LeagueTable(t.league, t.mode) }

// matchday is the number in the title: from the menu the last matchday
// played.
func (t *TableScreen) matchday() int {
	md := t.ss.S.Matchday[t.league]
	if !t.round && md > 1 {
		md--
	}
	return md
}

func (t *TableScreen) Update(app *App) error {
	in := &app.Input
	if lgIconHit(in, MenuButton.X, MenuButton.Y) || in.Escape {
		lgLeave(app, t.done)
		return nil
	}
	if !lgInside(in, tblX0, tblY0, tblX1-tblX0, tblY1-tblY0) {
		return nil
	}
	switch {
	case in.Y > tblLineBottom:
		switch {
		case in.X < 0x87:
			t.mode = game.TableHome
		case in.X < 0xd7:
			t.mode = game.TableTotal
		default:
			t.mode = game.TableAway
		}
	case in.Y < tblLineBottom && in.Y > 0x1a:
		r := (in.Y - 0x1a) / 7
		if rows := t.rows(); r < len(rows) {
			app.Push(NewTeamInfoMode(t.ss, rows[r].Club, tblInfoMode[t.mode]))
		}
	}
	return nil
}

func (t *TableScreen) Draw(app *App, dst *ebiten.Image) {
	g := app.Gfx
	s := t.ss.S
	l := t.league
	g.Background(dst)
	g.TextMid(dst, FontNormal, 0, 0x10c, 0xdc, "Tabelle "+s.LeagueName(l), ColBeige, ColBlack)
	g.Window(dst, tblX0, tblY0, tblX1, tblY1, ColBlue)
	icon := IconMainMenu
	if t.round {
		icon = IconWeiter
	}
	g.DrawIconButton(dst, icon, MenuButton.X, MenuButton.Y)
	FillRect(dst, 7, tblLineTop, 0x137-7+1, 1, g.PalColor(ColLight))
	FillRect(dst, 7, tblLineBottom, 0x137-7+1, 1, g.PalColor(ColLight))
	g.TextAt(dst, FontSmall, 9, 0x17, tblHeader, ColLight, NoShadow)
	for i, name := range tblModeNames {
		c := ColLight
		if i == t.mode {
			c = ColRed
		}
		x := 0x1e + i*0x66
		g.BevelButton(dst, x, tblBtnY, 0x39, 0xc, ColBlack)
		g.TextMid(dst, FontSmall, x+1, x+0x39, tblBtnY+8, name, c, NoShadow)
	}
	title := "Tabelle " + tblModeNames[t.mode] + "   " + itoa(t.matchday()) + ".SPIELTAG"
	g.TextMid(dst, FontNormal, 5, 0x139, 0x10, title, ColLight, ColGrey7)

	// Places: champion and UEFA cup, promotion, relegation.
	n := game.TeamsPerLeague[l]
	box := func(y0, y1, c int) { FillRect(dst, 9, y0, 0x13-9+1, y1-y0+1, g.PalColor(c)) }
	if l == 0 {
		box(0x1c, 0x22, 15)
		box(0x23, game.UEFAPlaces*7+0x22, 12)
	} else {
		box(0x1c, game.Relegated[l-1]*7+0x1b, 12)
	}
	box((n-game.Relegated[l])*7+0x1c, n*7+0x1b, ColRedDark)

	for i, r := range t.rows() {
		y := i*7 + tblRowY
		c := lgColor(s, r.Club)
		g.lgTextFig(dst, FontSmall, 9, y, lgPad(i+1, 2)+".", ColWhite)
		g.TextAt(dst, FontSmall, 0x15, y, s.Teams[r.Club].Name, c, NoShadow)
		g.lgTextFig(dst, FontSmall, 0x67, y, lgTableLine(r, s.PointsPerWin()), c)
	}
}

// lgTableLine is the text of a table row from BILANZ on (f_2a41_214c).
// win is the points per win: the 3-point rule shows the points alone.
func lgTableLine(r game.TableRow, win int) string {
	var b strings.Builder
	b.WriteString(r.Form)
	b.WriteString("   ")
	b.WriteString(lgPad(r.Games, 2) + "     ")
	b.WriteString(lgPad(r.Wins, 2) + " " + lgPad(r.Draws(), 2) + " " + lgPad(r.Losses, 2) + "    ")
	if win > 2 {
		b.WriteString("     " + lgPad(r.Points, 2) + " ")
	} else {
		b.WriteString(lgPad(r.Points, 2) + ":" + lgPad(r.MinusPoints(), 2))
	}
	b.WriteString("    ")
	b.WriteString(lgPad(r.Goals, 3) + ":" + lgPad(r.Against, 3) + "      ")
	d := r.Goals - r.Against
	switch {
	case d == 0:
		b.WriteString("#") // the ± glyph
	case d > 0:
		b.WriteString("+")
	default:
		b.WriteString("-")
		d = -d
	}
	b.WriteString(lgPad(d, 2))
	return b.String()
}

// ---------------------------------------------------------------------
// Spiele (f_2a41_120a)

// ResultsScreen shows the matches of one matchday of a league: from the
// menu with arrows to page through the matchdays, during the match day
// with the minute and WEITER.
type ResultsScreen struct {
	ss     *Session
	league int
	md     int
	live   bool
	minute int
	done   func(app *App)
}

// Geometry of f_2a41_120a.
const (
	resX0, resY0, resX1, resY1 = 1, 2, 1 + 0x13a, 2 + 0xb9
	resRowH                    = 0x11
	resPrevX, resNextX         = 1, 0x3b
	resIconY                   = 0xc1
)

// NewResultsScreen shows the next matchday of league l (the one to be
// played; the first one after the season) as opened from the main menu.
func NewResultsScreen(ss *Session, l int) Scene {
	md := ss.S.Matchday[l]
	if md > game.MatchdaysPerLeague[l] || md < 1 {
		md = 1
	}
	return &ResultsScreen{ss: ss, league: l, md: md}
}

// NewResultsRound shows the scores of the current matchday of league l
// during the match day at the given minute ("SPIELE Bundesliga
// 1.SPIELTAG 45.MINUTE"), with WEITER instead of HAUPT MENU and without
// the arrows. The scores are read from s.Results of matchday
// s.Matchday[l]. WEITER pops the screen and calls done.
func NewResultsRound(ss *Session, l, minute int, done func(app *App)) Scene {
	return &ResultsScreen{ss: ss, league: l, md: ss.S.Matchday[l], live: true, minute: minute, done: done}
}

func (r *ResultsScreen) Update(app *App) error {
	in := &app.Input
	n := game.MatchdaysPerLeague[r.league]
	switch {
	case lgIconHit(in, MenuButton.X, MenuButton.Y) || in.Escape:
		lgLeave(app, r.done)
	case !r.live && lgIconHit(in, resNextX, resIconY):
		if r.md++; r.md > n {
			r.md = 1
		}
	case !r.live && lgIconHit(in, resPrevX, resIconY):
		if r.md--; r.md < 1 {
			r.md = n
		}
	case lgInside(in, resX0, resY0, resX1-resX0, resY1-resY0) && in.Y > 0xe:
		k := (in.Y-0xe)/resRowH*2 + lgB2i(in.X > 0x8c)
		pairs := game.Fixtures(r.league, r.md)
		if k < 2*len(pairs) {
			p := pairs[k/2]
			team := p.Home
			if k%2 == 1 {
				team = p.Away
			}
			app.Push(NewTeamInfoMode(r.ss, team, k%2))
		}
	}
	return nil
}

func lgB2i(b bool) int {
	if b {
		return 1
	}
	return 0
}

// title is the headline of the screen.
func (r *ResultsScreen) title() string {
	s := r.ss.S
	t := "SPIELE  " + s.LeagueName(r.league) + "   " + itoa(r.md) + ".SPIELTAG  "
	if r.live {
		return t + "  " + itoa(r.minute) + ".MINUTE"
	}
	d := game.DateOf(s.StartYear, s.MatchdayDay(r.league, r.md))
	return t + itoa(d.Day) + "." + itoa(d.Month+1) + "."
}

func (r *ResultsScreen) Draw(app *App, dst *ebiten.Image) {
	g := app.Gfx
	s := r.ss.S
	l := r.league
	g.Background(dst)
	g.Window(dst, resX0, resY0, resX1, resY1, ColBlue)
	icon := IconMainMenu
	if r.live {
		icon = IconWeiter
	}
	g.DrawIconButton(dst, icon, MenuButton.X, MenuButton.Y)
	if !r.live {
		g.DrawIconButton(dst, IconLeft, resPrevX, resIconY)
		g.DrawIconButton(dst, IconRight, resNextX, resIconY)
	}
	FillRect(dst, 3, 0xe, 0x139-3+1, 1, g.PalColor(ColBeige))
	g.TextMid(dst, FontNormal, 7, 0x139, 0xb, r.title(), ColLight, ColGrey7)

	played := r.md < s.Matchday[l]+lgB2i(r.live)
	for i, p := range game.Fixtures(l, r.md) {
		y := i * resRowH
		c := lgColor(s, p.Home, p.Away)
		g.TextAt(dst, FontNormal, 3, y+0x18, s.Teams[p.Home].Name, c, NoShadow)
		g.TextAt(dst, FontNormal, 0x88, y+0x18, "-", c, NoShadow)
		g.TextAt(dst, FontNormal, 0x92, y+0x18, s.Teams[p.Away].Name, c, NoShadow)
		g.lgTextFig(dst, FontSmall, 5, y+0x1f, lgStrengthLine(s, p.Home), ColGrey4)
		g.lgTextFig(dst, FontSmall, 0x94, y+0x1f, lgStrengthLine(s, p.Away), ColGrey4)
		res := s.Results[l][r.md-1][i]
		switch {
		case res.Home == game.Postponed:
			if txt := lgPostponedDate(s, l, r.md, i); txt != "" {
				g.TextMid(dst, FontSmall, 0x128, 0x128, y+0x17, txt, c, NoShadow)
			}
		case played:
			g.TextAt(dst, FontNormal, 0x120, y+0x1a, itoa(res.Home), c, NoShadow)
			g.TextAt(dst, FontNormal, 0x129, y+0x1a, ":", c, NoShadow)
			g.TextAt(dst, FontNormal, 0x12e, y+0x1a, itoa(res.Away), c, NoShadow)
		default:
			g.TextAt(dst, FontNormal, 0x120, y+0x1a, "-:-", c, NoShadow)
		}
	}
}

// lgStrengthLine is "8. PLATZ, STÄRKE 75 (84,93,48)" (f_2a41_1d18): the
// table place and the strength of a club, the mean of its current, base
// and extra values, then the three means.
func lgStrengthLine(s *game.State, id int) string {
	t := &s.Teams[id]
	sum := func(v [3]int) int { return v[0] + v[1] + v[2] }
	cur, base, extra := sum(t.Strength), sum(t.Base), sum(t.Extra)
	return lgPad(lgTablePos(s, id)+1, 2) + ". PLATZ, STÄRKE " + lgPad((cur+base+extra)/9, 2) +
		" (" + lgPad(cur/3, 2) + "," + lgPad(base/3, 2) + "," + lgPad(extra/3, 2) + ")"
}

// lgTablePos returns the place of club id in its league table, 0-based
// (F:0x0efa, which the original keeps equal to the index in the table).
func lgTablePos(s *game.State, id int) int {
	l := game.LeagueOf(id)
	for i := 0; i < game.TeamsPerLeague[l]; i++ {
		if s.Tables[l][i] == id {
			return i
		}
	}
	return s.Clubs[id].Pos
}

// lgPostponedDate is the new date of a postponed game, "12.9. (N)", or ""
// when it has none.
func lgPostponedDate(s *game.State, l, md, pair int) string {
	for _, p := range s.Postponed {
		if p.Matchday == md && p.League == l && p.Pair == pair && p.Slot > 0 {
			d := game.DateOf(s.StartYear, game.EventDay(p.Slot))
			return itoa(d.Day) + "." + itoa(d.Month+1) + ". (N)"
		}
	}
	return ""
}

// ---------------------------------------------------------------------
// Stärken (f_2a41_3891)

// StrengthScreen lists the clubs of a league by strength. The button
// cycles through GESAMT and the three parts of the team.
type StrengthScreen struct {
	ss     *Session
	league int
	part   int // 0 GESAMT, 1 ABWEHR, 2 MITTELFELD, 3 STURM
}

// Geometry of f_2a41_3891.
const (
	strX0, strY0, strX1, strY1 = 0x3d, 0x11, 0x3d + 0xbb, 0x11 + 0xb2
)

var strPartNames = [4]string{"GESAMT", "ABWEHR", "MITTELFELD", "STURM"} // D:0x4bb0

const strHeader = "PL.     VEREIN               KONDITION   TECHNIK    FORM"

// NewStrengthScreen shows the strengths of the clubs of league l.
func NewStrengthScreen(ss *Session, l int) Scene {
	return &StrengthScreen{ss: ss, league: l}
}

// parts returns the strength values the current button selects.
func (st *StrengthScreen) parts() (from, to int) {
	if st.part == 0 {
		return 0, 2
	}
	return st.part - 1, st.part - 1
}

// order returns the club ids sorted by the selected strength, strongest
// first: an exchange sort over the clubs in id order.
func (st *StrengthScreen) order() []int {
	s := st.ss.S
	n := game.TeamsPerLeague[st.league]
	from, to := st.parts()
	ids := make([]int, n)
	val := make(map[int]int, n)
	for i := range ids {
		id := game.FirstID[st.league] + i
		ids[i] = id
		t := &s.Teams[id]
		for k := from; k <= to; k++ {
			val[id] += t.Base[k] + t.Strength[k] + t.Extra[k]
		}
	}
	for i := 0; i < n-1; i++ {
		for j := i + 1; j < n; j++ {
			if val[ids[i]] < val[ids[j]] {
				ids[i], ids[j] = ids[j], ids[i]
			}
		}
	}
	return ids
}

func (st *StrengthScreen) Update(app *App) error {
	in := &app.Input
	if lgIconHit(in, MenuButton.X, MenuButton.Y) || in.Escape {
		app.Pop()
		return nil
	}
	if !lgInside(in, strX0, strY0, strX1-strX0, strY1-strY0) {
		return nil
	}
	switch {
	case in.Y > 0xb1:
		st.part = (st.part + 1) % len(strPartNames)
	case in.Y > 0x25:
		r := (in.Y - 0x25) / 7
		if ids := st.order(); r < len(ids) {
			app.Push(NewTeamInfoMode(st.ss, ids[r], 2))
		}
	}
	return nil
}

func (st *StrengthScreen) Draw(app *App, dst *ebiten.Image) {
	g := app.Gfx
	s := st.ss.S
	g.Background(dst)
	g.Window(dst, strX0, strY0, strX1, strY1, ColBlue)
	g.DrawIconButton(dst, IconMainMenu, MenuButton.X, MenuButton.Y)
	FillRect(dst, 0x3f, 0x23, 0xf6-0x3f+1, 1, g.PalColor(ColLight))
	FillRect(dst, 0x3f, 0xb2, 0xf6-0x3f+1, 1, g.PalColor(ColLight))
	g.TextMid(dst, FontNormal, 0x3d, 0xf8, 0x19, "STÄRKEN "+s.LeagueName(st.league), ColLight, ColGrey7)
	g.TextAt(dst, FontSmall, 0x41, 0x21, strHeader, ColLight, NoShadow)
	g.BevelButton(dst, 0x85, 0xb5, 0x39, 0xc, ColBlack)
	g.TextMid(dst, FontSmall, 0x86, 0x85+0x39, 0xb5+8, strPartNames[st.part], ColRed, NoShadow)

	from, to := st.parts()
	cnt := to - from + 1
	mean := func(v [3]int) int {
		sum := 0
		for k := from; k <= to; k++ {
			sum += v[k]
		}
		return min(max(sum/cnt, 0), 99)
	}
	for i, id := range st.order() {
		y := (i + 6) * 7
		c := lgColor(s, id)
		t := &s.Teams[id]
		g.lgTextFig(dst, FontSmall, 0x41, y, lgPad(i+1, 2)+". "+t.Name, c)
		line := lgPad(mean(t.Strength), 2) + strings.Repeat(" ", 12) + lgPad(mean(t.Base), 2) +
			strings.Repeat(" ", 12) + lgPad(mean(t.Extra), 2)
		g.lgTextFig(dst, FontSmall, 0xa2, y, line, c)
	}
}
