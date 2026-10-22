package game

// Round-robin schedules of the original (D:0x6fc for 18 clubs, D:0x744 for
// 20 clubs). Round k pairs the letters of string k: (s[2i], s[2i+1]) =
// (home, away), letter - 'a' = position of the club id inside its league.
var fixtures18 = [17]string{
	"abcdefghijklmnopqr", "ronklijghefcdabqpm", "afchejglinkpmrbdqo",
	"rknglejchafbdqompi", "ajclengpirkodfbhqm", "rgnclajbhdfqmkoipe",
	"fhancpergoimdjblqk", "rcnbldjfhqkimgoepa", "arcoemgkhjfldnbpqi",
	"rbnflhjqigkemcoapd", "amckeijlhnfpdrboqg", "rfnjlqgeickambodph",
	"aicglnjphrfodmbkqe", "rjnqecgaibkdmfohpl", "aecqlrjohmfkdibgpn",
	"rncaebgdifkhmjolqp", "aqnolmjkhifgdebcpr",
}

var fixtures20 = [19]string{
	"abcdefghijklmnopqrst", "dahcnklibetqropmjgfs", "caglinkpqfsbedhjmrot",
	"dstmlhbqrkpingjcaefo", "gpceobsairktmfqdjlhn", "phaqdotirgnjesfklcbm",
	"oahrcsgtifkbqelnjpmd", "bithrjampleofgdksqnc", "mecqlrgbkaosjtnpidhf",
	"tlaibhqofjdgrnpcsmek", "mqhdcoientjbksgaprlf", "sitpegdjrcomahfnblqk",
	"hecmnbgsiqkopfldrtja", "mktcejbpoiqgaldnshfr", "nagoimrbjsckpdhqtfle",
	"slendrkiapfcqjbtmgoh", "hmpetdnsgkralqicjofb", "mjdfqnatcbkholspiger",
	"lmbdrsnojkfahigctepq",
}

// Pair is a match: club ids of the home and the away team.
type Pair struct {
	Home, Away int
}

// Fixtures returns the pairs of league l on matchday md (1-based), like
// f_2a41_10e3. The second half of the season repeats the first with home
// and away swapped.
func Fixtures(l, md int) []Pair {
	half := MatchdaysPerLeague[l] / 2
	swap := md > half
	if swap {
		md -= half
	}
	if md > half {
		md = half
	}
	var s string
	if TeamsPerLeague[l] == 18 {
		s = fixtures18[md-1]
	} else {
		s = fixtures20[md-1]
	}
	off := FirstID[l]
	pairs := make([]Pair, TeamsPerLeague[l]/2)
	for i := range pairs {
		h, a := int(s[2*i]-'a')+off, int(s[2*i+1]-'a')+off
		if swap {
			h, a = a, h
		}
		pairs[i] = Pair{h, a}
	}
	return pairs
}
