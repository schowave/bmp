package game

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/schowave/bmp/remake/internal/assets"
)

func mana(t *testing.T) *assets.Mana {
	t.Helper()
	data, err := os.ReadFile(filepath.Join("..", "..", "..", "bmp", "MANA.DAT"))
	if err != nil {
		t.Skip("MANA.DAT not found")
	}
	m, err := assets.ParseMana(data)
	if err != nil {
		t.Fatal(err)
	}
	return m
}

// autoplay plays until the given number of seasons is complete, ending
// every turn at once and letting expiring contracts run out.
func autoplay(t *testing.T, s *State, seasons int) {
	t.Helper()
	start := s.Seasons
	for steps := 0; s.Seasons-start < seasons; steps++ {
		if steps > 100000 {
			t.Fatal("no progress")
		}
		switch s.Loop.Phase {
		case PhaseTurn:
			for mi := range s.Managers {
				s.TakeInbox(mi)
			}
			s.EndTurn()
		case PhaseMatches:
			for md := s.NextMatchDay(); md != nil; md = s.NextMatchDay() {
				md.Run()
			}
		case PhaseContracts:
			for _, si := range s.ExpiringContracts() {
				_ = s.NegotiateContract(s.Loop.Current, si, 2, s.ContractHint(s.Loop.Current, si, 2)*2, 0)
			}
			s.ContractsDone()
		case PhaseGameOver:
			return
		}
	}
}

func TestSeasonSimulation(t *testing.T) {
	m := mana(t)
	s, err := NewGame(m, Options{Managers: []NewManager{{Name: "TEST", Team: 9}}, Level: 2, Seed: 1})
	if err != nil {
		t.Fatal(err)
	}
	mg := s.Managers[0]
	if mg.League != 2 || LeagueOf(mg.Team) != 2 {
		t.Fatalf("endless game should start in the Amateur-Oberliga: league %d team %d", mg.League, mg.Team)
	}
	if s.Teams[mg.Team].Name != "BORUSSIA DORTMUND" {
		t.Fatalf("manager team %q", s.Teams[mg.Team].Name)
	}
	if n := s.SquadSize(0); n != 20 {
		t.Fatalf("squad size %d", n)
	}
	if mg.Stadium[StStanding] != 12000 || mg.Stadium[StSeats] != 0 || mg.TicketPrice != 10 || mg.Money != 1500000 {
		t.Fatalf("start values: %+v", mg.Stadium)
	}
	s.Start()
	// Play the first season and check the tables at its end, before the
	// season end resets them.
	for s.Loop.Phase != PhaseTurn || !s.Loop.SeasonEnd {
		switch s.Loop.Phase {
		case PhaseTurn:
			s.TakeInbox(0)
			s.EndTurn()
		case PhaseMatches:
			for md := s.NextMatchDay(); md != nil; md = s.NextMatchDay() {
				md.Run()
			}
		default:
			t.Fatalf("phase %d", s.Loop.Phase)
		}
	}
	for l := 0; l < NumLeagues; l++ {
		for i := 0; i < TeamsPerLeague[l]; i++ {
			c := &s.Clubs[s.Tables[l][i]]
			if c.TotalGames() != MatchdaysPerLeague[l] {
				t.Errorf("league %d club %s: %d games", l, s.Teams[s.Tables[l][i]].Name, c.TotalGames())
			}
			if i > 0 && s.Clubs[s.Tables[l][i-1]].TotalPoints() < c.TotalPoints() {
				t.Errorf("league %d not sorted at %d", l, i)
			}
		}
	}
	if s.Cups.Holder[CupDFB] == 0 {
		t.Error("no DFB cup winner")
	}
	t.Logf("champion %s, DFB %s, money %d", s.Teams[s.Tables[0][0]].Name, s.Teams[s.Cups.Holder[CupDFB]-1].Name, mg.Money)
	autoplay(t, s, 3)
	t.Logf("after 3 more seasons: league %d, money %d, squad %d", mg.League, mg.Money, s.SquadSize(0))
	if s.SquadSize(0) == 0 {
		t.Error("squad empty")
	}
}

func TestOneYearGameEnds(t *testing.T) {
	m := mana(t)
	s, err := NewGame(m, Options{Managers: []NewManager{{Name: "A", Team: 0}, {Name: "B", Team: 1}},
		Length: OneYear, Seed: 7})
	if err != nil {
		t.Fatal(err)
	}
	for _, mg := range s.Managers {
		if mg.League != 0 {
			t.Fatalf("1-year game should start in the Bundesliga")
		}
	}
	s.Start()
	autoplay(t, s, 2)
	if s.Loop.Phase != PhaseGameOver {
		t.Fatalf("game not over: phase %d seasons %d", s.Loop.Phase, s.Seasons)
	}
}
