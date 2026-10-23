package ui

import (
	"image"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/schowave/bmp/remake/internal/game"
)

// EternalScreen is the eternal table of the German clubs and the eternal
// balance of the manager (f_262f_18e7). Like the original it is drawn once
// and waits for the HAUPT MENU button.
type EternalScreen struct {
	ss  *Session
	ids [game.NumGerman]int
}

// NewEternalScreen creates the eternal table.
func NewEternalScreen(ss *Session) Scene {
	e := &EternalScreen{ss: ss}
	s := ss.S
	for i := range e.ids {
		e.ids[i] = i
	}
	// The original's exchange sort: not stable for equal points.
	for i := 0; i < len(e.ids)-1; i++ {
		for j := i + 1; j < len(e.ids); j++ {
			if uint32(s.Clubs[e.ids[j]].Eternal) > uint32(s.Clubs[e.ids[i]].Eternal) {
				e.ids[i], e.ids[j] = e.ids[j], e.ids[i]
			}
		}
	}
	return e
}

func (e *EternalScreen) Update(app *App) error {
	if MenuButtonHit(&app.Input) {
		app.Pop()
	}
	return nil
}

var (
	eternalTitles = [5]string{"MEISTERSCHAFTEN", "DFB-POKALE", "LANDESMEISTER-POKALE", "POKALSIEGER-POKALE", "UEFA-POKALE"} // D:0x4a6c
	eternalRows   = [5]string{"PUN.", "TORE", "SIEGE", "NIED.", "UNEN."}                                                    // D:0x4a80
)

// trophyRim is the circle of radius 11 of f_3930_0c90 around a trophy.
var trophyRim = []string{
	"..........#######..........",
	"........##.......##........",
	".......#...........#.......",
	".....##.............##.....",
	".....#...............#.....",
	"....#.................#....",
	"...#...................#...",
	"...#...................#...",
	"..#.....................#..",
	"..#.....................#..",
	"..#.....................#..",
	"..#.....................#..",
	"..#.....................#..",
	"..#.....................#..",
	"..#.....................#..",
	"...#...................#...",
	"...#...................#...",
	"....#.................#....",
	".....#...............#.....",
	".....##.............##.....",
	".......#...........#.......",
	"........##.......##........",
	"..........#######..........",
}

func (g *Gfx) drawTrophyRim(dst *ebiten.Image, cx, cy, c int) {
	for dy, row := range trophyRim {
		for dx, ch := range row {
			if ch == '#' {
				FillRect(dst, cx+dx-13, cy+dy-11, 1, 1, g.PalColor(c))
			}
		}
	}
}

func (e *EternalScreen) Draw(app *App, dst *ebiten.Image) {
	g := app.Gfx
	s := e.ss.S
	m := e.ss.M()
	g.Background(dst)
	g.Window(dst, 5, 8, 5+0x87, 8+0xde, ColBlue)
	g.Window(dst, 0x91, 8, 0x91+0xa8, 8+0xb4, ColBlue)
	g.DrawMenuButton(dst)
	g.TextMid(dst, FontNormal, 5, 0x8c, 0x11, "Ewige Tabelle", ColBeige, -1)
	lineIncl(dst, 6, 0x13, 0x8b, 0x13, g.PalColor(ColBeige))
	g.TextMid(dst, FontNormal, 0x91, 0x139, 0x11, "Ewige Bilanz", ColBeige, -1)
	lineIncl(dst, 0x92, 0x13, 0x138, 0x13, g.PalColor(ColBeige))
	trophies := g.Sub(36, image.Rect(0, 0, 0x15, 0x71))
	Draw(dst, trophies, 0x9b, 0x16)
	Draw(dst, trophies, 0x119, 0x16)

	for i := 0; i < 5; i++ {
		g.TextMid(dst, FontSmall, 0xb0, 0x11e, i*0x17+0x1e, eternalTitles[i], ColLight, ColBlack)
		txt, c := " KOMMT VIELLEICHT NOCH...", ColLight
		if m.Titles[i] != 0 {
			txt, c = numText(int64(m.Titles[i]), 1, false), ColBeige
		}
		g.TextMid(dst, FontSmall, 0xb0, 0x119, i*0x17+0x26, txt, c, ColBlack)
		g.TextAt(dst, FontSmall, 0x94, i*7+0x9c, eternalRows[i], ColLight, -1)
	}
	g.TextAt(dst, FontSmall, 0xb7, 0x93, "GESAMT           HEIM         AUSWÄRTS", ColLight, -1)
	for i := 0; i < 5; i++ {
		var v, minus *[2]int
		switch i {
		case 0:
			v, minus = &m.EtPoints, &m.EtMinus
		case 1:
			v, minus = &m.EtGoals, &m.EtAgainst
		case 2:
			v = &m.EtWins
		case 3:
			v = &m.EtLosses
		default:
			v = &m.EtDraws
		}
		for c := 0; c < 3; c++ {
			pick := func(p *[2]int) int {
				switch c {
				case 0:
					return p[0] + p[1]
				case 1:
					return p[0]
				}
				return p[1]
			}
			txt := numText(int64(uint16(pick(v))), 4, false)
			x := 0x30*c + 0xb3
			if minus != nil {
				txt += ":" + numText(int64(uint16(pick(minus))), 4, false)
				x = 0x30*c + 0xa9
			}
			g.TextAt(dst, FontSmall, x, i*7+0x9c, txt, ColLight, -1)
		}
	}

	// the eternal table: the first 23 clubs plus the clubs of the managers
	y, extra := 0x1b, 0
	for i, id := range e.ids {
		c, mine := ColLight, false
		for _, o := range s.Managers {
			if o.Team == id {
				mine = true
				if i < 0x17 {
					extra++
				}
				c = ColBeige
				break
			}
		}
		if i >= extra+0x17 && !mine {
			continue
		}
		g.TextAt(dst, FontSmall, 9, y, numText(int64(i+1), 2, false)+". "+s.Teams[id].Name, c, -1)
		g.TextAt(dst, FontSmall, 0x6f, y, numText(s.Clubs[id].Eternal%100000, 4, false), c, -1)
		y += 7
		if extra-i == -0x16 && len(s.Managers) != extra {
			y += 0xe
			for dy := y - 0x13; dy < y-7; dy += 2 {
				FillRect(dst, 0xd, dy, 1, 1, g.PalColor(ColLight))
			}
		}
	}

	// the rims: green while the title is still possible
	for i := 0; i < 5; i++ {
		c := ColRed
		if (i == 0 && m.GoalFlags&1 == 0 && m.League == 0) || (i != 0 && m.CupStatus[i-1] == s.Cups.Round[i-1]) {
			c = 13
		}
		g.drawTrophyRim(dst, 0xa5, i*0x17+0x20, c)
	}
}
