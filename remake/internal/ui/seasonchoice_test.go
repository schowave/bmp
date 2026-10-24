package ui

import (
	"testing"

	"github.com/schowave/bmp/remake/internal/game"
	"github.com/schowave/bmp/remake/internal/seasons"
)

func TestSeasonChoiceHover(t *testing.T) {
	c := &SeasonChoice{entries: make([]seasons.Season, 38)}
	for _, h := range []struct{ x, y, want int }{
		{seasonLeftX, seasonTopY, 0},
		{seasonLeftX + 5, seasonTopY + fileRowH, 1},
		{seasonLeftX + seasonColW, seasonTopY, seasonRows},
		{seasonLeftX + 2*seasonColW, seasonTopY + fileRowH, 2*seasonRows + 1},
		{seasonLeftX + 2*seasonColW, seasonTopY + 5*fileRowH, -1}, // after the last entry
		{seasonLeftX - 1, seasonTopY, -1},
	} {
		if got := c.hover(&Input{X: h.x, Y: h.y}); got != h.want {
			t.Errorf("(%d,%d): %d, want %d", h.x, h.y, got, h.want)
		}
	}
	if seasonLabel(seasons.Season{}) != "ORIGINAL (1991/92)" || seasonLabel(seasons.Season{ID: "1999-00", Year: 1999}) != "SAISON 1999/00" {
		t.Error("labels")
	}
}

func TestSeasonChoiceOriginalOpensTheSetup(t *testing.T) {
	app, mana := testApp(t)
	app.Reset(&SeasonChoice{mana: mana, saveDir: t.TempDir(), entries: []seasons.Season{{}, {ID: "1995-96", Year: 1995}}})
	step(t, app, click(seasonLeftX+2, seasonTopY+2))
	s, ok := top(app).(*Setup)
	if !ok || s.opts.Season != "" || s.orig != mana {
		t.Fatalf("got %T", top(app))
	}
}

// TestSeasonSetupStartsTheSeason starts a game with the clubs of the
// original as a stand-in for a real season.
func TestSeasonSetupStartsTheSeason(t *testing.T) {
	app, mana := testApp(t)
	setup := NewSeasonSetup(mana, mana, t.TempDir(), "2008-09", 2008)
	app.Reset(setup)
	setup.names[0], setup.teams[0] = "TEST", setup.center
	step(t, app, rclick())
	if _, ok := top(app).(*setupOptions); !ok {
		t.Fatalf("no options: %T", top(app))
	}
	// The historic start is off for a real season.
	step(t, app, click(optLeftX+2, optLengthY+3*18+2))
	if setup.opts.Historic {
		t.Fatal("historic start switched on")
	}
	step(t, app, rclick())
	d, ok := app.stack[0].(*Driver)
	if !ok {
		t.Fatalf("game did not start: %T", top(app))
	}
	s := d.ss.S
	if s.Season != "2008-09" || s.StartYear != 2008 || s.LeagueName(2) != "3. Liga" || d.ss.Mana != mana {
		t.Fatalf("season %q year %d", s.Season, s.StartYear)
	}
	data, err := s.Save()
	if err != nil {
		t.Skip("not in a turn yet:", err)
	}
	l, err := LoadGame(data, mana)
	if err != nil || l.Season != "2008-09" {
		t.Fatalf("load: %v", err)
	}
	if id, _ := game.SaveSeason(data); id != "2008-09" {
		t.Fatalf("saved season %q", id)
	}
}
