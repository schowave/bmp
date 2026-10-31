package game

import "math"

// The load of a player is what the optimal lineup counts against letting
// him play today. A match adds 9 to his Erschöpfung (playFatigue, less one
// if his Kondition beats his Technik), a turn day of training takes about 3
// off at the intensity of the start (develop), and above loadLimit his
// training does little and an injury in training gets likelier. The rule is
// the remake's, not the engine's: the rating of a lineup does not count it.
// It plays the next turn days on twice, with the means of the dice: once
// with the player in the next match and once without, and in every match
// after it with him in both. What playing costs is how far the first runs
// above loadLimit more than the second, so that a short week, a match before
// the winter break or the last match of the season weigh as they should.

const (
	// loadLimit is the Erschöpfung above which develop cuts the training.
	loadLimit = 135
	// loadTurns is how many turn days the load looks ahead (four weeks).
	loadTurns = 8
	// loadCost is what a point of Erschöpfung above loadLimit on a turn
	// day costs, in hundredths of a point of a match: with a match a week
	// and the intensity of the start, a player at 130 who plays costs 7,
	// one at 135 costs 12, and in a week with two matches 12 and 14 (a
	// player whose Kondition beats his Technik 3, 10, 11 and 12).
	loadCost = 0.2
)

// loadCosts returns the load of every player of the squad of manager m, in
// hundredths of a point: 0 for one who is not tired.
func (s *State) loadCosts(m int) (costs [SquadMax]int) {
	mg := s.Managers[m]
	turns := s.loadTurns(m)
	for i := 0; i < SquadMax; i++ {
		sl := &s.Squads[m][i]
		if sl.Empty() {
			continue
		}
		p := &s.Persons[sl.Person]
		add := float64(9 - boolInt(p.Skill[0] > p.Skill[1]))
		play, rest := float64(sl.Fatigue), float64(sl.Fatigue)
		excess, first := 0.0, true
		for _, t := range turns {
			if t.match {
				play += add
				if !first {
					rest += add
				}
				first = false
			}
			play = s.trainingLoad(play, mg.Intensity, t.day)
			rest = s.trainingLoad(rest, mg.Intensity, t.day)
			excess += max(play-loadLimit, 0) - max(rest-loadLimit, 0)
		}
		costs[i] = int(math.Round(excess * loadCost))
	}
	return costs
}

// loadTurn is a turn day the load looks at: whether the event before it
// brings a match and the day of the training.
type loadTurn struct {
	match bool
	day   int
}

// loadTurns returns the next turn days of manager m up to loadTurns from the
// one of the next match on: the event of today if it brings him a match,
// else the next with a match of his league (the cups to come are not known).
// The turn days before that match count too, for the training they bring.
func (s *State) loadTurns(m int) []loadTurn {
	l := s.Managers[m].League
	_, _, today := s.NextOpponent(m)
	var turns []loadTurn
	day, n := s.Day, 0
	for e := s.Event; e < len(s.Events) && n < loadTurns; e++ {
		match := s.Events[e]&(1<<l) != 0 || (e == s.Event && today)
		if day = nextTurnDay(day); day > LastSeasonDay {
			break
		}
		if match || n > 0 {
			n++
		}
		turns = append(turns, loadTurn{match: match, day: day})
	}
	return turns
}

// trainingLoad is the Erschöpfung f after the training of a turn day on
// day with intensity in (develop), with the means of the dice and without
// the upper bound: in the winter break the intensity is at least 6 and, but
// on level 2, the training of each of the three skills adds 0 to 2.
func (s *State) trainingLoad(f float64, in, day int) float64 {
	winter := winterDay(day)
	if winter && in < 6 {
		in = 6
	}
	if f > 50 {
		f -= 5
	}
	f += float64(in-5) * 2
	if f > 130 && in < 6 {
		f -= 2
	}
	if winter && s.Level != 2 {
		f += 3
	}
	return math.Max(60, f)
}
