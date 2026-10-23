package game

import (
	"strings"
	"testing"
)

// origFormat fills a message format of D:0x267c like f_3091_0190 and turns
// it into the remake form: '^' -> "\n", the final '^' dropped, umlauts of
// the font encoding decoded.
func origFormat(f string, args ...string) string {
	var b strings.Builder
	for _, c := range f {
		if c == '$' || c == '#' {
			b.WriteString(args[0])
			args = args[1:]
			continue
		}
		b.WriteRune(c)
	}
	s := strings.TrimSuffix(b.String(), "^")
	return strings.NewReplacer("^", "\n", "[", "Ä", "|", "ö", "]", "Ü", "{", "ä", "}", "ü", "~", "ß").Replace(s)
}

func TestMessageTextsMatchOriginalFormats(t *testing.T) {
	cases := []struct{ got, want string }{
		{msgOfferExtension("TRAUTNER", 1, 3), origFormat("$ bietet an,^von # auf #^Jahre zu verl{ngern.^", "TRAUTNER", "1", "3")},
		{msgInterested("FC BAYERN", "TRAUTNER"), origFormat("$^ist an $^interessiert.^", "FC BAYERN", "TRAUTNER")},
		{msgRetires("TRAUTNER", 34), origFormat("$ h{ngt den^Fu~balljob im Alter von^# Jahren an den Nagel.^", "TRAUTNER", "34")},
		{msgNoExtension("TRAUTNER"), origFormat("$ k}ndigt an,^da~ er seinen Vertrag nicht^mehr verl{ngern wird.^", "TRAUTNER")},
		{msgInjury("TRAUTNER", "Bauchmuskelzerrung"), origFormat("$^$^$ ^", "Verletzung im Training:", "TRAUTNER", "(Bauchmuskelzerrung)")},
		{msgRiot(1234000), origFormat("$^$^$ ^", "Randalierer im Stadion", "richteten einen Sachschaden", "von 1.234.000 DM an.")},
		{msgCompletion(0), origFormat("$^$^$ ^", "Der Ausbau der Sitzpl{tze", "ist abgeschlossen.", "")},
		{msgCompletion(3), origFormat("$^$^$ ^", "Die Errichtung der neuen", "Flutlichtmasten", "ist abgeschlossen.")},
	}
	for _, c := range cases {
		if c.got != c.want {
			t.Errorf("got %q, want %q", c.got, c.want)
		}
	}
	if got := msgSeasonEnd(3, 1, 1992); got != "Saisonende\nSie beendeten Ihre Saison\n3.Platz der ZWEITE LIGA" {
		t.Errorf("season end: %q", got)
	}
}
