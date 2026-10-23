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
