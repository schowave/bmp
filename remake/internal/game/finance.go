package game

// strengthWeights is D:0x8e: weights of the three strength values for the
// three parts of the team (defence, midfield, attack).
var strengthWeights = [3][3]int{{80, 20, 0}, {25, 50, 25}, {0, 20, 80}}

// TeamPart returns the strength of part k of club t (f_0310_1468(t,k,0)).
func (s *State) TeamPart(t, k int) int {
	tm := &s.Teams[t]
	v := 0
	for i := 0; i < 3; i++ {
		v += (tm.Base[i] + tm.Strength[i] + tm.Extra[i]) * strengthWeights[k][i] / 100
	}
	return v + tm.Mood - 44
}

// TeamStrength is the combined strength used by the attendance formula.
func (s *State) TeamStrength(t int) int {
	return (s.TeamPart(t, 1) + s.TeamPart(t, 0) + s.TeamPart(t, 2)) * 2 / 15
}

// MonthlyIncome returns the regular income of a manager for one month
// (f_14a4_2636, "EINNAHMEN (SPENDEN, SPONSOREN)"). It also stores the base
// income like the original.
func (s *State) MonthlyIncome(mi int) int64 {
	m := s.Managers[mi]
	tm := &s.Teams[m.Team]
	sum := 0
	for k := 0; k < 3; k++ {
		sum += tm.Strength[k] + tm.Base[k]
	}
	base := int64(sum*15*s.SquadSize(mi)/3 + m.Popularity/5 + 25)
	m.BaseIncome = int(base & 0xffff)
	inc := base
	for i := 0; i < 8; i++ {
		inc += m.Ads[i]
	}
	// Faithful bug: every manager receives the interest the other human
	// managers pay on all their credits.
	for oi, o := range s.Managers {
		if oi != mi {
			inc += o.creditSum(true)
		}
	}
	if m.BalanceSum > 0 {
		day := s.Date().Day
		inc += m.BalanceSum / int64(day) / 75
	}
	if inc < 0 {
		inc = 0
	}
	return inc
}

// MonthlyExpenses returns the regular expenses of a manager for one month
// (f_14a4_2822, "AUSGABEN (GEHÄLTER, REISEKOSTEN)").
func (s *State) MonthlyExpenses(mi int) int64 {
	m := s.Managers[mi]
	e := int64(150000)
	for i := 0; i < SquadSlots; i++ {
		if sl := &s.Squads[mi][i]; !sl.Empty() {
			e += sl.Salary
		}
	}
	for i := 0; i < MarketSlots; i++ {
		sl := &s.Squads[MarketIndex][i]
		if !sl.Empty() && s.Persons[sl.Person].Owner == mi {
			e += sl.Salary
		}
	}
	e += s.stadiumUpkeep(m)
	e += int64(uint16(m.CostItem))
	e += int64((m.Youth+1)*10000) >> 4
	e += m.Ads[8]
	e += m.creditSum(true)
	if m.BalanceSum < 0 {
		day := s.Date().Day
		e += m.BalanceSum / int64(day) / -10
	}
	if m.League != 2 && m.Money > 2000000 {
		t := (m.Money-2000000)/500000*30000 + 20000
		if m.Money >= 4000000 {
			t = t * 3 / 2
		}
		e += t
	}
	if e < 0 {
		e = 0
	}
	return e
}

func (s *State) stadiumUpkeep(m *Manager) int64 {
	st := &m.Stadium
	v := (int64(3-m.League)*43750+(st[StFloodlight]+st[StScoreboard])*1375+st[StComfort]*3000+st[StSeats])*2 + st[StStanding]
	return v*2 + st[StCovered]
}

// RequiredAverage is the average attendance per home game needed to cover a
// year's deficit (f_14a4_2b15, "BENÖTIGTER SCHNITT").
func (s *State) RequiredAverage(mi int) int64 {
	m := s.Managers[mi]
	v := (12*s.MonthlyExpenses(mi) - 12*s.MonthlyIncome(mi)) / int64(TeamsPerLeague[m.League]-1) / int64(m.TicketPrice)
	if v < 0 {
		return 0
	}
	return v
}

// creditSum returns the sum of the amounts or of the monthly interest of
// all credits of a manager (f_112a_0f8d).
func (m *Manager) creditSum(interest bool) int64 {
	var v int64
	for k := range m.Credits {
		for j := range m.Credits[k] {
			c := &m.Credits[k][j]
			if c.Amount == 0 {
				continue
			}
			if interest {
				v += c.Interest
			} else {
				v += c.Amount
			}
		}
	}
	return v
}

// Debts returns the sum of all credits of a manager.
func (m *Manager) Debts() int64 { return m.creditSum(false) }

// InterestPerMonth returns the monthly interest of all credits.
func (m *Manager) InterestPerMonth() int64 { return m.creditSum(true) }

// driftBankRates changes the bank's interest table (f_112a_000a).
func (s *State) driftBankRates() {
	h := boolInt(s.Level > 2)
	s.BankBase = clamp(s.BankBase+s.Rand.Between(0, 2+h)+h-1, 2, 7)
	r, step := s.BankBase, 5-s.BankBase
	for i := 0; i < 6; i++ {
		s.BankRates[i] = r
		r += step
		if step > 1 {
			step--
		}
		r = clamp(r, 2, 7)
	}
}

// Bank lender index in Manager.Credits.
const LenderBank = 4

// TakeCredit errors.
type CreditError string

func (e CreditError) Error() string { return string(e) }

// Credit refusals of the original.
const (
	ErrBankLimit    CreditError = "Sie können nur maximal 4 Mio. DM Kredit von der Bank aufnehmen."
	ErrLenderLimit  CreditError = "Sie können maximal 1 Mio. DM Kredit von einem Mitspieler aufnehmen."
	ErrLenderMoney  CreditError = "So viel Geld besitzt der Mitspieler gar nicht..."
	ErrThreeCredits CreditError = "Schon 3 Kredite aufgenommen."
	ErrMonths       CreditError = "Maximal 24 Monate."
	ErrUsury        CreditError = "Wegen Wucher..."
	ErrMinRate      CreditError = "Mindest-Prozentsatz: 3 Prozent"
)

// TakeBankCredit takes a credit from the bank with the duration of row
// (0..5 = 3..18 months).
func (s *State) TakeBankCredit(mi int, amount int64, row int) error {
	m := s.Managers[mi]
	var sum int64
	for _, c := range m.Credits[LenderBank] {
		sum += c.Amount
	}
	if sum+amount > 4000000 {
		return ErrBankLimit
	}
	return s.addCredit(mi, LenderBank, amount, s.BankRates[row], (row+1)*3)
}

// TakeManagerCredit borrows from another human manager.
func (s *State) TakeManagerCredit(mi, lender int, amount int64, months, rate int) error {
	o := s.Managers[lender]
	if o.Money < amount {
		return ErrLenderMoney
	}
	var sum int64
	for _, c := range s.Managers[mi].Credits[lender] {
		sum += c.Amount
	}
	if sum+amount > 1000000 {
		return ErrLenderLimit
	}
	if months > 24 {
		return ErrMonths
	}
	if rate > 13 {
		return ErrUsury
	}
	if rate < 3 {
		return ErrMinRate
	}
	if err := s.addCredit(mi, lender, amount, rate, months); err != nil {
		return err
	}
	o.Money -= amount
	return nil
}

// addCredit fills a credit slot (f_112a_1039) and pays out the amount.
func (s *State) addCredit(mi, lender int, amount int64, rate, months int) error {
	m := s.Managers[mi]
	j := -1
	for k, c := range m.Credits[lender] {
		if c.Amount == 0 {
			j = k
			break
		}
	}
	if j < 0 {
		return ErrThreeCredits
	}
	d := s.Date()
	dm, dy := d.Month+months, d.Year
	for dm > 11 {
		dm -= 12
		dy++
	}
	dd := d.Day
	if dd > monthLen[dm] {
		dd = monthLen[dm]
	}
	m.Credits[lender][j] = Credit{
		Amount: amount, Interest: amount * int64(rate) / 100, Rate: rate,
		StartDay: d.Day, StartMonth: d.Month, StartYear: d.Year,
		DueDay: dd, DueMonth: dm, DueYear: dy,
	}
	m.Money += amount
	return nil
}

// payCredits pays interest and due credits of a manager (f_112a_0a6d step 7).
func (s *State) payCredits(mi int) {
	m := s.Managers[mi]
	d := s.Date()
	for k := range m.Credits {
		for j := range m.Credits[k] {
			c := &m.Credits[k][j]
			if c.Amount == 0 {
				continue
			}
			pay := c.DueDay == d.Day || (monthLen[c.DueMonth] > monthLen[d.Month] && d.Day == monthLen[d.Month])
			if !pay {
				continue
			}
			m.Money -= c.Interest
			if c.DueYear == d.Year && c.DueMonth == d.Month {
				lender := "der Bank"
				if k < len(s.Managers) && k != LenderBank {
					lender = s.Managers[k].Name
				}
				s.notify(mi, "Ihr Kredit von "+lender+" wurde heute fällig.")
				m.Money -= c.Amount
				if k != LenderBank && k < len(s.Managers) {
					s.Managers[k].Money += c.Amount
				}
				c.Amount = 0
			}
		}
	}
}

// daily runs the daily bookkeeping of a manager (f_112a_0a6d).
func (s *State) daily(mi int) {
	m := s.Managers[mi]
	s.constructionProgress(mi)
	for i := range s.CampTimers {
		x := s.CampTimers[i]
		sign := boolInt(x < 0) - boolInt(x > 0)
		x += sign
		if x == 0 {
			x = s.Rand.Between(20, 70) * sign
		}
		s.CampTimers[i] = x
	}
	if s.Rand.Between(0, 60) == 0 {
		s.driftBankRates()
	}
	m.BalanceSum += m.Money
	if m.CampDays != 0 {
		m.CampDays--
	}
	d := s.Date()
	if d.Day == monthLen[d.Month] {
		if m.TrikotMonths != 0 {
			m.TrikotMonths--
			if m.TrikotMonths == 0 {
				m.Ads[0] /= 10
			}
		}
		for b := 0; b < 6; b++ {
			if m.BoardMonths[b] != 0 {
				m.BoardMonths[b]--
				if m.BoardMonths[b] == 0 {
					m.Ads[1+b] /= 10
				}
			}
		}
		m.YouthPoints += m.Youth * 4
		m.Money += s.MonthlyIncome(mi)
		m.Money -= s.MonthlyExpenses(mi)
		m.BalanceSum = 0
		if m.Popularity < 95 && m.Ads[8]/2500 > int64(s.Rand.Between(0, 22)) {
			m.Popularity += s.Rand.Between(1, 3)
		}
	}
	s.payCredits(mi)
}

// seasonBudget sets the TV income and the cost item of a manager for a new
// season (f_08bc_0a63).
func (s *State) seasonBudget(mi int) {
	m := s.Managers[mi]
	lg := 2 - m.League
	w := uint16(lg*(s.Rand.Between(15, 20)+40) + int(uint16((lg*(110-m.Popularity))&0xfffe))*500)
	m.CostItem = int(uint16(int(w) * s.Rand.Between((lg+10)*10, (lg+4)*25) / 100))
	m.Ads[7] = int64((30*lg + m.Popularity) * 1000)
}

// stadiumAttraction rates the stadium 0..100 for sponsors (f_14a4_2bba).
func stadiumAttraction(m *Manager) int64 {
	st := &m.Stadium
	v := st[StCovered]/2 + st[StStanding] + st[StSeats]
	if v > 80000 {
		v = 80000
	}
	v += st[StScoreboard]*1900 + st[StFloodlight]*1500 + (2*st[StComfort]+st[StCondition])*1000
	if v > 100000 {
		v = 100000
	}
	return v / 1000
}

// sponsorOffers creates new advertising offers (f_14a4_2cb4).
func (s *State) sponsorOffers(mi int) {
	m := s.Managers[mi]
	r := &s.Rand
	for t := 0; t < 2; t++ {
		for k := 0; k < 10; k++ {
			v := int(stadiumAttraction(m))
			if v < 50 && m.Popularity > 29 {
				v += 20
			}
			rr := r.Between(15, 20)
			a := v/2 + 50
			tt := v/2 + rr
			qq := a/5 + 2*k
			offer := int64(r.Between(11*qq+58, 12*qq+62))
			if t == 0 {
				a *= 5
			}
			pk := ((7*k)/10 + 3) * 3
			cval := 10*k + r.Between(3, 20)
			if cval >= tt && r.Between(0, pk) != 0 {
				m.Offers[t][k] = 0
				continue
			}
			mlt := int64(r.Between(m.Popularity/7+100, m.Popularity/5+100))
			offer = offer * int64(a) * mlt / 100
			years := r.Between(0, 2)
			m.OfferYears[t][k] = years + 1
			lg := 2 - m.League
			if m.League == 1 {
				offer = offer * 107 / 100
			}
			offer = offer * int64(r.Between((years+9)*10, years*15+92)) / 100
			offer = offer*int64(r.Between((lg+5)*20, 23*lg+102))/100 + int64(s.Level*800) + 16500
			if t == 0 {
				offer += int64(s.Level*2000) - 11000
			}
			m.Offers[t][k] = offer
		}
	}
}

// Sponsor names of the offers (index k), as listed on the advertising
// screen.
var SponsorNames = [10]string{"Foto Ohne", "KBC", "Stahl KG", "V-Bank", "Pirelli", "Amiga Joker", "Gnomio", "Commodore", "Kron Software", "Software 2000"}

// AcceptTrikot accepts shirt offer k (f_262f_2be8).
func (s *State) AcceptTrikot(mi, k int) error {
	m := s.Managers[mi]
	if m.TrikotMonths != 0 {
		return CreditError("Sie stehen noch unter Vertrag")
	}
	m.TrikotMonths = m.OfferYears[0][k] * 12
	m.TrikotIndex = k
	m.Ads[0] = m.Offers[0][k]
	return nil
}

// AcceptBoard accepts board offer k for board b.
func (s *State) AcceptBoard(mi, b, k int) error {
	m := s.Managers[mi]
	if m.BoardMonths[b] != 0 {
		return CreditError("Sie stehen noch unter Vertrag")
	}
	m.BoardMonths[b] = m.OfferYears[1][k] * 12
	m.BoardIndex[b] = k
	m.Ads[1+b] = m.Offers[1][k]
	m.Offers[1][k] = 0
	return nil
}

// ChangeAdSpending changes the Werbeausgaben by ±2500 DM, wrapping around
// in 2500..50000.
func (s *State) ChangeAdSpending(mi int, up bool) {
	m := s.Managers[mi]
	if up {
		m.Ads[8] += 2500
	} else {
		m.Ads[8] -= 2500
	}
	if m.Ads[8] > 50000 {
		m.Ads[8] = 2500
	} else if m.Ads[8] < 2500 {
		m.Ads[8] = 50000
	}
}

// christmas runs the special dates (f_1a58_2a06): only Christmas costs
// money.
func (s *State) specialDate(k int) {
	if s.Rand.Between(0, 3) != 0 {
		return
	}
	if k != 0 {
		return
	}
	if s.Rand.Between(0, 1) != 0 {
		x := int64(s.Rand.Between(15, 85) * 10000)
		for mi, m := range s.Managers {
			m.Money -= x / int64(m.League+1)
			s.notify(mi, "Frohe Weihnachten! Ihre Spieler sind überwältigt von den Weihnachtspaketen: "+
				FormatMoney(x/int64(m.League+1))+" DM.")
		}
	}
}
