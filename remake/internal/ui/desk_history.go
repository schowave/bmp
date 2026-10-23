package ui

import (
	"github.com/hajimehoshi/ebiten/v2"

	"github.com/schowave/bmp/remake/internal/game"
)

// HistoryScreen is the history of the manager (f_262f_390e): one row per
// season with place, league, DFB-Pokal and European cup, 16 rows per page,
// and the bar graph "Gesamtentwicklung" of the overall places. A click into
// the list shows the next page; the HAUPT MENU button leaves.
type HistoryScreen struct {
	ss   *Session
	page int
}

// NewHistoryScreen creates the history screen.
func NewHistoryScreen(ss *Session) Scene { return &HistoryScreen{ss: ss} }

// historyRounds are the slots from D:0x24c0: the DFB round of a season is
// slot byte&7, the European round slot (byte&7)+1.
var historyRounds = [9]string{"AUSGABEN (GEHÄLTER, REISEKOSTEN)", "", "1.Runde ", "Achtelfinale ",
	"Viertelfinale ", "Halbfinale ", "Endspiel ", "DfB-Pokal", "Pokal der Landesmeister"}

// historyCups are D:0x24dc: the cup names of the European column.
var historyCups = [4]string{"DfB-Pokal", "Pokal der Landesmeister", "Pokal der Pokalsieger", "UEFA-Pokal"}

func (h *HistoryScreen) rows() int { return min(len(h.ss.M().History), 50) }

func (h *HistoryScreen) Update(app *App) error {
	in := &app.Input
	switch {
	case deskMenuHit(in):
		app.Pop()
	case in.Clicked(5, 8, 0x135, 0x90) && (h.page+1)*16 < h.rows():
		h.page++
	}
	return nil
}

func (h *HistoryScreen) Draw(app *App, dst *ebiten.Image) {
	g := app.Gfx
	s := h.ss.S
	m := h.ss.M()
	g.Background(dst)
	g.Window(dst, 5, 0x9d, 5+0x102, 0x9d+0x49, ColBlack)
	g.Window(dst, 5, 8, 5+0x134, 8+0x8f, ColBlue)
	g.DrawMenuButton(dst)
	g.TextAt(dst, FontNormal, 10, 0x11, "Sais. Pl.   Liga       DfB-Pokal       Europapokal", ColBeige, -1)
	line6(dst, 5, 0x13, 0x138, 0x13, g.C(ColBeige))
	g.TextMid(dst, FontNormal, 6, 0x106, 0xa5, "Gesamtentwicklung", ColBeige, -1)
	line6(dst, 6, 0xa7, 0x106, 0xa7, g.C(ColBeige))

	n := h.rows()
	// the bar graph: colours by the league of the place
	for i := 0; i < n; i++ {
		rank := int(m.History[i][0])
		c := 0x1b
		for y := 0x3a; y > 0; y-- {
			if game.FirstID[2]+1 > y {
				c = 0x1a
			}
			if game.FirstID[1]+1 > y {
				c = 0x19
			}
			if rank <= y {
				yy := int(uint8(y - 0x56))
				line6(dst, 5*i+7, yy, 5*i+0xb, yy, g.C(c))
			}
		}
	}

	if s.Seasons == 0 || n == 0 {
		g.TextMid(dst, FontSmall, 0, 0x140, 0x1d, "NOCH HABEN SIE KEINE HISTORISCH SONDERLICH WICHTIGEN ERFOLGE ERZIELT...", ColLight, ColGrey7)
		return
	}
	base := 1963
	if s.StartYear-s.Seasons > 1970 {
		base = 1991
	}
	y := 0x1d
	for r := h.page * 16; r < n && r < (h.page+1)*16; r++ {
		e := m.History[r]
		width := 2
		if r == 0 {
			width = 1
		}
		g.TextAt(dst, FontSmall, 0xc, y, numText(int64(base+m.DroppedSeasons+r), width, true), ColLight, -1)
		if e[0] == 0xff {
			g.TextAt(dst, FontSmall, 0x2a, y, "EXISTIERTEN SIE NOCH GAR NICHT ALS MANAGER...", ColLight, -1)
		} else {
			l := int(e[1]) % game.NumLeagues
			c := ColLight
			if e[0] == 1 && e[1] == 0 {
				c = ColBeige
			}
			g.TextAt(dst, FontSmall, 0x2a, y, numText(int64(int(e[0])-game.FirstID[l]), 2, false)+".", c, -1)
			g.TextMid(dst, FontSmall, 0x55, 0x55, y, game.LeagueNames[l], ColLight, -1)
			// the DFB-Pokal
			buf, c := game.LeagueNames[l], ColLight
			txt := "SIEGER"
			if e[2]&0x80 != 0 {
				c = ColBeige
			} else {
				buf = historyRounds[e[2]&7]
				txt = buf
			}
			g.TextMid(dst, FontSmall, 0x90, 0x90, y, txt, c, -1)
			// the European cups; a season without a round keeps the
			// previous text like the original
			c = ColLight
			if e[3]&0x80 != 0 {
				buf, c = "SIEGER ", ColBeige
			} else if r := e[3] & 7; r != 0 {
				buf = historyRounds[r+1]
			}
			if e[3]&0xf8 == 0 {
				txt = "nicht im wettbewerb"
			} else {
				cup := int(e[3]&0x18) >> 3
				name := historyCups[cup]
				if cup == 1 || cup == 2 {
					name = name[:18-cup] + "."
				}
				txt = buf + name
			}
			g.TextMid(dst, FontSmall, 0xb4, 0x139, y, txt, c, -1)
		}
		if (r+1)&0xf != 0 {
			line6(dst, 6, y+2, 0x138, y+2, g.C(ColGrey6))
		}
		y += 8
	}
}
