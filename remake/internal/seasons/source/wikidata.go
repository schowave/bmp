package source

import (
	"fmt"
	"net/url"
	"sort"
	"strconv"
	"strings"
)

const sparqlURL = "https://query.wikidata.org/sparql"

type sparqlResult struct {
	Results struct {
		Bindings []map[string]struct{ Value string }
	}
}

func (c *Client) sparql(query string) ([]map[string]string, error) {
	var r sparqlResult
	u := sparqlURL + "?" + url.Values{"query": {query}, "format": {"json"}}.Encode()
	if err := c.getJSON(u, &r); err != nil {
		return nil, err
	}
	out := make([]map[string]string, len(r.Results.Bindings))
	for i, b := range r.Results.Bindings {
		out[i] = map[string]string{}
		for k, v := range b {
			out[i][k] = v.Value
		}
	}
	return out, nil
}

// quote writes s as a SPARQL string literal.
func quote(s string) string {
	return `"` + strings.NewReplacer(`\`, `\\`, `"`, `\"`).Replace(s) + `"`
}

// titleValues is a VALUES clause binding ?title to German article titles.
func titleValues(titles []string) string {
	var b strings.Builder
	b.WriteString("VALUES ?title {")
	for _, t := range titles {
		b.WriteString(" " + quote(t) + "@de")
	}
	b.WriteString(" }\n?article schema:about ?club ; schema:isPartOf <https://de.wikipedia.org/> ; schema:name ?title .\n")
	return b.String()
}

// WikidataSquads returns the players Wikidata lists for the clubs (German
// article titles) in the season that starts in year: everyone whose
// membership overlaps August of year to May of the next. Games are the
// appearances of the membership when Wikidata has them, else -1.
func (c *Client) WikidataSquads(titles []string, year int) (map[string][]Player, error) {
	out := map[string][]Player{}
	for i := 0; i < len(titles); i += 20 {
		batch := titles[i:min(i+20, len(titles))]
		q := fmt.Sprintf(`SELECT ?title ?p ?pLabel ?familyLabel ?posLabel ?apps ?a ?b WHERE {
%s?p p:P54 ?st . ?st ps:P54 ?club ; pq:P580 ?a .
OPTIONAL { ?st pq:P582 ?b }
OPTIONAL { ?st pq:P1350 ?apps }
FILTER(?a <= "%d-05-31T00:00:00Z"^^xsd:dateTime && (!BOUND(?b) || ?b >= "%d-08-01T00:00:00Z"^^xsd:dateTime))
OPTIONAL { ?p wdt:P413 ?pos }
OPTIONAL { ?p wdt:P734 ?family }
SERVICE wikibase:label { bd:serviceParam wikibase:language "de,en". }
}`, titleValues(batch), year+1, year)
		rows, err := c.sparql(q)
		if err != nil {
			return nil, err
		}
		type key struct{ club, p string }
		seen := map[key]int{}
		for _, r := range rows {
			k := key{r["title"], r["p"]}
			pos := ParsePosition(r["posLabel"])
			if j, ok := seen[k]; ok {
				// A player with several positions: keep the first known.
				if out[k.club][j].Pos == PosNone {
					out[k.club][j].Pos = pos
				}
				continue
			}
			name := r["pLabel"]
			if strings.HasPrefix(name, "Q") {
				continue // no label
			}
			sur := r["familyLabel"]
			if sur == "" || strings.HasPrefix(sur, "Q") {
				sur = Surname(name)
			}
			games := -1
			if n, err := strconv.Atoi(r["apps"]); err == nil {
				games = n
			}
			since, _ := strconv.Atoi(r["a"][:min(4, len(r["a"]))])
			// A membership without an end that began long ago is mostly
			// one whose end nobody entered.
			if r["b"] == "" && since > 0 && since < year-10 {
				continue
			}
			seen[k] = len(out[k.club])
			out[k.club] = append(out[k.club], Player{Name: name, Surname: sur, Pos: pos, Games: games, Since: since, Source: "wikidata"})
		}
	}
	for club := range out {
		ps := out[club]
		sort.SliceStable(ps, func(a, b int) bool { return ps[a].Name < ps[b].Name })
	}
	return out, nil
}

// Club is what Wikidata says about a club.
type Club struct {
	Title    string // German article
	Country  string // ISO 3166-1 alpha-2, e.g. "DE"
	Football bool   // an association football club
}

// WikidataClubs looks up the articles: whether each is a football club and
// its country.
func (c *Client) WikidataClubs(titles []string) (map[string]Club, error) {
	out := map[string]Club{}
	for i := 0; i < len(titles); i += 100 {
		batch := titles[i:min(i+100, len(titles))]
		q := fmt.Sprintf(`SELECT ?title ?iso (COUNT(?fc) AS ?football) WHERE {
%sOPTIONAL { ?club wdt:P31/wdt:P279* wd:Q476028 . BIND(1 AS ?fc) }
OPTIONAL { ?club wdt:P17 ?country . ?country wdt:P297 ?iso }
} GROUP BY ?title ?iso`, titleValues(batch))
		rows, err := c.sparql(q)
		if err != nil {
			return nil, err
		}
		for _, r := range rows {
			cl := out[r["title"]]
			cl.Title = r["title"]
			if r["iso"] != "" && cl.Country == "" {
				cl.Country = r["iso"]
			}
			if n, _ := strconv.Atoi(r["football"]); n > 0 {
				cl.Football = true
			}
			out[r["title"]] = cl
		}
	}
	return out, nil
}
