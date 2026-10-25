package update

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/schowave/bmp/remake/internal/assets"
	"github.com/schowave/bmp/remake/internal/seasons"
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

// Before the first game the tables say nothing: the clubs rank by the
// season before, the league they played in first, then their strength
// there; a club down from the league above leads, one up from below trails.
func TestSeasonBeforeFirstGame(t *testing.T) {
	prev := previousSeason()
	for i := range prev.Clubs[:numGerman] {
		prev.Clubs[i].Name = fmt.Sprintf("CLUB %d", i)
		prev.Clubs[i].Strength = [3]int{90 - i, 90 - i, 90 - i}
	}
	var l1, l2, l3 []int
	for i := 0; i <= 16; i++ {
		l1 = append(l1, i)
	}
	l1 = append(l1, 20)    // up from the second league
	l2 = []int{17, 18, 19} // CLUB 17 down from the first
	for i := 21; i <= 35; i++ {
		l2 = append(l2, i)
	}
	for i := 36; i <= 55; i++ {
		l3 = append(l3, i)
	}
	var clubs []Club
	for _, lg := range []struct {
		code string
		ids  []int
	}{{"L1", l1}, {"L2", l2}, {"L3", l3}} {
		for k, i := range lg.ids {
			c := club(fmt.Sprintf("CLUB %d", i), lg.code, len(lg.ids)-k) // the places of Transfermarkt say nothing
			c.Games, c.Goals, c.Points = 0, "0:0", 0
			clubs = append(clubs, c)
		}
	}
	out, err := Season(Input{Year: 2027, Clubs: clubs, Previous: prev, Known: []*assets.Mana{prev}, NextLogo: 2000})
	if err != nil {
		t.Fatal(err)
	}
	c := out.Mana.Clubs
	if c[0].Name != "CLUB 0" || c[17].Name != "CLUB 20" || c[18].Name != "CLUB 17" || c[38].Name != "CLUB 38" {
		t.Errorf("order %q %q %q %q", c[0].Name, c[17].Name, c[18].Name, c[38].Name)
	}
	for _, x := range []struct {
		i, want int
	}{{0, strengthRange[0][1]}, {17, strengthRange[0][0]}, {18, strengthRange[1][1]}, {38, strengthRange[2][1]}} {
		if c[x.i].Strength != [3]int{x.want, x.want, x.want} {
			t.Errorf("club %d (%s): strengths %v, want %d", x.i, c[x.i].Name, c[x.i].Strength, x.want)
		}
	}
}

func TestMatchNames(t *testing.T) {
	known := []string{"WERDER BREMEN", "1.FC HEIDENHEIM", "PREUßEN MÜNSTER", "VFB STUTTGART", "VFB STUTTGART II", "FSV FRANKFURT"}
	got := matchNames([]string{"SV WERDER BREMEN", "1.FC HEIDENHEIM 1846", "PREUSSEN MÜNSTER", "VFB STUTTGART II",
		"VFB STUTTGART", "FC VORWÄRTS FRANKFURT/ODER"}, known, false)
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

// In another season a name only matches the same club: "RB LEIPZIG" is not
// the "VFB LEIPZIG" of earlier seasons.
func TestMatchNamesStrict(t *testing.T) {
	got := matchNames([]string{"RB LEIPZIG", "SG DYNAMO DRESDEN", "SV 07 ELVERSBERG"},
		[]string{"VFB LEIPZIG", "DYNAMO DRESDEN", "SV ELVERSBERG"}, true)
	if len(got) != 2 || got["SG DYNAMO DRESDEN"] != "DYNAMO DRESDEN" || got["SV 07 ELVERSBERG"] != "SV ELVERSBERG" {
		t.Errorf("got %v", got)
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

// From 2008 on the Regionalliga (tier 4) fills the places outside the
// leagues, and a European cup file the foreign clubs.
func TestSeasonWithRegionalligaAndCups(t *testing.T) {
	var clubs []Club
	for i := 1; i <= 18; i++ {
		clubs = append(clubs, club(fmt.Sprintf("FIRST %d", i), "L1", i), club(fmt.Sprintf("SECOND %d", i), "L2", i))
	}
	for i := 1; i <= 20; i++ {
		clubs = append(clubs, club(fmt.Sprintf("THIRD %d", i), "L3", i))
	}
	for i := 1; i <= 4; i++ {
		north, south := club(fmt.Sprintf("NORTH %d", i), "RLN", i), club(fmt.Sprintf("SOUTH %d", i), "RLS", i)
		north.Tier, south.Tier = 4, 4
		clubs = append(clubs, north, south)
	}
	cups := []Foreign{{Name: "Real Madrid", Cup: "CL"}, {Name: "FC Liverpool", Cup: "CL"}, {Name: "Ajax Amsterdam", Cup: "EL"}}
	prev := previousSeason()
	out, err := Season(Input{Year: 2026, Clubs: clubs, Foreign: cups, Previous: prev, Known: []*assets.Mana{prev}, NextLogo: 2000})
	if err != nil {
		t.Fatal(err)
	}
	c := out.Mana.Clubs
	// 18 of the 3. Liga stay, then the Regionalliga by place: the winners first.
	if c[56].Name != "NORTH 1" || c[57].Name != "SOUTH 1" || c[63].Name != "SOUTH 4" {
		t.Errorf("outside: %q %q %q", c[56].Name, c[57].Name, c[63].Name)
	}
	f := c[numGerman:]
	if f[0].Name != "REAL MADRID" || f[1].Name != "FC LIVERPOOL" || f[2].Name != "AJAX AMSTERDAM" || f[3].Name != prev.Clubs[numGerman].Name {
		t.Errorf("foreign: %q %q %q %q", f[0].Name, f[1].Name, f[2].Name, f[3].Name)
	}
	if f[0].Strength[0] != foreignStrength["CL"][1] || f[1].Strength[0] != foreignStrength["CL"][0] {
		t.Errorf("strengths %v %v", f[0].Strength, f[1].Strength)
	}
}

// TestRealFile updates 2026/27 from a file the Transfermarkt script wrote:
// its clubs keep the names and logos they have in the seasons.
func TestRealFile(t *testing.T) {
	clubs, err := ReadClubs(filepath.Join("testdata", "2026-27.json"))
	if err != nil {
		t.Fatal(err)
	}
	var known []*assets.Mana
	for _, s := range seasons.List() {
		m, err := seasons.Mana(s.ID)
		if err != nil {
			t.Fatal(err)
		}
		known = append(known, m)
	}
	prev, err := seasons.Mana("2025-26")
	if err != nil {
		t.Fatal(err)
	}
	out, err := Season(Input{Year: 2026, Clubs: clubs, Previous: prev, Known: known, NextLogo: 5000})
	if err != nil {
		t.Fatal(err)
	}
	if len(out.Logos) != 0 {
		t.Errorf("%d new logos: a club of the file did not find its name in the seasons", len(out.Logos))
	}
	names := map[string]bool{}
	for _, c := range out.Mana.Clubs[:numGerman] {
		if names[c.Name] {
			t.Errorf("%s twice", assets.FromCP437(c.Name))
		}
		names[c.Name] = true
	}
	// The clubs of the 1. and 2. Bundesliga of the file are the first 36.
	n := 0
	for _, c := range clubs {
		if c.League == "L1" || c.League == "L2" {
			n++
		}
	}
	if n != league1+league1 {
		t.Fatalf("%d clubs in L1 and L2", n)
	}
	for i, c := range out.Mana.Clubs[:36] {
		if c.Players[0] == "" {
			t.Errorf("club %d (%s) has no goalkeeper", i, assets.FromCP437(c.Name))
		}
	}
}

// A squads file changes only the players of the clubs it names, in the
// slots of MANA.DAT, and leaves foreign clubs alone.
func TestSetSquads(t *testing.T) {
	m := previousSeason()
	before := m.Clubs[1]
	squads := map[string]Squad{
		"WERDER BREMEN": {Goalkeepers: []string{"REINKE", "BORel", "THIRD"}, Forwards: []string{"AILTON", "KLASNIĆ"}},
		"FOREIGN 100":   {Goalkeepers: []string{"NOBODY"}},
	}
	n, err := SetSquads(m, squads)
	if err != nil {
		t.Fatal(err)
	}
	if n != 1 {
		t.Errorf("changed %d clubs, want 1", n)
	}
	p := m.Clubs[0].Players
	if len(p) != 20 || p[0] != "REINKE" || p[1] != "BOREL" || p[2] != "" || p[15] != "AILTON" || p[16] != "KLASNIC" {
		t.Errorf("players %q", p)
	}
	if fmt.Sprint(m.Clubs[1]) != fmt.Sprint(before) || m.Clubs[100].Players != nil {
		t.Errorf("other clubs changed: %v, %v", m.Clubs[1], m.Clubs[100])
	}
}

// A season with a squads file takes the squads by games for the clubs it
// names, under their names in the season, and those of the file for the rest.
func TestSeasonWithSquads(t *testing.T) {
	clubs := []Club{club("SV WERDER BREMEN", "L1", 1), club("FIRST 2", "L1", 2)}
	prev := previousSeason()
	squads := map[string]Squad{"WERDER BREMEN": {Goalkeepers: []string{"REINKE"}}}
	out, err := Season(Input{Year: 1997, Clubs: clubs, Previous: prev, Known: []*assets.Mana{prev}, NextLogo: 2000, Squads: squads})
	if err != nil {
		t.Fatal(err)
	}
	c := out.Mana.Clubs
	if c[0].Name != "WERDER BREMEN" || c[0].Players[0] != "REINKE" || c[0].Players[1] != "" {
		t.Errorf("club 0: %+v", c[0])
	}
	if c[1].Name != "FIRST 2" || c[1].Players[0] != "KEEPERF" {
		t.Errorf("club 1: %+v", c[1])
	}
}

// A club missing in the season before keeps the name it had in the season
// as it was, else in the latest season that has it, though another season
// has it under the name of Transfermarkt.
func TestSeasonKeepsEarlierName(t *testing.T) {
	clubs := []Club{club("SG DYNAMO DRESDEN", "L1", 1), club("FC VIKTORIA KÖLN", "L1", 2)}
	prev, before, older, newest := previousSeason(), previousSeason(), previousSeason(), previousSeason()
	cp := func(s string) string {
		b, err := assets.ToCP437(s)
		if err != nil {
			t.Fatal(err)
		}
		return b
	}
	before.Clubs[6].Name = cp("VIKTORIA KÖLN")
	older.Clubs[5].Name = "DYNAMO DRESDEN"
	newest.Clubs[5].Name, newest.Clubs[6].Name = "SG DYNAMO DRESDEN", cp("FC VIKTORIA KÖLN")
	out, err := Season(Input{Year: 2000, Clubs: clubs, Previous: prev, Before: before, Earlier: []*assets.Mana{older},
		Known: []*assets.Mana{older, prev, before, newest}, NextLogo: 2000})
	if err != nil {
		t.Fatal(err)
	}
	c := out.Mana.Clubs
	if c[0].Name != "DYNAMO DRESDEN" || c[0].Logo != 1005 || assets.FromCP437(c[1].Name) != "VIKTORIA KÖLN" {
		t.Errorf("clubs %s (logo %d), %s", c[0].Name, c[0].Logo, assets.FromCP437(c[1].Name))
	}
}

// Before 2008 the second teams of the third tier come after every other
// club, also after those of the season before; from 2008 on they rank by
// their table like any club.
func TestSecondTeamsBefore2008(t *testing.T) {
	var clubs []Club
	for i := 1; i <= 18; i++ {
		clubs = append(clubs, club(fmt.Sprintf("FIRST %d", i), "L1", i), club(fmt.Sprintf("SECOND %d", i), "L2", i))
	}
	for i := 1; i <= 18; i++ {
		north, south := club(fmt.Sprintf("NORTH %d", i), "RN", i), club(fmt.Sprintf("SOUTH %d", i), "RS", i)
		clubs = append(clubs, north, south)
	}
	clubs[36].Name = "WERDER BREMEN II" // NORTH 1
	clubs[37].Name = "BAYERN AMATEURE"  // SOUTH 1
	for _, year := range []int{2000, 2008} {
		out, err := Season(Input{Year: year, Clubs: clubs, Previous: previousSeason(), Known: []*assets.Mana{previousSeason()}, NextLogo: 2000})
		if err != nil {
			t.Fatal(err)
		}
		in := map[string]int{}
		for i, c := range out.Mana.Clubs[:numGerman] {
			in[assets.FromCP437(c.Name)] = i
		}
		_, bremen := in["WERDER BREMEN II"]
		_, bayern := in["BAYERN AMATEURE"]
		if year < secondTeamsFrom && (bremen || bayern) {
			t.Errorf("%d: second teams on places %v", year, in)
		}
		if year >= secondTeamsFrom && (!bremen || !bayern) {
			t.Errorf("%d: second teams missing", year)
		}
	}
}

// A second team without a logo of its own wears the logo of its first team.
func TestFirstTeam(t *testing.T) {
	known := []string{"HAMBURGER SV", "FC BAYERN MÜNCHEN", "TSV 1860 MÜNCHEN", "VFB STUTTGART II"}
	for name, want := range map[string]string{"HAMBURGER SV II": "HAMBURGER SV", "BAYERN MÜNCHEN AMATEURE": "FC BAYERN MÜNCHEN",
		"TSV 1860 MÜNCHEN II": "TSV 1860 MÜNCHEN", "VFB STUTTGART II": "", "HAMBURGER SV": "", "FC AUGSBURG II": ""} {
		if got := firstTeam(name, known); got != want {
			t.Errorf("%q: %q, want %q", name, got, want)
		}
	}
	clubs := []Club{club("CLUB 3 II", "L1", 1)}
	prev := previousSeason()
	out, err := Season(Input{Year: 2010, Clubs: clubs, Previous: prev, Known: []*assets.Mana{prev}, NextLogo: 2000})
	if err != nil {
		t.Fatal(err)
	}
	if c := out.Mana.Clubs[0]; c.Logo != 1003 || len(out.Logos) != 0 {
		t.Errorf("CLUB 3 II: logo %d, %d new logos", c.Logo, len(out.Logos))
	}
}

// A second team does not move up into the 2. Bundesliga, though it wins the
// 3. Liga, nor one whose name the game cuts: the next clubs of the 3. Liga
// take the two places.
func TestSecondTeamStaysDown(t *testing.T) {
	var clubs []Club
	for i := 1; i <= 18; i++ {
		clubs = append(clubs, club(fmt.Sprintf("FIRST %d", i), "L1", i), club(fmt.Sprintf("SECOND %d", i), "L2", i))
	}
	for i := 1; i <= 20; i++ {
		clubs = append(clubs, club(fmt.Sprintf("THIRD %d", i), "L3", i))
	}
	clubs[36].Name = "FC BAYERN II"              // THIRD 1
	clubs[37].Name = "RASENBALLSPORT LEIPZIG II" // THIRD 2, a name cut to "…LEIPZIG"
	out, err := Season(Input{Year: 2019, Clubs: clubs, Previous: previousSeason(), Known: []*assets.Mana{previousSeason()}, NextLogo: 2000})
	if err != nil {
		t.Fatal(err)
	}
	c := out.Mana.Clubs
	if c[36].Name != "THIRD 3" || c[37].Name != "THIRD 4" || c[38].Name != "FC BAYERN II" {
		t.Errorf("places 36..38: %q %q %q", c[36].Name, c[37].Name, c[38].Name)
	}
}

// A club the season keeps under another name than the season before is
// still the club of the season before, and does not fill a place twice.
func TestSeasonNoClubTwice(t *testing.T) {
	clubs := []Club{club("ROT-WEISS ERFURT", "L1", 1)}
	prev, before := previousSeason(), previousSeason()
	prev.Clubs[5].Name = "FC ROT-WEISS ERFURT"
	before.Clubs[9].Name = "ROT-WEISS ERFURT"
	out, err := Season(Input{Year: 2004, Clubs: clubs, Previous: prev, Before: before, Known: []*assets.Mana{prev, before}, NextLogo: 2000})
	if err != nil {
		t.Fatal(err)
	}
	n := 0
	for _, c := range out.Mana.Clubs[:numGerman] {
		if strings.Contains(c.Name, "ERFURT") {
			n++
		}
	}
	if n != 1 || out.Mana.Clubs[0].Name != "ROT-WEISS ERFURT" {
		t.Errorf("%d clubs of Erfurt, club 0 %q", n, out.Mana.Clubs[0].Name)
	}
}
