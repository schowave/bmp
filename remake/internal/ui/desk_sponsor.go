package ui

import (
	"image"
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/schowave/bmp/remake/internal/game"
)

// SponsorScreen is the advertising screen (f_262f_2be8, the full-screen
// picture PIC/46.CP): the six boards at the top and the shirt of the player
// at the left can be chosen; the black panel then shows the contract. A
// click on the logo box browses the offers of the sponsors with the arrows,
// "OK" accepts one, "NEIN" ends browsing. The buttons at the Werbeausgaben
// change them by 2500 DM. The right button leaves the screen.
type SponsorScreen struct {
	ss     *Session
	sel    int // -1 nothing, 0..5 a board, 10 the shirt
	browse bool
	k      int // the sponsor shown while browsing
	logo   int // logo in the box: -2 the picture's, -1 brown, else a sponsor
	panel  [2]string
	pLeft  bool // the panel shows "KEIN INTERESSE..." in one line
	shown  bool // the panel and the buttons have been drawn over the picture
	press  int  // pressed sprite: 1 OK, 2 NEIN, 3 up, 4 down
	pressX int
	btns   int // buttons: 0 the picture's, 1 blank, 2 OK/NEIN, 3 greyed (no offer)
	// arrows: the arrow boxes were wiped with the blank sprites when
	// browsing ended; they stay so.
	arrows bool
}

// NewSponsorScreen creates the advertising screen.
func NewSponsorScreen(ss *Session) *SponsorScreen {
	return &SponsorScreen{ss: ss, sel: -1, logo: -2}
}

// sponsorType is the offer type of the selection: 0 shirt, 1 boards.
func (sp *SponsorScreen) sponsorType() int {
	if sp.sel == 10 {
		return 0
	}
	return 1
}

func (sp *SponsorScreen) Update(app *App) error {
	in := &app.Input
	s := sp.ss.S
	mi := sp.ss.Mgr()
	m := sp.ss.M()
	if sp.press != 0 {
		if in.LeftHeld {
			return nil
		}
		p := sp.press
		sp.press = 0
		sp.release(app, p)
		return nil
	}
	if in.Right || in.Escape {
		app.Pop()
		return nil
	}
	if !in.Left {
		return nil
	}
	x, y := in.X, in.Y
	t := sp.sponsorType()
	// OK / NEIN
	if sp.browse && x > 0xd9 && x < 0x139 && y > 0x5a && y < 0x6d && m.Offers[t][sp.k] != 0 {
		sp.press, sp.pressX = 1, x
		if x > 0x108 {
			sp.press = 2
		}
		return nil
	}
	// Werbeausgaben
	if x > 0xd8 && x < 0x135 && y > 0xb9 && y < 0xc5 {
		s.ChangeAdSpending(mi, x >= 0x107)
		return nil
	}
	// a board or the shirt
	sel := -1
	switch {
	case x > 3 && x < 0x64 && y > 0x50 && y < 0xd2 && sp.sel != 10:
		sel = 10
	case x > 3 && x < 0x13c && y > 0x12 && y < 0x33:
		sel = (x - 4) / 0x34
		if y >= 0x50 {
			sel = 10
		}
	}
	if sel >= 0 {
		if sel != sp.sel {
			sp.choose(sel)
		}
		return nil
	}
	// the logo box starts browsing
	if sp.sel >= 0 && !sp.browse && x > 0x7e && x < 0xa0 && y > 0x43 && y < 0x6c {
		sp.browse = true
		sp.k = m.BoardIndex[min(sp.sel, 5)]
		if sp.sel == 10 {
			sp.k = m.TrikotIndex
		}
		sp.offer()
		return nil
	}
	// the arrows
	if sp.browse && x > 0xb4 && x < 0xd3 && y > 0x3b && y < 0x6c {
		sp.press = 3
		if y > 0x52 {
			sp.press = 4
		}
		return nil
	}
	return nil
}

// choose selects a board or the shirt and shows its contract.
func (sp *SponsorScreen) choose(sel int) {
	m := sp.ss.M()
	sp.sel = sel
	sp.browse = false
	sp.shown = true
	sp.btns = 1
	sp.logo = -1
	months, amount := m.TrikotMonths, m.Ads[0]
	if sel != 10 {
		months, amount = m.BoardMonths[sel], m.Ads[1+sel]
	}
	sp.panel = sponsorText(months, amount, false)
	sp.pLeft = false
}

// offer shows the offer of sponsor k (f_262f_281f) and its logo.
func (sp *SponsorScreen) offer() {
	m := sp.ss.M()
	t := sp.sponsorType()
	sp.logo = sp.k
	sp.shown = true
	if m.Offers[t][sp.k] == 0 {
		sp.btns = 3
		sp.panel = [2]string{"KEIN INTERESSE...", ""}
		sp.pLeft = true
		return
	}
	sp.btns = 2
	sp.panel = sponsorText(m.OfferYears[t][sp.k], m.Offers[t][sp.k], true)
	sp.pLeft = false
}

// release finishes a pressed button.
func (sp *SponsorScreen) release(app *App, p int) {
	s := sp.ss.S
	mi := sp.ss.Mgr()
	switch p {
	case 3:
		if sp.k < 9 {
			sp.k++
		}
		sp.offer()
	case 4:
		if sp.k > 0 {
			sp.k--
		}
		sp.offer()
	case 1:
		var err error
		if sp.sel == 10 {
			err = s.AcceptTrikot(mi, sp.k)
		} else {
			err = s.AcceptBoard(mi, sp.sel, sp.k)
		}
		if err != nil {
			t := NewDeskToast("Sie stehen noch", "unter Vertrag")
			t.Pal = 1
			app.Push(t)
			return
		}
		sp.endBrowse()
	case 2:
		sp.endBrowse()
	}
}

func (sp *SponsorScreen) endBrowse() {
	sp.browse = false
	sp.arrows = true
	sp.btns = 1
}

// sponsorText is f_262f_2551 for the black panel: the contract line and
// the amount padded to ten characters.
func sponsorText(n int, amount int64, years bool) [2]string {
	l1 := "KüNDBAR" // K}NDBAR: the original writes a small ü
	if n != 0 {
		l1 = "VERTRAG: " + numText(int64(n), 1, false)
		if years {
			l1 += "JAHR"
		} else {
			l1 += " MONAT"
		}
		if n > 1 {
			l1 += "E"
		}
	}
	return [2]string{l1, sponsorAmount(amount)}
}

func sponsorAmount(v int64) string {
	if v < 0 {
		v = -v
	}
	return numText(v, 10, false) + " DM"
}

func (sp *SponsorScreen) Draw(app *App, dst *ebiten.Image) {
	g := app.Gfx
	m := sp.ss.M()
	// The screen shows the picture with palette 1 (f_3930_0624), the
	// sprites and texts on it too.
	Draw(dst, g.PicPal(46, true, 1, -1), 0, 0)
	sheet := g.PicPal(31, false, 1, -1)
	logos := g.PicPal(32, false, 1, -1)
	c1 := func(i int) color.RGBA { return g.Palette(1)[i] }
	at := func(x, y int, t string, c int) {
		g.Text(dst, FontSmall, x, y-g.baseline(FontSmall), t, c1(c))
	}
	mid := func(y int, t string) {
		w := g.TextWidth(FontSmall, t)
		at(0xd9+(0x139-0xd9)/2-w/2, y, t, ColLight)
	}
	sub := func(img *ebiten.Image, x, y, w, h int) *ebiten.Image {
		return img.SubImage(image.Rect(x, y, x+w, y+h)).(*ebiten.Image)
	}

	// shirt and boards (f_262f_29a8)
	if m.TrikotMonths != 0 {
		k, sy := m.TrikotIndex, 0x43
		if k > 4 {
			k, sy = k-5, 0x55
		}
		Draw(dst, sub(logos, 28*k, sy, 0x1a, 0x10), 0x33, 0x6e)
	}
	for b := 0; b < 6; b++ {
		x := 0x34*b + 8
		if m.BoardMonths[b] != 0 {
			Draw(dst, sponsorLogo(logos, m.BoardIndex[b]), x, 0x13)
		} else {
			fill6(dst, x, 0x13, 0x34*b+0x2e, 0x31, c1(9))
		}
	}
	value := func(y int, v int64) {
		fill6(dst, 0xf0, y, 0x134, y+0xa, c1(ColBlack))
		x := 0xf0
		if v < 0 {
			at(x, y+7, "-", ColLight)
		}
		at(x, y+7, sponsorAmount(v), ColLight)
	}
	var boards int64
	for b := 1; b <= 6; b++ {
		boards += m.Ads[b]
	}
	value(0x80, m.Ads[0])
	value(0x8e, boards)
	value(0x9c, m.Ads[7])
	value(0xb9, m.Ads[8])
	value(0xd2, m.Ads[0]+boards+m.Ads[7]-m.Ads[8])

	// the selection frame (f_262f_23e4)
	if sp.sel == 10 {
		rect6(dst, 5, 0x3b, 0x6e, 0xe5, c1(ColWhite))
	} else if sp.sel >= 0 {
		x0, x1 := 0x34*sp.sel+5, 0x34*sp.sel+0x36
		line6(dst, x0, 0x32, x1, 0x32, c1(ColWhite))
		line6(dst, x1, 0x13, x1, 0x32, c1(ColWhite))
	}

	// the logo box, the arrows and the panel
	switch {
	case sp.logo == -1:
		fill6(dst, 0x7f, 0x43, 0xa4, 0x60, c1(9))
	case sp.logo >= 0:
		Draw(dst, sponsorLogo(logos, sp.logo), 0x7f, 0x43)
	}
	if sp.browse {
		up, down := 0, 0
		switch sp.press {
		case 3:
			up = 0x1e
		case 4:
			down = 0x1e
		}
		Draw(dst, sub(sheet, up, 0, 0x1e, 0x17), 0xb5, 0x3c)
		Draw(dst, sub(sheet, down, 0x17, 0x1e, 0x17), 0xb5, 0x54)
	} else if sp.arrows {
		Draw(dst, sub(sheet, 0x3c, 0, 0x1e, 0x17), 0xb5, 0x3c)
		Draw(dst, sub(sheet, 0x3c, 0x17, 0x1e, 0x17), 0xb5, 0x54)
	}
	if sp.shown {
		fill6(dst, 0xd9, 0x3c, 0x139, 0x5a, c1(ColBlack))
		if sp.pLeft {
			mid(0x4e, sp.panel[0])
		} else {
			mid(0x3e+7, sp.panel[0])
			mid(0x3e+0xc+7, sp.panel[1])
		}
		ok, nein := 0x60, 0x60
		if sp.btns == 2 {
			ok, nein = 0, 0
		}
		if sp.btns != 0 {
			Draw(dst, sub(sheet, ok, 0x2f, 0x30, 0x10), 0xd9, 0x5b)
			Draw(dst, sub(sheet, nein, 0x3f, 0x30, 0x10), 0x109, 0x5b)
		}
		switch sp.press {
		case 1:
			Draw(dst, sub(sheet, 0x30, 0x3f, 0x30, 0x10), 0xd9, 0x5b)
		case 2:
			Draw(dst, sub(sheet, 0x30, 0x2f, 0x30, 0x10), 0x10a, 0x5b)
		}
	}
}

// sponsorLogo is f_262f_248a: logo k of PIC/32.VGA, 38x30.
func sponsorLogo(sheet *ebiten.Image, k int) *ebiten.Image {
	sy := 0
	if k > 4 {
		k, sy = k-5, 0x21
	}
	return sheet.SubImage(image.Rect(0x27*k, sy+1, 0x27*k+0x26, sy+1+0x1e)).(*ebiten.Image)
}

var _ = game.NumGerman
