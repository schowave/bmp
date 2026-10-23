package ui

import (
	"testing"

	"github.com/schowave/bmp/remake/internal/game"
)

// During the relegation (from day 0x13b) the match screens of cup slot 1
// are titled "Relegation" like f_17c2_1ccb, not with the round of the
// Landesmeister whose slots it borrows.
func TestCupScreenRelegationTitle(t *testing.T) {
	_, mana := testApp(t)
	s, err := game.NewGame(mana, game.Options{Managers: []game.NewManager{{Name: "TEST", Team: 9}}, Level: 2, Seed: 1})
	if err != nil {
		t.Fatal(err)
	}
	ss := &Session{S: s, Mana: mana}
	s.Cups.Round[game.CupLandesmeister] = 5
	s.Day = 0x13b
	if _, title := NewCupRound(ss, game.CupLandesmeister, 45, nil).layout(); title != "Relegation    45.Minute" {
		t.Errorf("relegation: title %q", title)
	}
	s.Day = 0x13a
	if _, title := NewCupRound(ss, game.CupLandesmeister, 45, nil).layout(); title != game.RoundNames[5]+" "+game.CupNames[game.CupLandesmeister]+"    45.Minute" {
		t.Errorf("final: title %q", title)
	}
}

// A foreign club has no logo: its scoreboard shows the trophy of the cup
// (PIC/0x90+cup, f_1a58_042f), a German club its own logo.
func TestFixtureLogo(t *testing.T) {
	_, mana := testApp(t)
	s, err := game.NewGame(mana, game.Options{Managers: []game.NewManager{{Name: "TEST", Team: 9}}, Level: 2, Seed: 1})
	if err != nil {
		t.Fatal(err)
	}
	f := &game.Fixture{League: -1, Cup: game.CupUEFA, Home: 9, Away: game.NumGerman + 3}
	if got := fixtureLogo(s, f, f.Home); got != s.Teams[9].Logo {
		t.Errorf("German club: picture %d, want %d", got, s.Teams[9].Logo)
	}
	if got := fixtureLogo(s, f, f.Away); got != 147 {
		t.Errorf("foreign club in the UEFA-Pokal: picture %d, want 147", got)
	}
}
