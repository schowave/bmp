package game

// The optimal lineup is not a rule of the original: it rates lineups with
// the calculation of the original (lineupValues, teamStrengthAt, see
// lineupmodel.go) and looks for the best one. It changes nothing but the
// lineup of the manager; the generator of the game stays where it is.

// LineupRating is the rating of the lineup of manager m (lineupmodel.go): the
// expected points of a match against his next opponent, or against the clubs
// of his league without one, in hundredths of a point. Only used to compare
// lineups.
func (s *State) LineupRating(m int) int { return s.newRater(m).rate() }

// OptimalLineup is the result of OptimizeLineup: the rating of the lineup
// before, of the automatic lineups of the three formations (Formations) and
// of the lineup chosen.
type OptimalLineup struct {
	Before int
	Auto   [3]int
	After  int
	// Opponent is the club the lineup was rated against, -1 for the clubs of
	// the league, and Home tells whether the match is a home match.
	Opponent int
	Home     bool
	// Formation is the formation the lineup started from; Manual tells that
	// the lineup differs from the automatic one of it and so stays manual.
	Formation int
	Manual    bool
}

// OptimizeLineup sets the best lineup it finds for manager m: it starts with
// the automatic lineup of every formation, improves each by swapping two
// players (their numbers and cells) as long as that raises the rating by
// LineupRating, and takes the best. A swap may bring in a rested player from the
// bench or put a player on his side or into his row; a tired player rests
// unless he is worth it (tiredFatigue). Players who are
// injured or suspended stay out. The lineup stays automatic if the
// automatic lineup of the formation could not be improved, else it becomes
// manual; if the lineup the manager had is better than all of them, it is
// kept.
func (s *State) OptimizeLineup(m int) OptimalLineup {
	mg := s.Managers[m]
	lr := s.newRater(m)
	res := OptimalLineup{Before: lr.rate(), Opponent: lr.opp, Home: len(lr.venues) == 1 && lr.venues[0]}
	before := lr.score()
	mode := mg.AutoLineup
	sq := &s.Squads[m]
	orig := *sq

	// Every formation is improved; the best score wins, the earlier
	// formation on a tie.
	var autos, tuned [3][SquadSlots]Slot
	var after [3]int
	best := -1
	for f := range Formations {
		*sq = orig
		mg.AutoLineup = f + 2
		s.AutoLineup(m)
		autos[f] = *sq
		res.Auto[f] = lr.rate()
		after[f] = s.improveLineup(lr, lr.score())
		tuned[f] = *sq
		if best < 0 || after[f] > after[best] {
			best = f
		}
	}
	res.Formation = best
	*sq = tuned[best]
	res.Manual = tuned[best] != autos[best]

	if after[best] < before && s.playable(orig[:]) {
		// Never worse than what the manager had, if that could be played.
		*sq = orig
		mg.AutoLineup = mode
		res.After, res.Manual = res.Before, false
		return res
	}
	res.After = lr.rate()
	if res.Manual {
		mg.AutoLineup = 1
	} else {
		mg.AutoLineup = best + 2
	}
	return res
}

// tiredFatigue and tiredCost are the rule of the load: a player with an
// Erschöpfung of tiredFatigue or more plays only if he is worth tiredCost
// hundredths of a point more than who would play for him. A match adds 7 to
// 10 to the Erschöpfung of a player in the team, and above 135 his training
// does little (develop) and an injury in training gets likelier, so a tired
// player rests unless he makes a difference. The rule is the remake's, not
// the engine's: the rating of a lineup does not count it.
const (
	tiredFatigue = 130
	tiredCost    = 5
)

// score is what the search maximizes: the rating less tiredCost for every
// tired player in the team.
func (lr *lineupRater) score() int {
	x := lr.rate()
	for i := 0; i < SquadMax; i++ {
		sl := &lr.s.Squads[lr.m][i]
		if !sl.Empty() && sl.Lineup >= 1 && sl.Lineup <= 11 && sl.Fatigue >= tiredFatigue {
			x -= tiredCost
		}
	}
	return x
}

// lineupMinGain is what a swap or a move has to raise the rating by, in
// hundredths of a point per match: the model is no more exact than that, and
// a change for less is as likely a mistake of the model as a gain (it once
// swapped the goalkeepers for 1).
const lineupMinGain = 3

// improveLineup changes the lineup while its score (lineupRater.score) rises
// and returns the score reached: it swaps two available players, of the team or of the
// bench (their numbers and cells), and moves a player of the team to a free
// cell of the board within his line (defence, midfield, attack). A move to
// another line would also be rated by the calculation of the original, but
// it rewards what no manager would field: the bonus of the balance of the
// lines (lineupValues, bal) grows with the players in the midfield rows, and
// the search piled up the team there. The first change that helps is taken, so the result
// depends on nothing but the squad. It stops at a lineup where no single
// swap or move helps.
func (s *State) improveLineup(lr *lineupRater, rating int) int {
	sq := &s.Squads[lr.m]
	swap := func(a, b *Slot) {
		a.Lineup, b.Lineup = b.Lineup, a.Lineup
		a.Row, b.Row = b.Row, a.Row
		a.Col, b.Col = b.Col, a.Col
	}
	var taken [BoardCols][BoardRows]bool
	for improved := true; improved; {
		improved = false
		for a := 0; a < SquadMax; a++ {
			sa := &sq[a]
			if sa.Empty() || sa.Lineup < 1 || sa.Lineup > 11 {
				continue
			}
			for b := 0; b < SquadMax; b++ {
				sb := &sq[b]
				if b == a || sb.Empty() || !s.available(sb) {
					continue
				}
				inTeam := sb.Lineup >= 1 && sb.Lineup <= 11
				if inTeam && b < a {
					continue
				}
				swap(sa, sb)
				if !s.fitsCell(sa) || !s.fitsCell(sb) {
					swap(sa, sb)
					continue
				}
				if r := lr.score(); r >= rating+lineupMinGain {
					rating, improved = r, true
					break
				}
				swap(sa, sb)
			}
		}
		taken = [BoardCols][BoardRows]bool{}
		for i := 0; i < SquadMax; i++ {
			if sl := &sq[i]; !sl.Empty() && sl.Lineup >= 1 && sl.Lineup <= 11 {
				taken[sl.Col][sl.Row] = true
			}
		}
		for a := 0; a < SquadMax; a++ {
			sa := &sq[a]
			if sa.Empty() || sa.Lineup < 1 || sa.Lineup > 11 {
				continue
			}
			col, row := sa.Col, sa.Row
			// The best cell wins, of equal ones the nearest, so that a
			// player is moved only as far as it helps.
			floor := rating + lineupMinGain - 1
			bestC, bestR, best, bestDist := col, row, floor, 0
			for c := 0; c < BoardCols; c++ {
				for r := 0; r < BoardRows; r++ {
					if taken[c][r] || lineOfRow[7-r] != lineOfRow[7-row] {
						continue
					}
					sa.Col, sa.Row = c, r
					if !s.fitsCell(sa) {
						continue
					}
					x, dist := lr.score(), abs(c-col)+abs(r-row)
					if x > best || (x == best && x > floor && dist < bestDist) {
						bestC, bestR, best, bestDist = c, r, x, dist
					}
				}
			}
			sa.Col, sa.Row = bestC, bestR
			if best > floor {
				rating, improved = best, true
				taken[col][row], taken[bestC][bestR] = false, true
			}
		}
	}
	return rating
}

// playable tells whether the team of the squad has 11 players who can play.
func (s *State) playable(sq []Slot) bool {
	n := 0
	for i := range sq {
		sl := &sq[i]
		if sl.Empty() || sl.Lineup < 1 || sl.Lineup > 11 {
			continue
		}
		if !s.available(sl) {
			return false
		}
		n++
	}
	return n == 11
}

// fitsCell tells whether the player of sl may stand on his cell if he is in
// the team: the goalkeeper in the goal row and nobody else there, and a field
// player in his own line or next to it (a defender in the midfield, a
// forward in the midfield, but no defender up front). The calculation of the
// original punishes a wrong position so little that the search would
// otherwise field a goalkeeper as a striker.
func (s *State) fitsCell(sl *Slot) bool {
	if sl.Lineup < 1 || sl.Lineup > 11 {
		return true
	}
	pos := s.Persons[sl.Person].Position()
	if pos == PosKeeper || sl.Row == 7 {
		return pos == PosKeeper && sl.Row == 7
	}
	return abs(pos-(lineOfRow[7-sl.Row]+1)) <= 1
}
