package ui

import (
	"github.com/hajimehoshi/ebiten/v2"

	"github.com/schowave/bmp/remake/internal/assets"
	"github.com/schowave/bmp/remake/internal/game"
	"github.com/schowave/bmp/remake/internal/seasons"
)

// SeasonChoice is the screen before the setup of a new game where the
// player picks the clubs: on the left a button for those of the original,
// on the right the list of the real seasons (internal/seasons). The original
// has no such screen; it is drawn with the windows and buttons of the file
// screen. Without built-in seasons it goes straight to the setup.
type SeasonChoice struct {
	mana     *assets.Mana // the clubs of the original
	saveDir  string
	entries  []seasons.Season
	original Button
}

// Geometry of the screen: the original in a window on the left, the
// seasons in two columns of 19 rows on the right.
const (
	seasonRows  = 19
	seasonColW  = 96
	seasonLeftX = 124
	seasonTopY  = 37
)

// NewSeasonChoice creates the screen; mana holds the clubs of the original.
func NewSeasonChoice(mana *assets.Mana, saveDir string) Scene {
	list := seasons.List()
	if len(list) == 0 {
		return NewSetup(mana, saveDir)
	}
	return newSeasonChoice(mana, saveDir, list)
}

func newSeasonChoice(mana *assets.Mana, saveDir string, list []seasons.Season) *SeasonChoice {
	return &SeasonChoice{mana: mana, saveDir: saveDir, entries: list,
		original: Button{X: 18, Y: 92, W: 80, H: 12, Label: "ORIGINAL"}}
}

// seasonLabel is the text of a season in the list.
func seasonLabel(s seasons.Season) string { return s.Label() }

// seasonCell is the top-left corner of the row of entry i.
func seasonCell(i int) (x, y int) {
	return seasonLeftX + i/seasonRows*seasonColW, seasonTopY + i%seasonRows*fileRowH
}

// hover returns the entry under the pointer, or -1.
func (c *SeasonChoice) hover(in *Input) int {
	if in.X < seasonLeftX || in.X >= seasonLeftX+2*seasonColW || in.Y < seasonTopY || in.Y >= seasonTopY+seasonRows*fileRowH {
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
		r := seasonRow{x: x, y: y, text: seasonLabel(s), colour: ColLight, bar: i == h}
		if r.bar {
			r.colour ^= 31
		}
		out[i] = r
	}
	return out
}

func (c *SeasonChoice) Update(app *App) error {
	in := &app.Input
	if c.original.Hit(in) {
		app.Replace(c.withBack(NewSetup(c.mana, c.saveDir)))
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
	app.Replace(c.withBack(NewSeasonSetup(c.mana, m, c.saveDir, s.ID, s.Year)))
	return nil
}

// withBack lets a right click in setup s come back here while nothing is
// chosen, to look at the clubs of another season.
func (c *SeasonChoice) withBack(s *Setup) *Setup {
	s.back = func() Scene { return newSeasonChoice(c.mana, c.saveDir, c.entries) }
	return s
}

// LoadGame restores a saved game with the clubs it was started with: those
// of the original (orig) or of its real season.
func LoadGame(data []byte, orig *assets.Mana) (*game.State, error) {
	id, err := game.SaveSeason(data)
	if err != nil {
		return nil, err
	}
	m, err := seasons.ManaOr(orig, id)
	if err != nil {
		return nil, err
	}
	return game.Load(data, m)
}

func (c *SeasonChoice) Draw(app *App, dst *ebiten.Image) {
	g := app.Gfx
	g.Background(dst)
	g.TextMid(dst, FontBig, 0, 320, 20, "WELCHE SAISON", ColBeige, ColBlack)
	// Left: the clubs of the original.
	g.Window(dst, 6, 33, 110, 172, ColBlue)
	g.TextMid(dst, FontSmall, 6, 110, 70, "VEREINE DES", ColLight, NoShadow)
	g.TextMid(dst, FontSmall, 6, 110, 78, "ORIGINALS", ColLight, NoShadow)
	drawDialogButton(app, dst, &c.original)
	g.TextMid(dst, FontSmall, 6, 110, 116, "STAND 1991/92", ColOrange, NoShadow)
	// Right: the real seasons.
	g.Window(dst, 116, 33, 314, 172, ColBlue)
	for _, r := range c.rows(c.hover(&app.Input)) {
		if r.bar {
			FillRect(dst, r.x-2, r.y, seasonColW-4, 5, g.PalColor(ColBlue^31))
		}
		g.TextAt(dst, FontSmall, r.x, r.y+4, r.text, r.colour, NoShadow)
	}
	g.TextMid(dst, FontNormal, 0, 320, 196, "Mit welchen Vereinen wird gespielt?", ColBeige, ColBlack)
}
