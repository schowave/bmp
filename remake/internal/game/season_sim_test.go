package game

import "testing"

// seasonResult is where manager 0 ended a season of his league, and the
// game at that point (his squad, money, ...).
type seasonResult struct {
	Place, Points, GoalDiff int
	S                       *State
}

// playSeason plays the first season of a new game with one manager at the
// club team (NewManager.Team) and level 2 to the last matchday of his
// league: turn, if not nil, acts before every turn (a lineup strategy, a
// transfer), matches run without substitutions, expiring contracts end. For
// experiments with the rules: compare strategies over several seeds, the
// game is deterministic for a seed.
func playSeason(t *testing.T, team int, seed uint32, turn func(s *State)) seasonResult {
	t.Helper()
	s, err := NewGame(mana(t), Options{Managers: []NewManager{{Name: "T", Team: team}}, Level: 2, Seed: seed})
	if err != nil {
		t.Fatal(err)
	}
	s.Start()
	l := s.Managers[0].League
	for steps := 0; s.Matchday[l] <= MatchdaysPerLeague[l]; steps++ {
		if steps > 100000 {
			t.Fatalf("season not over, phase %d", s.Loop.Phase)
		}
		switch s.Loop.Phase {
		case PhaseTurn:
			if turn != nil {
				turn(s)
			}
			s.EndTurn()
		case PhaseMatches:
			for md := s.NextMatchDay(); md != nil; md = s.NextMatchDay() {
				md.Run()
			}
		case PhaseContracts:
			s.ContractsDone()
		default:
			t.Fatalf("phase %d", s.Loop.Phase)
		}
	}
	mt := s.Managers[0].Team
	for i, r := range s.LeagueTable(l, TableTotal) {
		if r.Club == mt {
			return seasonResult{Place: i + 1, Points: r.Points, GoalDiff: r.Goals - r.Against, S: s}
		}
	}
	t.Fatalf("club %d not in the table of league %d", mt, l)
	return seasonResult{}
}

// Pressing o before every match day earns more points in a season than the
// automatic lineup (a simulation of 36 seasons gave about 11 more).
func TestOptimalLineupWinsPoints(t *testing.T) {
	if testing.Short() {
		t.Skip("long")
	}
	auto, opt := 0, 0
	for seed := uint32(1); seed <= 2; seed++ {
		auto += playSeason(t, 9, seed, nil).Points
		opt += playSeason(t, 9, seed, func(s *State) { s.OptimizeLineup(0) }).Points
	}
	if opt <= auto {
		t.Errorf("points with the optimal lineup %d, with the automatic one %d", opt, auto)
	}
	t.Logf("points in 2 seasons: automatic %d, optimal %d", auto, opt)
}

// Within the winter break the loop queues the flag WINTERPAUSE once for a run
// of days, outside it none; the season end queues SAISONENDE.
func TestBanners(t *testing.T) {
	s := lineupState(t)
	s.Banners = nil
	s.Day = 100
	s.afterEvent()
	if len(s.Banners) != 0 {
		t.Fatalf("banners before the winter break: %v", s.Banners)
	}
	s.Day = 140
	s.afterEvent()
	s.afterEvent()
	if b, ok := s.TakeBanner(); !ok || b != "WINTERPAUSE" || len(s.Banners) != 0 {
		t.Fatalf("winter break: %q %v, left %v", b, ok, s.Banners)
	}
	s.seasonEnd1()
	if b, ok := s.TakeBanner(); !ok || b != "SAISONENDE" {
		t.Fatalf("season end: %q %v", b, ok)
	}
}
