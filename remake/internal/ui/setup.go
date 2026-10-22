package ui

import (
	"time"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/schowave/bmp/remake/internal/assets"
	"github.com/schowave/bmp/remake/internal/game"
)

// Setup is the start screen of a new game (f_08bc_2184): choose a club
// from the logo strip, enter the names of up to four managers, then the
// options with the right mouse button.
type Setup struct {
	mana    *assets.Mana
	saveDir string
	center  int // MANA.DAT index of the club in the middle of the strip
	names   [4]string
	teams   [4]int
	opts    game.Options
	stripX  int
	slotsX  [4]int
}

// NewSetup creates the setup screen.
func NewSetup(mana *assets.Mana, saveDir string) *Setup {
	s := &Setup{mana: mana, saveDir: saveDir, center: 3, stripX: 4, slotsX: [4]int{10, 90, 170, 250}}
	s.teams = [4]int{-1, -1, -1, -1}
	s.opts.Level = 2
	return s
}

const (
	stripY = 42
	slotY  = 103
)

func (s *Setup) clubAt(k int) int {
	return ((s.center+k-3)%game.NumGerman + game.NumGerman) % game.NumGerman
}

func (s *Setup) Update(app *App) error {
	in := &app.Input
	if in.Clicked(s.stripX, stripY, 309, 48) {
		k := (in.X - s.stripX - 13) / 41
		switch {
		case k < 3:
			s.center = (s.center - (3 - k) + game.NumGerman) % game.NumGerman
		case k > 3:
			s.center = (s.center + (k - 3)) % game.NumGerman
		}
		return nil
	}
	if in.WheelDelta != 0 {
		if in.WheelDelta > 0 {
			s.center = (s.center + game.NumGerman - 1) % game.NumGerman
		} else {
			s.center = (s.center + 1) % game.NumGerman
		}
	}
	for i, x := range s.slotsX {
		i := i
		if in.Clicked(x, slotY+115, 58, 18) || (in.Clicked(x, slotY, 58, 50) && s.names[i] == "") {
			app.Push(NewTextInput("Manager "+itoa(i+1), "NAME", s.names[i], 18, func(app *App, v string, ok bool) {
				if ok {
					s.names[i] = v
					if v == "" {
						s.teams[i] = -1
					}
				}
			}))
			return nil
		}
		if in.Clicked(x, slotY, 58, 50) {
			taken := false
			for k, t := range s.teams {
				if k != i && t == s.center {
					taken = true
				}
			}
			if !taken {
				s.teams[i] = s.center
			}
			return nil
		}
	}
	if in.Right || in.Enter {
		if s.ready() {
			app.Push(newSetupOptions(s))
		} else {
			app.Push(NewMessage("An Alle:", "Bitte Namen eintragen und einen Verein in das Feld über dem Gesicht setzen."))
		}
	}
	return nil
}

func (s *Setup) ready() bool {
	n := 0
	for i := range s.names {
		if s.names[i] != "" {
			if s.teams[i] < 0 {
				return false
			}
			n++
		}
	}
	return n > 0
}

func (s *Setup) start(app *App) {
	opt := s.opts
	opt.Managers = nil
	for i := range s.names {
		if s.names[i] != "" {
			opt.Managers = append(opt.Managers, game.NewManager{Name: s.names[i], Team: s.teams[i], Face: i})
		}
	}
	opt.Seed = uint32(time.Now().UnixNano())
	st, err := game.NewGame(s.mana, opt)
	if err != nil {
		app.Push(NewMessage("An Alle:", err.Error()))
		return
	}
	st.Start()
	ss := &Session{S: st, Mana: s.mana, SaveDir: s.saveDir}
	app.Reset(NewDriver(ss))
}

func (s *Setup) Draw(app *App, dst *ebiten.Image) {
	g := app.Gfx
	g.Background(dst)
	Draw(dst, g.Pic(4, false), 0, 0)
	Draw(dst, g.Sub(34, rectWH(0, 0, 309, 66)), s.stripX, stripY)
	for k := 0; k < 7; k++ {
		c := s.clubAt(k)
		Draw(dst, g.Pic(s.mana.Clubs[c].Logo, false), s.stripX+13+41*k, stripY+8)
	}
	FrameRect(dst, s.stripX+13+41*3-1, stripY+7, 40, 40, g.C(ColBeige))
	g.TextCentered(dst, FontNormal, s.stripX, stripY+51, 309, cp437(s.mana.Clubs[s.center].Name), g.C(ColBlack))
	for i, x := range s.slotsX {
		Draw(dst, g.Pic(35, false), x, slotY)
		if t := s.teams[i]; t >= 0 {
			Draw(dst, g.Pic(s.mana.Clubs[t].Logo, false), x+10, slotY+9)
		}
		Draw(dst, g.Sub(0, rectWH(i*41, 0, 40, 40)), x+9, slotY+65)
		g.TextCentered(dst, FontSmall, x, slotY+121, 58, s.names[i], g.C(ColOrange))
	}
}

// cp437 converts a MANA.DAT name for display.
func cp437(s string) string {
	m := map[byte]rune{0x8e: 'Ä', 0x99: 'Ö', 0x9a: 'Ü', 0x84: 'ä', 0x94: 'ö', 0x81: 'ü', 0xe1: 'ß'}
	var r []rune
	for i := 0; i < len(s); i++ {
		if u, ok := m[s[i]]; ok {
			r = append(r, u)
		} else {
			r = append(r, rune(s[i]))
		}
	}
	return string(r)
}

// setupOptions is the option box: level, game length, historic start.
type setupOptions struct {
	s *Setup
}

func newSetupOptions(s *Setup) *setupOptions { return &setupOptions{s: s} }

const optX, optY = 78, 70

func (o *setupOptions) Update(app *App) error {
	in := &app.Input
	opt := &o.s.opts
	for k := 0; k < 6; k++ {
		x, y := optX+104+(k%2)*24, optY+10+(k/2)*18
		if in.Clicked(x, y, 20, 12) {
			opt.Level = k
		}
	}
	lens := []game.GameLength{game.Endless, game.OneYear, game.ThreeYears}
	for i, l := range lens {
		y := optY + 70 + i*16
		if in.Clicked(optX+104, y, 20, 12) {
			opt.Length = l
		}
	}
	if in.Clicked(optX+104, optY+118, 20, 12) {
		opt.Historic = true
	}
	if in.Clicked(optX+128, optY+118, 20, 12) {
		opt.Historic = false
	}
	if in.Right || in.Enter {
		app.Pop()
		o.s.start(app)
	}
	if in.Escape {
		app.Pop()
	}
	return nil
}

func (o *setupOptions) Draw(app *App, dst *ebiten.Image) {
	g := app.Gfx
	opt := &o.s.opts
	FillRect(dst, optX, optY, 160, 140, g.C(ColBlue))
	FrameRect(dst, optX, optY, 160, 140, g.C(ColGrey3))
	btn := func(x, y int, label string, on bool) {
		b := Button{X: x, Y: y, W: 20, H: 12, Label: label}
		b.Draw(g, dst, on)
	}
	g.Text(dst, FontNormal, optX+8, optY+12, "Spiel-Level", g.C(ColLight))
	for k := 0; k < 6; k++ {
		btn(optX+104+(k%2)*24, optY+10+(k/2)*18, itoa(k), opt.Level == k)
	}
	labels := []string{"Endlosspiel", "1-Jahres-Spiel", "3-Jahres-Spiel"}
	lens := []game.GameLength{game.Endless, game.OneYear, game.ThreeYears}
	for i, l := range labels {
		y := optY + 70 + i*16
		g.Text(dst, FontNormal, optX+8, y+2, l, g.C(ColLight))
		btn(optX+104, y, "AN", opt.Length == lens[i])
		btn(optX+128, y, "AUS", opt.Length != lens[i])
	}
	g.Text(dst, FontNormal, optX+8, optY+120, "Histor. Start", g.C(ColLight))
	btn(optX+104, optY+118, "AN", opt.Historic)
	btn(optX+128, optY+118, "AUS", !opt.Historic)
	g.TextCentered(dst, FontSmall, optX, optY+133, 160, "RECHTE MAUSTASTE: SPIEL STARTEN", g.C(ColBeige))
}
