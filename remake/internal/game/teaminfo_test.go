package game

import "testing"

func TestTeamInfoQueries(t *testing.T) {
	s := &State{}
	s.Matchday[0] = 3
	// Matchday 1: club 0 at home against club 1, 3:0.
	s.Results[0][0][0] = Score{3, 0}
	// Matchday 2: club 3 at home against club 0, 2:1.
	s.Results[0][1][6] = Score{2, 1}

	games := s.LeagueGames(0)
	if len(games) != 2 || games[0] != (PlayedGame{Opp: 1, Home: true, For: 3, Against: 0}) ||
		games[1] != (PlayedGame{Opp: 3, Home: false, For: 1, Against: 2}) {
		t.Fatalf("games %+v", games)
	}
	ser := s.TeamSeries(0)
	if ser[SeriesLost][VenueTotal] != 1 || ser[SeriesWon][VenueTotal] != 0 || ser[SeriesWon][VenueHome] != 1 ||
		ser[SeriesNotWon][VenueAway] != 1 || ser[SeriesCleanSheet][VenueTotal] != 0 {
		t.Fatalf("series %v", ser)
	}
	rec := s.TeamRecords(0)
	if r := rec[RecordWin]; r.Hi() != 3 || r.Lo() != 0 || r.Opp != 1 {
		t.Errorf("home win %+v", r)
	}
	// The away defeat is stored as the score of the game, home team first.
	if r := rec[4+RecordDefeat]; r.Hi() != 2 || r.Lo() != 1 || r.Opp != 3 {
		t.Errorf("away defeat %+v", r)
	}
	if r := rec[RecordScored]; r.Hi() != 3 {
		t.Errorf("home scored %+v", r)
	}
	if r := rec[4+RecordScored]; r.Lo() != 1 || r.Hi() != 0 {
		t.Errorf("away scored %+v", r)
	}
	if r := rec[RecordConceded]; r.Raw != 0 {
		t.Errorf("home conceded %+v", r)
	}
	home, away := s.HeadToHead(3, 0)
	if len(home) != 1 || home[0].For != 2 || home[0].Against != 1 || len(away) != 0 {
		t.Errorf("head to head %+v %+v", home, away)
	}
	if e := s.MatchdayEvent(0, 1); e != 0 {
		t.Errorf("matchday 1 at event %d", e)
	}
}
