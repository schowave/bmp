package ui

import (
	"github.com/hajimehoshi/ebiten/v2"

	"github.com/schowave/bmp/remake/internal/seasons"
)

// SeasonChoice is the screen the key S opens in the setup of a new game,
// where the player picks the clubs: on the left those of the original
// with the WEITER button, on the right the list of the real seasons
// (internal/seasons), picked under the bar of the file screen. Either goes
// back to the setup with those clubs and the names typed so far; a right
// click goes back unchanged. The original has no such screen; it is drawn
// with the windows and buttons of the others.
type SeasonChoice struct {
	from    *Setup // the setup it was opened from
	entries []seasons.Season
}

// Geometry of the screen: the original in a window on the left, the
// seasons under a heading in three columns of 17 rows (51 seasons) on the
// right, with the rule that changes below them.
const (
	seasonRows  = 17
	seasonCols  = 3
	seasonColW  = 64
	seasonLeftX = 124
	seasonTopY  = 46
)

// originalButton is the WEITER button of the original, in the middle of
// the left window.
var originalButton = Button{X: 6 + (104-ButtonW)/2, Y: 78, W: ButtonW, H: ButtonH}

func newSeasonChoice(from *Setup) *SeasonChoice {
	return &SeasonChoice{from: from, entries: from.seasons}
}

// seasonCell is the top-left corner of the row of entry i.
func seasonCell(i int) (x, y int) {
	return seasonLeftX + i/seasonRows*seasonColW, seasonTopY + i%seasonRows*fileRowH
}

// hover returns the entry under the pointer, or -1.
func (c *SeasonChoice) hover(in *Input) int {
	if in.X < seasonLeftX || in.X >= seasonLeftX+seasonCols*seasonColW || in.Y < seasonTopY || in.Y >= seasonTopY+seasonRows*fileRowH {
		return -1
	}
	i := (in.X-seasonLeftX)/seasonColW*seasonRows + (in.Y-seasonTopY)/fileRowH
	if i >= len(c.entries) {
		return -1
	}
	return i
}

// seasonRow is an entry as the screen shows it.
type seasonRow struct {
	x, y   int
	text   string
	colour int
	bar    bool // under the pointer: drawn with the XOR bar of the file screen
}

// rows lays out the entries, with entry h under the pointer.
func (c *SeasonChoice) rows(h int) []seasonRow {
	out := make([]seasonRow, len(c.entries))
	for i, s := range c.entries {
		x, y := seasonCell(i)
		r := seasonRow{x: x, y: y, text: s.Label(), colour: ColLight, bar: i == h}
		if r.bar {
			r.colour ^= 31
		}
		out[i] = r
	}
	return out
}

func (c *SeasonChoice) Update(app *App) error {
	in := &app.Input
	orig, dir := c.from.orig, c.from.saveDir
	if in.Right || in.Escape {
		app.Replace(c.from)
		return nil
	}
	if originalButton.Hit(in) {
		app.Replace(c.setup(NewSetup(orig, dir)))
		return nil
	}
	i := c.hover(in)
	if !in.Left || i < 0 {
		return nil
	}
	s := c.entries[i]
	m, err := seasons.Mana(s.ID)
	if err != nil {
		app.Push(NewMessage("An Alle:", "Die Saison "+s.Label()+" ist nicht lesbar."))
		return nil
	}
	app.Replace(c.setup(NewSeasonSetup(orig, m, dir, s)))
	return nil
}

// setup is s, the setup with the chosen clubs, with the seasons and the
// names typed so far; the clubs are chosen anew.
func (c *SeasonChoice) setup(s *Setup) *Setup {
	s.seasons, s.names = c.entries, c.from.names
	return s
}

func (c *SeasonChoice) Draw(app *App, dst *ebiten.Image) {
	g := app.Gfx
	g.Background(dst)
	g.TextMid(dst, FontBig, 0, 320, 20, "WELCHE SAISON", ColBeige, ColBlack)
	// Left: the clubs and rules of the original and the button to play them.
	g.Window(dst, 6, 33, 110, 172, ColBlue)
	g.TextMid(dst, FontNormal, 6, 110, 62, "ORIGINAL", ColLight, NoShadow)
	g.DrawIconButton(dst, IconWeiter, originalButton.X, originalButton.Y)
	for i, l := range []string{"VEREINE UND REGELN", "DES ORIGINALS,", "STAND 1991/92"} {
		g.TextMid(dst, FontSmall, 6, 110, originalButton.Y+ButtonH+12+8*i, l, ColGrey2, NoShadow)
	}
	// Right: the real seasons.
	g.Window(dst, 116, 33, 314, 172, ColBlue)
	g.TextAt(dst, FontSmall, seasonLeftX, 42, "ECHTE SAISONS", ColOrange, NoShadow)
	for _, r := range c.rows(c.hover(&app.Input)) {
		if r.bar {
			FillRect(dst, r.x-2, r.y, seasonColW-4, 5, g.PalColor(ColBlue^31))
		}
		g.TextAt(dst, FontSmall, r.x, r.y+4, r.text, r.colour, NoShadow)
	}
	g.TextAt(dst, FontSmall, seasonLeftX, 170, "AB 1995/96 3 PUNKTE PRO SIEG", ColGrey2, NoShadow)
	g.TextMid(dst, FontNormal, 0, 320, 196, "Saison anklicken, WEITER: Original, rechts: zurück", ColBeige, ColBlack)
}
