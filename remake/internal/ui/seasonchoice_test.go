package ui

import (
	"testing"

	"github.com/schowave/bmp/remake/internal/game"
	"github.com/schowave/bmp/remake/internal/seasons"
)

func TestSeasonChoiceHover(t *testing.T) {
	c := &SeasonChoice{entries: make([]seasons.Season, 2*seasonRows+2)}
	for _, h := range []struct{ x, y, want int }{
		{seasonLeftX, seasonTopY, 0},
		{seasonLeftX + 5, seasonTopY + fileRowH, 1},
		{seasonLeftX + seasonColW, seasonTopY, seasonRows},
		{seasonLeftX + 2*seasonColW, seasonTopY + fileRowH, 2*seasonRows + 1},
		{seasonLeftX + 2*seasonColW, seasonTopY + 2*fileRowH, -1}, // after the last entry
		{seasonLeftX + 3*seasonColW, seasonTopY, -1},
		{seasonLeftX - 1, seasonTopY, -1},
	} {
		if got := c.hover(&Input{X: h.x, Y: h.y}); got != h.want {
			t.Errorf("(%d,%d): %d, want %d", h.x, h.y, got, h.want)
		}
	}
	c = &SeasonChoice{entries: []seasons.Season{seasons.ForYear(1999)}}
	if r := c.rows(-1); r[0].text != "1999/00" {
		t.Errorf("label %q", r[0].text)
	}
}

// The key S in the setup opens the season choice; a season or the
// original goes back to the setup with their clubs and the names typed so
// far, a right click goes back unchanged.
func TestSetupKeyOpensTheSeasonChoice(t *testing.T) {
	app, mana := testApp(t)
	setup := NewSetup(mana, t.TempDir())
	setup.seasons = []seasons.Season{{ID: "1995-96", Year: 1995}}
	setup.names[0], setup.teams[0] = "TEST", setup.center
	app.Reset(setup)
	step(t, app, Input{Chars: []rune{'s'}})
	if _, ok := top(app).(*SeasonChoice); !ok {
		t.Fatalf("key S: %T", top(app))
	}
	step(t, app, rclick())
	if top(app) != setup {
		t.Fatalf("right click: %T", top(app))
	}
	step(t, app, Input{Chars: []rune{'S'}})
	step(t, app, click(seasonLeftX+2, seasonTopY+2))
	s, ok := top(app).(*Setup)
	if !ok || s.opts.Season != "1995-96" || s.orig != mana || s.names[0] != "TEST" || s.teams[0] != -1 || len(s.seasons) != 1 {
		t.Fatalf("season: %T %+v", top(app), s)
	}
	if l, r := s.seasonHint(); l != "SAISON 1995/96" || r != "S: SAISON WÄHLEN" {
		t.Errorf("hint %q %q", l, r)
	}
	step(t, app, Input{Chars: []rune{'s'}})
	step(t, app, click(originalButton.X+2, originalButton.Y+2))
	s = top(app).(*Setup)
	if s.opts.Season != "" || s.mana != mana || s.names[0] != "TEST" {
		t.Fatalf("original: %+v", s)
	}
	if l, _ := s.seasonHint(); l != "ORIGINAL 1991/92" {
		t.Errorf("hint %q", l)
	}
	// While a name is typed, S is a letter of it.
	s.edit = 1
	step(t, app, Input{Chars: []rune{'S'}})
	if top(app) != s || s.names[1] != "S" {
		t.Fatalf("typing: %T %q", top(app), s.names[1])
	}
}

// Without built-in seasons S does nothing and the setup shows no hint.
func TestSetupWithoutSeasons(t *testing.T) {
	app, mana := testApp(t)
	setup := NewSetup(mana, t.TempDir())
	if l, r := setup.seasonHint(); l != "" || r != "" {
		t.Errorf("hint %q %q", l, r)
	}
	app.Reset(setup)
	step(t, app, Input{Chars: []rune{'s'}})
	if top(app) != setup {
		t.Fatalf("key S: %T", top(app))
	}
}

// TestSeasonSetupStartsTheSeason starts a game with the clubs of the
// original as a stand-in for a real season.
func TestSeasonSetupStartsTheSeason(t *testing.T) {
	app, mana := testApp(t)
	setup := NewSeasonSetup(mana, mana, t.TempDir(), seasons.ForYear(2008))
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
		t.Fatal("save:", err)
	}
	l, err := LoadGame(data, mana)
	if err != nil || l.Season != "2008-09" {
		t.Fatalf("load: %v", err)
	}
	if id, _ := game.SaveSeason(data); id != "2008-09" {
		t.Fatalf("saved season %q", id)
	}
}
