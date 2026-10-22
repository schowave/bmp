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
		s.notify(mi, s.Teams[m.Team].Name+" gewinnt die Deutsche Meisterschaft")
	}
	rec[1] = byte(l)
	rec[2] = byte(m.CupStatus[CupDFB] + 1)
	if s.Cups.Holder[CupDFB] == m.Team+1 {
		rec[2] += 0x80
		m.Titles[1]++
	}
	for cup := 1; cup < NumCups; cup++ {
		st := m.CupStatus[cup]
		if st == 0 || st == NotInCup {
			continue
		}
		rec[3] = byte(cup*8 + st)
		if s.Cups.Holder[cup] == m.Team+1 {
			rec[3] += 0x80
			m.Titles[1+cup]++
		}
	}
	m.History = append(m.History, rec)
}

// topScorerInTeam reports whether the best scorer of the manager's league
// plays for his club (f_14a4_1ad5(1)).
func (s *State) topScorerInTeam(mi int) bool {
	m := s.Managers[mi]
	best, bestGoals := -1, 0
	for p := 1; p < NumPersons; p++ {
		ps := &s.Persons[p]
		if ps.Team == 0xff || ps.Team >= NumGerman || LeagueOf(ps.Team) != m.League {
			continue
		}
		if ps.LeagueGoals > bestGoals {
			best, bestGoals = p, ps.LeagueGoals
		}
	}
	return best >= 0 && s.Persons[best].Team == m.Team
}

// seasonEnd1 runs the season end up to the contract talks.
func (s *State) seasonEnd1() {
	n := len(s.Managers)
	s.Loop.before = make([]int64, n)
	s.Loop.scorer = make([]bool, n)
	for mi, m := range s.Managers {
		s.Loop.before[mi] = s.Clubs[m.Team].Eternal
		m.GoalFlags = 0
		s.Loop.scorer[mi] = s.topScorerInTeam(mi)
	}
	s.addEternalPoints()
	k := min(s.Seasons+1, 5)
	r := 0
	for mi, m := range s.Managers {
		delta := s.Clubs[m.Team].Eternal - s.Loop.before[mi]
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
		s.notify(mi, "Saisonende: Sie beendeten Ihre Saison auf Platz "+itoa(pos)+" der "+LeagueNames[m.League]+".")
		if m.SeasonFlags&1 != 0 {
			if m.League == 0 {
				m.Money += 1600000
				s.notify(mi, "Gratulation zum Aufstieg (1.600.000 DM).")
			} else {
				m.Money += 800000
				s.notify(mi, "Gratulation zum Aufstieg (800.000 DM).")
			}
			if m.TrikotMonths == 0 {
				m.Ads[0] /= 10
			} else {
				m.TrikotMonths = 0
				s.notify(mi, "Ihre Werbepartner erlauben es, die Verträge jederzeit zu kündigen. Die Einnahmen bleiben solange erhalten.")
			}
			for b := 0; b < 6; b++ {
				if m.BoardMonths[b] == 0 {
					m.Ads[1+b] /= 10
				}
				m.BoardMonths[b] = 0
			}
		}
		if m.SeasonFlags&2 != 0 {
			s.notify(mi, "Ihr Abstieg erschüttert Millionen Fans.")
		}
		if m.SeasonFlags&4 != 0 {
			s.notify(mi, "DFB verweigert die Lizenz. Sie steigen ab.")
		}
		if m.SeasonFlags&8 != 0 {
			s.notify(mi, "Sie beginnen neu mit 500.000 DM.")
		}
		if s.Loop.scorer[mi] {
			m.Money += 250000
			s.notify(mi, "Torschützenkönig aus Ihrem Team (250.000 DM).")
		}
		if s.QualifiedForEurope(m.Team) {
			s.notify(mi, "Sie sind für den Europapokal qualifiziert.")
		}
		s.youthPlayer(mi)
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
	_ = r
}

// LeagueNames are the texts of the original.
var LeagueNames = [3]string{"Bundesliga", "Zweite Liga", "Am.-Oberliga"}

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
	s.notify(mi, "Ein Jugendspieler steigt in den Kader auf: "+ps.Name+".")
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
			s.notify(mi, ps.Name+" hängt den Fußballjob im Alter von "+itoa(ps.Age)+" Jahren an den Nagel.")
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
	s.notify(mi, name+" kehrt Ihrem Verein den Rücken... Sie erhalten eine Ablösesumme von "+FormatMoney(comp)+" DM.")
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
	s.Seasons++
	s.Day++
	for {
		d := s.Date()
		if d.Day == 28 && d.Month == 6 {
			break
		}
		for mi := range s.Managers {
			s.daily(mi)
			for i := 0; i < s.countLineup(mi, 0); i++ {
				s.injuryCountdown(&s.Squads[mi][i])
			}
		}
		s.Day++
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
	s.reshufflePersons()
	for c := 0; c < NumCups; c++ {
		s.drawCup(c)
	}
	s.beginEvent()
}
