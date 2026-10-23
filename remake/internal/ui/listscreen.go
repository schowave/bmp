package ui

import (
	"github.com/hajimehoshi/ebiten/v2"
)

// ListScreen is a screen with a title, one table and the HAUPTMENÜ button.
// Arrows (if Prev/Next are set) page through the content.
type ListScreen struct {
	Title     string
	Sub       string
	Table     *TableView
	Prev      func()
	Next      func()
	OnClick   func(app *App, row int)
	OnRight   func(app *App, row int)
	ExtraDraw func(app *App, dst *ebiten.Image)
}

var (
	arrowLeft  = Button{X: 214, Y: 196, W: ButtonW, H: ButtonH}
	arrowRight = Button{X: 160, Y: 196, W: ButtonW, H: ButtonH}
)

func (l *ListScreen) Update(app *App) error {
	in := &app.Input
	left, right := l.Table.Update(in)
	if left >= 0 && l.OnClick != nil {
		l.OnClick(app, left)
		return nil
	}
	if right >= 0 && l.OnRight != nil {
		l.OnRight(app, right)
		return nil
	}
	if l.Prev != nil && arrowRight.Hit(in) {
		l.Prev()
		return nil
	}
	if l.Next != nil && arrowLeft.Hit(in) {
		l.Next()
		return nil
	}
	if MenuButtonHit(in) {
		app.Pop()
	}
	return nil
}

func (l *ListScreen) Draw(app *App, dst *ebiten.Image) {
	g := app.Gfx
	g.Background(dst)
	g.Panel(dst, 4, 3, 312, 17)
	if l.Sub != "" {
		g.Text(dst, FontNormal, 10, 7, l.Title, g.C(ColBeige))
		w := 300 - g.TextWidth(FontNormal, l.Title) - 10
		g.TextRight(dst, FontSmall, 310, 8, g.Fit(FontSmall, l.Sub, w), g.C(ColLight))
	} else {
		g.TextCentered(dst, FontNormal, 4, 7, 312, l.Title, g.C(ColBeige))
	}
	l.Table.Draw(g, dst)
	if l.Prev != nil {
		g.DrawIconButton(dst, IconLeft, arrowRight.X, arrowRight.Y)
	}
	if l.Next != nil {
		g.DrawIconButton(dst, IconRight, arrowLeft.X, arrowLeft.Y)
	}
	if l.ExtraDraw != nil {
		l.ExtraDraw(app, dst)
	}
	g.DrawMenuButton(dst)
}
