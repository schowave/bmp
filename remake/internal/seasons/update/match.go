package update

import (
	"regexp"
	"strings"
	"unicode"
)

// Transfermarkt writes names its own way ("SV WERDER BREMEN", "1.FC
// HEIDENHEIM 1846"); the seasons of the remake keep the names they had. A
// club of a Transfermarkt file takes the name it has in the seasons when
// the two match.

var (
	yearsSuffix = regexp.MustCompile(`\s*\(-?\s*\d{4}\)\s*$`)
	amateure    = regexp.MustCompile(`\bAMATEURE?\b`)
	secondTeam  = regexp.MustCompile(`\b(II|U ?2[0-3])\b`)
	// clubForms are the words of the legal form of a club, left out of the
	// core of a name.
	clubForms = map[string]bool{"FC": true, "SV": true, "VFL": true, "VFB": true, "TSV": true, "SC": true, "SG": true,
		"FSV": true, "SPVGG": true, "TUS": true, "VFR": true, "BSV": true, "SSV": true, "KSV": true, "1": true, "EV": true,
		"E": true, "V": true, "FK": true, "AC": true, "ASC": true, "BV": true, "TSG": true, "SPORTFREUNDE": true,
		"VEREIN": true, "SPORTVEREIN": true, "SPORTCLUB": true}
)

// clubWords are the words of a club name in capitals without accents,
// "(- 1994)" and with "Amateure" (or the cut "Amateur") as II.
func clubWords(s string) []string {
	s = yearsSuffix.ReplaceAllString(strings.TrimSpace(s), "")
	s = strings.NewReplacer("ß", "SS", "ä", "AE", "ö", "OE", "ü", "UE", "Ä", "AE", "Ö", "OE", "Ü", "UE").Replace(s)
	var b strings.Builder
	for _, r := range strings.ToUpper(s) {
		switch {
		case r < 0x80 && (unicode.IsLetter(r) || unicode.IsDigit(r)):
			b.WriteRune(r)
		case translit[r] != "":
			b.WriteString(translit[r])
		default:
			b.WriteByte(' ')
		}
	}
	return strings.Fields(amateure.ReplaceAllString(b.String(), "II"))
}

// clubKeys are the full key of a name and the key without its legal form
// ("SV", "FC").
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

func isSecond(s string) bool { return secondTeam.MatchString(strings.Join(clubWords(s), " ")) }

// letters is a core without its digits: the year of a club's founding comes
// and goes in its name ("SV 07 ELVERSBERG", "TSG 1899 HOFFENHEIM").
func letters(core string) string {
	return strings.Map(func(r rune) rune {
		if unicode.IsDigit(r) {
			return -1
		}
		return r
	}, core)
}

// matchNames finds for each name of names the name of known it means: first
// the same name, then among the known names still free the only one with
// the same core, the only one with the same core without digits, else the
// only one whose core contains it or is contained
// in it and makes at least 60 % of it ("HEIDENHEIM" in "HEIDENHEIM 1846",
// not "FRANKFURT" in "VORWÄRTS FRANKFURT/ODER"); strict leaves out the last
// ("LEIPZIG" in "RB LEIPZIG" is another club in another season). A second
// team only matches a second team. Names without a match are left out.
func matchNames(names, known []string, strict bool) map[string]string {
	out := map[string]string{}
	taken := map[string]bool{}
	for _, n := range names {
		full, _ := clubKeys(n)
		for _, k := range known {
			if f, _ := clubKeys(k); f == full && !taken[k] {
				out[n], taken[k] = k, true
				break
			}
		}
	}
	for _, n := range names {
		if _, ok := out[n]; ok {
			continue
		}
		_, core := clubKeys(n)
		if core == "" {
			continue
		}
		only := func(fit func(c string) bool) string {
			found := ""
			for _, k := range known {
				if taken[k] || isSecond(k) != isSecond(n) {
					continue
				}
				if _, c := clubKeys(k); fit(c) {
					if found != "" {
						return ""
					}
					found = k
				}
			}
			return found
		}
		m := only(func(c string) bool { return c == core })
		if m == "" && len(letters(core)) >= 5 {
			m = only(func(c string) bool { return letters(c) == letters(core) })
		}
		if m == "" && !strict {
			m = only(func(c string) bool {
				short, long := c, core
				if len(short) > len(long) {
					short, long = long, short
				}
				return len(short) >= 5 && 10*len(short) >= 6*len(long) && strings.Contains(long, short)
			})
		}
		if m != "" {
			out[n], taken[m] = m, true
		}
	}
	return out
}

// firstTeam is the name of known that is the first team of the second team
// name ("HAMBURGER SV" for "HAMBURGER SV II"); "" when name is no second
// team or none or several match.
func firstTeam(name string, known []string) string {
	if !isSecond(name) {
		return ""
	}
	base := strings.TrimSpace(secondTeam.ReplaceAllString(strings.Join(clubWords(name), " "), ""))
	var firsts []string
	for _, k := range known {
		if !isSecond(k) {
			firsts = append(firsts, k)
		}
	}
	return matchNames([]string{base}, firsts, false)[base]
}
