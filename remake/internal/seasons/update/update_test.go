package update

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/schowave/bmp/remake/internal/assets"
)

func club(name, league string, rank int) Club {
	c := Club{Name: name, League: league, Rank: rank, Games: 34, Goals: fmt.Sprintf("%d:%d", 60-rank, 20+rank), Points: 70 - 2*rank}
	c.Players.Goalkeepers = []string{"KEEPER" + name[:1], "SECOND"}
	return c
}

// previousSeason is a season of 64 German clubs G0..G63 with logos 1000+i
// and 136 foreign ones.
func previousSeason() *assets.Mana {
	m := &assets.Mana{Clubs: make([]assets.ClubRecord, assets.NumClubs)}
	for i := range m.Clubs {
		m.Clubs[i] = assets.ClubRecord{Name: fmt.Sprintf("FOREIGN %d", i), Strength: [3]int{50, 50, 50}}
		if i < numGerman {
			m.Clubs[i] = assets.ClubRecord{Name: fmt.Sprintf("CLUB %d", i), Logo: 1000 + i, Strength: [3]int{60, 60, 60},
				Players: make([]string, 20)}
		}
	}
	m.Clubs[0].Name = "WERDER BREMEN"
	return m
}

// A season since 2008: 18 + 18 + 20 clubs in the file, the rest from the
// season before; names, logos and foreign clubs carry over.
func TestSeason(t *testing.T) {
	var clubs []Club
	for i := 1; i <= 18; i++ {
		clubs = append(clubs, club(fmt.Sprintf("FIRST %d", i), "L1", i))
		clubs = append(clubs, club(fmt.Sprintf("SECOND %d", i), "L2", i))
	}
	for i := 1; i <= 20; i++ {
		clubs = append(clubs, club(fmt.Sprintf("THIRD %d", i), "L3", i))
	}
	clubs[0].Name = "SV WERDER BREMEN" // FIRST 1 under the name of Transfermarkt
	prev := previousSeason()
	out, err := Season(Input{Year: 2026, Clubs: clubs, Previous: prev, Known: []*assets.Mana{prev}, NextLogo: 2000})
	if err != nil {
		t.Fatal(err)
	}
	c := out.Mana.Clubs
	if c[0].Name != "WERDER BREMEN" || c[0].Logo != 1000 || c[0].Players[0] != "KEEPERF" {
		t.Errorf("club 0: %+v", c[0])
	}
	if c[18].Name != "SECOND 1" || c[36].Name != "THIRD 1" || c[37].Name != "THIRD 2" || c[38].Name != "THIRD 3" {
		t.Errorf("leagues 2 and 3 start with %q, %q, %q, %q", c[18].Name, c[36].Name, c[37].Name, c[38].Name)
	}
	// The 18 clubs of the third tier left, then those of the season before.
	if c[56].Name != "CLUB 1" || c[63].Name != "CLUB 8" || c[56].Logo != 1001 {
		t.Errorf("filled: %q %q (logo %d)", c[56].Name, c[63].Name, c[56].Logo)
	}
	if len(out.Logos) != 55 || c[1].Logo != 2000 {
		t.Errorf("%d new logos, club 1 logo %d", len(out.Logos), c[1].Logo)
	}
	if c[numGerman].Name != prev.Clubs[numGerman].Name || c[assets.NumClubs-1].Name != prev.Clubs[assets.NumClubs-1].Name {
		t.Error("foreign clubs not taken over")
	}
	if c[0].Strength[1] != strengthRange[0][1] || c[17].Strength[1] != strengthRange[0][0] {
		t.Errorf("strengths %v %v", c[0].Strength, c[17].Strength)
	}
	if _, err := assets.EncodeClubs(out.Mana); err != nil {
		t.Fatal(err)
	}
}

func TestMatchNames(t *testing.T) {
	known := []string{"WERDER BREMEN", "1.FC HEIDENHEIM", "PREUßEN MÜNSTER", "VFB STUTTGART", "VFB STUTTGART II", "FSV FRANKFURT"}
	got := matchNames([]string{"SV WERDER BREMEN", "1.FC HEIDENHEIM 1846", "PREUSSEN MÜNSTER", "VFB STUTTGART II",
		"VFB STUTTGART", "FC VORWÄRTS FRANKFURT/ODER"}, known)
	want := map[string]string{"SV WERDER BREMEN": "WERDER BREMEN", "1.FC HEIDENHEIM 1846": "1.FC HEIDENHEIM",
		"PREUSSEN MÜNSTER": "PREUßEN MÜNSTER", "VFB STUTTGART II": "VFB STUTTGART II", "VFB STUTTGART": "VFB STUTTGART"}
	if len(got) != len(want) {
		t.Errorf("got %v", got)
	}
	for k, v := range want {
		if got[k] != v {
			t.Errorf("%q: %q, want %q", k, got[k], v)
		}
	}
}

func TestReadClubsWantsTheTable(t *testing.T) {
	f := filepath.Join(t.TempDir(), "2026-27.json")
	if err := os.WriteFile(f, []byte(`[{"name": "A", "league": "L1", "players": {}}]`), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := ReadClubs(f); err == nil {
		t.Fatal("a file without table places was read")
	}
}
