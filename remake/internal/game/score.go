package game

// performance is f_322b_2864: the sporting part of the high score.
func (s *State) performance(mi int) int {
	m := s.Managers[mi]
	v := 57 - s.Clubs[m.Team].Pos - FirstID[m.League]
	if v < 0 {
		v = 0
	}
	p := int64(v / 2)
	for c := 0; c < NumCups; c++ {
		if b := m.CupStatus[c]; b >= 1 && b <= 6 {
			p += int64(2 * b)
		}
	}
	n := min(s.Seasons, 50)
	for i := 0; i < n && i < len(m.History); i++ {
		r := m.History[i]
		p += int64((58 - int(r[0])) / 2)
		if r[2]&0x80 != 0 {
			p += 20
			if r[3]&0x80 != 0 {
				p += 25
			} else {
				p += int64(3 * (r[3] & 7))
			}
		} else {
			p += int64(2 * (r[2] & 7))
		}
	}
	if n != 0 {
		p /= int64(n + 1)
	}
	return int(int16(p * 10))
}

// Score is the high-score value of manager mi (f_322b_2a2a).
func (s *State) Score(mi int) int {
	m := s.Managers[mi]
	sc := s.performance(mi)
	var assets int64
	for i := 0; i < s.SquadSize(mi); i++ {
		assets += s.PlayerValue(mi, i, 0)
	}
	assets += stadiumAttraction(m)*135000 + m.Money - m.Debts()
	sc += int(int16(assets / 80000))
	sc += 40*m.Titles[0] + 20*m.Titles[1] - (500 - 60*(m.Titles[2]+m.Titles[3]+m.Titles[4]))
	if s.EndYear > 2000 {
		sc += 300
	}
	if sc < 1 {
		sc = 1
	}
	return sc
}

// HighscoreEntry is a line of the high-score table.
type HighscoreEntry struct {
	Manager string
	Club    string
	Titles  [3]int // Meisterschaften, DFB-Pokale, Europapokale
	Score   int
}

// HighscoreEntry returns the entry of manager mi.
func (s *State) HighscoreEntry(mi int) HighscoreEntry {
	m := s.Managers[mi]
	return HighscoreEntry{Manager: m.Name, Club: s.Teams[m.Team].Name,
		Titles: [3]int{m.Titles[0], m.Titles[1], m.Titles[2] + m.Titles[3] + m.Titles[4]}, Score: s.Score(mi)}
}

// HighscoreKind returns the file suffix of the high-score table: game
// length digit and displayed level ("HIGH.02").
func (s *State) HighscoreKind() string {
	t := 0
	switch s.EndYear {
	case 1964, 1993:
		t = 1
	case 1966, 1995:
		t = 3
	}
	return itoa(t) + itoa(5-s.Level)
}
