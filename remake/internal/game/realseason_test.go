package game

import (
	"testing"

	"github.com/schowave/bmp/remake/internal/seasons"
)

// TestRealSeasonsPlay plays one season with the clubs of real seasons of
// each era: the Oberliga, the Regionalliga, the 3. Liga, the running one.
func TestRealSeasonsPlay(t *testing.T) {
	if testing.Short() {
		t.Skip("plays whole seasons")
	}
	for _, s := range seasons.List() {
		if s.Year != 1990 && s.Year != 1995 && s.Year != 2008 && s.Year != 2026 {
			continue
		}
		t.Run(s.ID, func(t *testing.T) {
			m, err := seasons.Mana(s.ID)
			if err != nil {
				t.Fatal(err)
			}
			g, err := NewGame(m, Options{Managers: []NewManager{{Name: "A", Team: 0}, {Name: "B", Team: 40}},
				Level: 2, Seed: 7, Season: s.ID, Year: s.Year})
			if err != nil {
				t.Fatal(err)
			}
			g.Start()
			autoplay(t, g, 1)
			if g.StartYear != s.Year+1 {
				t.Fatalf("after one season: %d", g.StartYear)
			}
			for p := 1; p < NumPersons; p++ {
				if g.Persons[p].Name == "" {
					t.Fatalf("person %d without a name", p)
				}
			}
		})
	}
}

// TestRealSeasonBeyondTheLast plays from the last built-in season into years
// no season covers, saving and loading on the way.
func TestRealSeasonBeyondTheLast(t *testing.T) {
	if testing.Short() {
		t.Skip("plays whole seasons")
	}
	list := seasons.List()
	last := list[len(list)-1]
	m, err := seasons.Mana(last.ID)
	if err != nil {
		t.Fatal(err)
	}
	g, err := NewGame(m, Options{Managers: []NewManager{{Name: "A", Team: 20}}, Level: 2, Seed: 3, Season: last.ID, Year: last.Year})
	if err != nil {
		t.Fatal(err)
	}
	g.Start()
	for k := 0; k < 3; k++ {
		autoplay(t, g, 1)
		for g.Loop.Phase != PhaseTurn {
			autoplayStep(g)
		}
		data, err := g.Save()
		if err != nil {
			t.Fatal(err)
		}
		if g, err = Load(data, m); err != nil {
			t.Fatal(err)
		}
	}
	if g.StartYear != last.Year+3 || g.Season != last.ID || g.PointsPerWin() != 3 || g.LeagueName(2) != "3. Liga" {
		t.Fatalf("year %d season %q points %d league %q", g.StartYear, g.Season, g.PointsPerWin(), g.LeagueName(2))
	}
}
