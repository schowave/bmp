package ui

import "github.com/hajimehoshi/ebiten/v2"

// The remake's hidden keys o leave what they did as a note on the screen,
// in a free place, instead of a message that goes again: in the footer of the
// lineup, training and stadium screens while the screen is open, and in the
// middle of the main menu until the turn ends.

// noteLine is the distance of the lines of a note.
func noteLine(font Font) int {
	if font == FontSmall {
		return 7
	}
	return 10
}

// drawNote draws the lines of a note centred between x0 and x1, the first
// on baseline y, each cut to the width.
func drawNote(g *Gfx, dst *ebiten.Image, font Font, x0, x1, y int, lines []string) {
	for i, l := range lines {
		g.TextMid(dst, font, x0, x1, y+i*noteLine(font), g.Fit(font, l, x1-x0), ColWhite, ColBlack)
	}
}

// opponentNote names the opponent a key worked for, or nothing: "GEGEN X"
// for a home match, "BEI X" for an away match.
func opponentNote(name string, opponent int, home bool) []string {
	if opponent < 0 {
		return nil
	}
	if !home {
		return []string{"BEI " + name}
	}
	return []string{"GEGEN " + name}
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
