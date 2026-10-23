package game

import "testing"

// personState has one human manager in the Bundesliga with 20 players and
// 4 persons abroad; the other persons play in the 2. Bundesliga.
func personState() *State {
	s := &State{Managers: []*Manager{{Team: 5, League: 0}}, Rand: Rand{Seed: 9}}
	for p := 1; p < NumPersons; p++ {
		s.Persons[p] = Person{Team: 30, Owner: OwnerFree, Skill: [3]int{50, 50, 50}, Age: 25}
	}
	for p := 1; p <= 20; p++ {
		s.Persons[p].Team, s.Persons[p].Owner = 5, 0
		s.Squads[0][p-1].Person = p
	}
	for p := 21; p <= 24; p++ {
		s.Persons[p].Team = 100
	}
	return s
}

// f_14a4_1798 returns the share of each league: its human squads among
// the persons taken (human squads and abroad), applied to the others.
func TestReshuffleShare(t *testing.T) {
	s := personState()
	share := s.reshufflePersons()
	foreign := 0
	for p := 1; p < NumPersons; p++ {
		if tm := s.Persons[p].Team; tm >= NumGerman && tm != 0xff {
			foreign++
		}
	}
	taken := 20 + foreign
	want := [NumLeagues]int{20 * 100 / taken * (150 - taken) / 100, 0, 0}
	if share != want {
		t.Errorf("share %v, want %v (%d abroad)", share, want, foreign)
	}
}

// f_0cb5_2987 draws a club of the league without a human manager, from
// its first 13 ids.
func TestLeagueTeam(t *testing.T) {
	s := personState()
	for range 2000 {
		for k := 0; k < NumLeagues; k++ {
			tm := s.leagueTeam(k)
			if tm < FirstID[k] || tm > FirstID[k]+12 || s.ManagerOf(tm) != nil {
				t.Fatalf("league %d: club %d", k, tm)
			}
		}
	}
}

// At the season end the league of the human takes persons not owned by a
// human from the other leagues (f_0cb5_2756); human players stay.
func TestMovePersons(t *testing.T) {
	s := personState()
	s.movePersons()
	inLeague0 := 0
	for p := 1; p < NumPersons; p++ {
		if tm := s.Persons[p].Team; tm <= 17 && s.Persons[p].Owner > 3 {
			inLeague0++
		}
	}
	if inLeague0 < 60 {
		t.Errorf("%d free persons in the Bundesliga", inLeague0)
	}
	for p := 1; p <= 20; p++ {
		if s.Persons[p].Team != 5 {
			t.Errorf("human player %d moved to %d", p, s.Persons[p].Team)
		}
	}
}

// f_0310_05f1 puts at most 8 rescheduled games on one date.
func TestRescheduleLimitPerDate(t *testing.T) {
	for _, n := range []int{8, 9} {
		s := &State{Events: seasonEvents}
		s.Matchday = [NumLeagues]int{1, 1, 1}
		s.Event = 60
		free := s.Event + 1
		for free < 90 && s.Events[free] != 0 {
			free++
		}
		for i := 0; i < n; i++ {
			s.Postponed[i] = PostponedGame{Slot: free, Matchday: 1, League: 1, Pair: i}
		}
		s.Events[free] = EvReplay
		if !s.reschedule(2, 0, n) {
			t.Fatalf("%d games: no date", n)
		}
		got := s.Postponed[n].Slot
		if (n == 8) != (got == free) {
			t.Errorf("%d games on date %d: rescheduled to %d", n, free, got)
		}
	}
}

// f_2a41_2d33 writes the place of every manager's club at the matchday of
// the sorted league, also for a manager of another league.
func TestSortTableWritesEveryManager(t *testing.T) {
	s, err := NewGame(mana(t), Options{Managers: []NewManager{{Name: "EINS", Team: 9}, {Name: "ZWEI", Team: 4}}, Level: 2, Seed: 2})
	if err != nil {
		t.Fatal(err)
	}
	s.Managers[1].League = 1
	s.Matchday[0] = 5
	s.Clubs[s.Managers[1].Team].Pos = 7
	s.Managers[1].PosAfter[4] = 10
	s.sortTable(0)
	if got := s.Managers[1].PosAfter[4]; got != 7 {
		t.Errorf("manager of another league: place %d, want 7", got)
	}
}

// The best scorer of f_14a4_1ad5 needs two goals; of equal scorers the one
// with fewer games wins, and the reserve clubs of the Oberliga do not count.
func TestTopScorerInTeam(t *testing.T) {
	s := &State{Managers: []*Manager{{Team: 40, League: 2}}}
	for p := 1; p < NumPersons; p++ {
		s.Persons[p].Team = 0xff
	}
	s.Persons[5] = Person{Team: 40, LeagueGoals: 1, LeagueGames: 3}
	if s.topScorerInTeam(0) {
		t.Error("one goal makes a top scorer")
	}
	s.Persons[5].LeagueGoals = 12
	s.Persons[3] = Person{Team: 41, LeagueGoals: 12, LeagueGames: 20}
	if !s.topScorerInTeam(0) {
		t.Error("the scorer with fewer games loses")
	}
	s.Persons[2] = Person{Team: 60, LeagueGoals: 30, LeagueGames: 20}
	if !s.topScorerInTeam(0) {
		t.Error("a reserve club counts")
	}
	s.Persons[3].LeagueGames = 2
	if s.topScorerInTeam(0) {
		t.Error("the scorer of the other club with fewer games loses")
	}
}

// The third league takes the name of the real third tier of its season.
func TestLeagueName(t *testing.T) {
	for _, c := range []struct {
		l, year int
		want    string
	}{
		{1, 1963, "Regionalliga"}, {1, 1973, "Regionalliga"}, {1, 1974, "Zweite Liga"}, {1, 2010, "Zweite Liga"},
		{2, 1963, "Amateurliga"}, {2, 1977, "Amateurliga"}, {2, 1978, "Am.-Oberliga"}, {2, 1993, "Am.-Oberliga"},
		{2, 1994, "Regionalliga"}, {2, 2007, "Regionalliga"}, {2, 2008, "3. Liga"}, {0, 1963, "Bundesliga"},
	} {
		if got := LeagueName(c.l, c.year); got != c.want {
			t.Errorf("league %d in %d: %q, want %q", c.l, c.year, got, c.want)
		}
	}
	if got := msgChristmasAmounts(300000, 1993)[1]; got != "AMATEUR-OBERLIGA: 100.000 DM (INCL. MWST.)" {
		t.Errorf("christmas 1993: %q", got)
	}
	if got := msgChristmasAmounts(300000, 1995)[1]; got != "REGIONALLIGA: 100.000 DM (INCL. MWST.)" {
		t.Errorf("christmas 1995: %q", got)
	}
	if got := msgChristmasAmounts(300000, 1963); got[0] != "BUNDESLIGA: 300.000 DM, REGIONALLIGA: 150.000 DM," ||
		got[1] != "AMATEURLIGA: 100.000 DM (INCL. MWST.)" {
		t.Errorf("christmas 1963: %q", got)
	}
}

// History record r belongs to the season that started r seasons after
// the oldest kept one; the last is the season just over.
func TestHistoryYear(t *testing.T) {
	s := &State{Managers: []*Manager{{History: make([][4]byte, 3)}}, StartYear: 1995}
	if y := s.HistoryYear(0, 2); y != 1994 {
		t.Errorf("last record in the new season: %d", y)
	}
	s.Loop.Phase = PhaseContracts
	if y := s.HistoryYear(0, 0); y != 1993 {
		t.Errorf("first record at the season end: %d", y)
	}
}

// A new game puts the persons without a club into the leagues with a
// share (f_14a4_19fb): here only the Bundesliga of the human has one.
func TestDistributePersons(t *testing.T) {
	s := personState()
	for p := 100; p <= 110; p++ {
		s.Persons[p].Team = 0xff
	}
	s.distributePersons()
	// The share of the Bundesliga is larger than the persons without a
	// club, so none is left.
	for p := 1; p < NumPersons; p++ {
		if s.Persons[p].Team == 0xff {
			t.Errorf("person %d without a club", p)
		}
	}
}
