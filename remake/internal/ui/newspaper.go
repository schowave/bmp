package ui

import (
	"image"
	"image/color"
	"strings"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/schowave/bmp/remake/internal/game"
)

// The newspaper "Sportnachrichten" after a league match day (f_2e3a_0ea3).
// The original switches colours 0..15 to a grey ramp for it: colour i shows
// grey level bitrev4(i) of 16 (table D:0x9394), so the pictures (PIC/44 and
// the photos PIC/200..229, which hold values 0..15) and the text colours
// 0, 6, 9, 14 and 15 come out grey.

// newsPalette returns the colour of index i (0..15) on the newspaper page.
func newsPalette(i int) color.RGBA {
	l := 0
	for b := 0; b < 4; b++ {
		if i&(1<<b) != 0 {
			l |= 8 >> b
		}
	}
	v := uint8(4 * l)
	v = v<<2 | v>>4
	return color.RGBA{v, v, v, 0xff}
}

// Colours of the newspaper page (indices of the newspaper palette).
const (
	newsInk    = 0
	newsPrefix = 6
	newsBorder = 9
	newsText   = 14
	newsPaper  = 15
)

// NewspaperScene shows the newspaper of one manager; a click closes it.
type NewspaperScene struct {
	ss   *Session
	md   *game.MatchDay
	mi   int
	done func(*App)
	np   *game.Newspaper
	pics map[int]*ebiten.Image
}

// NewNewspaper returns the newspaper of manager mi after the match day md,
// or nil when he gets none (option "Zeitung" off, cup match day, his team
// did not play). The content is made, with the random numbers of the
// original, when the scene is first updated or drawn. done is called after
// the scene has been popped by the closing click.
func NewNewspaper(ss *Session, md *game.MatchDay, mi int, done func(*App)) Scene {
	if md == nil || !md.HasNewspaper(mi) {
		return nil
	}
	return &NewspaperScene{ss: ss, md: md, mi: mi, done: done, pics: map[int]*ebiten.Image{}}
}

// PushNewspapers shows the newspapers of all human managers after the
// match day md, one after the other in manager order like f_2e3a_23aa, and
// calls done when the last one is closed (at once when there is none).
func PushNewspapers(app *App, ss *Session, md *game.MatchDay, done func(*App)) {
	var next func(app *App, mi int)
	next = func(app *App, mi int) {
		for ; mi < len(ss.S.Managers); mi++ {
			k := mi
			if sc := NewNewspaper(ss, md, k, func(app *App) { next(app, k+1) }); sc != nil {
				app.Push(sc)
				return
			}
		}
		if done != nil {
			done(app)
		}
	}
	next(app, 0)
}

func (n *NewspaperScene) content(app *App) *game.Newspaper {
	if n.np == nil {
		g := app.Gfx
		n.np = n.md.Newspaper(n.mi, func(s string) int { return g.TextWidth(FontSmall, s) })
		if n.np == nil {
			n.np = &game.Newspaper{Photo: game.NewsFirstPhoto}
		}
	}
	return n.np
}

func (n *NewspaperScene) Update(app *App) error {
	n.content(app)
	in := &app.Input
	if in.Left || in.Right || in.Enter || in.Escape {
		app.Pop()
		if n.done != nil {
			n.done(app)
		}
	}
	return nil
}

// pic returns picture k in the colours of the newspaper.
func (n *NewspaperScene) pic(g *Gfx, k int) *ebiten.Image {
	if img, ok := n.pics[k]; ok {
		return img
	}
	p, err := g.dir.Pic(k, false)
	if err != nil {
		return g.missing(err)
	}
	rgba := image.NewRGBA(image.Rect(0, 0, p.W, p.H))
	for i, c := range p.Pix {
		rgba.Set(i%p.W, i/p.W, newsPalette(int(c)&15))
	}
	img := ebiten.NewImageFromImage(rgba)
	n.pics[k] = img
	return img
}

// drawNewsText draws s left-aligned at x with its baseline at y.
func drawNewsText(g *Gfx, dst *ebiten.Image, font Font, x, y int, s string, c int) {
	g.Text(dst, font, x, y-g.baseline(font), s, newsPalette(c))
}

func (n *NewspaperScene) Draw(app *App, dst *ebiten.Image) {
	g := app.Gfx
	np := n.content(app)
	dst.Fill(newsPalette(newsBorder))
	FillRect(dst, 5, 1, 0x13a-5+1, 0xee-1+1, newsPalette(newsPaper))
	Draw(dst, n.pic(g, 44).SubImage(image.Rect(0, 0, 299, 30)).(*ebiten.Image), 10, 3)

	photo := n.pic(g, np.Photo)
	w, h := photo.Bounds().Dx(), photo.Bounds().Dy()
	x, y := (0x6b-w)/2+0xca, (0x6a-h)/2+0x3e
	FrameRect(dst, x-2, y-2, w+4, h+4, newsPalette(newsInk))
	Draw(dst, photo, x, y)

	for i, s := range np.Headline {
		yy := game.NewsHeadlineY1
		if i == 1 {
			yy = game.NewsHeadlineY2
		}
		// f_06c7_087c centres at x0 + (x1-x0)/2 - w/2 + 1 with the width
		// of the original; the shots match x0 + (x1-x0)/2 - w/2 with the
		// width of the remake (two pixels less for this font).
		drawNewsText(g, dst, FontNormal, 0x13f/2-g.TextWidth(FontNormal, s)/2, yy, s, newsInk)
	}
	for _, l := range np.Article {
		drawNewsText(g, dst, FontSmall, l.X, l.Y, l.Text, newsInk)
	}
	for _, l := range np.Box {
		drawNewsLine(g, dst, l)
	}
}

// drawNewsLine draws a line of the line-up box like f_2e3a_060a.
func drawNewsLine(g *Gfx, dst *ebiten.Image, l game.NewsLine) {
	if !l.Split {
		drawNewsText(g, dst, FontSmall, l.X, l.Y, l.Text, newsInk)
		return
	}
	i := strings.IndexByte(l.Text, ':')
	if i < 0 {
		drawNewsText(g, dst, FontSmall, l.X, l.Y, l.Text, newsText)
		return
	}
	pre := l.Text[:i+1]
	drawNewsText(g, dst, FontSmall, l.X, l.Y, pre, newsPrefix)
	rest := ""
	if i+2 <= len(l.Text) {
		rest = l.Text[i+2:]
	}
	drawNewsText(g, dst, FontSmall, l.X+g.TextWidth(FontSmall, pre), l.Y, rest, newsText)
}
