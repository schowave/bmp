package ui

import (
	"image"

	"github.com/hajimehoshi/ebiten/v2"
)

// Positions of the main menu, in canvas pixels.
var (
	menuLeftX    = 7
	menuRightX   = 265
	menuRowsY    = [3]int{109, 150, 193}
	submenuCols  = [3]int{80, 135, 188}
	headerPanelY = 41
)

// MenuItem is one icon of a submenu. A nil Open shows the cobweb: the item
// is not available.
type MenuItem struct {
	Icon Icon
	Open func(app *App)
}

// MenuGroup is an icon of the main menu with the submenu it opens.
type MenuGroup struct {
	Icon  Icon
	Items [9]MenuItem // 3x3, column by column
	// Action runs instead of a submenu when there are no items.
	Action func(app *App)
}

// Header is what the panel at the top shows.
type Header struct {
	Face     int // 0..3, picture part of PIC/0.VGA
	Matchday string
	Message  [2]string
	Manager  string
	Logo     int
	Weekday  string
	Day      int
	Month    string
	Year     int
}

// MainMenu is the hub of a manager's turn.
type MainMenu struct {
	Header func() Header
	Left   [3]MenuGroup
	Right  [3]MenuGroup
	open   *MenuGroup
}

func (m *MainMenu) Update(app *App) error {
	in := &app.Input
	if !in.Left {
		if in.Right && m.open != nil {
			m.open = nil
		}
		return nil
	}
	for i := range 3 {
		for _, side := range []struct {
			x int
			g *MenuGroup
		}{{menuLeftX, &m.Left[i]}, {menuRightX, &m.Right[i]}} {
			if in.Clicked(side.x, menuRowsY[i], ButtonW, ButtonH) {
				if side.g.Action != nil {
					m.open = nil
					side.g.Action(app)
				} else {
					m.open = side.g
				}
				return nil
			}
		}
	}
	if m.open != nil {
		for k, it := range m.open.Items {
			x, y := submenuCols[k/3], menuRowsY[k%3]
			if it.Open != nil && in.Clicked(x, y, ButtonW, ButtonH) {
				it.Open(app)
				return nil
			}
		}
	}
	return nil
}

func (m *MainMenu) Draw(app *App, dst *ebiten.Image) {
	g := app.Gfx
	Draw(dst, g.Pic(22, true), 0, 0)
	Draw(dst, g.Pic(4, false), 0, 0)
	Draw(dst, g.Pic(23, false), 0, headerPanelY)
	if m.Header != nil {
		drawHeader(app, dst, m.Header())
	}
	for i := range 3 {
		g.DrawIconButton(dst, m.Left[i].Icon, menuLeftX, menuRowsY[i])
		g.DrawIconButton(dst, m.Right[i].Icon, menuRightX, menuRowsY[i])
	}
	if m.open != nil {
		for k, it := range m.open.Items {
			icon := it.Icon
			if it.Open == nil {
				icon = IconCobweb
			}
			g.DrawIconButton(dst, icon, submenuCols[k/3], menuRowsY[k%3])
		}
	}
}

func drawHeader(app *App, dst *ebiten.Image, h Header) {
	g := app.Gfx
	light := g.PalColor(0x1c)
	dark := g.PalColor(0x08)
	black := g.PalColor(0)
	if h.Face >= 0 && h.Face < 4 {
		Draw(dst, g.Sub(0, image.Rect(h.Face*41, 0, h.Face*41+40, 40)), 14, headerPanelY+15)
	}
	g.TextCentered(dst, FontNormal, 72, headerPanelY+14, 106, h.Matchday, black)
	g.TextCentered(dst, FontNormal, 72, headerPanelY+50, 106, h.Manager, black)
	for i, s := range h.Message {
		g.TextCentered(dst, FontNormal, 64, headerPanelY+26+i*9, 116, s, light)
	}
	if h.Logo > 0 {
		Draw(dst, g.Pic(h.Logo, false), 188, headerPanelY+22)
	}
	g.TextCentered(dst, FontNormal, 244, headerPanelY+17, 60, h.Weekday, dark)
	g.TextCentered(dst, FontBig, 244, headerPanelY+28, 60, itoa(h.Day)+".", dark)
	g.TextCentered(dst, FontNormal, 244, headerPanelY+45, 60, h.Month, dark)
	g.TextCentered(dst, FontNormal, 244, headerPanelY+54, 60, itoa(h.Year), dark)
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	neg := n < 0
	if neg {
		n = -n
	}
	var b []byte
	for n > 0 {
		b = append([]byte{byte('0' + n%10)}, b...)
		n /= 10
	}
	if neg {
		b = append([]byte{'-'}, b...)
	}
	return string(b)
}
