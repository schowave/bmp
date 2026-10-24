package build

import (
	"fmt"
	"image"
	"image/color"
	"strings"
	"testing"

	"github.com/schowave/bmp/remake/internal/assets"
	"github.com/schowave/bmp/remake/internal/seasons/source"
)

func TestClubName(t *testing.T) {
	for label, want := range map[string]string{
		"1. FC Kaiserslautern":            "1.FC KAISERSLAUTERN",
		"FC St. Pauli":                    "FC ST.PAULI",
		"Eintracht Braunschweig":          "EINTRACHT BRAUNSCHWEIG",
		"Borussia Mönchengladbach":        "BOR. MÖNCHENGLADBACH",
		"Grasshopper Club Zürich":         "GRASSHOPPER ZÜRICH",
		"Panathinaikos Athen (Fußball)":   "PANATHINAIKOS ATHEN",
		"Beşiktaş Istanbul":               "BESIKTAS ISTANBUL",
		"Preußen Münster":                 "PREUßEN MÜNSTER",
		"Atlético Madrid":                 "ATLETICO MADRID",
		"Sportfreunde Siegen":             "SPORTFREUNDE SIEGEN",
		"Spielvereinigung Greuther Fürth": "SPVGG GREUTHER FÜRTH",
	} {
		if got := clubName(label); got != want {
			t.Errorf("%q: %q, want %q", label, got, want)
		}
		if got := clubName(label); length(got) > clubNameMax {
			t.Errorf("%q: %q too long", label, got)
		}
	}
	if got := playerName("Đurđević"); got != "DURDEVIC" {
		t.Errorf("player %q", got)
	}
}

// rows makes a tier of n clubs named prefix+i in the order of their places.
func rows(tier int, group, prefix string, n int) []source.TierRow {
	var out []source.TierRow
	for i := 1; i <= n; i++ {
		name := fmt.Sprintf("%s%d", prefix, i)
		out = append(out, source.TierRow{Tier: tier, Row: source.Row{Group: group, Rank: i, Won: 30 - i, Lost: i,
			GoalsFor: 60 - i, GoalsAgainst: 20 + i, Club: name, Label: name}})
	}
	return out
}

func leagues(p []placed) map[string]int {
	m := map[string]int{}
	for _, c := range p {
		m[c.row.Club] = c.league
	}
	return m
}

func TestPlaceClubs(t *testing.T) {
	// 1991/92: 20 clubs in the Bundesliga, 24 in two groups of the 2. Liga.
	var tables []source.TierRow
	tables = append(tables, rows(1, "", "B", 20)...)
	tables = append(tables, rows(2, "Nord", "N", 12)...)
	tables = append(tables, rows(2, "Süd", "S", 12)...)
	tables = append(tables, rows(3, "Oberliga Nord", "ON", 18)...)
	tables = append(tables, rows(3, "Oberliga Süd", "OS", 18)...)
	tables = append(tables, source.TierRow{Tier: 3, Row: source.Row{Rank: 1, Club: "Bayern München Amateure", Label: "FC Bayern München Amateure"}})
	p, err := placeClubs(1991, tables)
	if err != nil {
		t.Fatal(err)
	}
	if len(p) != numGerman {
		t.Fatalf("%d clubs", len(p))
	}
	l := leagues(p)
	for club, want := range map[string]int{"B18": 0, "B19": 1, "B20": 1, "N9": 1, "S9": 1, "N10": 2, "S10": 2, "ON1": 2, "ON9": 3} {
		if got, ok := l[club]; !ok || got != want {
			t.Errorf("%s: league %d (%v), want %d", club, got, ok, want)
		}
	}
	if _, ok := l["Bayern München Amateure"]; ok {
		t.Error("a reserve team in the Oberliga")
	}
	// From 1994 on the 2. Liga has 18: the best of the third tier fill it.
	tables = append(rows(1, "", "B", 18), rows(2, "", "Z", 18)...)
	tables = append(tables, rows(3, "Nord", "RN", 18)...)
	tables = append(tables, rows(3, "Süd", "RS", 18)...)
	p, err = placeClubs(1995, tables)
	if err != nil {
		t.Fatal(err)
	}
	l = leagues(p)
	if l["RN1"] != 1 || l["RS1"] != 1 || l["RN2"] != 2 {
		t.Errorf("fill: %d %d %d", l["RN1"], l["RS1"], l["RN2"])
	}
	// Moved up, they rank last in their new league.
	s := strengths(p)
	if s["RN1"][1] >= s["Z18"][1] || s["Z1"][1] != strengthRange[1][1] || s["B18"][1] != strengthRange[0][0] {
		t.Errorf("strengths: %v %v %v %v", s["RN1"], s["Z18"], s["Z1"], s["B18"])
	}
	// From 2008 on: 3. Liga with second teams, outside places from the
	// Regionalliga without them.
	tables = append(rows(1, "", "B", 18), rows(2, "", "Z", 18)...)
	tables = append(tables, rows(3, "", "D", 20)...)
	tables = append(tables, source.TierRow{Tier: 3, Row: source.Row{Rank: 21, Club: "Werder Bremen II", Label: "Werder Bremen II"}})
	tables = append(tables, rows(4, "West", "RW", 18)...)
	p, err = placeClubs(2008, tables)
	if err != nil {
		t.Fatal(err)
	}
	l = leagues(p)
	if l["D1"] != 1 || l["D2"] != 1 || l["D3"] != 2 || l["Werder Bremen II"] != 2 || l["RW1"] != 2 || l["RW2"] != 3 {
		t.Errorf("2008: %v", l)
	}
	if _, err := placeClubs(1995, rows(1, "", "B", 18)); err == nil {
		t.Error("a season without lower leagues was placed")
	}
}

func TestPickPlayers(t *testing.T) {
	var ps []source.Player
	add := func(name, sur string, pos, games int) {
		ps = append(ps, source.Player{Name: name, Surname: sur, Pos: pos, Games: games})
	}
	add("Oliver Kahn", "Kahn", source.PosGoal, 34)
	add("Bernd Dreher", "Dreher", source.PosGoal, 1)
	add("Sven Scheuer", "Scheuer", source.PosGoal, 0)
	add("Thomas Müller", "Müller", source.PosAttack, 30)
	add("Gerd Müller", "Müller", source.PosAttack, 33)
	add("Nobody", "Nobody", source.PosNone, 34)
	got := pickPlayers(ps, nil, false)
	if len(got) != 20 || got[0] != "KAHN" || got[1] != "DREHER" || got[2] != "" || got[15] != "G.MÜLLER" || got[16] != "T.MÜLLER" {
		t.Fatalf("got %q", got)
	}
}

func TestPickPlayersPrefersTransfermarkt(t *testing.T) {
	wiki := []source.Player{
		{Name: "Oliver Kahn", Surname: "Kahn", Pos: source.PosGoal, Games: 34},
		{Name: "Old Keeper", Surname: "Keeper", Pos: source.PosGoal, Games: 30},
		{Name: "Gerd Müller", Surname: "Müller", Pos: source.PosAttack, Games: 33},
		{Name: "Only Wiki", Surname: "Wiki", Pos: source.PosAttack, Games: 20},
	}
	tm := []source.Player{
		{Name: "TOMIC", Surname: "TOMIC", Pos: source.PosGoal, Games: -1},
		{Name: "KAHN", Surname: "KAHN", Pos: source.PosGoal, Games: -1},
		{Name: "SCHEUER", Surname: "SCHEUER", Pos: source.PosGoal, Games: -1},
		{Name: "MÜLLER", Surname: "MÜLLER", Pos: source.PosMidfield, Games: -1},
	}
	got := pickPlayers(wiki, tm, false)
	// Kahn has games, then the order of Transfermarkt; the Wikipedia keeper
	// Transfermarkt does not list stays out.
	if got[0] != "KAHN" || got[1] != "TOMIC" {
		t.Errorf("goalkeepers %q", got[:2])
	}
	// Sorted by market value, the order of Transfermarkt alone counts.
	if v := pickPlayers(wiki, tm, true); v[0] != "TOMIC" || v[1] != "KAHN" {
		t.Errorf("by value: goalkeepers %q", v[:2])
	}
	// Transfermarkt's position wins, Wikipedia fills the forwards.
	if got[8] != "MÜLLER" || got[15] != "WIKI" {
		t.Errorf("midfield %q, forwards %q", got[8:15], got[15:])
	}
}

func TestPickForeign(t *testing.T) {
	orig := &assets.Mana{Clubs: make([]assets.ClubRecord, assets.NumClubs)}
	for i := numGerman; i < assets.NumClubs; i++ {
		orig.Clubs[i] = assets.ClubRecord{Name: fmt.Sprintf("ORIG %d", i), Strength: [3]int{50, 50, 50}}
	}
	orig.Clubs[numGerman].Name = "AJAX AMSTERDAM"
	europe := []source.EuroClub{
		{Cup: 0, Club: "Ajax Amsterdam", Label: "Ajax Amsterdam", Mentions: 9},
		{Cup: 0, Club: "Juventus Turin", Label: "Juventus Turin", Mentions: 3},
		{Cup: 2, Club: "Hamburger SV", Label: "Hamburger SV", Mentions: 3},
	}
	got := pickForeign(europe, orig, map[string]bool{"HAMBURGER SV": true})
	if len(got) != assets.NumClubs-numGerman || got[0].Name != "AJAX AMSTERDAM" || got[1].Name != "JUVENTUS TURIN" {
		t.Fatalf("got %v %v", got[0], got[1])
	}
	if got[0].Strength[0] != foreignStrength[0][1] || got[1].Strength[0] != foreignStrength[0][0] {
		t.Errorf("strengths %v %v", got[0].Strength, got[1].Strength)
	}
	if got[2].Name != "ORIG 65" {
		t.Errorf("fill starts with %q: Ajax is there already", got[2].Name)
	}
}

func TestLogos(t *testing.T) {
	a, b := LogoFileName("Datei:FC Erzgebirge Aue.svg"), LogoFileName("Datei:Fc erzgebirge aue.svg")
	if strings.ToLower(a) != a || a == b || !strings.HasPrefix(a, "fc_erzgebirge_aue.svg-") {
		t.Errorf("logo files %q %q", a, b)
	}
	if c := crestColors("Schwarz-Gelb"); c[0] != 0 || c[1] != 23 {
		t.Errorf("Schwarz-Gelb: %v", c)
	}
	if c := crestColors("Rot und Weiß"); c[0] != 17 || c[1] != 29 {
		t.Errorf("Rot und Weiß: %v", c)
	}
	var pal assets.Palette
	for i := range pal {
		pal[i] = color.RGBA{uint8(i), uint8(i), uint8(i), 255}
	}
	pal[17] = color.RGBA{255, 0, 0, 255}
	// A red square, half transparent around it.
	img := image.NewNRGBA(image.Rect(0, 0, 76, 38))
	for y := 0; y < 38; y++ {
		for x := 19; x < 57; x++ {
			img.Set(x, y, color.NRGBA{250, 10, 10, 255})
		}
	}
	p := convertLogo(img, &pal)
	if p.W != logoSize || p.H != logoSize || p.Pix[19*logoSize+19] != 17 || p.Pix[0] != logoBackground || p.Pix[19*logoSize+2] != logoBackground {
		t.Fatalf("logo: %v %v %v", p.Pix[19*logoSize+19], p.Pix[0], p.Pix[19*logoSize+2])
	}
	c := crest("Rot und Weiß")
	if c.Pix[0] != logoBackground || c.Pix[15*logoSize+12] != 17 || c.Pix[15*logoSize+25] != 29 {
		t.Fatalf("crest %v %v", c.Pix[15*logoSize+12], c.Pix[15*logoSize+25])
	}
}
