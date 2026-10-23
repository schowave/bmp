package game

import (
	"encoding/json"
	"testing"
)

func seasonGame(t *testing.T, season string, year int, length GameLength) *State {
	t.Helper()
	s, err := NewGame(mana(t), Options{Managers: []NewManager{{Name: "TEST", Team: 9}}, Level: 2, Seed: 1,
		Length: length, Season: season, Year: year})
	if err != nil {
		t.Fatal(err)
	}
	return s
}

func TestSeasonStartYearAndLength(t *testing.T) {
	for _, c := range []struct {
		season  string
		year    int
		length  GameLength
		start   int
		end     int
		endless bool
		file    string
	}{
		{"", 0, OneYear, 1992, 1993, false, "HIGH.12"},
		{"", 0, ThreeYears, 1992, 1995, false, "HIGH.32"},
		{"", 0, Endless, 1992, 22251, true, "HIGH.02"},
		{"2010-11", 2010, OneYear, 2010, 2011, false, "HIGH.12"},
		{"2010-11", 2010, ThreeYears, 2010, 2013, false, "HIGH.32"},
		{"1990-91", 1990, Endless, 1990, 22249, true, "HIGH.02"},
	} {
		s := seasonGame(t, c.season, c.year, c.length)
		if s.StartYear != c.start || s.FirstYear != c.start || s.EndYear != c.end || s.Endless() != c.endless ||
			s.Length() != c.length || s.HighscoreFile() != c.file || s.Season != c.season || !s.Modern {
			t.Errorf("%+v: start %d first %d end %d endless %v length %v file %s", c, s.StartYear, s.FirstYear,
				s.EndYear, s.Endless(), s.Length(), s.HighscoreFile())
		}
	}
	h, err := NewGame(mana(t), Options{Managers: []NewManager{{Name: "TEST", Team: 9}}, Historic: true, Length: ThreeYears})
	if err != nil {
		t.Fatal(err)
	}
	if h.StartYear != 1963 || h.EndYear != 1966 || h.Length() != ThreeYears {
		t.Errorf("historic: %d..%d %v", h.StartYear, h.EndYear, h.Length())
	}
}

func TestPointsPerWin(t *testing.T) {
	for _, c := range []struct {
		season string
		year   int
		want   int
	}{{"", 1992, 2}, {"", 1995, 3}, {"", 2010, 3}, {"1994-95", 1994, 2}, {"1995-96", 1995, 3}, {"2025-26", 2025, 3}} {
		if got := (&State{Season: c.season, StartYear: c.year}).PointsPerWin(); got != c.want {
			t.Errorf("%+v: %d", c, got)
		}
	}
	s := seasonGame(t, "1995-96", 1995, Endless)
	pairs := []Pair{{Home: 0, Away: 1}, {Home: 2, Away: 3}}
	s.applyResults(pairs, []Score{{Home: 2, Away: 0}, {Home: 1, Away: 1}}, true)
	if s.Clubs[0].TotalPoints() != 3 || s.Clubs[1].TotalPoints() != 0 || s.Clubs[2].TotalPoints() != 1 {
		t.Fatalf("points %d %d %d", s.Clubs[0].TotalPoints(), s.Clubs[1].TotalPoints(), s.Clubs[2].TotalPoints())
	}
}

func TestSaveKeepsTheSeason(t *testing.T) {
	s := seasonGame(t, "2010-11", 2010, OneYear)
	s.Start()
	data, err := s.Save()
	if err != nil {
		t.Fatal(err)
	}
	if id, err := SaveSeason(data); err != nil || id != "2010-11" {
		t.Fatalf("got %q, %v", id, err)
	}
	l, err := Load(data, mana(t))
	if err != nil || l.Season != "2010-11" || l.FirstYear != 2010 || l.Length() != OneYear {
		t.Fatalf("got %+v, %v", l.Season, err)
	}
	// A save from before the seasons gets the first year of its start.
	old, _ := json.Marshal(saveFile{Version: saveVersion, State: &State{Modern: true, EndYear: 1995}})
	o, err := Load(old, nil)
	if err != nil || o.FirstYear != 1992 || o.Length() != ThreeYears {
		t.Fatalf("old save: first year %d, length %v, %v", o.FirstYear, o.Length(), err)
	}
	if id, err := SaveSeason(old); err != nil || id != "" {
		t.Fatalf("old save season %q, %v", id, err)
	}
}
