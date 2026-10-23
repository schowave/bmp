package ui

import (
	"image"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/schowave/bmp/remake/internal/game"
)

// StadiumScreen shows and extends the stadium (f_0000_0602): the picture,
// a panel for the orders at the top right and the table "MOMENTAN / NACH
// AUSBAU". Pointing at a row names it in the status line; clicking it opens
// the order in the panel: a number field for places and the ticket price,
// a list of levels for the other items, then the question "NA KLAR !" /
// "ACH NEE...". The right button ends a number input or leaves a list.
type StadiumScreen struct {
	ss     *Session
	hover  int // row of the status line, -1 none
	mode   int
	row    int // the row (1..8) of the order
	value  int64
	num    deskNumber
	list   int // highlighted level, -1 none
	lim    game.BuildLimit
	order  game.BuildOrder
	listN  int // last row index of the level list
	pressd bool
}

const (
	stIdle = iota
	stNumber
	stLevels
	stConfirm
)

// NewStadiumScreen creates the stadium screen.
func NewStadiumScreen(ss *Session) *StadiumScreen {
	return &StadiumScreen{ss: ss, hover: -1, list: -1}
}

var (
	// stadiumStatus are D:0x2818: the status line of the rows.
	stadiumStatus = [9]string{"", "SITZPLÄTZE AUSBAUEN", "STEHPLÄTZE AUSBAUEN", "PLÄTZE ÜBERDACHEN",
		"FLUTLICHTMASTEN AUSBAUEN", "ANZEIGETAFEL ERRICHTEN", "STADION RENOVIEREN", "KOMFORT VERBESSERN",
		"EINTRITTSPREIS ÄNDERN"}
	// stadiumLabels are D:0x2868: the rows of the table.
	stadiumLabels = [9]string{"GESAMTKAPAZITÄT", "SITZPLÄTZE", "STEHPLÄTZE", "ÜBERDACHTE PLÄTZE",
		"FLUTLICHT", "ANZEIGETAFEL", "ZUSTAND", "KOMFORT", "EINTRITTSPREIS"}
	stadiumSizes  = [4]string{"IS NICH", "KLEIN", "MITTEL", "GROß"}                                              // D:0x2858
	stadiumGrades = [7]string{"", "UNGENUEGEND", "MANGELHAFT", "AUSREICHEND", "BEFRIEDIGEND", "GUT", "SEHR GUT"} // D:0x283c
)

// stadiumLevel is the text of level v of item k (3..6).
func stadiumLevel(k int, v int64) string {
	if k >= game.StCondition {
		return stadiumGrades[clampInt(int(v), 0, 6)]
	}
	return stadiumSizes[clampInt(int(v), 0, 3)]
}

func clampInt(v, lo, hi int) int { return max(lo, min(hi, v)) }

// deskStatus draws the status line f_0000_243f(k) of kind k with a text:
// centred under the screen, left at x=2 for the prompts of the bank.
func (g *Gfx) deskStatus(dst *ebiten.Image, s string, left bool) {
	if s == "" {
		return
	}
	if left {
		g.TextAt(dst, FontNormal, 2, 0xe1, s, ColLight, -1)
		return
	}
	w := g.TextWidth(FontNormal, s)
	g.TextAt(dst, FontNormal, 0x85-w/2, 0xe1, s, ColLight, ColBlack)
}

func (st *StadiumScreen) Update(app *App) error {
	in := &app.Input
	s := st.ss.S
	mi := st.ss.Mgr()
	m := st.ss.M()
	switch st.mode {
	case stNumber:
		if in.Right {
			st.mode = stIdle
			if st.num.value == st.num.start {
				return nil
			}
			v := st.num.value
			if st.row == 8 {
				s.SetTicketPrice(mi, int(v))
				return nil
			}
			st.confirm(app, v)
			return nil
		}
		st.num.update(app)
		return nil
	case stLevels:
		st.list = -1
		if in.X > 0x96 && in.X < 0x120 && in.Y > 0x19 && in.Y < 0x42 {
			if r := (in.Y - 0x1c) / 7; r <= st.listN {
				st.list = r
			}
		}
		if in.Right {
			st.mode = stIdle
			return nil
		}
		if in.Left && st.list >= 0 {
			l := int64(st.list + 1)
			if st.row > 5 {
				l++
			}
			if l <= st.lim.Max && l > st.lim.Now {
				st.mode = stIdle
				st.confirm(app, l)
			}
		}
		return nil
	case stConfirm:
		if !in.Left {
			return nil
		}
		switch {
		case in.X > 0x9e && in.X < 0x9e+0x39 && in.Y > 0x51 && in.Y < 0x51+0xc:
			st.mode = stIdle
			if err := s.ConfirmBuild(mi, st.order); err != nil {
				app.Push(NewDeskToast("Zu wenig", "Kleingeld vorhanden..."))
			}
		case in.X > 0xe3 && in.X < 0xe3+0x39 && in.Y > 0x51 && in.Y < 0x51+0xc:
			st.mode = stIdle
			s.DeclineBuild(st.order.Item)
		}
		return nil
	}

	st.hover = -1
	if in.X > 6 && in.X < 0x121 && in.Y > 0x72 && in.Y < 0xb9 {
		st.hover = (in.Y - 0x72) / 8
	}
	if in.Left && st.hover > 0 {
		st.open(app, st.hover)
		return nil
	}
	if deskMenuHit(in) {
		app.Pop()
	}
	_ = m
	return nil
}

// open starts the order of row p (1..8) like f_0000_0602.
func (st *StadiumScreen) open(app *App, p int) {
	s := st.ss.S
	mi := st.ss.Mgr()
	m := st.ss.M()
	st.row = p
	if p == 8 {
		st.mode = stNumber
		st.num = deskNumber{x: 0xb8, y: 0x1b, value: int64(m.TicketPrice), start: int64(m.TicketPrice), min: 5, max: 25, step: 1}
		return
	}
	k := p - 1
	st.lim = s.StadiumLimit(mi, k)
	switch {
	case st.lim.Locked:
		st.row = 0
		app.Push(NewDeskToast("Im Moment keine", "Baufirma aufzutreiben."))
		return
	case k <= game.StCovered && st.lim.NoMoney:
		app.Push(st.toast("Kein Geld für", "neue Plätze da..."))
		return
	case k > game.StCovered && st.lim.Now >= st.lim.Top:
		app.Push(st.toast("Was denn...?", "NOCH besser ???"))
		return
	case k > game.StCovered && st.lim.NoMoney:
		app.Push(st.toast("Da müssen Sie erst", "noch sparen..."))
		return
	}
	if k <= game.StCovered {
		st.mode = stNumber
		st.num = deskNumber{x: 0xb8, y: 0x1b, min: 0, max: st.lim.Max, step: 1000}
		return
	}
	st.mode = stLevels
	st.list = -1
	st.listN = 2
	if k >= game.StCondition {
		st.listN = 4
	}
}

// toast shows a message while the header of the order stays in the panel.
func (st *StadiumScreen) toast(a, b string) *DeskToast {
	st.mode = stHeaderOnly
	t := NewDeskToast(a, b)
	t.Done = func(app *App) { st.mode = stIdle; st.row = 0 }
	return t
}

const stHeaderOnly = 99

// confirm quotes the order and asks (f_0000_0000).
func (st *StadiumScreen) confirm(app *App, amount int64) {
	s := st.ss.S
	m := st.ss.M()
	k := st.row - 1
	cost := amount / 1000 * game.BuildCost[k]
	if k > game.StCovered {
		cost = (amount - m.Stadium[k] - m.StadiumBuild[k]) * game.BuildCost[k]
	}
	if m.Money-cost < 0 {
		app.Push(NewDeskToast("Zu wenig", "Kleingeld vorhanden..."))
		return
	}
	st.order = s.QuoteBuild(st.ss.Mgr(), k, amount)
	st.mode = stConfirm
}

func (st *StadiumScreen) Draw(app *App, dst *ebiten.Image) {
	g := app.Gfx
	m := st.ss.M()
	g.Background(dst)
	st.drawPicture(g, dst)
	g.Window(dst, 0x96, 0xf, 0x96+0x8b, 0xf+0x52, ColBlue)
	g.Window(dst, 6, 0x68, 6+0x11b, 0x68+0x52, ColBlue)
	g.DrawMenuButton(dst)
	st.drawTable(g, dst)

	if st.mode == stIdle {
		line6(dst, 0x97, 0x54, 0x120, 0x54, g.C(ColBeige))
	}
	if st.mode != stConfirm {
		g.TextAt(dst, FontSmall, 0x9a, 0x5e, "IHR KONTOSTAND:"+numText(m.Money, 1, false)+" DM", ColLight, -1)
	}
	if st.mode != stIdle && st.row > 0 {
		st.drawOrder(g, dst)
	}
	if st.hover >= 0 {
		g.deskStatus(dst, stadiumStatus[st.hover], false)
	}
}

// drawOrder draws the panel of the current order.
func (st *StadiumScreen) drawOrder(g *Gfx, dst *ebiten.Image) {
	m := st.ss.M()
	p, k := st.row, st.row-1
	g.deskMid(dst, FontSmall, 0x97, 0x120, 0x16, stadiumStatus[p], ColBeige, ColBlack)
	line6(dst, 0x97, 0x18, 0x120, 0x18, g.C(ColBeige))
	if st.mode == stConfirm {
		st.drawConfirm(g, dst)
		return
	}
	kostet := ""
	switch {
	case p < 4:
		kostet = "KOSTET " + numText(game.BuildCost[k], 1, false) + " DM PRO 1000"
	case p < 8:
		kostet = "KOSTET " + numText(game.BuildCost[k], 1, false) + " DM PRO PUNKT"
	}
	g.TextAt(dst, FontSmall, 0x9a, 0x49, kostet, ColLight, -1)
	if st.mode == stHeaderOnly && p < 4 {
		return
	}
	line6(dst, 0x97, 0x42, 0x120, 0x42, g.C(ColBeige))
	switch {
	case p < 4:
		max := st.lim.Max
		g.TextAt(dst, FontSmall, 0x9a, 0x50, "MAX. GRößE: "+numText(max, 1, false), ColLight, -1)
		g.TextAt(dst, FontSmall, 0x9a, 0x57, " KOSTEN: "+numText(max/1000*game.BuildCost[k], 1, false)+" DM.", ColLight, -1)
	case p < 8:
		if st.mode == stHeaderOnly {
			return
		}
		now, max := st.lim.Now, st.lim.Max
		first, last, y0 := 1, 3, 0x19
		if p >= 6 {
			first, last, y0 = 2, 6, 0x12
		}
		for di := first; di <= last; di++ {
			c := ColLight
			if int64(di) <= now || int64(di) > max {
				c = ColGrey3
			}
			y := 7*di + y0
			hi := st.list == di-first
			if hi {
				r := di - first
				fill6(dst, 0x97, 7*(r+4), 0x11f, 7*r+0x20, g.C(ColBlue^31))
				c ^= 31
			}
			g.TextAt(dst, FontSmall, 0x9a, y, stadiumLevel(k, int64(di)), c, -1)
			if int64(di) > now || now == 0 {
				g.TextAt(dst, FontSmall, 0xd8, y, numText((int64(di)-now)*game.BuildCost[k], 9, false)+" DM", c, -1)
			}
		}
		g.TextAt(dst, FontSmall, 0x9a, 0x50, "MAX. STATUS: "+stadiumLevel(k, max), ColLight, -1)
		g.TextAt(dst, FontSmall, 0x9a, 0x57, "(KOSTEN: "+numText((max-now)*game.BuildCost[k], 1, false)+" DM)", ColLight, -1)
	}
	if st.mode == stNumber {
		st.num.draw(g, dst)
	}
	_ = m
}

// drawConfirm draws the question of f_0000_0000.
func (st *StadiumScreen) drawConfirm(g *Gfx, dst *ebiten.Image) {
	m := st.ss.M()
	o := st.order
	k := o.Item
	var l1, l2 string
	c2 := ColGrey2
	switch {
	case k <= game.StCovered:
		l1 = "AUSBAU UM " + numText(o.Amount, 1, false) + " AUF "
		l2 = numText(o.Amount+m.Stadium[k]+m.StadiumBuild[k], 1, false) + " " + stadiumLabels[k+1]
	case k < game.StCondition:
		l1, l2, c2 = "NEUE GRößE: ", stadiumLevel(k, o.Amount), ColLight
	default:
		l1, l2, c2 = "NEUER STATUS: ", stadiumLevel(k, o.Amount), ColLight
	}
	g.deskMid(dst, FontSmall, 0x97, 0x120, 0x25, l1, ColGrey2, -1)
	g.deskMid(dst, FontSmall, 0x97, 0x120, 0x2e, l2, c2, -1)
	g.deskMid(dst, FontSmall, 0x97, 0x120, 0x37, "KOSTEN: "+numText(o.Cost, 1, false)+" DM", ColGrey2, -1)
	bau := "BAUZEIT: CIRKA"
	if o.Again {
		bau = "BAUZEIT: INSGESAMT CA."
	}
	g.deskMid(dst, FontSmall, 0x97, 0x120, 0x40, bau, ColGrey2, -1)
	g.deskMid(dst, FontSmall, 0x97, 0x120, 0x49, numText(int64(o.Days/7+1), 1, false)+" WOCHEN", ColLight, ColBlack)
	g.deskButton(dst, 0x9e, 0x51, "NA KLAR !", ColRed)
	g.deskButton(dst, 0xe3, 0x51, "ACH NEE...", ColLight)
}

// deskButton is the dialog button f_3091_123d: 57x12, black with a bevel.
func (g *Gfx) deskButton(dst *ebiten.Image, x, y int, label string, c int) {
	g.BevelButton(dst, x, y, 0x39, 0xc, ColBlack)
	g.deskMid(dst, FontSmall, x+1, x+0x39, y+8, label, c, -1)
}

// drawTable is f_0000_1a49: the table of the stadium.
func (st *StadiumScreen) drawTable(g *Gfx, dst *ebiten.Image) {
	m := st.ss.M()
	cur, pend, timer := &m.Stadium, &m.StadiumBuild, &m.BuildTimer
	g.TextAt(dst, FontSmall, 0x76, 0x6f, "MOMENTAN", ColLight, -1)
	g.TextAt(dst, FontSmall, 0xc0, 0x6f, "NACH AUSBAU", ColLight, -1)
	for r := 0; r < 9; r++ {
		y := 8*r + 0x77
		g.TextAt(dst, FontSmall, 0xa, y, stadiumLabels[r], ColGrey2, -1)
		now, after := "", ""
		switch {
		case r == 0:
			total := cur[game.StSeats] + cur[game.StStanding]
			now = numText(total, 1, false)
			if timer[game.StSeats] != 0 || timer[game.StStanding] != 0 {
				after = numText(total+pend[game.StSeats]+pend[game.StStanding], 1, false)
			}
		case r <= 3:
			now = numText(cur[r-1], 1, false)
			if timer[r-1] != 0 {
				after = numText(cur[r-1]+pend[r-1], 1, false)
			}
		case r <= 7:
			now = stadiumLevel(r-1, cur[r-1])
			if pend[r-1] != 0 {
				after = stadiumLevel(r-1, cur[r-1]+pend[r-1])
			}
		default:
			now = numText(int64(m.TicketPrice), 1, false) + " DM"
		}
		g.TextAt(dst, FontSmall, 0x76, y, now, ColGrey2, -1)
		c := ColGrey4
		if after == "" {
			after = now
		} else {
			c = ColGrey2
			if r != 0 {
				g.TextAt(dst, FontSmall, 0xf3, y, numText(int64(timer[r-1]/7+1), 2, false)+" WOCHE(N)", ColGrey2, -1)
			}
		}
		g.TextAt(dst, FontSmall, 0xc0, y, after, c, -1)
	}
}

// drawPicture is f_0000_16e3: the stadium with floodlight masts and
// scoreboard from PIC/40.VGA.
func (st *StadiumScreen) drawPicture(g *Gfx, dst *ebiten.Image) {
	m := st.ss.M()
	g.Window(dst, 6, 0xf, 6+0x8b, 0xf+0x52, ColGrey7)
	Draw(dst, g.Sub(game.StadiumPicture(m), rectWH(0, 0, 0x8a, 0x51)), 7, 0x10)
	total := m.Capacity()
	size := deskB2I(total > 14000) + deskB2I(total > 34000) + deskB2I(total > 54000)
	z := int(m.Stadium[game.StCondition]-1) >> 1
	sheet := g.PicKeyed(40, 0)
	if v := int(m.Stadium[game.StFloodlight]); v != 0 {
		w := 3 - v
		for j := 0; j < 2; j++ {
			x := 7 + int(stadiumMastX[size*2+j])
			y := 0x10 + int(stadiumMastY[size]) - 1
			sx := (w-3*z)*0x11 + 0x67
			Draw(dst, sheet.SubImage(image.Rect(sx, 1, sx+0x11, 1+int(stadiumMastH[size]))).(*ebiten.Image), x, y)
		}
	}
	if v := int(m.Stadium[game.StScoreboard]); v != 0 {
		sx, sy := 0x43-0x21*z, (v-1)*0x16+0x29
		x, y := 7+int(stadiumBoardX[size]), 0x10+int(stadiumBoardY[size])
		Draw(dst, sheet.SubImage(image.Rect(sx, sy, sx+0x1f, sy+0x14)).(*ebiten.Image), x, y)
	}
}

// Positions of the overlays per stadium size (D:0x6c..0x83).
var (
	stadiumMastH  = [4]byte{23, 23, 36, 36}
	stadiumMastX  = [8]byte{9, 111, 9, 111, 9, 115, 9, 115}
	stadiumMastY  = [4]byte{1, 0, 1, 1}
	stadiumBoardX = [4]byte{53, 53, 55, 55}
	stadiumBoardY = [4]byte{3, 1, 0, 0}
)

// deskNumber is the number field f_3091_0ba5 (PIC/21.VGA): a left click on
// the upper half raises the value by step, on the lower half lowers it.
type deskNumber struct {
	x, y                   int
	value, start, min, max int64
	step                   int64
}

func (n *deskNumber) update(app *App) {
	in := &app.Input
	if !in.Left || in.X <= n.x || in.X >= n.x+0x46 || in.Y <= n.y || in.Y >= n.y+0x27 {
		return
	}
	p, err := app.Gfx.dir.Pic(21, false)
	if err != nil {
		return
	}
	px, py := in.X-n.x, in.Y-n.y
	if px >= p.W || py >= p.H || p.Pix[py*p.W+px] != 4 {
		return
	}
	if in.Y < n.y+0x12 {
		n.value += n.step
	} else {
		n.value -= n.step
	}
	if n.value < n.min {
		n.value = n.min
	} else if n.value > n.max {
		n.value = n.max
	}
}

func (n *deskNumber) draw(g *Gfx, dst *ebiten.Image) {
	Draw(dst, g.Pic(21, false), n.x, n.y)
	g.deskMid(dst, FontNormal, n.x, n.x+0x46, n.y-2+0x18, numText(n.value, 1, false), 9, ColBlack)
}
