package ui

import (
	"github.com/hajimehoshi/ebiten/v2"

	"github.com/schowave/bmp/remake/internal/game"
)

// CupScreen shows the pairings of the current round of a cup with their
// results (f_17c2_1ccb): from the menu, after a draw and during a cup match
// day. The DFB-Pokal is grouped by the leagues of the two clubs.
type CupScreen struct {
	ss     *Session
	cup    game.Cup
	minute int // 0: overview with HAUPTMENÜ, else the minute of a match day
	done   func(app *App)

	y0   int // top of the window
	rows []cupRow
}

type cupRow struct {
	y          int // baseline
	home, away int // team ids, -1 for a header line
}

// cupGroups are the league classes (1 Bundesliga, 2 Zweite Liga, 4
// Amateur-Oberliga) of the DFB groups, D:0x0655 and D:0x065d: away*8+home
// in one order or the other.
var cupGroups = [6][2]int{{9, 9}, {10, 17}, {12, 33}, {18, 18}, {34, 20}, {36, 36}}

var cupGroupTitles = [6]string{
	"BUNDESLIGA GEGEN BUNDESLIGA:",
	"BUNDESLIGA GEGEN ZWEITE LIGA:",
	"BUNDESLIGA GEGEN AM.-OBERLIGA:",
	"ZWEITE LIGA GEGEN ZWEITE LIGA:",
	"ZWEITE LIGA GEGEN AM.-OBERLIGA:",
	"AM.-OBERLIGA GEGEN AM.-OBERLIGA:",
}

// NewCupScreen shows the current round of cup c from the menu.
func NewCupScreen(ss *Session, c game.Cup) Scene { return NewCupRound(ss, c, 0, nil) }

// NewCupRound shows the round of cup c; minute > 0 shows the state of a
// match day with the WEITER button. done runs when the screen closes.
func NewCupRound(ss *Session, c game.Cup, minute int, done func(app *App)) *CupScreen {
	return &CupScreen{ss: ss, cup: c, minute: minute, done: done}
}

func leagueClass(id int) int {
	if id >= game.NumGerman {
		return 0
	}
	return 1 << game.LeagueOf(id)
}

// layout computes the window and the rows like the original.
func (c *CupScreen) layout() (rowsH int, title string) {
	s := c.ss.S
	pairs := s.CupPairs(c.cup)
	lines := len(pairs) + 1
	if c.cup == game.CupDFB {
		for _, g := range cupGroups {
			empty := true
			for _, p := range pairs {
				k := leagueClass(p.Away)*8 + leagueClass(p.Home)
				if k == g[0] || k == g[1] {
					empty = false
				}
			}
			if empty {
				lines++
			}
		}
	} else {
		lines -= 6
	}
	rowsH = lines*7 + 0x3a
	mid := int(int8((0xe9 - rowsH) / 2))
	c.y0 = mid - 0x14

	round := s.Cups.Round[game.CupDFB]
	if c.cup != game.CupDFB {
		round = s.Cups.Round[game.CupLandesmeister]
	}
	title = game.RoundNames[round] + " " + game.CupNames[c.cup]
	if c.minute != 0 && s.Day >= 0x13b {
		// The relegation plays in the slots of the Landesmeister
		// (f_0310_02bf); during its match day f_17c2_1ccb names it.
		title = "Relegation"
	}
	if c.minute == 0 {
		if d, ok := c.nextDate(); ok {
			title += "    " + itoa(d.Day) + "." + itoa(d.Month+1) + "."
		}
	} else {
		title += "    " + itoa(c.minute) + ".Minute"
	}

	c.rows = c.rows[:0]
	y := c.y0 + 16
	if c.cup == game.CupDFB {
		y += 8
	}
	for gi, g := range cupGroups {
		if c.cup == game.CupDFB {
			c.rows = append(c.rows, cupRow{y: y, home: -1, away: gi})
		} else if gi > 0 {
			break
		}
		y += 7
		empty := true
		for _, p := range pairs {
			k := leagueClass(p.Away)*8 + leagueClass(p.Home)
			if c.cup != game.CupDFB || k == g[0] || k == g[1] {
				c.rows = append(c.rows, cupRow{y: y, home: p.Home, away: p.Away})
				y += 7
				empty = false
			}
		}
		if c.cup == game.CupDFB && empty {
			c.rows = append(c.rows, cupRow{y: y, home: -1, away: -1})
			y += 7
		}
	}
	return rowsH, title
}

// nextDate is the date of the next event of the cup, if the season still
// has one.
func (c *CupScreen) nextDate() (game.Date, bool) {
	s := c.ss.S
	if s.Day > 0x13a {
		return game.Date{}, false
	}
	mask := byte(c.cup * 2)
	if mask == 0 {
		mask = 1
	}
	for e := s.Event; e < game.NumEvents; e++ {
		if v := s.Events[e]; v >= 8 && mask&(v>>3) != 0 {
			return game.DateOf(s.StartYear, game.EventDay(e)), true
		}
	}
	return game.Date{}, false
}

func (c *CupScreen) button() Button {
	return MenuButton
}

func (c *CupScreen) Update(app *App) error {
	in := &app.Input
	c.layout()
	if b := c.button(); b.Hit(in) {
		app.Pop()
		if c.done != nil {
			c.done(app)
		}
		return nil
	}
	if in.Left && in.Y > c.y0+5 {
		for _, r := range c.rows {
			if r.home >= 0 && in.Y > r.y-6 && in.Y <= r.y {
				t := r.home
				if in.X > 0x82 {
					t = r.away
				}
				if t < game.NumGerman {
					app.Push(NewTeamInfo(c.ss, t))
				}
				return nil
			}
		}
	}
	return nil
}

func (c *CupScreen) Draw(app *App, dst *ebiten.Image) {
	g := app.Gfx
	s := c.ss.S
	g.Background(dst)
	rowsH, title := c.layout()
	g.Window(dst, 5, c.y0+5, 5+0x134, c.y0+5+rowsH-8, ColBlue)
	icon := IconMainMenu
	if c.minute != 0 {
		icon = IconWeiter
	}
	g.DrawIconButton(dst, icon, MenuButton.X, MenuButton.Y)
	FillRect(dst, 7, c.y0+15, 0x137-7+1, 1, g.PalColor(ColBeige))
	g.TextMid(dst, FontNormal, 5, 0x139, c.y0+13, title, ColBeige, ColGrey7)

	colour := func(t int) int {
		if c.cup == game.CupDFB {
			if s.ManagerOf(t) != nil {
				return ColBeige
			}
		} else if t < game.NumGerman {
			return ColBeige
		}
		return ColLight
	}
	pairs := s.CupPairs(c.cup)
	for _, r := range c.rows {
		switch {
		case r.home == -1 && r.away >= 0:
			g.TextMid(dst, FontSmall, 0, 0x12b, r.y, cupGroupTitles[r.away], ColGrey2, ColBlack)
		case r.home == -1:
			g.TextMid(dst, FontSmall, 0, 0x118, r.y, "@@@@@@@@@", ColGrey2, ColGrey7)
		default:
			k := 0
			for i, p := range pairs {
				if p.Home == r.home && p.Away == r.away {
					k = i
				}
			}
			g.TextMid(dst, FontSmall, 0, 0x82, r.y, s.Teams[r.home].Name, colour(r.home), ColGrey7)
			g.TextMid(dst, FontSmall, 0, 0x118, r.y, "GEGEN", ColGrey3, ColBlack)
			g.TextMid(dst, FontSmall, 0x8c, 0x10e, r.y, s.Teams[r.away].Name, colour(r.away), ColGrey7)
			h, a := s.Cups.Result[c.cup][2*k], s.Cups.Result[c.cup][2*k+1]
			res := "(" + itoa(game.ResultOf(h)) + ":" + itoa(a) + ")"
			switch {
			case h > 19:
				res += "n.E."
			case h > 9:
				res += "n.V."
			}
			x := 0x10c
			if c.cup != game.CupDFB {
				x -= 0x13
			}
			g.TextAt(dst, FontSmall, x, r.y, res, ColLight, NoShadow)
			if c.cup != game.CupDFB {
				f := s.Cups.First[c.cup]
				g.TextAt(dst, FontSmall, 0x11d, r.y, "("+itoa(f[2*k])+":"+itoa(f[2*k+1])+")", ColLight, NoShadow)
			}
		}
	}
}
