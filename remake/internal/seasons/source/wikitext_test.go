package source

import (
	"reflect"
	"testing"
)

const tableArticle = `Intro {{Fußballtabelle/Zeile|Rang=1|S=1|U=0|N=0|ET=1|GT=0|Verein=[[Not A Table]]}}
{{Fußballtabelle/Fuß}}
== Nord ==
=== Abschlusstabelle ===
{{Fußballtabelle/Kopf}}
{{Fußballtabelle/Zeile|Farbe=ccffcc |Rang= 1 |S=19 |U=11 |N= 4 |ET=76 |GT=38 |Verein=[[Borussia Dortmund]] (M) }}
{{Fußballtabelle/Zeile|Rang= 2 |S=5 |U=1 |N= 28 |ET=20 |GT=90 |Verein=[[KFC Uerdingen 05|Bayer 05 Uerdingen]]<ref>x|y</ref> }}
=== Kreuztabelle ===
{{Fußballtabelle/Zeile|Rang= 9 |S=0 |U=0 |N=0 |ET=0 |GT=0 |Verein=[[Ignored]]}}
== Süd ==
=== Abschlusstabelle ===
{{Fußballtabelle/Zeile|Rang= 1 |S=20 |U=5 |N= 9 |ET=60 |GT=30 |Verein=[[FC Bayern München]]}}
== Relegation ==
=== Abschlusstabelle ===
{{Fußballtabelle/Zeile|Rang= 1 |S=2 |U=0 |N=0 |ET=4 |GT=1 |Verein=[[TSV Havelse]]}}
`

func TestTable(t *testing.T) {
	got := Table(tableArticle)
	want := []Row{
		{Group: "Nord", Rank: 1, Won: 19, Drawn: 11, Lost: 4, GoalsFor: 76, GoalsAgainst: 38, Club: "Borussia Dortmund", Label: "Borussia Dortmund"},
		{Group: "Nord", Rank: 2, Won: 5, Drawn: 1, Lost: 28, GoalsFor: 20, GoalsAgainst: 90, Club: "KFC Uerdingen 05", Label: "Bayer 05 Uerdingen"},
		{Group: "Süd", Rank: 1, Won: 20, Drawn: 5, Lost: 9, GoalsFor: 60, GoalsAgainst: 30, Club: "FC Bayern München", Label: "FC Bayern München"},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %+v", got)
	}
	// Without a table heading the first block counts.
	got = Table("x\n{{Fußballtabelle/Zeile|Rang=1|S=1|U=0|N=0|ET=1|GT=0|Verein=[[A]]}}\n{{Fußballtabelle/Fuß}}\n" +
		"{{Fußballtabelle/Zeile|Rang=1|S=1|U=0|N=0|ET=1|GT=0|Verein=[[B]]}}")
	if len(got) != 1 || got[0].Club != "A" {
		t.Fatalf("first block: %+v", got)
	}
}

const squadArticle = `== Kader ==
=== [[1. FC Kaiserslautern]] ===
[[Datei:Logo 1 FC Kaiserslautern.svg|198px]]<br />
{| class="wikitable sortable"
! Nat. !! Name !! Position !! Spiele !! Tore
|-
| || style="text-align:left" data-sort-value="Ehrmann, Gerry"| [[Gerry Ehrmann]] || data-sort-value="1" | Tor || 33 || 0
|-
| {{YUG|#}} || style="text-align:left" data-sort-value="Hajradinovic, Elvis"| [[Elvis Hajradinović]] || data-sort-value="2" | Abwehr || colspan="2" data-sort-value="0" | ohne Einsatz
|}
=== [[FC Ingolstadt 04]] ===
[[Datei:FC-Ingolstadt logo.svg|100px]]
{| class="wikitable"
|-
| {{0}}1 || {{AUT|#}} ||style="text-align:left" {{PersonZelle|Ramazan|Özcan}} ||data-sort-value="1"| Tor || 33 || 0
|-
| {{0}}8 || {{BRA|#}} ||style="text-align:left"| [[Roger de Oliveira Bernardo|Roger]] ||data-sort-value="3"| Mittelfeld || '''34''' || 0
|-
| 11 || ||style="text-align:left" {{PersonZelle|Tomáš|Pekhart}}<sup>ab 4</sup> ||data-sort-value="4"| Sturm || 12 || 0
|}
`

func TestSquads(t *testing.T) {
	got := Squads(squadArticle)
	want := []Squad{
		{Club: "1. FC Kaiserslautern", Label: "1. FC Kaiserslautern", Logo: "Datei:Logo 1 FC Kaiserslautern.svg", Players: []Player{
			{Name: "Gerry Ehrmann", Surname: "Ehrmann", Pos: PosGoal, Games: 33, Source: "wikipedia"},
			{Name: "Elvis Hajradinović", Surname: "Hajradinovic", Pos: PosDefence, Games: 0, Source: "wikipedia"},
		}},
		{Club: "FC Ingolstadt 04", Label: "FC Ingolstadt 04", Logo: "Datei:FC-Ingolstadt logo.svg", Players: []Player{
			{Name: "Ramazan Özcan", Surname: "Özcan", Pos: PosGoal, Games: 33, Source: "wikipedia"},
			{Name: "Roger", Surname: "Roger", Pos: PosMidfield, Games: 34, Source: "wikipedia"},
			{Name: "Tomáš Pekhart", Surname: "Pekhart", Pos: PosAttack, Games: 12, Source: "wikipedia"},
		}},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %+v", got)
	}
}

func TestParsePosition(t *testing.T) {
	for s, want := range map[string]int{"Tor": PosGoal, "Torwart": PosGoal, "Abwehr": PosDefence, "Innenverteidiger": PosDefence,
		"Mittelfeld": PosMidfield, "defensiver Mittelfeldspieler": PosMidfield, "Sturm": PosAttack, "Mittelstürmer": PosAttack,
		"Flügelspieler": PosAttack, "Trainer": PosNone} {
		if got := ParsePosition(s); got != want {
			t.Errorf("%q: %d, want %d", s, got, want)
		}
	}
}

func TestSurnameAndTitles(t *testing.T) {
	for name, want := range map[string]string{"Gerry Ehrmann": "Ehrmann", "Louis van Gaal": "van Gaal", "Roger": "Roger",
		"Axel Roos (Fußballspieler)": "Roos"} {
		if got := Surname(name); got != want {
			t.Errorf("%q: %q", name, got)
		}
	}
	if SeasonTitle(1999) != "1999/2000" || SeasonTitle(2009) != "2009/10" {
		t.Error("season titles")
	}
	if to, ok := redirectTarget("#WEITERLEITUNG [[Werder Bremen#Geschichte]]"); !ok || to != "Werder Bremen" {
		t.Errorf("redirect %q", to)
	}
	if m := euroArticles(1998); m[1] != "Europapokal der Pokalsieger 1998/99" || m[2] != "UEFA-Pokal 1998/99" || m[0] != "UEFA Champions League 1998/99" {
		t.Errorf("cups 1998: %v", m)
	}
	if m := euroArticles(2024); len(m) != 3 || m[3] != "UEFA Conference League 2024/25" {
		t.Errorf("cups 2024: %v", m)
	}
}

func TestCurrentSquad(t *testing.T) {
	text := `== Mannschaft ==
==== Aktueller Kader ====
{| class="wikitable"
! Nr. !! Nat. !! Name
|-
! colspan="6" style="background:#AD1732"|Torwart
|-
| '''{{0}}1''' || {{DEU|#}} || style="text-align:left" |[[Jan Reichert]] {{Kapitän}} || 28. Juni 2001 || 2021
|-
| '''13''' || {{POL|#}} || style="text-align:left" |Robin Lisewski || 13. Februar 2006 || 2025 || [[Borussia Dortmund#Jugend|Borussia Dortmund [Jugend]]]
<!-- AUSGELIEHEN
|-
| ''' ''' || {{SVK|#}} || style="text-align:left" |[[Michal Kukučka]] || 2002 || 2024
-->
|-
! colspan="6"|Abwehr
|-
| {{0}}4 || {{GRC|#}} || style="text-align:left" |[[Alexios Kalogeropoulos]] || 2004 || 2026
|}
==== Trainerstab ====
| style="text-align:left" | [[Somebody Else]]
`
	got := CurrentSquad(text, "2026/27")
	if len(got) != 3 || got[0].Surname != "Reichert" || got[1].Name != "Robin Lisewski" || got[2].Pos != PosDefence {
		t.Fatalf("got %+v", got)
	}
}

func TestSquadsWithoutLinks(t *testing.T) {
	text := "== Legende ==\n* Name\n=== FC Bayern München ===\n{{Hauptartikel|FC Bayern München}}\n{|\n|-\n" +
		"| 1 || || style=\"text-align:left\" {{PersonZelle|Manuel|Neuer}}<sup>U19</sup> ||data-sort-value=\"1\"| Tor || 30 || 0\n|}\n"
	got := Squads(text)
	if len(got) != 1 || got[0].Club != "FC Bayern München" || len(got[0].Players) != 1 || got[0].Players[0].Surname != "Neuer" {
		t.Fatalf("got %+v", got)
	}
	if p := plainText("Barjuan<sup>U19</sup>"); p != "Barjuan" {
		t.Errorf("sup: %q", p)
	}
}

func TestCurrentSquadVariants(t *testing.T) {
	text := `== Erste Mannschaft ==
=== Kader 2026/27 ===
{| class="wikitable"
! colspan="6" style="text-align:center"| Tor
|-
| {{0}}1 || style="text-align:center"| {{DEU|#}} || [[Timon Wellenreuther]] || style="text-align:left"| 3. Dezember 1995 || 2026
|}
== Zweite Mannschaft ==
=== Kader 2026/27 ===
! colspan="6"| Tor
| 1 || [[Someone Else]] || 2001
`
	got := CurrentSquad(text, "2026/27")
	if len(got) != 1 || got[0].Name != "Timon Wellenreuther" {
		t.Fatalf("got %+v", got)
	}
	if p, ok := squadRow("| 9 || style=\"text-align:left\" {{SortKeyName|Stefan Kießling|Stefan Kießling (Fußballspieler)}} || Sturm || 30"); !ok || p.Name != "Stefan Kießling" || p.Surname != "Kießling" {
		t.Fatalf("SortKeyName: %+v", p)
	}
}
