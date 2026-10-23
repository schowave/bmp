package ui

import (
	"github.com/hajimehoshi/ebiten/v2"
)

// MessagePanel is the message list of the main menu (f_3091_05c4): three
// grey fields in the middle of the menu with the buttons up, X and down of
// PIC/2.VGA. It shows the newest messages first, the first line of a
// message (its date) a little apart.
type MessagePanel struct {
	Messages func() [][]string // the lines of each message
	top      int
	pressed  int // button drawn pressed, -1 = none
}

// Geometry of the panel.
const (
	msgPanelX0, msgPanelX1 = 0x3a, 0xf5
	msgButtonX             = 0xf5
	msgPanelY              = 0x71
	msgFieldPitch          = 0x29
	msgButtonW, msgButtonH = 0x12, 0x28
)

// msgPanelArea reports whether a click opens the panel: the free middle
// of the main menu.
func msgPanelArea(in *Input) bool { return in.Clicked(0x39, 0x70, 0x106-0x39+1, 0xea-0x70+1) }

// Update handles a click; it returns false when the panel closes.
func (p *MessagePanel) Update(in *Input) bool {
	p.pressed = -1
	if !in.Left {
		return true
	}
	for b := 0; b < 3; b++ {
		if in.Clicked(msgButtonX, msgPanelY+b*msgFieldPitch, msgButtonW, msgButtonH) {
			p.pressed = b
			n := len(p.Messages())
			switch b {
			case 0:
				if p.top > 0 {
					p.top--
				}
			case 1:
				return false
			case 2:
				if p.top+1 < n {
					p.top++
				}
			}
		}
	}
	return true
}

// Draw draws the panel with the messages of the current page.
func (p *MessagePanel) Draw(g *Gfx, dst *ebiten.Image) {
	FillRect(dst, 0x3b, 0x72, 0xf5-0x3b+1, 0xea-0x72+1, g.PalColor(ColGrey4))
	FrameRect(dst, 0x39, 0x70, 0x106-0x39+1, 0xea-0x70+1, g.PalColor(ColGrey7))
	for i := 0; i < 3; i++ {
		y0 := msgPanelY + i*msgFieldPitch
		FillRect(dst, 0x3a, y0, 0xf2-0x3a+1, 1, g.PalColor(ColLight))
		FillRect(dst, 0x3a, y0, 1, 0x27, g.PalColor(ColLight))
		FillRect(dst, 0x3b, y0+0x27, 0xf3-0x3b+1, 1, g.PalColor(ColGrey7))
		FillRect(dst, 0xf3, y0+1, 1, 0x27, g.PalColor(ColGrey7))
	}
	Draw(dst, g.Sub(2, rectWH(0, 0, msgButtonW, 0x7a)), msgButtonX, msgPanelY)
	if p.pressed >= 0 {
		y := p.pressed * msgButtonH
		Draw(dst, g.Sub(2, rectWH(0x13, y, msgButtonW, msgButtonH)), msgButtonX, msgPanelY+y)
	}
	msgs := p.Messages()
	if len(msgs) == 0 {
		g.TextMid(dst, FontNormal, msgPanelX0, msgPanelX1, 0xb1, "Keine Nachricht vorhanden.", ColLight, ColBlack)
		return
	}
	y := 0x7a
	for i := p.top; i < len(msgs) && i < p.top+3; i++ {
		// Each line is centred as it is, with its blanks; a line wider than
		// the field reaches over its edges as in the original.
		for k, l := range msgs[i] {
			c := 24
			if k == 0 {
				c = 25
			}
			if y <= 0xe8 {
				g.TextMid(dst, FontNormal, msgPanelX0, msgPanelX1, y, l, c, ColBlack)
			}
			if k == 0 {
				y += 10
			} else {
				y += 8
			}
		}
		y += 7
	}
}
