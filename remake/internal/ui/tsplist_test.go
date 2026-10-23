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
