package build

import (
	"regexp"
	"strings"
	"unicode"
)

// Lengths of the names in MANA.DAT, without the NUL.
const (
	clubNameMax   = 22
	playerNameMax = 25
)

// translit spells letters the fonts of the game do not have with those it
// has. The fonts have A–Z, digits, some punctuation and Ä Ö Ü ß.
var translit = map[rune]string{
	'À': "A", 'Á': "A", 'Â': "A", 'Ã': "A", 'Å': "A", 'Ą': "A", 'Ă': "A", 'Ā': "A",
	'Æ': "AE", 'Ç': "C", 'Ć': "C", 'Č': "C", 'Ď': "D", 'Đ': "D", 'Ð': "D",
	'È': "E", 'É': "E", 'Ê': "E", 'Ë': "E", 'Ę': "E", 'Ě': "E", 'Ė': "E", 'Ē': "E",
	'Ğ': "G", 'Ģ': "G", 'Ì': "I", 'Í': "I", 'Î': "I", 'Ï': "I", 'İ': "I", 'Ī': "I", 'Į': "I",
	'Ķ': "K", 'Ł': "L", 'Ľ': "L", 'Ĺ': "L", 'Ļ': "L", 'Ñ': "N", 'Ń': "N", 'Ň': "N", 'Ņ': "N",
	'Ò': "O", 'Ó': "O", 'Ô': "O", 'Õ': "O", 'Ø': "O", 'Ō': "O", 'Ő': "Ö", 'Œ': "OE",
	'Ř': "R", 'Ŕ': "R", 'Ś': "S", 'Š': "S", 'Ş': "S", 'Ș': "S", 'Ť': "T", 'Ţ': "T", 'Ț': "T",
	'Ù': "U", 'Ú': "U", 'Û': "U", 'Ů': "U", 'Ū': "U", 'Ų': "U", 'Ű': "Ü",
	'Ý': "Y", 'Ÿ': "Y", 'Ź': "Z", 'Ž': "Z", 'Ż': "Z", 'Þ': "TH",
	'’': "'", '‘': "'", '´': "'", '`': "'", '–': "-", '—': "-", '·': ".",
}

// upper writes s in capitals with only the letters of the game's fonts.
func upper(s string) string {
	var b strings.Builder
	for _, r := range s {
		if r == 'ß' {
			b.WriteRune(r)
			continue
		}
		r = unicode.ToUpper(r)
		switch {
		case r < 0x80 || r == 'Ä' || r == 'Ö' || r == 'Ü':
			b.WriteRune(r)
		case translit[r] != "":
			b.WriteString(translit[r])
		case unicode.IsSpace(r):
			b.WriteByte(' ')
		}
	}
	return strings.Join(strings.Fields(b.String()), " ")
}

// length counts characters as the game stores them (one byte each).
func length(s string) int { return len([]rune(s)) }

// cut shortens s to n characters.
func cut(s string, n int) string {
	r := []rune(s)
	if len(r) > n {
		r = r[:n]
	}
	return strings.TrimSpace(string(r))
}

var (
	disambiguation = regexp.MustCompile(`\s*\([^)]*\)$`)
	numberDot      = regexp.MustCompile(`\b(\d+)\. `)
)

// clubAbbrev are the shortenings of the original ("EINTR. BRAUNSCHWEIG",
// "STUTTGARTER KICK."), tried in this order until a name fits.
var clubAbbrev = [][2]string{
	{"BORUSSIA", "BOR."}, {"EINTRACHT", "EINTR."}, {"KICKERS", "KICK."}, {"FORTUNA", "FORT."},
	{"SPORTFREUNDE", "SPFR."}, {"SPIELVEREINIGUNG", "SPVGG"}, {"ALEMANNIA", "ALEM."}, {"ARMINIA", "ARM."},
	{"VIKTORIA", "VIKT."}, {"PREUßEN", "PR."}, {"WESTFALIA", "WESTF."}, {"GERMANIA", "GERM."},
	{"OLYMPIQUE", "OLYMP."}, {"SPORTING", "SP."}, {"ATHLETIC", "ATH."}, {"DYNAMO", "DYN."},
	{"DINAMO", "DIN."}, {"LOKOMOTIVE", "LOK"}, {"LOKOMOTIV", "LOK"}, {"UNITED", "UTD."},
	{"WANDERERS", "WAND."}, {"ROVERS", "ROV."}, {"ATHLETICO", "ATL."}, {"ATLÉTICO", "ATL."},
	{"SPARTAK", "SP."}, {"SPORTCLUB", "SC"}, {"FUSSBALLCLUB", "FC"}, {"TURN- UND SPORTVEREIN", "TSV"},
	{" CLUB ", " "}, {"GRASSHOPPER", "GRASSHOPP."}, {"ATHEN", "ATH."}, {"FOOTBALL", "F."},
}

// clubName writes the name of a club as the game shows it: the name the
// table of the season shows, in capitals, "1. FC" as "1.FC", shortened like
// the original when it is too long.
func clubName(label string) string {
	s := disambiguation.ReplaceAllString(strings.TrimSpace(label), "")
	s = upper(s)
	s = numberDot.ReplaceAllString(s, "$1.")
	s = strings.ReplaceAll(s, "ST. ", "ST.")
	for _, a := range clubAbbrev {
		if length(s) <= clubNameMax {
			break
		}
		s = strings.ReplaceAll(s, a[0], a[1])
	}
	// Then the form of the club: "SV", "FC" … at the start.
	for length(s) > clubNameMax {
		f := strings.Fields(s)
		if len(f) < 2 || length(f[0]) > 4 {
			break
		}
		s = strings.Join(f[1:], " ")
	}
	return cut(s, clubNameMax)
}

// playerName is the surname in capitals, at most 25 characters; of two
// spellings ("Andonow/Andonov") the first.
func playerName(surname string) string {
	surname, _, _ = strings.Cut(surname, "/")
	return cut(upper(surname), playerNameMax)
}
