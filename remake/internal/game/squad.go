package game

// Value flags of PlayerValue (f_2277_25de).
const (
	ValueSalary = 1 // monthly salary instead of the market value
	ValueLoan   = 2 // (unused by the original for market values)
	ValueNoise  = 4 // asking price: add some noise
)

var valueWeights = [2][6]int64{{58, 4, 4, 5, 4, 25}, {70, 8, 7, 5, 8, 2}}

// PlayerValue returns the market value or the monthly salary of the
// player in slot sl of squad m (f_2277_25de).
func (s *State) PlayerValue(m, sl, flags int) int64 {
	slot := &s.Squads[m][sl]
	p := &s.Persons[slot.Person]
	w := valueWeights[flags&1]
	v := int64((slot.Skill[0]+slot.Skill[1])/2) * w[0] / 100
	t1 := int64(((slot.Goals[0]+slot.Goals[1])*30 + slot.Goals[2]*40) / 100)
	t1 = clamp64(t1, 0, 20)
	t2 := (int64(slot.Career[0])*50 + int64(slot.Career[1])*20 + int64(slot.Career[2])*30) / 100
	t2 = clamp64(t2, 0, 200)
	v += w[1] * t1 * 5 / 100
	v += (t2 * 100 / 200) * w[2] / 100
	v += w[3] * int64(slot.Tendency) / 100
	t3 := clamp64(int64(6*slot.Cards[0]+slot.Cards[1]), 0, 20)
	v += w[4] * t3 * 5 / 100
	t4 := clamp64(int64(45-p.Age), 0, 27)
	v += (t4 * 100 / 27) * w[5] / 100
	if flags&ValueSalary != 0 {
		v = v * v * v / 1000
		x := slot.Skill[0] + slot.Skill[1]
		if x > 100 {
			x = (x-100)/2 + 100
		}
		if x > 135 {
			v = v * int64(x-35) / 100
		}
		if v < 2 {
			v = 2
		}
		v *= 85
	} else {
		v *= 10000
		if v > 110000 {
			v -= 100000
		}
		if flags&ValueLoan != 0 {
			v /= 3
		}
		if slot.Status&0x80 != 0 {
			if slot.Buyer > 63 {
				v = v * 130 / 100
			} else {
				v = v * int64(s.Rand.Between(95, 100)) / 100
			}
		}
	}
	v = v * 5 / 3
	if flags&ValueNoise != 0 && flags&ValueSalary == 0 {
		v += int64(s.Rand.Between16(int(v/20), int(v/10))) - v/15
	}
	if flags&ValueLoan != 0 && flags&ValueSalary == 0 {
		v = v * int64(s.Rand.Between(75, 95)) / 100
	}
	if flags&ValueSalary != 0 {
		return v / 100 * 100
	}
	return v / 1000 * 1000
}

func clamp64(v, lo, hi int64) int64 {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}

// initTendency gives a slot a random tendency (f_2277_000a).
func (s *State) initTendency(slot *Slot) {
	r := &s.Rand
	slot.Tendency = r.Between(35, 65)
	slot.Drift = r.Between(1, 13)
	slot.Drift |= r.Between(1, 3) << 4
	slot.Drift |= r.Between(0, 1) << 7
}

// ErrSquadFull and ErrMarketFull are the refusals of addPerson.
var (
	ErrSquadFull  = CreditError("Schon 24 Mann im Team")
	ErrMarketFull = CreditError("Schon 12 Mann auf dem Transfermarkt")
)

// ownedCount returns the players belonging to manager o, including those
// he put on the market (f_112a_00b4).
func (s *State) ownedCount(o int) int {
	n := 0
	for i := 0; i < SquadMax; i++ {
		if !s.Squads[o][i].Empty() {
			n++
		}
	}
	for h := 0; h <= MarketIndex; h++ {
		if h == o || (h < MarketIndex && h >= len(s.Managers)) {
			continue
		}
		for _, sl := range s.squad(h) {
			if !sl.Empty() && s.Persons[sl.Person].Owner == o {
				n++
			}
		}
	}
	return n
}

// addPerson inserts person p into squad m, keeping the slots sorted by
// person index (f_1ecd_37d8). It returns the slot index.
func (s *State) addPerson(m, p, years, loan, max int) (int, error) {
	sq := &s.Squads[m]
	if max == MarketSlots && !sq[MarketSlots-1].Empty() {
		return -1, ErrMarketFull
	}
	person := &s.Persons[p]
	if m != MarketIndex && person.Team != s.Managers[m].Team && s.ownedCount(m) >= max {
		return -1, ErrSquadFull
	}
	i := 0
	for i < SquadMax && !sq[i].Empty() && sq[i].Person < p {
		i++
	}
	if !sq[i].Empty() {
		copy(sq[i+1:], sq[i:SquadSlots-1])
	}
	slot := &sq[i]
	*slot = Slot{Person: p}
	slot.Fatigue = s.Rand.Between(80, 120)
	s.initTendency(slot)
	slot.Years = years
	slot.Loan = loan
	slot.Skill = person.Skill
	flags := ValueSalary
	if loan == 0x63 {
		flags = ValueSalary | ValueLoan
	}
	slot.Salary = s.PlayerValue(m, i, flags)
	if m != MarketIndex && person.Team != s.Managers[m].Team {
		person.LeagueGoals, person.LeagueGames = 0, 0
		person.Team = s.Managers[m].Team
	}
	return i, nil
}

// removeSlot removes slot i of squad m and closes the gap (f_1ecd_10ee).
// Without force a player owned by somebody else stays.
func (s *State) removeSlot(m, i int, force bool, n int) bool {
	sq := &s.Squads[m]
	if sq[i].Empty() {
		return true
	}
	if !force && s.Persons[sq[i].Person].Owner != m {
		return false
	}
	copy(sq[i:n], sq[i+1:n+1])
	sq[n] = Slot{}
	return true
}

// findPerson returns the squad and slot of person p, or -1.
func (s *State) findPerson(p int) (int, int) {
	for h := 0; h <= MarketIndex; h++ {
		if h < MarketIndex && h >= len(s.Managers) {
			continue
		}
		for i, sl := range s.squad(h) {
			if sl.Person == p {
				return h, i
			}
		}
	}
	return -1, -1
}

// Injury durations in weeks (D:0x5bc) and names (D:0x23b4).
var (
	injuryWeeks = [18]int{1, 2, 1, 2, 2, 4, 2, 1, 1, 6, 12, 5, 7, 10, 11, 9, 20, 14}
	InjuryNames = [18]string{"Oberschenkelzerrung", "Prellung d. Knöchels", "Platzwunde am Kopf",
		"Stauchung d. Fußgelenkes", "Sprunggelenkprellung", "Bänderdehnung", "Bauchmuskelzerrung",
		"Bluterguss am Knie", "Nasenbeinbruch", "Außenbandanriß", "Schienbeinbruch", "Muskelfaserriß",
		"Meniskusverletzung", "Innenbandabriß", "Wadenbeinbruch", "Kapselriß", "Leistenbruch", "Kreuzbandriß"}
)

// injure injures the player of a slot (f_14a4_30cf).
func (s *State) injure(slot *Slot) {
	r := &s.Rand
	p := &s.Persons[slot.Person]
	if p.Skill[2] > 20 {
		p.Skill[2] -= r.Between(12, 19)
	}
	k := r.Between(0, 17)
	if r.Between(0, 1) != 0 && r.Between(2, 4) < injuryWeeks[k] {
		k = r.Between(0, 17)
	}
	slot.Out = injuryWeeks[k] + r.Between(0, injuryWeeks[k]/4+1)
	slot.Status |= 2
	slot.Injury = k
	slot.Lineup = 0
}

// CupDay is F:0x513e: league suspensions do not apply on cup days.
func (s *State) available(slot *Slot) bool {
	st := slot.Status & 3
	return st == 0 || (st == 1 && s.CupDay)
}

// countLineup returns the number of slots with a player (mode 0), on the
// pitch (1) or on the bench (2) (f_3091_1109).
func (s *State) countLineup(m, mode int) int {
	n := 0
	limit := SquadMax
	if m == MarketIndex {
		limit = MarketSlots
	}
	for i := 0; i < limit; i++ {
		sl := &s.Squads[m][i]
		if sl.Empty() {
			continue
		}
		switch mode {
		case 0:
			n++
		case 1:
			if sl.Lineup >= 1 && sl.Lineup <= 11 {
				n++
			}
		case 2:
			if sl.Lineup >= 12 {
				n++
			}
		}
	}
	return n
}

// Formations of the automatic lineup (D:0x541c): GK, ABW, MIT, ANG.
var Formations = [3][4]int{{1, 4, 4, 2}, {1, 3, 5, 2}, {1, 3, 4, 3}}

// FormationNames are shown on the lineup screen.
var FormationNames = [3]string{"4-4-2", "3-5-2", "3-4-3"}

var (
	formationRow   = [4]int{7, 6, 3, 0}       // D:0x542e
	formationStart = [6]int{0, 3, 2, 1, 0, 0} // D:0x5428
)

// MaxReserves is F:0x56ee - 11.
const MaxReserves = 4

// pickBest returns the best available player for position k who is not
// yet in the lineup (f_1ecd_3635), or -1.
func (s *State) pickBest(m, k int) int {
	mode := s.Managers[m].AutoLineup
	pos := k
	for {
		best, bestV := -1, -2
		gk := 0
		for i := 0; i < SquadMax; i++ {
			sl := &s.Squads[m][i]
			if sl.Empty() || sl.Lineup != 0 || !s.available(sl) {
				continue
			}
			if s.Persons[sl.Person].Position() != pos {
				continue
			}
			v := sl.Strength()
			if k != PosKeeper && pos == PosKeeper {
				// A goalkeeper offered for a field position: the last one
				// found is taken.
				v = -1
				if gk > 0 {
					v = 1
				}
				gk++
			}
			if v >= bestV {
				best, bestV = i, v
			}
		}
		if best >= 0 {
			return best
		}
		if k == 3 || (k == 2 && mode < 4) {
			pos = (pos + 3) % 4
		} else {
			pos = (pos + 1) % 4
		}
		if pos == k {
			return -1
		}
	}
}

// AutoLineup sets the lineup automatically if the manager has chosen an
// automatic formation (f_1ecd_3360).
func (s *State) AutoLineup(m int) {
	mode := s.Managers[m].AutoLineup
	if mode <= 1 {
		return
	}
	f := mode - 2
	sq := &s.Squads[m]
	for i := 0; i < SquadMax; i++ {
		sq[i].Lineup = 0
	}
	shirt := 1
	for k := 0; k < 4; k++ {
		need := Formations[f][k]
		col := 0
		for need > 0 {
			i := s.pickBest(m, k)
			if i < 0 {
				break
			}
			sl := &sq[i]
			sl.Lineup = shirt
			shirt++
			need--
			sl.Row = formationRow[k]
			sl.Col = formationStart[Formations[f][k]] + col
			if f == 0 && shirt == 3 {
				sl.Row--
				sl.Col++
			}
			if f == 0 && shirt == 6 {
				sl.Row--
				sl.Col--
			}
			col += 2
			if Formations[f][k] == 5 && col != 2 && col != 6 {
				col--
			}
		}
	}
	for k, n := 0, 12; k < 4; k, n = k+1, n+1 {
		i := s.pickBest(m, k)
		if i < 0 {
			break
		}
		sq[i].Lineup = n
	}
	s.renumber(m)
}

// renumber numbers the players in the team 1..n in slot order
// (f_1ecd_24cd).
func (s *State) renumber(m int) {
	n := 1
	for i := 0; i < SquadMax; i++ {
		sl := &s.Squads[m][i]
		if !sl.Empty() && sl.Lineup >= 1 && sl.Lineup <= 11 {
			sl.Lineup = n
			n++
		}
	}
}

var lineOfRow = [8]int{0, 0, 0, 1, 1, 1, 2, 2} // D:0x2a2, index 7-row

func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}

func (s *State) sideDiff(sl *Slot) int { return abs(s.Persons[sl.Person].Side - sl.Col) }

func (s *State) posDiff(sl *Slot) int {
	return abs(s.Persons[sl.Person].Pos - ((sl.Row-7)*-75/7 + 5))
}

// LineupStrength turns the lineup of manager m into the strength values of
// his club (f_0f9d_0002). detailed adds the tactical bonuses and penalties
// used before matches.
func (s *State) LineupStrength(m int, detailed bool) {
	mg := s.Managers[m]
	r := &s.Rand
	bal := [3]int{-9, -8, -5}
	var pen, n, ksum, fsum [3]int
	tsum := [3]int{8, 8, 8}
	wing, gk, kt, cnt := 0, 1, 0, 0
	for i := 0; i < s.countLineup(m, 0); i++ {
		sl := &s.Squads[m][i]
		if sl.Lineup < 1 || sl.Lineup > 11 {
			continue
		}
		cnt++
		rr := 7 - sl.Row
		if rr < 0 || rr > 7 {
			rr = clamp(rr, 0, 7)
		}
		li := lineOfRow[rr]
		k, t := sl.Skill[0], sl.Skill[1]
		if detailed {
			col := sl.Col
			if col == 1 || col == 5 {
				wing++
			}
			if col == 0 || col == 6 {
				wing += 2
			}
			sd := s.sideDiff(sl)
			if rr < 3 {
				bal[0] += 2
				if rr == 1 && col == 3 && sd < 2 && k < t-10 {
					tsum[li] += 15
				}
				if rr == 0 && col == 3 {
					if s.Persons[sl.Person].Position() == PosKeeper {
						gk = 0
					} else {
						gk = 2
					}
				}
			} else {
				switch rr {
				case 3:
					bal[0]++
					bal[1]++
					if col > 1 && col < 5 && sd < 2 && k < t-10 {
						tsum[li] += 15
					}
				case 4, 5:
					bal[1] += 2
				case 6:
					bal[1]++
					bal[2]++
				default:
					bal[2] += 2
				}
				if sd > 2 {
					pen[li] += r.Between(2, 6)
				}
				if s.posDiff(sl) > 25 {
					pen[li] += r.Between(2, 6)
				}
			}
			kt += k - t
			ksum[li] += sl.Games[0]/6 - sl.Fatigue/20 + 3
			tsum[li] += sl.Goals[0] - 1
		}
		ksum[li] += k
		tsum[li] += t
		fsum[li] += sl.Skill[2]
		n[li]++
	}
	for i := 0; i < 3; i++ {
		tsum[i] += (s.Level - 5) * pen[i] * 20 / 100
	}
	if cnt == 0 {
		tsum = [3]int{}
	}
	if cnt < 8 {
		mg.Effort = ForfeitEffort
	} else {
		mg.Effort = 0
	}
	for j := 0; cnt != 0 && cnt < 11; j = (j + 1) % 3 {
		n[j]++
		cnt++
	}
	if detailed {
		if s.Level == 0 && wing > 8 {
			tsum[2] -= 100 * n[2]
		}
		if gk != 0 {
			g := gk - 1
			tsum[0] = tsum[0] * r.Between(10*g+5, 20*g+5) / 100
			tsum[1] = tsum[1] * r.Between(10, 40) / 100
			ksum[0] = ksum[0] * r.Between(5*(g+2), 30*g+20) / 100
		}
		tsum[0] = clamp(tsum[0], 0, 32000)
		tsum[1] = clamp(tsum[1], 0, 32000)
		ksum[0] = clamp(ksum[0], 0, 32000)
		if bal[1] < 1 {
			if bal[0] > 0 {
				bal[2] += bal[1] - 1
			}
			if bal[2] > 0 {
				bal[0] += bal[1] - 1
			}
		}
		if bal[0] < -1 && r.Between(7, 10) > bal[0]+10 {
			bal[1] -= r.Between(3, 5)
		}
		for i := 0; i < 3; i++ {
			tsum[i] += s.Level
			if ksum[i] != 0 {
				ksum[i] += mg.EffortSetting - 16
			}
			if tsum[i] != 0 && s.Level < 4 {
				tsum[i] += (mg.Subs*3 + n[i]*bal[i]) * 10
				ksum[i] += n[i] * bal[i] * 5
			}
			if tsum[i] < 0 {
				tsum[i] = 0
			}
			if ksum[i] < 0 {
				ksum[i] = 0
			}
		}
		kt = clamp(kt/9, -4, 4)
		if kt < 0 {
			kt = 0
		}
		kt = clamp(kt+mg.EffortSetting, 0, 40)
		if mg.Effort != ForfeitEffort {
			mg.Effort = kt
		}
	}
	tm := &s.Teams[mg.Team]
	for i := 0; i < 3; i++ {
		if n[i] <= 1 {
			tm.Strength[i], tm.Base[i], tm.Extra[i] = 0, 0, 0
			continue
		}
		tm.Strength[i] = (ksum[i] / n[i]) & 0xff
		tm.Base[i] = (tsum[i] / n[i]) & 0xff
		tm.Extra[i] = (fsum[i] / n[i]) & 0xff
	}
}
