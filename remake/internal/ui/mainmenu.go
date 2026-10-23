package ui

import (
	"image"
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"
)

// Positions of the main menu, in canvas pixels.
var (
	menuLeftX    = 5
	menuRightX   = 266
	menuRowsY    = [3]int{110, 153, 196}
	submenuCols  = [3]int{82, 136, 190}
	submenuRowsY = [3]int{112, 154, 196}
	headerPanelY = 41
	// The banner PIC/4.VGA is drawn with its top rows above the screen.
	bannerY = -3
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
	// Places are the table places (0-based) after the matchdays played,
	// drawn as a curve in the box under the matchday; Chart turns it on.
	Places []int
	Chart  bool
}

// MainMenu is the hub of a manager's turn.
type MainMenu struct {
	Header func() Header
	Left   [3]MenuGroup
	Right  [3]MenuGroup
	// Calendar runs when the calendar sheet is clicked: like in the
	// original this ends the turn and goes on to the next date.
	Calendar func(app *App)
	open     *MenuGroup
	// pressed is the group icon last clicked: it keeps the pressed frame
	// until the screen is built again.
	pressed *MenuGroup
	// Messages is the message panel; NewMessages opens it at the start.
	Messages    *MessagePanel
	NewMessages func() bool
	msgOpen     bool
	started     bool
}

// calendarRect is the calendar sheet of the header panel (PIC/23.VGA).
func calendarRect() (x, y, w, h int) { return 240, headerPanelY, 68, 66 }

func (m *MainMenu) Update(app *App) error {
	in := &app.Input
	if !m.started {
		m.started = true
		if m.Messages != nil && m.NewMessages != nil && m.NewMessages() {
			m.msgOpen = true
			return nil
		}
	}
	if m.msgOpen {
		// The panel takes all clicks until X closes it.
		m.msgOpen = m.Messages.Update(in)
		return nil
	}
	if !in.Left {
		if in.Right && m.open != nil {
			m.open = nil
		}
		return nil
	}
	if x, y, w, h := calendarRect(); m.Calendar != nil && in.Clicked(x, y, w, h) {
		m.open, m.pressed = nil, nil
		m.Calendar(app)
		return nil
	}
	for i := range 3 {
		for _, side := range []struct {
			x int
			g *MenuGroup
		}{{menuLeftX, &m.Left[i]}, {menuRightX, &m.Right[i]}} {
			if in.Clicked(side.x, menuRowsY[i], ButtonW, ButtonH) {
				m.pressed = side.g
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
			x, y := submenuCols[k/3], submenuRowsY[k%3]
			if in.Clicked(x, y, ButtonW, ButtonH) {
				if it.Open != nil {
					m.pressed = nil
					it.Open(app)
				}
				return nil
			}
		}
	}
	if m.Messages != nil && msgPanelArea(in) {
		m.open, m.pressed = nil, nil
		m.msgOpen = true
		m.Messages.top = 0
	}
	return nil
}

func (m *MainMenu) Draw(app *App, dst *ebiten.Image) {
	g := app.Gfx
	Draw(dst, g.Pic(22, true), 0, 0)
	Draw(dst, g.Pic(4, false), 0, bannerY)
	Draw(dst, g.Pic(23, false), 0, headerPanelY)
	if m.Header != nil {
		drawHeader(app, dst, m.Header())
	}
	for i := range 3 {
		g.drawIconButton(dst, m.Left[i].Icon, menuLeftX, menuRowsY[i], m.pressed == &m.Left[i])
		g.drawIconButton(dst, m.Right[i].Icon, menuRightX, menuRowsY[i], m.pressed == &m.Right[i])
	}
	if m.msgOpen {
		m.Messages.Draw(g, dst)
	} else if m.open != nil {
		for k, it := range m.open.Items {
			icon := it.Icon
			if it.Open == nil {
				icon = IconCobweb
			}
			g.DrawIconButton(dst, icon, submenuCols[k/3], submenuRowsY[k%3])
		}
	}
}

func drawHeader(app *App, dst *ebiten.Image, h Header) {
	g := app.Gfx
	light := g.PalColor(0x1c)
	dark := g.PalColor(0x08)
	black := g.PalColor(0)
	if h.Face >= 0 && h.Face < 4 {
		Draw(dst, g.Sub(0, image.Rect(h.Face*41, 0, h.Face*41+40, 40)), 15, headerPanelY+18)
	}
	g.TextCentered(dst, FontNormal, 66, headerPanelY+11, 106, h.Matchday, black)
	g.TextCentered(dst, FontNormal, 67, headerPanelY+51, 106, h.Manager, black)
	for i, s := range h.Message {
		g.TextCentered(dst, FontSmall, 64, headerPanelY+27+i*8, 116, g.Fit(FontSmall, s, 114), light)
	}
	if h.Chart {
		drawPlaces(g, dst, h.Places)
	}
	if h.Logo > 0 {
		Draw(dst, g.Pic(h.Logo, false), 188, headerPanelY+22)
	}
	// The sheet is printed grey-blue with a shadow to the right, the day
	// darker.
	ink, inkShadow := g.C(ColGrey4), g.C(ColGrey7)
	g.TextCenteredShadow(dst, FontNormal, 244, headerPanelY+13, 60, h.Weekday, ink, inkShadow)
	g.TextCenteredShadow(dst, FontBig, 243, headerPanelY+24, 60, itoa(h.Day)+".", g.C(ColGrey6), dark)
	g.TextCenteredShadow(dst, FontNormal, 244, headerPanelY+43, 60, h.Month, ink, inkShadow)
	g.TextCenteredShadow(dst, FontNormal, 244, headerPanelY+53, 60, itoa(h.Year), ink, inkShadow)
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

// drawPlaces draws the curve of the table places in the box of the header
// (f_08bc_0b5f): from the left edge at y=74 to one point per matchday, the
// last at the right edge; without a matchday a dark line.
func drawPlaces(g *Gfx, dst *ebiten.Image, places []int) {
	const x0, x1, y0 = 0x3f, 0xb1, 0x4a
	if len(places) == 0 {
		drawLine(dst, x0, y0, x1, y0, g.C(ColGrey7))
		return
	}
	px, py := x0, y0
	for k, p := range places {
		x := x0 + (k+1)*(x1-x0)/len(places)
		y := 0x41 + p
		drawLine(dst, px, py, x, y, g.C(ColBeige))
		px, py = x, y
	}
}

// drawLine draws a line of single pixels.
func drawLine(dst *ebiten.Image, x0, y0, x1, y1 int, c color.Color) {
	dx, dy := abs(x1-x0), -abs(y1-y0)
	sx, sy := 1, 1
	if x0 > x1 {
		sx = -1
	}
	if y0 > y1 {
		sy = -1
	}
	err := dx + dy
	for {
		FillRect(dst, x0, y0, 1, 1, c)
		if x0 == x1 && y0 == y1 {
			return
		}
		e2 := 2 * err
		if e2 >= dy {
			err += dy
			x0 += sx
		}
		if e2 <= dx {
			err += dx
			y0 += sy
		}
	}
}

func abs(v int) int {
	if v < 0 {
		return -v
	}
	return v
}
