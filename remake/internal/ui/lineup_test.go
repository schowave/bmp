package ui

import (
	"strings"
	"testing"

	"github.com/schowave/bmp/remake/internal/game"
)

func lineupApp(t *testing.T) (*App, *game.State, *SquadScreen) {
	t.Helper()
	app, mana := testApp(t)
	s, err := game.NewGame(mana, game.Options{Managers: []game.NewManager{{Name: "TEST", Team: 9}}, Level: 2, Seed: 5})
	if err != nil {
		t.Fatal(err)
	}
	s.Start()
	for i := range s.Squads[0] {
		s.Squads[0][i].Status &^= 3
	}
	ss := &Session{S: s, Mana: mana, SaveDir: t.TempDir()}
	app.Reset(NewDriver(ss))
	q := NewSquadScreen(ss)
	app.Push(q)
	return app, s, q
}

// cellXY is the middle of a cell of the tactics board.
func cellXY(col, row int) (int, int) { return col*16 + lnCellX + 6, row*14 + lnCellY + 6 }

func TestLineupBoard(t *testing.T) {
	app, s, q := lineupApp(t)
	step(t, app, click(lnIconX+10, lnIconY[1]+10))
	if q.view != lineupPitch {
		t.Fatal("pitch icon did not open the board")
	}
	// A formation button sets the automatic lineup.
	step(t, app, click(60+4+10, lnFormY+10))
	if s.Managers[0].AutoLineup != 3 {
		t.Fatalf("AutoLineup %d, want 3 (3-5-2)", s.Managers[0].AutoLineup)
	}
	gk := s.PlayerAt(0, 3, 7)
	if gk < 0 {
		t.Fatal("no goalkeeper on the board")
	}
	// Picking a player switches the automatic lineup off with a message.
	x, y := cellXY(3, 7)
	step(t, app, click(x, y))
	if _, ok := top(app).(*lineupToast); !ok || s.Managers[0].AutoLineup != 1 || q.picked != gk {
		t.Fatalf("pick: toast %T, auto %d, picked %d", top(app), s.Managers[0].AutoLineup, q.picked)
	}
	for i := 0; i < 100 && top(app) != Scene(q); i++ {
		step(t, app, Input{})
	}
	if top(app) != Scene(q) {
		t.Fatal("message did not close")
	}
	// Moving him to an empty cell, then swapping with another player.
	x, y = cellXY(0, 0)
	step(t, app, click(x, y))
	if s.PlayerAt(0, 0, 0) != gk || q.picked != -1 {
		t.Fatal("not moved to the empty cell")
	}
	other := s.PlayerAt(0, 1, 6)
	step(t, app, click(x, y))
	x, y = cellXY(1, 6)
	step(t, app, click(x, y))
	if s.PlayerAt(0, 1, 6) != gk || s.PlayerAt(0, 0, 0) != other {
		t.Fatal("not swapped")
	}
	// The list icon closes the board, HAUPT MENU the screen.
	step(t, app, click(lnIconX+10, lnIconY[0]+10))
	if q.view != lineupList {
		t.Fatal("board not closed")
	}
	step(t, app, click(lnIconX+10, lnIconY[3]+10))
	if top(app) == Scene(q) {
		t.Fatal("HAUPT MENU did not close the screen")
	}
}

func TestLineupListClicks(t *testing.T) {
	app, s, q := lineupApp(t)
	s.ManualLineup(0)
	in := -1
	for i := 0; i < game.SquadMax; i++ {
		if l := s.Squads[0][i].Lineup; l >= 1 && l <= 11 {
			in = i
			break
		}
	}
	rowY := func(i int) int { return lnRow0Y + 6*i - 2 }
	// A click on the name does nothing, one on the number takes him out.
	step(t, app, click(60, rowY(in)))
	if s.Squads[0][in].Lineup == 0 {
		t.Fatal("click on the name changed the lineup")
	}
	step(t, app, click(10, rowY(in)))
	if s.Squads[0][in].Lineup != 0 || !q.vac.Set {
		t.Fatal("click on the number did not take the player out")
	}
	// A right click makes him a reserve player (if there is room).
	for i := 0; i < game.SquadMax; i++ {
		if s.Squads[0][i].Lineup >= 12 {
			s.Squads[0][i].Lineup = 0
		}
	}
	step(t, app, rclick2(10, rowY(in)))
	if s.Squads[0][in].Lineup != 12 {
		t.Fatalf("right click: number %d, want 12", s.Squads[0][in].Lineup)
	}
	// The EINSATZ meter follows the mouse.
	step(t, app, click(0x113+10, 20))
	if s.Managers[0].EffortSetting != 10 {
		t.Fatalf("EffortSetting %d, want 10", s.Managers[0].EffortSetting)
	}
	// The contracts icon shows the contract list.
	step(t, app, click(lnIconX+10, lnIconY[2]+10))
	if q.view != lineupContracts || q.pressed() != 2 {
		t.Fatal("contracts icon did not switch the view")
	}
}

func rclick2(x, y int) Input { return Input{X: x, Y: y, Right: true} }

// matchApp starts the first match day of the human club and shows it.
func matchApp(t *testing.T) (*App, *game.State, *Session, *game.MatchDay, *MatchScreen) {
	t.Helper()
	app, s, _ := lineupApp(t)
	app.Pop()
	var md *game.MatchDay
	for i := 0; i < 40 && md == nil; i++ {
		switch s.Loop.Phase {
		case game.PhaseTurn:
			s.EndTurn()
		case game.PhaseMatches:
			for md = s.NextMatchDay(); md != nil && md.Done; md = s.NextMatchDay() {
			}
		}
	}
	if md == nil {
		t.Fatal("no match day")
	}
	md.Step()
	ss := &Session{S: s, SaveDir: t.TempDir()}
	m := NewMatchScreen(ss, md)
	if len(m.panels) == 0 {
		t.Fatal("no scoreboard of the own club")
	}
	app.Push(m)
	return app, s, ss, md, m
}

// A click on the logo of the own club on a scoreboard opens the
// substitutions (f_0310_3314); a click elsewhere plays the period at once.
func TestMatchLogoOpensSubstitutions(t *testing.T) {
	app, s, _, md, m := matchApp(t)
	p := m.panels[0]
	f := md.Fixtures[p.fixture]
	x := p.x + 0x16 + 10
	if f.Away == s.Managers[0].Team {
		x = p.x + 0x65 + 10
	}
	step(t, app, click(x, p.y+0x23+10))
	if q, ok := top(app).(*SquadScreen); !ok || q.md != md || m.fast {
		t.Fatalf("logo click: %T, fast %v", top(app), m.fast)
	}
	app.Pop()
	step(t, app, click(p.x+0x50, p.y+5))
	if top(app) != Scene(m) || !m.fast {
		t.Fatalf("click beside: %T, fast %v", top(app), m.fast)
	}
}

// A match day without a match of a human team shows nothing and does not
// wait: a frame plays the period (spec_match §8).
func TestQuietMatchDayRunsAtOnce(t *testing.T) {
	app, s, _ := lineupApp(t)
	app.Pop()
	ss := &Session{S: s, SaveDir: t.TempDir()}
	for i := 0; i < 400; i++ {
		switch s.Loop.Phase {
		case game.PhaseTurn:
			s.EndTurn()
			continue
		case game.PhaseMatches:
		default:
			t.Fatalf("phase %v", s.Loop.Phase)
		}
		md := s.NextMatchDay()
		if md == nil {
			continue
		}
		m := NewMatchScreen(ss, md)
		if !m.quiet() {
			md.Run()
			continue
		}
		app.Push(m)
		step(t, app, Input{})
		if md.Minute < 45 {
			t.Fatalf("minute %d after one frame of a quiet match day", md.Minute)
		}
		return
	}
	t.Skip("no match day without the human team")
}

// rowXY is a point on the number of slot i in the lineup list.
func rowXY(i int) (int, int) { return lnListX0 + 3, lnRow0Y - 4 + 6*i + 2 }

// During a match the lineup substitutes: a player of the team is taken
// out, a reserve player comes in with his number; a player neither in the
// team nor a reserve cannot take part, and WEITER goes back to the match.
func TestMatchLineupSubstitutes(t *testing.T) {
	app, s, ss, md, m := matchApp(t)
	q := NewMatchLineup(ss, md, 0)
	app.Push(q)
	out, in, none := -1, -1, -1
	for i := range s.Squads[0] {
		sl := &s.Squads[0][i]
		field := !sl.Empty() && s.Persons[sl.Person].Position() != game.PosKeeper
		switch {
		case field && sl.Lineup >= 1 && sl.Lineup <= 11 && out < 0:
			out = i
		case field && sl.Lineup > 11 && sl.Status&3 == 0 && in < 0:
			in = i
		case !sl.Empty() && sl.Lineup == 0 && none < 0:
			none = i
		}
	}
	if out < 0 || in < 0 || none < 0 {
		t.Fatalf("slots out %d in %d none %d", out, in, none)
	}
	step(t, app, click(rowXY(none)))
	if ts, ok := top(app).(*lineupToast); !ok || ts.lines != game.ErrNotOnPitch {
		t.Fatalf("player not nominated: %T", top(app))
	}
	for i := 0; i < 200 && top(app) != Scene(q); i++ {
		step(t, app, Input{})
	}
	nr := s.Squads[0][out].Lineup
	step(t, app, click(rowXY(out)))
	if q.out != out {
		t.Fatalf("taken out %d, want %d", q.out, out)
	}
	step(t, app, click(rowXY(in)))
	if s.Squads[0][in].Lineup != nr || s.Squads[0][out].Lineup != 0 || s.Managers[0].Subs != 1 || q.out != -1 {
		t.Fatalf("substitution: in %d out %d subs %d", s.Squads[0][in].Lineup, s.Squads[0][out].Lineup, s.Managers[0].Subs)
	}
	step(t, app, click(lnIconX+10, lnIconY[3]+10))
	if top(app) != Scene(m) {
		t.Fatalf("WEITER: %T", top(app))
	}
}

// An injury of the human club is shown first; the lineup opens after it
// (f_1a58_1f75). A sent-off player is not replaced: no lineup.
func TestInjuryShowsThenSubstitutes(t *testing.T) {
	for _, kind := range []game.EventKind{game.EvInjury, game.EvRed} {
		app, s, _, md, m := matchApp(t)
		p := m.panels[0]
		side := 0
		if md.Fixtures[p.fixture].Away == s.Managers[0].Team {
			side = 1
		}
		md.Events = append(md.Events, game.MatchEvent{Kind: kind, Manager: 0, Fixture: p.fixture, Side: side, Minute: md.Minute})
		m.collect(app)
		if top(app) != Scene(m) || len(m.shows) == 0 {
			t.Fatalf("%v before the event: %T, %d shows", kind, top(app), len(m.shows))
		}
		for i := 0; i < 1000 && top(app) == Scene(m) && len(m.shows) > 0; i++ {
			step(t, app, Input{})
		}
		q, ok := top(app).(*SquadScreen)
		if kind == game.EvInjury && (!ok || q.md != md || len(m.shows) > 0) {
			t.Fatalf("after the injury: %T, %d shows left", top(app), len(m.shows))
		}
		if kind == game.EvRed && ok {
			t.Fatal("the lineup opens after a red card")
		}
	}
}

// The screens after a period follow each other at once: closing one opens
// the next in the same tick, so the match screen does not flash between
// them.
func TestMatchEndScreensFollowAtOnce(t *testing.T) {
	app, s, _, _, m := matchApp(t)
	for l := range s.Display.Results {
		s.Display.HalfTime[l], s.Display.Results[l], s.Display.Table[l] = true, true, true
	}
	shown := 0
	for i := 0; i < 5000; i++ {
		switch top(app).(type) {
		case *MatchScreen:
			step(t, app, click(160, 120))
		case *ResultsScreen, *TableScreen, *CupScreen:
			shown++
			step(t, app, click(MenuButton.X+10, MenuButton.Y+10))
			if top(app) == Scene(m) && len(m.end) > 0 {
				t.Fatalf("match screen on top after screen %d", shown)
			}
		case *NewspaperScene:
			shown++
			step(t, app, click(160, 120))
		default:
			if shown < 3 {
				t.Fatalf("only %d screens after the match day", shown)
			}
			return
		}
	}
	t.Fatal("match day did not end")
}

// Without a human match the match screen shows nothing: it plays on to the
// next screen in the same tick and is never on top on its own.
func TestQuietMatchScreenNeverOnTop(t *testing.T) {
	app, s, _, _, m := matchApp(t)
	for l := range s.Display.Results {
		s.Display.HalfTime[l], s.Display.Results[l], s.Display.Table[l] = true, true, true
	}
	m.panels = nil // as if no human club played
	step(t, app, Input{})
	shown := 0
	for i := 0; i < 100; i++ {
		switch top(app).(type) {
		case *MatchScreen:
			t.Fatalf("match screen on top after screen %d", shown)
		case *ResultsScreen, *TableScreen, *CupScreen:
			shown++
			step(t, app, click(MenuButton.X+10, MenuButton.Y+10))
		case *NewspaperScene:
			shown++
			step(t, app, click(160, 120))
		default:
			if shown < 3 {
				t.Fatalf("only %d screens after the match day", shown)
			}
			return
		}
	}
	t.Fatal("match day did not end")
}

// The key o on the open board sets the optimal lineup and says what it is
// worth.
func TestLineupOptimalKey(t *testing.T) {
	app, s, q := lineupApp(t)
	step(t, app, Input{Chars: []rune("o")})
	if _, ok := top(app).(*lineupToast); ok {
		t.Fatal("o works without the board")
	}
	step(t, app, click(lnIconX+10, lnIconY[1]+10))
	rating := s.LineupRating(0)
	opp, _, ok := s.NextOpponent(0)
	step(t, app, Input{Chars: []rune("o")})
	toast, ok2 := top(app).(*lineupToast)
	if !ok2 {
		t.Fatalf("no message, top is %T", top(app))
	}
	if s.LineupRating(0) < rating {
		t.Errorf("rating fell from %d to %d", rating, s.LineupRating(0))
	}
	if toast.lines[0] != "OPTIMAL "+pointsText(s.LineupRating(0)) {
		t.Errorf("message %q", toast.lines)
	}
	if !strings.HasPrefix(toast.lines[1], "STAND. ") {
		t.Errorf("message %q", toast.lines)
	}
	switch {
	case ok && !strings.HasPrefix(toast.third, "GEGEN "+s.Teams[opp].Name[:3]):
		t.Errorf("message %q does not name the opponent %s", toast.lines, s.Teams[opp].Name)
	case !ok && toast.third != "":
		t.Errorf("message %q", toast.lines)
	}
	// The text must fit into the box (167..317), also with three digits:
	// midX moves a text that would end behind x = 319 to the left edge.
	for _, l := range []string{"OPTIMAL 2,00", "STAND. 2,00/2,00/2,00"} {
		if w := app.Gfx.TextWidth(FontNormal, l); 244-w/2+1+w > 0x13f {
			t.Errorf("%q is %d pixels wide", l, w)
		}
	}
	if q.picked != -1 {
		t.Error("a picked player stays picked")
	}
}

// The name of the opponent is cut to fit the box.
func TestLineupOptimalKeyLongName(t *testing.T) {
	app, s, _ := lineupApp(t)
	opp, _, ok := s.NextOpponent(0)
	if !ok {
		t.Skip("no match today")
	}
	s.Teams[opp].Name = "WWWWWWWWWWWWWWWWWWWWWW"
	step(t, app, click(lnIconX+10, lnIconY[1]+10))
	step(t, app, Input{Chars: []rune("o")})
	toast := top(app).(*lineupToast)
	if w := app.Gfx.TextWidth(FontNormal, toast.third); w > lnToastWidth || !strings.HasPrefix(toast.third, "GEGEN WW") {
		t.Errorf("%q is %d pixels wide", toast.third, w)
	}
}

func TestDecimalText(t *testing.T) {
	for _, c := range []struct {
		n, places int
		want      string
	}{{135, 2, "1,35"}, {7, 2, "0,07"}, {410, 1, "41,0"}, {-12, 1, "-1,2"}, {5, 3, "0,005"}} {
		if got := decimalText(c.n, c.places); got != c.want {
			t.Errorf("%d, %d places: %q, want %q", c.n, c.places, got, c.want)
		}
	}
}

func TestPointsText(t *testing.T) {
	for n, want := range map[int]string{135: "1,35", 7: "0,07", 0: "0,00", -12: "-0,12", 200: "2,00"} {
		if got := pointsText(n); got != want {
			t.Errorf("%d: %q, want %q", n, got, want)
		}
	}
}

// Pressing the key again changes nothing and names no old lineup.
func TestLineupOptimalKeyTwice(t *testing.T) {
	app, _, _ := lineupApp(t)
	step(t, app, click(lnIconX+10, lnIconY[1]+10))
	var first string
	for n := 0; n < 2; n++ {
		step(t, app, Input{Chars: []rune("o")})
		toast := top(app).(*lineupToast)
		if n == 0 {
			first = toast.lines[0]
		} else if toast.lines[0] != first {
			t.Errorf("second press: %q, first %q", toast.lines[0], first)
		}
		for i := 0; i < 200; i++ {
			if _, ok := top(app).(*lineupToast); !ok {
				break
			}
			step(t, app, Input{})
		}
	}
}

// After an injury a click on a reserve player brings him on: the team has a
// place free, nobody has to be taken out first.
func TestMatchLineupReserveAfterInjury(t *testing.T) {
	app, s, ss, md, _ := matchApp(t)
	field, bench := -1, -1
	for i := 0; i < game.SquadMax; i++ {
		sl := &s.Squads[0][i]
		if sl.Empty() || s.Persons[sl.Person].Position() == game.PosKeeper || sl.Status&3 != 0 {
			continue
		}
		switch {
		case sl.Lineup >= 1 && sl.Lineup <= 11 && field < 0:
			field = i
		case sl.Lineup >= 12 && bench < 0:
			bench = i
		}
	}
	if field < 0 || bench < 0 {
		t.Skip("no field player or no reserve")
	}
	sl := &s.Squads[0][field]
	sl.Lineup, sl.Status, sl.Out = 0, sl.Status|2, 3
	q := NewMatchLineup(ss, md, 0)
	app.Push(q)
	step(t, app, click(lnListX0+3, lnRow0Y+6*bench-2))
	if l := s.Squads[0][bench].Lineup; l < 1 || l > 11 {
		t.Fatalf("reserve has number %d, top %T", l, top(app))
	}
	if s.CountOnPitch(0) != 11 {
		t.Errorf("%d on the pitch", s.CountOnPitch(0))
	}
}

// During a match a click on the name substitutes like one on the number.
func TestMatchLineupSubstitutesByName(t *testing.T) {
	app, s, ss, md, _ := matchApp(t)
	q := NewMatchLineup(ss, md, 0)
	app.Push(q)
	step(t, app, click(lnIconX+10, lnIconY[1]+10)) // with the tactics board open
	out, in := -1, -1
	for i := range s.Squads[0] {
		sl := &s.Squads[0][i]
		field := !sl.Empty() && s.Persons[sl.Person].Position() != game.PosKeeper
		switch {
		case field && sl.Lineup >= 1 && sl.Lineup <= 11 && out < 0:
			out = i
		case field && sl.Lineup > 11 && sl.Status&3 == 0 && in < 0:
			in = i
		}
	}
	if out < 0 || in < 0 {
		t.Fatalf("slots out %d in %d", out, in)
	}
	nameX := lnListX0 + lineupColW[0] + lineupColW[1] + 10
	nr := s.Squads[0][out].Lineup
	step(t, app, click(nameX, lnRow0Y+6*out-2))
	// He has left the board: a click on his cell picks nobody.
	o := &s.Squads[0][out]
	col, row := o.Col, o.Row
	step(t, app, click(cellXY(o.Col, o.Row)))
	if q.picked >= 0 {
		t.Errorf("picked slot %d on the cell of the player taken out", q.picked)
	}
	for i := 0; i < 200 && top(app) != Scene(q); i++ {
		step(t, app, Input{}) // the message that the automatic lineup is off
	}
	step(t, app, click(nameX, lnRow0Y+6*in-2))
	n := &s.Squads[0][in]
	if n.Lineup != nr || o.Lineup != 0 {
		t.Fatalf("substitution by name: in %d out %d", n.Lineup, o.Lineup)
	}
	if n.Col != col || n.Row != row {
		t.Errorf("came on at %d,%d, not on the cell %d,%d of the player taken out", n.Col, n.Row, col, row)
	}
}

// Before the matches the flag names the competition for f_3930_01e7(0x32),
// then the matches open.
func TestFlagScreen(t *testing.T) {
	app, _, _, md, _ := matchApp(t)
	if got := flagText(md); got != "Ligaspiel" {
		t.Errorf("flag %q, want Ligaspiel", got)
	}
	opened := false
	b := newFlagScreen("Ligaspiel", func(*App) { opened = true })
	app.Push(b)
	for i := 1; i < waitFrames(0x32); i++ {
		step(t, app, click(160, 120))
		if opened || top(app) != Scene(b) {
			t.Fatalf("frame %d: flag gone early", i)
		}
	}
	step(t, app, Input{})
	if !opened || top(app) == Scene(b) {
		t.Fatal("the matches did not open after the flag")
	}
}

// The board leaves out the player taken out for a substitution, and during a
// match the formation buttons.
func TestBoardDecisions(t *testing.T) {
	app, _, ss, md, _ := matchApp(t)
	q := NewMatchLineup(ss, md, 0)
	app.Push(q)
	if q.formationButtons() {
		t.Error("formation buttons during a match")
	}
	all := q.boardSlots()
	q.out = all[0]
	if got := q.boardSlots(); len(got) != len(all)-1 || got[0] == all[0] {
		t.Errorf("board %v, taken out %d", got, all[0])
	}
	if !NewSquadScreen(ss).formationButtons() {
		t.Error("no formation buttons outside a match")
	}
}

// A message of the original keeps its band and box; one of the remake is
// centred in the band as wide as the box, and a third line makes it taller.
func TestToastLayout(t *testing.T) {
	orig := newLineupToast(game.ErrNoSubs).layout()
	if orig.x0 != 0xb9 || orig.x1 != 0x12f || orig.bottom != 44 || len(orig.lines) != 2 {
		t.Errorf("original %+v", orig)
	}
	r := &lineupToast{lines: [2]string{"A", "B"}, third: "C", wide: true}
	lay := r.layout()
	if lay.x0 != 169 || lay.x1 != 319 || lay.bottom != 44+lnToastLineH || len(lay.ys) != 3 || lay.ys[2] != 0x1a+2*lnToastLineH {
		t.Errorf("remake %+v", lay)
	}
}
