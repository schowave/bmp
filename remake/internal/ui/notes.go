package ui

import (
	"github.com/hajimehoshi/ebiten/v2"

	"github.com/schowave/bmp/remake/internal/game"
)

// The remake's hidden keys o leave what they did as a note on the screen,
// in a free place, instead of a message that goes again: in the footer of the
// lineup, training and stadium screens while the screen is open, and in the
// middle of the main menu until the turn ends. The notes of all of them are
// written here.

// optimizeRune is the hidden key of the remake that optimises.
const optimizeRune = 'o'

// optimizeKey reports the key o among the keys typed this tick.
func optimizeKey(in *Input) bool {
	for _, c := range in.Chars {
		if c == optimizeRune {
			return true
		}
	}
	return false
}

// lineupNote is the note of the lineup key: the rating and its gain, the two
// changes worth most, the opponent and the automatic lineups of the three
// formations.
func lineupNote(s *game.State, mi int, res game.OptimalLineup) []string {
	note := []string{"OPTIMAL " + pointsText(res.After) + " " + signedDecimal(res.After-res.Before, 2)}
	if len(res.Changes) == 0 {
		note = append(note, "AUFSTELLUNG BLEIBT")
	}
	for k, ch := range res.Changes {
		if k == 2 {
			break
		}
		note = append(note, changeNote(s, mi, ch))
	}
	note = append(note, opponentNote(s, res.Opponent, res.Home)...)
	return append(note,
		"STAND. "+pointsText(res.Auto[0])+"/"+pointsText(res.Auto[1])+"/"+pointsText(res.Auto[2]))
}

// gainText is the value of a change with a blank before it, "" if it is not
// rated.
func gainText(ch game.LineupChange) string {
	if !ch.Rated {
		return ""
	}
	return " " + signedDecimal(ch.Gain, 2)
}

// changeNote names a change of the optimal lineup: "MÜLLER FÜR MEIER +0,06",
// "HOFF TAUSCHT MIT KRAUS +0,05", "HOFF UMGESTELLT +0,03" or "MEIER RAUS".
func changeNote(s *game.State, mi int, ch game.LineupChange) string {
	name := func(i int) string { return s.Persons[s.Squads[mi][i].Person].Name }
	switch {
	case ch.In < 0:
		return name(ch.Out) + " RAUS"
	case ch.With >= 0:
		return name(ch.In) + " TAUSCHT MIT " + name(ch.With) + gainText(ch)
	case ch.Out < 0:
		return name(ch.In) + " UMGESTELLT" + gainText(ch)
	}
	return name(ch.In) + " FÜR " + name(ch.Out) + gainText(ch)
}

// changeHint is the footer for slot i after the lineup key: why the player
// came in, went out or moved, "FÜR MEIER +0,06 (ERSCH. 138)", or for one it
// left alone why he plays or not (reasonHint). A move worth +0,00 helps only with the others; a
// change that cannot be undone alone has no value.
func changeHint(s *game.State, mi int, res *game.OptimalLineup, i int) string {
	sq := &s.Squads[mi]
	name := func(j int) string { return s.Persons[sq[j].Person].Name }
	// why is the value of the change and what speaks against the player of
	// slot j.
	why := func(j int, ch game.LineupChange) string {
		switch {
		case sq[j].Status&3 == 2:
			return " (VERLETZT)"
		case sq[j].Status&3 == 1 && !s.CupDay:
			return " (GESPERRT)"
		case res.Load[j] > 0:
			return gainText(ch) + " (ERSCH. " + itoa(sq[j].Fatigue) + ")"
		}
		return gainText(ch)
	}
	for _, ch := range res.Changes {
		switch {
		case ch.In == i && ch.Out >= 0:
			return "FÜR " + name(ch.Out) + why(ch.Out, ch)
		case ch.In == i && ch.With >= 0:
			return "TAUSCHT MIT " + name(ch.With) + gainText(ch)
		case ch.With == i:
			return "TAUSCHT MIT " + name(ch.In) + gainText(ch)
		case ch.In == i:
			return "UMGESTELLT" + gainText(ch)
		case ch.Out == i && ch.In >= 0:
			return "RAUS, " + name(ch.In) + " SPIELT" + why(i, ch)
		case ch.Out == i:
			return "RAUS" + why(i, ch)
		}
	}
	return reasonHint(s, mi, res, i)
}

// reasonHint says why the lineup key let the player of slot i play or not:
// "SPIELT, +0,05 VOR POVLSEN" against the best one who could come in for
// him, "BANK, -0,04 STATT ZORC (ERSCH. 140)" against the one he would best
// replace, "FEHLT: VERLETZT" for one who cannot play. A difference below
// game.LineupMinGain, for which the key changes nothing, is "GLEICH GUT".
func reasonHint(s *game.State, mi int, res *game.OptimalLineup, i int) string {
	sl := &s.Squads[mi][i]
	r := res.Reasons[i]
	tired := ""
	if res.Load[i] > 0 {
		tired = " (ERSCH. " + itoa(sl.Fatigue) + ")"
	}
	switch {
	case sl.Status&3 == 2:
		return "FEHLT: VERLETZT"
	case sl.Status&3 == 1 && !s.CupDay:
		return "FEHLT: GESPERRT"
	case sl.InTeam() && r.Alt < 0:
		return "SPIELT, KEIN ERSATZ" + tired
	case r.Alt < 0:
		return "BANK, KEIN PLATZ FÜR IHN"
	}
	alt := s.Persons[s.Squads[mi][r.Alt].Person].Name
	switch {
	case r.Margin > -game.LineupMinGain && r.Margin < game.LineupMinGain && sl.InTeam():
		return "SPIELT, " + alt + " GLEICH GUT" + tired
	case r.Margin > -game.LineupMinGain && r.Margin < game.LineupMinGain:
		return "BANK, GLEICH GUT WIE " + alt + tired
	case sl.InTeam():
		return "SPIELT, " + signedDecimal(r.Margin, 2) + " VOR " + alt + tired
	}
	return "BANK, " + signedDecimal(r.Margin, 2) + " STATT " + alt + tired
}

// trainingNote is the note of the training key: the strength with the new
// plan and with the one before.
func trainingNote(res game.OptimalTraining) []string {
	return []string{"OPTIMAL " + tenthsText(res.After) + "  BISHER " + tenthsText(res.Before)}
}

// priceNote is the note of the ticket price key: the price, its gain, the
// gate money and the opponent, or that no home match is left.
func priceNote(s *game.State, tp game.OptimalPrice) []string {
	if tp.Opponent < 0 {
		return []string{"KEIN HEIMSPIEL MEHR IN DER SAISON"}
	}
	return append([]string{priceLine(tp) + "  EINNAHME " + game.FormatMoney(tp.AfterGate)},
		opponentNote(s, tp.Opponent, true)...)
}

// turnNote is the note of the key of the main menu: each rating with its
// gain against the one before, and the opponent of the lineup.
func turnNote(s *game.State, ln game.OptimalLineup, tr game.OptimalTraining, tp game.OptimalPrice) []string {
	note := []string{
		"AUFSTELLUNG " + pointsText(ln.After) + " " + signedDecimal(ln.After-ln.Before, 2),
		"TRAINING " + tenthsText(tr.After) + " " + signedDecimal(tr.After-tr.Before, 1),
	}
	if tp.Opponent >= 0 {
		note = append(note, priceLine(tp))
	}
	return append(note, opponentNote(s, ln.Opponent, ln.Home)...)
}

// priceLine names the optimal ticket price and what it gains at the next
// home match: "EINTRITT 14 +12.000".
func priceLine(tp game.OptimalPrice) string {
	return "EINTRITT " + itoa(tp.After) + " " + withSign(tp.AfterGate-tp.BeforeGate >= 0, game.FormatMoney(tp.AfterGate-tp.BeforeGate))
}

// opponentNote names the opponent (a club id, -1 none) a key worked for:
// "GEGEN X" for a home match, "BEI X" for an away match.
func opponentNote(s *game.State, opponent int, home bool) []string {
	if opponent < 0 {
		return nil
	}
	if !home {
		return []string{"BEI " + s.Teams[opponent].Name}
	}
	return []string{"GEGEN " + s.Teams[opponent].Name}
}

// pointsText writes n hundredths of a point with a decimal comma, "1,35".
func pointsText(n int) string { return decimalText(n, 2) }

// tenthsText writes n tenths with a decimal comma, "41,0".
func tenthsText(n int) string { return decimalText(n, 1) }

// signedDecimal is decimalText with a plus for a gain: "+0,21", "-0,05".
func signedDecimal(n, places int) string { return withSign(n >= 0, decimalText(n, places)) }

// decimalText writes n with places digits after a decimal comma:
// decimalText(135, 2) is "1,35", decimalText(410, 1) "41,0".
func decimalText(n, places int) string {
	pre := ""
	if n < 0 {
		pre, n = "-", -n
	}
	div := 1
	for range places {
		div *= 10
	}
	frac := itoa(n % div)
	for len(frac) < places {
		frac = "0" + frac
	}
	return pre + itoa(n/div) + "," + frac
}

// withSign puts a plus before the text of a number that is not negative (the
// text of a negative one has its minus).
func withSign(gain bool, text string) string {
	if gain {
		return "+" + text
	}
	return text
}

// noteLines are the lines of a note as drawNote shows them: each cut to the
// width between x0 and x1.
func noteLines(g *Gfx, font Font, x0, x1 int, lines []string) []string {
	out := make([]string, len(lines))
	for i, l := range lines {
		out[i] = g.Fit(font, l, x1-x0)
	}
	return out
}

// noteLine is the distance of the lines of a note.
func noteLine(font Font) int {
	if font == FontSmall {
		return 7
	}
	return 10
}

// drawNote draws the lines of a note (noteLines) centred between x0 and x1,
// the first on baseline y.
func drawNote(g *Gfx, dst *ebiten.Image, font Font, x0, x1, y int, lines []string) {
	for i, l := range noteLines(g, font, x0, x1, lines) {
		g.TextMid(dst, font, x0, x1, y+i*noteLine(font), l, ColWhite, ColBlack)
	}
}

// notePages splits the parts into pages that fit into the width, as many
// parts on a page as fit, two blanks between them; a part wider than the
// width alone is cut.
func notePages(g *Gfx, font Font, w int, parts ...string) []string {
	var pages []string
	page := ""
	for _, p := range parts {
		switch next := page + "  " + p; {
		case page == "":
			page = g.Fit(font, p, w)
		case g.TextWidth(font, next) <= w:
			page = next
		default:
			pages = append(pages, page)
			page = g.Fit(font, p, w)
		}
	}
	if page != "" {
		pages = append(pages, page)
	}
	return pages
}
