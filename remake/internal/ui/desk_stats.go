package ui

import (
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/schowave/bmp/remake/internal/game"
)

// StatsScreen is the statistics of the desk (f_262f_0af7): series and
// records of the club, attendance, finances and the monthly income and
// expenses. The original draws it once and waits for the HAUPT MENU
// button, so the values are computed when the screen opens.
type StatsScreen struct {
	ss *Session

	series, best [game.NumSeries][3]int
	records      [game.NumRecords]game.Record
	club         game.ClubStats
	required     int64
	income       int64
	expenses     int64
}

// NewFinanceScreen creates the statistics screen.
func NewFinanceScreen(ss *Session) Scene {
	s, mi := ss.S, ss.Mgr()
	t := ss.M().Team
	st := &StatsScreen{ss: ss, series: s.TeamSeries(t), best: s.BestSeries(t), records: s.TeamRecords(t)}
	if t < game.NumGerman {
		st.club = s.Clubs[t]
	}
	st.required = s.RequiredAverage(mi)
	st.income = s.MonthlyIncome(mi)
	st.expenses = s.MonthlyExpenses(mi)
	return st
}

// statsRows are the texts of D:0x2448, rows 0..30; '!' marks a heading.
var statsRows = [31]string{
	"!SERIEN", "GEWONNEN", "VERLOREN", "UNENTSCHIEDEN", "NICHT GEWONNEN", "NICHT VERLOREN",
	"OHNE GEGENTOR", "OHNE TORERFOLG", "TORE/SPIEL", "GEGENTORE/SPIEL",
	"!REKORDE", "HöCHSTER H-SIEG", "HöCHSTE H-NIEDERL.", "ERZIELTE H-TORE", "KASSIERTE H-TORE",
	"HöCHSTER A-SIEG", "HöCHSTE A-NIEDERL.", "ERZIELTE A-TORE", "KASSIERTE A-TORE",
	"!ZUSCHAUER", "BESUCHERZAHL", "SCHNITT/SPIEL", "BEN|TIGTER SCHNITT", "ZUSCHAUERREKORD: ", "MINUSKULISSE: ",
	"!FINANZEN", "KONTOSTAND", "SCHULDEN GESAMT",
	"!EINNAHMEN+AUSGABEN / MONAT", "EINNAHMEN (SPENDEN, SPONSOREN)", "AUSGABEN (GEH[LTER, REISEKOSTEN)",
}

// statsVenue is D:0x54b2: the venue of the goal columns G, H, A for
// f_262f_21e1 (0 home, 1 away, 2 both).
var statsVenue = [3]int{2, 0, 1}

// deskSeriesText is f_262f_0a9f: a series length, "^@" (a dash) for none.
func deskSeriesText(v int) string {
	if v == 0 {
		return "^@"
	}
	return numText(int64(v), 2, false)
}

// deskPerGame is f_262f_21e1 for one column: goals (against) per game of
// the venue with one decimal.
func deskPerGame(c *game.ClubStats, venue int, against bool) string {
	games, goals := 0, 0
	for v := 0; v < 2; v++ {
		if venue == 2 || venue == v {
			games += c.Games[v]
			if against {
				goals += c.Against[v]
			} else {
				goals += c.Goals[v]
			}
		}
	}
	x := 0
	if games != 0 {
		x = goals * 10 / games
	}
	return numText(int64(x/10), 1, false) + "." + numText(int64(x%10), 1, false)
}

// deskUmlauts turns the umlaut codes of the original strings into Go text.
func deskUmlauts(s string) string {
	r := []rune{}
	for _, c := range s {
		switch c {
		case '[':
			c = 'Ä'
		case '|':
			c = 'ö'
		case ']':
			c = 'Ü'
		case '{':
			c = 'ä'
		case '}':
			c = 'ü'
		case '~':
			c = 'ß'
		}
		r = append(r, c)
	}
	return string(r)
}

func (st *StatsScreen) Update(app *App) error {
	if MenuButtonHit(&app.Input) {
		app.Pop()
	}
	return nil
}

func (st *StatsScreen) Draw(app *App, dst *ebiten.Image) {
	g := app.Gfx
	s := st.ss.S
	m := st.ss.M()
	g.Background(dst)
	g.Window(dst, 5, 8, 5+0xc3, 8+0xb4, ColBlue)
	g.Window(dst, 0xcd, 8, 0xcd+0x6c, 8+0xb4, ColBlue)
	g.Window(dst, 5, 0xc0, 5+0xc3, 0xc0+0x27, ColBlue)
	g.DrawMenuButton(dst)

	x, y := 8, 0x10
	for row, label := range statsRows {
		if row == 0x13 {
			x, y = 0xd1, 0x10
		}
		if row == 0x1c {
			x, y = 8, 0xc5
		}
		label = deskUmlauts(label)
		if label[0] == '!' {
			y += 6
			g.deskHeading(dst, x, y-2, label[1:])
			y += 0xb
			continue
		}
		g.TextAt(dst, FontSmall, x, y, label, ColLight, -1)
		val := func(vx, vy int, txt string) { g.TextAt(dst, FontSmall, vx, vy, txt, ColGrey2, -1) }
		switch {
		case row < 10:
			for c := 0; c < 3; c++ {
				var txt string
				switch row {
				case 8, 9:
					txt = deskPerGame(&st.club, statsVenue[c], row == 9)
				default:
					txt = deskSeriesText(st.series[row-1][c]) + "(" + deskSeriesText(st.best[row-1][c]) + ")"
				}
				val(c*0x22+0x54, y, txt)
			}
		case row < 0x13:
			val(7*12, y, recordText(s, row-0xb, st.records[row-0xb]))
		case row < 0x1c:
			txt := ""
			var num int64 = -1 // shown after the label (records)
			switch row {
			case 0x14:
				txt = "(NOCH KEINE...)"
				if m.HomeGames != 0 {
					txt = numText(m.SeasonAudience, 2, false)
				}
			case 0x15:
				txt = "(NOCH KEINER...)"
				if m.HomeGames != 0 {
					txt = numText(m.SeasonAudience/int64(m.HomeGames), 2, false)
				}
			case 0x16:
				txt = numText(st.required, 2, false)
			case 0x17:
				txt = "(NOCH KEINER...)"
				if m.RecordAudience != 0 {
					txt = "(" + s.Teams[m.RecordOpponent].Name + ")"
					num = m.RecordAudience
				}
			case 0x18:
				txt = "(NOCH KEINE...)"
				if m.MinAudience != 99999 {
					txt = "(" + s.Teams[m.MinOpponent].Name + ")"
					num = m.MinAudience
				}
			case 0x1a:
				txt = numText(m.Money, 2, false) + " DM"
			case 0x1b:
				txt = numText(m.Debts(), 2, false) + " DM"
			}
			if num > 0 {
				val(x+g.TextWidth(FontSmall, label), y, numText(num, 2, false))
			}
			val(x, y+6, txt)
		default:
			v := st.income
			if row == 0x1e {
				v = st.expenses
			}
			val(x+0x8c, y, numText(v, 7, false)+" DM")
		}
		switch {
		case row > 0x1b:
			y += 8
		case row > 0x12:
			y += 15
		default:
			y += 8
		}
		if row == 0x18 {
			y += 0x12
		}
	}
	g.TextAt(dst, FontNormal, 0x5c, 0x14, "G       H       A", ColBeige, -1)

	// attendance graph
	fillIncl(dst, 0xd1, 0x69, 0x130, 0x76, g.PalColor(ColBlack))
	frameIncl(dst, 0xd0, 0x68, 0x131, 0x77, g.PalColor(ColGrey6))
	// meter of the balance
	fillIncl(dst, 0xd1, 0xab, 0x130, 0xb8, g.PalColor(ColBlack))
	frameIncl(dst, 0xd0, 0xaa, 0x131, 0xb9, g.PalColor(ColGrey6))
	n := min(m.HomeGames, len(m.Attendance))
	hi, lo := int8(-127), int8(127)
	for i := 0; i < n; i++ {
		v := int(uint8(m.Attendance[i]))
		if v > int(hi) {
			hi = int8(v)
		}
		if int(lo) > v {
			lo = int8(v)
		}
	}
	rng := int(int8(hi - lo))
	if rng == 0 {
		rng = 1
	}
	for i := 0; i < n; i++ {
		h := (int(uint8(m.Attendance[i]))-int(lo))*11/rng + 2
		frameIncl(dst, 5*i+0xd1, 0x76-h, 5*i+0xd6, 0x76, statsPal(0x15))
		fillIncl(dst, 5*i+0xd2, 0x77-h, 5*i+0xd6, 0x76, g.PalColor(ColRedDark))
	}
	v := m.Money - m.Debts()
	k := 0
	t := int64(-2375000)
	for x := 0xd1; x < 0x131; x++ {
		if v < t && v+50000 >= t {
			lineIncl(dst, x, 0xab, x, 0xb8, statsPal(k+0x13))
		} else {
			lineIncl(dst, x, 0xae, x, 0xb5, statsPal(k+0x13))
		}
		if (x+1-0xd1)%7 == 0 && k < 0xc {
			k++
		}
		t += 50000
	}
	if v <= -2375000 {
		lineIncl(dst, 0xd1, 0xab, 0xd1, 0xb8, statsPal(0x13))
	}
	if v > 2325000 {
		lineIncl(dst, 0x130, 0xab, 0x130, 0xb8, statsPal(0x1f))
	}
}

// statsPalette is D:0x5498: the colours 31 down to 19 the statistics
// screen sets for its meter (green to red, 4 bits per component).
var statsPalette = [13]int{0x0c0, 0x2c0, 0x4c0, 0x6c0, 0x8c0, 0xac0, 0xcc0, 0xca0, 0xd80, 0xd60, 0xd40, 0xd20, 0xd00}

// statsPal returns the colour p (19..31) of the statistics screen.
func statsPal(p int) color.RGBA {
	w := statsPalette[0x1f-p]
	return deskVGA((w>>8&15)<<2, (w>>4&15)<<2, (w&15)<<2)
}
