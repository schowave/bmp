package source

import (
	"errors"
	"fmt"
	"regexp"
	"sort"
	"strings"
	"time"
)

// RunningSeason is the first year of the season being played now: a
// season starts in July.
func RunningSeason() int {
	now := time.Now()
	if now.Month() < time.July {
		return now.Year() - 1
	}
	return now.Year()
}

// SeasonTitle is the season as Wikipedia writes it: "1995/96", "1999/2000".
func SeasonTitle(year int) string {
	if year == 1999 {
		return "1999/2000"
	}
	return fmt.Sprintf("%d/%02d", year, (year+1)%100)
}

// TierRow is a row of a league table and the tier of the league: 1 =
// Bundesliga, 2 = 2. Bundesliga, 3 = the third tier of the year (Oberliga
// until 1994, Regionalliga until 2008, then 3. Liga), 4 = the Regionalliga
// from 2008 on.
type TierRow struct {
	Tier int
	Row
}

// SecondTeam matches the second teams of clubs ("Werder Bremen II",
// "Bayern München Amateure"); their articles often redirect to the club's.
var secondTeam = regexp.MustCompile(`(?i)\b(II|Amateure|U ?2[0-3])\b`)

// IsSecondTeam reports whether a club or its name is a second team.
func IsSecondTeam(s string) bool { return secondTeam.MatchString(s) }

// EuroClub is a club of a European cup of the season.
type EuroClub struct {
	Cup      int // 0 Landesmeister/Champions League, 1 Pokalsieger, 2 UEFA-Pokal/Europa League, 3 Conference League, 4 Intertoto
	Club     string
	Label    string
	Country  string
	Mentions int // how often the article links the club: more for a club that went further
}

// Season is everything fetched for one season.
type Season struct {
	Year     int
	Tables   []TierRow
	Squads   map[string][]Player // by club
	Logos    map[string]string   // by club: the logo file of the season
	Colors   map[string]string   // by club: its colours, "Schwarz-Gelb"
	Europe   []EuroClub
	Articles []string // the articles read, for the credits
	Missing  []string // problems worth a look by hand
}

// thirdTier returns the articles of the third tier of year (tier 3) and,
// from 2008 on, of the Regionalliga (tier 4).
func (c *Client) tierArticles(year int) (map[int][]string, error) {
	s := SeasonTitle(year)
	arts := map[int][]string{1: {"Fußball-Bundesliga " + s}, 2: {"2. Fußball-Bundesliga " + s}}
	switch {
	case year >= 2008:
		arts[3] = []string{"3. Fußball-Liga " + s}
		arts[4] = []string{"Fußball-Regionalliga " + s}
	case year >= 1994:
		arts[3] = []string{"Fußball-Regionalliga " + s}
	default:
		// The Oberliga: one article per group, listed in the overview.
		text, err := c.Raw("Fußball-Oberliga " + s)
		if err != nil {
			return nil, err
		}
		arts[3] = groupArticles(text, s, "liga")
		if len(arts[3]) == 0 {
			return nil, fmt.Errorf("Fußball-Oberliga %s: no groups", s)
		}
	}
	return arts, nil
}

// groupArticles are the articles of the groups of a league of season s
// that an overview links: their titles end with the season and contain
// kind ("liga", "Regionalliga").
func groupArticles(overview, s, kind string) []string {
	var out []string
	seen := map[string]bool{}
	for _, l := range Links(overview) {
		ll := strings.ToLower(l)
		if strings.HasSuffix(l, s) && strings.Contains(ll, strings.ToLower(kind)) && !strings.Contains(ll, "verbandsliga") &&
			!strings.Contains(ll, "landesliga") && !strings.Contains(ll, "bundesliga") && !strings.Contains(ll, "frauen") &&
			!seen[l] {
			seen[l] = true
			out = append(out, l)
		}
	}
	return out
}

// squadArticles are the squad articles of a season by tier.
func squadArticles(year int) map[int]string {
	s := SeasonTitle(year)
	m := map[int]string{1: "Kader der deutschen Fußball-Bundesliga " + s, 2: "Kader der 2. Fußball-Bundesliga " + s}
	if year >= 2008 {
		m[3] = "Kader der 3. Fußball-Liga " + s
	}
	return m
}

// euroArticles are the European cups of year.
func euroArticles(year int) map[int]string {
	s := SeasonTitle(year)
	m := map[int]string{}
	if year <= 1991 {
		m[0] = "Europapokal der Landesmeister " + s
	} else {
		m[0] = "UEFA Champions League " + s
	}
	if year <= 1998 {
		m[1] = "Europapokal der Pokalsieger " + s
	}
	if year <= 2008 {
		m[2] = "UEFA-Pokal " + s
	} else {
		m[2] = "UEFA Europa League " + s
	}
	switch {
	case year >= 2024:
		m[3] = "UEFA Conference League " + s
	case year >= 2021:
		m[3] = "UEFA Europa Conference League " + s
	}
	if year >= 1995 && year <= 2008 {
		m[4] = fmt.Sprintf("UEFA Intertoto Cup %d", year)
	}
	return m
}

// FetchSeason fetches the season that starts in year.
func (c *Client) FetchSeason(year int) (*Season, error) {
	out := &Season{Year: year, Squads: map[string][]Player{}, Logos: map[string]string{}, Colors: map[string]string{}}
	arts, err := c.tierArticles(year)
	if err != nil {
		return nil, err
	}
	for tier := 1; tier <= 4; tier++ {
		// Index loop: an overview appends the articles of its groups.
		for i := 0; i < len(arts[tier]); i++ {
			a := arts[tier][i]
			text, err := c.Raw(a)
			if errors.Is(err, ErrNotFound) {
				out.Missing = append(out.Missing, "article missing: "+a)
				continue
			}
			if err != nil {
				return nil, err
			}
			out.Articles = append(out.Articles, a)
			rows := Table(text)
			group := ""
			if len(arts[tier]) > 1 {
				group = strings.TrimSpace(strings.TrimSuffix(a, SeasonTitle(year)))
			}
			if len(rows) == 0 && len(arts[tier]) == 1 && tier >= 3 {
				// An overview of groups with an article each (the
				// Regionalliga from 2012 on).
				if groups := groupArticles(text, SeasonTitle(year), "Regionalliga"); len(groups) > 0 {
					arts[tier] = append(arts[tier], groups...)
					continue
				}
			}
			if len(rows) == 0 {
				out.Missing = append(out.Missing, "no table: "+a)
			}
			for _, r := range rows {
				if group != "" {
					r.Group = group
				}
				out.Tables = append(out.Tables, TierRow{Tier: tier, Row: r})
			}
		}
	}
	// Squads: the squad articles, then Wikidata for the clubs without.
	for _, a := range squadArticles(year) {
		text, err := c.Raw(a)
		if errors.Is(err, ErrNotFound) {
			continue
		}
		if err != nil {
			return nil, err
		}
		out.Articles = append(out.Articles, a)
		for _, sq := range Squads(text) {
			out.Squads[sq.Club] = append(out.Squads[sq.Club], sq.Players...)
			if sq.Logo != "" {
				out.Logos[sq.Club] = sq.Logo
			}
		}
	}
	// One name per club: the article the title redirects to.
	var titles []string
	for _, r := range out.Tables {
		titles = append(titles, r.Club)
	}
	for club := range out.Squads {
		titles = append(titles, club)
	}
	canon, err := c.Resolve(unique(titles))
	if err != nil {
		return nil, err
	}
	name := func(t string) string {
		if IsSecondTeam(t) {
			return t // not the club its article redirects to
		}
		if n := canon[t]; n != "" {
			return n
		}
		return t
	}
	for i := range out.Tables {
		out.Tables[i].Club = name(out.Tables[i].Club)
	}
	squads, logos := map[string][]Player{}, map[string]string{}
	for club, ps := range out.Squads {
		squads[name(club)] = append(squads[name(club)], ps...)
	}
	for club, l := range out.Logos {
		logos[name(club)] = l
	}
	out.Squads, out.Logos = squads, logos
	var without []string
	for _, r := range out.Tables {
		if len(out.Squads[r.Club]) < 11 && !IsSecondTeam(r.Club) {
			without = append(without, r.Club)
		}
	}
	without = unique(without)
	if len(without) > 0 {
		wd, err := c.WikidataSquads(without, year)
		if err != nil {
			return nil, err
		}
		for club, ps := range wd {
			if len(ps) > len(out.Squads[club]) {
				out.Squads[club] = ps
			}
		}
	}
	// The running season: the squad in the club article, unless a squad
	// article has one.
	if year >= RunningSeason() {
		for _, club := range unique(clubsOf(out.Tables)) {
			if len(out.Squads[club]) >= 18 || IsSecondTeam(club) {
				continue
			}
			text, err := c.Raw(club)
			if errors.Is(err, ErrNotFound) {
				continue
			}
			if err != nil {
				return nil, err
			}
			if ps := CurrentSquad(text, SeasonTitle(year)); len(ps) > len(out.Squads[club]) {
				out.Squads[club] = ps
			}
		}
	}
	for _, club := range without {
		if len(out.Squads[club]) < 11 {
			out.Missing = append(out.Missing, fmt.Sprintf("squad of %s: %d players", club, len(out.Squads[club])))
		}
	}
	// The current logo and the colours from the infobox of the club.
	var clubs []string
	for _, r := range out.Tables {
		clubs = append(clubs, r.Club)
	}
	for _, club := range unique(clubs) {
		info, err := c.ClubInfobox(club)
		if err != nil {
			return nil, err
		}
		if out.Logos[club] == "" {
			out.Logos[club] = info.Logo
		}
		out.Colors[club] = info.Colors
	}
	if err := c.fetchEurope(out); err != nil {
		return nil, err
	}
	return out, nil
}

// fetchEurope collects the foreign clubs of the European cups.
func (c *Client) fetchEurope(out *Season) error {
	type ref struct {
		cup   int
		count int
		label string
	}
	refs := map[string]*ref{}
	var titles []string
	arts := euroArticles(out.Year)
	cups := make([]int, 0, len(arts))
	for cup := range arts {
		cups = append(cups, cup)
	}
	sort.Ints(cups)
	for _, cup := range cups {
		a := arts[cup]
		text, err := c.Raw(a)
		if errors.Is(err, ErrNotFound) {
			out.Missing = append(out.Missing, "article missing: "+a)
			continue
		}
		if err != nil {
			return err
		}
		out.Articles = append(out.Articles, a)
		for _, m := range linkRE.FindAllStringSubmatch(text, -1) {
			t := strings.TrimSpace(m[1])
			if t == "" || strings.Contains(t, ":") {
				continue
			}
			r := refs[t]
			if r == nil {
				label := plainText(m[2])
				if label == "" {
					label = t
				}
				r = &ref{cup: cup, label: label}
				refs[t] = r
				titles = append(titles, t)
			}
			if r.cup == cup {
				r.count++
			}
		}
	}
	canon, err := c.Resolve(titles)
	if err != nil {
		return err
	}
	var real []string
	for _, t := range titles {
		if canon[t] != "" {
			real = append(real, canon[t])
		}
	}
	info, err := c.WikidataClubs(unique(real))
	if err != nil {
		return err
	}
	best := map[string]*EuroClub{}
	for _, t := range titles {
		club := canon[t]
		in, ok := info[club]
		if club == "" || !ok || !in.Football || in.Country == "" || in.Country == "DE" {
			continue
		}
		r := refs[t]
		e := best[club]
		if e == nil {
			e = &EuroClub{Cup: r.cup, Club: club, Label: r.label, Country: in.Country}
			best[club] = e
		}
		if r.cup < e.Cup {
			e.Cup = r.cup
		}
		if r.cup == e.Cup {
			e.Mentions += r.count
		}
	}
	for _, e := range best {
		out.Europe = append(out.Europe, *e)
	}
	sort.Slice(out.Europe, func(a, b int) bool {
		x, y := out.Europe[a], out.Europe[b]
		if x.Cup != y.Cup {
			return x.Cup < y.Cup
		}
		if x.Mentions != y.Mentions {
			return x.Mentions > y.Mentions
		}
		return x.Club < y.Club
	})
	return nil
}

func clubsOf(rows []TierRow) []string {
	out := make([]string, len(rows))
	for i, r := range rows {
		out[i] = r.Club
	}
	return out
}

func unique(xs []string) []string {
	seen := map[string]bool{}
	var out []string
	for _, x := range xs {
		if x != "" && !seen[x] {
			seen[x] = true
			out = append(out, x)
		}
	}
	return out
}
