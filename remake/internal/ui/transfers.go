package ui

import (
	"strings"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/schowave/bmp/remake/internal/game"
)

// TransferScreen is the transfer market (f_2277_04a5): the own squad on
// the left, the market at the top right with LEIHEN and KAUFEN, below it
// the panel with the bid and the Kontostand. A click on an own player puts
// him on the market (or shows the offer of a club), a click on an own
// listed player takes him back, a click on another player asks for a bid.
type TransferScreen struct {
	ss          *Session
	own, market *tsList
	loan        bool

	bid      tsNumField
	bidSlot  int  // market slot of the bid being typed
	showBid  bool // "Ihr Angebot:" stays while the answer is shown
	offer    *game.Offer
	offerAns func(app *App, sell bool)
}

// NewTransferScreen creates the transfer market screen.
func NewTransferScreen(ss *Session) *TransferScreen {
	mi := ss.Mgr()
	return &TransferScreen{ss: ss,
		own: &tsList{m: mi, x: 4, head: 30, right: 153, foot: 182, mask: tsMaskTransfer, mode: 0,
			title: "IHRE MANNSCHAFT", hitX: 2, rowY: 0x24, hitRight: 0x9b, hintY: 0xbe},
		market: &tsList{m: game.MarketIndex, x: 163, head: 29, right: 315, foot: 107, mask: tsMaskMarket, mode: 1,
			title: "TRANSFERMARKT", hitX: 163, rowY: 0x1d, hitRight: 0xa1 + 0x99, hintY: 0xbe, rowBase: 1}}
}

// tsToastText splits a refusal of the game into the two lines of the
// notice.
func tsToastText(app *App, msg string, done func(app *App)) {
	for _, l := range [][2]string{
		{"Dieser Spieler hört", "am Saisonende auf."},
		{"Schon 3 Spieler auf", "dem Transfermarkt"},
		{"Dieser Spieler ist", "nur ausgeliehen"},
		{"Angebot wurde", "bereits abgelehnt"},
		{"Sie haben nicht", "genug Geld"},
		{"Ihr Angebot", "wurde abgelehnt"},
		{"Der Transfer", "findet nicht statt"},
	} {
		if msg == l[0]+" "+l[1] {
			app.Push(newTsToast(l[0], l[1], done))
			return
		}
	}
	app.Push(newTsToast(msg, "", done))
}

// marketSlot returns the market slot under the pointer, or -1.
func (t *TransferScreen) marketSlot(in *Input) int {
	s := t.ss.S
	r := (in.Y - t.market.rowY) / 6
	if in.Y < t.market.rowY || in.X < 0xa1 || in.X > 0xa1+0x9b || r < 1 || r > s.SquadSize(game.MarketIndex) {
		return -1
	}
	return r - 1
}

func (t *TransferScreen) Update(app *App) error {
	s := t.ss.S
	mi := t.ss.Mgr()
	in := &app.Input
	t.market.mode = 1
	if t.loan {
		t.market.mode = 2
	}
	t.market.updatePrices(s)
	if t.bid.active {
		if t.bid.update(in) {
			t.placeBid(app)
		}
		return nil
	}
	if t.offer != nil {
		switch {
		case tsButtonHit(in, 0xaf, 0x6e):
			t.answerOffer(app, false)
		case tsButtonHit(in, 0xf5, 0x6e):
			t.answerOffer(app, true)
		}
		return nil
	}
	if MenuButtonHit(in) {
		s.LineupStrength(mi, false)
		app.Pop()
		return nil
	}
	switch {
	case tsButtonHit(in, 0xaf, 0x6e):
		t.loan = true
		return nil
	case tsButtonHit(in, 0xf5, 0x6e):
		t.loan = false
		return nil
	}
	if !in.Left && !in.Right {
		return nil
	}
	if si, _, ok := t.own.cell(s, in.X, in.Y); ok {
		if in.Right {
			app.Push(NewPlayerInfo(t.ss, mi, si))
			return nil
		}
		t.clickOwn(app, si)
		return nil
	}
	if i := t.marketSlot(in); i >= 0 {
		p := &s.Persons[s.Squads[game.MarketIndex][i].Person]
		if in.Right {
			if p.Owner == mi {
				app.Push(NewPlayerInfo(t.ss, game.MarketIndex, i))
			}
			return nil
		}
		t.clickMarket(app, i)
	}
	return nil
}

// clickOwn handles a click on an own player.
func (t *TransferScreen) clickOwn(app *App, si int) {
	s := t.ss.S
	mi := t.ss.Mgr()
	if s.Squads[mi][si].Nego&0x80 == 0 {
		if o, ok := s.SquadOffer(mi, si); ok {
			t.showOffer(o, func(app *App, sell bool) { s.AnswerSquadOffer(mi, si, o, sell) })
			return
		}
	}
	if err := s.ListPlayer(mi, si); err != nil {
		tsToastText(app, err.Error(), nil)
	}
}

// clickMarket handles a click on a player of the market.
func (t *TransferScreen) clickMarket(app *App, i int) {
	s := t.ss.S
	mi := t.ss.Mgr()
	sl := &s.Squads[game.MarketIndex][i]
	p := &s.Persons[sl.Person]
	switch {
	case sl.Rejected&0x80 != 0 && sl.Rejected&(1<<mi) != 0:
		tsToastText(app, "Angebot wurde bereits abgelehnt", nil)
	case p.Owner == mi:
		if o, ok := s.ListedOffer(mi, i); ok {
			t.showOffer(o, func(app *App, sell bool) { s.AnswerListedOffer(mi, i, o, sell) })
			return
		}
		if _, err := s.Bid(mi, i, 0, false); err != nil {
			tsToastText(app, "Der Transfer findet nicht statt", nil)
		}
	default:
		t.bidSlot = i
		t.showBid = true
		t.bid.start(0xa7, 0x98, 7, FontNormal, ColLight)
	}
}

func (t *TransferScreen) showOffer(o game.Offer, answer func(app *App, sell bool)) {
	t.offer = &o
	t.offerAns = answer
}

func (t *TransferScreen) answerOffer(app *App, sell bool) {
	ans := t.offerAns
	t.offer, t.offerAns = nil, nil
	ans(app, sell)
}

// placeBid sends the typed bid to the seller.
func (t *TransferScreen) placeBid(app *App) {
	s := t.ss.S
	mi := t.ss.Mgr()
	i := t.bidSlot
	bid := t.bid.value()
	end := func(*App) { t.showBid = false }
	if bid <= 0 {
		t.showBid = false
		return
	}
	p := &s.Persons[s.Squads[game.MarketIndex][i].Person]
	seller := ""
	if p.Owner < len(s.Managers) {
		seller = s.Managers[p.Owner].Name
	}
	res, err := s.Bid(mi, i, bid, t.loan)
	if err != nil {
		if strings.HasPrefix(err.Error(), "DIESER BETRAG") {
			app.Push(NewQuestion(seller+":", "DIESER BETRAG LIEGT\nAUßERHALB DES ERLAUBTEN !", "NIE", "NEIN",
				func(app *App, _ bool) { end(app) }))
			return
		}
		tsToastText(app, err.Error(), end)
		return
	}
	t.bidResult(app, i, bid, res, seller, p.Name)
}

func (t *TransferScreen) bidResult(app *App, i int, bid int64, res game.BuyResult, seller, name string) {
	s := t.ss.S
	mi := t.ss.Mgr()
	end := func(*App) { t.showBid = false }
	switch res {
	case game.BidAskOtherManager:
		app.Push(NewQuestion(seller+":", "NEHMEN SIE DAS ANGEBOT ("+Money(bid)+")\nFÜR "+name+" AN ?", "NA GUT.", "NEIN !",
			func(app *App, yes bool) {
				res, err := s.AnswerManagerBid(mi, i, bid, t.loan, yes)
				if err != nil {
					tsToastText(app, err.Error(), end)
					return
				}
				t.bidResult(app, i, bid, res, seller, name)
			}))
	case game.BidNeedsContract:
		si, err := s.BeginPurchase(mi, i, bid)
		if err != nil {
			tsToastText(app, "Der Transfer findet nicht statt", end)
			return
		}
		app.Push(newTsPanel(t.ss, mi, si, tsPanelPurchase, func(app *App, ok bool) {
			if err := s.FinishPurchase(ok); err != nil {
				tsToastText(app, err.Error(), end)
				return
			}
			t.showBid = false
			if z := s.PendingSubsidy; z > 0 {
				app.Push(NewQuestion(t.ss.M().Name+":", "NEHMEN SIE EINEN SPONSOR-\nZUSCHUß VON\n"+Money(z)+" AN ?",
					"ABER IMMER!", "LIEBER NICHT.", func(app *App, yes bool) { s.AcceptSubsidy(mi, yes) }))
			}
		}))
	default:
		t.showBid = false
	}
}

func (t *TransferScreen) Draw(app *App, dst *ebiten.Image) {
	g := app.Gfx
	s := t.ss.S
	mi := t.ss.Mgr()
	in := &app.Input
	g.Background(dst)
	g.Window(dst, 0, 0xd, 0x9d, 0xd+0xb5, ColBlue)
	g.Window(dst, 0xa1, 0xd, 0xa1+0x9b, 0xd+0x70, ColBlue)
	g.DrawMenuButton(dst)
	g.Window(dst, 0xa1, 0x83, 0xa1+0x9b, 0x83+0x31, ColBlue)
	g.TextAt(dst, FontNormal, 0xa5, 0xa4, "Kontostand:", ColBeige, NoShadow)
	FillRect(dst, 0xa3, 0xa6, 0x13a-0xa3+1, 1, g.PalColor(ColBeige))
	g.TextAt(dst, FontNormal, 0xa5, 0xb0, Money(t.ss.M().Money), ColLight, NoShadow)

	busy := t.bid.active || t.offer != nil || len(app.stack) > 0 && app.stack[len(app.stack)-1] != Scene(t)
	hoverOwn, hoverMarket := -1, -1
	if !busy {
		if si, _, ok := t.own.cell(s, in.X, in.Y); ok {
			hoverOwn = si
		}
		hoverMarket = t.marketSlot(in)
	}
	t.own.m = mi
	t.own.draw(g, dst, s, mi, hoverOwn)
	t.market.draw(g, dst, s, mi, hoverMarket)
	if hoverOwn >= 0 {
		t.own.drawHint(g, dst, s, in.X, in.Y)
	} else if hoverMarket >= 0 {
		p := &s.Persons[s.Squads[game.MarketIndex][hoverMarket].Person]
		from := ""
		switch {
		case p.Owner == game.OwnerMarket && p.Team < game.NumTeams:
			from = s.Teams[p.Team].Name
		case p.Owner < len(s.Managers):
			from = s.Managers[p.Owner].Name
		}
		h := "VON " + from + ", " + itoa(p.Age) + " J." + game.SideText(p.Side)
		g.TextMid(dst, FontSmall, 2, 0x9b, 0xbe, h, ColLight, NoShadow)
	}

	if t.offer != nil {
		g.TextAt(dst, FontNormal, 0xa5, 0x8e, "Angebot", ColLight, NoShadow)
		g.TextAt(dst, FontNormal, 0xa5, 0x98, "Ablöse", ColLight, NoShadow)
		g.TextAt(dst, FontNormal, 0xd9, 0x8e, ": "+Money(t.offer.Amount)+".", ColLight, NoShadow)
		g.TextAt(dst, FontNormal, 0xd9, 0x98, ": "+Money(t.offer.Credited)+".", ColLight, NoShadow)
		FillRect(dst, 0xa3, 0x9a, 0x13a-0xa3+1, 1, g.PalColor(ColBeige))
		tsButton(g, dst, 0xaf, 0x6e, "BEHALTEN", ColLight)
		tsButton(g, dst, 0xf5, 0x6e, "VERKAUFEN", ColRed)
		return
	}
	lc, kc := ColLight, ColRed
	if t.loan {
		lc, kc = ColRed, ColLight
	}
	tsButton(g, dst, 0xaf, 0x6e, "LEIHEN", lc)
	tsButton(g, dst, 0xf5, 0x6e, "KAUFEN", kc)
	if t.showBid {
		g.TextAt(dst, FontNormal, 0xa5, 0x8c, "Ihr Angebot:", ColLight, NoShadow)
		FillRect(dst, 0xa3, 0x8e, 0x13a-0xa3+1, 1, g.PalColor(ColLight))
		t.bid.draw(g, dst)
	}
}
