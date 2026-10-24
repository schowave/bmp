package ui

import (
	"image"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/schowave/bmp/remake/internal/game"
)

// MatchScreen plays a match day live like the original (f_0310_15db,
// f_0310_2303): a scoreboard panel (PIC/38.VGA) for every match of a human
// team, the minute at the bottom and, for a chance, card or injury of a
// human team, the scene window in the free half of the screen. After each
// period the results are shown as the options ask, at the end the tables.
type MatchScreen struct {
	ss *Session
	md *game.MatchDay

	tick    int
	seen    int // events already shown
	shows   []*matchShow
	scene   bool // the scene window is open
	sceneY  int
	fast    bool             // left click: play the rest of the period at once
	builder string           // author of the last goal scene (right click)
	end     []func(app *App) // screens after a period, each opens the next
	closed  bool             // matchDayEnd has queued the last screens

	panels []matchPanel
}

// matchPanel is the scoreboard of one match with a human team.
type matchPanel struct {
	fixture int
	x, y    int
}

// matchShow is an event shown in the scene window, or the ticker line of
// a goal of a club marked with ANZEIGEN (f_0f9d_0c3b).
type matchShow struct {
	ev     game.MatchEvent
	frames int // left to show; the last result frames show the result
	result int
	ticker string
	// A goal scene (Torszenen on): pre frames of the empty stadium, the
	// scene, post frames in which 'r' or 'w' replays it, then the result.
	scene     *matchScene
	pre, post int
	pen       bool
	// after runs when the show is over: the lineup after a red card or an
	// injury of a human team.
	after func(app *App)
}

// Panel positions (D:0x0632, D:0x063c).
var matchPanelPos = [4][2]int{{0, 0}, {161, 0}, {0, 115}, {161, 115}}

// NewMatchScreen creates the screen for a match day.
func NewMatchScreen(ss *Session, md *game.MatchDay) *MatchScreen {
	m := &MatchScreen{ss: ss, md: md}
	for _, mg := range ss.S.Managers {
		for fi, f := range md.Fixtures {
			if f.Home != mg.Team && f.Away != mg.Team {
				continue
			}
			dup := false
			for _, p := range m.panels {
				dup = dup || p.fixture == fi
			}
			if !dup && len(m.panels) < 4 {
				pos := matchPanelPos[len(m.panels)]
				m.panels = append(m.panels, matchPanel{fixture: fi, x: pos[0], y: pos[1]})
			}
		}
	}
	return m
}

// framesPerMinute is the delay of f_0310_2303: f_3930_01e7(0x26 - speed/2).
func (m *MatchScreen) framesPerMinute() int {
	sp := m.ss.S.Display.Speed
	if sp >= 0x47 {
		return 0
	}
	return waitFrames(0x26 - sp/2)
}

// chanceBlink is one half of a blink of "Chance für X !": f_3930_01e7(6)
// dark, then f_3930_01e7(6) lit, four times.
var chanceBlink = waitFrames(6)

func (m *MatchScreen) Update(app *App) error {
	in := &app.Input
	if len(m.end) > 0 {
		m.next(app)
		return nil
	}
	if len(in.Chars) > 0 && in.Chars[0] == ' ' {
		app.Push(NewOptionsScreen(m.ss))
		return nil
	}
	if len(m.shows) > 0 {
		sh := m.shows[0]
		switch {
		case sh.scene != nil && sh.pre > 0:
			sh.pre--
		case sh.scene != nil && !sh.scene.done:
			sh.scene.done = sh.scene.gs.Update()
		case sh.scene != nil && sh.post > 0:
			sh.post--
			for _, c := range in.Chars {
				if (c == 'r' || c == 'w') && sh.scene.gs.Replay() {
					sh.scene.done = false
					sh.post = 20
				}
			}
		default:
			sh.frames--
			if sh.frames <= 0 || in.Left {
				m.shows = m.shows[1:]
				if sh.after != nil {
					sh.after(app)
				}
			}
		}
		return nil
	}
	if in.Right && m.builder != "" {
		app.Push(NewAlertText("Der Erbauer der\nletzten gespielten\nTorszene war:\n" + m.builder))
		return nil
	}
	if in.Left {
		if mi := m.logoHit(in); mi >= 0 {
			app.Push(NewMatchLineup(m.ss, m.md, mi))
			return nil
		}
		m.fast = true
	}
	if !m.fast && !m.quiet() {
		m.tick++
		if m.tick < m.framesPerMinute() {
			return nil
		}
	}
	m.tick = 0
	for {
		more := m.md.Step()
		m.collect(app)
		if to := m.md.PeriodEnded(); to != 0 {
			m.fast = false
			m.periodEnd(to)
		}
		if !more && !m.closed {
			m.matchDayEnd()
		}
		// Without a human match nothing is shown and nothing waits (spec
		// §8): the period is played at once.
		if !m.quiet() || !more || len(m.end) > 0 {
			return nil
		}
	}
}

// quiet reports a match day without a match of a human team, or extra time
// without one (MatchDay.Silent).
func (m *MatchScreen) quiet() bool { return len(m.panels) == 0 || m.md.Silent() }

// Passing implements passer: without a human match the screen shows nothing
// and only plays on to the next screen.
func (m *MatchScreen) Passing() bool { return m.quiet() }

// collect turns the new events of human teams into scenes.
func (m *MatchScreen) collect(app *App) {
	for ; m.seen < len(m.md.Events); m.seen++ {
		e := m.md.Events[m.seen]
		if m.panelOf(e.Fixture) < 0 {
			m.tickerGoal(e)
			continue
		}
		if m.fast && e.Kind != game.EvInjury && e.Kind != game.EvRed && e.Kind != game.EvYellowRed {
			continue
		}
		p := m.panelOf(e.Fixture)
		if !m.scene {
			m.scene = true
			m.sceneY = (3 - p) / 2 * 0x71
		}
		// The chance text blinks four times, then the result stays for
		// f_3930_01e7(0x78 - speed), a card or an injury for 0x82 - speed.
		base := 0x82
		switch e.Kind {
		case game.EvGoal, game.EvMiss, game.EvPenaltyGoal, game.EvPenaltyMiss:
			base = 0x78
		}
		d := waitFrames(base - m.ss.S.Display.Speed)
		sh := &matchShow{ev: e, frames: d, result: d}
		switch e.Kind {
		case game.EvGoal, game.EvMiss, game.EvPenaltyGoal, game.EvPenaltyMiss:
			if m.ss.S.Display.GoalScenes {
				goal := e.Kind == game.EvGoal || e.Kind == game.EvPenaltyGoal
				pen := e.Kind == game.EvPenaltyGoal || e.Kind == game.EvPenaltyMiss
				if sc, _, p, err := PickScene(app.Gfx.dir, &m.ss.S.Rand, goal, pen); err == nil {
					gs := NewGoalScene(app.Gfx, sc, e.Side == 1)
					sh.scene = &matchScene{gs: gs}
					sh.pre, sh.post, sh.pen = waitFrames(0x1e), waitFrames(0x14), p
					m.builder = gs.Builder()
				}
			}
			if sh.scene == nil {
				sh.frames += 8 * chanceBlink // the blinking "Chance für X !"
			}
		}
		m.shows = append(m.shows, sh)
		// After an injury the substitution screen opens (f_1a58_1f75); a
		// sent-off player is not replaced.
		if e.Manager >= 0 && e.Kind == game.EvInjury && !m.md.Done {
			mi := e.Manager
			sh.after = func(app *App) { app.Push(NewMatchLineup(m.ss, m.md, mi)) }
		}
	}
}

// matchScene is a goal scene being played.
type matchScene struct {
	gs   *GoalScene
	done bool
}

// shownScore is the score of fixture fi on its scoreboard: a goal counts
// only once its show reaches the result with the goal picture, not while
// "Chance für X !" blinks or the goal scene runs.
func (m *MatchScreen) shownScore(fi int) (int, int) {
	h, a := m.md.Score(fi)
	if h < 0 {
		return h, a
	}
	for i, sh := range m.shows {
		e := sh.ev
		if sh.ticker != "" || e.Fixture != fi || e.Kind != game.EvGoal && e.Kind != game.EvPenaltyGoal {
			continue
		}
		if i == 0 && sh.resultShown() {
			continue
		}
		if e.Side == 0 {
			h--
		} else {
			a--
		}
	}
	return h, a
}

// resultShown reports whether the show has reached its result.
func (sh *matchShow) resultShown() bool {
	if sh.scene != nil {
		return sh.pre == 0 && sh.scene.done && sh.post == 0
	}
	return sh.frames <= sh.result
}

// tickerGoal shows "HOME - AWAY   h:a" at the bottom for a goal in a
// match of a club whose goals are announced.
func (m *MatchScreen) tickerGoal(e game.MatchEvent) {
	s := m.ss.S
	f := m.md.Fixtures[e.Fixture]
	if e.Kind != game.EvGoal && e.Kind != game.EvPenaltyGoal || !s.Ticker(f.Home) && !s.Ticker(f.Away) || m.fast || m.quiet() {
		return
	}
	h, a := m.md.Score(e.Fixture)
	text := s.Teams[f.Home].Name + " - " + s.Teams[f.Away].Name + "   " + itoa(h%10) + ":" + itoa(a%10)
	m.shows = append(m.shows, &matchShow{ev: e, frames: waitFrames(0x46), ticker: text})
}

// logoHit returns the manager whose club logo on a scoreboard is under the
// pointer, or -1 (f_0310_3314). The original takes the home logo from
// x+0x22, right of where it is drawn at x+0x16; the drawn logo counts too.
func (m *MatchScreen) logoHit(in *Input) int {
	if m.md.Done {
		return -1
	}
	s := m.ss.S
	for _, p := range m.panels {
		if in.Y <= p.y+0x22 || in.Y >= p.y+0x49 {
			continue
		}
		f := m.md.Fixtures[p.fixture]
		team := -1
		switch {
		case in.X >= p.x+0x16 && in.X < p.x+0x4c:
			team = f.Home
		case in.X > p.x+99 && in.X < p.x+0x8f:
			team = f.Away
		}
		for mi, mg := range s.Managers {
			if team >= 0 && mg.Team == team {
				return mi
			}
		}
	}
	return -1
}

func (m *MatchScreen) panelOf(fi int) int {
	for i, p := range m.panels {
		if p.fixture == fi {
			return i
		}
	}
	return -1
}

// periodEnd queues what the original shows after a period (end of
// f_0310_2303): the results of the leagues, the Nachholspiele, or the cup
// round.
func (m *MatchScreen) periodEnd(to int) {
	s := m.ss.S
	md := m.md
	if md.Replays() {
		if s.Display.Replays && replayRound != nil {
			m.end = append(m.end, func(app *App) { app.Push(replayRound(m.ss, md, to, m.next)) })
		}
		return
	}
	if cups := md.Cups(); cups != 0 {
		if s.Day > 0x13b || cups&1 != 0 && s.Display.DFB || cups&^1 != 0 && s.Display.Europe {
			for c := game.CupDFB; c < game.NumCups; c++ {
				if cups&(1<<c) != 0 {
					c := c
					m.end = append(m.end, func(app *App) { app.Push(NewCupRound(m.ss, c, to, m.next)) })
				}
			}
		}
		return
	}
	for l := 0; l < game.NumLeagues; l++ {
		if md.Leagues()&(1<<l) == 0 {
			continue
		}
		show := to == 45 && s.Display.HalfTime[l] || to == 90 && s.Display.Results[l]
		if show && resultsRound != nil {
			l := l
			m.end = append(m.end, func(app *App) { app.Push(resultsRound(m.ss, md, l, to, m.next)) })
		}
	}
}

// matchDayEnd queues the tables of the leagues (f_2a41_214c) and closes.
func (m *MatchScreen) matchDayEnd() {
	s := m.ss.S
	m.closed = true
	if m.md.Cups() == 0 && !m.md.Replays() {
		for l := 0; l < game.NumLeagues; l++ {
			if m.md.Leagues()&(1<<l) != 0 && s.Display.Table[l] && tableRound != nil {
				l := l
				m.end = append(m.end, func(app *App) { app.Push(tableRound(m.ss, l, m.next)) })
			}
		}
	}
	// Then the newspaper of each manager (f_2e3a_23aa), then back.
	m.end = append(m.end, func(app *App) {
		PushNewspapers(app, m.ss, m.md, func(app *App) { app.Pop() })
	})
}

// resultsRound and replayRound show the results of a league or of the
// Nachholspiele after a period, tableRound the table after the match day;
// set by the league screens.
var (
	resultsRound = func(ss *Session, md *game.MatchDay, league, minute int, done func(*App)) Scene {
		return NewResultsRound(ss, league, minute, done)
	}
	replayRound = NewReplayRound
	tableRound  = func(ss *Session, league int, done func(*App)) Scene {
		return NewTableRound(ss, league, done)
	}
)

// next opens the next queued screen of m.end. Each screen calls it when it
// closes, in the same tick, so that the match screen is not drawn for a
// frame between two of them.
func (m *MatchScreen) next(app *App) {
	if len(m.end) == 0 && m.md.Over() && !m.closed {
		// After the last period the match day ends in this tick too: the
		// tables and the newspaper follow the results without the match
		// screen in between.
		m.md.Step()
		m.collect(app)
		m.matchDayEnd()
	}
	if len(m.end) == 0 {
		return
	}
	f := m.end[0]
	m.end = m.end[1:]
	f(app)
}

func (m *MatchScreen) Draw(app *App, dst *ebiten.Image) {
	g := app.Gfx
	if m.quiet() {
		return
	}
	dst.Fill(g.PalColor(ColBlack))
	for i := range m.panels {
		m.drawPanel(g, dst, &m.panels[i])
	}
	minute := m.md.Minute
	if minute > 0 {
		minute--
	}
	if len(m.shows) > 0 && m.shows[0].ticker != "" {
		if m.scene {
			m.drawScene(app, dst, minute)
		}
		FillRect(dst, 0, 0xe7, 320, 9, g.PalColor(ColBlack))
		g.TextMid(dst, FontNormal, 0, 0x140, 0xed, m.shows[0].ticker, ColWhite, NoShadow)
		return
	}
	if m.scene {
		m.drawScene(app, dst, minute)
	} else {
		m.drawMinuteBar(g, dst, minute)
	}
}

// fixtureLogo is the picture of club t on the scoreboard of fixture f
// (f_1a58_042f): the logo of a German club; a foreign club has none and
// shows the trophy of its cup, PIC/0x90+cup. The scene window copies these.
func fixtureLogo(s *game.State, f *game.Fixture, t int) int {
	if t >= game.NumGerman {
		return 0x90 + int(f.Cup)
	}
	return s.Teams[t].Logo
}

func logoSub(g *Gfx, logo int) *ebiten.Image {
	return g.Pic(logo, false).SubImage(image.Rect(0, 0, 38, 38)).(*ebiten.Image)
}

// drawPanel draws a scoreboard (f_1a58_042f, f_1a58_009d, f_1a58_024e).
func (m *MatchScreen) drawPanel(g *Gfx, dst *ebiten.Image, p *matchPanel) {
	s := m.ss.S
	f := m.md.Fixtures[p.fixture]
	x, y := p.x, p.y
	Draw(dst, g.Sub(38, rectWH(0, 0, 155, 114)), x+3, y)
	m.panelName(g, dst, s.Teams[f.Home].Name, x+5, x+0x4b, y+12)
	m.panelName(g, dst, s.Teams[f.Away].Name, x+0x55, x+0x99, y+12)
	Draw(dst, logoSub(g, fixtureLogo(s, f, f.Home)), x+0x16, y+0x23)
	Draw(dst, logoSub(g, fixtureLogo(s, f, f.Away)), x+0x65, y+0x23)
	h, a := m.shownScore(p.fixture)
	for side, v := range []int{h, a} {
		for v > 9 {
			v -= 10
		}
		if v < 0 {
			v = 0
		}
		Draw(dst, g.Sub(38, rectWH(v*14, 114, 11, 15)), x+0x24+side*0x4e, y+0x10)
	}
	// Attendance of a home team with a manager, in red when sold out.
	att, col := "(AUSW.)", ColLight
	if hm := s.ManagerOf(f.Home); hm != nil {
		att = game.FormatMoney(f.Attendance)
		if f.Attendance == hm.Capacity() {
			col = ColRedDark
		}
	}
	g.TextMid(dst, FontSmall, x+9, x+0x33, y+99, att, col, NoShadow)
	for mi, mg := range s.Managers {
		side := -1
		switch mg.Team {
		case f.Home:
			side = 0
		case f.Away:
			side = 1
		}
		if side < 0 {
			continue
		}
		x0 := x + 0x7e + side*16
		FillRect(dst, x0, y+0x4d, 11, 0x70-0x4d+1, g.PalColor(12))
		for i, v := range m.counters(mi, p.fixture, side) {
			g.TextAt(dst, FontSmall, x0, y+0x52+i*7, itoa(v), ColLight, NoShadow)
		}
	}
}

// panelName draws a club name; a name wider than 60 pixels is split at its
// first space (f_1a58_0350).
func (m *MatchScreen) panelName(g *Gfx, dst *ebiten.Image, name string, x0, x1, y int) {
	if g.TextWidth(FontSmall, name) > 0x3c {
		for i, c := range name {
			if c == ' ' {
				g.TextMid(dst, FontSmall, x0, x1, y-5, name[:i], ColLight, NoShadow)
				name, y = name[i+1:], y+1
				break
			}
		}
	}
	g.TextMid(dst, FontSmall, x0, x1, y, name, ColLight, NoShadow)
}

// counters are the lines of a panel for manager mi: yellow and red cards,
// injured, players on the pitch, chances.
func (m *MatchScreen) counters(mi, fi, side int) [5]int {
	var c [5]int
	for _, e := range m.md.Events[:m.seen] {
		if e.Fixture != fi {
			continue
		}
		own := e.Side == side
		switch e.Kind {
		case game.EvYellow:
			if own {
				c[0]++
			}
		case game.EvYellowRed, game.EvRed:
			if own {
				c[1]++
			}
		case game.EvInjury:
			if own {
				c[2]++
			}
		case game.EvGoal, game.EvMiss, game.EvPenaltyGoal, game.EvPenaltyMiss:
			if own {
				c[4]++
			}
		}
	}
	c[3] = m.ss.S.OnPitch(mi)
	return c
}

// drawMinuteBar draws the minute and the two half bars at the bottom
// (f_0310_1def, f_0310_2086).
func (m *MatchScreen) drawMinuteBar(g *Gfx, dst *ebiten.Image, minute int) {
	for _, b := range [][2]int{{35, 126}, {194, 287}} {
		FillRect(dst, b[0], 234, b[1]-b[0]+1, 4, g.PalColor(ColGrey4))
		FillRect(dst, b[0], 234, b[1]-b[0], 1, g.PalColor(ColLight))
		FillRect(dst, b[0], 234, 1, 4, g.PalColor(ColLight))
		FillRect(dst, b[1], 234, 1, 4, g.PalColor(ColGrey7))
		FillRect(dst, b[0]+1, 237, b[1]-b[0], 1, g.PalColor(ColGrey7))
	}
	bar := func(x0, x1 int) {
		if x1 < x0 {
			x0, x1 = x1, x0
		}
		FillRect(dst, x0, 235, x1-x0+1, 1, g.PalColor(ColYellow))
		FillRect(dst, x0, 236, x1-x0+1, 1, g.PalColor(ColWhite))
	}
	switch {
	case minute <= 45:
		if minute > 0 {
			bar(36, 36+2*minute-1)
		}
	case minute <= 90:
		bar(36, 125)
		bar(285-2*(minute-45)+1, 285)
	default:
		bar(36, 125)
		bar(196, 285)
	}
	m.drawMinute(g, dst, minute, 0xef)
}

// drawMinute writes "N.Minute" centred with its baseline at y.
func (m *MatchScreen) drawMinute(g *Gfx, dst *ebiten.Image, minute, y int) {
	FillRect(dst, 0x7f, y-6, 0xc1-0x7f+1, 7, g.PalColor(ColBlack))
	if minute > 0 {
		g.TextMid(dst, FontNormal, 4, 0x140, y, itoa(minute)+".Minute", ColWhite, ColGrey4)
	}
}

// drawScene draws the scene window (f_1a58_0ca3).
func (m *MatchScreen) drawScene(app *App, dst *ebiten.Image, minute int) {
	g := app.Gfx
	s := m.ss.S
	u := m.sceneY
	FillRect(dst, 0, u, 320, 0x7f, g.PalColor(ColBlack))
	FrameRect(dst, 0, u, 320, 0x7f, g.PalColor(ColLight))
	for k := 0; k < 2; k++ {
		Draw(dst, g.Sub(8, rectWH(0xa3, 0x66, 13, 13)), k*0x4e+0x73, u+0x6c)
	}
	for k := 0; k < 2; k++ {
		iv := k * 0xf1
		FrameRect(dst, 0x12+iv, u+3, 0x3b-0x12+1, 0x2c-3+1, g.PalColor(ColGrey2))
		FrameRect(dst, 0x12+iv, u+0x3b, 0x3c-0x12+1, 0x66-0x3b+1, g.PalColor(ColGrey2))
	}
	var f *game.Fixture
	var ev *game.MatchEvent
	if len(m.shows) > 0 && m.shows[0].ticker == "" {
		ev = &m.shows[0].ev
		f = m.md.Fixtures[ev.Fixture]
	} else if len(m.panels) > 0 {
		f = m.md.Fixtures[m.panels[0].fixture]
	}
	if f != nil {
		Draw(dst, logoSub(g, fixtureLogo(s, f, f.Home)), 0x14, u+5)
		Draw(dst, logoSub(g, fixtureLogo(s, f, f.Away)), 0x105, u+5)
		for mi, mg := range s.Managers {
			if mg.Team == f.Home || mg.Team == f.Away {
				x := 0x14
				if mg.Team == f.Away {
					x = 0x105
				}
				Draw(dst, g.Sub(0, rectWH(mg.Face*41+1, 0, 40, 40)), x, u+0x3d)
				_ = mi
			}
		}
	}
	var sh *matchShow
	if ev != nil {
		sh = m.shows[0]
	}
	if sh != nil && sh.scene != nil && sh.pre == 0 {
		sh.scene.gs.Draw(dst, 0x42, u+2)
	} else {
		Draw(dst, g.Sub(8, rectWH(0, 0, 0xbc, 0x66)), 0x42, u+2)
	}
	m.drawMinute(g, dst, minute, u+0x77)
	if ev == nil {
		return
	}
	team := f.Home
	if ev.Side == 1 {
		team = f.Away
	}
	iconX := ev.Side*0xf1 + 0x14
	if sh.scene != nil && (sh.pre > 0 || !sh.scene.done || sh.post > 0) {
		return // the scene runs; clocks and minute stay
	}
	FillRect(dst, 10, u+0x6c, 0x136-10+1, 0x79-0x6c+1, g.PalColor(ColBlack))
	switch ev.Kind {
	case game.EvGoal, game.EvMiss, game.EvPenaltyGoal, game.EvPenaltyMiss:
		if sh.scene != nil && (sh.pre > 0 || !sh.scene.done || sh.post > 0) {
			return
		}
		if sh.scene == nil && sh.frames > sh.result {
			// "Chance für X !" blinks before the result: dark first.
			if (sh.result+8*chanceBlink-sh.frames)/chanceBlink%2 == 1 {
				pre := "Chance für "
				if ev.Kind == game.EvPenaltyGoal || ev.Kind == game.EvPenaltyMiss {
					pre = "Elfmeter für "
				}
				g.TextMid(dst, FontNormal, 0, 0x13f, u+0x75, pre+s.Teams[team].Name+" !", ColBeige, ColBlack)
			}
			return
		}
		goal := ev.Kind == game.EvGoal || ev.Kind == game.EvPenaltyGoal
		line := "Chance vergeben: " + ev.Player
		if goal {
			Draw(dst, g.Sub(8, rectWH(0x7c, 0x66, 39, 40)), iconX, u+0x3d)
			line = "Torschütze: " + ev.Player
			if ev.Kind == game.EvGoal { // not for a shoot-out kick
				line += " (" + itoa(ev.Goals) + ")" // f_1a58_0ca3: league, cup and Europe goals
			}
		}
		if ev.Player == "" {
			line = ""
		}
		g.TextMid(dst, FontNormal, 0, 0x13f, u+0x70, line, ColLight, ColBlack)
		if goal && ev.Assist != "" && !sh.pen {
			g.TextMid(dst, FontNormal, 0, 0x13f, u+0x7a, "nach Vorlage von "+ev.Assist, ColLight, ColBlack)
		}
	case game.EvYellow, game.EvYellowRed, game.EvRed, game.EvInjury:
		icon := map[game.EventKind]int{game.EvYellow: 1, game.EvYellowRed: 0x2a, game.EvRed: 0x2a, game.EvInjury: 0x53}[ev.Kind]
		Draw(dst, g.Sub(8, rectWH(icon, 0x66, 40, 40)), iconX, u+0x3d)
		col := ColRed
		if ev.Kind == game.EvYellow {
			col = ColYellow
		}
		g.TextMid(dst, FontNormal, 0, 0x13f, u+0x75, m.md.EventText(*ev), col, ColDarkRed)
	}
}
