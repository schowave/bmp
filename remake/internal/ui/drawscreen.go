package ui

import (
	"github.com/hajimehoshi/ebiten/v2"

	"github.com/schowave/bmp/remake/internal/game"
)

// DrawShow presents a new cup draw (f_17c2_0006, f_17c2_0378,
// f_17c2_089a): the question whether to watch it, the draw on the board of
// PIC/47.VGA, then the overview of the round.
type DrawShow struct {
	ss    *Session
	cup   game.Cup
	state int // drawAsk, drawRun, drawDone
	shown int // plates on the board
	t     int // frames of the moving plate
}

const (
	drawAsk = iota
	drawRun
	drawDone
)

// drawTitles are the headlines of the draw screen (D:0x24dc).
var drawTitles = [game.NumCups]string{"DFB-POKAL", "POKAL DER LANDESMEISTER", "POKAL DER POKALSIEGER", "UEFA-POKAL"}

// Position of the board and the slots on it.
const (
	boardX, boardY = 19, 26
	plateW, plateH = 113, 7
	plateTop       = 38
	platePitch     = 10
	drawMoveFrames = 40
)

// NewDrawShow creates the presentation of the draw of cup c.
func NewDrawShow(ss *Session, c game.Cup) *DrawShow { return &DrawShow{ss: ss, cup: c, state: -1} }

func (d *DrawShow) slots() []int {
	var out []int
	for _, p := range d.ss.S.CupPairs(d.cup) {
		out = append(out, p.Home, p.Away)
	}
	return out
}

func (d *DrawShow) overview(app *App) {
	app.Replace(NewCupRound(d.ss, d.cup, 0, nil))
}

// Passing implements passer: before its first tick the show has not asked
// yet whether to watch the draw.
func (d *DrawShow) Passing() bool { return d.state == -1 }

func (d *DrawShow) Update(app *App) error {
	s := d.ss.S
	in := &app.Input
	switch d.state {
	case -1:
		on := s.Display.Europe
		if d.cup == game.CupDFB {
			on = s.Display.DFB
		}
		if !on {
			app.Pop()
			return nil
		}
		if d.cup == game.CupDFB && len(s.CupPairs(d.cup)) < 4 {
			// Semi-final and final: no question, only the overview.
			d.overview(app)
			return nil
		}
		d.state = drawAsk
		cupText := "DFB-POKAL-"
		if d.cup != game.CupDFB {
			cupText = "EUROPA-POKAL-"
		}
		app.Push(NewQuestion("An Alle:", "MöCHTEN SIE DEM UNÜBERTROFFENEM\nSCHAUSPIEL EINER "+cupText+"\nAUSLOSUNG BEIWOHNEN ?",
			"ABER KLAR", "KEIN GEDANKE", func(app *App, yes bool) {
				if yes {
					d.state = drawRun
				} else {
					app.Pop()
				}
			}))
	case drawRun:
		n := len(d.slots())
		if in.Left {
			d.shown = n
		}
		if d.shown < n {
			d.t++
			if d.t >= drawMoveFrames {
				d.t = 0
				d.shown++
			}
		}
		if d.shown >= n {
			d.state = drawDone
		}
	case drawDone:
		if in.Left || in.Right || in.Enter {
			d.overview(app)
		}
	}
	return nil
}

// plateLabel is the text of a plate: the name and, in the DFB-Pokal, the
// league in brackets.
func (d *DrawShow) plateLabel(t int) (string, int) {
	s := d.ss.S
	name := s.Teams[t].Name
	if t >= game.NumGerman {
		return name, ColBlack
	}
	l := game.LeagueOf(t)
	if d.cup == game.CupDFB {
		name += " (" + itoa(l+1) + ")"
	}
	return name, [3]int{ColBlack, ColGrey7, 26}[l]
}

func (d *DrawShow) drawPlate(g *Gfx, dst *ebiten.Image, t, x, y int) {
	label, c := d.plateLabel(t)
	FillRect(dst, x, y, plateW, plateH, g.PalColor(ColBeige))
	g.TextMid(dst, FontSmall, x, x+plateW-1, y+5, label, c, NoShadow)
}

func slotPos(i int) (int, int) {
	x := 37
	if i%2 == 1 {
		x = 167
	}
	return x, plateTop + i/2*platePitch
}

func (d *DrawShow) Draw(app *App, dst *ebiten.Image) {
	g := app.Gfx
	g.Background(dst)
	g.TextMid(dst, FontBig, 0, 0x140, 0x14, drawTitles[d.cup], ColBeige, ColBlack)
	if d.state < drawRun {
		return
	}
	Draw(dst, g.Pic(47, false), boardX, boardY)
	slots := d.slots()
	for i := 0; i < d.shown && i < len(slots); i++ {
		x, y := slotPos(i)
		d.drawPlate(g, dst, slots[i], x, y)
	}
	if d.shown < len(slots) {
		// The next plate comes out of the drum and moves to its slot.
		tx, ty := slotPos(d.shown)
		sx, sy := 160-plateW/2, 210
		f := d.t * 100 / drawMoveFrames
		d.drawPlate(g, dst, slots[d.shown], sx+(tx-sx)*f/100, sy+(ty-sy)*f/100)
	}
	if d.state == drawDone {
		g.TextMid(dst, FontSmall, 0x5a, 0xed, 0xd9, "TASTE DRÜCKEN", ColWhite, ColBlack)
	}
}
