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

// In the squad list of the transfer screen a player with an offer of a club
// is white; in the contract list he keeps his colour.
func TestOfferRowWhite(t *testing.T) {
	_, ss := tsTestSession(t)
	s := ss.S
	sl := &s.Squads[ss.Mgr()][3]
	squad, contracts := &tsList{m: ss.Mgr(), mode: 0}, &tsList{m: ss.Mgr(), mode: 3}
	plain := squad.rowColor(s, sl)
	sl.Status |= 0x40
	if got := squad.rowColor(s, sl); got != ColWhite || plain == ColWhite {
		t.Errorf("colour %d with an offer, %d without", got, plain)
	}
	if got := contracts.rowColor(s, sl); got == ColWhite {
		t.Error("the contract list marks the offer too")
	}
}

// An own listed player with an offer is white in the market list, a player
// of somebody else with an offer is not.
func TestListedOfferRowWhite(t *testing.T) {
	_, ss := tsTestSession(t)
	s := ss.S
	sl := &s.Squads[game.MarketIndex][0]
	if sl.Empty() {
		t.Skip("empty market")
	}
	market := &tsList{m: game.MarketIndex, viewer: ss.Mgr(), mode: 1}
	sl.Status |= 0x80
	s.Persons[sl.Person].Owner = ss.Mgr()
	if got := market.rowColor(s, sl); got != ColWhite {
		t.Errorf("own listed player with an offer: colour %d", got)
	}
	s.Persons[sl.Person].Owner = game.MarketIndex
	if got := market.rowColor(s, sl); got == ColWhite {
		t.Error("a player of the market is white")
	}
}
