package game

import "math"

// The optimal training is not a rule of the original: it plays the rest of
// the season with the training of the original (develop, trainingInjury,
// playFatigue) on copies of the game, for every setting of the training
// screen it tries, and keeps the one whose best eleven are strongest on the
// average. It changes nothing but the settings of the manager; the generator
// of the game stays where it is.

// TrainingPlan is a setting of the training screen: the points of
// Kondition, Spiel, Schuss and Taktik (1..10 each, at most 20), of Tor,
// Abwehr, Mittelfeld and Angriff (1..5 each, at most 10) and the intensity
// (1..10).
type TrainingPlan struct {
	Training    [4]int
	PosTraining [4]int
	Intensity   int
}

// OptimalTraining is the result of OptimizeTraining: the strength of the
// best eleven with the plan before and with the plan chosen, in tenths of a
// point of strength ((Kondition + Technik + Form) / 3, less the Erschöpfung
// as the lineup counts it), the mean over the rest of the season.
type OptimalTraining struct {
	Before, After int
	Plan          TrainingPlan
}

const (
	trainingSeed = 20240401
	// The screening of the training points runs a few series over a part of
	// the season, the choice of the intensity and of the positions more of
	// them over all of it.
	screenRuns, screenSteps = 2, 30
	fullRuns                = 6
	screenKeep              = 5 // settings of the screening that go on
)

// PlanOf returns the training plan of manager m.
func (s *State) PlanOf(m int) TrainingPlan {
	mg := s.Managers[m]
	return TrainingPlan{Training: mg.Training, PosTraining: mg.PosTraining, Intensity: mg.Intensity}
}

// OptimizeTraining sets the training plan that gives manager m the strongest
// best eleven over the rest of the season: it screens every split of the 20
// training points at intensity 6 and 10, then tries every intensity for the
// best splits, then moves single points between the positions while that
// helps. The two intensities of the screening are the two good ones of the
// original: from about 8 the Erschöpfung of the players runs up to 150, and
// above 135 develop cuts the training by more than the intensity adds.
func (s *State) OptimizeTraining(m int) OptimalTraining {
	before := s.PlanOf(m)
	res := OptimalTraining{Before: s.trainingValue(m, before, fullRuns, 0)}

	type scored struct {
		p TrainingPlan
		v int
	}
	var screen []scored
	for _, in := range []int{6, 10} {
		for _, tr := range trainingSplits() {
			p := TrainingPlan{Training: tr, PosTraining: before.PosTraining, Intensity: in}
			screen = append(screen, scored{p, s.trainingValue(m, p, screenRuns, screenSteps)})
		}
	}
	// The best splits, each once.
	var top []TrainingPlan
	for len(top) < screenKeep && len(screen) > 0 {
		bi := 0
		for i, c := range screen {
			if c.v > screen[bi].v {
				bi = i
			}
		}
		b := screen[bi]
		dup := false
		for _, p := range top {
			dup = dup || p.Training == b.p.Training
		}
		if !dup {
			top = append(top, b.p)
		}
		screen = append(screen[:bi], screen[bi+1:]...)
	}

	best, bestV := before, res.Before
	for _, p := range top {
		for in := 1; in <= 10; in++ {
			p.Intensity = in
			if v := s.trainingValue(m, p, fullRuns, 0); v > bestV {
				best, bestV = p, v
			}
		}
	}
	for improved := true; improved; {
		improved = false
		for _, pos := range positionMoves(best.PosTraining) {
			p := best
			p.PosTraining = pos
			if v := s.trainingValue(m, p, fullRuns, 0); v > bestV {
				best, bestV, improved = p, v, true
			}
		}
	}
	res.Plan, res.After = best, bestV
	mg := s.Managers[m]
	mg.Training, mg.PosTraining, mg.Intensity = best.Training, best.PosTraining, best.Intensity
	return res
}

// trainingSplits returns every split of the 20 training points into
// Kondition, Spiel, Schuss and Taktik, 1..10 each: the screen keeps each
// row at 1 or more, and fewer than 20 points only waste training.
func trainingSplits() [][4]int {
	var out [][4]int
	for a := 1; a <= 10; a++ {
		for b := 1; b <= 10; b++ {
			for c := 1; c <= 10; c++ {
				if d := 20 - a - b - c; d >= 1 && d <= 10 {
					out = append(out, [4]int{a, b, c, d})
				}
			}
		}
	}
	return out
}

// positionMoves returns the settings of the position points one point away
// from pos: a point moved from one position to another, or one of the 10
// added; each position keeps 1..5.
func positionMoves(pos [4]int) [][4]int {
	var out [][4]int
	sum := pos[0] + pos[1] + pos[2] + pos[3]
	for to := 0; to < 4; to++ {
		if pos[to] >= 5 {
			continue
		}
		if sum < 10 {
			p := pos
			p[to]++
			out = append(out, p)
		}
		for from := 0; from < 4; from++ {
			if from != to && pos[from] > 1 {
				p := pos
				p[from]--
				p[to]++
				out = append(out, p)
			}
		}
	}
	return out
}

// trainingValue plays the rest of the season runs times with plan p on
// copies of the game and returns the mean strength of the best eleven over
// its turn days, in tenths (bestEleven). A turn day brings the match of the
// event before it (playFatigue for the eleven), the injuries in training
// and develop, as afterEvent and turn do; the market, the money and the
// lineup screen are left out. steps > 0 stops after that many turn days.
func (s *State) trainingValue(m int, p TrainingPlan, runs, steps int) int {
	l := s.Managers[m].League
	sum := 0.0
	for r := 0; r < runs; r++ {
		c := s.trainingCopy(m)
		c.Rand = Rand{Seed: trainingSeed + uint32(r)}
		mg := c.Managers[m]
		mg.Training, mg.PosTraining, mg.Intensity = p.Training, p.PosTraining, p.Intensity
		total, n := 0.0, 0
		for e := c.Event; e < len(c.Events) && (steps == 0 || n < steps); e++ {
			match := c.Events[e]&(1<<l) != 0
			if match {
				for _, i := range c.bestEleven(m, true) {
					c.playFatigue(&c.Squads[m][i])
				}
			}
			for c.Day++; c.Day%7 != 0 && (c.Day-4)%7 != 0; c.Day++ {
			}
			if c.Day > LastSeasonDay {
				break
			}
			sq := &c.Squads[m]
			for i := 0; i < c.countLineup(m, 0); i++ {
				c.trainingInjury(&sq[i], !match)
			}
			c.develop(m)
			total += c.elevenStrength(m)
			n++
		}
		if n > 0 {
			sum += total / float64(n)
		}
	}
	return int(math.Round(sum / float64(runs) * 10))
}

// trainingCopy returns a copy of the game in which the training of manager
// m can run: the arrays are copied with the State, the managers anew.
func (s *State) trainingCopy(m int) *State {
	c := *s
	// The notices of the copy are not shown.
	c.Notices = nil
	c.Managers = make([]*Manager, len(s.Managers))
	for i, mg := range s.Managers {
		cm := *mg
		c.Managers[i] = &cm
	}
	c.Managers[m].AutoLineup = 2
	return &c
}

// playerStrength is what a player gives the strength of his line:
// Kondition, Technik and Form, less the Erschöpfung as lineupValues counts
// it.
func playerStrength(sl *Slot) float64 {
	return float64(sl.Skill[0]+sl.Skill[1]+sl.Skill[2])/3 - float64(sl.Fatigue)/60
}

// restStrength is what a tired player (Erschöpfung tiredFatigue or more)
// counts less when the eleven of a match are picked, as the optimal lineup
// rests him unless he is worth it.
const restStrength = 5

// bestEleven returns the slots of the best goalkeeper and the ten best field
// players who can play. With rest a tired player counts restStrength lower.
func (s *State) bestEleven(m int, rest bool) []int {
	value := func(sl *Slot) float64 {
		v := playerStrength(sl)
		if rest && sl.Fatigue >= tiredFatigue {
			v -= restStrength
		}
		return v
	}
	keeper, field := -1, []int{}
	for i := 0; i < SquadMax; i++ {
		sl := &s.Squads[m][i]
		if sl.Empty() || !s.available(sl) {
			continue
		}
		if s.Persons[sl.Person].Position() == PosKeeper {
			if keeper < 0 || value(sl) > value(&s.Squads[m][keeper]) {
				keeper = i
			}
			continue
		}
		field = append(field, i)
	}
	for i := 1; i < len(field); i++ {
		for j := i; j > 0 && value(&s.Squads[m][field[j]]) > value(&s.Squads[m][field[j-1]]); j-- {
			field[j], field[j-1] = field[j-1], field[j]
		}
	}
	if len(field) > 10 {
		field = field[:10]
	}
	if keeper >= 0 {
		field = append(field, keeper)
	}
	return field
}

// elevenStrength is the mean strength of the best eleven (playerStrength);
// a missing player counts 0.
func (s *State) elevenStrength(m int) float64 {
	sum := 0.0
	for _, i := range s.bestEleven(m, false) {
		sum += playerStrength(&s.Squads[m][i])
	}
	return sum / 11
}
