package ui

import (
	"testing"

	"github.com/schowave/bmp/remake/internal/game"
)

func leagueSession(t *testing.T) (*App, *Session) {
	t.Helper()
	app, mana := testApp(t)
	s, err := game.NewGame(mana, game.Options{Managers: []game.NewManager{{Name: "TEST", Team: 9}}, Level: 2, Seed: 5})
	if err != nil {
		t.Fatal(err)
	}
	s.Start()
	ss := &Session{S: s, Mana: mana, SaveDir: t.TempDir()}
	app.Reset(NewDriver(ss))
	return app, ss
}

func TestTableScreenClicks(t *testing.T) {
	app, ss := leagueSession(t)
	ts := NewTableScreen(ss, 0).(*TableScreen)
	app.Push(ts)
	// The buttons switch the mode; the whole strip below the table counts.
	step(t, app, click(0x90, 0xb0))
	if ts.mode != game.TableTotal {
		t.Fatalf("mode %d after GESAMT", ts.mode)
	}
	step(t, app, click(40, 180))
	if ts.mode != game.TableHome {
		t.Fatalf("mode %d after HEIM", ts.mode)
	}
	step(t, app, click(0xd7, 180))
	if ts.mode != game.TableAway {
		t.Fatalf("mode %d after AUSWÄRTS", ts.mode)
	}
	// A row opens the club info in the mode of the table.
	step(t, app, click(60, 0x1b+7*3))
	ti, ok := top(app).(*TeamInfo)
	if !ok {
		t.Fatal("row click did not open the club info")
	}
	if want := ss.S.LeagueTable(0, game.TableAway)[3].Club; ti.team != want || ti.mode != 1 {
		t.Fatalf("info for %d mode %d, want %d mode 1", ti.team, ti.mode, want)
	}
	app.Pop()
	// The right button does nothing, HAUPT MENU leaves.
	step(t, app, rclick())
	if top(app) != Scene(ts) {
		t.Fatal("right click left the table")
	}
	step(t, app, click(MenuButton.X+10, MenuButton.Y+10))
	if top(app) == Scene(ts) {
		t.Fatal("HAUPT MENU did not leave")
	}
}

func TestTableRoundWeiter(t *testing.T) {
	app, ss := leagueSession(t)
	called := false
	sc := NewTableRound(ss, 1, func(*App) { called = true })
	app.Push(sc)
	step(t, app, click(MenuButton.X+10, MenuButton.Y+10))
	if !called || top(app) == sc {
		t.Fatal("WEITER did not pop and call done")
	}
}

func TestResultsScreenClicks(t *testing.T) {
	app, ss := leagueSession(t)
	rs := NewResultsScreen(ss, 0).(*ResultsScreen)
	app.Push(rs)
	if rs.md != 1 {
		t.Fatalf("first matchday %d", rs.md)
	}
	step(t, app, click(resPrevX+20, resIconY+20))
	if rs.md != game.MatchdaysPerLeague[0] {
		t.Fatalf("left arrow from 1 gave %d", rs.md)
	}
	step(t, app, click(resNextX+20, resIconY+20))
	step(t, app, click(resNextX+20, resIconY+20))
	if rs.md != 2 {
		t.Fatalf("right arrow gave %d", rs.md)
	}
	// The away team of the third match.
	step(t, app, click(200, 0xe+2*resRowH+5))
	ti, ok := top(app).(*TeamInfo)
	if !ok {
		t.Fatal("click did not open the club info")
	}
	if p := game.Fixtures(0, 2)[2]; ti.team != p.Away || ti.mode != 1 {
		t.Fatalf("info for %d mode %d, want %d mode 1", ti.team, ti.mode, p.Away)
	}
}

func TestResultsRound(t *testing.T) {
	app, ss := leagueSession(t)
	called := false
	rs := NewResultsRound(ss, 2, 45, func(*App) { called = true }).(*ResultsScreen)
	app.Push(rs)
	// No arrows during the match day.
	step(t, app, click(resPrevX+20, resIconY+20))
	if rs.md != ss.S.Matchday[2] {
		t.Fatal("the arrows work during the match day")
	}
	if got := rs.title(); got != "SPIELE  Am.-Oberliga   "+itoa(rs.md)+".SPIELTAG    45.MINUTE" {
		t.Fatalf("title %q", got)
	}
	step(t, app, click(MenuButton.X+10, MenuButton.Y+10))
	if !called {
		t.Fatal("WEITER did not call done")
	}
}

func TestStrengthScreen(t *testing.T) {
	app, ss := leagueSession(t)
	st := NewStrengthScreen(ss, 1).(*StrengthScreen)
	app.Push(st)
	for i := 1; i <= 4; i++ {
		step(t, app, click(100, 0xb3))
		if st.part != i%4 {
			t.Fatalf("part %d after %d clicks", st.part, i)
		}
	}
	ids := st.order()
	for i := 1; i < len(ids); i++ {
		sum := func(id int) int {
			tm := &ss.S.Teams[id]
			v := 0
			for k := range 3 {
				v += tm.Strength[k] + tm.Base[k] + tm.Extra[k]
			}
			return v
		}
		if sum(ids[i-1]) < sum(ids[i]) {
			t.Fatalf("not sorted at %d", i)
		}
	}
	step(t, app, click(100, 0x26+7))
	if ti, ok := top(app).(*TeamInfo); !ok || ti.team != ids[1] || ti.mode != 2 {
		t.Fatal("row click did not open the info of the second club")
	}
}

func TestTableLine(t *testing.T) {
	r := game.TableRow{Games: 1, Wins: 1, Points: 2, Goals: 4, Form: "@@@@@@@s"}
	f := string(lgFigSpace)
	want := "@@@@@@@s   " + f + "1     " + f + "1 " + f + "0 " + f + "0    " + f + "2:" + f + "0    " +
		f + f + "4:" + f + f + "0      +" + f + "4"
	if got := lgTableLine(r); got != want {
		t.Fatalf("%q\nwant %q", got, want)
	}
}
