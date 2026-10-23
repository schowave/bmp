package ui

import (
	"testing"

	"github.com/schowave/bmp/remake/internal/game"
)

// The price of a listed player with an offer draws a random number. The
// list draws it when its rows change, not on every frame.
func TestMarketPricesDrawOnce(t *testing.T) {
	_, mana := testApp(t)
	s, err := game.NewGame(mana, game.Options{Managers: []game.NewManager{{Name: "TEST", Team: 9}}, Level: 2, Seed: 5})
	if err != nil {
		t.Fatal(err)
	}
	s.Start()
	sl := &s.Squads[game.MarketIndex][0]
	s.Persons[sl.Person].Owner = 0
	sl.Status |= 0x80
	sl.Buyer = 10

	l := &tsList{m: game.MarketIndex, mode: 1}
	before := s.Rand
	l.updatePrices(s)
	once := s.Rand
	if once == before {
		t.Fatal("no draw for the listed player")
	}
	for range 10 {
		l.updatePrices(s)
	}
	if s.Rand != once || l.price(0) == 0 {
		t.Errorf("prices drawn again, price %d", l.price(0))
	}

	l.mode = 2
	l.updatePrices(s)
	if s.Rand == once {
		t.Error("a new mode keeps the old prices")
	}
}

// The name of an own player a club made an offer for is light, in the squad
// list and in the market list; the rest of the row keeps its colour.
func TestOfferNameLight(t *testing.T) {
	_, ss := tsTestSession(t)
	s := ss.S
	mi := ss.Mgr()
	sl := &s.Squads[mi][3]
	squad := &tsList{m: mi, mode: 0}
	base := squad.rowColor(s, sl)
	if got := squad.nameColor(s, sl, mi, base); got != base {
		t.Errorf("name colour %d without an offer", got)
	}
	sl.Status |= 0x40
	if got := squad.nameColor(s, sl, mi, base); got != ColBeige || squad.rowColor(s, sl) != base {
		t.Errorf("name colour %d with an offer, row %d", got, squad.rowColor(s, sl))
	}
	ms := &s.Squads[game.MarketIndex][0]
	if ms.Empty() {
		return
	}
	market := &tsList{m: game.MarketIndex, mode: 1}
	ms.Status |= 0x80
	s.Persons[ms.Person].Owner = mi
	if got := market.nameColor(s, ms, mi, 0); got != ColBeige {
		t.Errorf("own listed player with an offer: name colour %d", got)
	}
	s.Persons[ms.Person].Owner = game.MarketIndex
	if got := market.nameColor(s, ms, mi, 0); got == ColBeige {
		t.Error("a player of the market is marked")
	}
}
