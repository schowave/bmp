package ui

import (
	"strconv"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/schowave/bmp/remake/internal/game"
)

// BankScreen takes credits (f_112a_167f): the rates of the bank by
// duration, the credits of the chosen lender in the list at the bottom
// left (the middle of its scroll bar switches to the sums of that lender),
// the account at the bottom right. The icons at the right choose the lender:
// the other human managers and the bank. Clicking a row of the rates asks
// for the amount (and from a manager for duration and rate) in the status
// line; Enter confirms, Escape cancels.
type BankScreen struct {
	ss      *Session
	lenders []int // manager indexes and game.LenderBank, top to bottom
	lender  int
	hover   int // row of the rates under the pointer, -1 none
	status  string
	sums    bool // the list shows "SCHULDEN HIER" (toggled by the middle of the bar)
	sumFlag bool

	input  *deskKeyInput
	prompt string
	step   int // 1 amount, 2 months, 3 rate
	row    int
	amount int64
	months int
}

// NewBankScreen creates the bank screen.
func NewBankScreen(ss *Session) *BankScreen {
	b := &BankScreen{ss: ss, hover: -1, lender: game.LenderBank}
	for i := range ss.S.Managers {
		if i != ss.Mgr() {
			b.lenders = append(b.lenders, i)
		}
	}
	b.lenders = append(b.lenders, game.LenderBank)
	return b
}

func (b *BankScreen) lenderName() string {
	if b.lender == game.LenderBank {
		return "der Bank"
	}
	return b.ss.S.Managers[b.lender].Name
}

func (b *BankScreen) Update(app *App) error {
	in := &app.Input
	if b.input != nil {
		done, ok := b.input.update(in)
		if !done {
			return nil
		}
		v := b.input.value()
		b.input = nil
		if !ok || v == 0 {
			b.endInput()
			return nil
		}
		b.next(app, v)
		return nil
	}
	s := b.ss.S
	// lender icons
	if in.Left {
		for i, l := range b.lenders {
			if in.In(0x10c, 0x2d*i+0xf, ButtonW, ButtonH) && l != b.lender {
				b.lender = l
				b.sums = false
				return nil
			}
		}
	}
	// the middle of the scroll bar
	if in.Left && in.X > 0x83 && in.X < 0x92 && in.Y > 0x84 && in.Y < 0x9f {
		b.sumFlag = !b.sumFlag
		b.sums = b.sumFlag
		return nil
	}
	row := -1
	if in.X > 0xa0 && in.X < 0x103 && in.Y > 0x1e && in.Y < 0x5d {
		row = (in.Y - 0x1f) / 11
	}
	switch {
	case row >= 0 && row != b.hover:
		if b.lender == game.LenderBank {
			b.status = "Kredit zu " + numText(int64(s.BankRates[row]), 1, false) + "% aufnehmen."
		} else if b.hover == -1 {
			b.status = "Kredit von " + s.Managers[b.lender].Name + " aufnehmen."
		}
		b.hover = row
	case row < 0 && b.hover != -1:
		b.status = ""
		b.hover = -1
	}
	if in.Left && b.hover >= 0 {
		b.row = b.hover
		b.ask(1)
		return nil
	}
	if deskMenuHit(in) {
		app.Pop()
	}
	return nil
}

// bankPrompts are the status lines 24..26 of f_0000_243f with the x of
// their input fields.
var bankPrompts = [4]struct {
	text   string
	x, max int
}{{}, {"Wie hoch soll der Kredit sein ?", 0xba, 7}, {"Laufzeit des Kredites (Mon.) ?", 0xd2, 2}, {"Prozentsatz des Kredites ?", 0xaf, 2}}

func (b *BankScreen) ask(step int) {
	b.step = step
	p := bankPrompts[step]
	b.prompt = p.text
	b.input = &deskKeyInput{x: p.x, y: 0xe1, max: p.max}
}

func (b *BankScreen) endInput() {
	b.step = 0
	b.prompt = ""
	b.status = ""
	b.hover = -1
}

// next handles an entered value.
func (b *BankScreen) next(app *App, v int64) {
	s := b.ss.S
	mi := b.ss.Mgr()
	switch b.step {
	case 1:
		b.amount = v
		if err := s.CheckCreditAmount(mi, b.lender, v); err != nil {
			app.Push(b.refusal(err))
			b.endInput()
			return
		}
		if b.lender == game.LenderBank {
			b.take(app, (b.row+1)*3, s.BankRates[b.row])
			return
		}
		b.ask(2)
	case 2:
		if v > 24 {
			app.Push(NewDeskToast("Maximal", "24 Monate."))
			b.endInput()
			return
		}
		b.months = int(v)
		b.ask(3)
	case 3:
		switch {
		case v > 13:
			m1 := newDeskMsg4("Wegen Wucher werden Sie", "mit einer", "Disziplinarstrafe von", "von 1.000.000 DM belegt.")
			m1.Done = func(app *App) {
				app.Push(newDeskMsg4("Okay...", "War nur ein Scherz...", "Das war aber trotzdem", "kein fairer Zug..."))
			}
			app.Push(m1)
			b.endInput()
		case v < 3:
			app.Push(newDeskMsg4("Mindest-Prozentsatz:", "lächerliche", "minimale", "3 Prozent."))
			b.endInput()
		default:
			b.take(app, b.months, int(v))
		}
	}
}

// take places the credit.
func (b *BankScreen) take(app *App, months, rate int) {
	s := b.ss.S
	mi := b.ss.Mgr()
	var err error
	if b.lender == game.LenderBank {
		err = s.TakeBankCredit(mi, b.amount, b.row)
	} else {
		err = s.TakeManagerCredit(mi, b.lender, b.amount, months, rate)
	}
	if err == game.ErrThreeCredits {
		app.Push(NewDeskToast("Schon 3 Kredite", "aufgenommen."))
	}
	if err == nil {
		b.sums = false
	}
	b.endInput()
}

// refusal is the message box of a refused amount.
func (b *BankScreen) refusal(err error) *deskMsg4 {
	switch err {
	case game.ErrLenderMoney:
		return newDeskMsg4("So viel Geld", "besitzt", b.ss.S.Managers[b.lender].Name, "gar nicht...")
	case game.ErrLenderLimit:
		return newDeskMsg4("Sie können nur", "maximal 1 Mio. DM", "Kredit von einem", "Mitspieler aufnehmen.")
	}
	// the typo "maxmimal" is the original's
	return newDeskMsg4("Sie können nur", "maxmimal 4 Mio. DM", "Kredit von der", "Bank aufnehmen.")
}

func (b *BankScreen) Draw(app *App, dst *ebiten.Image) {
	g := app.Gfx
	s := b.ss.S
	m := b.ss.M()
	g.Background(dst)
	for i := 0; i < 2; i++ {
		for j := 0; j < 2; j++ {
			x, y := i*0x90+6, j*0x59+0xf
			g.Window(dst, x, y, x+0x8b-0x1e*i, y+0x52, ColGrey7)
		}
	}
	Draw(dst, g.Sub(7, rectWH(0, 0, 0x8a, 0x51)), 7, 0x10)
	Draw(dst, g.Sub(7, rectWH(0x8b, 0, 0xe, 0x51)), 0x83, 0x69)
	for i, l := range b.lenders {
		y := 0x2d*i + 0xf
		g.drawIconButton(dst, IconDesk, 0x10c, y, l == b.lender)
		if l == game.LenderBank {
			Draw(dst, g.Sub(7, rectWH(0x9b, 0, 0x20, 0x17)), 0x115, y+9)
		} else {
			f := s.Managers[l].Face
			Draw(dst, g.Sub(0, rectWH(f*41+5, 8, 0x20, 0x17)), 0x115, y+9)
		}
	}
	g.DrawMenuButton(dst)

	// the credits of the lender
	line6(dst, 7, 0x84, 0x82, 0x84, g.C(ColGrey6))
	line6(dst, 7, 0x9f, 0x82, 0x9f, g.C(ColGrey6))
	if b.sums {
		g.TextAt(dst, FontSmall, 9, 0x8f, "SCHULDEN HIER: "+numText(s.CreditSum(b.ss.Mgr(), b.lender, false), 1, false)+" DM", ColLight, -1)
		g.TextAt(dst, FontSmall, 9, 0x98, "ZINSEN HIER: "+numText(s.CreditSum(b.ss.Mgr(), b.lender, true), 1, false)+" DM", ColLight, -1)
	} else {
		n := 0
		for _, c := range m.Credits[b.lender] {
			if c.Amount == 0 {
				continue
			}
			y := (n + 4) * 0x1c
			if n == 1 {
				y--
			}
			n++
			l1 := numText(int64(c.StartDay), 1, false) + "." + numText(int64(c.StartMonth+1), 1, false) + "." +
				numText(int64(c.StartYear), 1, true) + ": " + numText(c.Amount, 1, false) + " DM (" + numText(int64(c.Rate), 1, false) + "%)"
			l2 := "FÄLLIG AM " + numText(int64(c.DueDay), 1, false) + "." + numText(int64(c.DueMonth+1), 1, false) + "." +
				numText(int64(c.DueYear), 1, true)
			g.TextAt(dst, FontSmall, 9, y, l1, ColLight, -1)
			g.TextAt(dst, FontSmall, 9, y+8, l2, ColLight, -1)
			g.TextAt(dst, FontSmall, 9, y+0x10, "ZINSEN PRO MONAT: "+numText(c.Interest, 1, false)+" DM", ColLight, -1)
		}
	}

	// the rates
	g.TextAt(dst, FontNormal, 0xa0, 0x18, "Dauer  Zinssatz", ColLight, -1)
	for i := 0; i < 6; i++ {
		y := i*0xb + 0x25
		x := 0xa0
		if i > 2 {
			x -= 3
		}
		g.TextAt(dst, FontNormal, x, y, numText(int64((i+1)*3), 2, false)+" Mon.", ColGrey2, -1)
		g.TextAt(dst, FontNormal, 0xdf, y, numText(int64(s.BankRates[i]), 2, false)+"%", ColGrey2, -1)
		if i < 5 {
			line6(dst, 0x97, i*0xb+0x27, 0x102, i*0xb+0x27, g.C(ColGrey6))
		}
	}

	// the account
	line6(dst, 0x97, 0x84, 0x102, 0x84, g.C(ColGrey6))
	line6(dst, 0x97, 0x9f, 0x102, 0x9f, g.C(ColGrey6))
	mid := func(y int, t string) { g.deskMid(dst, FontNormal, 0x97, 0x103, y, t, ColLight, ColBlack) }
	mid(0x72, "Kontostand:")
	mid(0x7e, numText(m.Money, 1, false)+" DM")
	mid(0x8d, "Gesamtschulden:")
	mid(0x99, numText(m.Debts(), 1, false)+" DM")
	mid(0xa8, "Zinsen/Monat:")
	mid(0xb4, numText(m.InterestPerMonth(), 1, false)+" DM")

	if b.prompt != "" {
		g.deskStatus(dst, b.prompt, true)
	} else {
		g.deskStatus(dst, b.status, false)
	}
	if b.input != nil {
		b.input.draw(g, dst, app.frame)
	}
}

// deskKeyInput is the number input of f_076b_07d7 in the status line:
// digits in beige with a small block cursor, Enter ends, Escape gives 0.
type deskKeyInput struct {
	x, y int
	max  int
	text string
}

func (k *deskKeyInput) update(in *Input) (done, ok bool) {
	for _, c := range in.Chars {
		if c >= '0' && c <= '9' && len(k.text) < k.max {
			k.text += string(c)
		}
	}
	if in.Backspace && k.text != "" {
		k.text = k.text[:len(k.text)-1]
	}
	switch {
	case in.Escape:
		k.text = ""
		return true, false
	case in.Enter:
		return true, true
	}
	return false, false
}

func (k *deskKeyInput) value() int64 {
	v, _ := strconv.ParseInt(k.text, 10, 64)
	return v
}

func (k *deskKeyInput) draw(g *Gfx, dst *ebiten.Image, frame int) {
	g.TextAt(dst, FontNormal, k.x, k.y, k.text, ColBeige, -1)
	x := k.x + g.TextWidth(FontNormal, k.text)
	fill6(dst, x+2, k.y-4, x+4, k.y, g.C(ColBeige))
}

// deskMsg4 is the message box f_3091_0e3a: four lines on dark red and an
// OKAY button.
type deskMsg4 struct {
	lines [4]string
	Done  func(app *App)
}

func newDeskMsg4(a, b, c, d string) *deskMsg4 { return &deskMsg4{lines: [4]string{a, b, c, d}} }

func (d *deskMsg4) Update(app *App) error {
	in := &app.Input
	if (in.Left && in.X > 0xb2 && in.X < 0xb2+0x39 && in.Y > 0xa0 && in.Y < 0xa0+0xc) || in.Enter || in.Escape {
		app.Pop()
		if d.Done != nil {
			d.Done(app)
		}
	}
	return nil
}

func (d *deskMsg4) Draw(app *App, dst *ebiten.Image) {
	g := app.Gfx
	g.Window(dst, 0x3c, 0x6e, 0x3c+0xb9, 0x6e+0x41, ColDarkRed)
	for i, l := range d.lines {
		g.deskMid(dst, FontNormal, 0x3c, 0xf5, 0x7b+i*10, l, ColSand, -1)
	}
	g.deskButton(dst, 0xb2, 0xa0, "OKAY", 0x1a)
}
