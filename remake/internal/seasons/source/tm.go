package source

import (
	"encoding/json"
	"os"
	"regexp"
	"strings"
	"unicode"
)

// TMClub is a club of a season file of Transfermarkt squads: its name and
// its players by position, surnames in capitals.
type TMClub struct {
	Name    string `json:"name"`
	League  string `json:"league"`
	Players struct {
		Goalkeepers []string `json:"goalkeepers"`
		Defenders   []string `json:"defenders"`
		Midfielders []string `json:"midfielders"`
		Forwards    []string `json:"forwards"`
	} `json:"players"`
}

// ReadTM reads a season file of Transfermarkt squads.
func ReadTM(file string) ([]TMClub, error) {
	b, err := os.ReadFile(file)
	if err != nil {
		return nil, err
	}
	var clubs []TMClub
	if err := json.Unmarshal(b, &clubs); err != nil {
		return nil, err
	}
	return clubs, nil
}

// Squad returns the players of c in the order of the file.
func (c TMClub) Squad() []Player {
	var out []Player
	for k, names := range [4][]string{c.Players.Goalkeepers, c.Players.Defenders, c.Players.Midfielders, c.Players.Forwards} {
		for _, n := range names {
			if n = strings.TrimSpace(n); n != "" {
				out = append(out, Player{Name: n, Surname: n, Pos: PosGoal + k, Games: -1, Source: "transfermarkt"})
			}
		}
	}
	return out
}

var (
	yearsSuffix = regexp.MustCompile(`\s*\(-?\s*\d{4}\)\s*$`)
	amateure    = regexp.MustCompile(`\bAMATEURE\b`)
	clubForms   = map[string]bool{"FC": true, "SV": true, "VFL": true, "VFB": true, "TSV": true, "SC": true, "SG": true,
		"FSV": true, "SPVGG": true, "TUS": true, "VFR": true, "BSV": true, "SSV": true, "KSV": true, "1": true, "EV": true,
		"E": true, "V": true, "FK": true, "AC": true, "ASC": true, "BV": true, "TSG": true, "SPORTFREUNDE": true,
		"VEREIN": true, "SPORTVEREIN": true, "SPORTCLUB": true}
)

// clubWords are the words of a club name in capitals without accents,
// "(- 1994)" and with "Amateure" as II.
func clubWords(s string) []string {
	s = yearsSuffix.ReplaceAllString(strings.TrimSpace(s), "")
	s = strings.NewReplacer("ß", "SS", "ä", "AE", "ö", "OE", "ü", "UE", "Ä", "AE", "Ö", "OE", "Ü", "UE").Replace(s)
	var b strings.Builder
	for _, r := range strings.ToUpper(s) {
		switch {
		case r < 0x80 && (unicode.IsLetter(r) || unicode.IsDigit(r)):
			b.WriteRune(r)
		case Translit[r] != "":
			b.WriteString(Translit[r])
		default:
			b.WriteByte(' ')
		}
	}
	return strings.Fields(amateure.ReplaceAllString(b.String(), "II"))
}

// clubKeys are the full key of a name and the key without forms like
// "FC" or "SV".
func clubKeys(s string) (full, core string) {
	var f, c strings.Builder
	for _, w := range clubWords(s) {
		f.WriteString(w)
		if !clubForms[w] {
			c.WriteString(w)
		}
	}
	return f.String(), c.String()
}

// MatchTM finds for each Transfermarkt club the club of the tables it
// means, by its index in clubs. First every club takes a club of the same
// name; the others then take, among the clubs still free, the only one with
// the same name without its form ("SV", "FC"), else the only one whose name
// contains theirs or is contained in it and makes at least 60 % of it
// ("VIKTORIA KÖLN" in "SC VIKTORIA KÖLN 1904", not "FRANKFURT" in
// "VORWÄRTS FRANKFURT/ODER"). A second team only matches a second team.
func MatchTM(clubs []TMClub, tables []TierRow) map[int]string {
	out := map[int]string{}
	taken := map[string]bool{}
	for i, c := range clubs {
		full, _ := clubKeys(c.Name)
		for _, r := range candidates(c.Name, tables) {
			if f1, _ := clubKeys(r.Club); f1 == full || keyOf(r.Label) == full {
				if !taken[r.Team()] {
					out[i] = r.Team()
					taken[r.Team()] = true
				}
				break
			}
		}
	}
	for i, c := range clubs {
		if _, ok := out[i]; ok {
			continue
		}
		var free []TierRow
		for _, r := range candidates(c.Name, tables) {
			if !taken[r.Team()] {
				free = append(free, r)
			}
		}
		if m := matchLoose(c.Name, free); m != "" {
			out[i] = m
			taken[m] = true
		}
	}
	return out
}

func keyOf(s string) string { f, _ := clubKeys(s); return f }

// candidates are the rows a name may mean: second teams for a second team,
// the others else.
func candidates(name string, tables []TierRow) []TierRow {
	second := IsSecondTeam(strings.Join(clubWords(name), " "))
	var out []TierRow
	for _, r := range tables {
		if (IsSecondTeam(r.Club) || IsSecondTeam(r.Label)) == second {
			out = append(out, r)
		}
	}
	return out
}

// matchLoose is the only club of tables with the core name of name, else
// the only one whose core contains it or is contained in it by 60 %.
func matchLoose(name string, tables []TierRow) string {
	_, core := clubKeys(name)
	only := func(fit func(c string) bool) string {
		found := ""
		for _, r := range tables {
			for _, s := range []string{r.Club, r.Label} {
				if _, c := clubKeys(s); fit(c) {
					if found != "" && found != r.Team() {
						return ""
					}
					found = r.Team()
				}
			}
		}
		return found
	}
	if core == "" {
		return ""
	}
	if m := only(func(c string) bool { return c == core }); m != "" {
		return m
	}
	return only(func(c string) bool {
		short, long := c, core
		if len(short) > len(long) {
			short, long = long, short
		}
		return len(short) >= 5 && 10*len(short) >= 6*len(long) && strings.Contains(long, short)
	})
}
