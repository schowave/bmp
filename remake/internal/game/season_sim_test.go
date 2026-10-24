package game

import (
	"slices"
	"testing"
)

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
func TestFlags(t *testing.T) {
	s := lineupState(t)
	flags := func() []string {
		var out []string
		for _, n := range s.TakeNotices() {
			if n.Kind == NoticeFlag {
				out = append(out, n.Title)
			}
		}
		return out
	}
	s.TakeNotices()
	s.Day = 100
	s.afterEvent()
	if f := flags(); len(f) != 0 {
		t.Fatalf("flags before the winter break: %v", f)
	}
	s.Day = 140
	s.afterEvent()
	s.afterEvent()
	if f := flags(); !slices.Equal(f, []string{"WINTERPAUSE"}) {
		t.Fatalf("winter break: %v", f)
	}
	s.seasonEnd1()
	if f := flags(); len(f) == 0 || f[0] != "SAISONENDE" {
		t.Fatalf("season end: %v", f)
	}
}

// The original shows the championship of the last manager in his history
// step, before SAISONENDE (f_1a58_3176 @3945, then f_1a58_4597 @3c45), and
// the season report after it (@43b5).
func TestSeasonEndOrder(t *testing.T) {
	s := lineupState(t)
	m := s.Managers[0]
	m.League = 0
	s.Clubs[m.Team].Pos = 0
	s.EndYear = 0
	s.Loop.Phase, s.Loop.Current, s.Loop.SeasonEnd = PhaseTurn, 0, true
	s.TakeNotices()
	s.EndTurn()
	var kinds []NoticeKind
	for _, n := range s.TakeNotices() {
		if n.Kind == NoticeTrophy || n.Kind == NoticeFlag || n.Kind == NoticeSeasonEnd {
			kinds = append(kinds, n.Kind)
		}
	}
	if want := []NoticeKind{NoticeTrophy, NoticeFlag, NoticeSeasonEnd}; !slices.Equal(kinds, want) {
		t.Errorf("season end shows %v, want %v", kinds, want)
	}
}
