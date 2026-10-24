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

// lineupNote is the note of the lineup key: the rating, the automatic
// lineups of the three formations and the opponent.
func lineupNote(s *game.State, res game.OptimalLineup) []string {
	note := []string{"OPTIMAL " + pointsText(res.After),
		"STAND. " + pointsText(res.Auto[0]) + "/" + pointsText(res.Auto[1]) + "/" + pointsText(res.Auto[2])}
	return append(note, opponentNote(s, res.Opponent, res.Home)...)
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

// fitParts joins as many of the parts as fit into the width, two blanks
// between them: a part that does not fit is left out, not cut.
func fitParts(g *Gfx, font Font, w int, parts ...string) string {
	out := ""
	for _, p := range parts {
		next := p
		if out != "" {
			next = out + "  " + p
		}
		if g.TextWidth(font, next) > w {
			break
		}
		out = next
	}
	return out
}
