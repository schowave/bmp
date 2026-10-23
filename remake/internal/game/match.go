package game

import "strconv"

// EventKind is what happened in a match minute.
type EventKind int

// Event kinds.
const (
	EvGoal EventKind = iota
	EvMiss
	EvYellow
	EvYellowRed
	EvRed
	EvInjury
	EvPenaltyGoal
	EvPenaltyMiss
	EvForfeit
)

// MatchEvent is shown in the live match screens.
type MatchEvent struct {
	Minute  int
	Fixture int // index into MatchDay.Fixtures
	Kind    EventKind
	Side    int    // 0 home, 1 away
	Player  string // scorer, booked or injured player (human teams only)
	Assist  string
	Text    string
	Manager int // manager concerned, -1 for AI teams
}

// Fixture is one match of a match day.
type Fixture struct {
	League     int // 0..2, or -1 for a cup match
	MD         int // matchday of a league match (1-based)
	Cup        Cup // cup of cup matches
	Pair       int // pair index (league) or slot index of the home team (cup)
	Home, Away int
	Chances    [2][8]int // minutes of the planned chances
	Forfeit    int       // 1 home forfeits, 2 away forfeits
	// Final keeps the score of a cup match: the cup code clears the
	// results when the next round is drawn.
	Final      [2]int
	HasFinal   bool
	Attendance int64
	Watched    bool // a human team plays
}

// humanMatch is the per-manager state during a match (F:0x0178 etc.).
type humanMatch struct {
	plays        bool
	fixture      int
	side         int
	yellowBudget int
	redPossible  bool
	booked       []int
	keeperSubs   int
	fieldSubs    int
	planned      [2]int
	played       [2]int
}

// MatchDay plays all matches of one event of the calendar: a league
// matchday, a cup round or replays.
type MatchDay struct {
	s        *State
	Fixtures []*Fixture
	Events   []MatchEvent
	Minute   int
	Done     bool
	Title    string

	leagueMask int
	cupMask    int
	replay     bool
	periods    [][2]int
	period     int
	humans     []humanMatch
	extra      bool
	pens       bool
	onFinish   func()
	started    int // number of periods started
	ended      int // last minute of the period the last Step ended, 0 = none
}

func (md *MatchDay) score(f *Fixture) (int, int) {
	s := md.s
	if f.HasFinal {
		return f.Final[0], f.Final[1]
	}
	if f.League >= 0 {
		r := s.Results[f.League][f.MD-1][f.Pair]
		return r.Home, r.Away
	}
	return s.Cups.Result[f.Cup][f.Pair], s.Cups.Result[f.Cup][f.Pair+1]
}

func (md *MatchDay) setScore(f *Fixture, h, a int) {
	s := md.s
	if f.League >= 0 {
		s.Results[f.League][f.MD-1][f.Pair] = Score{h, a}
		return
	}
	s.Cups.Result[f.Cup][f.Pair] = h
	s.Cups.Result[f.Cup][f.Pair+1] = a
}

// Score returns the current score of a fixture without cup markers.
func (md *MatchDay) Score(i int) (int, int) {
	h, a := md.score(md.Fixtures[i])
	if h == Postponed && md.Fixtures[i].League >= 0 {
		return -1, -1
	}
	return ResultOf(h), a
}

// Penalties reports whether the fixture was decided by a shoot-out, and
// ExtraTime whether it went to extra time.
func (md *MatchDay) Penalties(i int) bool {
	h, _ := md.score(md.Fixtures[i])
	return md.Fixtures[i].League < 0 && h >= markPenalties
}

// ExtraTime reports whether a cup match went to extra time.
func (md *MatchDay) ExtraTime(i int) bool {
	h, _ := md.score(md.Fixtures[i])
	return md.Fixtures[i].League < 0 && h >= markExtraTime
}

// newMatchDay prepares the matches of an event. leagueMask has a bit per
// league, cupMask a bit per cup.
func (s *State) newMatchDay(leagueMask, cupMask int, replay bool) *MatchDay {
	md := &MatchDay{s: s, leagueMask: leagueMask, cupMask: cupMask, replay: replay}
	md.humans = make([]humanMatch, len(s.Managers))
	md.periods = [][2]int{{1, 45}, {46, 90}}
	s.CupDay = cupMask != 0
	if replay {
		md.Title = "Nachholspiele"
		for r := range s.Postponed {
			p := &s.Postponed[r]
			if p.Slot != s.Event || p.Matchday == 0 {
				continue
			}
			pairs := Fixtures(p.League, p.Matchday)
			md.Fixtures = append(md.Fixtures, &Fixture{League: p.League, MD: p.Matchday, Pair: p.Pair,
				Home: pairs[p.Pair].Home, Away: pairs[p.Pair].Away})
		}
	}
	for l := 0; l < NumLeagues && !replay; l++ {
		if leagueMask&(1<<l) == 0 {
			continue
		}
		pairs := Fixtures(l, s.Matchday[l])
		res := s.Results[l][s.Matchday[l]-1][:len(pairs)]
		s.snapshotMatchday(pairs, res)
		for i, p := range pairs {
			if res[i].Home == Postponed {
				continue
			}
			res[i] = Score{}
			md.Fixtures = append(md.Fixtures, &Fixture{League: l, MD: s.Matchday[l], Pair: i, Home: p.Home, Away: p.Away})
		}
	}
	for c := CupDFB; c < NumCups; c++ {
		if cupMask&(1<<c) == 0 {
			continue
		}
		for i, p := range s.CupPairs(c) {
			md.Fixtures = append(md.Fixtures, &Fixture{League: -1, Cup: c, Pair: 2 * i, Home: p.Home, Away: p.Away})
			s.Cups.Result[c][2*i], s.Cups.Result[c][2*i+1] = 0, 0
		}
	}
	if replay {
		// Replays keep their own result rows; start from 0:0.
		for _, f := range md.Fixtures {
			s.Results[f.League][f.MD-1][f.Pair] = Score{}
		}
	}
	for i, f := range md.Fixtures {
		md.prepare(i, f)
	}
	return md
}

// prepare handles a match before kick-off for the human managers: forfeit,
// attendance and gate money, squad updates (f_1a58_20b2).
func (md *MatchDay) prepare(fi int, f *Fixture) {
	s := md.s
	comp := 0
	if f.League < 0 {
		comp = 1
		if f.Cup > 0 {
			comp = 2
		}
	}
	split := 1
	if comp != 0 && f.Cup == 0 {
		split = 2
	}
	final := comp == 1 && s.Cups.Round[CupDFB] > 4
	d := s.Date()
	exempt := d.Month == 5 && d.Day >= 10
	var homeAtt int64
	homePrice := 0
	// The home team's manager first: the away manager of a DFB-Pokal tie
	// shares his gate.
	var order []int
	for _, side := range []int{f.Home, f.Away} {
		for mi, m := range s.Managers {
			if m.Team == side {
				order = append(order, mi)
			}
		}
	}
	for _, mi := range order {
		m := s.Managers[mi]
		side := 0
		if m.Team == f.Away {
			side = 1
		}
		if m.Effort == ForfeitEffort && !exempt {
			// f_1a58_2051: the match is not played at all.
			s.alert(mi, BackdropNone, msgForfeit(m.Name))
			m.Money -= 200000
			f.Forfeit = side + 1
			f.Watched = false
			for k := range md.humans {
				if md.humans[k].plays && md.humans[k].fixture == fi {
					md.humans[k].plays = false
				}
			}
			return
		}
		f.Watched = true
		hm := &md.humans[mi]
		hm.plays, hm.fixture, hm.side = true, fi, side
		if final {
			f.Attendance = 76000
			m.Money += 532000
		} else if side == 0 {
			imp := 1
			if comp > 1 {
				imp = 2
				if s.Cups.Round[CupLandesmeister] > 4 {
					imp = 3
				}
			}
			att := s.Attendance(mi, f.Home, f.Away, imp, false, 0)
			if comp == 0 {
				if m.HomeGames < len(m.Attendance) {
					m.Attendance[m.HomeGames] = int(att / 1000)
				}
				m.SeasonAudience += att
				if att >= m.RecordAudience {
					m.RecordAudience, m.RecordOpponent = att, f.Away
				}
				if att <= m.MinAudience {
					m.MinAudience, m.MinOpponent = att, f.Away
				}
				m.HomeGames++
			} else if b := bucket(f.Away); b < m.League {
				att += int64(s.Rand.Between16(int(att)/(8-3*(m.League-b)), int(att)))
				if att > m.Capacity() {
					att = m.Capacity()
				}
			}
			m.Money += att * int64(m.TicketPrice) / int64(split)
			homeAtt, homePrice = att, m.TicketPrice
			f.Attendance = att
		} else if split == 2 {
			if homeAtt == 0 {
				price := AITicket[bucket(f.Home)] + s.Rand.Between(0, 1)
				att := s.Attendance(mi, f.Home, f.Away, 1, true, price)
				m.Money += int64(price) * att / 2
				f.Attendance = att
			} else {
				m.Money += homeAtt * int64(homePrice) / 2
			}
		}
		md.squadBeforeMatch(mi, comp)
		if side == 0 {
			// Crowd trouble (the original uses the low words of the longs).
			x := clamp(int(int16(int16(m.Stadium[StStanding])-int16(m.Stadium[StSeats])))/1000, 0, 30)
			if s.Rand.Between(0, (30-x)*4+15) == 0 {
				m.RiotPending = true
			}
		}
	}
}

// squadBeforeMatch updates suspensions, appearances and fatigue (§6.3 of
// the match spec).
func (md *MatchDay) squadBeforeMatch(mi, comp int) {
	s := md.s
	for i := 0; i < MarketSlots; i++ {
		sl := &s.Squads[MarketIndex][i]
		if !sl.Empty() && s.Persons[sl.Person].Owner == mi && sl.Status&1 != 0 && sl.Out > 0 && comp == 0 {
			sl.Out--
		}
	}
	for i := 0; i < s.countLineup(mi, 0); i++ {
		sl := &s.Squads[mi][i]
		if sl.Status&1 != 0 && sl.Out > 0 && comp == 0 {
			sl.Out--
		}
		sl.Rating = 0
		if sl.Lineup >= 1 && sl.Lineup <= 11 {
			p := &s.Persons[sl.Person]
			k := boolInt(p.Skill[0] > p.Skill[1])
			sl.Fatigue += s.Rand.Between(2, 4) - k + 6
			if comp == 0 {
				p.LeagueGames++
			}
			ci := boolInt(comp >= 1) + boolInt(comp > 1)
			sl.Games[ci]++
			sl.Career[ci]++
		}
		sl.Fatigue = clamp(sl.Fatigue, 50, 150)
	}
	hm := &md.humans[mi]
	if s.countLineup(mi, 0) < 6 || comp != 0 {
		hm.yellowBudget, hm.redPossible = 0, false
	} else {
		hm.yellowBudget, hm.redPossible = 6, true
	}
	hm.keeperSubs, hm.fieldSubs = 1, 2
	hm.booked = nil
}

// strengthAt is S(t, kind, minute) of f_0310_1468: the Technik part wears
// off during the match.
func (s *State) strengthAt(t, kind, minute int) int {
	tm := &s.Teams[t]
	v := 0
	for i := 0; i < 3; i++ {
		b := tm.Base[i]
		x := b + (10*minute)/(-1-b)
		if x < 0 {
			x = 0
		}
		v += (x + tm.Strength[i] + tm.Extra[i]) * strengthWeights[kind][i] / 100
	}
	return v + tm.Mood - 44
}

var chanceSpread = [20]int{1, 2, 2, 2, 2, 2, 3, 3, 3, 3, 3, 3, 3, 3, 4, 4, 4, 4, 4, 4} // D:0x2aa

// planChances computes the number of chances of both sides in a period
// (f_0f9d_08e9).
func (md *MatchDay) planChances(f *Fixture, from, to int) (int, int) {
	r := &md.s.Rand
	s := md.s
	pick := func(x, y int) int {
		if r.Between(0, 1) == 1 && x > y {
			return x
		}
		return y
	}
	h, a := 0, 0
	h += r.Between(0, 1)
	sh, sa := s.strengthAt(f.Home, 1, from), s.strengthAt(f.Away, 1, from)
	d := sa/(-3) + sh/3
	if d > 0 {
		n := d / 5
		if d <= 19 {
			n = chanceSpread[d]
		}
		x, y := r.Between(1, n), r.Between(1, n)
		if r.Between(0, 1) != 0 && d > 8 {
			h += pick(x, y)
		} else {
			h = x
		}
		if d < 25 {
			a = r.Between(0, 2)
		}
	} else {
		e := -d
		n := e / 5
		if e <= 19 {
			n = chanceSpread[e]
		}
		if e < 25 {
			h += r.Between(0, 2)
		}
		x, y := r.Between(1, n), r.Between(1, n)
		if r.Between(0, 2) == 0 && e > 8 {
			a += pick(x, y)
		} else {
			a = x
		}
	}
	a += r.Between(0, 2)
	h += r.Between(0, 3)
	for h > 8 || a > 8 {
		h--
		a--
	}
	if h > 4 && r.Between(0, 1) != 0 {
		h--
	}
	if a > 4 && r.Between(0, 1) != 0 {
		a--
	}
	if (h > 2 || a > 2) && r.Between(0, 1) != 0 {
		h--
		a--
	}
	h, a = max(h, 0), max(a, 0)
	if sh == 0 {
		h = 0
		if sa > 20 {
			a = 8
		}
	}
	if sa == 0 {
		a = 0
		if sh > 20 {
			h = 8
		}
	}
	h = (to - from) * h / 45
	a = (to - from) * a / 45
	if from > 90 {
		h++
		a++
	}
	return h, a
}

// assignMinutes spreads cnt chances over the minutes of a period
// (f_0310_12fe). In cups a minute may repeat, like in the original.
func (md *MatchDay) assignMinutes(f *Fixture, side, cnt, from, to int) {
	r := &md.s.Rand
	f.Chances[side] = [8]int{}
	if cnt > 8 {
		cnt = 8
	}
	for i := 0; i < cnt; i++ {
		for try := 0; try < 100; try++ {
			mm := r.Between(from, to)
			dup := false
			if f.League >= 0 {
				for j := 0; j < i; j++ {
					if f.Chances[side][j] == mm {
						dup = true
						break
					}
				}
			}
			if !dup {
				f.Chances[side][i] = mm
				break
			}
		}
	}
}

// Step plays the next minute. It returns false when the match day is over.
// The step after the last minute ends the match day, so that what is shown
// after the last period still sees the cups before the winners advance.
func (md *MatchDay) Step() bool {
	if md.Done {
		return false
	}
	if md.period >= len(md.periods) {
		md.ended = 0
		md.finish()
		return false
	}
	from, to := md.periods[md.period][0], md.periods[md.period][1]
	if md.started <= md.period {
		md.started = md.period + 1
		md.startPeriod(from, to)
		md.Minute = from
	}
	md.playMinute(md.Minute)
	md.ended = 0
	if md.Minute == to {
		md.ended = to
		md.endPeriod(to)
		md.period++
		if md.period >= len(md.periods) {
			md.afterRegularTime()
		}
	}
	md.Minute++
	return true
}

// Run plays the whole match day.
func (md *MatchDay) Run() {
	for md.Step() {
	}
}

func (md *MatchDay) startPeriod(from, to int) {
	for mi := range md.humans {
		md.humans[mi].played = [2]int{}
	}
	for _, f := range md.Fixtures {
		if f.Forfeit != 0 {
			continue
		}
		if f.League >= 0 {
			if h, _ := md.score(f); h == Postponed {
				continue
			}
		}
		h, a := md.planChances(f, from, to)
		md.assignMinutes(f, 0, h, from, to)
		md.assignMinutes(f, 1, a, from, to)
		for mi := range md.humans {
			if md.humans[mi].plays && md.Fixtures[md.humans[mi].fixture] == f {
				md.humans[mi].planned = [2]int{h, a}
			}
		}
	}
}

func (md *MatchDay) playMinute(minute int) {
	for fi, f := range md.Fixtures {
		if f.Forfeit != 0 {
			continue
		}
		if h, _ := md.score(f); f.Watched && (minute <= 90 || h >= markExtraTime) {
			md.humanEvents(fi, f, minute)
		}
		for j := 0; j < 8; j++ {
			for side := 0; side < 2; side++ {
				if f.Chances[side][j] == minute {
					md.chance(fi, f, side, minute)
					if f.League < 0 {
						f.Chances[side][j] = 0
					}
				}
			}
		}
	}
}

// chance resolves a chance (f_0f9d_0c3b).
func (md *MatchDay) chance(fi int, f *Fixture, side, minute int) {
	s := md.s
	r := &s.Rand
	dA := (s.strengthAt(f.Away, 2, minute) - s.strengthAt(f.Home, 0, minute)) / 3
	dH := (s.strengthAt(f.Home, 2, minute) - s.strengthAt(f.Away, 0, minute)) / 3
	hs, as := md.score(f)
	if minute > 90 && hs < markExtraTime {
		return
	}
	own, dd := hs, dH
	if side == 1 {
		own, dd = as, dA
	}
	goal := false
	if !(own == 9 || own >= 19) {
		var pass bool
		if side == 0 {
			pass = r.Between(0, 4) >= 2
		} else {
			pass = r.Between(0, 1) != 0
		}
		if pass {
			if dd > 0 {
				goal = r.Between(0, (dd>>2)+2) != 0
			} else {
				k := (-dd) >> 2
				goal = r.Between(0, k) == 0 || r.Between(0, 6) == 0
			}
		}
	}
	if goal {
		if side == 0 {
			hs++
		} else {
			as++
		}
		md.setScore(f, hs, as)
	}
	md.chanceEvent(fi, f, side, minute, goal, false)
}

// chanceEvent records a chance of a watched match and updates the player
// statistics of human teams (f_1a58_0ca3 types 0 and 1).
func (md *MatchDay) chanceEvent(fi int, f *Fixture, side, minute int, goal, penalty bool) {
	s := md.s
	ev := MatchEvent{Minute: minute, Fixture: fi, Side: side, Manager: -1}
	switch {
	case goal && penalty:
		ev.Kind = EvPenaltyGoal
	case penalty:
		ev.Kind = EvPenaltyMiss
	case goal:
		ev.Kind = EvGoal
	default:
		ev.Kind = EvMiss
	}
	att, def := f.Home, f.Away
	if side == 1 {
		att, def = def, att
	}
	comp := 0
	if f.League < 0 {
		comp = 1
		if f.Cup > 0 {
			comp = 2
		}
	}
	for mi := range md.humans {
		hm := &md.humans[mi]
		if !hm.plays || md.Fixtures[hm.fixture] != f {
			continue
		}
		hm.played[side]++
	}
	if am := s.ManagerOf(att); am != nil && f.Watched {
		mi := s.managerIndex(am)
		ev.Manager = mi
		sc := s.pickScorer(mi, 0, boolInt(goal))
		asIdx := -1
		if s.countLineup(mi, 1) > 1 && sc >= 0 {
			for {
				asIdx = s.pickScorer(mi, 1, 1)
				if asIdx != sc {
					break
				}
			}
		}
		if sc >= 0 {
			sl := &s.Squads[mi][sc]
			ev.Player = s.Persons[sl.Person].Name
			if goal && !penalty {
				if comp == 0 {
					s.Persons[sl.Person].LeagueGoals++
				}
				sl.CareerGoals[comp]++
				sl.Goals[comp]++
				sl.Rating += 15
			} else if !goal {
				sl.Rating -= 10
			}
			if asIdx >= 0 && !penalty {
				as := &s.Squads[mi][asIdx]
				ev.Assist = s.Persons[as.Person].Name
				as.Rating += 10
			}
		}
	} else if dm := s.ManagerOf(def); dm != nil && f.Watched {
		mi := s.managerIndex(dm)
		for i := 0; i < s.countLineup(mi, 0); i++ {
			sl := &s.Squads[mi][i]
			if sl.Lineup < 1 || sl.Lineup > 11 {
				continue
			}
			keeper := s.Persons[sl.Person].Pos == 0
			switch {
			case goal && keeper:
				sl.Rating -= 8
			case goal:
				sl.Rating -= 5
			case keeper:
				sl.Rating += 8
			default:
				sl.Rating -= 3
			}
		}
	}
	if goal || f.Watched {
		md.Events = append(md.Events, ev)
	}
}

// pickScorer chooses the scorer (kind 0) or the assistant (kind 1) of a
// chance of manager mi's team (f_0310_2c9a).
func (s *State) pickScorer(mi, kind, typ int) int {
	r := &s.Rand
	if s.countLineup(mi, 1) == 0 {
		return -1
	}
	n := s.countLineup(mi, 0)
	for {
		i := r.Between(0, n-1)
		sl := &s.Squads[mi][i]
		if sl.Lineup < 1 || sl.Lineup > 11 {
			continue
		}
		k, t, p := sl.Skill[0], sl.Skill[1], sl.Skill[2]
		var w int
		if kind != 0 || typ != 0 {
			w = (3*t)/(5-3*kind) + k/5 + p/5
		} else {
			w = r.Between(120, 170)
		}
		pos := s.Persons[sl.Person].Pos
		w += (pos / 25) * (6 - kind) * 7
		w += 46 * sl.Goals[0]
		w += boolInt(kind == 0)*300 + (7-abs(sl.Row))*200
		if sl.Row < 2 {
			w += 1200
		}
		w += (abs(sl.Col-3) - 2*boolInt(kind == 0)) * 5
		rm, pm := s.sideDiff(sl), s.posDiff(sl)
		if kind != 0 || typ != 0 {
			w += (5-rm)*5 + (35 - pm)
		} else {
			w += rm*5 + pm
		}
		ok := r.Between(0, 3500) < w
		if pos == 0 && r.Between(0, 20) != 0 {
			ok = false
		}
		if ok {
			return i
		}
	}
}

// humanEvents creates cards and injuries for the human teams of a fixture
// (f_0310_2ee5).
func (md *MatchDay) humanEvents(fi int, f *Fixture, minute int) {
	s := md.s
	r := &s.Rand
	g := s.Level
	replan := -1
	for mi := range md.humans {
		hm := &md.humans[mi]
		if !hm.plays || md.Fixtures[hm.fixture] != f {
			continue
		}
		if s.countLineup(mi, 1) < 4 {
			continue
		}
		m := s.Managers[mi]
		h := clamp(40-m.Effort, 0, 40)
		if hm.redPossible && r.Between(0, (g+5)*50+70*h) == 0 {
			hm.redPossible = false
			if p := s.pickVictim(mi); p >= 0 {
				md.sendOff(fi, mi, p, minute, false)
				replan = mi
			}
		}
		if hm.yellowBudget > 0 && r.Between(0, 5*g+2*h+37) == 0 {
			hm.yellowBudget--
			var p int
			booked := false
			for {
				p = s.pickVictim(mi)
				booked = false
				for _, b := range hm.booked {
					if b == p {
						booked = true
					}
				}
				if hm.redPossible || !booked || p < 0 {
					break
				}
			}
			if p >= 0 {
				if booked {
					hm.redPossible = false
					md.sendOff(fi, mi, p, minute, true)
				} else {
					hm.booked = append(hm.booked, p)
					sl := &s.Squads[mi][p]
					sl.Rating -= 10
					sl.Cards[1]++
					md.Events = append(md.Events, MatchEvent{Minute: minute, Fixture: fi, Kind: EvYellow,
						Side: hm.side, Player: s.Persons[sl.Person].Name, Manager: mi})
				}
			}
		}
		if r.Between(0, 55*h+60*g+200) == 0 {
			if p := s.pickVictim(mi); p >= 0 {
				sl := &s.Squads[mi][p]
				s.injure(sl)
				md.Events = append(md.Events, MatchEvent{Minute: minute, Fixture: fi, Kind: EvInjury,
					Side: hm.side, Player: s.Persons[sl.Person].Name, Text: InjuryNames[sl.Injury], Manager: mi})
				s.LineupStrength(mi, true)
				replan = mi
			}
		}
	}
	if replan >= 0 {
		md.replan(replan, minute)
	}
}

// pickVictim chooses the player who gets a card or an injury (f_0310_1d45).
func (s *State) pickVictim(mi int) int {
	r := &s.Rand
	n := s.countLineup(mi, 0)
	if s.countLineup(mi, 1) == 0 {
		return -1
	}
	for {
		i := r.Between(0, n-1)
		sl := &s.Squads[mi][i]
		if sl.Lineup < 1 || sl.Lineup > 11 || sl.Status&3 != 0 {
			continue
		}
		d := sl.Skill[1] - sl.Skill[0]
		if d > 0 {
			if d > r.Between(0, 3) {
				return i
			}
		} else if r.Between(0, 10) == 0 {
			return i
		}
	}
}

func (md *MatchDay) sendOff(fi, mi, p, minute int, yellowRed bool) {
	s := md.s
	sl := &s.Squads[mi][p]
	kind := EvRed
	if yellowRed {
		kind = EvYellowRed
		sl.Cards[2]++
		sl.Cards[1]--
		sl.Out = s.Rand.Between(1, 1)
	} else {
		sl.Cards[0]++
		sl.Out = s.Rand.Between(1, 7)
	}
	sl.Lineup = 0
	sl.Status |= 1
	sl.Rating -= 10
	md.Events = append(md.Events, MatchEvent{Minute: minute, Fixture: fi, Kind: kind,
		Side: md.humans[mi].side, Player: s.Persons[sl.Person].Name, Manager: mi})
	s.LineupStrength(mi, true)
}

// replan reschedules the remaining chances of a human's match after a red
// card, an injury or a substitution (f_0310_347f).
func (md *MatchDay) replan(mi, minute int) {
	hm := &md.humans[mi]
	if !hm.plays || md.period >= len(md.periods) {
		return
	}
	f := md.Fixtures[hm.fixture]
	from, to := md.periods[md.period][0], md.periods[md.period][1]
	n0, n1 := md.planChances(f, from, to)
	n := [2]int{n0, n1}
	for i := 0; i < 2; i++ {
		p := hm.planned[i]
		if p <= n[i] {
			n[i] -= p
		}
		p -= hm.played[i]
		p += n[i]
		if p < 0 {
			p = 0
		}
		md.assignMinutes(f, i, p, minute, to)
	}
}

// CanSubstitute checks a click on slot i of the lineup during a match
// (f_1ecd_1560 with N players): the player must be in the team or a
// reserve, and a substitution of his kind (keeper or field player) must be
// left.
func (md *MatchDay) CanSubstitute(mi, i int) error {
	s := md.s
	sl := &s.Squads[mi][i]
	if !md.humans[mi].plays || md.Done || sl.Empty() || sl.Lineup == 0 {
		return ErrNotOnPitch
	}
	if !md.subsLeft(mi, s.Persons[sl.Person].Position() == PosKeeper) {
		return ErrNoSubs
	}
	return nil
}

func (md *MatchDay) subsLeft(mi int, keeper bool) bool {
	hm := &md.humans[mi]
	return keeper && hm.keeperSubs > 0 || !keeper && hm.fieldSubs > 0
}

// Substitute exchanges a player on the pitch for one from the bench during
// a match (f_1ecd_1560 with N players).
func (md *MatchDay) Substitute(mi, out, in int) error {
	s := md.s
	hm := &md.humans[mi]
	if !hm.plays || md.Done {
		return ErrNotOnPitch
	}
	o, n := &s.Squads[mi][out], &s.Squads[mi][in]
	if o.Lineup < 1 || o.Lineup > 11 {
		return ErrNotOnPitch
	}
	keeper := s.Persons[o.Person].Position() == PosKeeper
	if !md.subsLeft(mi, keeper) {
		return ErrNoSubs
	}
	if n.Empty() || n.Lineup >= 1 && n.Lineup <= 11 {
		return ErrNotOnPitch
	}
	if !s.available(n) {
		if n.Status&3 == 1 {
			return ErrSuspendedShort
		}
		return ErrInjuredShort
	}
	if keeper {
		hm.keeperSubs--
	} else {
		hm.fieldSubs--
	}
	n.Lineup, o.Lineup = o.Lineup, 0
	n.Row, n.Col = o.Row, o.Col
	n.Fatigue += s.Rand.Between(2, 4) + 4
	comp := 0
	if f := md.Fixtures[hm.fixture]; f.League < 0 {
		comp = 1 + boolInt(f.Cup > 0)
	}
	n.Games[comp]++
	n.Career[comp]++
	s.Managers[mi].Subs++
	s.LineupStrength(mi, true)
	md.replan(mi, md.Minute)
	return nil
}

func (md *MatchDay) endPeriod(to int) {
	s := md.s
	for _, f := range md.Fixtures {
		switch f.Forfeit {
		case 1:
			md.setScore(f, 0, 2)
		case 2:
			md.setScore(f, 2, 0)
		}
	}
	if to != 90 {
		return
	}
	if md.replay {
		for _, f := range md.Fixtures {
			md.applyLeague(f, f.MD)
		}
		return
	}
	for l := 0; l < NumLeagues; l++ {
		if md.leagueMask&(1<<l) == 0 {
			continue
		}
		pairs := Fixtures(l, s.Matchday[l])
		res := s.Results[l][s.Matchday[l]-1][:len(pairs)]
		s.applyResults(pairs, res, true)
		for i, p := range pairs {
			if res[i].Home != Postponed {
				s.moodAfterMatch(p.Home, res[i].Home-res[i].Away)
				s.moodAfterMatch(p.Away, res[i].Away-res[i].Home)
			}
		}
		s.sortTable(l)
		s.aiScorers(l, pairs, res)
	}
}

// applyLeague adds a replayed game to the table of its league.
func (md *MatchDay) applyLeague(f *Fixture, mdNum int) {
	s := md.s
	r := s.Results[f.League][mdNum-1][f.Pair]
	pair := []Pair{{f.Home, f.Away}}
	s.Clubs[f.Home].snapshot()
	s.Clubs[f.Away].snapshot()
	s.applyResults(pair, []Score{r}, true)
	s.moodAfterMatch(f.Home, r.Home-r.Away)
	s.moodAfterMatch(f.Away, r.Away-r.Home)
	// The original plays replays as league 0: only that table is sorted
	// now, the others at their next matchday.
	s.sortTable(0)
	s.aiScorersOf(f.Home, r.Home)
	s.aiScorersOf(f.Away, r.Away)
}

// moodAfterMatch changes the mood of a club after a league match
// (f_2a41_1fec).
func (s *State) moodAfterMatch(t, gd int) {
	if t >= NumGerman {
		return
	}
	sign := -1
	if gd > 0 {
		sign = 1
	}
	s.Teams[t].Mood = clamp(s.Teams[t].Mood+sign*s.Rand.Between(0, 2*abs(gd)), 47, 53)
}

var aiGoalTable = [10]int{0, 2, 3, 4, 5, 6, 7, 8, 9, 10} // D:0x5331

// aiScorers attributes goals of clubs without a human manager to persons
// for the top-scorer list (f_14a4_1662).
func (s *State) aiScorers(l int, pairs []Pair, res []Score) {
	for i, p := range pairs {
		if res[i].Home == Postponed {
			continue
		}
		s.aiScorersOf(p.Home, res[i].Home)
		s.aiScorersOf(p.Away, res[i].Away)
	}
}

func (s *State) aiScorersOf(team, goals int) {
	if s.ManagerOf(team) != nil {
		return
	}
	r := &s.Rand
	var cand []int
	for p := 1; p < NumPersons; p++ {
		ps := &s.Persons[p]
		if ps.Team != team {
			continue
		}
		if r.Between(0, 100) < 94 {
			ps.LeagueGames++
			if r.Between(10, 99) < r.Between(5, 25)+ps.Pos {
				cand = append(cand, p)
			}
		}
	}
	if len(cand) <= 1 {
		return
	}
	for g := 1; g <= goals; g++ {
		lim := 10
		if len(cand) < 10 {
			lim = aiGoalTable[len(cand)] - 1
		}
		if lim > 2 {
			lim--
		}
		if r.Between(0, 4) == 0 {
			continue
		}
		for {
			if r.Between(0, lim) != 0 {
				s.Persons[cand[r.Between(0, len(cand)-1)]].LeagueGoals++
				break
			}
			if !(r.Between(0, 1)+2 < len(cand)) {
				break
			}
		}
	}
}

// afterRegularTime adds extra time or a shoot-out for undecided cup ties.
func (md *MatchDay) afterRegularTime() {
	s := md.s
	if md.cupMask == 0 {
		return
	}
	if !md.extra {
		md.extra = true
		any := false
		for c := CupDFB; c < NumCups; c++ {
			if md.cupMask&(1<<c) != 0 && len(s.needsExtraTime(c, md.cupPairs(c), 0)) > 0 {
				any = true
			}
		}
		if any {
			md.periods = append(md.periods, [2]int{91, 105}, [2]int{106, 120})
		}
		return
	}
	if !md.pens {
		md.pens = true
		for c := CupDFB; c < NumCups; c++ {
			if md.cupMask&(1<<c) == 0 {
				continue
			}
			for _, i := range s.needsExtraTime(c, md.cupPairs(c), 1) {
				md.shootout(c, i)
			}
		}
	}
}

func (md *MatchDay) cupPairs(c Cup) int {
	n := 0
	for _, f := range md.Fixtures {
		if f.League < 0 && f.Cup == c {
			n += 2
		}
	}
	return n
}

// shootout decides a tie by penalties (f_0310_356d). The shoot-out score
// replaces the match score, marked with +20.
func (md *MatchDay) shootout(c Cup, slot int) {
	s := md.s
	r := &s.Rand
	var f *Fixture
	fi := -1
	for i, x := range md.Fixtures {
		if x.League < 0 && x.Cup == c && x.Pair == slot {
			f, fi = x, i
		}
	}
	if f == nil {
		return
	}
	if !f.Watched {
		var h, a int
		for {
			h, a = r.Between(2, 5), r.Between(2, 5)
			if h != a {
				break
			}
		}
		md.setScore(f, h+markPenalties, a)
		return
	}
	h, a := 0, 0
	first := r.Between(0, 1)
	side, rounds := first, 0
	for rounds < 5 || h == a || side != first {
		goal := min(r.Between(0, 2), 1) == 1
		md.chanceEvent(fi, f, side, 120, goal, true)
		if goal {
			if side == 0 {
				h++
			} else {
				a++
			}
		}
		side ^= 1
		if side == first {
			rounds++
		}
	}
	md.setScore(f, h+markPenalties, a)
}

// finish ends the match day: cup winners advance.
func (md *MatchDay) finish() {
	if md.Done {
		return
	}
	md.Done = true
	s := md.s
	for _, f := range md.Fixtures {
		if f.League < 0 {
			f.Final[0], f.Final[1] = md.score(f)
			f.HasFinal = true
		}
	}
	for c := CupDFB; c < NumCups; c++ {
		if md.cupMask&(1<<c) != 0 {
			s.advanceCup(c)
		}
	}
	if md.cupMask&0x0e != 0 && !s.Cups.PlayingRelegation {
		s.afterEuropeanRound()
	}
	if md.onFinish != nil {
		md.onFinish()
	}
	s.CupDay = false
}

// EventText describes an event like the match screens of the original.
func (md *MatchDay) EventText(e MatchEvent) string {
	switch e.Kind {
	case EvGoal:
		t := "Torschütze: " + e.Player
		if e.Assist != "" {
			t += " nach Vorlage von " + e.Assist
		}
		return t
	case EvMiss:
		return "Chance vergeben: " + e.Player
	case EvYellow:
		return "Gelbe Karte: " + e.Player
	case EvYellowRed:
		return "Gelb-Rote Karte: " + e.Player
	case EvRed:
		return "Rote Karte: " + e.Player
	case EvInjury:
		return "Verletzt: " + e.Player + " (" + e.Text + ")"
	case EvPenaltyGoal:
		return "Elfmeter: Tor"
	case EvPenaltyMiss:
		return "Elfmeter: verschossen"
	}
	return strconv.Itoa(int(e.Kind))
}
