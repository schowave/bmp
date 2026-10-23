package game

import "testing"

// The optimal training changes nothing but the plan of the manager, keeps
// the limits of the training screen and is no worse than the plan before.
func TestOptimizeTraining(t *testing.T) {
	s := lineupState(t)
	before := snap(s)
	squads := s.Squads
	old := s.PlanOf(0)
	res := s.OptimizeTraining(0)
	if got := snap(s); got != before {
		t.Error("generator, teams or effort changed")
	}
	if s.Squads != squads {
		t.Error("a squad changed")
	}
	p := s.PlanOf(0)
	if p != res.Plan {
		t.Errorf("plan %+v, result %+v", p, res.Plan)
	}
	sum := func(v [4]int, lo, hi int) int {
		n := 0
		for _, x := range v {
			if x < lo || x > hi {
				t.Errorf("value %d of %v outside %d..%d", x, v, lo, hi)
			}
			n += x
		}
		return n
	}
	if sum(p.Training, 1, 10) > 20 || sum(p.PosTraining, 1, 5) > 10 || p.Intensity < 1 || p.Intensity > 10 {
		t.Errorf("plan %+v outside the screen", p)
	}
	if res.After < res.Before || res.Before != s.trainingValue(0, old, fullRuns, 0) {
		t.Errorf("before %d (plan %+v), after %d", res.Before, old, res.After)
	}
	t.Logf("plan %+v: strength %d before, %d after", p, res.Before, res.After)
}

// trainingSplits are the splits of the 20 points with 1..10 each.
func TestTrainingSplits(t *testing.T) {
	seen := map[[4]int]bool{}
	for _, tr := range trainingSplits() {
		if tr[0]+tr[1]+tr[2]+tr[3] != 20 || seen[tr] {
			t.Fatalf("split %v", tr)
		}
		for _, x := range tr {
			if x < 1 || x > 10 {
				t.Fatalf("split %v", tr)
			}
		}
		seen[tr] = true
	}
	if len(seen) != 633 {
		t.Errorf("%d splits, want 633", len(seen))
	}
	for _, p := range positionMoves([4]int{1, 3, 3, 3}) {
		if s := p[0] + p[1] + p[2] + p[3]; s > 10 {
			t.Errorf("move %v has %d points", p, s)
		}
	}
}

// With the optimal training a season ends with a stronger eleven than with
// the training a new game starts with.
func TestOptimalTrainingStrengthensTheEleven(t *testing.T) {
	if testing.Short() {
		t.Skip("long")
	}
	plain := playSeason(t, 9, 1, func(s *State) { s.OptimizeLineup(0) })
	n := 0
	opt := playSeason(t, 9, 1, func(s *State) {
		if n%10 == 0 {
			s.OptimizeTraining(0)
		}
		n++
		s.OptimizeLineup(0)
	})
	a, b := plain.S.elevenStrength(0), opt.S.elevenStrength(0)
	if b <= a {
		t.Errorf("eleven strength %.1f with the optimal training, %.1f without", b, a)
	}
	t.Logf("eleven strength at the end: %.1f, with the optimal training %.1f; points %d and %d", a, b, plain.Points, opt.Points)
}

// The optimal ticket price earns the most expected gate money of all prices
// at the next home match and leaves the generator alone.
func TestOptimizeTicketPrice(t *testing.T) {
	s := lineupState(t)
	r := s.Rand
	res := s.OptimizeTicketPrice(0)
	if s.Rand != r {
		t.Error("the generator moved")
	}
	if res.Opponent < 0 {
		t.Fatal("no home match found")
	}
	_, imp, _ := s.nextHomeMatch(0)
	for p := 5; p <= 25; p++ {
		if g := s.expectedGate(0, res.Opponent, imp, p); g > res.AfterGate {
			t.Errorf("price %d earns %d, more than %d at %d", p, g, res.AfterGate, res.After)
		}
	}
	if s.Managers[0].TicketPrice != res.After || res.AfterGate < res.BeforeGate {
		t.Errorf("result %+v, price %d", res, s.Managers[0].TicketPrice)
	}
	t.Logf("%+v", res)
}
