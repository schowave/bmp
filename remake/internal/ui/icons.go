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
	IconSave
	IconLoad
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
	Draw(dst, g.Pic(25, false), x, y)
	img := g.IconImage(i)
	b := img.Bounds()
	Draw(dst, img, x+(ButtonW-b.Dx())/2, y+(ButtonH-b.Dy())/2)
}
