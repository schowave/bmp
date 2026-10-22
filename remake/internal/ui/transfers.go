package ui

import (
	"github.com/hajimehoshi/ebiten/v2"

	"github.com/schowave/bmp/remake/internal/game"
)

// TransferScreen is the transfer market (f_2277_04a5): own squad on the
// left, the market on the right.
type TransferScreen struct {
	ss          *Session
	own, market *TableView
	loan        bool
	buy, lend   Button
}

// NewTransferScreen creates the transfer market screen.
func NewTransferScreen(ss *Session) *TransferScreen {
	cols := []Column{{"NAME", 2, 62, Left}, {"POS", 66, 16, Left}, {"ST", 84, 12, Right}, {"AL", 98, 12, Right}, {"WERT", 112, 42, Right}}
	t := &TransferScreen{ss: ss,
		own:    NewTableView(2, 24, 157, 168, cols),
		market: NewTableView(161, 24, 157, 168, append([]Column(nil), cols...))}
	t.market.Columns[4].Title = "PREIS"
	t.buy = Button{X: 4, Y: 208, W: 50, H: 13, Label: "KAUFEN"}
	t.lend = Button{X: 58, Y: 208, W: 50, H: 13, Label: "LEIHEN"}
	return t
}

func (t *TransferScreen) refresh() {
	s := t.ss.S
	mi := t.ss.Mgr()
	t.own.Rows = t.own.Rows[:0]
	for i := 0; i < s.SquadSize(mi); i++ {
		sl := &s.Squads[mi][i]
		p := &s.Persons[sl.Person]
		row := Row{Cells: []string{p.Name, posShort[p.Position()], itoa(sl.Strength()), itoa(p.Age), formatMoney(s.PlayerValue(mi, i, 0))}}
		if sl.Status&0x40 != 0 {
			row.Color = ColYellow
		}
		t.own.Rows = append(t.own.Rows, row)
	}
	t.market.Rows = t.market.Rows[:0]
	for i := 0; i < s.SquadSize(game.MarketIndex); i++ {
		sl := &s.Squads[game.MarketIndex][i]
		p := &s.Persons[sl.Person]
		row := Row{Cells: []string{p.Name, posShort[p.Position()], itoa(sl.Strength()), itoa(p.Age), formatMoney(s.MarketPrice(i, t.loan))}}
		if p.Owner == mi {
			row.Color = ColBeige
			if sl.Status&0x80 != 0 {
				row.Color = ColYellow
			}
		}
		if sl.Rejected&0x80 != 0 && sl.Rejected&(1<<mi) != 0 {
			row.Color = ColGrey5
		}
		t.market.Rows = append(t.market.Rows, row)
	}
}

func (t *TransferScreen) Update(app *App) error {
	s := t.ss.S
	mi := t.ss.Mgr()
	name := t.ss.M().Name + ":"
	in := &app.Input
	t.refresh()
	if t.buy.Hit(in) {
		t.loan = false
	}
	if t.lend.Hit(in) {
		t.loan = true
	}
	if l, r := t.own.Update(in); r >= 0 {
		app.Push(NewPlayerInfo(t.ss, mi, r))
		return nil
	} else if l >= 0 {
		if o, ok := s.SquadOffer(mi, l); ok {
			t.offerDialog(app, o, func(sell bool) { s.AnswerSquadOffer(mi, l, o, sell) })
			return nil
		}
		pname := s.Persons[s.Squads[mi][l].Person].Name
		app.Push(NewQuestion(name, pname+" auf den Transfermarkt setzen ?", "JA", "NEIN", func(app *App, yes bool) {
			if yes {
				if err := s.ListPlayer(mi, l); err != nil {
					app.Push(NewMessage(name, err.Error()))
				}
			}
		}))
		return nil
	}
	if l, r := t.market.Update(in); r >= 0 {
		if s.Persons[s.Squads[game.MarketIndex][r].Person].Owner == mi {
			app.Push(NewPlayerInfo(t.ss, game.MarketIndex, r))
		}
		return nil
	} else if l >= 0 {
		t.clickMarket(app, l)
		return nil
	}
	if MenuButtonHit(in) {
		s.LineupStrength(mi, false)
		app.Pop()
	}
	return nil
}

func (t *TransferScreen) offerDialog(app *App, o game.Offer, answer func(sell bool)) {
	s := t.ss.S
	text := s.Teams[o.Club].Name + ": Angebot: " + Money(o.Amount) + ". Abzüglich " + Money(o.Reduce) +
		" für den laufenden Vertrag erhalten Sie " + Money(o.Credited) + "."
	app.Push(NewQuestion(t.ss.M().Name+":", text, "VERKAUFEN", "BEHALTEN", func(app *App, sell bool) { answer(sell) }))
}

func (t *TransferScreen) clickMarket(app *App, i int) {
	s := t.ss.S
	mi := t.ss.Mgr()
	name := t.ss.M().Name + ":"
	sl := &s.Squads[game.MarketIndex][i]
	p := &s.Persons[sl.Person]
	if p.Owner == mi {
		if o, ok := s.ListedOffer(mi, i); ok {
			t.offerDialog(app, o, func(sell bool) { s.AnswerListedOffer(mi, i, o, sell) })
			return
		}
		app.Push(NewQuestion(name, p.Name+" vom Transfermarkt zurücknehmen ?", "JA", "NEIN", func(app *App, yes bool) {
			if yes {
				if _, err := s.Bid(mi, i, 0, false); err != nil {
					app.Push(NewMessage(name, err.Error()))
				}
			}
		}))
		return
	}
	verb := "KAUFEN"
	if t.loan {
		verb = "LEIHEN"
	}
	from := "VON " + teamName(s, p.Team)
	if p.Owner < game.MarketIndex {
		from = "VON " + s.Managers[p.Owner].Name
	}
	app.Push(NewNumberInput(p.Name+" "+verb, from+", "+itoa(p.Age)+" J. - Ihr Angebot:", t.ss.S.MarketPrice(i, t.loan),
		func(app *App, bid int64, ok bool) {
			if !ok || bid <= 0 {
				return
			}
			res, err := s.Bid(mi, i, bid, t.loan)
			if err != nil {
				app.Push(NewMessage(name, err.Error()))
				return
			}
			switch res {
			case game.BidAskOtherManager:
				seller := s.Managers[p.Owner].Name
				app.Push(NewQuestion(seller+":", "NEHMEN SIE DAS ANGEBOT ("+Money(bid)+") FÜR "+p.Name+" AN ?", "NA GUT.", "NEIN !",
					func(app *App, yes bool) {
						if err := s.AnswerManagerBid(mi, i, bid, t.loan, yes); err != nil {
							app.Push(NewMessage(name, err.Error()))
						}
					}))
			case game.BidNeedsContract:
				t.purchaseContract(app, i, bid)
			case game.BidDone:
				app.Push(NewMessage(name, "Ihr Angebot wurde angenommen !"))
			}
		}))
}

// purchaseContract negotiates the contract of a player bought from the AI.
func (t *TransferScreen) purchaseContract(app *App, i int, bid int64) {
	s := t.ss.S
	mi := t.ss.Mgr()
	name := t.ss.M().Name + ":"
	si, err := s.BeginPurchase(mi, i, bid)
	if err != nil {
		app.Push(NewMessage(name, err.Error()))
		return
	}
	pname := s.Persons[s.Squads[mi][si].Person].Name
	finish := func(app *App, agreed bool) {
		if err := s.FinishPurchase(agreed); err != nil {
			app.Push(NewMessage(name, err.Error()))
			return
		}
		app.Push(NewMessage(name, pname+" spielt ab sofort für Sie."))
		if z := s.PendingSubsidy; z > 0 {
			app.Push(NewQuestion(name, "NEHMEN SIE EINEN SPONSOR-ZUSCHUSS VON "+Money(z)+" AN ?", "ABER IMMER!", "LIEBER NICHT.",
				func(app *App, yes bool) { s.AcceptSubsidy(mi, yes) }))
		}
	}
	app.Push(NewNumberInput(pname, "VERTRAGSDAUER (IN SAISONS)", 2, func(app *App, y int64, ok bool) {
		if !ok {
			finish(app, false)
			return
		}
		hint := s.ContractHint(mi, si, int(y))
		app.Push(NewNumberInput(pname, "GEHALT PRO MONAT (IN DM), WUNSCH ca. "+formatMoney(hint), hint,
			func(app *App, sal int64, ok bool) {
				if !ok {
					finish(app, false)
					return
				}
				if err := s.NegotiateContract(mi, si, int(y), sal, 0); err != nil {
					app.Push(NewMessage(name, err.Error()))
					finish(app, false)
					return
				}
				finish(app, true)
			}))
	}))
}

func (t *TransferScreen) Draw(app *App, dst *ebiten.Image) {
	g := app.Gfx
	g.Background(dst)
	g.Panel(dst, 2, 3, 157, 17)
	g.TextCentered(dst, FontNormal, 2, 7, 157, "IHRE MANNSCHAFT", g.C(ColBeige))
	g.Panel(dst, 161, 3, 157, 17)
	g.TextCentered(dst, FontNormal, 161, 7, 157, "TRANSFERMARKT", g.C(ColBeige))
	t.refresh()
	t.own.Draw(g, dst)
	t.market.Draw(g, dst)
	g.Text(dst, FontSmall, 4, 196, "Kontostand: "+Money(t.ss.M().Money), g.C(ColWhite))
	t.buy.Draw(g, dst, !t.loan)
	t.lend.Draw(g, dst, t.loan)
	g.Text(dst, FontSmall, 4, 226, "LINKS: ANBIETEN/BIETEN  RECHTS: INFO", g.C(ColLight))
	g.DrawMenuButton(dst)
}
