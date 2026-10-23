package source

import (
	"regexp"
	"strconv"
	"strings"
)

// Row is a line of a league table of a Wikipedia article
// ({{Fußballtabelle/Zeile}}).
type Row struct {
	Group        string // the section of the group (Staffel), e.g. "Nord"
	Rank         int
	Won, Drawn   int
	Lost         int
	GoalsFor     int
	GoalsAgainst int
	Club         string // the linked article
	Label        string // the name shown in the table
}

// Points under the 3-point rule, for ordering only.
func (r Row) Points() int { return 3*r.Won + r.Drawn }

// Games is the number of games played.
func (r Row) Games() int { return r.Won + r.Drawn + r.Lost }

// heading reads a section heading "== Title ==" and returns its level.
func heading(line string) (string, int, bool) {
	t := strings.TrimSpace(line)
	if !strings.HasPrefix(t, "==") {
		return "", 0, false
	}
	n := 0
	for n < len(t) && t[n] == '=' {
		n++
	}
	title := strings.TrimSpace(strings.Trim(t, "= "))
	return plainText(title), n, true
}

// sections walks the lines of text and calls f with the current headings
// by level (index 2 = "==", 3 = "===", 4 = "====").
func sections(text string, f func(line string, heads [6]string)) {
	var heads [6]string
	for _, line := range strings.Split(text, "\n") {
		if h, n, ok := heading(line); ok && n < len(heads) {
			heads[n] = h
			for k := n + 1; k < len(heads); k++ {
				heads[k] = ""
			}
			continue
		}
		f(line, heads)
	}
}

// isTableHeading reports whether a heading introduces the final table.
func isTableHeading(h string) bool {
	h = strings.ToLower(h)
	return strings.Contains(h, "abschlusstabelle") || h == "tabelle" || h == "endstand"
}

// Table returns the final table of a season article: the rows under the
// headings "Abschlusstabelle" or "Tabelle", grouped by the section above
// (the Staffel). An article without such a heading gives the first block
// of rows.
func Table(text string) []Row {
	var rows, first []Row
	inFirst := true
	sections(text, func(line string, heads [6]string) {
		r, ok := tableRow(line)
		if !ok {
			if len(first) > 0 && strings.Contains(line, "Fußballtabelle/Fuß") {
				inFirst = false
			}
			return
		}
		if inFirst {
			first = append(first, r)
		}
		for lvl := 5; lvl >= 2; lvl-- {
			if heads[lvl] == "" {
				continue
			}
			if !isTableHeading(heads[lvl]) {
				return
			}
			if lvl > 2 {
				r.Group = heads[lvl-1]
			}
			if isPlayoff(r.Group) {
				return
			}
			rows = append(rows, r)
			return
		}
	})
	if len(rows) == 0 {
		return first
	}
	return rows
}

// isPlayoff reports whether a section holds a play-off instead of a
// league: relegation, promotion rounds, finals.
func isPlayoff(group string) bool {
	g := strings.ToLower(group)
	for _, w := range []string{"relegation", "aufstieg", "abstieg", "qualifikation", "endrunde", "meisterrunde", "finale"} {
		if strings.Contains(g, w) {
			return true
		}
	}
	return false
}

var rowTemplate = regexp.MustCompile(`\{\{\s*Fußballtabelle/Zeile\s*\|`)

// tableRow parses a {{Fußballtabelle/Zeile|...}} line.
func tableRow(line string) (Row, bool) {
	loc := rowTemplate.FindStringIndex(line)
	if loc == nil {
		return Row{}, false
	}
	body, ok := templateBody(line[loc[0]:])
	if !ok {
		return Row{}, false
	}
	p := params(body)
	num := func(k string) int {
		v := strings.TrimSpace(plainText(p[k]))
		n, _ := strconv.Atoi(strings.TrimLeft(v, "+"))
		return n
	}
	r := Row{Rank: num("Rang"), Won: num("S"), Drawn: num("U"), Lost: num("N"), GoalsFor: num("ET"), GoalsAgainst: num("GT")}
	r.Club, r.Label = firstLink(p["Verein"])
	if r.Club == "" {
		return Row{}, false
	}
	return r, true
}

// templateBody returns the text between "{{" at the start of s and its
// matching "}}".
func templateBody(s string) (string, bool) {
	depth := 0
	for i := 0; i+1 < len(s); i++ {
		switch s[i : i+2] {
		case "{{":
			depth++
			i++
		case "}}":
			depth--
			i++
			if depth == 0 {
				return s[2 : i-1], true
			}
		}
	}
	return "", false
}

// splitTop splits s at sep where it is not inside [[ ]] or {{ }}.
func splitTop(s, sep string) []string {
	var out []string
	depth, start := 0, 0
	for i := 0; i < len(s); i++ {
		switch {
		case strings.HasPrefix(s[i:], "[[") || strings.HasPrefix(s[i:], "{{"):
			depth++
			i++
		case strings.HasPrefix(s[i:], "]]") || strings.HasPrefix(s[i:], "}}"):
			if depth > 0 {
				depth--
			}
			i++
		case depth == 0 && strings.HasPrefix(s[i:], sep):
			out = append(out, s[start:i])
			start = i + len(sep)
			i += len(sep) - 1
		}
	}
	return append(out, s[start:])
}

// params reads the named parameters of a template body.
func params(body string) map[string]string {
	m := map[string]string{}
	for _, part := range splitTop(body, "|")[1:] {
		k, v, ok := strings.Cut(part, "=")
		if ok {
			m[strings.TrimSpace(k)] = strings.TrimSpace(v)
		}
	}
	return m
}

var linkRE = regexp.MustCompile(`\[\[([^\]|#]+)(?:#[^\]|]*)?(?:\|([^\]]*))?\]\]`)

// firstLink returns the target and the label of the first article link.
func firstLink(s string) (target, label string) {
	for _, m := range linkRE.FindAllStringSubmatch(s, -1) {
		t := strings.TrimSpace(m[1])
		if isFileLink(t) || strings.Contains(t, ":") {
			continue
		}
		l := strings.TrimSpace(m[2])
		if l == "" {
			l = t
		}
		return t, plainText(l)
	}
	return "", ""
}

func isFileLink(t string) bool {
	l := strings.ToLower(t)
	return strings.HasPrefix(l, "datei:") || strings.HasPrefix(l, "file:") || strings.HasPrefix(l, "bild:") || strings.HasPrefix(l, "image:")
}

// Links returns the targets of all article links in text, in order, with
// duplicates.
func Links(text string) []string {
	var out []string
	for _, m := range linkRE.FindAllStringSubmatch(text, -1) {
		t := strings.TrimSpace(m[1])
		if t == "" || strings.Contains(t, ":") {
			continue
		}
		out = append(out, t)
	}
	return out
}

var (
	templateRE = regexp.MustCompile(`\{\{[^{}]*\}\}`)
	refRE      = regexp.MustCompile(`(?s)<ref[^>/]*/>|<ref[^>]*>.*?</ref>|<sup[^>]*>.*?</sup>`)
	tagRE      = regexp.MustCompile(`<[^>]+>`)
)

// plainText removes links (keeping their labels), templates, references,
// tags and bold/italic quotes.
func plainText(s string) string {
	s = refRE.ReplaceAllString(s, "")
	s = linkRE.ReplaceAllStringFunc(s, func(m string) string {
		sm := linkRE.FindStringSubmatch(m)
		if sm[2] != "" {
			return sm[2]
		}
		return sm[1]
	})
	for templateRE.MatchString(s) {
		s = templateRE.ReplaceAllString(s, "")
	}
	s = tagRE.ReplaceAllString(s, "")
	s = strings.ReplaceAll(s, "'''", "")
	s = strings.ReplaceAll(s, "''", "")
	s = strings.ReplaceAll(s, "&nbsp;", " ")
	return strings.Join(strings.Fields(s), " ")
}

// Position of a player.
const (
	PosNone = iota
	PosGoal
	PosDefence
	PosMidfield
	PosAttack
)

// ParsePosition reads a position as the squad articles and Wikidata name
// it; PosNone when it is none of the four.
func ParsePosition(s string) int {
	l := strings.ToLower(s)
	switch {
	case l == "tor" || strings.Contains(l, "torwart") || strings.Contains(l, "torhüter") || strings.Contains(l, "goalkeeper"):
		return PosGoal
	case strings.Contains(l, "abwehr") || strings.Contains(l, "verteidig") || strings.Contains(l, "libero") ||
		strings.Contains(l, "defender") || strings.Contains(l, "back"):
		return PosDefence
	case strings.Contains(l, "mittelfeld") || strings.Contains(l, "midfield"):
		return PosMidfield
	case l == "sturm" || l == "angriff" || strings.Contains(l, "stürmer") || strings.Contains(l, "flügel") ||
		strings.Contains(l, "forward") || strings.Contains(l, "striker") || strings.Contains(l, "winger"):
		return PosAttack
	}
	return PosNone
}

// Player is a member of a squad.
type Player struct {
	Name    string // as the article shows it
	Surname string
	Pos     int
	Games   int // league games in the season, -1 when unknown
	Since   int // the year the player joined the club, 0 when unknown
	Source  string
}

// Squad is the squad of a club in a squad article.
type Squad struct {
	Club    string // the linked article
	Label   string
	Logo    string // the file of the logo of the season, "Datei:..."
	Players []Player
}

var sortValueRE = regexp.MustCompile(`data-sort-value="([^"]*)"`)

// personCell matches the templates that write a name sortable by surname:
// {{PersonZelle|Vorname|Nachname}}, {{SortKeyName|Vorname|Nachname}}.
var personCell = regexp.MustCompile(`\{\{\s*(PersonZelle|SortKeyName|SortName)\s*\|`)

// Squads parses a squad article ("Kader der deutschen Fußball-Bundesliga
// 1990/91"): a section per club with its logo and a table of players.
func Squads(text string) []Squad {
	var out []Squad
	var cur *Squad
	var row []string
	flush := func() {
		if cur != nil && len(row) > 0 {
			if p, ok := squadRow(strings.Join(row, "\n")); ok {
				cur.Players = append(cur.Players, p)
			}
		}
		row = row[:0]
	}
	for _, line := range strings.Split(text, "\n") {
		if _, n, ok := heading(line); ok {
			flush()
			cur = nil
			if n >= 2 {
				club, label := firstLink(line)
				if club == "" {
					// Newer articles name the club without a link.
					h, _, _ := heading(line)
					club, label = h, h
				}
				if club != "" {
					out = append(out, Squad{Club: club, Label: label})
					cur = &out[len(out)-1]
				}
			}
			continue
		}
		if cur == nil {
			continue
		}
		if cur.Logo == "" {
			if m := linkRE.FindStringSubmatch(line); m != nil && isFileLink(strings.TrimSpace(m[1])) {
				cur.Logo = "Datei:" + strings.TrimSpace(strings.SplitN(m[1], ":", 2)[1])
			}
		}
		t := strings.TrimSpace(line)
		switch {
		case strings.HasPrefix(t, "|-") || strings.HasPrefix(t, "|}"):
			flush()
		case strings.HasPrefix(t, "|"):
			row = append(row, t)
		}
	}
	flush()
	// Sections without players (legend, notes) are no squads.
	squads := out[:0]
	for _, s := range out {
		if len(s.Players) > 0 {
			squads = append(squads, s)
		}
	}
	return squads
}

// squadRow reads a table row of a squad article.
func squadRow(row string) (Player, bool) {
	row = strings.ReplaceAll(row, "\n|", "||")
	cells := splitTop(strings.TrimPrefix(strings.TrimSpace(row), "|"), "||")
	p := Player{Games: -1, Source: "wikipedia"}
	posCell := -1
	for i, c := range cells {
		content := cellContent(c)
		switch {
		case p.Name == "" && personCell.MatchString(c):
			body, _ := templateBody(c[personCell.FindStringIndex(c)[0]:])
			parts := splitTop(body, "|")
			var pos []string
			for _, a := range parts[1:] {
				if !strings.Contains(a, "=") {
					pos = append(pos, disambig.ReplaceAllString(plainText(a), ""))
				}
			}
			if len(pos) == 0 {
				continue
			}
			if strings.Contains(pos[0], " ") || len(pos) == 1 {
				// {{SortKeyName|Vorname Nachname|Linkziel}}
				p.Name, p.Surname = pos[0], Surname(pos[0])
			} else {
				// {{PersonZelle|Vorname|Nachname}}
				p.Name, p.Surname = pos[0]+" "+pos[1], pos[1]
			}
		case p.Name == "" && strings.Contains(c, "[["):
			_, label := firstLink(c)
			if label == "" {
				continue
			}
			p.Name = label
			if m := sortValueRE.FindStringSubmatch(c); m != nil && strings.Contains(m[1], ",") {
				p.Surname = strings.TrimSpace(strings.SplitN(m[1], ",", 2)[0])
			}
		case p.Name != "" && posCell < 0:
			if pos := ParsePosition(content); pos != PosNone {
				p.Pos, posCell = pos, i
			}
		case posCell >= 0 && i == posCell+1:
			if n, err := strconv.Atoi(strings.Fields(content + " x")[0]); err == nil {
				p.Games = n
			} else if strings.Contains(content, "ohne Einsatz") {
				p.Games = 0
			}
		}
	}
	if p.Name == "" || p.Pos == PosNone {
		return Player{}, false
	}
	if p.Surname == "" {
		p.Surname = Surname(p.Name)
	}
	return p, true
}

// cellContent is the text of a table cell without its attributes
// ('style="..." | text').
func cellContent(c string) string {
	parts := splitTop(c, "|")
	return plainText(parts[len(parts)-1])
}

// Surname guesses the surname of a name: the last word, or the name itself
// for a single name (Brazilian players).
func Surname(name string) string {
	name, _, _ = strings.Cut(name, " (")
	f := strings.Fields(name)
	if len(f) == 0 {
		return ""
	}
	// Particles stay with the surname: "van Gaal", "de Jong".
	i := len(f) - 1
	for i > 0 {
		switch strings.ToLower(f[i-1]) {
		case "van", "von", "de", "da", "di", "dos", "der", "den", "le", "la", "ter", "ten":
			i--
			continue
		}
		break
	}
	return strings.Join(f[i:], " ")
}

var (
	commentRE = regexp.MustCompile(`(?s)<!--.*?-->`)
	disambig  = regexp.MustCompile(`\s*\([^)]*\)$`)
)

// CurrentSquad reads the squad of the first team in a club article: the
// section "Aktueller Kader" or "Kader <season>", a table whose header rows
// name the position ("! colspan=6 | Torwart") and whose rows hold the
// players. Players in comments (loans) are left out.
func CurrentSquad(text, season string) []Player {
	text = commentRE.ReplaceAllString(text, "")
	var out []Player
	in, pos := false, PosNone
	level := 0
	var heads [6]string
	for _, line := range strings.Split(text, "\n") {
		if h, n, ok := heading(line); ok {
			if in && n <= level {
				break
			}
			if n < len(heads) {
				heads[n] = h
				for k := n + 1; k < len(heads); k++ {
					heads[k] = ""
				}
			}
			l := strings.ToLower(h)
			if !in && (strings.Contains(l, "aktueller kader") || strings.Contains(l, "kader") && strings.Contains(h, season)) &&
				!otherTeam(heads[:n]) {
				in, level = true, n
			}
			continue
		}
		if !in {
			continue
		}
		t := strings.TrimSpace(line)
		switch {
		case strings.HasPrefix(t, "!"):
			if p := ParsePosition(cellContent(strings.TrimPrefix(t, "!"))); p != PosNone {
				pos = p
			}
		case strings.HasPrefix(t, "|") && !strings.HasPrefix(t, "|-") && !strings.HasPrefix(t, "|}") && pos != PosNone:
			if name := rowName(splitTop(strings.TrimPrefix(t, "|"), "||")); name != "" {
				out = append(out, Player{Name: name, Surname: Surname(name), Pos: pos, Games: -1, Source: "wikipedia"})
			}
		}
	}
	return out
}

// otherTeam reports whether headings lead to a team other than the first:
// the second team, youth, women.
func otherTeam(heads []string) bool {
	for _, h := range heads {
		l := strings.ToLower(h)
		for _, w := range []string{"zweite", "u23", "u21", "u19", "jugend", "frauen", "junior", "amateur", "ii"} {
			if strings.Contains(l, w) {
				return true
			}
		}
	}
	return false
}

var (
	digits = regexp.MustCompile(`\d`)
	// clubLike matches what names a club or a youth team, not a player.
	clubLike = regexp.MustCompile(`[\[(]|Jugend|\b(FC|SV|VfL|VfB|TSV|SC|SG|FSV|SpVgg|Borussia|Eintracht|Hansa|Tennis|Kickers)\b`)
)

// playerLike reports whether a cell text can be the name of a player.
func playerLike(s string) bool { return s != "" && !digits.MatchString(s) && !clubLike.MatchString(s) }

// rowName finds the player in the cells of a squad row: the left-aligned
// cell without digits (a birthday is left-aligned too), else the first
// person link.
func rowName(cells []string) string {
	for _, c := range cells {
		if strings.Contains(c, "text-align:left") {
			if n := cellContent(c); playerLike(n) {
				return n
			}
		}
	}
	for _, c := range cells {
		if strings.Contains(c, "[[") {
			if _, label := firstLink(c); playerLike(label) {
				return label
			}
		}
	}
	return ""
}
