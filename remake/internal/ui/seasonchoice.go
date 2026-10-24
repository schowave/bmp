package ui

import (
	"github.com/hajimehoshi/ebiten/v2"

	"github.com/schowave/bmp/remake/internal/assets"
	"github.com/schowave/bmp/remake/internal/game"
	"github.com/schowave/bmp/remake/internal/seasons"
)

// SeasonChoice is the screen before the setup of a new game where the
// player picks the clubs: those of the original, or those of a real season
// (internal/seasons). The original has no such screen; it is drawn like the
// file screen. Without built-in seasons it goes straight to the setup.
type SeasonChoice struct {
	mana    *assets.Mana // the clubs of the original
	saveDir string
	entries []seasons.Season // entries[0] is the original (empty ID)
}

// Geometry of the screen: three columns of 18 rows.
const (
	seasonRows  = 18
	seasonColW  = 100
	seasonLeftX = 12
	seasonTopY  = 37
)

// NewSeasonChoice creates the screen; mana holds the clubs of the original.
func NewSeasonChoice(mana *assets.Mana, saveDir string) Scene {
	list := seasons.List()
	if len(list) == 0 {
		return NewSetup(mana, saveDir)
	}
	return &SeasonChoice{mana: mana, saveDir: saveDir, entries: append([]seasons.Season{{}}, list...)}
}

// seasonLabel is the text of an entry.
func seasonLabel(s seasons.Season) string {
	if s.ID == "" {
		// The clubs of the original are those of 1991/92 (its modern start
		// plays them from 1992 on).
		return "ORIGINAL (1991/92)"
	}
	return "SAISON " + s.Label()
}

// seasonCell is the top-left corner of the row of entry i.
func seasonCell(i int) (x, y int) {
	return seasonLeftX + i/seasonRows*seasonColW, seasonTopY + i%seasonRows*fileRowH
}

// hover returns the entry under the pointer, or -1.
func (c *SeasonChoice) hover(in *Input) int {
	if in.X < seasonLeftX || in.X >= seasonLeftX+3*seasonColW || in.Y < seasonTopY || in.Y >= seasonTopY+seasonRows*fileRowH {
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
		if s.ID == "" {
			r.colour = ColOrange
		}
		if r.bar {
			r.colour ^= 31
		}
		out[i] = r
	}
	return out
}

func (c *SeasonChoice) Update(app *App) error {
	in := &app.Input
	i := c.hover(in)
	if !in.Left || i < 0 {
		return nil
	}
	s := c.entries[i]
	if s.ID == "" {
		app.Replace(NewSetup(c.mana, c.saveDir))
		return nil
	}
	m, err := seasons.Mana(s.ID)
	if err != nil {
		app.Push(NewMessage("An Alle:", "Die Saison "+s.Label()+" ist nicht lesbar."))
		return nil
	}
	app.Replace(NewSeasonSetup(c.mana, m, c.saveDir, s.ID, s.Year))
	return nil
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
	g.Window(dst, 6, 33, 314, 172, ColBlue)
	for _, r := range c.rows(c.hover(&app.Input)) {
		if r.bar {
			FillRect(dst, r.x-2, r.y, seasonColW-4, 5, g.PalColor(ColBlue^31))
		}
		g.TextAt(dst, FontSmall, r.x, r.y+4, r.text, r.colour, NoShadow)
	}
	g.TextMid(dst, FontNormal, 0, 320, 196, "Mit welchen Vereinen wird gespielt?", ColBeige, ColBlack)
}
