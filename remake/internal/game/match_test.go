package game

import "testing"

// f_0310_0995 shows the flag of the competition before it plays the matches,
// so before the forfeit of f_1a58_2051.
func TestMatchFlagBeforeForfeit(t *testing.T) {
	s := lineupState(t)
	s.EndTurn()
	s.Managers[0].Effort = ForfeitEffort
	s.TakeNotices()
	if md := s.NextMatchDay(); md == nil || md.Title != "Ligaspiel" {
		t.Fatalf("first match day %+v", md)
	}
	q := s.TakeNotices()
	if len(q) != 2 || q[0].Kind != NoticeFlag || q[0].Title != "Ligaspiel" || q[1].Kind != NoticeAlert {
		t.Fatalf("notices %+v", q)
	}
}

// An AI shoot-out (f_0310_356d) draws who shoots first before it looks at
// the teams, then the goals of both sides until they differ.
func TestAIShootoutDraws(t *testing.T) {
	for seed := uint32(1); seed < 50; seed++ {
		s := &State{Rand: Rand{Seed: seed}}
		f := &Fixture{League: -1, Cup: 0, Pair: 4, Home: 1, Away: 2}
		md := &MatchDay{s: s, Fixtures: []*Fixture{f}}
		md.shootout(0, 4)

		want := Rand{Seed: seed}
		want.Between(0, 1)
		var h, a int
		for h == a {
			h, a = want.Between(2, 5), want.Between(2, 5)
		}
		if got := s.Cups.Result[0][4]; got != h+markPenalties || s.Cups.Result[0][5] != a {
			t.Fatalf("seed %d: %d:%d, want %d:%d", seed, got, s.Cups.Result[0][5], h+markPenalties, a)
		}
		if s.Rand.Seed != want.Seed {
			t.Fatalf("seed %d: generator at %d, want %d", seed, s.Rand.Seed, want.Seed)
		}
	}
}

// f_14a4_14d4 does nothing for a team without goals: no draws, and its
// players get no league game.
func TestAIScorersWithoutGoals(t *testing.T) {
	s := &State{Rand: Rand{Seed: 7}}
	for p := 1; p < 12; p++ {
		s.Persons[p].Team = 3
	}
	s.aiScorersOf(3, 0)
	if s.Rand.Seed != 7 {
		t.Errorf("generator moved to %d", s.Rand.Seed)
	}
	for p := 1; p < 12; p++ {
		if s.Persons[p].LeagueGames != 0 {
			t.Fatalf("person %d: %d league games", p, s.Persons[p].LeagueGames)
		}
	}

	s.aiScorersOf(3, 1)
	if s.Rand.Seed == 7 {
		t.Error("a team with a goal draws nothing")
	}
}

// On a day of the 2. Bundesliga alone, f_0310_2303 still plans the chances
// of the other two leagues: their draws are made, their results stay.
func TestIdleLeaguesArePlanned(t *testing.T) {
	s, err := NewGame(mana(t), Options{Managers: []NewManager{{Name: "TEST", Team: 9}}, Level: 2, Seed: 3})
	if err != nil {
		t.Fatal(err)
	}
	md := s.newMatchDay(2, 0, false)
	for _, f := range md.Fixtures {
		if f.League != 1 {
			t.Fatalf("fixture of league %d", f.League)
		}
	}
	before := s.Rand
	results := s.Results
	md.startPeriod(1, 45)
	all := s.Rand

	s.Rand = before
	md.planFixtures(1, 45, func(*Fixture) bool { return true })
	if s.Rand == all {
		t.Error("the idle leagues draw nothing")
	}
	if s.Results[0] != results[0] || s.Results[2] != results[2] {
		t.Error("planning changes the results of an idle league")
	}

	// Planning the three leagues in their order gives the same draws.
	s.Rand = before
	md.planIdle(0, 1, 45)
	md.planFixtures(1, 45, func(*Fixture) bool { return true })
	md.planIdle(2, 1, 45)
	if s.Rand != all {
		t.Error("the leagues are planned out of order")
	}
}

// twoManagerTie is a DFB-Pokal tie of two human managers; the away
// manager has the lower index.
func twoManagerTie(t *testing.T) (*State, *MatchDay, *Fixture) {
	t.Helper()
	s, err := NewGame(mana(t), Options{Managers: []NewManager{{Name: "GAST", Team: 9}, {Name: "HEIM", Team: 4}}, Level: 2, Seed: 11})
	if err != nil {
		t.Fatal(err)
	}
	s.Cups.Round[CupDFB] = 1
	f := &Fixture{League: -1, Cup: CupDFB, Home: s.Managers[1].Team, Away: s.Managers[0].Team}
	md := &MatchDay{s: s, Fixtures: []*Fixture{f}, humans: make([]humanMatch, 2)}
	return s, md, f
}

// f_1a58_20b2 goes through the managers by index: the away manager who
// comes first gets the estimate for an AI home team as his share.
func TestTieManagersInIndexOrder(t *testing.T) {
	s, md, f := twoManagerTie(t)
	r0, money := s.Rand, s.Managers[0].Money
	price := AITicket[bucket(f.Home)] + s.Rand.Between(0, 1)
	want := money + int64(price)*s.Attendance(0, f.Home, f.Away, 1, true, price)/2
	s.Rand = r0
	md.prepare(0, f)
	if got := s.Managers[0].Money; got != want {
		t.Errorf("away share %d, want %d", got-money, want-money)
	}
	if !md.humans[0].plays || !md.humans[1].plays || md.humans[0].side != 1 {
		t.Errorf("humans %+v", md.humans)
	}
}

// When the away manager forfeits, the home manager still gets his gate.
func TestAwayForfeitKeepsHomeGate(t *testing.T) {
	s, md, f := twoManagerTie(t)
	s.Managers[0].Effort = ForfeitEffort
	home := s.Managers[1].Money
	md.prepare(0, f)
	if f.Forfeit != 2 || f.Watched || md.humans[0].plays || md.humans[1].plays {
		t.Fatalf("forfeit %d watched %v humans %+v", f.Forfeit, f.Watched, md.humans)
	}
	if s.Managers[1].Money <= home || f.Attendance == 0 {
		t.Errorf("home manager got %d for %d spectators", s.Managers[1].Money-home, f.Attendance)
	}
}

// f_0000_272d offers the sponsor subsidy whenever rand(0,6) hits, also
// one of 0 DM after a cheap deal.
func TestSubsidyOfZero(t *testing.T) {
	offered := 0
	for seed := uint32(1); seed < 200; seed++ {
		s := &State{Rand: Rand{Seed: seed}, Managers: []*Manager{{}}}
		s.subsidy(10000)
		if !s.SubsidyOffered {
			continue
		}
		offered++
		if s.PendingSubsidy != 0 {
			t.Fatalf("seed %d: subsidy %d for a deal of 10000", seed, s.PendingSubsidy)
		}
		s.AcceptSubsidy(0, false)
		if s.SubsidyOffered {
			t.Fatal("the offer stays after the answer")
		}
	}
	if offered == 0 {
		t.Error("never offered")
	}
}

// After an injury the strength and the re-planning wait for the
// substitution screen; without one, the next minute closes it.
func TestInjuryWaitsForSubstitution(t *testing.T) {
	s, err := NewGame(mana(t), Options{Managers: []NewManager{{Name: "TEST", Team: 9}}, Level: 2, Seed: 6})
	if err != nil {
		t.Fatal(err)
	}
	s.Start()
	md := s.newMatchDay(7, 0, false)
	md.Step()
	md.injured, md.replanMi, md.replanMinute = []int{0}, 0, md.Minute-1
	if got := md.Injured(); len(got) != 1 || got[0] != 0 {
		t.Fatalf("injured %v", got)
	}
	md.LineupClosed(0)
	if len(md.injured) != 0 || md.replanMi != -1 {
		t.Fatalf("after the screen: injured %v, replan %d", md.injured, md.replanMi)
	}

	md.injured, md.replanMi, md.replanMinute = []int{0}, 0, md.Minute-1
	md.Step()
	if len(md.injured) != 0 || md.replanMi != -1 {
		t.Errorf("next minute: injured %v, replan %d", md.injured, md.replanMi)
	}
}

// After an injury the team has a place free: a player from the bench comes
// on without one taken out, like a player put into the team outside a match; a full team,
// or one with a sent-off player, takes nobody in.
func TestSubstituteInAfterInjury(t *testing.T) {
	s, md, f := twoManagerTie(t)
	md.prepare(0, f)
	if !md.humans[0].plays || md.humans[0].allowed != 11 {
		t.Fatalf("humans %+v", md.humans[0])
	}
	field, bench := -1, -1
	for i := 0; i < SquadMax; i++ {
		sl := &s.Squads[0][i]
		if sl.Empty() || s.Persons[sl.Person].Position() == PosKeeper {
			continue
		}
		switch {
		case sl.Lineup >= 1 && sl.Lineup <= 11 && field < 0:
			field = i
		case sl.Lineup >= 12 && s.available(sl) && bench < 0:
			bench = i
		}
	}
	if field < 0 || bench < 0 {
		t.Fatal("no field player or no reserve")
	}
	if err := md.SubstituteIn(0, bench); err == nil {
		t.Fatal("a full team took a player in")
	}
	inj := &s.Squads[0][field]
	s.injure(inj)
	if err := md.SubstituteIn(0, bench); err != nil {
		t.Fatal(err)
	}
	in := &s.Squads[0][bench]
	// He goes in as outside a match: a free cell of his position, numbers
	// 1..11 in slot order.
	if in.Lineup < 1 || in.Lineup > 11 || in.Col < 0 || s.PlayerAt(0, in.Col, in.Row) != bench ||
		s.CountOnPitch(0) != 11 || md.humans[0].fieldSubs != 1 {
		t.Errorf("came on as %d at %d,%d, %d on the pitch, %d field subs left",
			in.Lineup, in.Col, in.Row, s.CountOnPitch(0), md.humans[0].fieldSubs)
	}
	numbers := map[int]bool{}
	for i := 0; i < SquadMax; i++ {
		if sl := &s.Squads[0][i]; !sl.Empty() && sl.Lineup >= 1 && sl.Lineup <= 11 {
			numbers[sl.Lineup] = true
		}
	}
	if len(numbers) != 11 {
		t.Errorf("numbers %v", numbers)
	}
	// A sent-off player is not replaced.
	other := -1
	for i := 0; i < SquadMax; i++ {
		if sl := &s.Squads[0][i]; i != bench && sl.Lineup >= 12 && s.available(sl) {
			other = i
			break
		}
	}
	md.sendOff(0, 0, bench, 30, false)
	if md.humans[0].allowed != 10 {
		t.Errorf("allowed %d after a red card", md.humans[0].allowed)
	}
	if other >= 0 {
		if err := md.SubstituteIn(0, other); err == nil {
			t.Error("a sent-off player was replaced")
		}
	}
}

// Extra time is shown only for a tie of a human team: with the human match
// decided after 90 minutes the extra time of the other ties is silent.
func TestExtraTimeSilentWithoutHumanTie(t *testing.T) {
	for _, humanTied := range []bool{false, true} {
		s, md, f := twoManagerTie(t)
		f.Watched, f.Pair = true, 0
		ai := &Fixture{League: -1, Cup: CupDFB, Pair: 2, Home: 30, Away: 31}
		md.Fixtures = append(md.Fixtures, ai)
		md.cupMask = 1
		md.periods = [][2]int{{1, 45}, {46, 90}}
		s.Cups.Result[CupDFB][0], s.Cups.Result[CupDFB][1] = 3, 0
		if humanTied {
			s.Cups.Result[CupDFB][1] = 3
		}
		s.Cups.Result[CupDFB][2], s.Cups.Result[CupDFB][3] = 1, 1
		md.period = 2
		md.afterRegularTime()
		if len(md.periods) != 4 {
			t.Fatalf("human tied %v: periods %v", humanTied, md.periods)
		}
		if md.Silent() == humanTied {
			t.Errorf("human tied %v: silent %v", humanTied, md.Silent())
		}
	}
}

// f_17c2_1226 applies both DFB tests after 90 and after 120 minutes: a tie
// level or won 10:0 by the home club goes to extra time.
func TestDFBExtraTimeTests(t *testing.T) {
	s := &State{}
	c := &s.Cups
	c.Result[CupDFB] = [32]int{1, 1, 10, 0, 2, 1, 21, 11}
	got := s.needsExtraTime(CupDFB, 8)
	if len(got) != 3 || got[0] != 0 || got[1] != 2 || got[2] != 6 {
		t.Fatalf("ties %v", got)
	}
	if r := c.Result[CupDFB]; r[0] != 11 || r[2] != 20 || r[4] != 2 || r[6] != 31 {
		t.Fatalf("results %v", r[:8])
	}
}

// f_0310_0463 saves the count before it clamps it to 8: with n = 9 a pass
// that postpones nothing is followed by a second one.
func TestPostponeSecondPassAtNine(t *testing.T) {
	nine := false
	for seed := uint32(1); seed < 200; seed++ {
		s := &State{Rand: Rand{Seed: seed}, Event: 44}
		s.Matchday[0] = 1
		for p := range s.Results[0][0] {
			s.Results[0][0][p].Home = Postponed
		}
		s.postpone(0)

		want := Rand{Seed: seed}
		n := 9
		if want.Between(0, 1) != 0 {
			n = 8
		}
		n = want.Between(0, n)
		passes := 0
		if n > 0 {
			passes = 1
		}
		if n == 9 {
			passes, nine = 2, true
		}
		for i := 0; i < passes*len(s.Postponed); i++ {
			want.Between(0, TeamsPerLeague[0]/2-1)
		}
		if s.Rand.Seed != want.Seed {
			t.Fatalf("seed %d (n=%d): generator at %d, want %d", seed, n, s.Rand.Seed, want.Seed)
		}
	}
	if !nine {
		t.Fatal("no seed drew n = 9")
	}
}
