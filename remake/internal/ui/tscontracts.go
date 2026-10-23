package ui

import (
	"github.com/hajimehoshi/ebiten/v2"

	"github.com/schowave/bmp/remake/internal/game"
)

// ContractScreen is the contract list of the team group (f_1ecd_1560 with
// 'c' / f_2277_2a8f with -1): the lineup screen showing the contracts. A
// click on a player opens the negotiation panel. At the season end
// (seasonEnd) it negotiates the expiring contracts of manager Current one
// after the other (the season-end loop of f_1a58_3176) and continues the
// game when done.
type ContractScreen struct {
	ss        *Session
	seasonEnd bool
	inner     Scene // the lineup screen in its contract view
	list      *tsList
	busy      bool // a negotiation of the season end is running
}

// NewContractScreen creates the contract screen.
func NewContractScreen(ss *Session, seasonEnd bool) *ContractScreen {
	c := &ContractScreen{ss: ss, seasonEnd: seasonEnd}
	if !seasonEnd {
		c.inner = NewSquadContracts(ss)
	}
	return c
}

// tsContractList is the contract list of manager m in the window of the
// lineup screen.
func tsContractList(m int) *tsList {
	return &tsList{m: m, x: 6, head: 24, right: 0xec, foot: 0xb0, mask: tsMaskContracts, mode: 3,
		title: "IHRE MANNSCHAFT", hitX: 6, rowY: 0x1e, hitRight: 0xef, hintY: 0xb8}
}

// Passing implements passer: at the season end the screen only opens the
// talks with the next player, or closes.
func (c *ContractScreen) Passing() bool { return c.seasonEnd && !c.busy }

func (c *ContractScreen) Update(app *App) error {
	if !c.seasonEnd {
		return c.inner.Update(app)
	}
	s := c.ss.S
	if c.busy {
		return nil
	}
	exp := s.ExpiringContracts()
	if len(exp) == 0 {
		s.ContractsDone()
		app.Pop()
		return nil
	}
	mi := s.Loop.Current
	si := exp[0]
	name := s.Persons[s.Squads[mi][si].Person].Name
	c.busy = true
	app.Push(newTsPanel(c.ss, mi, si, tsPanelSeason, func(app *App, ok bool) {
		if ok {
			app.Push(newTsMessage(name, "bleibt Ihnen auch", "die nächste Saison", "erhalten.",
				func(*App) { c.busy = false }))
			return
		}
		comp := s.ReleasePlayer(mi, si)
		app.Push(newTsMessage(name, "kehrt Ihrem Verein", "den Rücken...", "", func(app *App) {
			app.Push(newTsMessage("Sie erhalten eine", "Ablösesumme von", itoa64(comp)+" DM", "",
				func(*App) { c.busy = false }))
		}))
	}))
	return nil
}

func (c *ContractScreen) Draw(app *App, dst *ebiten.Image) {
	if !c.seasonEnd {
		c.inner.Draw(app, dst)
		return
	}
	g := app.Gfx
	g.Background(dst)
	g.Window(dst, 2, 7, 2+0xee, 7+0xb5, ColBlue)
	if c.list == nil || c.list.m != c.ss.S.Loop.Current {
		c.list = tsContractList(c.ss.S.Loop.Current)
	}
	c.list.draw(g, dst, c.ss.S, c.ss.Mgr(), -1)
}

// negotiate starts the talks with squad slot si of the current manager:
// the reasons why he does not talk (as a message), or the panel.
func (c *ContractScreen) negotiate(app *App, si int) {
	s := c.ss.S
	mi := c.ss.Mgr()
	sl := &s.Squads[mi][si]
	name := s.Persons[sl.Person].Name
	switch {
	case sl.Loan != 0:
		app.Push(newTsMessage(name, "ist leider von", s.Teams[sl.Loan&0x7f].Name, "nur ausgeliehen.", nil))
	case sl.Nego&0x80 != 0:
		app.Push(newTsMessage(name, "wird sich mit", "Ablauf des Vertrages", "zur Ruhe setzen.", nil))
	case sl.Nego > 0 && sl.Nego < 100:
		app.Push(newTsMessage(name, "ist im Moment", "nicht verhandlungs-", "bereit.", nil))
	default:
		app.Push(newTsPanel(c.ss, mi, si, tsPanelNormal, nil))
	}
}

// Purposes of the negotiation panel.
const (
	tsPanelNormal   = iota // contract list: "Ihr Angebot wurde angenommen !"
	tsPanelPurchase        // contract of a player being bought
	tsPanelSeason          // expiring contract at the season end
)

// tsPanel is the negotiation panel of f_2277_2a8f below the list: the
// player, the contract length and the salary, which are typed in after a
// click on their line, and the buttons NEUER VERTRAG and ABBRUCH.
type tsPanel struct {
	ss      *Session
	m, si   int
	purpose int
	done    func(app *App, ok bool)

	years  int   // typed contract length, 0 = none
	salary int64 // typed (or proposed) salary, 0 = none
	shownY string
	shownS string
	field  tsNumField
	edit   int // 0 years, 1 salary while typing
	thanks int // frames left of "Ihr Angebot wurde angenommen !"
	wait   bool
}

func newTsPanel(ss *Session, m, si, purpose int, done func(app *App, ok bool)) *tsPanel {
	sl := &ss.S.Squads[m][si]
	return &tsPanel{ss: ss, m: m, si: si, purpose: purpose, done: done,
		shownY: itoa(sl.Years), shownS: itoa64(sl.Salary)}
}

func (p *tsPanel) slot() *game.Slot { return &p.ss.S.Squads[p.m][p.si] }

func (p *tsPanel) close(app *App, ok bool) {
	if app.stack[len(app.stack)-1] == Scene(p) {
		app.Pop()
	}
	if p.done != nil {
		p.done(app, ok)
	}
}

func (p *tsPanel) Update(app *App) error {
	in := &app.Input
	if p.wait {
		return nil
	}
	if p.thanks > 0 {
		p.thanks--
		if p.thanks == 0 {
			p.close(app, true)
		}
		return nil
	}
	if p.field.active {
		if p.field.update(in) {
			p.commit()
		}
		return nil
	}
	switch {
	case tsButtonHit(in, 0xb8, 0xca):
		p.offer(app)
		return nil
	case tsButtonHit(in, 0xb8, 0xd8):
		p.cancel(app)
		return nil
	}
	if in.Left && in.X > 8 && in.X < 0xb4 && in.Y > 0xd2 && in.Y < 0xe2 {
		if in.Y > 0xd9 {
			p.edit = 1
			p.field.start(0x87, 0xe1, 6, FontSmall, ColGrey2)
		} else if p.slot().Loan == 0 {
			p.edit = 0
			p.field.start(0x87, 0xd9, 1, FontSmall, ColGrey2)
		}
	}
	return nil
}

// commit takes the typed value (f_2277_2a8f): a contract length proposes
// the salary the player would like.
func (p *tsPanel) commit() {
	s := p.ss.S
	sl := p.slot()
	empty := p.field.text == ""
	v := p.field.value()
	if p.edit == 0 {
		if empty {
			p.years = 0
			p.shownY = itoa(sl.Years)
			return
		}
		p.years = int(v)
		p.shownY = itoa(p.years)
		p.salary = s.ContractHint(p.m, p.si, p.years)
		p.shownS = itoa64(p.salary)
		return
	}
	if empty {
		p.shownS = itoa64(sl.Salary)
		return
	}
	p.salary = v
	p.shownS = itoa64(v)
}

// offer evaluates NEUER VERTRAG.
// forced is the contract length the player insists on, or 0.
func (p *tsPanel) forced() int {
	if sl := p.slot(); sl.Nego >= 100 && sl.Nego&0x80 == 0 {
		return sl.Nego - 100
	}
	return 0
}

// untouched reports whether nothing usable was typed (f_2277_2a8f checks it
// after either button): no length, and no salary or no contract left. A
// purchase during the season skips the check.
func (p *tsPanel) untouched() bool {
	s := p.ss.S
	return (p.purpose == tsPanelNormal || s.Day > 321) && p.forced() == 0 && p.years == 0 &&
		(p.salary == 0 || p.slot().Years == 0)
}

// cancel is ABBRUCH. With nothing typed during the season nothing changes;
// otherwise the player refuses to talk for a while.
func (p *tsPanel) cancel(app *App) {
	s := p.ss.S
	if p.years > 4 {
		s.CancelNegotiation(p.m, p.si)
		p.wait = true
		app.Push(newTsMessage("Wer wird sich denn SO", "lange verpflichten ?", "(Mal abgesehen", "von Bundi's)",
			func(app *App) { p.close(app, false) }))
		return
	}
	if p.untouched() && s.Day <= 321 {
		p.close(app, false)
		return
	}
	s.CancelNegotiation(p.m, p.si)
	p.close(app, false)
}

func (p *tsPanel) offer(app *App) {
	s := p.ss.S
	sl := p.slot()
	name := s.Persons[sl.Person].Name
	forced := p.forced()
	fail := func(l0, l1, l2, l3 string) {
		s.CancelNegotiation(p.m, p.si)
		if l0 == "" {
			p.close(app, false)
			return
		}
		p.wait = true
		app.Push(newTsMessage(l0, l1, l2, l3, func(app *App) { p.close(app, false) }))
	}
	if p.years > 4 {
		fail("Wer wird sich denn SO", "lange verpflichten ?", "(Mal abgesehen", "von Bundi's)")
		return
	}
	if p.untouched() {
		if s.Day > 321 {
			fail("", "", "", "")
			return
		}
		p.close(app, false) // nothing typed: nothing changes
		return
	}
	y, sal := p.years, p.salary
	if y == 0 {
		y = sl.Years
		if forced != 0 {
			y = forced
		}
	}
	if sal == 0 {
		sal = sl.Salary
	}
	if y > sl.Years && sal < sl.Salary {
		fail("So dumm", "ist", name, "leider nicht...")
		return
	}
	if forced != 0 && (sal < sl.Salary || y != forced) {
		fail("", "", "", "")
		return
	}
	if err := s.NegotiateContract(p.m, p.si, y, sal, forced); err != nil {
		p.wait = true
		app.Push(newTsMessage(name, "ist nicht an", "Ihrem Angebot", "interessiert.",
			func(app *App) { p.close(app, false) }))
		return
	}
	if p.purpose == tsPanelNormal {
		p.thanks = 60
		return
	}
	p.close(app, true)
}

func (p *tsPanel) Draw(app *App, dst *ebiten.Image) {
	g := app.Gfx
	s := p.ss.S
	sl := p.slot()
	ps := &s.Persons[sl.Person]
	g.Window(dst, 4, 199, 4+0xee, 199+0x1f, ColBlue)
	if p.thanks > 0 {
		g.TextMid(dst, FontNormal, 6, 0xf4, 0xda, "Ihr Angebot wurde angenommen !", ColLight, ColBlue)
		return
	}
	g.TextAt(dst, FontSmall, 8, 0xcf, ps.Name+", "+itoa(ps.Age)+" JAHRE ALT, "+itoa(sl.Games[0])+" LIGASPIELE",
		ColLight, NoShadow)
	FillRect(dst, 8, 0xd1, 0xb4-8+1, 1, g.PalColor(ColLight))
	g.TextAt(dst, FontSmall, 8, 0xd9, "VERTRAGSDAUER (IN SAISONS)", ColGrey2, NoShadow)
	g.TextAt(dst, FontSmall, 0x82, 0xd9, ":", ColGrey2, NoShadow)
	g.TextAt(dst, FontSmall, 8, 0xe1, "GEHALT PRO MONAT (IN DM)", ColGrey2, NoShadow)
	g.TextAt(dst, FontSmall, 0x82, 0xe1, ":", ColGrey2, NoShadow)
	if p.field.active && p.edit == 0 {
		p.field.draw(g, dst)
	} else {
		g.TextAt(dst, FontSmall, 0x87, 0xd9, p.shownY, ColGrey2, NoShadow)
	}
	if p.field.active && p.edit == 1 {
		p.field.draw(g, dst)
	} else {
		g.TextAt(dst, FontSmall, 0x87, 0xe1, p.shownS, ColGrey2, NoShadow)
	}
	tsButton(g, dst, 0xb8, 0xca, "NEUER VERTRAG", ColRed)
	tsButton(g, dst, 0xb8, 0xd8, "ABBRUCH", ColLight)
}
