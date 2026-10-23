package game

import "strings"

// Texts of the messages in the message panel. "\n" is the line break ('^'
// in the original). Panel messages of the original come from the formats at
// D:0x267c (f_3091_0190; '$' a string, '#' a number with thousands dots):
//
//	0 "$ bietet an,^von # auf #^Jahre zu verl{ngern.^"
//	1 "$^ist an $^interessiert.^"
//	2 "$^$^$ ^" (f_0000_239e, popup variant 5)
//	3 "$ h{ngt den^Fu~balljob im Alter von^# Jahren an den Nagel.^"
//	4 "$ k}ndigt an,^da~ er seinen Vertrag nicht^mehr verl{ngern wird.^"
//
// The panel draws each part that ends with '^' as a line, so the final '^'
// adds nothing; the "$ " of format 2 leaves a trailing space on the third
// line (and an empty third part becomes a blank line), which is kept here.
//
// The other texts are shown by the original as 4-line OKAY dialogs
// (f_3091_0e3a) or on screens of their own; their lines are joined with "\n"
// in the order of the original lines (empty trailing lines dropped).

// msgLines3 is format 2: three lines, the last with a trailing space.
func msgLines3(a, b, c string) string { return a + "\n" + b + "\n" + c + " " }

// Format 0 (f_0cb5_13bd): a player with a one-year contract offers to
// extend it.
func msgOfferExtension(name string, from, to int) string {
	return name + " bietet an,\nvon " + itoa(from) + " auf " + itoa(to) + "\nJahre zu verlängern."
}

// Format 1 (f_0cb5_13bd): a club is interested in a player.
func msgInterested(club, name string) string {
	return club + "\nist an " + name + "\ninteressiert."
}

// Format 3 (f_0cb5_0012): a player retires.
func msgRetires(name string, age int) string {
	return name + " hängt den\nFußballjob im Alter von\n" + itoa(age) + " Jahren an den Nagel."
}

// Format 4 (f_0cb5_13bd): a player will not extend his contract.
func msgNoExtension(name string) string {
	return name + " kündigt an,\ndaß er seinen Vertrag nicht\nmehr verlängern wird."
}

// Format 2: injury in training (D:0x2794; the injury in brackets,
// f_14a4_30cf).
func msgInjury(name, injury string) string {
	return msgLines3("Verletzung im Training:", name, "("+injury+")")
}

// Format 2: riot damage (D:0x277c..0x2784, " DM an." D:0x4f7c).
func msgRiot(dmg int64) string {
	return msgLines3("Randalierer im Stadion", "richteten einen Sachschaden", "von "+FormatMoney(dmg)+" DM an.")
}

// Format 2: comfort downgrade (D:0x2788..0x2790).
func msgComfort() string {
	return msgLines3("In der neuen Komfortbewertung", "wird ihr Stadion um eine Note", "schlechter beurteilt.")
}

// Format 2 (f_0000_20e1): a construction is finished. Items 3, 4 and 6 name
// the object on the second line; the others have an empty third line.
var (
	completionHead = [NumStadiumItems]string{"Der Ausbau der Sitzplätze", "Der Ausbau der Stehplätze",
		"Die Überdachung des Stadions", "Die Errichtung der neuen", "Die Errichtung der neuen",
		"Die Stadionrenovierung", "Die Verbesserung des"}
	completionObject = [NumStadiumItems]string{3: "Flutlichtmasten", 4: "Anzeigetafel", 6: "Stadion-Komforts"}
)

func msgCompletion(k int) string {
	if o := completionObject[k]; o != "" {
		return msgLines3(completionHead[k], o, "ist abgeschlossen.")
	}
	return msgLines3(completionHead[k], "ist abgeschlossen.", "")
}

// Dialog (f_112a_0a6d and f_14a4_0906, D:0x4e50/0x4e54): a credit is due.
func msgCreditDue(lender string) string {
	return "Ihr Kredit von\n" + lender + "\nwurde heute fällig."
}

// Christmas card (f_1a58_2a06): title D:0x2724, five lines D:0x26f0..
// 0x2700 and, in the small font, the amounts D:0x2728..0x2734; x is the
// Bundesliga amount, the 2. Liga pays x/2, the Oberliga x/3. Without
// packages the card shows the title only.
const msgChristmasTitle = "Frohe Weihnachten !"

var msgChristmasLines = [5]string{
	"Ihre Spieler sind überwältigt und",
	"zu Tränen gerührt, als sie Ihre",
	"großzügigen Weihnachtspakete unter",
	"dem lammettabehangenen Christbaum",
	"erspähen. Was für eine Freude !",
}

func msgChristmasAmounts(x int64, year int) []string {
	third := "AMATEUR-OBERLIGA"
	if n := LeagueName(2, year); n != LeagueNames[2] {
		third = strings.ToUpper(n)
	}
	return []string{
		"BUNDESLIGA: " + FormatMoney(x) + " DM, 2.LIGA: " + FormatMoney(x/2) + " DM,",
		third + ": " + FormatMoney(x/3) + " DM (INCL. MWST.)",
	}
}

// Birthday cards of f_1a58_2a06 (special dates 1 and 2): "Oh happy day
// !!!" (D:0x539e), eight lines D:0x2704.. with the name after the fourth
// (D:0x53af, 0x53ba), and the two lines shown when the card is clicked
// away within the minute (D:0x2738, 0x273c).
const msgBirthdayTitle = "Oh happy day !!!"

func msgBirthday(k int) []string {
	name := "JENS ONNEN"
	if k != 1 {
		name = "WERNER KRAHE"
	}
	return []string{
		"Sehr geehrte Mitspieler !",
		"Heute ist ein äußerst denkwürdiger Tag.",
		"Es ist der Geburtstag eines der beiden",
		"glorreichen Programmierer: " + name,
		"Laßt uns eine Minute in ehrerbietendem",
		"Schweigen verharren und dem gedenken,",
		"ohne den dieses affenstarke Spiel nie",
		"das Licht der Welt erblickt hätte...",
	}
}

var msgBirthdayShame = []string{"Das war keine Minute !", "Schämen Sie Sich !"}

// Dialogs of f_112a_314d (D:0x4eb0.., 0x4ec0..).
const (
	msgRelegation  = "Ihre Mannschaft\nbestreitet das\nRelegationsspiel.\nViel Glück."
	msgWinterBreak = "Achtung !\nDies ist der letzte\nSpieltag vor der\nWinterpause."
)

// Goal dialogs of f_112a_314d: manager name, "<club> hat", then two lines.
func msgGoalLost(manager, club, goal string) string {
	return manager + "\n" + club + " hat\ndie letzte Chance verspielt,\n" + goal
}

func msgGoalWon(manager, club, goal string) string {
	return manager + "\n" + club + " hat\n" + goal + "\nauf jeden Fall gesichert."
}

// Dialog of f_1a58_2051: a forfeited match (no full stop after "Strafe").
func msgForfeit(manager string) string {
	return manager + "\nIhr Spiel wird mit 0:2\ngewertet. Sie zahlen\n200.000 DM Strafe"
}

// Season end report screen of f_0cb5_0012: "Sie beendeten Ihre Saison"
// (D:0x2744), "<n>.Platz der <LIGA>" with the league name in capitals
// (D:0x274c, f_3091_105d), the heading "Ereignisse" (D:0x2748) and single
// event lines; "Keine besonderen Ereignisse..." (D:0x2770) when there are
// none. The original does not draw the title "Saisonende" (D:0x2740): its
// title function f_06c7_09f9 ignores the text.
const (
	msgSeasonHead   = "Sie beendeten Ihre Saison"
	msgSeasonEvents = "Ereignisse"
	msgNoEvents     = "Keine besonderen Ereignisse..."
)

func msgSeasonPlace(pos, league, year int) string {
	return itoa(pos) + ".Platz der " + strings.ToUpper(LeagueName(league, year))
}

// msgSeasonEnd is the head of the report as one text.
func msgSeasonEnd(pos, league, year int) string {
	return "Saisonende\n" + msgSeasonHead + "\n" + msgSeasonPlace(pos, league, year)
}

// Dialogs D:0x4e08.. and 0x4e18.. that f_0cb5_0012 shows with chance 1/5
// over the season report.
const (
	msgJoke1 = "Ihre Spieler werden\nälter und schwächer...\n \nPech gehabt."
	msgJoke2 = "Mit freundlichem Gruß\nvom Bundesliga Manager I:\nWar nur ein\nSCHERZ !!!"
)

const (
	msgPromoted1   = "Gratulation zum Aufstieg (1.600.000 DM)."
	msgPromoted2   = "Gratulation zum Aufstieg (800.000 DM)."
	msgRelegated   = "Ihr Abstieg erschüttert Millionen Fans."
	msgNoLicence   = "DFB verweigert die Lizenz. Sie steigen ab."
	msgRestart     = "Sie beginnen neu mit 500.000 DM."
	msgTopScorer   = "Torschützenkönig aus Ihrem Team (250.000 DM)."
	msgEurope      = "Sie sind für den Europapokal qualifiziert."
	msgYouthPlayer = "Ein Jugendspieler steigt in den Kader auf."
	// Dialog D:0x4f44..0x4f50 after a promotion.
	msgAdsFree = "Ihre Werbepartner erlauben\nes, die Verträge jederzeit\nzu kündigen. Die Einnahmen\nbleiben solange erhalten."
)

// Title screen of f_17c2_274d: "<club>", "gewinnt die" (D:0x26c0),
// "Deutsche Meisterschaft" (D:0x26bc); for cup c (1..4) "gewinnt den"
// (D:0x26c4) and the cup name (D:0x24d8 + c*4).
func msgTrophy(club string, trophy int) string {
	if trophy == 0 {
		return club + "\ngewinnt die\nDeutsche Meisterschaft"
	}
	return club + "\ngewinnt den\n" + CupNames[trophy-1]
}

// Dialogs of f_0cb5_0012 when a player with an expired contract leaves:
// "<name>/kehrt Ihrem Verein/den Rücken..." and "Sie erhalten eine/
// Ablösesumme von/<n> DM".
func msgReleased(name string, comp int64) string {
	return name + "\nkehrt Ihrem Verein\nden Rücken...\nSie erhalten eine\nAblösesumme von\n" + FormatMoney(comp) + " DM"
}
