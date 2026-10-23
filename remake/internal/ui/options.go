package ui

import (
	"github.com/hajimehoshi/ebiten/v2"

	"github.com/schowave/bmp/remake/internal/game"
)

// OptionsScreen is the options screen of the disk group (f_262f_01b8): AN/AUS
// switches for what the match days show and the speed slider. Its hidden
// keys change the level and cheat (game.State.Cheat).
type OptionsScreen struct {
	ss    *Session
	flash int // frames left of the frame that answers a hidden key
}

// NewOptionsScreen creates the options screen.
func NewOptionsScreen(ss *Session) Scene { return &OptionsScreen{ss: ss} }

// optSwitchPos are the 15 switches (D:0x5454 x, D:0x5464 y), in the order
// of the flags D:0x5fe..0x60c: Halbzeitstände, Ergebnisse, Tabelle of each
// league, Torszenen, DfB-Pokal, Europapokale, Nachholspiele, Zeitung,
// Blenden.
var optSwitchPos = [15][2]int{
	{122, 35}, {122, 54}, {122, 73}, {180, 35}, {180, 54}, {180, 73}, {238, 35}, {238, 54}, {238, 73},
	{122, 100}, {122, 119}, {122, 138}, {122, 157}, {238, 100}, {238, 119},
}

// optFlag returns the display flag of switch i.
func optFlag(d *game.Display, i int) *bool {
	if i < 3*game.NumLeagues {
		l := i / 3
		switch i % 3 {
		case 0:
			return &d.HalfTime[l]
		case 1:
			return &d.Results[l]
		default:
			return &d.Table[l]
		}
	}
	return []*bool{&d.GoalScenes, &d.DFB, &d.Europe, &d.Replays, &d.Newspaper, &d.Fades}[i-3*game.NumLeagues]
}

// The speed slider: the left 100x19 pixels of PIC/33.VGA at 190,155, the
// knob moves over x = 202..272 (speed 0..70).
const (
	sliderX, sliderY = 190, 155
	knobX0, knobY    = 202, 161
	maxSpeed         = 70
)

func (o *OptionsScreen) Update(app *App) error {
	in := &app.Input
	d := &o.ss.S.Display
	if o.flash > 0 {
		o.flash--
	}
	for _, c := range in.Chars {
		if o.ss.S.Cheat(o.ss.Mgr(), c) {
			o.flash = 8
		}
	}
	if in.Left {
		for i, p := range optSwitchPos {
			if in.In(p[0], p[1], 50, 18) {
				*optFlag(d, i) = in.X < p[0]+26
			}
		}
	}
	if (in.Left || in.LeftHeld) && in.X > 190 && in.X < 290 && in.Y > 151 && in.Y < 171 {
		x := min(max(in.X, knobX0), knobX0+maxSpeed)
		d.Speed = x - knobX0
	}
	// Only the HAUPT MENU button leaves the screen.
	if MenuButtonHit(in) {
		app.Pop()
	}
	return nil
}

func (o *OptionsScreen) Draw(app *App, dst *ebiten.Image) {
	g := app.Gfx
	s := o.ss.S
	d := &s.Display
	g.Background(dst)
	g.Window(dst, 13, 13, 306, 181, ColBlue)
	g.DrawMenuButton(dst)
	ende := " NIE."
	if s.EndYear < 2000 {
		ende = itoa(s.EndYear)
	}
	g.TextAt(dst, FontSmall, 16, 23, "V2.0 - ENDE: "+ende+" (LEVEL "+itoa(5-s.Level)+")", ColGrey3, NoShadow)
	label := func(x0, x1, y int, t string) {
		g.TextAt(dst, FontNormal, x0+(x1-x0-g.TextWidth(FontNormal, t))/2, y, t, ColWhite, ColBlack)
	}
	for i, t := range []string{"Halbzeitstände", "Ergebnisse", "Tabelle", "Torszenen", "DfB-Pokal", "Europapokale", "Nachholspiele"} {
		label(22, 121, []int{46, 64, 83, 111, 130, 148, 168}[i], t)
	}
	label(172, 237, 111, "Zeitung")
	label(172, 237, 130, "Blenden")
	for l, x := range []int{125, 184, 242} {
		g.TextAt(dst, FontNormal, x, 29, itoa(l+1)+".Liga", ColLight, NoShadow)
	}
	// The lines take the colour of the last text: light, then white.
	FillRect(dst, 16, 17, 289, 1, g.PalColor(ColLight))
	FillRect(dst, 16, 98, 289, 1, g.PalColor(ColWhite))
	FillRect(dst, 16, 180, 289, 1, g.PalColor(ColWhite))
	label(182, 300, 150, "Spielgeschwindigkeit")
	for i, p := range optSwitchPos {
		on := *optFlag(d, i)
		drawOptSwitch(g, dst, p[0], p[1], false, on)
		drawOptSwitch(g, dst, p[0]+26, p[1], true, !on)
	}
	Draw(dst, g.Sub(33, rectWH(0, 0, 100, 19)), sliderX, sliderY)
	drawKnob(g, dst, min(max(d.Speed, 0), maxSpeed))
	if o.flash > 0 {
		c := g.PalColor(ColLight)
		FillRect(dst, 0, 0, ScreenW, 1, c)
		FillRect(dst, 0, 0, 1, ScreenH, c)
		FillRect(dst, ScreenW-1, 0, 1, ScreenH, c)
	}
}

// drawKnob draws the track and the knob of the speed slider (f_262f_0086).
func drawKnob(g *Gfx, dst *ebiten.Image, speed int) {
	FillRect(dst, knobX0, knobY, 76, 8, g.PalColor(ColGrey5))
	x := knobX0 + speed
	FillRect(dst, x, knobY, 1, 1, g.PalColor(16))
	FillRect(dst, x, knobY+1, 1, 6, g.PalColor(17))
	FillRect(dst, x+1, knobY, 5, 1, g.PalColor(17))
	FillRect(dst, x+1, knobY+6, 5, 1, g.PalColor(19))
	FillRect(dst, x+5, knobY+1, 1, 6, g.PalColor(19))
	FillRect(dst, x+1, knobY+1, 4, 5, g.PalColor(18))
}
