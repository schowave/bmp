package game

// swapIdentity swaps everything that belongs to the clubs in slots a and b
// (f_0310_0b24). Afterwards the club that was at id a is at id b and the
// other way round. League tables hold ids and are not touched.
func (s *State) swapIdentity(a, b int, holders bool) {
	if a == b {
		return
	}
	swap := func(x int) int {
		switch x {
		case a:
			return b
		case b:
			return a
		}
		return x
	}
	for i := range s.Persons {
		s.Persons[i].Team = swap(s.Persons[i].Team)
	}
	for _, m := range s.Managers {
		m.Team = swap(m.Team)
	}
	if holders {
		c := &s.Cups
		for k := range c.Holder {
			if c.Holder[k] != 0 {
				c.Holder[k] = swap(c.Holder[k]-1) + 1
			}
		}
		if c.DFBRunnerUp != 0 {
			c.DFBRunnerUp = swap(c.DFBRunnerUp-1) + 1
		}
	}
	s.Teams[a], s.Teams[b] = s.Teams[b], s.Teams[a]
	s.Clubs[a], s.Clubs[b] = s.Clubs[b], s.Clubs[a]
	s.Stats.swap(a, b)
}

// shuffleIDs gives every club a random id inside its league, so that the
// schedule changes every season (f_0310_09c5). The visible table order
// stays the same.
func (s *State) shuffleIDs() {
	for l := 0; l < NumLeagues; l++ {
		lo := FirstID[l]
		hi := lo + TeamsPerLeague[l] - 1
		for k := 0; k < 55; k++ {
			a := s.Rand.Between(lo, hi)
			b := s.Rand.Between(lo, hi)
			s.swapIdentity(a, b, true)
			t := s.Tables[l][:TeamsPerLeague[l]]
			i, j := indexOf(t, a), indexOf(t, b)
			if i >= 0 && j >= 0 {
				t[i], t[j] = t[j], t[i]
			}
		}
	}
}

func indexOf(xs []int, v int) int {
	for i, x := range xs {
		if x == v {
			return i
		}
	}
	return -1
}

// driftStrengths changes the strength of all German clubs a little: p = 1
// every half week, p = 10 at the start of a season (f_0f9d_0697). Clubs of
// human managers are recomputed from their squads afterwards anyway.
func (s *State) driftStrengths(p int) {
	ranges := [3][2]int{{70, 93}, {40, 74}, {15, 45}}
	r := &s.Rand
	for t := 0; t < NumGerman; t++ {
		lg := LeagueOf(t)
		x := r.Between(0, 2*p)
		if p != 1 {
			s.Teams[t].Logo &^= 0x80
			p = 3
		}
		st, lo, hi := 1, 1, 99
		if p == 3 {
			st, lo, hi = 3, ranges[lg][0], ranges[lg][1]
		}
		tm := &s.Teams[t]
		for i := 0; i < 3; i++ {
			tm.Strength[i] = clamp(tm.Strength[i]+r.Between(0, 2*(st+x))-st-p, lo, hi)
		}
		for i := 0; i < 3; i++ {
			v := tm.Base[i]
			if r.Between(0, 2) == 0 || p == 3 {
				v += r.Between(0, 2*(st+x)) - p - st
			}
			tm.Base[i] = clamp(v, lo, hi)
		}
		for i := 0; i < 3; i++ {
			tm.Extra[i] = clamp(tm.Extra[i]+r.Between(0, 6)-3, 45, 55)
		}
	}
}

func clamp(v, lo, hi int) int {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}

// addEternalPoints adds the season's eternal-table points of every German
// club (f_1a58_4597).
func (s *State) addEternalPoints() {
	c := &s.Cups
	for t := 0; t < NumGerman; t++ {
		pos := s.Clubs[t].Pos
		base := 0
		if t <= 57 {
			base = (boolInt(t < 18)+boolInt(t < 38))*25 - pos + 19
		}
		for cup, bonus := range [NumCups]int{20, 50, 40, 30} {
			if c.Holder[cup] == t+1 {
				base += bonus
			}
		}
		switch {
		case t < 18 && pos == 0:
			base += 12
		case t < 18 && pos <= 4:
			base += 3
		case t >= 18 && t <= 57 && pos == 0:
			base += 3
		}
		s.Clubs[t].Eternal += int64(base)
	}
}

func boolInt(b bool) int {
	if b {
		return 1
	}
	return 0
}

// promotionRelegation moves clubs between the leagues at the end of a
// season (f_1a58_3176 @3d99). It includes the licence refusal for clubs of
// managers with more than 2 million DM debts.
func (s *State) promotionRelegation() {
	var forced [NumLeagues]int
	for _, m := range s.Managers {
		m.SeasonFlags = 0
		if m.Money < -2000000 && m.League < 2 {
			m.SeasonFlags = 4
			l := m.League
			t := s.Tables[l][:TeamsPerLeague[l]]
			i := indexOf(t, m.Team)
			if i >= 0 {
				copy(t[i:], t[i+1:])
				t[len(t)-1] = m.Team
			}
			forced[l]++
			m.League++
		}
	}
	for l := 0; l < NumLeagues; l++ {
		if l == 0 && s.Cups.RelegationOutcome == 0 {
			forced[0]++
		}
		if forced[l] > Relegated[l] {
			forced[l] -= Relegated[l]
		} else {
			forced[l] = 0
		}
		s.Matchday[l] = 1
		cnt := forced[l] + Relegated[l]
		for k := 0; k < cnt; k++ {
			rel := s.Tables[l][TeamsPerLeague[l]-1-k]
			var prom int
			if l < 2 {
				prom = s.Tables[l+1][k]
			} else {
				base := TeamsPerLeague[0] + TeamsPerLeague[1] + TeamsPerLeague[2]
				if k == 0 {
					prom = s.Rand.Between(base+3, base+4)
				} else {
					prom = base + 3 - k
				}
			}
			if l == 0 && cnt-k == 1 && s.Cups.RelegationOutcome == 0 && RelegationPlayoff[0] {
				continue
			}
			if mr := s.ManagerOf(rel); mr != nil && mr.SeasonFlags == 0 {
				if l < 2 {
					mr.SeasonFlags = 2
					mr.League++
				} else {
					// The human club stays: together with the swap below
					// this swaps back (original behaviour).
					mr.SeasonFlags = 8
					mr.Money = 500000
					s.swapIdentity(rel, prom, true)
				}
			}
			if mp := s.ManagerOf(prom); mp != nil && mp.SeasonFlags == 0 {
				mp.League--
				mp.SeasonFlags = 1
			}
			s.swapIdentity(rel, prom, true)
		}
	}
}
