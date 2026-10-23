package game

import "testing"

func TestLeagueTableModes(t *testing.T) {
	s := &State{}
	for i := 0; i < TeamsPerLeague[0]; i++ {
		s.Tables[0][i] = i
		s.Clubs[i].Form = emptyForm
	}
	// Club 5 wins 3:0 at home against club 2; club 2 wins 1:0 at home
	// against club 7.
	for _, g := range []struct {
		p Pair
		r Score
	}{{Pair{5, 2}, Score{3, 0}}, {Pair{2, 7}, Score{1, 0}}} {
		s.snapshotMatchday([]Pair{g.p}, []Score{g.r})
		s.applyResults([]Pair{g.p}, []Score{g.r}, true)
	}
	s.sortTable(0)

	total := s.LeagueTable(0, TableTotal)
	for i, r := range total {
		if r.Club != s.Tables[0][i] {
			t.Fatalf("total table differs from the sorted table at %d: %d vs %d", i, r.Club, s.Tables[0][i])
		}
	}
	if total[0].Club != 5 || total[0].Points != 2 || total[0].Goals != 3 {
		t.Errorf("total leader %+v", total[0])
	}
	home := s.LeagueTable(0, TableHome)
	if home[0].Club != 5 || home[1].Club != 2 || home[1].Games != 1 || home[1].Against != 0 {
		t.Errorf("home table %+v %+v", home[0], home[1])
	}
	away := s.LeagueTable(0, TableAway)
	// Club 2 lost its away game 0:3, club 7 lost 0:1: the clubs without an
	// away game come first, then 7, then 2.
	n := len(away)
	if away[n-1].Club != 2 || away[n-2].Club != 7 || away[n-1].Losses != 1 || away[n-1].Draws() != 0 {
		t.Errorf("away table end %+v %+v", away[n-2], away[n-1])
	}
	if s.Tables[0][0] != 5 {
		t.Error("LeagueTable changed the table")
	}
}
