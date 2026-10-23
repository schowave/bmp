package ui

import (
	"image"

	"github.com/hajimehoshi/ebiten/v2"
)

// Icon is a cell of the icon sheet PIC/3.VGA (6 columns, 7 rows).
type Icon int

// Icons of PIC/3.VGA, row by row.
const (
	IconDesk Icon = iota
	IconClub
	IconTeam
	IconCompetitions
	IconDisk
	IconContinue // the manager tearing his hair: end the turn
	IconFinanceChart
	IconHistory
	IconTrophies
	IconStadium
	IconBank
	IconSponsors
	IconTable1
	IconTable2
	IconTable3
	IconResults1
	IconResults2
	IconResults3
	IconStrength1
	IconStrength2
	IconStrength3
	IconLineup
	IconContracts
	IconTransfers
	IconTraining
	IconCamp
	IconPlayerInfo
	IconDFBPokal
	IconLandesmeister
	IconPokalsieger
	IconUEFACup
	IconLoad // disk with an arrow: LADEN
	IconSave // arrow into the disk: SPEICHERN
	IconHighscore
	IconOptions
	IconMainMenu
	IconWeiter
	IconPitch
	IconNew
	IconRight
	IconLeft
	IconCobweb
)

var iconCols = [6][2]int{{1, 30}, {33, 64}, {67, 97}, {99, 131}, {134, 161}, {165, 196}}
var iconRows = [7][2]int{{0, 21}, {25, 45}, {49, 69}, {73, 93}, {97, 117}, {121, 142}, {144, 166}}

// IconImage returns the picture of an icon.
func (g *Gfx) IconImage(i Icon) *ebiten.Image {
	c, r := int(i)%6, int(i)/6
	return g.Sub(3, image.Rect(iconCols[c][0], iconRows[r][0], iconCols[c][1], iconRows[r][1]))
}

// Size of the marble frame of an icon button (PIC/25.VGA).
const (
	ButtonW = 50
	ButtonH = 41
)

// DrawIconButton draws an icon in its marble frame at x, y.
func (g *Gfx) DrawIconButton(dst *ebiten.Image, i Icon, x, y int) {
	g.drawIconButton(dst, i, x, y, false)
}

// drawIconButton draws an icon button; a pressed one (the open group of the
// main menu) has the frame PIC/25.VGA instead of PIC/24.VGA.
func (g *Gfx) drawIconButton(dst *ebiten.Image, i Icon, x, y int, pressed bool) {
	frame := 24
	if pressed {
		frame = 25
	}
	Draw(dst, g.Pic(frame, false), x, y)
	// The window of the frame is blue inside a black line.
	FillRect(dst, x+9, y+9, 32, 23, g.C(ColBlue))
	img := g.IconImage(i)
	b := img.Bounds()
	// Vertically the sheet is a grid of 24 pixels; a cell of 23 rows goes
	// 9 pixels below the top of the frame.
	c, r := int(i)%6, int(i)/6
	ix, iy := x+(ButtonW-b.Dx())/2, y+9
	img = g.Sub(3, image.Rect(iconCols[c][0], r*24, iconCols[c][1], min(r*24+23, 167)))
	if i == IconCobweb {
		// The cobweb is the only icon on black: the game shows it on blue.
		img = g.SubKeyed(3, 0, image.Rect(iconCols[c][0], iconRows[r][0], iconCols[c][1], iconRows[r][1]))
		iy = y + 9 + iconRows[r][0] - r*24
	}
	Draw(dst, img, ix, iy)
}
