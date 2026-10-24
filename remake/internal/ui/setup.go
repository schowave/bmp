package ui

import (
	"time"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/schowave/bmp/remake/internal/assets"
	"github.com/schowave/bmp/remake/internal/game"
	"github.com/schowave/bmp/remake/internal/seasons"
)

// Setup is the start screen of a new game (f_08bc_2184): choose a club
// from the logo strip, enter the names of up to four managers, then the
// options with the right mouse button; a click on the banner loads a game
// instead. With NEU in the disk menu the same
// screen adds managers to a running game. With real seasons built in, the
// key S opens the season choice (not in the original), named left and
// right of the club name.
type Setup struct {
	mana    *assets.Mana // the clubs of the game
	orig    *assets.Mana // the clubs of the original, for the session
	saveDir string
	ss      *Session // the running game for NEU, else nil

	clubs  []setupClub // the strip, in the order of the club ids
	center int         // index into clubs of the club in the middle
	names  [4]string
	teams  [4]int // index into clubs, -1 = none
	fixed  int    // managers already in the game (NEU)
	edit   int    // name field being typed in, -1 = none
	opts   game.Options
	stripX int
	slotsX [4]int
	held   int // frames the left button has been held on the strip
	// seasons are the real seasons the key S chooses from; none without
	// built-in seasons and for NEU.
	seasons []seasons.Season
}

type setupClub struct {
	name string
	logo int
	id   int // MANA.DAT index for a new game, club id for NEU
}

// NewSetup creates the setup screen of a new game. Like the original, which
// shuffles the club ids of each league before (f_0310_09c5), the strip
// shows the clubs of each league in a random order.
func NewSetup(mana *assets.Mana, saveDir string) *Setup {
	s := newSetup(mana, saveDir)
	r := &game.Rand{Seed: uint32(time.Now().UnixNano())}
	for l := 0; l < game.NumLeagues; l++ {
		first, n := game.FirstID[l], game.TeamsPerLeague[l]
		for i := n - 1; i > 0; i-- {
			k := r.Between(0, i)
			s.clubs[first+i], s.clubs[first+k] = s.clubs[first+k], s.clubs[first+i]
		}
	}
	return s
}

func newSetup(mana *assets.Mana, saveDir string) *Setup {
	s := &Setup{mana: mana, orig: mana, saveDir: saveDir, center: 3, edit: -1, stripX: 5, slotsX: [4]int{12, 92, 172, 252}}
	s.teams = [4]int{-1, -1, -1, -1}
	s.opts.Level = 2
	for i := 0; i < game.NumGerman; i++ {
		s.clubs = append(s.clubs, setupClub{name: game.FromCP437(mana.Clubs[i].Name), logo: mana.Clubs[i].Logo, id: i})
	}
	return s
}

// NewStart creates the setup of a new game after the title: with the clubs
// of the original, and the real seasons to choose from with S.
func NewStart(mana *assets.Mana, saveDir string) *Setup {
	s := NewSetup(mana, saveDir)
	s.seasons = seasons.List()
	return s
}

// NewSeasonSetup creates the setup of a new game of real season sn that
// starts in year; mana holds its clubs, orig those of the original.
func NewSeasonSetup(orig, mana *assets.Mana, saveDir string, sn seasons.Season) *Setup {
	s := NewSetup(mana, saveDir)
	s.orig = orig
	s.opts.Season, s.opts.Year = sn.ID, sn.Year
	return s
}

// NewAddManagers creates the setup screen for NEU: the managers of the game
// are shown, new ones can be added.
func NewAddManagers(ss *Session) *Setup {
	s := newSetup(ss.Mana, ss.SaveDir)
	s.ss = ss
	st := ss.S
	for i := range s.clubs {
		s.clubs[i] = setupClub{name: st.Teams[i].Name, logo: st.Teams[i].Logo, id: i}
	}
	for i, m := range st.Managers {
		s.names[i], s.teams[i] = m.Name, m.Team
	}
	s.fixed = len(st.Managers)
	return s
}

const (
	stripY = 40
	slotY  = 105
	// The hint to the season choice: the clubs played with left of the
	// club name, the key right of it, on the marble of the strip.
	seasonHintY              = 96
	hintLeftX0, hintLeftX1   = 5, 85
	hintRightX0, hintRightX1 = 235, 314
	// Frames until a held button scrolls the strip again.
	stripRepeat = 12
)

func (s *Setup) clubAt(k int) int {
	n := len(s.clubs)
	return ((s.center+k-3)%n + n) % n
}

func (s *Setup) Update(app *App) error {
	in := &app.Input
	if s.edit >= 0 {
		if s.typeName(in) {
			return nil
		}
	}
	// At the start a left click on the banner opens the load screen
	// (f_08bc_2184 calls f_322b_120c for y <= 35); without a loaded game it
	// comes back here.
	if s.ss == nil && in.Left && in.Y <= 35 {
		app.Push(NewFileScreen(&Session{Mana: s.orig, SaveDir: s.saveDir}, false))
		return nil
	}
	// A click left of the middle scrolls the strip back, right of it on;
	// holding the button keeps it moving.
	onStrip := in.In(s.stripX, stripY, 309, 48)
	if onStrip && (in.Left || in.LeftHeld) {
		if in.Left || s.held >= stripRepeat && s.held%(stripRepeat/2) == 0 {
			n := len(s.clubs)
			if in.X < ScreenW/2 {
				s.center = (s.center + n - 1) % n
			} else {
				s.center = (s.center + 1) % n
			}
		}
		s.held++
		return nil
	}
	s.held = 0
	if s.choosesSeason(in) {
		app.Replace(newSeasonChoice(s))
		return nil
	}
	for i, x := range s.slotsX {
		if i < s.fixed {
			continue
		}
		if in.Clicked(x, slotY+113, 58, 20) {
			s.edit = i
			return nil
		}
		if in.Clicked(x, slotY, 58, 60) {
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
		switch {
		case s.ss != nil:
			s.addManagers(app)
		case s.ready():
			app.Push(newSetupOptions(s))
		}
	}
	return nil
}

// typeName handles the keys while a name is typed in its field; it
// returns true when the input was used.
func (s *Setup) typeName(in *Input) bool {
	i := s.edit
	for _, c := range in.Chars {
		if c >= 0x20 && len([]rune(s.names[i])) < 18 {
			s.names[i] += string(c)
		}
	}
	if in.Backspace && s.names[i] != "" {
		r := []rune(s.names[i])
		s.names[i] = string(r[:len(r)-1])
	}
	if in.Enter || in.Escape || in.Left || in.Right {
		s.edit = -1
		if s.names[i] == "" {
			s.teams[i] = -1
		}
		return in.Enter || in.Escape
	}
	return len(in.Chars) > 0 || in.Backspace
}

// choosesSeason reports the key S of the season choice.
func (s *Setup) choosesSeason(in *Input) bool {
	if len(s.seasons) == 0 {
		return false
	}
	for _, c := range in.Chars {
		if c == 's' || c == 'S' {
			return true
		}
	}
	return false
}

// season is the real season the setup plays, ok false with the clubs of
// the original.
func (s *Setup) season() (sn seasons.Season, ok bool) {
	sn, err := seasons.ParseID(s.opts.Season)
	return sn, err == nil
}

// seasonHint is the hint left and right of the club name: the clubs the
// setup plays with ("ORIGINAL 1991/92" or "SAISON 1995/96") and the key
// that changes them; empty without built-in seasons.
func (s *Setup) seasonHint() (left, right string) {
	if len(s.seasons) == 0 {
		return "", ""
	}
	left = "ORIGINAL 1991/92"
	if sn, ok := s.season(); ok {
		left = "SAISON " + sn.Label()
	}
	return left, "S: SAISON WÄHLEN"
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

// addManagers adds the new managers to the running game and goes back.
func (s *Setup) addManagers(app *App) {
	for i := s.fixed; i < len(s.names); i++ {
		if s.names[i] == "" || s.teams[i] < 0 {
			continue
		}
		if err := s.ss.S.AddManager(s.names[i], s.clubs[s.teams[i]].id, i); err != nil {
			app.Push(NewMessage("An Alle:", err.Error()))
			return
		}
	}
	app.Pop()
}

func (s *Setup) start(app *App) {
	opt := s.opts
	opt.Managers = nil
	for i := range s.names {
		if s.names[i] != "" {
			opt.Managers = append(opt.Managers, game.NewManager{Name: s.names[i], Team: s.clubs[s.teams[i]].id, Face: i})
		}
	}
	opt.Seed = uint32(time.Now().UnixNano())
	st, err := game.NewGame(s.mana, opt)
	if err != nil {
		app.Push(NewMessage("An Alle:", err.Error()))
		return
	}
	st.Start()
	ss := &Session{S: st, Mana: s.orig, SaveDir: s.saveDir}
	app.Reset(NewDriver(ss))
}

func (s *Setup) Draw(app *App, dst *ebiten.Image) {
	g := app.Gfx
	g.Background(dst)
	Draw(dst, g.Sub(34, rectWH(0, 0, 309, 65)), s.stripX, stripY)
	Draw(dst, g.Pic(4, false), 0, bannerY)
	for k := 0; k < 7; k++ {
		Draw(dst, g.Pic(s.clubs[s.clubAt(k)].logo, false), s.stripX+12+41*k, stripY+7)
	}
	g.TextCentered(dst, FontNormal, s.stripX+1, stripY+52, 309, s.clubs[s.center].name, g.PalColor(ColBlack))
	if left, right := s.seasonHint(); left != "" {
		g.TextMid(dst, FontSmall, hintLeftX0, hintLeftX1, seasonHintY, left, ColBeige, ColBlack)
		g.TextMid(dst, FontSmall, hintRightX0, hintRightX1, seasonHintY, right, ColBeige, ColBlack)
	}
	for i, x := range s.slotsX {
		Draw(dst, g.Pic(35, false), x, slotY)
		if t := s.teams[i]; t >= 0 {
			Draw(dst, g.Pic(s.clubs[t].logo, false), x+10, slotY+9)
		}
		Draw(dst, g.Sub(0, rectWH(i*41+1, 0, 39, 40)), x+9, slotY+65)
		// The name is typed into its field, with a block cursor.
		g.TextAt(dst, FontSmall, x+8, slotY+125, s.names[i], ColOrange, NoShadow)
		if s.edit == i && (app.frame/20)%2 == 0 {
			FillRect(dst, x+10+g.TextWidth(FontSmall, s.names[i]), slotY+121, 3, 5, g.PalColor(ColOrange))
		}
	}
}

// setupOptions is the option box: level, game length, historic start.
type setupOptions struct {
	s *Setup
}

func newSetupOptions(s *Setup) *setupOptions { return &setupOptions{s: s} }

// Geometry of the option box (f_08bc_2184): a window from 78,69 to
// 242,211; the switches are the cells of PIC/33.VGA, 24x17 pixels.
const (
	optLeftX, optRightX = 181, 207
	optLevelY           = 75
	optLengthY          = 134
)

// optCell returns a cell of PIC/33.VGA: AN or AUS (aus = true), off or on.
func optCell(g *Gfx, aus, on bool) *ebiten.Image {
	x := 100
	if aus {
		x += 24
	}
	if on {
		x += 48
	}
	return g.Sub(33, rectWH(x, 0, 24, 17))
}

// drawOptSwitch draws an AN/AUS switch.
func drawOptSwitch(g *Gfx, dst *ebiten.Image, x, y int, aus, on bool) {
	Draw(dst, optCell(g, aus, on), x, y)
}

// drawOptDigit draws a level button: the cell without its text and a
// digit, grey or orange when chosen.
func drawOptDigit(g *Gfx, dst *ebiten.Image, x, y, k int, on bool) {
	Draw(dst, optCell(g, false, on), x, y)
	FillRect(dst, x+3, y+3, 18, 11, g.PalColor(ColGrey6))
	c := ColGrey4
	if on {
		c = ColOrange
		FillRect(dst, x+21, y+2, 1, 1, g.PalColor(ColBlack))
		FillRect(dst, x+2, y+14, 1, 1, g.PalColor(ColBlack))
		FillRect(dst, x, y+16, 1, 1, g.PalColor(ColLight))
	} else {
		FillRect(dst, x+2, y+2, 1, 1, g.PalColor(ColLight))
	}
	g.TextMid(dst, FontNormal, x, x+24, y+11, itoa(k), c, ColBlack)
}

// seasonLabel is the row that replaces the historic start in a game of a
// real season ("Saison 1995/96"), empty for the clubs of the original.
func (o *setupOptions) seasonLabel() string {
	if sn, ok := o.s.season(); ok {
		return "Saison " + sn.Label()
	}
	return ""
}

var optLengths = []game.GameLength{game.Endless, game.OneYear, game.ThreeYears}

func (o *setupOptions) Update(app *App) error {
	in := &app.Input
	opt := &o.s.opts
	for k := 0; k < 6; k++ {
		x, y := optLeftX+(k%2)*(optRightX-optLeftX), optLevelY+(k/2)*18
		if in.Clicked(x, y, 24, 17) {
			opt.Level = k
		}
	}
	for i, l := range optLengths {
		y := optLengthY + i*18
		if in.Clicked(optLeftX, y, 24, 17) {
			opt.Length = l
		}
	}
	// A real season has its own year: no historic start.
	y := optLengthY + 3*18
	if _, ok := o.s.season(); !ok {
		if in.Clicked(optLeftX, y, 24, 17) {
			opt.Historic = true
		}
		if in.Clicked(optRightX, y, 24, 17) {
			opt.Historic = false
		}
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
	g.Window(dst, 78, 69, 242, 211, ColBlue)
	label := func(y int, s string) {
		g.TextMid(dst, FontNormal, 83, 182, y+12, s, ColLight, ColBlack)
	}
	label(optLevelY, "Spiel-Level")
	for k := 0; k < 6; k++ {
		drawOptDigit(g, dst, optLeftX+(k%2)*(optRightX-optLeftX), optLevelY+(k/2)*18, k, opt.Level == k)
	}
	labels := []string{"Endlosspiel", "1-Jahres-Spiel", "3-Jahres-Spiel"}
	for i, l := range labels {
		y := optLengthY + i*18
		label(y, l)
		on := opt.Length == optLengths[i]
		drawOptSwitch(g, dst, optLeftX, y, false, on)
		drawOptSwitch(g, dst, optRightX, y, true, !on)
	}
	y := optLengthY + 3*18
	if sl := o.seasonLabel(); sl != "" {
		label(y, sl)
		return
	}
	label(y, "Histor. Start")
	drawOptSwitch(g, dst, optLeftX, y, false, opt.Historic)
	drawOptSwitch(g, dst, optRightX, y, true, !opt.Historic)
}
