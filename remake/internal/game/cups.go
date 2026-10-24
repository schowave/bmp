package game

// Cup is one of the four cups, in the order of the original.
type Cup int

// The cups: 0 DFB-Pokal, 1 Pokal der Landesmeister, 2 Pokal der
// Pokalsieger, 3 UEFA-Pokal.
const (
	CupDFB Cup = iota
	CupLandesmeister
	CupPokalsieger
	CupUEFA
)

// NumCups is the number of cups.
const NumCups = 4

// CupNames are the texts of the original.
var CupNames = [NumCups]string{"DfB-Pokal", "Pokal der Landesmeister", "Pokal der Pokalsieger", "UEFA-Pokal"}

// RoundNames are the round texts (index = round 1..5).
var RoundNames = [6]string{"", "1.Runde", "Achtelfinale", "Viertelfinale", "Halbfinale", "Endspiel"}

// pairsPerRound is D:0x07d6.
var pairsPerRound = [6]int{32, 16, 8, 4, 2, 1}

// NotInCup is the cup status of a manager whose club is not taking part.
const NotInCup = 0x1e

// Score markers on the home goals of a cup result.
const (
	markExtraTime = 10
	markPenalties = 20
)

// Cups is the state of the four cup competitions.
type Cups struct {
	Round  [NumCups]int     // F:0x0008: 1..5
	Slots  [NumCups][32]int // F:0x76ca: team ids, pair k = slots 2k (home), 2k+1
	Result [NumCups][32]int // F:0x4c7e: goals of the current leg, with markers
	First  [NumCups][32]int // F:0x21c2: first-leg goals, stored per slot
	Leg    [NumCups]int     // F:0x1d17: 1 = the next game is a second leg

	// Holder (team id + 1, 0 = none): D:0x07ac DFB winner, D:0x07ad..af
	// German winners of the European cups, D:0x07b0 losing DFB finalist.
	Holder      [NumCups]int
	DFBRunnerUp int

	// Qualified lists the German clubs for each European cup (F:0x57aa).
	Qualified [NumCups][]int
	Used      []int // F:0x56fe

	RelegationOutcome int // F:0x3060: 0 the Bundesliga club stayed up
	RelegationStash   [2]int
	PlayingRelegation bool // F:0x57dc
}

// ResultOf strips the extra-time and penalty markers like the original
// reader (while x > 9: x -= 10).
func ResultOf(x int) int {
	for x > 9 {
		x -= 10
	}
	return x
}

func (s *State) isManagerTeam(id int) bool { return s.ManagerOf(id) != nil }

func (c *Cups) isUsed(id int) bool {
	for _, u := range c.Used {
		if u == id {
			return true
		}
	}
	return false
}

// qualifyEurope fills the European qualifiers from the final Bundesliga
// table and the cup holders (f_17c2_0ef2).
func (s *State) qualifyEurope() {
	c := &s.Cups
	c.Used = nil
	for k := CupLandesmeister; k < NumCups; k++ {
		c.Qualified[k] = nil
	}
	add := func(cup Cup, id int) {
		c.Qualified[cup] = append(c.Qualified[cup], id)
		c.Used = append(c.Used, id)
	}
	add(CupLandesmeister, s.Tables[0][0])
	if c.Holder[CupLandesmeister] != 0 {
		if h := c.Holder[CupLandesmeister] - 1; !c.isUsed(h) {
			add(CupLandesmeister, h)
		} else {
			add(CupLandesmeister, s.Tables[0][1])
		}
	}
	if w := c.Holder[CupDFB] - 1; c.Holder[CupDFB] != 0 && !c.isUsed(w) {
		add(CupPokalsieger, w)
	} else if f := c.DFBRunnerUp - 1; c.DFBRunnerUp != 0 && !c.isUsed(f) {
		add(CupPokalsieger, f)
	}
	if c.Holder[CupPokalsieger] != 0 {
		if h := c.Holder[CupPokalsieger] - 1; !c.isUsed(h) {
			add(CupPokalsieger, h)
		} else if f := c.DFBRunnerUp - 1; c.DFBRunnerUp != 0 && !c.isUsed(f) {
			add(CupPokalsieger, f)
		}
	}
	places := UEFAPlaces
	if c.Holder[CupUEFA] != 0 {
		if h := c.Holder[CupUEFA] - 1; !c.isUsed(h) {
			add(CupUEFA, h)
		} else {
			places++
		}
	}
	added := 0
	for i := 1; i < TeamsPerLeague[0] && added < places; i++ {
		id := s.Tables[0][i]
		if c.isUsed(id) {
			continue
		}
		add(CupUEFA, id)
		added++
	}
}

// Qualified reports whether a German club plays in Europe next season
// (f_17c2_11e7).
func (s *State) QualifiedForEurope(id int) bool { return s.Cups.isUsed(id) }

// drawCup draws round 1 of a cup (f_17c2_09e0).
func (s *State) drawCup(cup Cup) {
	c := &s.Cups
	c.Round[cup] = 0
	r := &s.Rand
	if cup == CupLandesmeister {
		for k := CupLandesmeister; k < NumCups; k++ {
			c.First[k] = [32]int{}
		}
	}
	for _, m := range s.Managers {
		m.CupStatus[cup] = NotInCup
	}
	slots := &c.Slots[cup]
	if cup == CupDFB {
		taken := map[int]bool{}
		for _, m := range s.Managers {
			for {
				k := r.Between(0, 31)
				if !taken[k] {
					taken[k] = true
					slots[k] = m.Team
					m.CupStatus[cup] = 1
					break
				}
			}
		}
		for i := 0; i < 32; i++ {
			if taken[i] {
				continue
			}
			for {
				id := r.Between(0, 57)
				dup := s.isManagerTeam(id)
				for j := 0; j < i && !dup; j++ {
					dup = slots[j] == id
				}
				if !dup {
					slots[i] = id
					break
				}
			}
		}
		s.dfbHomeRule()
	} else {
		for i := 0; i < 32; i++ {
			for {
				id := r.Between(64, 199)
				dup := s.isManagerTeam(id)
				for j := 0; j < i && !dup; j++ {
					dup = slots[j] == id
				}
				for o := CupDFB; o < NumCups && !dup; o++ {
					if o == cup {
						continue
					}
					for _, x := range c.Slots[o] {
						if x == id {
							dup = true
							break
						}
					}
				}
				if !dup {
					slots[i] = id
					break
				}
			}
		}
		c.Leg[cup] = 0
		for _, q := range c.Qualified[cup] {
			for {
				k := r.Between(1, 31)
				if slots[k] >= NumGerman {
					slots[k] = q
					break
				}
			}
			if m := s.ManagerOf(q); m != nil {
				m.CupStatus[cup] = 1
			}
		}
	}
	c.Round[cup] = 1
}

// dfbHomeRule lets an Amateur-Oberliga club play at home against a club of
// the first two leagues in rounds 1..3 (f_17c2_1187).
func (s *State) dfbHomeRule() {
	c := &s.Cups
	if c.Round[CupDFB] >= 4 {
		return
	}
	sl := &c.Slots[CupDFB]
	for i := 1; i < 32; i += 2 {
		if LeagueMask(sl[i]) == 4 && LeagueMask(sl[i-1]) != 4 {
			sl[i], sl[i-1] = sl[i-1], sl[i]
		}
	}
}

// CupPairs returns the pairs of the current round of a cup.
func (s *State) CupPairs(cup Cup) []Pair {
	c := &s.Cups
	round := c.Round[CupDFB]
	if cup != CupDFB {
		round = c.Round[CupLandesmeister]
	}
	n := pairsPerRound[round]
	pairs := make([]Pair, n)
	for i := range pairs {
		pairs[i] = Pair{c.Slots[cup][2*i], c.Slots[cup][2*i+1]}
	}
	return pairs
}

// tieOut decides a two-legged tie for slot i (f_17c2_15e8): 1 = the club
// in slot i is out, 0 = it is through, 0x1e = still level. The penalty
// score replaces the second-leg score, as in the original.
func (s *State) tieOut(cup Cup, i int) int {
	c := &s.Cups
	h2 := c.Result[cup][i] % 10
	a2 := c.Result[cup][i+1]
	f1, f2 := c.First[cup][i], c.First[cup][i+1]
	q, p := f1+h2, f2+a2
	if s.Day < 316 || q != p {
		if q < p || (q == p && f1 < a2) {
			return 1
		}
		if !(q == p && f1 == a2) {
			return 0
		}
	}
	return NotInCup
}

// needsExtraTime marks ties that are level after 90 (pass 0) or 120 (pass
// 1) minutes with +10 and reports whether there is any (f_17c2_1226).
func (s *State) needsExtraTime(cup Cup, n, pass int) []int {
	c := &s.Cups
	var level []int
	for i := 0; i < n; i += 2 {
		if cup == CupDFB {
			h, a := c.Result[cup][i], c.Result[cup][i+1]
			if (pass == 0 && h == a) || (pass == 1 && h-a == markExtraTime) {
				c.Result[cup][i] += markExtraTime
				level = append(level, i)
			}
			continue
		}
		if c.Leg[cup] != 0 && s.tieOut(cup, i) == NotInCup {
			c.Result[cup][i] += markExtraTime
			level = append(level, i)
		}
	}
	return level
}

// advanceCup handles the winners after a cup event (f_17c2_16dc).
func (s *State) advanceCup(cup Cup) {
	c := &s.Cups
	round := c.Round[cup]
	n := pairsPerRound[round] * 2
	sl := &c.Slots[cup]
	if cup == CupDFB {
		for i := 0; i < n; i += 2 {
			h, a := ResultOf(c.Result[cup][i]), c.Result[cup][i+1]
			if h < a {
				sl[i], sl[i+1] = sl[i+1], sl[i]
			}
			if m := s.ManagerOf(sl[i]); m != nil {
				m.CupStatus[cup] = round + 1
			}
		}
		if n == 2 {
			c.Holder[CupDFB] = sl[0] + 1
			c.DFBRunnerUp = sl[1] + 1
		}
		if round <= 4 {
			s.clearCupResults()
			s.nextCupRound(cup)
		}
		return
	}
	if c.Leg[cup] == 0 {
		c.Leg[cup] = 1
		for i := 0; i < n; i += 2 {
			c.First[cup][i] = c.Result[cup][i+1]
			c.First[cup][i+1] = ResultOf(c.Result[cup][i])
			sl[i], sl[i+1] = sl[i+1], sl[i]
		}
		return
	}
	c.Leg[cup] = 0
	for i := 0; i < n; i += 2 {
		if s.tieOut(cup, i) != 0 {
			sl[i], sl[i+1] = sl[i+1], sl[i]
		}
		if !c.PlayingRelegation {
			if m := s.ManagerOf(sl[i]); m != nil {
				m.CupStatus[cup] = round + 1
			}
		}
	}
	if n == 2 && !c.PlayingRelegation {
		if sl[0] < NumGerman {
			c.Holder[cup] = sl[0] + 1
		} else {
			c.Holder[cup] = 0
		}
	}
}

// afterEuropeanRound moves all three European cups to the next round once
// a second leg has been played.
func (s *State) afterEuropeanRound() {
	c := &s.Cups
	if c.PlayingRelegation || c.Leg[CupLandesmeister] != 0 || c.Round[CupLandesmeister] > 4 {
		return
	}
	s.clearCupResults()
	for k := CupLandesmeister; k < NumCups; k++ {
		s.nextCupRound(k)
	}
}

func (s *State) clearCupResults() { s.Cups.Result = [NumCups][32]int{} }

// nextCupRound compacts the winners and shuffles them (f_17c2_13a2).
func (s *State) nextCupRound(cup Cup) {
	c := &s.Cups
	if c.Round[cup] == 5 {
		return
	}
	if cup != CupDFB {
		for k := CupLandesmeister; k < NumCups; k++ {
			c.First[k] = [32]int{}
		}
	}
	n := pairsPerRound[c.Round[cup]]
	c.Round[cup]++
	sl := &c.Slots[cup]
	for k := 0; k < n && k < 16; k++ {
		sl[k] = sl[2*k]
	}
	for i := 0; i < 20; i++ {
		a := s.Rand.Between(1, n) - 1
		b := s.Rand.Between(1, n) - 1
		sl[a], sl[b] = sl[b], sl[a]
	}
	if cup == CupDFB {
		s.dfbHomeRule()
	}
	s.queueDraw(cup)
}
