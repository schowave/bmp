package game

import (
	"testing"

	"github.com/schowave/bmp/remake/internal/seasons"
)

// TestRealSeasonsPlay plays one season with the clubs of real seasons of
// each era: the Oberliga, the Regionalliga, the 3. Liga, the running one.
func TestRealSeasonsPlay(t *testing.T) {
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
