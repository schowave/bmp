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
		return "ORIGINAL (1992/93)"
	}
	return "SAISON " + s.Label()
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
		app.Push(NewMessage("An Alle:", err.Error()))
		return nil
	}
	app.Replace(NewSeasonSetup(c.mana, m, c.saveDir, s.ID, s.Year))
	return nil
}

// LoadGame restores a saved game with the clubs it was started with: those
// of the original (orig) or of a real season.
func LoadGame(data []byte, orig *assets.Mana) (*game.State, error) {
	id, err := game.SaveSeason(data)
	if err != nil {
		return nil, err
	}
	m := orig
	if id != "" {
		if m, err = seasons.Mana(id); err != nil {
			return nil, err
		}
	}
	return game.Load(data, m)
}

func (c *SeasonChoice) Draw(app *App, dst *ebiten.Image) {
	g := app.Gfx
	g.Background(dst)
	g.TextMid(dst, FontBig, 0, 320, 20, "WELCHE SAISON ?", ColBeige, ColBlack)
	g.Window(dst, 6, 33, 314, 172, ColBlue)
	h := c.hover(&app.Input)
	for i, s := range c.entries {
		col, row := i/seasonRows, i%seasonRows
		x, y := seasonLeftX+col*seasonColW, seasonTopY+row*fileRowH
		fg := ColLight
		if s.ID == "" {
			fg = ColOrange
		}
		if i == h {
			// The bar under the pointer like the file screen (XOR).
			FillRect(dst, x-2, y, seasonColW-4, 5, g.PalColor(ColBlue^31))
			fg ^= 31
		}
		g.TextAt(dst, FontSmall, x, y+4, seasonLabel(s), fg, NoShadow)
	}
	g.TextMid(dst, FontSmall, 0, 320, 190, "LINKE MAUSTASTE: VEREINE DIESER SAISON", ColGrey2, ColBlack)
}
