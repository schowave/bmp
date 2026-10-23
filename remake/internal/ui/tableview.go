package ui

import (
	"github.com/hajimehoshi/ebiten/v2"
)

// Align of a table column.
type Align int

// Alignments.
const (
	Left Align = iota
	Right
	Center
)

// Column of a TableView. X is relative to the table.
type Column struct {
	Title string
	X, W  int
	Align Align
}

// Row of a TableView.
type Row struct {
	Cells []string
	Color int // palette index, 0 = default
	Bar   int // palette index of a background bar, 0 = none
}

// TableView draws rows in columns on a blue panel and lets the user
// scroll and pick rows.
type TableView struct {
	X, Y, W, H int
	Font       Font
	RowH       int
	Columns    []Column
	Rows       []Row
	Top        int // first visible row
	Selected   int // -1 = none
	Hover      int
	NoHeader   bool
	TitleColor int
}

// NewTableView creates a table view.
func NewTableView(x, y, w, h int, cols []Column) *TableView {
	return &TableView{X: x, Y: y, W: w, H: h, Font: FontSmall, RowH: 7, Columns: cols, Selected: -1, Hover: -1}
}

func (t *TableView) headerH() int {
	if t.NoHeader {
		return 2
	}
	return 10
}

// Visible returns the number of rows that fit.
func (t *TableView) Visible() int { return (t.H - t.headerH() - 2) / t.RowH }

// Update handles scrolling and returns the row clicked with the left (or
// right) button, or -1.
func (t *TableView) Update(in *Input) (left, right int) {
	left, right = -1, -1
	t.Hover = -1
	if !in.In(t.X, t.Y, t.W, t.H) {
		return
	}
	if in.WheelDelta < 0 && t.Top+t.Visible() < len(t.Rows) {
		t.Top++
	}
	if in.WheelDelta > 0 && t.Top > 0 {
		t.Top--
	}
	y := in.Y - t.Y - t.headerH()
	if y < 0 {
		return
	}
	r := t.Top + y/t.RowH
	if r >= len(t.Rows) || y/t.RowH >= t.Visible() {
		return
	}
	t.Hover = r
	if in.Left {
		left = r
	}
	if in.Right {
		right = r
	}
	return
}

// Scroll moves the view by n rows.
func (t *TableView) Scroll(n int) {
	t.Top += n
	if t.Top > len(t.Rows)-t.Visible() {
		t.Top = len(t.Rows) - t.Visible()
	}
	if t.Top < 0 {
		t.Top = 0
	}
}

// Draw draws the table.
func (t *TableView) Draw(g *Gfx, dst *ebiten.Image) {
	g.Panel(dst, t.X, t.Y, t.W, t.H)
	tc := ColBeige
	if t.TitleColor != 0 {
		tc = t.TitleColor
	}
	if !t.NoHeader {
		for _, c := range t.Columns {
			g.drawCell(dst, FontSmall, t.X+c.X, t.Y+2, c, c.Title, g.PalColor(tc))
		}
		FillRect(dst, t.X+2, t.Y+9, t.W-4, 1, g.PalColor(ColGrey5))
	}
	y := t.Y + t.headerH()
	for i := t.Top; i < len(t.Rows) && i < t.Top+t.Visible(); i++ {
		row := t.Rows[i]
		if row.Bar != 0 {
			FillRect(dst, t.X+2, y, t.W-4, t.RowH, g.PalColor(row.Bar))
		}
		if i == t.Selected {
			FillRect(dst, t.X+2, y, t.W-4, t.RowH, g.PalColor(ColGrey6))
		} else if i == t.Hover {
			FillRect(dst, t.X+2, y, t.W-4, t.RowH, g.PalColor(ColGrey7))
		}
		col := ColLight
		if row.Color != 0 {
			col = row.Color
		}
		for k, c := range t.Columns {
			if k < len(row.Cells) {
				g.drawCell(dst, t.Font, t.X+c.X, y+(t.RowH-g.FontHeight(t.Font))/2, c, row.Cells[k], g.PalColor(col))
			}
		}
		y += t.RowH
	}
	if len(t.Rows) > t.Visible() {
		// scroll bar
		bh := t.H - t.headerH() - 2
		th := max(4, bh*t.Visible()/len(t.Rows))
		ty := t.Y + t.headerH() + (bh-th)*t.Top/max(1, len(t.Rows)-t.Visible())
		FillRect(dst, t.X+t.W-3, t.Y+t.headerH(), 2, bh, g.PalColor(ColGrey7))
		FillRect(dst, t.X+t.W-3, ty, 2, th, g.PalColor(ColGrey3))
	}
}

func (g *Gfx) drawCell(dst *ebiten.Image, font Font, x, y int, c Column, s string, col interface {
	RGBA() (r, g, b, a uint32)
}) {
	s = g.Fit(font, s, c.W)
	switch c.Align {
	case Right:
		g.Text(dst, font, x+c.W-g.TextWidth(font, s), y, s, col)
	case Center:
		g.Text(dst, font, x+(c.W-g.TextWidth(font, s))/2, y, s, col)
	default:
		g.Text(dst, font, x, y, s, col)
	}
}
