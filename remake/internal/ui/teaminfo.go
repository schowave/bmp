package ui

import (
	"strings"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/schowave/bmp/remake/internal/game"
)

// TeamInfo is the popup "Info über <CLUB>" (f_2a41_000e) with its second
// page, the remaining fixtures of the club (f_0000_28c4, RESTPROGRAMM).
type TeamInfo struct {
	ss   *Session
	team int
	// mode is the table the popup was opened from (param_2): 0 home,
	// 1 away, 2 total. It selects the goals per game, the series and the
	// records shown.
	mode int

	rest   bool // the RESTPROGRAMM page is shown
	second bool // RESTPROGRAMM shows the second half of the season
}

// Geometry of the original (param_3 = 0): the window from 0x59,0x37,
// 0xe0 wide and 0xae high, the buttons at y 0xd6.
const (
	tiX    = 0x59
	tiY    = 0x37
	tiBtnY = 0xd6
)

// NewTeamInfo shows the information about a club as opened from a cup
// list (mode 2, total).
func NewTeamInfo(ss *Session, team int) Scene { return NewTeamInfoMode(ss, team, 2) }

// NewTeamInfoMode shows the information about a club; mode is the table
// it is opened from: 0 HEIM, 1 AUSWÄRTS, 2 GESAMT.
func NewTeamInfoMode(ss *Session, team, mode int) *TeamInfo {
	return &TeamInfo{ss: ss, team: team, mode: mode}
}

var (
	tiModeNames = [3]string{"(HEIM):  ", "(AUSWÄRTS):  ", "(GESAMT):  "} // D:0x4af8
	// tiVenue is D:0x54d8: the series venue of each mode.
	tiVenue      = [3]int{game.VenueHome, game.VenueAway, game.VenueTotal}
	tiVenueNames = [3]string{" ", "ZUHAUSE ", "AUSWÄRTS "} // D:0x4ad0
	tiSeries     = [game.NumSeries]string{"GEWONNEN", "VERLOREN", "UNENTSCHIEDEN",
		"NICHT GEWONNEN", "NICHT VERLOREN", "OHNE GEGENTOR", "OHNE TORERFOLG"} // D:0x4adc
	tiRecordNames = [8]string{"HÖCHSTER H-SIEG", "HÖCHSTE H-NIEDERL.", "ERZIELTE H-TORE",
		"KASSIERTE H-TORE", "HÖCHSTER A-SIEG", "HÖCHSTE A-NIEDERL.", "ERZIELTE A-TORE",
		"KASSIERTE A-TORE"} // D:0x2474
	tiTotalRecordNames = [4]string{"HÖCHSTER SIEG", "HÖCHSTE NIEDERL.", "ERZIELTE TORE",
		"KASSIERTE TORE"} // D:0x4b04
)

// tiButtonX are the x positions of the three dialog buttons.
var tiButtonX = [3]int{tiX + 4, tiX + 0x54, tiX + 0xa4}

func (ti *TeamInfo) german() bool { return ti.team < game.NumGerman }

// buttons returns the labels of the buttons in the order of the original
// (f_3091_123d registers them; f_3091_11cc returns 1 + the index).
func (ti *TeamInfo) buttons() (xs []int, labels []string) {
	if ti.rest {
		other := "RÜCKRUNDE "
		if ti.second {
			other = "HINRUNDE "
		}
		return tiButtonX[:], []string{"REICHT MIR", "INFO", other}
	}
	if ti.german() {
		return tiButtonX[:], []string{"REICHT MIR", "RESTPROGRAMM", "ANZEIGEN"}
	}
	return []int{tiButtonX[0], tiButtonX[2]}, []string{"REICHT MIR", "ANZEIGEN"}
}

// hit returns 1 + the index of the button under a left click, or 0
// (f_3091_11cc: strictly inside x..x+0x39, y..y+0xc).
func (ti *TeamInfo) hit(in *Input) int {
	if !in.Left {
		return 0
	}
	xs, _ := ti.buttons()
	for i, x := range xs {
		if in.X > x && in.X < x+0x39 && in.Y > tiBtnY && in.Y < tiBtnY+0xc {
			return i + 1
		}
	}
	return 0
}

func (ti *TeamInfo) Update(app *App) error {
	in := &app.Input
	if in.Right || in.Escape {
		app.Pop()
		return nil
	}
	b := ti.hit(in)
	s := ti.ss.S
	switch {
	case b == 0:
	case ti.rest:
		switch b {
		case 1:
			app.Pop()
		case 2:
			ti.rest = false
		case 3:
			ti.second = !ti.second
		}
	case b == 1:
		app.Pop()
	case b == 2 && ti.german():
		l := game.LeagueOf(ti.team)
		ti.rest = true
		ti.second = s.Matchday[l] > game.MatchdaysPerLeague[l]/2
	default:
		s.ToggleTicker(ti.team)
	}
	return nil
}

// button draws a dialog button (f_3091_123d): black with a bevel, the
// label in colour c.
func tiButton(g *Gfx, dst *ebiten.Image, x int, label string, c int) {
	g.BevelButton(dst, x, tiBtnY, 0x39, 0xc, ColBlack)
	g.TextMid(dst, FontSmall, x+1, x+0x39, tiBtnY+8, label, c, NoShadow)
}

func (ti *TeamInfo) Draw(app *App, dst *ebiten.Image) {
	g := app.Gfx
	s := ti.ss.S
	g.Window(dst, tiX, tiY, tiX+0xe0, tiY+0xae, ColBlue)
	title := "Info über " + s.Teams[ti.team].Name
	if ti.rest {
		title = "Hinspiele " + s.Teams[ti.team].Name
		if ti.second {
			title = "Rückspiele " + s.Teams[ti.team].Name
		}
	}
	g.TextMid(dst, FontNormal, tiX, tiX+0xe0, 0x3f, title, ColBeige, ColBlack)
	FillRect(dst, tiX+1, 0x41, 0xdf, 1, g.PalColor(ColBeige))

	xs, labels := ti.buttons()
	for i, x := range xs {
		c := ColSand
		if !ti.rest && i == len(xs)-1 && s.Ticker(ti.team) {
			c = ColRedDark // D:0x54dc: 10, 17
		}
		tiButton(g, dst, x, labels[i], c)
	}
	switch {
	case ti.rest:
		ti.drawRest(g, dst)
	case !ti.german():
		for i, l := range []string{"IHRE SPÄHER KÖNNEN LEIDER NUR", "AUF DEUTSCHEM BODEN OPERIEREN...",
			"WILL SAGEN: KEINE INFORMATIONEN VERFÜGBAR."} {
			g.TextMid(dst, FontSmall, tiX, tiX+0xe0, []int{0x7b, 0x83, 0x8d}[i], l, ColLight, ColBlack)
		}
	default:
		ti.drawInfo(g, dst)
	}
}

func (ti *TeamInfo) drawInfo(g *Gfx, dst *ebiten.Image) {
	s := ti.ss.S
	t := ti.team
	c := &s.Clubs[t]
	left := func(y int, txt string, col int) { g.TextAt(dst, FontSmall, tiX+3, y, txt, col, NoShadow) }
	right := func(y int, txt string) { g.TextAt(dst, FontSmall, tiX+0x6b, y, txt, ColLight, NoShadow) }

	head := itoa(c.Pos+1) + ". PLATZ IN DER " + strings.ToUpper(game.LeagueNames[game.LeagueOf(t)]) +
		", STÄRKE: " + itoa(s.InfoStrength(t))
	g.TextMid(dst, FontSmall, tiX, tiX+0xe0, 0x49, head, ColLight, ColBlack)

	balance := func(v int) string {
		return itoa(c.Wins[v]) + " SIEGE, " + itoa(c.Games[v]-c.Wins[v]-c.Losses[v]) + " UNENT., " +
			itoa(c.Losses[v]) + " NIED."
	}
	left(0x53, "HEIMBILANZ:", ColLight)
	right(0x53, balance(0))
	left(0x5b, "AUSWÄRTSBILANZ:", ColLight)
	right(0x5b, balance(1))
	left(99, "DIE LETZTEN SPIELE:", ColLight)
	for i := 0; i < 8 && i < len(c.Form); i++ {
		g.TextAt(dst, FontSmall, tiX+0x6b+i*7, 99, c.Form[i:i+1], ColLight, NoShadow)
	}

	left(0x6b, "TORE PRO SPIEL "+tiModeNames[ti.mode], ColLight)
	right(0x6b, goalsPerGame(c, ti.mode))

	left(0x73, "SERIEN "+tiModeNames[ti.mode], ColLight)
	y := 0x7b
	venue := tiVenue[ti.mode]
	ser := s.TeamSeries(t)
	for k, n := 0, 0; k < game.NumSeries && n < 4; k++ {
		if v := ser[k][venue]; v > 1 {
			left(y, itoa(v)+" SPIELE IN FOLGE "+tiSeries[k]+tiVenueNames[venue], ColLight)
			y += 8
			n++
		}
	}
	if y == 0x7b {
		g.TextAt(dst, FontSmall, tiX+10, 0x7b, "KEINE BEMERKENSWERTE SERIE VORHANDEN.", ColGrey2, NoShadow)
		y = 0x83
	}
	y += 2

	rec := s.TeamRecords(t)
	for i := 0; i < 8; i++ {
		if (ti.mode == 0) != (i < 4) {
			continue
		}
		if ti.mode == 2 {
			left(y, tiTotalRecordNames[i-4], ColLight)
		} else {
			left(y, tiRecordNames[i], ColLight)
		}
		var txt string
		if ti.mode == 2 {
			h, a := rec[i-4], rec[i]
			txt = recordText(s, i-4, h)
			if h.Raw != 0 || a.Raw != 0 {
				hv, av := h.Raw, a.Raw
				if i < 6 {
					hv, av = h.Hi()-h.Lo(), a.Hi()-a.Lo()
				}
				if av < hv {
					txt += " (H)"
				} else {
					txt = recordText(s, i, a) + " (A)"
				}
			}
		} else {
			txt = recordText(s, i, rec[i])
		}
		right(y, txt)
		y += 8
	}

	y2 := y + 2
	own := ti.ss.M().Team
	if own != t {
		left(y2, "HISTORISCHE ERGEBNISSE GEGEN "+s.Teams[own].Name, ColLight)
		home, away := s.HeadToHead(t, own)
		txt := "H: "
		for k, games := range [][]game.PlayedGame{home, away} {
			for i, gm := range games {
				if i > 0 {
					txt += ","
				}
				txt += itoa(gm.For) + ":" + itoa(gm.Against)
			}
			if k == 0 {
				if len(home) == 0 {
					txt += " NOCH KEINE"
				}
				txt += " -- A: "
			}
		}
		if strings.HasSuffix(txt, " ") {
			txt += " NOCH KEINE"
		}
		left(y+10, txt, ColLight)
		y2 = y + 0x14
	}
	txt := "PUNKTE IN DER EWIGEN TABELLE: "
	if c.Eternal == 0 {
		txt += "NOCH KEINEN EINZIGEN..."
	} else {
		txt += game.FormatMoney(c.Eternal)
	}
	left(y2, txt, ColGrey2)
}

// recordText is f_2a41_0f0e: the record k of a club as text.
func recordText(s *game.State, k int, r game.Record) string {
	if r.Raw == 0 {
		return "NOCH KEINE EINTRAGUNG."
	}
	var v string
	switch k & 3 {
	case game.RecordWin, game.RecordDefeat:
		v = itoa(r.Hi()) + ":" + itoa(r.Lo())
	default:
		n := r.Hi()
		if n == 0 {
			n = r.Lo()
		}
		v = itoa(n)
	}
	return v + " (" + s.Teams[r.Opp].Name + ")"
}

// goalsPerGame is f_262f_21e1: goals and goals against per game with one
// decimal, "1.5:0.8".
func goalsPerGame(c *game.ClubStats, mode int) string {
	games, goals, against := 0, 0, 0
	for v := 0; v < 2; v++ {
		if mode == 2 || mode == v {
			games += c.Games[v]
			goals += c.Goals[v]
			against += c.Against[v]
		}
	}
	f := func(n int) string {
		x := 0
		if games != 0 {
			x = n * 10 / games
		}
		return itoa(x/10) + "." + itoa(x%10)
	}
	return f(goals) + ":" + f(against)
}

// drawRest draws the remaining fixtures (f_0000_28c4): the matchdays of
// one half of the season with opponent, its place and strength, and the
// date or the result.
func (ti *TeamInfo) drawRest(g *Gfx, dst *ebiten.Image) {
	s := ti.ss.S
	t := ti.team
	l := game.LeagueOf(t)
	g.TextAt(dst, FontSmall, 0x68, 0x48, "GEGNER", ColLight, NoShadow)
	g.TextAt(dst, FontSmall, 0xbe, 0x48, "PLATZ (STÄRKE)", ColLight, NoShadow)
	g.TextMid(dst, FontSmall, 0x108, 0x108, 0x48, "DATUM", ColLight, NoShadow)
	g.TextAt(dst, FontSmall, 0x115, 0x48, "ERGEBNIS", ColLight, NoShadow)
	FillRect(dst, 0x5a, 0x4a, 0x138-0x5a+1, 1, g.PalColor(ColLight))

	half := game.MatchdaysPerLeague[l] / 2
	first := 1
	if ti.second {
		first += half
	}
	y := 0x51
	for md := first; md < first+half; md++ {
		for p, pr := range game.Fixtures(l, md) {
			var opp, col int
			var txt string
			switch t {
			case pr.Home:
				opp, col, txt = pr.Away, ColLight, "(H) "
			case pr.Away:
				opp, col, txt = pr.Home, ColGrey2, "(A) "
			default:
				continue
			}
			g.TextAt(dst, FontSmall, 0x5e, y, txt+s.Teams[opp].Name, col, NoShadow)
			place := itoa(s.Clubs[opp].Pos+1) + ". (" + itoa(s.InfoStrength(opp)) + ")"
			g.TextAt(dst, FontSmall, 199, y, place, col, NoShadow)
			r := s.Results[l][md-1][p]
			if r.Home == game.Postponed || md >= s.Matchday[l] {
				d := game.DateOf(s.StartYear, game.EventDay(s.GameEvent(l, md, p)))
				g.TextMid(dst, FontSmall, 0x10a, 0x10a, y, itoa(d.Day)+"."+itoa(d.Month+1)+".", col, NoShadow)
			} else {
				own, other := r.Home, r.Away
				if t == pr.Away {
					own, other = other, own
				}
				g.TextAt(dst, FontSmall, 0x11b, y, itoa(own), col, NoShadow)
				g.TextAt(dst, FontSmall, 0x120, y, ":", col, NoShadow)
				g.TextAt(dst, FontSmall, 0x125, y, itoa(other), col, NoShadow)
			}
			y += 7
		}
	}
}
