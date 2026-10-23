package ui

import (
	"image"
	"strings"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/schowave/bmp/remake/internal/game"
)

// TrainingScreen sets the training (f_112a_274c): rows of footballs for
// Kondition, Spiel, Schuss, Taktik (20 points) and Tor, Abwehr,
// Mittelfeld, Angriff (10 points), the intensity, the points left in the
// bars at the sides, the K-T-E meter of the team and the JUGEND meter.
type TrainingScreen struct {
	ss *Session
}

// NewTrainingScreen creates the training screen.
func NewTrainingScreen(ss *Session) *TrainingScreen { return &TrainingScreen{ss: ss} }

var (
	tsTrainLeft  = [4]string{"Kondition", "Spiel", "Schuss", "Taktik"}
	tsTrainRight = [4]string{"Tor", "Abwehr", "Mittelfeld", "Angriff"}
)

func tsSum(v [4]int) int { return v[0] + v[1] + v[2] + v[3] }

func (t *TrainingScreen) Update(app *App) error {
	in := &app.Input
	m := t.ss.M()
	if MenuButtonHit(in) {
		app.Pop()
		return nil
	}
	if !in.Left {
		return nil
	}
	x, y := in.X, in.Y
	// The JUGEND meter.
	if x > 0xd0 && x < 0x105 && y > 0xc0 && y < 0xe9 {
		m.Youth = min(max(x-0xd9, 0), 0x22)
		return nil
	}
	for i := 0; i < 4; i++ {
		if !((y > i*0x25+0x21 && y < i*0x25+0x33) || (y > 0xbc && y < 0xce && x < 0xbe)) {
			continue
		}
		row := (y - 0x21) / 0x25
		if x >= 0x1f && x <= 0xbd {
			n := (x-0x1e)/16 + 1
			if row > 3 {
				m.Intensity = n
			} else if m.Training[row]-n+20-tsSum(m.Training) >= 0 {
				m.Training[row] = n
			}
		} else if x > 0xd1 && x < 0x121 && row < 4 {
			n := (x-0xd1)/16 + 1
			if m.PosTraining[row]-n+10-tsSum(m.PosTraining) >= 0 {
				m.PosTraining[row] = n
			}
		}
		break
	}
	return nil
}

func (t *TrainingScreen) Draw(app *App, dst *ebiten.Image) {
	g := app.Gfx
	m := t.ss.M()
	g.Background(dst)
	g.DrawMenuButton(dst)
	pic := func(r image.Rectangle, x, y int) { Draw(dst, g.Sub(1, r), x, y) }
	bar := image.Rect(0, 0, 0x15, 0xa5)
	pic(bar, 3, 0xe)
	pic(bar, 0x12a, 0xe)
	long, cont := image.Rect(0x15, 0, 0x15+0x54, 0x15), image.Rect(0x17, 0, 0x17+0x52, 0x15)
	for i := 0; i < 4; i++ {
		y := i*0x25 + 0x22
		pic(long, 0x1e, y)
		pic(cont, 0x70, y)
		g.TextMid(dst, FontNormal, 0x1d, 0xbf, i*0x25+0x3e, tsTrainLeft[i], ColBeige, ColBlack)
		pic(long, 0xd1, y)
		g.TextMid(dst, FontNormal, 0xd0, 0x122, i*0x25+0x3e, tsTrainRight[i], ColBeige, ColBlack)
	}
	pic(long, 0x1e, 0xbc)
	pic(cont, 0x70, 0xbc)
	g.TextMid(dst, FontNormal, 0x1d, 0xc1, 0xd7, "Intensität", ColBeige, ColBlack)

	ball := g.SubKeyed(1, 0, image.Rect(0x15, 0x15, 0x15+15, 0x15+15))
	row := func(x0, y, n int) {
		for k := 0; k < n; k++ {
			Draw(dst, ball, x0+k*16, y+1)
		}
	}
	for i := 0; i < 4; i++ {
		row(0x1f, i*0x25+0x23, m.Training[i])
		row(0xd2, i*0x25+0x23, m.PosTraining[i])
	}
	row(0x1f, 0xbd, m.Intensity)
	// The points left, stacked from the bottom of the side bars.
	for k := 1; k <= 20-tsSum(m.Training); k++ {
		Draw(dst, ball, 5, 0xb1-16*k)
	}
	for k := 1; k <= 10-tsSum(m.PosTraining); k++ {
		Draw(dst, ball, 0x12c, 0xb1-16*k)
	}
	t.drawKTE(g, dst)
	// JUGEND: the length of the line is the youth work, below its cost.
	Draw(dst, g.Sub(0x25, image.Rect(0, 0x36, 0x32, 0x36+0x29)), 0xd1, 0xc1)
	FillRect(dst, 0xd9, 0xd4, m.Youth+1, 2, g.PalColor(ColWhite))
	g.TextMid(dst, FontSmall, 0xd3, 0x105, 0xdf, itoa64(int64(m.Youth+1)*10000/16), ColWhite, NoShadow)
}

// drawKTE draws the meter of the team (f_0f9d_1122): Kondition and Technik
// of the players in the team and their Erschöpfung.
func (t *TrainingScreen) drawKTE(g *Gfx, dst *ebiten.Image) {
	s := t.ss.S
	mi := t.ss.Mgr()
	var k, te, e, n int
	for i := 0; i < s.SquadSize(mi); i++ {
		sl := &s.Squads[mi][i]
		if sl.Lineup != 0 && sl.Lineup < 12 {
			n++
			k += sl.Skill[0]
			te += sl.Skill[1]
			e += sl.Fatigue
		}
	}
	n = max(n, 1)
	FillRect(dst, 3, 0xb7, 0x16-3+1, 0xe8-0xb7+1, g.PalColor(ColGrey7))
	FrameRect(dst, 3, 0xb7, 0x16-3+1, 0xe8-0xb7+1, g.PalColor(ColLight))
	vals := [3]int{k / n, te / n, e/n - 0x32}
	for i, l := range [3]string{"K", "T", "E"} {
		g.TextAt(dst, FontSmall, i*6+5, 0xe6, l, i+13, NoShadow)
		top := min(0x26-vals[i]*0x26/100, 0x26) + 0xb9
		x := i*6 + 5
		FillRect(dst, x, top, 4, 0xdf-top+1, g.PalColor(i+13))
		FrameRect(dst, x, top, 4, 0xdf-top+1, g.PalColor(ColBlack))
	}
}

// CampScreen books a training camp (f_112a_0145): the eight camps of
// PIC/5.VGA with their ratings, a cobweb over the closed ones, and the
// text under the pointer at the bottom.
type CampScreen struct {
	ss     *Session
	booked int // frames left of the picture shown after booking, -1 = none
}

// NewCampScreen creates the training camp screen.
func NewCampScreen(ss *Session) *CampScreen { return &CampScreen{ss: ss, booked: -1} }

var tsRatingColor = [5]int{24, 1, 15, 19, 18} // D:0x5278

// hover returns what is under the pointer: a rating letter (0..4) or a
// camp (5 + camp), -1 for nothing.
func (c *CampScreen) hover(x, y int) int {
	for row := 0; row < 4; row++ {
		if y <= row*0x34+8 || y >= row*0x34+0x2a {
			continue
		}
		if (x > 5 && x < 0x2d) || (x > 0xde && x < 0x106) {
			return ((x - 6) % 0xd9) >> 3
		}
		if (x > 0x33 && x < 0x83) || (x > 0x88 && x < 0xd9) {
			col := 0
			if x > 0x88 {
				col = 1
			}
			return 5 + col*4 + row
		}
	}
	return -1
}

func (c *CampScreen) Update(app *App) error {
	in := &app.Input
	if c.booked >= 0 {
		c.booked--
		if c.booked < 0 {
			app.Pop()
		}
		return nil
	}
	if MenuButtonHit(in) {
		app.Pop()
		return nil
	}
	h := c.hover(in.X, in.Y)
	if !in.Left || h < 5 {
		return nil
	}
	camp := h - 5
	s := c.ss.S
	switch err := s.BookCamp(c.ss.Mgr(), camp); err {
	case nil:
		c.booked = 100
	case game.ErrCampClosed:
		app.Push(newTsToast(game.CampNames[camp], "hat nicht geöffnet...", nil))
	case game.ErrCampMoney:
		app.Push(newTsToast("Keine finanzielle", "Möglichkeit gegeben...", nil))
	case game.ErrCampTooSoon:
		app.Push(newTsToast("Einmal pro Woche", "reicht doch...", nil))
	default:
		app.Push(newTsToast(err.Error(), "", nil))
	}
	return nil
}

func (c *CampScreen) Draw(app *App, dst *ebiten.Image) {
	g := app.Gfx
	s := c.ss.S
	g.Background(dst)
	for i := 0; i < 8; i++ {
		col, row := i/4, i%4
		Draw(dst, g.Sub(5, rectWH(col*0x82, row*0x27, 0x82, 0x27)), col*0x84+4, row*0x34+7)
		if !s.CampOpen(i) {
			Draw(dst, g.SubKeyed(0x2b, 0, rectWH(0, 0, 0x50, 0x21)), col*0x56+0x34, row*0x34+9)
		}
		y := row*0x34 + 0x33
		g.TextMid(dst, FontSmall, col*0x56+0x2f, col*0x56+0x8d, y, strings.ToUpper(game.CampNames[i]), ColWhite, ColBlack)
		for k := 0; k < 5; k++ {
			g.TextAt(dst, FontSmall, col*0xd9+k*8+8, y, "EFPRV"[k:k+1], tsRatingColor[k], ColBlack)
		}
	}
	g.DrawMenuButton(dst)
	if h := c.hover(app.Input.X, app.Input.Y); h >= 0 && c.booked < 0 {
		var t string
		if h < 5 {
			t = game.CampRatingNames[h]
		} else {
			t = game.CampNames[h-5] + ", " + itoa64(s.CampPrice(c.ss.Mgr(), h-5)) + " DM/Woche."
		}
		g.TextMid(dst, FontNormal, 0, 0x10a, 0xe1, t, ColLight, ColBlack)
	}
	if c.booked >= 0 && c.booked < 100 {
		// The team leaves for the camp (f_112a_072d, PIC/28.VGA).
		g.Window(dst, 0x84, 0x4e, 0x84+0x39, 0x4e+0x3a, ColGrey7)
		Draw(dst, g.Sub(0x1c, rectWH(0, 0, 0x37, 0x38)), 0x85, 0x4f)
	}
}

// ScorerScreen is "Die Besten der Liga" (f_14a4_1ad5): the best scorers
// of the league of the manager, with LIGA (all) and SPIELER (only the
// players of the human managers).
type ScorerScreen struct {
	ss      *Session
	players bool
	rows    []game.Scorer
}

// NewScorerScreen creates the scorer list.
func NewScorerScreen(ss *Session) Scene {
	sc := &ScorerScreen{ss: ss}
	sc.refresh()
	return sc
}

func (sc *ScorerScreen) refresh() {
	sc.rows = sc.ss.S.TopScorers(sc.ss.M().League, sc.players)
	if len(sc.rows) > 20 {
		sc.rows = sc.rows[:20]
	}
}

func (sc *ScorerScreen) Update(app *App) error {
	in := &app.Input
	if MenuButtonHit(in) {
		app.Pop()
		return nil
	}
	if tsButtonHit(in, 0x36, 0xaf) || tsButtonHit(in, 0xd4, 0xaf) {
		sc.players = !sc.players
		sc.refresh()
		return nil
	}
	if (in.Left || in.Right) && in.X > 0x23 && in.X < 0x11d && in.Y > 0x1d && in.Y < 0x22+7*len(sc.rows)-7 {
		if r := (in.Y - 0x1e) / 7; r < len(sc.rows) {
			sc.info(app, sc.rows[r].Person)
		}
	}
	return nil
}

// info opens the player info if the player plays for a human manager or
// is on the transfer market.
func (sc *ScorerScreen) info(app *App, p int) {
	s := sc.ss.S
	for m := 0; m <= game.MarketIndex; m++ {
		if m < game.MarketIndex && m >= len(s.Managers) {
			continue
		}
		for i := 0; i < s.SquadSize(m); i++ {
			if s.Squads[m][i].Person == p {
				app.Push(NewPlayerInfo(sc.ss, m, i))
				return
			}
		}
	}
}

func (sc *ScorerScreen) Draw(app *App, dst *ebiten.Image) {
	g := app.Gfx
	s := sc.ss.S
	g.Background(dst)
	g.Window(dst, 0x23, 8, 0x23+0xfa, 8+0xb5, ColBlue)
	g.DrawMenuButton(dst)
	FillRect(dst, 0x24, 0x1a, 0x11c-0x24+1, 1, g.PalColor(ColLight))
	title := "Die Besten der Liga"
	if sc.players {
		title = "Die Besten der Spieler"
	}
	g.TextMid(dst, FontNormal, 0x23, 0x11d, 0x10, title, ColLight, ColGrey7)
	g.TextAt(dst, FontSmall, 0x26, 0x18, "PL. SPIELER                 VEREIN", ColLight, NoShadow)
	g.TextAt(dst, FontSmall, 0xd2, 0x18, "TORE       TORE/SPIEL", ColLight, NoShadow)
	for i, r := range sc.rows {
		y := 0x22 + i*7
		ps := &s.Persons[r.Person]
		c := ColLight
		if s.ManagerOf(ps.Team) != nil {
			c = ColBeige
		}
		g.TextAt(dst, FontSmall, 0x26, y, tsPad(int64(r.Rank), 2)+". "+ps.Name, c, NoShadow)
		g.TextAt(dst, FontSmall, 0x72, y, s.Teams[ps.Team].Name, c, NoShadow)
		g.TextAt(dst, FontSmall, 0xd4, y, tsPad(int64(r.Goals), 2), c, NoShadow)
		q := r.Goals * 10
		if r.Games != 0 {
			q /= r.Games
		}
		g.TextAt(dst, FontSmall, 0xfc, y, itoa(q/10)+"."+itoa(q%10), c, NoShadow)
	}
	if len(sc.rows) == 0 {
		g.TextMid(dst, FontSmall, 0x23, 0x11d, 0x28, "NOCH KEINE EINTRÄGE VORHANDEN...", ColLight, ColBlack)
	}
	sel := [2]int{17, 10}
	p := 0
	if sc.players {
		p = 1
	}
	tsButton(g, dst, 0x36, 0xaf, "LIGA", sel[p])
	tsButton(g, dst, 0xd4, 0xaf, "SPIELER", sel[1-p])
}
