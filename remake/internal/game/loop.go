package game

// Phase is what the game is waiting for.
type Phase int

// Phases of the game loop.
const (
	// PhaseTurn: manager Current takes his turn (main menu) until EndTurn.
	PhaseTurn Phase = iota
	// PhaseMatches: NextMatchDay returns the match days of the event.
	PhaseMatches
	// PhaseContracts: at the season end, expiring contracts of manager
	// Current have to be settled (ExpiringContracts, ReleasePlayer).
	PhaseContracts
	// PhaseGameOver: the chosen game length is over.
	PhaseGameOver
)

// Loop is the position of the game loop f_1a58_3176.
type Loop struct {
	Phase     Phase
	Current   int  // manager whose turn it is
	SeasonEnd bool // final turns of the season
	MatchStep int
	Before    []int64
	Scorer    []bool
}

// Start begins the game after NewGame: the first event's turns.
func (s *State) Start() {
	s.beginEvent()
}

// beginEvent runs the top of the loop for event s.Event and starts the
// turns, or skips events without games.
func (s *State) beginEvent() {
	for {
		for mi := range s.Managers {
			s.daily(mi)
		}
		s.driftStrengths(1)
		if s.eventValue() != 0 {
			for mi := range s.Managers {
				s.AutoLineup(mi)
				s.LineupStrength(mi, false)
			}
			s.Loop.Phase = PhaseTurn
			s.Loop.Current = 0
			s.Loop.SeasonEnd = false
			s.beforeTurn()
			return
		}
		if s.afterEvent() {
			return
		}
	}
}

func (s *State) eventValue() byte {
	ev := s.Events[s.Event]
	if ev == 9 {
		ev = 0
	}
	return ev
}

// beforeTurn runs what happens before a manager's turn: sometimes the
// transfer market changes (f_1a58_3176 @3b65).
func (s *State) beforeTurn() {
	if s.Rand.Between(0, len(s.Managers)+3) == 0 {
		s.refreshMarket()
	}
	s.goalMessages(s.Loop.Current)
}

// EndTurn ends the turn of the current manager.
func (s *State) EndTurn() {
	if s.Loop.Phase != PhaseTurn {
		return
	}
	mi := s.Loop.Current
	if s.Loop.SeasonEnd {
		s.recordHistory(mi)
	}
	if mi+1 < len(s.Managers) {
		s.Loop.Current++
		if !s.Loop.SeasonEnd {
			s.beforeTurn()
		}
		return
	}
	if s.Loop.SeasonEnd {
		s.seasonEnd1()
		return
	}
	for mi := range s.Managers {
		s.LineupStrength(mi, true)
		s.Managers[mi].Subs = 0
	}
	s.Loop.Phase = PhaseMatches
	s.Loop.MatchStep = 0
}

// Match day kinds returned by NextMatchDay, in the order of the original.
const (
	stepLeague = iota
	stepDFB
	stepEurope
	stepReplay
	stepDone
)

// NextMatchDay returns the next match day of the current event, or nil
// when all have been played; then the game continues to the next turn.
func (s *State) NextMatchDay() *MatchDay {
	if s.Loop.Phase != PhaseMatches {
		return nil
	}
	ev := s.eventValue()
	for s.Loop.MatchStep < stepDone {
		step := s.Loop.MatchStep
		s.Loop.MatchStep++
		switch step {
		case stepLeague:
			if ev&7 != 0 {
				if ev&1 != 0 {
					s.postpone(0)
				}
				s.postpone(1)
				s.postpone(2)
				md := s.newMatchDay(int(ev&7), 0, false)
				md.Title = "Ligaspiel"
				return md
			}
		case stepDFB:
			if ev&EvDFB != 0 {
				md := s.newMatchDay(0, 1, false)
				md.Title = CupNames[CupDFB] + " " + RoundNames[s.Cups.Round[CupDFB]]
				return md
			}
		case stepEurope:
			if ev&EvEurope == EvRelegation {
				return s.relegationDay()
			}
			if ev&EvEurope != 0 {
				md := s.newMatchDay(0, 0x0e, false)
				md.Title = "Europapokal " + RoundNames[s.Cups.Round[CupLandesmeister]]
				return md
			}
		case stepReplay:
			if ev&EvReplay != 0 && s.hasReplays() {
				return s.newMatchDay(0, 0, true)
			}
		}
	}
	s.finishReplays()
	s.afterMatches()
	return nil
}

// afterMatches continues the loop after the matches of an event.
func (s *State) afterMatches() {
	if s.afterEvent() {
		return
	}
	s.beginEvent()
}

// afterEvent runs LAB_3397 of the loop: matchday counters, days until the
// next turn day, daily and twice-weekly processing. It returns true when
// the loop stopped in another phase (season end).
func (s *State) afterEvent() bool {
	for l := 0; l < NumLeagues; l++ {
		if s.Events[s.Event]&(1<<l) != 0 {
			s.Matchday[l]++
		}
	}
	for mi := range s.Managers {
		for i := 0; i < SquadMax; i++ {
			if s.Squads[mi][i].Out != 0 {
				s.Squads[mi][i].Lineup = 0
			}
		}
	}
	s.Day++
	s.checkSpecialDates()
	for {
		for mi := range s.Managers {
			s.daily(mi)
		}
		s.Day++
		s.checkSpecialDates()
		if s.Day%7 == 0 || (s.Day-4)%7 == 0 {
			break
		}
	}
	for mi := range s.Managers {
		s.turn(mi)
		if s.Day%14 == 0 {
			s.sponsorOffers(mi)
		}
	}
	if s.Day <= LastSeasonDay {
		s.Event++
		return false
	}
	// Final turns of the season.
	for mi := range s.Managers {
		s.AutoLineup(mi)
		s.LineupStrength(mi, false)
	}
	s.Loop.Phase = PhaseTurn
	s.Loop.Current = 0
	s.Loop.SeasonEnd = true
	return true
}

var specialDates = [3][2]int{{24, 12}, {19, 11}, {12, 4}} // D:0x53d0 / D:0x53d4

func (s *State) checkSpecialDates() {
	d := DateOf(s.StartYear, s.Day-1)
	for k, sd := range specialDates {
		if d.Day == sd[0] && d.Month == sd[1]-1 {
			s.specialDate(k)
		}
	}
}

// postpone moves some winter games to later dates (f_0310_0463).
func (s *State) postpone(l int) {
	e := s.Event
	if e < 25 || e > 69 {
		return
	}
	r := &s.Rand
	d := abs(e - 44)
	n := 9 - d/3
	if n != 0 && r.Between(0, 1) != 0 {
		n = 8 - d/3
	}
	n = r.Between(0, n)
	var picked []int
	md := s.Matchday[l]
	for {
		if n <= 0 {
			return
		}
		if n > 8 {
			n = 8
		}
		before := n
		for rec := 0; rec < len(s.Postponed); rec++ {
			if s.Postponed[rec].Slot != 0 {
				continue
			}
			p := r.Between(0, TeamsPerLeague[l]/2-1)
			if indexOf(picked, p) >= 0 || s.Results[l][md-1][p].Home == Postponed {
				continue
			}
			if s.reschedule(l, p, rec) {
				n--
				picked = append(picked, p)
				break
			}
		}
		if n == before {
			return
		}
	}
}

// reschedule finds a new date for pair p of league l (f_0310_05f1).
func (s *State) reschedule(l, p, rec int) bool {
	md := s.Matchday[l]
	pair := Fixtures(l, md)[p]
	slot := s.Event
	for {
		slot++
		if slot > 89 {
			return false
		}
		if slot >= 36 && slot <= 58 {
			slot = 59
		}
		if v := s.Events[slot]; v != 0 && v != EvReplay {
			continue
		}
		conflict := false
		for _, q := range s.Postponed {
			if q.Slot != slot || q.Matchday == 0 {
				continue
			}
			o := Fixtures(q.League, q.Matchday)[q.Pair]
			if o.Home == pair.Home || o.Home == pair.Away || o.Away == pair.Home || o.Away == pair.Away {
				conflict = true
				break
			}
		}
		if !conflict {
			break
		}
	}
	s.Postponed[rec] = PostponedGame{Slot: slot, Matchday: md, League: l, Pair: p}
	s.Results[l][md-1][p] = Score{Home: Postponed}
	s.Events[slot] = EvReplay
	return true
}

func (s *State) hasReplays() bool {
	for _, p := range s.Postponed {
		if p.Slot == s.Event && p.Matchday != 0 {
			return true
		}
	}
	return false
}

// finishReplays frees the records of today's replays.
func (s *State) finishReplays() {
	if s.Events[s.Event]&EvReplay == 0 {
		return
	}
	for i := range s.Postponed {
		if s.Postponed[i].Slot == s.Event {
			s.Postponed[i] = PostponedGame{}
		}
	}
	s.Events[s.Event] = 0
}

// relegationDay plays a leg of the relegation (f_0310_02bf): the 16th of
// the Bundesliga against the 3rd of the 2. Liga, in cup slot 1.
func (s *State) relegationDay() *MatchDay {
	c := &s.Cups
	savedRound := c.Round[CupLandesmeister]
	savedSlots := [2]int{c.Slots[1][0], c.Slots[1][1]}
	savedFirst := [2]int{c.First[1][0], c.First[1][1]}
	savedLeg := c.Leg[1]
	savedResult := [2]int{c.Result[1][0], c.Result[1][1]}
	c.Round[CupLandesmeister] = 5
	c.PlayingRelegation = true
	if !s.RelegationLeg2 {
		c.Leg[1] = 0
		c.First[1][0], c.First[1][1] = 0, 0
		c.Slots[1][0], c.Slots[1][1] = s.Tables[1][2], s.Tables[0][15]
	} else {
		c.Leg[1] = 1
		c.First[1][0], c.First[1][1] = c.RelegationStash[0], c.RelegationStash[1]
		c.Slots[1][0], c.Slots[1][1] = s.Tables[0][15], s.Tables[1][2]
	}
	md := s.newMatchDay(0, 2, false)
	md.Title = "Relegationsspiel"
	md.onFinish = func() {
		if !s.RelegationLeg2 {
			c.RelegationStash = [2]int{c.First[1][0], c.First[1][1]}
			s.RelegationLeg2 = true
		} else {
			c.RelegationOutcome = s.tieOut(1, 0)
			s.RelegationLeg2 = false
		}
		c.Round[CupLandesmeister] = savedRound
		c.Slots[1][0], c.Slots[1][1] = savedSlots[0], savedSlots[1]
		c.First[1][0], c.First[1][1] = savedFirst[0], savedFirst[1]
		c.Leg[1] = savedLeg
		c.Result[1][0], c.Result[1][1] = savedResult[0], savedResult[1]
		c.PlayingRelegation = false
	}
	return md
}

// goalMessages shows the "letzte Chance" messages of a manager
// (f_112a_314d).
func (s *State) goalMessages(mi int) {
	m := s.Managers[mi]
	l := m.League
	d := s.Date()
	n := TeamsPerLeague[l]
	if d.Month == 5 && d.Day == 13 && s.Matchday[l]-MatchdaysPerLeague[l] == 1 {
		pos := s.Clubs[m.Team].Pos
		if (l == 0 && pos == 15) || (l == 1 && pos == 2) {
			s.notify(mi, "Ihre Mannschaft bestreitet das Relegationsspiel. Viel Glück.")
			return
		}
	}
	if d.Month == 5 && d.Day > 13 {
		return
	}
	if d.Month == 11 && d.Day == 2 {
		s.notify(mi, "Achtung ! Dies ist der letzte Spieltag vor der Winterpause.")
	}
	idx := [3][3]int{{0, 4, n - 4}, {0, 1, n - 5}, {0, 3, n - 5}}[l]
	pts := func(t int) int { return s.Clubs[t].TotalPoints() }
	maxpts := func(t int) int {
		return pts(t) + (MatchdaysPerLeague[l]-s.Clubs[t].TotalGames())*PointsPerWin
	}
	me := m.Team
	club := s.Teams[me].Name
	lost := [3]string{"MEISTER zu werden.", "den UEFA-CUP zu erreichen.", "DIE KLASSE ZU ERHALTEN."}
	won := [3]string{"die MEISTERSCHAFT", "den UEFA-CUP-PLATZ", "den KLASSENERHALT"}
	if l > 0 {
		lost[1] = "AUFZUSTEIGEN."
		won[1] = "den AUFSTIEG"
	}
	for k := 0; k < 3; k++ {
		ref := s.Tables[l][idx[k]]
		show := l == 0 || k != 0
		if maxpts(me) < pts(ref) && m.GoalFlags&(1<<k) == 0 {
			m.GoalFlags |= 1 << k
			if show {
				s.notify(mi, club+" hat die letzte Chance verspielt, "+lost[k])
			}
		}
		best := 0
		for i := idx[k] + 1; i < n; i++ {
			best = max(best, maxpts(s.Tables[l][i]))
		}
		if best < pts(me) && m.GoalFlags&(1<<(k+3)) == 0 {
			m.GoalFlags |= 1 << (k + 3)
			if show {
				s.notify(mi, club+" hat "+won[k]+" auf jeden Fall gesichert.")
			}
		}
	}
}
