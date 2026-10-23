package ui

import (
	"image"
	"strings"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/schowave/bmp/remake/internal/game"
)

// ShowNotices shows the notices the game queued (game.State.TakeNotices)
// one after another, each with the dialog or screen of the original, and
// then calls done (which may be nil). Notices queued while they are shown
// follow at the end.
func ShowNotices(app *App, ss *Session, done func(*App)) {
	showNoticeList(app, ss, ss.S.TakeNotices(), done)
}

func showNoticeList(app *App, ss *Session, list []game.Notice, done func(*App)) {
	if len(list) == 0 {
		if ss.S.HasNotices() {
			ShowNotices(app, ss, done)
		} else if done != nil {
			done(app)
		}
		return
	}
	n := list[0]
	showNotice(app, ss, n, func(app *App) { showNoticeList(app, ss, list[1:], done) })
}

// showNotice pushes the scenes of one notice; next runs when it is closed.
func showNotice(app *App, ss *Session, n game.Notice, next func(*App)) {
	switch n.Kind {
	case game.NoticeAlert:
		withBackdrop(app, ss, n, func(app *App, closed func(*App)) {
			a := NewAlert(n.Lines...)
			a.Done = closed
			app.Push(a)
		}, next)
	case game.NoticePopup:
		withBackdrop(app, ss, n, func(app *App, closed func(*App)) {
			app.Push(newNoticePopup(ss, n, closed))
		}, next)
	case game.NoticeCard:
		app.Push(&cardScreen{n: n, start: ss.S.StartYear, done: next})
	case game.NoticeTrophy:
		app.Push(&trophyScreen{n: n, done: next})
	case game.NoticeSeasonEnd:
		app.Push(&seasonEndScreen{ss: ss, n: n, done: next})
	default:
		next(app)
	}
}

// withBackdrop pushes the backdrop of a notice, then the notice itself
// (show); when the notice is closed the backdrop goes too and next runs.
func withBackdrop(app *App, ss *Session, n game.Notice, show func(*App, func(*App)), next func(*App)) {
	var back Scene
	switch n.Backdrop {
	case game.BackdropMenu:
		if n.Manager == ss.Mgr() {
			back = NewTurnMenu(ss)
		} else {
			back = &deskBackdrop{ss: ss, mi: n.Manager}
		}
	case game.BackdropDesk:
		back = &deskBackdrop{ss: ss, mi: n.Manager}
	}
	if back == nil {
		show(app, next)
		return
	}
	app.Push(drawOnly{back})
	show(app, func(app *App) {
		app.Pop()
		next(app)
	})
}

// drawOnly shows a scene without letting it handle input: the backdrop
// lies under the notice and never gets to the top while it is shown.
type drawOnly struct{ s Scene }

func (d drawOnly) Update(*App) error                { return nil }
func (d drawOnly) Draw(app *App, dst *ebiten.Image) { d.s.Draw(app, dst) }

// deskBackdrop is f_14a4_260a: the marble (f_08bc_3b92) with the face,
// name and logo of manager mi at the top right (f_3091_185c).
type deskBackdrop struct {
	ss *Session
	mi int
}

func (d *deskBackdrop) Update(*App) error { return nil }

func (d *deskBackdrop) Draw(app *App, dst *ebiten.Image) {
	app.Gfx.Background(dst)
	drawManagerBox(app, dst, d.ss, d.mi)
}

// drawManagerBox is f_3091_185c: the face of manager mi in a blue window
// at 265,5, his name under it and the club logo in a black window at
// 265,60.
func drawManagerBox(app *App, dst *ebiten.Image, ss *Session, mi int) {
	if mi < 0 || mi >= len(ss.S.Managers) {
		return
	}
	g := app.Gfx
	m := ss.S.Managers[mi]
	g.Window(dst, 0x109, 5, 0x109+0x2a, 5+0x2a, ColBlue)
	if m.Face >= 0 && m.Face < 4 {
		Draw(dst, g.Sub(0, image.Rect(m.Face*41, 0, m.Face*41+40, 40)), 0x10a, 6)
	}
	g.TextMid(dst, FontNormal, 0x113, 0x127, 0x39, m.Name, ColBlack, ColLight)
	g.Window(dst, 0x109, 0x3c, 0x109+0x2a, 0x3c+0x29, ColBlack)
	if logo := ss.S.Teams[m.Team].Logo; logo > 0 {
		pic := g.Pic(logo, false)
		b := pic.Bounds()
		r := image.Rect(b.Min.X, b.Min.Y, min(b.Max.X, b.Min.X+40), min(b.Max.Y, b.Min.Y+38))
		Draw(dst, pic.SubImage(r).(*ebiten.Image), 0x10b, 0x3e)
	}
}

// NewAlertText creates the alert box of f_3091_0e3a from a text with up to
// four lines separated by "\n", e.g. "Der Erbauer der\nletzten
// gespielten\nTorszene war:\n"+name.
func NewAlertText(text string) *AlertBox { return NewAlert(strings.Split(text, "\n")...) }

// noticePopup is the message panel of f_3091_05c4 with a single message:
// it closes with its X button.
type noticePopup struct {
	panel MessagePanel
	done  func(*App)
}

func newNoticePopup(ss *Session, n game.Notice, done func(*App)) *noticePopup {
	d := game.DateOf(ss.S.StartYear, n.Day)
	msg := append([]string{itoa(d.Day) + ". " + game.MonthNames[d.Month] + " " + itoa(d.Year)}, n.Lines...)
	p := &noticePopup{done: done}
	p.panel = MessagePanel{Messages: func() [][]string { return [][]string{msg} }, pressed: -1}
	return p
}

func (p *noticePopup) Update(app *App) error {
	if !p.panel.Update(&app.Input) || app.Input.Escape {
		app.Pop()
		p.done(app)
	}
	return nil
}

func (p *noticePopup) Draw(app *App, dst *ebiten.Image) { p.panel.Draw(app.Gfx, dst) }

// cardScreen is the card of a special date (f_1a58_2a06): dark red with an
// orange frame, the picture (PIC/42.VGA Christmas tree, PIC/41.VGA
// birthday cake) in a white frame, the date at the top, the title in
// orange and the text in beige. The Christmas card waits for a click; a
// birthday card closes after a minute, and a click before scolds the
// player ("Das war keine Minute !") and closes after a while.
type cardScreen struct {
	n      game.Notice
	start  int
	done   func(*App)
	frames int
	shame  int // frames since the early click, 0 = none
}

// Durations of the birthday card at 60 frames per second: the minute of
// the timer (0x438 ticks of 18.2 Hz) and the pause after the scolding
// (f_3930_01e7(0x96)).
const (
	cardMinute = 0x438 * 60 * 10 / 182
	cardShame  = 0x96 * 60 * 10 / 182
)

func (c *cardScreen) close(app *App) {
	app.Pop()
	c.done(app)
}

func (c *cardScreen) Update(app *App) error {
	in := &app.Input
	c.frames++
	if c.shame > 0 {
		if c.shame++; c.shame > cardShame {
			c.close(app)
		}
		return nil
	}
	click := in.Left || in.Right || in.Enter || in.Escape || len(in.Chars) > 0
	if !c.n.Birthday {
		if click {
			c.close(app)
		}
		return nil
	}
	if c.frames > cardMinute {
		c.close(app)
	} else if click {
		c.shame = 1
	}
	return nil
}

func (c *cardScreen) Draw(app *App, dst *ebiten.Image) {
	g := app.Gfx
	fill6(dst, 0, 0, 319, 239, g.C(ColDarkRed))
	rect6(dst, 0, 0, 319, 239, g.C(ColOrange))
	pic := 42
	if c.n.Birthday {
		pic = 41
	}
	Draw(dst, g.Sub(pic, rectWH(0, 0, 0x30, 0x39)), 0x88, 0x34)
	rect6(dst, 0x86, 0x32, 0xb7, 0x6c, g.C(ColWhite))
	d := game.DateOf(c.start, c.n.Day)
	head := game.WeekdayNames[game.Weekday(c.n.Day)] + ", " + itoa(d.Day) + ". " + game.MonthNames[d.Month]
	g.TextMid(dst, FontNormal, 0, 319, 8, head, ColBeige, ColBlack)
	if c.n.Birthday {
		g.TextMid(dst, FontNormal, 0, 319, 0x21, c.n.Title, ColOrange, ColBlack)
		for i, l := range c.n.Lines {
			g.TextMid(dst, FontNormal, 0, 319, 0x7d+i*11, l, ColBeige, ColBlack)
		}
		if c.shame > 0 && len(c.n.Small) == 2 {
			g.TextMid(dst, FontNormal, 0, 319, 0xe4, c.n.Small[0], ColBeige, ColBlack)
			g.TextMid(dst, FontNormal, 0, 319, 0xee, c.n.Small[1], ColBeige, ColBlack)
		}
		return
	}
	g.TextMid(dst, FontNormal, 0, 319, 0x23, c.n.Title, ColOrange, ColBlack)
	for i, l := range c.n.Lines {
		g.TextMid(dst, FontNormal, 0, 319, 0x8c+i*11, l, ColBeige, ColBlack)
	}
	for i, l := range c.n.Small {
		g.TextMid(dst, FontSmall, 0, 319, 0xde+i*8, l, ColBeige, ColBlack)
	}
}

// trophyScreen is f_17c2_274d: the German flag (f_0310_00fa: beige border,
// black, red and gold stripes), the club in the big font, "gewinnt die" in
// the normal font, the trophy (PIC/50.VGA + cup) and the title at the
// bottom; a click closes it.
type trophyScreen struct {
	n    game.Notice
	done func(*App)
}

func (t *trophyScreen) Update(app *App) error {
	in := &app.Input
	if in.Left || in.Enter || in.Escape || len(in.Chars) > 0 {
		app.Pop()
		t.done(app)
	}
	return nil
}

func (t *trophyScreen) Draw(app *App, dst *ebiten.Image) {
	g := app.Gfx
	fill6(dst, 0, 0, 319, 239, g.C(ColBeige))
	fill6(dst, 1, 1, 0x13e, 0x4f, g.C(ColBlack))
	fill6(dst, 1, 0x50, 0x13e, 0xa0, g.C(ColRed))
	fill6(dst, 1, 0xa1, 0x13e, 0xee, g.C(ColOrange))
	line := func(i int) string {
		if i < len(t.n.Lines) {
			return t.n.Lines[i]
		}
		return ""
	}
	g.TextMid(dst, FontBig, 0, 319, 0x21, line(0), ColWhite, ColBlack)
	g.TextMid(dst, FontBig, 0, 319, 0xdd, line(2), ColWhite, ColBlack)
	g.TextMid(dst, FontNormal, 0, 319, 0x3c, line(1), ColWhite, ColBlack)
	Draw(dst, g.Sub(50+t.n.Cup, rectWH(0, 0, 0x58, 0x58)), 0x74, 0x54)
}

// seasonEndScreen is the season report of f_0cb5_0012 on the backdrop of
// f_14a4_260a: a window with "Sie beendeten Ihre Saison" and the place,
// and a window "Ereignisse" with one line per event. Its dialogs come up
// while it is drawn, its popups (retirements) after the click.
type seasonEndScreen struct {
	ss     *Session
	n      game.Notice
	done   func(*App)
	dialog int // dialogs shown
	popup  int // popups shown, -1 = waiting for the click
	shown  int // lines drawn
	busy   bool
}

func (s *seasonEndScreen) Update(app *App) error {
	if s.busy {
		return nil
	}
	if s.dialog < len(s.n.Dialogs) {
		d := s.n.Dialogs[s.dialog]
		s.dialog++
		s.shown = d.At
		a := NewAlert(d.Lines...)
		app.Push(a)
		return nil
	}
	if s.shown < len(s.n.Lines) {
		s.shown = len(s.n.Lines)
		return nil
	}
	if s.popup == 0 {
		in := &app.Input
		if !(in.Left || in.Enter || in.Escape || len(in.Chars) > 0) {
			return nil
		}
	}
	if s.popup < len(s.n.Popups) {
		p := newNoticePopup(s.ss, s.n.Popups[s.popup], func(*App) { s.busy = false })
		s.popup++
		s.busy = true
		app.Push(p)
		return nil
	}
	app.Pop()
	s.done(app)
	return nil
}

func (s *seasonEndScreen) Draw(app *App, dst *ebiten.Image) {
	g := app.Gfx
	g.Background(dst)
	drawManagerBox(app, dst, s.ss, s.n.Manager)
	g.Window(dst, 0x17, 0x19, 0x17+0xde, 0x19+0x3c, ColBlue)
	g.Window(dst, 10, 0x7e, 10+0x127, 0x7e+0x52, ColBlue)
	g.TextMid(dst, FontNormal, 0x17, 0xf5, 0x21, s.n.Title, ColBeige, NoShadow)
	line6(dst, 0x18, 0x24, 0xf4, 0x24, g.C(ColBeige))
	g.TextMid(dst, FontNormal, 10, 0x131, 0x86, s.n.Heading, ColBeige, NoShadow)
	line6(dst, 0xb, 0x89, 0x130, 0x89, g.C(ColBeige))
	if len(s.n.Lines) > 0 {
		g.TextMid(dst, FontNormal, 0x17, 0xf5, 0x3c, s.n.Lines[0], ColLight, NoShadow)
	}
	for i := 1; i < s.shown && i < len(s.n.Lines); i++ {
		g.TextMid(dst, FontNormal, 10, 0x131, 0x94+(i-1)*13, s.n.Lines[i], ColBeige, NoShadow)
	}
}

// NoticeDemo is a scene for screenshots: it shows game.SampleNotices one
// after another over the main menu of a demo game (bmp -demo notices).
func NoticeDemo(ss *Session) Scene {
	for _, n := range game.SampleNotices(ss.S) {
		ss.S.AddNotice(n)
	}
	return &noticeDemo{ss: ss, menu: NewTurnMenu(ss)}
}

type noticeDemo struct {
	ss      *Session
	menu    *MainMenu
	started bool
}

func (d *noticeDemo) Update(app *App) error {
	if !d.started {
		d.started = true
		ShowNotices(app, d.ss, nil)
	}
	return nil
}

func (d *noticeDemo) Draw(app *App, dst *ebiten.Image) { d.menu.Draw(app, dst) }
