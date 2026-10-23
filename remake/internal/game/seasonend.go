package game

// recordHistory books the season into the eternal balance and the history
// of manager mi (f_1a58_3176 @3763).
func (s *State) recordHistory(mi int) {
	m := s.Managers[mi]
	c := &s.Clubs[m.Team]
	for h := 0; h < 2; h++ {
		m.EtPoints[h] += c.Points[h]
		m.EtMinus[h] += 2*c.Games[h] - c.Points[h]
		m.EtGoals[h] += c.Goals[h]
		m.EtAgainst[h] += c.Against[h]
		m.EtGames[h] += c.Games[h]
		m.EtWins[h] += c.Wins[h]
		m.EtLosses[h] += c.Losses[h]
		m.EtDraws[h] += c.Games[h] - c.Losses[h] - c.Wins[h]
	}
	m.SeasonAudience, m.RecordAudience, m.HomeGames, m.MinAudience = 0, 0, 0, 99999
	if len(m.History) >= 50 {
		m.History = m.History[1:]
		m.DroppedSeasons++
	}
	l := m.League
	var rec [4]byte
	rec[0] = byte(c.Pos + 1 + FirstID[l])
	if c.Pos == 0 && l == 0 {
		m.Titles[0]++
		s.trophy(mi, 0)
	}
	rec[1] = byte(l)
	// Unlike the European cups the original does not skip NotInCup here:
	// the record gets 0x1f, which performance counts as round 7.
	rec[2] = byte(m.CupStatus[CupDFB] + 1)
	if s.Cups.Holder[CupDFB] == m.Team+1 {
		rec[2] += 0x80
		m.Titles[1]++
	}
	for cup := CupLandesmeister; cup < NumCups; cup++ {
		st := m.CupStatus[cup]
		if st == 0 || st == NotInCup {
			continue
		}
		rec[3] = byte(int(cup)*8 + st)
		if s.Cups.Holder[cup] == m.Team+1 {
			rec[3] += 0x80
			m.Titles[1+cup]++
		}
	}
	m.History = append(m.History, rec)
}

// topScorerInTeam reports whether the best scorer of the manager's league
// plays for his club (f_14a4_1ad5(1)): the first of the top-scorer list,
// which needs at least two goals.
func (s *State) topScorerInTeam(mi int) bool {
	m := s.Managers[mi]
	top := s.TopScorers(m.League, false)
	return len(top) > 0 && s.Persons[top[0].Person].Team == m.Team
}

// seasonEnd1 runs the season end up to the contract talks.
func (s *State) seasonEnd1() {
	s.banner("SAISONENDE") // f_1a58_4597
	n := len(s.Managers)
	s.Loop.Before = make([]int64, n)
	s.Loop.Scorer = make([]bool, n)
	for mi, m := range s.Managers {
		s.Loop.Before[mi] = s.Clubs[m.Team].Eternal
		m.GoalFlags = 0
		s.Loop.Scorer[mi] = s.topScorerInTeam(mi)
	}
	s.addEternalPoints()
	k := min(s.Seasons+1, 5)
	r := 0
	for mi, m := range s.Managers {
		delta := s.Clubs[m.Team].Eternal - s.Loop.Before[mi]
		r = m.Popularity + int((uint32(delta*100)/108)/uint32(k)) - int(uint16(m.Popularity))/k
		r = clamp(r, 0, 100)
	}
	for t := range s.Clubs {
		s.Clubs[t].resetSeason()
	}
	s.promotionRelegation()
	s.driftStrengths(10)
	// Original bug: every manager gets the value of the last one.
	for _, m := range s.Managers {
		if r < 30 && m.League < 2 {
			r = 30
		}
		if r < 61 && m.League == 0 {
			r = 61
		}
		m.Popularity = r
	}
	year := DateOf(s.StartYear, s.Day).Year
	if s.EndYear == year {
		s.Loop.Phase = PhaseGameOver
		return
	}
	for mi := range s.Managers {
		s.Highscores = append(s.Highscores, s.HighscoreEntry(mi))
		s.sponsorOffers(mi)
	}
	s.shuffleIDs()
	s.qualifyEurope()
	s.seasonReport()
	s.Loop.Phase = PhaseContracts
	s.Loop.Current = 0
	s.skipSettledContracts()
}

// seasonReport is the first part of f_0cb5_0012: bonuses, youth players,
// aging and retirement, season statistics.
func (s *State) seasonReport() {
	r := &s.Rand
	for mi, m := range s.Managers {
		pos := 0
		if len(m.History) > 0 {
			h := m.History[len(m.History)-1]
			pos = int(h[0]) - FirstID[h[1]]
		}
		s.seasonScreen(mi, pos, m.League)
		if m.SeasonFlags&1 != 0 {
			if m.League == 0 {
				m.Money += 1600000
				s.seasonLine(mi, msgPromoted1)
			} else {
				m.Money += 800000
				s.seasonLine(mi, msgPromoted2)
			}
			if m.TrikotMonths == 0 {
				m.Ads[0] /= 10
			} else {
				m.TrikotMonths = 0
				s.seasonDialog(mi, msgAdsFree)
			}
			for b := 0; b < 6; b++ {
				if m.BoardMonths[b] == 0 {
					m.Ads[1+b] /= 10
				}
				m.BoardMonths[b] = 0
			}
		}
		if m.SeasonFlags&2 != 0 {
			s.seasonLine(mi, msgRelegated)
		}
		if m.SeasonFlags&4 != 0 {
			s.seasonLine(mi, msgNoLicence)
		}
		if m.SeasonFlags&8 != 0 {
			s.seasonLine(mi, msgRestart)
		}
		if s.Loop.Scorer[mi] {
			m.Money += 250000
			s.seasonLine(mi, msgTopScorer)
		}
		if s.QualifiedForEurope(m.Team) {
			s.seasonLine(mi, msgEurope)
		}
		s.youthPlayer(mi)
		s.seasonDone(mi)
		// The joke boxes over the report (only in the DM version, which
		// this is).
		if r.Between(0, 4) == 0 {
			s.seasonDialog(mi, msgJoke1)
			s.seasonDialog(mi, msgJoke2)
		}
		if mi == 0 {
			s.agePlayers()
		}
		s.retirePlayers(mi)
	}
	for mi := range s.Managers {
		for i := 0; i < SquadMax; i++ {
			sl := &s.Squads[mi][i]
			sl.Cards, sl.Goals, sl.Games = [3]int{}, [3]int{}, [3]int{}
		}
	}
}

// LeagueNames are the texts of the original.
var LeagueNames = [3]string{"Bundesliga", "Zweite Liga", "Am.-Oberliga"}

// LeagueName is the name of league l in the season that starts in year.
// The original always says Zweite Liga and Am.-Oberliga; the remake
// follows the real second and third tier: Regionalliga until 1973/74,
// then Zweite Liga; Amateurliga until 1977/78, Am.-Oberliga until 1993/94,
// Regionalliga until 2007/08, then 3. Liga.
func LeagueName(l, year int) string {
	switch {
	case l == 1 && year < 1974:
		return "Regionalliga"
	case l == 2 && year < 1978:
		return "Amateurliga"
	case l == 2 && year >= 2008:
		return "3. Liga"
	case l == 2 && year >= 1994:
		return "Regionalliga"
	}
	return LeagueNames[l]
}

// LeagueName is the name of league l in the current season.
func (s *State) LeagueName(l int) string { return LeagueName(l, s.StartYear) }

// HistoryYear is the year in which the season of history record r of
// manager mi started.
func (s *State) HistoryYear(mi, r int) int {
	last := s.StartYear - 1
	if s.Loop.Phase == PhaseContracts || s.Loop.Phase == PhaseGameOver {
		last = s.StartYear // recorded, the new season has not begun
	}
	return last - (len(s.Managers[mi].History) - 1 - r)
}

// youthPlayer promotes a youth player (f_0cb5_0012 @033d).
func (s *State) youthPlayer(mi int) {
	r := &s.Rand
	m := s.Managers[mi]
	year := DateOf(s.StartYear, s.Day).Year
	if !((year > 1966 && !s.Modern) || (year > 1995 && s.Modern)) {
		return
	}
	y := m.YouthPoints / 12
	y = y + y/-3
	m.YouthPoints = y * 12
	if y <= 30 || r.Between(0, 2) != 0 || s.ownedCount(mi) >= 23 {
		return
	}
	y >>= 1
	var p int
	for {
		p = r.Between(1, 150)
		if s.Persons[p].Owner == OwnerFree {
			break
		}
	}
	y = clamp(y, 10, 90)
	ps := &s.Persons[p]
	ps.Side = r.Between(0, 6)
	ps.Age = r.Between(17, 19)
	ps.Skill[2] = r.Between(45, 55)
	ps.Skill[1] = r.Between(y-3, y+3)
	ps.Skill[0] = r.Between(y-3, y+3)
	ps.Team = m.Team
	si, err := s.addPerson(mi, p, 0, 0, SquadMax)
	if err != nil {
		return
	}
	ps.Owner = mi
	sl := &s.Squads[mi][si]
	sl.Salary = s.PlayerValue(mi, si, ValueSalary) * 50 / 100
	sl.Years = r.Between(2, 3)
	s.seasonLine(mi, msgYouthPlayer)
}

// agePlayers makes everybody a year older and sends listed players back
// (f_0cb5_0012 aging part).
func (s *State) agePlayers() {
	for p := 1; p < NumPersons; p++ {
		ps := &s.Persons[p]
		ps.Age++
		h, i := s.findPerson(p)
		if h < 0 {
			continue
		}
		sl := &s.Squads[h][i]
		sl.Message = false
		sl.Status &= 0x3f
		sl.Years--
		if sl.Years < 0 {
			sl.Years = 0
		}
		if ps.Owner == h {
			continue
		}
		old := *sl
		if ps.Owner < MarketIndex && ps.Owner < len(s.Managers) {
			if ni, err := s.addPerson(ps.Owner, p, 1, 0, SquadMax+1); err == nil {
				s.Squads[ps.Owner][ni] = old
				s.Squads[ps.Owner][ni].Nego, s.Squads[ps.Owner][ni].Loan = 0, 0
			}
		} else {
			ps.Owner = OwnerFree
		}
		limit := SquadMax
		if h == MarketIndex {
			limit = MarketSlots
		}
		// Remove from the holder; the index may have moved by the insert.
		if hh, ii := s.findPersonIn(h, p); ii >= 0 {
			s.removeSlot(hh, ii, true, limit)
		}
	}
}

func (s *State) findPersonIn(h, p int) (int, int) {
	for i, sl := range s.squad(h) {
		if sl.Person == p {
			return h, i
		}
	}
	return h, -1
}

// retirePlayers retires old players and creates new young ones in their
// place (f_0cb5_0012 @retire).
func (s *State) retirePlayers(mi int) {
	r := &s.Rand
	last := mi == len(s.Managers)-1
	for p := 1; p < NumPersons; p++ {
		ps := &s.Persons[p]
		if last {
			ps.LeagueGoals, ps.LeagueGames = 0, 0
		}
		h, i := s.findPerson(p)
		if h >= 0 && h != mi {
			continue
		}
		rr := r.Between(32, 34)
		retire := false
		if h < 0 {
			retire = ps.Age > rr
		} else {
			sl := &s.Squads[h][i]
			retire = sl.Years == 0 && sl.Nego&0x80 != 0
		}
		if !retire {
			continue
		}
		if h >= 0 {
			s.seasonPopup(mi, msgRetires(ps.Name, ps.Age))
			s.removeSlot(h, i, true, SquadMax)
			ps.Owner = OwnerFree
		}
		ps.Age = r.Between(18, 25)
		b := r.Between(30, 92)
		ps.Skill[0] = r.Between(b-5, b+5)
		r.Between(0, 6)
		ps.Skill[1] = r.Between(b-5, b+5)
	}
}

// ExpiringContracts returns the squad slots of manager Current whose
// contract ends now.
func (s *State) ExpiringContracts() []int {
	if s.Loop.Phase != PhaseContracts {
		return nil
	}
	mi := s.Loop.Current
	var out []int
	for i := 0; i < s.countLineup(mi, 0); i++ {
		if s.Squads[mi][i].Years == 0 {
			out = append(out, i)
		}
	}
	return out
}

// ReleasePlayer lets a player with an expired contract go; the club gets
// half his value as compensation.
func (s *State) ReleasePlayer(mi, si int) int64 {
	comp := s.PlayerValue(mi, si, 0) / 2 / 1000 * 1000
	name := s.Persons[s.Squads[mi][si].Person].Name
	s.Managers[mi].Money += comp
	s.removeSlot(mi, si, true, SquadMax)
	s.notify(mi, msgReleased(name, comp))
	s.AutoLineup(mi)
	return comp
}

// ContractsDone continues after the contract talks of manager Current.
func (s *State) ContractsDone() {
	if s.Loop.Phase != PhaseContracts {
		return
	}
	for _, si := range s.ExpiringContracts() {
		s.ReleasePlayer(s.Loop.Current, si)
		// Indexes shift after a removal; start again.
		s.ContractsDone()
		return
	}
	s.Loop.Current++
	s.skipSettledContracts()
}

func (s *State) skipSettledContracts() {
	for s.Loop.Current < len(s.Managers) && len(s.ExpiringContracts()) == 0 {
		s.Loop.Current++
	}
	if s.Loop.Current >= len(s.Managers) {
		s.seasonEnd2()
	}
}

// seasonEnd2 finishes the season end after the contract talks: summer
// break and the new season (f_1a58_3176 @43b5..).
func (s *State) seasonEnd2() {
	s.movePersons()
	s.Seasons++
	s.Day++
	for {
		d := s.Date()
		for mi := range s.Managers {
			s.daily(mi)
			for i := 0; i < s.countLineup(mi, 0); i++ {
				s.injuryCountdown(&s.Squads[mi][i])
			}
		}
		s.Day++
		if d.Day == 28 && d.Month == 6 {
			break
		}
	}
	s.Event = 0
	s.Day = 0
	s.StartYear++
	s.Results = [NumLeagues][38][10]Score{}
	s.Events = seasonEvents
	s.Postponed = [20]PostponedGame{}
	for mi := range s.Managers {
		s.seasonBudget(mi)
	}
	s.clearCupResults()
	for c := CupDFB; c < NumCups; c++ {
		s.drawCup(c)
	}
	s.Draws = append(s.Draws, CupDFB, CupLandesmeister, CupPokalsieger, CupUEFA)
	s.beginEvent()
}

// personRanges are the team ids of the leagues and of the rest (D:0x2272).
var personRanges = [4][2]int{{0, 17}, {18, 37}, {38, 57}, {58, 255}}

// movePersons ends the season report (f_0cb5_2756): after a reshuffle each
// league takes persons not owned by a human from the other ranges until it
// has its share, then the persons are reshuffled again.
func (s *State) movePersons() {
	var need [4]int
	share := s.reshufflePersons()
	copy(need[:], share[:])
	var list [4][]int
	for k, rg := range personRanges {
		for p := 1; p < NumPersons; p++ {
			if t := s.Persons[p].Team; t >= rg[0] && t <= rg[1] {
				if need[k] != 0 {
					need[k]--
				}
				if s.Persons[p].Owner > 3 {
					list[k] = append(list[k], p)
				}
			}
		}
	}
	var left [4]int
	for k := range list {
		left[k] = len(list[k])
	}
	for k := range need {
		for tries := 0; need[k] > 0 && tries < 1000; tries++ {
			for j := range list {
				for j != k && left[j] > 0 && need[k] > 0 {
					idx := 0
					for try := 0; try <= 1000; try++ {
						idx = s.Rand.Between(0, len(list[j])-1)
						if list[j][idx] != 0 {
							break
						}
					}
					p := list[j][idx]
					list[j][idx] = 0
					t := s.leagueTeam(k)
					if p != 0 {
						s.Persons[p].Team = t
					}
					left[j]--
					need[k]--
				}
			}
		}
	}
	s.reshufflePersons()
}

// leagueTeam draws a club of league k without a human manager, mostly one
// of the first ids (f_0cb5_2987); for the rest any id from 58.
func (s *State) leagueTeam(k int) int {
	r := &s.Rand
	if k >= NumLeagues {
		return r.Between(personRanges[k][0], 255)
	}
	for {
		x := r.Between(0, 3)
		if r.Between(0, 1) != 0 {
			x = r.Between(0, 6)
		}
		if r.Between(0, 5) == 0 {
			x = r.Between(0, 12)
		}
		if t := FirstID[k] + x; s.ManagerOf(t) == nil {
			return t
		}
	}
}
