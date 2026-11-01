package game

import "sort"

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
	// Cup tells that the match is one of a cup: the lineup is rated by the
	// chance to go through, and nobody rests for his load.
	Cup bool
	// Formation is the formation of the automatic lineup the lineup equals,
	// or the one it started from; Manual tells that the lineup differs from
	// the automatic one of every formation and so stays manual.
	Formation int
	Manual    bool
	// Changes are the changes against the lineup before, the most worth
	// first; none when the lineup before stays.
	Changes []LineupChange
	// Load is what playing today costs every player of the squad (loadCosts).
	Load [SquadMax]int
	// Reasons say for every player of the squad why he plays or not.
	Reasons [SquadMax]PlayerReason
}

// PlayerReason is why a player plays in the lineup chosen or not: Alt is the
// best other player for his place, Margin the score (lineupRater.score) of
// the lineup with him less the score with Alt instead, in hundredths of a
// point. For a player of the team Alt is the best one who could come in for
// him, for one outside the team the player of the team he would best
// replace. Alt is -1 when there is none: nobody fits his place, or he cannot
// play.
type PlayerReason struct {
	Alt, Margin int
}

// LineupChange is one change of the optimal lineup against the lineup
// before: In plays for Out on a cell of the board, or In stayed in the team
// and Moved to another cell, With the player of the team he swapped cells
// with (else -1). In without Out came into a team that had no 11 players who
// could play, Out without In (In -1) left one.
type LineupChange struct {
	In, Out, With int
	Moved         bool
	// Gain is what the change is worth: the score of the lineup
	// (lineupRater.score) less the score with the change undone, in
	// hundredths of a point. Rated is false, and Gain 0, when the change
	// cannot be undone alone: Out cannot play, or not on the cell of In
	// (a forward for a defender when the formation changed), or nobody left
	// for In.
	Gain  int
	Rated bool
}

// OptimizeLineup sets the best lineup it finds for manager m. The search
// starts from the automatic lineup of every formation and from the lineup
// the manager has, improves each (improveLineup) and takes the best. A swap
// may bring in a rested player from the bench or put a player on his side
// or into his row; a tired player rests unless he is worth what his load
// costs (loadCosts). Players who are injured or suspended stay out. The
// lineup stays automatic if it equals the automatic lineup of a formation,
// else it becomes manual; if the lineup the manager had is not beaten by
// LineupMinGain, it is kept.
func (s *State) OptimizeLineup(m int) OptimalLineup {
	mg := s.Managers[m]
	lr := s.newRater(m)
	res := OptimalLineup{Before: lr.rate(), Opponent: lr.opp, Home: len(lr.venues) == 1 && lr.venues[0], Cup: lr.cup, Load: lr.load}
	before := lr.score()
	mode := mg.AutoLineup
	sq := &s.Squads[m]
	orig := *sq

	// Every start is improved; the best score wins, the earlier start on a
	// tie. The starts are the formations, then the lineup of the manager.
	var autos [3][SquadSlots]Slot
	var tuned [4][SquadSlots]Slot
	after := [4]int{}
	best := -1
	for f := 0; f < 4; f++ {
		*sq = orig
		if f < 3 {
			mg.AutoLineup = f + 2
			s.AutoLineup(m)
			autos[f] = *sq
			res.Auto[f] = lr.rate()
		} else if !s.playable(orig[:]) {
			break
		}
		after[f] = s.improveLineup(lr, lr.score())
		tuned[f] = *sq
		if best < 0 || after[f] > after[best] {
			best = f
		}
	}
	*sq = tuned[best]
	after[best] = s.kickLineup(lr, after[best])
	tuned[best] = *sq
	classify := func(start int) {
		res.Formation, res.Manual = start, true
		if start == 3 {
			res.Formation = max(mode-2, 0)
		}
		for f := range autos {
			if *sq == autos[f] {
				res.Formation, res.Manual = f, false
				break
			}
		}
	}
	classify(best)

	keep := after[best] < before+LineupMinGain && s.playable(orig[:])
	if keep {
		// A start that rates no worse but fields the stronger team is no
		// change for the sake of a gain: it takes the place of the
		// manager's lineup.
		if c := stronger(&orig, &tuned, after, before); c >= 0 {
			*sq = tuned[c]
			classify(c)
			after[best], keep = after[c], false
		}
	}
	if keep {
		// What the manager had stays unless the new lineup gains
		// LineupMinGain, as every single change must, if it can be played.
		*sq = orig
		mg.AutoLineup = mode
		res.After, res.Manual = res.Before, false
		if s.breakTies(lr) {
			res.After, res.Manual = lr.rate(), true
			mg.AutoLineup = 1
			res.Changes = s.lineupChanges(lr, &orig)
		}
		res.Reasons = s.lineupReasons(lr)
		return res
	}
	if s.breakTies(lr) {
		res.Manual = true
	}
	res.After = lr.rate()
	res.Changes = s.lineupChanges(lr, &orig)
	res.Reasons = s.lineupReasons(lr)
	if res.Manual {
		mg.AutoLineup = 1
	} else {
		mg.AutoLineup = res.Formation + 2
	}
	return res
}

// score is what the search maximizes: the rating less the load of every
// player in the team (loadCosts).
func (lr *lineupRater) score() int {
	x := lr.rate()
	for i := 0; i < SquadMax; i++ {
		if lr.s.Squads[lr.m][i].InTeam() {
			x -= lr.load[i]
		}
	}
	return x
}

// LineupMinGain is what a swap or a move has to raise the rating by, in
// hundredths of a point per match: the model is no more exact than that, and
// a change for less is as likely a mistake of the model as a gain (it once
// swapped the goalkeepers for 1). The screens count a difference below it
// as none.
const LineupMinGain = 3

// improveLineup changes the lineup while its score (lineupRater.score) rises
// and returns the score reached. Each step looks at every swap of two
// available players, of the team or of the bench (their numbers and cells),
// and at every move of a player of the team to a free cell of the board
// within his line (defence, midfield, attack), and makes the one that helps
// most (steepest ascent: the first that helps would decide by the order of
// the squad and stop sooner). A move to another line would also be rated by
// the calculation of the original, but it rewards what no manager would
// field: the bonus of the balance of the lines (lineupValues, bal) grows with
// the players in the midfield rows, and the search piled up the team there.
// It stops at a lineup where no single swap or move helps.
func (s *State) improveLineup(lr *lineupRater, rating int) int {
	sq := &s.Squads[lr.m]
	for {
		floor := rating + LineupMinGain - 1
		best, bestDist := floor, 0
		var step *lineupStep
		for a := 0; a < SquadMax; a++ {
			for b := 0; b < SquadMax; b++ {
				if !s.swappable(sq, a, b) {
					continue
				}
				swapPlaces(&sq[a], &sq[b])
				if s.fitsCell(&sq[a]) && s.fitsCell(&sq[b]) {
					if x := lr.score(); x > best {
						best, step = x, &lineupStep{a: a, b: b}
					}
				}
				swapPlaces(&sq[a], &sq[b])
			}
		}
		var taken [BoardCols][BoardRows]bool
		for i := 0; i < SquadMax; i++ {
			if sl := &sq[i]; sl.InTeam() {
				taken[sl.Col][sl.Row] = true
			}
		}
		for a := 0; a < SquadMax; a++ {
			sa := &sq[a]
			if !sa.InTeam() {
				continue
			}
			col, row := sa.Col, sa.Row
			for c := 0; c < BoardCols; c++ {
				for r := 0; r < BoardRows; r++ {
					if taken[c][r] || lineOfRow[7-r] != lineOfRow[7-row] {
						continue
					}
					sa.Col, sa.Row = c, r
					if !s.fitsCell(sa) {
						continue
					}
					// Of equal cells of a player the nearest, so that he is
					// moved only as far as it helps.
					x, dist := lr.score(), abs(c-col)+abs(r-row)
					if x > best || (x == best && step != nil && step.move && step.a == a && dist < bestDist) {
						best, bestDist, step = x, dist, &lineupStep{move: true, a: a, col: c, row: r}
					}
				}
			}
			sa.Col, sa.Row = col, row
		}
		switch {
		case step == nil:
			return rating
		case step.move:
			sq[step.a].Col, sq[step.a].Row = step.col, step.row
		default:
			swapPlaces(&sq[step.a], &sq[step.b])
		}
		rating = best
	}
}

// breakTies decides what the rating leaves open: first the players of their
// own line (preferNatural), then the stronger (preferStronger). It reports
// whether it changed the lineup.
func (s *State) breakTies(lr *lineupRater) bool {
	natural := s.preferNatural(lr)
	return s.preferStronger(lr) || natural
}

// natural tells whether the player of sl stands in the line of his
// position: the goalkeeper in goal, a field player on a cell of the line
// that is his.
func (s *State) natural(sl *Slot) bool {
	pos := s.Persons[sl.Person].Position()
	if pos == PosKeeper || sl.Row == 7 {
		return pos == PosKeeper && sl.Row == 7
	}
	return pos == lineOfRow[7-sl.Row]+1
}

// preferNatural puts a player of the bench whose position is the line in the
// place of a field player who stands out of his line, as long as the score
// falls by less than LineupMinGain in all: the rating does not know who
// scores and who gets the cards, which the engine picks by the position.
// Every swap leaves one player less out of his line, so it stops. It reports
// whether it changed the lineup.
func (s *State) preferNatural(lr *lineupRater) (changed bool) {
	sq := &s.Squads[lr.m]
	floor := lr.score() - LineupMinGain + 1
	for {
		a, b, best := -1, -1, floor-1
		for i := 0; i < SquadMax; i++ {
			if !sq[i].InTeam() || s.natural(&sq[i]) {
				continue
			}
			for j := 0; j < SquadMax; j++ {
				if !s.swappable(sq, i, j) || sq[j].InTeam() {
					continue
				}
				swapPlaces(&sq[i], &sq[j])
				if s.natural(&sq[j]) && s.fitsCell(&sq[i]) && s.fitsCell(&sq[j]) {
					if x := lr.score(); x > best {
						a, b, best = i, j, x
					}
				}
				swapPlaces(&sq[i], &sq[j])
			}
		}
		if a < 0 {
			return changed
		}
		swapPlaces(&sq[a], &sq[b])
		changed = true
	}
}

// strongerGap is how much stronger a player of the bench must be, in
// sixtieths of a point of Strength (playerStrength counts the Erschöpfung
// as a sixtieth), to take the place of a player the rating counts as good.
const strongerGap = 60

// preferStronger lets a player of the bench who is stronger and less tired
// take the place of a player of his position, when the score (the rating
// less the load) does not fall: the rating is no better than that for what
// it leaves out of the match (cards, injuries, substitutions) and for the
// clubs after the next, and a player who is as good for the match but
// stronger is never worse. Every swap raises the sum of the strengths of
// the team, so it stops. It reports whether it changed the lineup.
func (s *State) preferStronger(lr *lineupRater) (changed bool) {
	sq := &s.Squads[lr.m]
	for {
		base := lr.score()
		a, b, gap := -1, -1, strongerGap-1
		for i := 0; i < SquadMax; i++ {
			for j := 0; j < SquadMax; j++ {
				if !s.swappable(sq, i, j) || sq[j].InTeam() {
					continue
				}
				pi, pj := s.Persons[sq[i].Person], s.Persons[sq[j].Person]
				g := sq[j].Strength()*60 - sq[j].Fatigue - (sq[i].Strength()*60 - sq[i].Fatigue)
				if g <= gap || pi.Position() != pj.Position() {
					continue
				}
				swapPlaces(&sq[i], &sq[j])
				if s.fitsCell(&sq[i]) && s.fitsCell(&sq[j]) && lr.score() >= base {
					a, b, gap = i, j, g
				}
				swapPlaces(&sq[i], &sq[j])
			}
		}
		if a < 0 {
			return changed
		}
		swapPlaces(&sq[a], &sq[b])
		changed = true
	}
}

// stronger returns the start whose tuned lineup rates at least as well as
// the manager's (before) and fields a team of more strength, less fatigue
// counted, than orig, the most so; -1 if none does.
func stronger(orig *[SquadSlots]Slot, tuned *[4][SquadSlots]Slot, after [4]int, before int) int {
	weight := func(sq *[SquadSlots]Slot) (w int) {
		for i := range sq {
			if sq[i].InTeam() {
				w += sq[i].Strength()*60 - sq[i].Fatigue
			}
		}
		return w
	}
	best, top := -1, weight(orig)
	for f := range tuned {
		if after[f] == 0 || after[f] < before {
			continue
		}
		if w := weight(&tuned[f]); w > top {
			best, top = f, w
		}
	}
	return best
}

// lineupStep is a change improveLineup can make: a swap of the slots a and b,
// or a move of a to the cell col, row.
type lineupStep struct {
	move           bool
	a, b, col, row int
}

// swappable tells whether improveLineup may swap slot a of the team with
// slot b: a player of the bench who can play, or another of the team (each
// pair once).
func (s *State) swappable(sq *[SquadSlots]Slot, a, b int) bool {
	sa, sb := &sq[a], &sq[b]
	return b != a && sa.InTeam() && !sb.Empty() && s.available(sb) && (!sb.InTeam() || b > a)
}

// kickLineup gets the lineup out of the local optimum improveLineup stops
// at and returns the score reached: it makes every swap of two players that
// improveLineup could make, even one that lowers the score, improves the
// lineup from there and keeps the best lineup found; the next kicks start
// from that one. A single swap or move
// often cannot leave the optimum where two together would, as when a player
// goes to the bench and another one takes his cell. A second round of kicks
// from the lineup found gained a tenth of what the first did and doubled the
// time.
func (s *State) kickLineup(lr *lineupRater, rating int) int {
	sq := &s.Squads[lr.m]
	best := *sq
	for a := 0; a < SquadMax; a++ {
		for b := 0; b < SquadMax; b++ {
			*sq = best
			if !s.swappable(sq, a, b) {
				continue
			}
			swapPlaces(&sq[a], &sq[b])
			if !s.fitsCell(&sq[a]) || !s.fitsCell(&sq[b]) {
				continue
			}
			if x := s.improveLineup(lr, lr.score()); x >= rating+LineupMinGain {
				rating, best = x, *sq
			}
		}
	}
	*sq = best
	return rating
}

// swapPlaces swaps the numbers and the cells of two players.
func swapPlaces(a, b *Slot) {
	a.Lineup, b.Lineup = b.Lineup, a.Lineup
	swapCells(a, b)
}

// swapCells swaps the cells of two players.
func swapCells(a, b *Slot) {
	a.Row, b.Row = b.Row, a.Row
	a.Col, b.Col = b.Col, a.Col
}

// lineupChanges returns the changes of the lineup against orig, the most
// worth first. A player who came into the team plays for one who left it,
// preferably the one who stood on his cell, else one of his line; a player
// of both moved if his cell differs. What a change is worth is the score of
// the lineup less the score with that change alone undone.
func (s *State) lineupChanges(lr *lineupRater, orig *[SquadSlots]Slot) []LineupChange {
	sq := &s.Squads[lr.m]
	var outs []int
	for i := 0; i < SquadMax; i++ {
		if orig[i].InTeam() && !sq[i].InTeam() {
			outs = append(outs, i)
		}
	}
	var changes []LineupChange
	for i := 0; i < SquadMax; i++ {
		sl := &sq[i]
		if !sl.InTeam() {
			continue
		}
		if o := &orig[i]; o.InTeam() {
			if o.Col != sl.Col || o.Row != sl.Row {
				if ch, ok := s.moveChange(lr.m, orig, i, changes); ok {
					changes = append(changes, ch)
				}
			}
			continue
		}
		pick := -1
		for k, o := range outs {
			oc := &orig[o]
			switch {
			case oc.Col == sl.Col && oc.Row == sl.Row:
				pick = k
			case pick < 0 && lineOfRow[7-oc.Row] == lineOfRow[7-sl.Row]:
				pick = k
			}
		}
		if pick < 0 && len(outs) > 0 {
			pick = 0
		}
		ch := LineupChange{In: i, Out: -1, With: -1}
		if pick >= 0 {
			ch.Out = outs[pick]
			outs = append(outs[:pick], outs[pick+1:]...)
		}
		changes = append(changes, ch)
	}
	for _, o := range outs {
		changes = append(changes, LineupChange{In: -1, Out: o, With: -1})
	}

	base := lr.score()
	for k := range changes {
		ch := &changes[k]
		switch {
		case ch.In >= 0 && ch.Out >= 0:
			a, b := &sq[ch.In], &sq[ch.Out]
			if !s.available(b) {
				continue
			}
			swapPlaces(a, b)
			if s.fitsCell(b) {
				ch.Gain, ch.Rated = base-lr.score(), true
			}
			swapPlaces(a, b)
		case ch.Moved:
			a, o := &sq[ch.In], &orig[ch.In]
			if at := s.PlayerAt(lr.m, o.Col, o.Row); at >= 0 {
				b := &sq[at]
				swapCells(a, b)
				if s.fitsCell(a) && s.fitsCell(b) {
					ch.Gain, ch.Rated = base-lr.score(), true
				}
				swapCells(a, b)
			} else {
				col, row := a.Col, a.Row
				a.Col, a.Row = o.Col, o.Row
				ch.Gain, ch.Rated = base-lr.score(), true
				a.Col, a.Row = col, row
			}
		}
	}
	sort.SliceStable(changes, func(i, j int) bool { return changes[i].Gain > changes[j].Gain })
	return changes
}

// lineupReasons returns why every player plays in the lineup or not: the
// best swap with a player on the other side of the team, as improveLineup
// would make it (numbers and cells, both on cells that fit).
func (s *State) lineupReasons(lr *lineupRater) (rs [SquadMax]PlayerReason) {
	sq := &s.Squads[lr.m]
	base := lr.score()
	best := [SquadMax]int{}
	for i := range rs {
		rs[i].Alt = -1
	}
	for a := 0; a < SquadMax; a++ {
		sa := &sq[a]
		if !sa.InTeam() {
			continue
		}
		for b := 0; b < SquadMax; b++ {
			sb := &sq[b]
			if sb.Empty() || sb.InTeam() || !s.available(sb) {
				continue
			}
			swapPlaces(sa, sb)
			if s.fitsCell(sa) && s.fitsCell(sb) {
				x := lr.score()
				if rs[a].Alt < 0 || x > best[a] {
					rs[a].Alt, best[a] = b, x
				}
				if rs[b].Alt < 0 || x > best[b] {
					rs[b].Alt, best[b] = a, x
				}
			}
			swapPlaces(sa, sb)
		}
	}
	for i := range rs {
		if rs[i].Alt < 0 {
			continue
		}
		if sq[i].InTeam() {
			rs[i].Margin = base - best[i]
		} else {
			rs[i].Margin = best[i] - base
		}
	}
	return rs
}

// moveChange is the change of slot i, who stays in the team on another
// cell: one of its own, or a swap with the player of his old cell if that
// one stood on the new cell. If the swap is among the changes already, it
// reports false.
func (s *State) moveChange(m int, orig *[SquadSlots]Slot, i int, changes []LineupChange) (LineupChange, bool) {
	sl, o := &s.Squads[m][i], &orig[i]
	j := s.PlayerAt(m, o.Col, o.Row)
	if j < 0 || !orig[j].InTeam() || orig[j].Col != sl.Col || orig[j].Row != sl.Row {
		return LineupChange{In: i, Out: -1, With: -1, Moved: true}, true
	}
	for _, ch := range changes {
		if ch.In == j && ch.With == i {
			return LineupChange{}, false
		}
	}
	return LineupChange{In: i, Out: -1, With: j, Moved: true}, true
}

// playable tells whether the team of the squad has 11 players who can play.
func (s *State) playable(sq []Slot) bool {
	n := 0
	for i := range sq {
		sl := &sq[i]
		if !sl.InTeam() {
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
	if !sl.InTeam() {
		return true
	}
	pos := s.Persons[sl.Person].Position()
	if pos == PosKeeper || sl.Row == 7 {
		return pos == PosKeeper && sl.Row == 7
	}
	return abs(pos-(lineOfRow[7-sl.Row]+1)) <= 1
}
