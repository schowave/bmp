package game

import "testing"

// snapshot is everything OptimizeLineup must leave alone.
type snapshot struct {
	rand   Rand
	teams  [NumTeams]Team
	effort int
}

func snap(s *State) snapshot {
	return snapshot{s.Rand, s.Teams, s.Managers[0].Effort}
}

func checkLineup(t *testing.T, s *State) {
	t.Helper()
	cells := map[[2]int]bool{}
	n, numbers := 0, map[int]bool{}
	for i := 0; i < SquadMax; i++ {
		sl := &s.Squads[0][i]
		if sl.Empty() || sl.Lineup == 0 {
			continue
		}
		if numbers[sl.Lineup] {
			t.Fatalf("number %d twice", sl.Lineup)
		}
		numbers[sl.Lineup] = true
		if sl.Lineup <= 11 {
			n++
			c := [2]int{sl.Col, sl.Row}
			if cells[c] {
				t.Fatalf("cell %v twice", c)
			}
			cells[c] = true
			if !s.available(sl) {
				t.Fatalf("slot %d is out but in the team", i)
			}
		}
	}
	if n != 11 {
		t.Fatalf("%d players in the team", n)
	}
}

func TestOptimizeLineupLeavesTheGameAlone(t *testing.T) {
	s := lineupState(t)
	before := snap(s)
	squads := s.Squads
	res := s.OptimizeLineup(0)
	if got := snap(s); got != before {
		t.Error("generator, teams or effort changed")
	}
	for m := 1; m < len(s.Squads); m++ {
		if s.Squads[m] != squads[m] {
			t.Errorf("squad %d changed", m)
		}
	}
	checkLineup(t, s)
	if got := s.LineupRating(0); got != res.After {
		t.Errorf("rating %d, result says %d", got, res.After)
	}
}

func TestOptimizeLineupNotWorse(t *testing.T) {
	for _, team := range []int{9, 20, 40} {
		s, err := NewGame(mana(t), Options{Managers: []NewManager{{Name: "TEST", Team: team}}, Level: 2, Seed: 5})
		if err != nil {
			t.Fatal(err)
		}
		s.Start()
		for i := range s.Squads[0] {
			s.Squads[0][i].Status &^= 3
		}
		s.SetFormation(0, 1)
		res := s.OptimizeLineup(0)
		checkLineup(t, s)
		if res.After < res.Before {
			t.Errorf("team %d: %d after, %d before", team, res.After, res.Before)
		}
		// A new lineup must gain LineupMinGain against the old one too.
		if res.After != res.Before && res.After < res.Before+LineupMinGain {
			t.Errorf("team %d: %d after, %d before: less than the minimum gain", team, res.After, res.Before)
		}
		for f, a := range res.Auto {
			if res.After < a {
				t.Errorf("team %d: %d after, %d for %s", team, res.After, a, FormationNames[f])
			}
		}
		if res.Manual != (s.Managers[0].AutoLineup == 1) {
			t.Errorf("team %d: manual %v, mode %d", team, res.Manual, s.Managers[0].AutoLineup)
		}
		t.Logf("team %d: before %d, auto %v, after %d, %s manual %v", team, res.Before, res.Auto, res.After,
			FormationNames[res.Formation], res.Manual)
	}
}

// A lineup without a goalkeeper in goal is the worst there is.
func TestOptimizeLineupFindsGoalkeeper(t *testing.T) {
	s := lineupState(t)
	gk := s.PlayerAt(0, 3, 7)
	sub := findLineup(s, 0, func(sl *Slot) bool {
		return sl.Lineup == 0 && s.Persons[sl.Person].Position() != PosKeeper
	})
	if gk < 0 || sub < 0 {
		t.Fatal("no goalkeeper or substitute")
	}
	a, b := &s.Squads[0][gk], &s.Squads[0][sub]
	a.Lineup, b.Lineup = b.Lineup, a.Lineup
	a.Row, b.Row = b.Row, a.Row
	a.Col, b.Col = b.Col, a.Col
	bad := s.LineupRating(0)
	res := s.OptimizeLineup(0)
	if res.Before != bad || res.After <= bad {
		t.Fatalf("before %d (%d), after %d", res.Before, bad, res.After)
	}
	if i := s.PlayerAt(0, 3, 7); i < 0 || s.Persons[s.Squads[0][i].Person].Position() != PosKeeper {
		t.Errorf("no goalkeeper in goal: slot %d", i)
	}
}

func TestOptimizeLineupKeepsInjuredOut(t *testing.T) {
	s := lineupState(t)
	best := findLineup(s, 0, func(sl *Slot) bool { return sl.Lineup != 0 && s.Persons[sl.Person].Position() == PosAttack })
	if best < 0 {
		t.Fatal("no forward in the team")
	}
	sl := &s.Squads[0][best]
	sl.Skill = [3]int{99, 99, 99}
	sl.Status |= 2
	sl.Out = 5
	s.OptimizeLineup(0)
	if sl.Lineup != 0 {
		t.Errorf("injured player has number %d", sl.Lineup)
	}
	checkLineup(t, s)
}

// The mean of the dice stands for the dice: the rating is close to the
// average of the ratings with the real generator.
func TestLineupRatingIsTheMean(t *testing.T) {
	s := lineupState(t)
	lr := s.newLeagueRater(0)
	r := Rand{Seed: 1}
	sum, n := 0, 400
	for i := 0; i < n; i++ {
		sum += lr.ofValues(s.lineupValues(0, true, &r))
	}
	if got, want := lr.rate(), sum/n; got < want-5 || got > want+5 {
		t.Errorf("rating %d, average of the dice %d", got, want)
	}
}

// The chance table agrees with the engine: the expected chances of a few
// midfield differences against a run of chanceCounts, and the odds of a goal
// against the code of chance.
func TestChanceModel(t *testing.T) {
	r := Rand{Seed: 99}
	for _, d := range []int{-30, -8, 0, 5, 20} {
		var h, a int
		n := 20000
		for i := 0; i < n; i++ {
			x, y := chanceCounts(&r, 3*(100+d), 300, 1, 45)
			h, a = h+x, a+y
		}
		eh, ea := expectedChances(d)
		if dh, da := eh-float64(h)/float64(n), ea-float64(a)/float64(n); dh > 0.05 || dh < -0.05 || da > 0.05 || da < -0.05 {
			t.Errorf("d %d: table %.2f/%.2f, run %.2f/%.2f", d, eh, ea, float64(h)/float64(n), float64(a)/float64(n))
		}
	}
	for _, c := range []struct{ side, dd int }{{0, 12}, {1, 12}, {0, -6}, {1, -20}, {0, 0}} {
		goals, n := 0, 100000
		for i := 0; i < n; i++ {
			var pass bool
			if c.side == 0 {
				pass = r.Between(0, 4) >= 2
			} else {
				pass = r.Between(0, 1) != 0
			}
			if pass {
				if c.dd > 0 {
					if r.Between(0, (c.dd>>2)+2) != 0 {
						goals++
					}
				} else if k := (-c.dd) >> 2; r.Between(0, k) == 0 || r.Between(0, 6) == 0 {
					goals++
				}
			}
		}
		if got, want := goalOdds(c.side, c.dd), float64(goals)/float64(n); got < want-0.01 || got > want+0.01 {
			t.Errorf("side %d dd %d: odds %.3f, run %.3f", c.side, c.dd, got, want)
		}
	}
}

// The result is a lineup where no single move to a free cell of the same line
// helps.
func TestOptimizeLineupMovesToFreeCells(t *testing.T) {
	s := lineupState(t)
	// A defender stands in the attack, far from his row.
	d := findLineup(s, 0, func(sl *Slot) bool {
		return sl.Lineup != 0 && s.Persons[sl.Person].Position() == PosDefence
	})
	if d < 0 {
		t.Fatal("no defender in the team")
	}
	s.MoveOnBoard(0, d, 0, 0)
	bad := s.LineupRating(0)
	res := s.OptimizeLineup(0)
	checkLineup(t, s)
	if res.After < bad {
		t.Errorf("rating fell from %d to %d", bad, res.After)
	}
	for i := 0; i < SquadMax; i++ {
		sl := &s.Squads[0][i]
		if sl.Empty() || sl.Lineup < 1 || sl.Lineup > 11 {
			continue
		}
		col, row := sl.Col, sl.Row
		for c := 0; c < BoardCols; c++ {
			for r := 0; r < BoardRows; r++ {
				if s.PlayerAt(0, c, r) >= 0 || lineOfRow[7-r] != lineOfRow[7-row] {
					continue
				}
				sl.Col, sl.Row = c, r
				if !s.fitsCell(sl) {
					continue
				}
				if x := s.LineupRating(0); x >= res.After+LineupMinGain {
					t.Errorf("slot %d to %d,%d: rating %d > %d", i, c, r, x, res.After)
				}
			}
		}
		sl.Col, sl.Row = col, row
	}
}

// Nobody is fielded out of place: the goalkeeper in goal, field players in
// their own line or next to it.
func TestOptimizeLineupKeepsPositions(t *testing.T) {
	for _, team := range []int{9, 20, 40, 55} {
		s, err := NewGame(mana(t), Options{Managers: []NewManager{{Name: "TEST", Team: team}}, Level: 2, Seed: 5})
		if err != nil {
			t.Fatal(err)
		}
		s.Start()
		for i := range s.Squads[0] {
			s.Squads[0][i].Status &^= 3
		}
		s.OptimizeLineup(0)
		for i := 0; i < SquadMax; i++ {
			sl := &s.Squads[0][i]
			if !sl.Empty() && !s.fitsCell(sl) {
				t.Errorf("team %d: %s %d in row %d", team, PositionNames[s.Persons[sl.Person].Position()], sl.Lineup, sl.Row)
			}
		}
	}
}

// chanceGoal resolves a chance like MatchDay.chance, without the score.
func chanceGoal(r *Rand, h, a *Team, side, minute int) bool {
	dd := (teamStrengthAt(h, 2, minute) - teamStrengthAt(a, 0, minute)) / 3
	if side == 1 {
		dd = (teamStrengthAt(a, 2, minute) - teamStrengthAt(h, 0, minute)) / 3
	}
	if side == 0 && r.Between(0, 4) < 2 || side == 1 && r.Between(0, 1) == 0 {
		return false
	}
	if dd > 0 {
		return r.Between(0, (dd>>2)+2) != 0
	}
	k := (-dd) >> 2
	return r.Between(0, k) == 0 || r.Between(0, 6) == 0
}

// diceRun plays the lineup with the strength values v against refs with the
// dice of the engine and returns the points per match in hundredths.
func diceRun(lr *lineupRater, v lineupValues, refs []Team, n int) float64 {
	us := Team{Mood: lr.mood, Strength: v.strength, Base: v.base, Extra: v.extra}
	r := Rand{Seed: 7}
	points := 0
	for k := range refs {
		for i := 0; i < n; i++ {
			for _, home := range lr.venues {
				h, a := &us, &refs[k]
				if !home {
					h, a = a, h
				}
				var goals [2]int
				for _, from := range []int{1, 46} {
					ch, ca := chanceCounts(&r, teamStrengthAt(h, 1, from), teamStrengthAt(a, 1, from), from, from+44)
					for side, cnt := range [2]int{ch, ca} {
						for c := 0; c < cnt; c++ {
							if goals[side] < 9 && chanceGoal(&r, h, a, side, r.Between(from, from+44)) {
								goals[side]++
							}
						}
					}
				}
				own, other := goals[0], goals[1]
				if !home {
					own, other = other, own
				}
				switch {
				case own > other:
					points += lr.s.PointsPerWin()
				case own == other:
					points++
				}
			}
		}
	}
	return float64(points) / float64(n*len(refs)*len(lr.venues)) * 100
}

// The rating is the points the engine's own dice give: a run of
// chanceCounts, minutes and chanceGoal agrees with it. A few clubs of the
// league stand for all of them.
func TestLineupRatingAgreesWithADiceRun(t *testing.T) {
	s := lineupState(t)
	lr := s.newLeagueRater(0)
	refs := lr.refs[:5]
	for _, optimize := range []bool{false, true} {
		if optimize {
			s.OptimizeLineup(0)
		}
		v := s.lineupValues(0, true, meanDice{})
		got, want := diceRun(lr, v, refs, 4000), lr.ofValuesAgainst(v, refs)
		t.Logf("optimized %v: dice run %.1f, rating %d", optimize, got, want)
		if got < float64(want)-5 || got > float64(want)+5 {
			t.Errorf("optimized %v: dice run %.1f, rating %d", optimize, got, want)
		}
	}
}

// playToday makes the event of manager 0 a league match day, with an
// opponent.
func playToday(t *testing.T, s *State) (opp int, home bool) {
	t.Helper()
	for e := range s.Events {
		s.Event = e
		if o, h, ok := s.NextOpponent(0); ok && !s.humanTeam(o) {
			return o, h
		}
	}
	t.Fatal("no event with a match of manager 0")
	return 0, false
}

// With a match today the lineup is rated against the opponent in his venue,
// and what the dice of the engine give agrees with the rating.
func TestOptimizeLineupAgainstTheOpponent(t *testing.T) {
	s := lineupState(t)
	opp, home := playToday(t, s)
	lr := s.newRater(0)
	if lr.opp != opp || len(lr.venues) != 1 || lr.venues[0] != home || len(lr.refs) != 1 {
		t.Fatalf("rater: opponent %d (want %d), venues %v, %d refs", lr.opp, opp, lr.venues, len(lr.refs))
	}
	before := snap(s)
	res := s.OptimizeLineup(0)
	if res.Opponent != opp || res.Home != home {
		t.Errorf("result against %d (home %v), want %d (%v)", res.Opponent, res.Home, opp, home)
	}
	if got := snap(s); got != before {
		t.Error("generator, teams or effort changed")
	}
	checkLineup(t, s)
	if res.After < res.Before || res.After != s.LineupRating(0) {
		t.Errorf("before %d, after %d, rating %d", res.Before, res.After, s.LineupRating(0))
	}
	var sum float64
	for k := 0; k < lr.samples; k++ {
		r := Rand{Seed: ratingSeed + uint32(k)}
		v := s.lineupValues(0, true, &r)
		sum += diceRun(lr, v, lr.refs, 1500)
	}
	if got, want := sum/float64(lr.samples), float64(res.After); got < want-8 || got > want+8 {
		t.Errorf("dice run %.1f, rating %d", got, res.After)
	}
	t.Logf("against %s (home %v): before %d, auto %v, after %d", s.Teams[opp].Name, home, res.Before, res.Auto, res.After)
}

// An opponent with a human manager is not known, so the league counts.
func TestRaterFallsBackToTheLeague(t *testing.T) {
	s := lineupState(t)
	opp, _ := playToday(t, s)
	s.Managers = append(s.Managers, &Manager{Team: opp})
	if lr := s.newRater(0); lr.opp != -1 || len(lr.venues) != 2 {
		t.Errorf("opponent %d, venues %v", lr.opp, lr.venues)
	}
}

// Without a match today the league counts.
func TestOptimizeLineupWithoutAMatch(t *testing.T) {
	s := lineupState(t)
	found := false
	for e := range s.Events {
		s.Event = e
		if _, _, ok := s.NextOpponent(0); !ok {
			found = true
			break
		}
	}
	if !found {
		t.Skip("every event has a match of manager 0")
	}
	res := s.OptimizeLineup(0)
	if res.Opponent != -1 || res.After < res.Before {
		t.Errorf("opponent %d, before %d, after %d", res.Opponent, res.Before, res.After)
	}
	checkLineup(t, s)
}

// A tired player rests when a fresh one on the bench is as good, and plays
// when he makes a difference.
func TestOptimizeLineupRestsTiredPlayers(t *testing.T) {
	for _, star := range []bool{false, true} {
		s := lineupState(t)
		tired, fresh := -1, -1
		for i := 0; i < SquadMax; i++ {
			sl := &s.Squads[0][i]
			if sl.Empty() || s.Persons[sl.Person].Position() != PosMidfield {
				continue
			}
			switch {
			case sl.Lineup >= 1 && sl.Lineup <= 11 && tired < 0:
				tired = i
			case (sl.Lineup == 0 || sl.Lineup > 11) && s.available(sl) && fresh < 0:
				fresh = i
			}
		}
		if tired < 0 || fresh < 0 {
			t.Fatal("no midfielder in the team or on the bench")
		}
		a, b := &s.Squads[0][tired], &s.Squads[0][fresh]
		b.Skill, b.Fatigue, b.Tendency = a.Skill, 80, a.Tendency
		s.Persons[b.Person].Side, s.Persons[b.Person].Pos = s.Persons[a.Person].Side, s.Persons[a.Person].Pos
		a.Fatigue = 140
		if star {
			a.Skill = [3]int{99, 99, 55}
		}
		res := s.OptimizeLineup(0)
		plays := a.Lineup >= 1 && a.Lineup <= 11
		if plays != star {
			t.Errorf("star %v: the tired player plays %v", star, plays)
		}
		if res.Load[tired] <= 0 || res.Load[fresh] != 0 {
			t.Errorf("load: tired %d, fresh %d", res.Load[tired], res.Load[fresh])
		}
		// The rest is a change of its own, worth what it gains.
		found := false
		for _, ch := range res.Changes {
			if ch.Out == tired {
				found = true
				if ch.In < 0 || !ch.Rated || ch.Gain < LineupMinGain {
					t.Errorf("rest of the tired player: %+v", ch)
				}
			}
		}
		if found == star {
			t.Errorf("star %v: changes %+v", star, res.Changes)
		}
	}
}

// In a cup match nobody rests for his load: the tired star of
// TestOptimizeLineupRestsTiredPlayers plays when the club has a DFB-Pokal
// match, and a draw counts half a win there whatever a win gives.
func TestOptimizeLineupCupIgnoresLoad(t *testing.T) {
	s := lineupState(t)
	tired, fresh := -1, -1
	for i := 0; i < SquadMax; i++ {
		sl := &s.Squads[0][i]
		if sl.Empty() || s.Persons[sl.Person].Position() != PosMidfield {
			continue
		}
		switch {
		case sl.Lineup >= 1 && sl.Lineup <= 11 && tired < 0:
			tired = i
		case (sl.Lineup == 0 || sl.Lineup > 11) && s.available(sl) && fresh < 0:
			fresh = i
		}
	}
	if tired < 0 || fresh < 0 {
		t.Fatal("no midfielder in the team or on the bench")
	}
	a, b := &s.Squads[0][tired], &s.Squads[0][fresh]
	b.Skill, b.Fatigue, b.Tendency = a.Skill, 80, a.Tendency
	s.Persons[b.Person].Side, s.Persons[b.Person].Pos = s.Persons[a.Person].Side, s.Persons[a.Person].Pos
	// A little better than the fresh one, not worth his load, and in the team.
	a.Fatigue = 140
	a.Skill[0], a.Skill[1] = min(b.Skill[0]+2, 99), min(b.Skill[1]+2, 99)
	orig := s.Squads[0]
	if res := s.OptimizeLineup(0); res.Cup || a.Lineup >= 1 && a.Lineup <= 11 {
		t.Fatalf("league: cup %v, the tired player plays %v", res.Cup, a.Lineup)
	}
	s.Squads[0] = orig

	// The same club in a round of the DFB-Pokal.
	opp := 0
	if s.Managers[0].Team == 0 {
		opp = 1
	}
	s.Events[s.Event] = EvDFB
	s.Cups.Round[CupDFB] = 1
	s.Cups.Slots[CupDFB][0], s.Cups.Slots[CupDFB][1] = s.Managers[0].Team, opp
	res := s.OptimizeLineup(0)
	if !res.Cup || res.Opponent != opp || !res.Home {
		t.Fatalf("cup match: %+v", res)
	}
	if res.Load != [SquadMax]int{} {
		t.Errorf("load in a cup match: %v", res.Load)
	}
	if a.Lineup < 1 || a.Lineup > 11 {
		t.Errorf("the better player rests in a cup match (lineup %d)", a.Lineup)
	}
}

// Each change undone costs what it says it gains, and the changes add up to
// the lineup chosen.
func TestOptimizeLineupChanges(t *testing.T) {
	s := lineupState(t)
	orig := s.Squads[0]
	res := s.OptimizeLineup(0)
	if len(res.Changes) == 0 {
		t.Fatal("no changes")
	}
	ins := map[int]bool{}
	for k, ch := range res.Changes {
		if k > 0 && ch.Gain > res.Changes[k-1].Gain {
			t.Errorf("changes not sorted: %+v", res.Changes)
		}
		if ch.In < 0 {
			continue
		}
		ins[ch.In] = true
		if ch.With >= 0 {
			ins[ch.With] = true
		}
		sl := &s.Squads[0][ch.In]
		if !sl.InTeam() {
			t.Errorf("slot %d of a change is not in the team", ch.In)
		}
		if ch.Out >= 0 && (!orig[ch.Out].InTeam() || s.Squads[0][ch.Out].InTeam()) {
			t.Errorf("slot %d did not leave the team", ch.Out)
		}
	}
	for i := 0; i < SquadMax; i++ {
		sl, o := &s.Squads[0][i], &orig[i]
		if sl.InTeam() && (!o.InTeam() || o.Col != sl.Col || o.Row != sl.Row) && !ins[i] {
			t.Errorf("slot %d changed without a change", i)
		}
	}
}

// A tired player costs more in a week with two matches than in one with a
// match, less before a break, and nothing when he is fresh or the season
// ends.
func TestLoadCosts(t *testing.T) {
	s := lineupState(t)
	l := s.Managers[0].League
	i := findLineup(s, 0, func(sl *Slot) bool { return sl.InTeam() })
	sl := &s.Squads[0][i]
	load := func(events ...byte) int {
		s.Events = [len(s.Events)]byte{}
		copy(s.Events[:], events)
		s.Event, s.Day = 0, 0
		return s.loadCosts(0)[i]
	}
	m := byte(1 << l)
	sl.Fatigue = 80
	if c := load(m, 0, m, 0, m, 0, m, 0); c != 0 {
		t.Errorf("fresh player costs %d", c)
	}
	sl.Fatigue = 132
	weekly := load(m, 0, m, 0, m, 0, m, 0)
	short := load(m, m, m, m, m, m, m, m)
	last := load(m, 0, 0, 0, 0, 0, 0, 0)
	if weekly <= 0 || short <= weekly || last >= weekly {
		t.Errorf("weekly %d, short week %d, before a break %d", weekly, short, last)
	}
	s.Day = LastSeasonDay
	if c := s.loadCosts(0)[i]; c != 0 {
		t.Errorf("at the season end he costs %d", c)
	}
	t.Logf("weekly %d, short week %d, before a break %d", weekly, short, last)

	// The training before a match that is not today lets him recover.
	if later := load(0, m, 0, m, 0, m, 0, m, 0); later >= weekly {
		t.Errorf("match a turn later costs %d, today %d", later, weekly)
	}
	// In the winter break the training adds to the Erschöpfung, but on
	// level 2.
	winter := func(level int) int {
		s.Level = level
		s.Events = [len(s.Events)]byte{}
		copy(s.Events[:], []byte{m, 0, m, 0, m, 0, m, 0})
		s.Event, s.Day = 0, 140
		return s.loadCosts(0)[i]
	}
	if w2, w3 := winter(2), winter(3); w3 <= w2 {
		t.Errorf("winter break: level 3 costs %d, level 2 %d", w3, w2)
	}
}

// A player who comes into a team that had no 11 players is new, not moved,
// and has no value.
func TestOptimizeLineupNewcomer(t *testing.T) {
	s := lineupState(t)
	out := findLineup(s, 0, func(sl *Slot) bool { return sl.InTeam() && s.Persons[sl.Person].Position() != PosKeeper })
	s.Squads[0][out].Lineup = 0
	s.renumber(0)
	res := s.OptimizeLineup(0)
	checkLineup(t, s)
	found := false
	for _, ch := range res.Changes {
		if ch.In >= 0 && ch.Out < 0 && !ch.Moved {
			found = true
			if ch.Rated || ch.With >= 0 {
				t.Errorf("newcomer %+v", ch)
			}
		}
	}
	if !found {
		t.Errorf("no newcomer in %+v", res.Changes)
	}
}

// Every player of the team who has a substitute is worth more than him (the
// search stops where no swap helps), and every player outside the team who
// could play names the one he would replace.
func TestOptimizeLineupReasons(t *testing.T) {
	s := lineupState(t)
	res := s.OptimizeLineup(0)
	n := 0
	for i := 0; i < SquadMax; i++ {
		sl, r := &s.Squads[0][i], res.Reasons[i]
		switch {
		case sl.Empty():
			continue
		case !s.available(sl):
			if r.Alt >= 0 {
				t.Errorf("slot %d cannot play but has %+v", i, r)
			}
		case sl.InTeam():
			if r.Alt >= 0 && (s.Squads[0][r.Alt].InTeam() || r.Margin <= -LineupMinGain) {
				t.Errorf("slot %d in the team: %+v", i, r)
			}
			n++
		default:
			if r.Alt < 0 || !s.Squads[0][r.Alt].InTeam() || r.Margin >= LineupMinGain {
				t.Errorf("slot %d outside the team: %+v", i, r)
			}
		}
	}
	if n != 11 {
		t.Errorf("%d players in the team", n)
	}
}

// A stronger player of the bench takes the place of a weaker one of his
// position when the score does not fall, and a weaker or hardly stronger one
// does not.
func TestPreferStronger(t *testing.T) {
	s := lineupState(t)
	a, b := -1, -1
	for i := 0; i < SquadMax; i++ {
		sl := &s.Squads[0][i]
		if sl.Empty() || s.Persons[sl.Person].Position() != PosMidfield {
			continue
		}
		switch {
		case sl.Lineup >= 1 && sl.Lineup <= 11 && a < 0:
			a = i
		case (sl.Lineup == 0 || sl.Lineup > 11) && s.available(sl) && b < 0:
			b = i
		}
	}
	if a < 0 || b < 0 {
		t.Fatal("no midfielder in the team or on the bench")
	}
	sa, sb := &s.Squads[0][a], &s.Squads[0][b]
	sb.Tendency, sb.Fatigue, sa.Fatigue = sa.Tendency, 80, 80
	s.Persons[sb.Person].Side, s.Persons[sb.Person].Pos = s.Persons[sa.Person].Side, s.Persons[sa.Person].Pos
	plays := func(sl *Slot) bool { return sl.Lineup >= 1 && sl.Lineup <= 11 }

	// The weakest player of the squad stays on the bench.
	sa.Skill, sb.Skill = [3]int{99, 99, 99}, [3]int{1, 1, 1}
	s.preferStronger(s.newRater(0))
	if !plays(sa) || plays(sb) {
		t.Errorf("a weaker player took a place: %v %v", plays(sa), plays(sb))
	}
	// The strongest takes the place of the weakest of his position.
	sa.Skill, sb.Skill = [3]int{1, 1, 1}, [3]int{99, 99, 99}
	if !s.preferStronger(s.newRater(0)) || plays(sa) || !plays(sb) {
		t.Errorf("the stronger player did not take the place: %v %v", plays(sa), plays(sb))
	}
	if s.preferStronger(s.newRater(0)) {
		t.Error("a second round changed the lineup")
	}
}

// A player out of his line gives his place to one of the bench whose position
// the line is, if that costs the score less than LineupMinGain, and keeps it
// if the other is much weaker.
func TestPreferNatural(t *testing.T) {
	s := lineupState(t)
	a, b := -1, -1
	for i := 0; i < SquadMax; i++ {
		sl := &s.Squads[0][i]
		if sl.Empty() || s.Persons[sl.Person].Position() != PosMidfield {
			continue
		}
		switch {
		case sl.Lineup >= 1 && sl.Lineup <= 11 && a < 0:
			a = i
		case (sl.Lineup == 0 || sl.Lineup > 11) && s.available(sl) && b < 0:
			b = i
		}
	}
	if a < 0 || b < 0 {
		t.Fatal("no midfielder in the team or on the bench")
	}
	sa, sb := &s.Squads[0][a], &s.Squads[0][b]
	sb.Tendency, sb.Fatigue, sa.Fatigue = sa.Tendency, 80, 80
	s.Persons[sb.Person].Side, s.Persons[sb.Person].Pos = s.Persons[sa.Person].Side, s.Persons[sa.Person].Pos
	// a is a defender in a cell of the midfield.
	s.Persons[sa.Person].Pos = 30
	plays := func(sl *Slot) bool { return sl.Lineup >= 1 && sl.Lineup <= 11 }
	if s.natural(sa) {
		t.Fatal("the defender in the midfield is natural")
	}

	sa.Skill, sb.Skill = [3]int{99, 99, 99}, [3]int{1, 1, 1}
	s.preferNatural(s.newRater(0))
	if !plays(sa) || plays(sb) {
		t.Errorf("a much weaker player took the place: %v %v", plays(sa), plays(sb))
	}
	sa.Skill, sb.Skill = [3]int{60, 60, 60}, [3]int{60, 60, 60}
	if !s.preferNatural(s.newRater(0)) || plays(sa) || !plays(sb) {
		t.Errorf("the natural player did not take the place: %v %v", plays(sa), plays(sb))
	}
	if s.preferNatural(s.newRater(0)) {
		t.Error("a second round changed the lineup")
	}
}

func TestStrongerStartReplacesKeptLineup(t *testing.T) {
	var orig [SquadSlots]Slot
	var tuned [4][SquadSlots]Slot
	team := func(sq *[SquadSlots]Slot, skill int) {
		sq[0] = Slot{Person: 1, Lineup: 1, Skill: [3]int{skill, skill, skill}}
	}
	team(&orig, 62)
	team(&tuned[1], 86)
	team(&tuned[2], 70)
	if got := stronger(&orig, &tuned, [4]int{0, 100, 100, 0}, 100); got != 1 {
		t.Errorf("stronger = %d, want 1", got)
	}
	if got := stronger(&orig, &tuned, [4]int{0, 99, 100, 0}, 100); got != 2 {
		t.Errorf("a start that rates worse: stronger = %d, want 2", got)
	}
	if got := stronger(&orig, &tuned, [4]int{}, 100); got != -1 {
		t.Errorf("no start: stronger = %d, want -1", got)
	}
}
