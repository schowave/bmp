package game

import "math"

// Construction costs per 1000 places (items 0..2) or per level (3..6)
// (D:0x44) and base construction times in weeks (D:0x63).
var (
	BuildCost  = [NumStadiumItems]int64{150000, 100000, 160000, 380000, 440000, 550000, 650000}
	buildWeeks = [NumStadiumItems]int{15, 15, 20, 10, 10, 8, 4}
)

// Texts of the stadium screen.
var (
	SizeNames   = [4]string{"IS NICH", "KLEIN", "MITTEL", "GROß"}
	GradeNames  = [7]string{"", "UNGENUEGEND", "MANGELHAFT", "AUSREICHEND", "BEFRIEDIGEND", "GUT", "SEHR GUT"}
	BuildTitles = [8]string{"SITZPLÄTZE AUSBAUEN", "STEHPLÄTZE AUSBAUEN", "PLÄTZE ÜBERDACHEN",
		"FLUTLICHTMASTEN AUSBAUEN", "ANZEIGETAFEL ERRICHTEN", "STADION RENOVIEREN", "KOMFORT VERBESSERN",
		"EINTRITTSPREIS ÄNDERN"}
)

// Capacity returns seats plus standing places.
func (m *Manager) Capacity() int64 { return m.Stadium[StSeats] + m.Stadium[StStanding] }

// constructionProgress advances the construction sites of manager mi by a
// day (f_0000_20e1). The lockouts are global and, like in the original,
// count down once per manager.
func (s *State) constructionProgress(mi int) {
	m := s.Managers[mi]
	for k := 0; k < NumStadiumItems; k++ {
		if s.BuildLockout[k] != 0 {
			s.BuildLockout[k]--
		}
		if m.BuildTimer[k] == 0 {
			continue
		}
		m.BuildTimer[k]--
		if m.BuildTimer[k] == 0 {
			m.Stadium[k] += m.StadiumBuild[k]
			m.StadiumBuild[k] = 0
			if s.popupWindow() {
				s.popup(mi, BackdropDesk, msgCompletion(k))
			} else {
				s.notify(mi, msgCompletion(k))
			}
		}
	}
}

// BuildLimit describes what can be ordered for a stadium item.
type BuildLimit struct {
	Locked  bool  // no construction company available
	Max     int64 // places (items 0..2) or highest level (3..6)
	Now     int64 // current level incl. pending construction (3..6)
	Top     int64 // best possible level (3..6)
	NoMoney bool
}

// StadiumLimit returns the limits of the construction dialog for item k
// (f_0000_0602).
func (s *State) StadiumLimit(mi, k int) BuildLimit {
	m := s.Managers[mi]
	st, pd := &m.Stadium, &m.StadiumBuild
	if s.BuildLockout[k] != 0 {
		return BuildLimit{Locked: true}
	}
	if k <= StCovered {
		var capMax int64
		if k == StCovered {
			capMax = st[StSeats] + st[StStanding] - st[StCovered] - pd[StCovered]
		} else {
			capMax = 130000 - (st[StSeats] + pd[StSeats] + st[StStanding] + pd[StStanding])
		}
		afford := m.Money / BuildCost[k] * 1000
		if afford <= 0 {
			return BuildLimit{NoMoney: true}
		}
		if afford < capMax {
			capMax = afford
		}
		if capMax < 0 {
			capMax = 0
		}
		return BuildLimit{Max: capMax / 1000 * 1000}
	}
	top := int64(3)
	if k >= StCondition {
		top = 6
	}
	now := st[k] + pd[k]
	if now >= top {
		return BuildLimit{Now: now, Top: top, Max: now}
	}
	afford := m.Money / BuildCost[k]
	if afford <= 0 {
		return BuildLimit{Now: now, Top: top, NoMoney: true}
	}
	if afford > top {
		afford = top
	}
	max := now + afford
	if max > top {
		max = top
	}
	return BuildLimit{Now: now, Top: top, Max: max}
}

// BuildOrder is a quote for a construction order.
type BuildOrder struct {
	Item   int
	Amount int64 // places, or the new level for items 3..6
	Cost   int64
	Days   int
	Again  bool // an order for this item is already running
}

// QuoteBuild computes cost and construction time of an order (first part
// of f_0000_0000). It draws the random construction time.
func (s *State) QuoteBuild(mi, k int, amount int64) BuildOrder {
	m := s.Managers[mi]
	var cost int64
	if k <= StCovered {
		cost = amount / 1000 * BuildCost[k]
	} else {
		cost = (amount - m.Stadium[k] - m.StadiumBuild[k]) * BuildCost[k]
	}
	old := m.BuildTimer[k] * s.Rand.Between(20, 80) / 100
	days := (old + 7*buildWeeks[k]) * s.Rand.Between(80, 120) / 100
	return BuildOrder{Item: k, Amount: amount, Cost: cost, Days: days, Again: old != 0}
}

// ErrNoMoney is the refusal of a construction order.
var ErrNoMoney = CreditError("Zu wenig Kleingeld vorhanden...")

// ConfirmBuild places an order quoted by QuoteBuild ("NA KLAR !").
func (s *State) ConfirmBuild(mi int, o BuildOrder) error {
	m := s.Managers[mi]
	if m.Money-o.Cost < 0 {
		return ErrNoMoney
	}
	m.Money -= o.Cost
	k := o.Item
	m.StadiumBuild[k] += o.Amount
	if k > StCovered {
		m.StadiumBuild[k] = o.Amount - m.Stadium[k]
	}
	m.BuildTimer[k] = int(uint16(o.Days))
	return nil
}

// DeclineBuild is "ACH NEE...": no construction company for this item for
// 15..55 days.
func (s *State) DeclineBuild(k int) {
	s.BuildLockout[k] = s.Rand.Between(15, 55)
}

// SetTicketPrice changes the ticket price (5..25 DM).
func (s *State) SetTicketPrice(mi, price int) {
	s.Managers[mi].TicketPrice = clamp(price, 5, 25)
}

// StadiumPicture returns the picture number of the stadium (f_0000_16e3).
func StadiumPicture(m *Manager) int {
	total := m.Capacity()
	size := boolInt(total > 14000) + boolInt(total > 34000) + boolInt(total > 54000)
	return 11 + 3*size - int(m.Stadium[StCondition]-1)/2
}

var (
	aiCapacity  = [4]int{4500, 2200, 1200, 600} // D:0x2c8, ×10
	AITicket    = [4]int{16, 14, 10, 8}         // D:0x5390
	leagueRange = [4][2]int{{0, 17}, {18, 37}, {38, 57}, {58, 255}}
)

// bucket returns the league class of a club id (D:0x2272 ranges).
func bucket(id int) int {
	for j, r := range leagueRange {
		if id >= r[0] && id <= r[1] {
			return j
		}
	}
	return 3
}

// Attendance computes the crowd of a match in the stadium of manager mi
// (f_0f9d_11e0). imp is the importance (1 league/DFB, 2 Europe, 3 late
// Europe rounds); generic uses an estimated capacity for an AI home club.
func (s *State) Attendance(mi, h, a, imp int, generic bool, price int) int64 {
	m := s.Managers[mi]
	r := &s.Rand
	st := &m.Stadium
	pos := func(t int) int {
		if t < NumGerman {
			return s.Clubs[t].Pos
		}
		return 5
	}
	hp, ap := pos(h), pos(a)
	hs, as := s.TeamStrength(h), s.TeamStrength(a)
	if price == 0 {
		price = m.TicketPrice
	}
	l := m.League
	lf := 1 << (2 - l)
	if lf == 4 && hp > 14 {
		hp = 14
	}
	if hs == 0 && as == 0 {
		return 1
	}
	x := ((1-float64(ap-1)*2.0*0.01)*float64(as) + (1-float64(hp-1)*2.0*0.01)*float64(hs)) *
		0.23255813953488372 * 1150.0
	rr := r.Between16((price-10)*1050, price*1550-14400)
	att := int64(math.Trunc(x - float64(rr) + 1000.0))
	q := (hs + as) * 100 / 180
	att += int64(r.Between16(5*q, 9*q))*st[StComfort] - int64(20*q)
	att += int64(r.Between16(4*q, 12*q))*st[StCondition] - int64(31*q)
	att += (st[StFloodlight]*10 - 8) * int64(q)
	att += (st[StScoreboard]*10 - 5) * int64(q)
	att += (st[StCovered]/1000 - 8) * int64(q)
	if m.Popularity > 50 {
		att += int64(uint16(30*m.Popularity - 1500))
	}
	att -= int64(2000 * l)
	if s.Level != 0 {
		att += 1200
	}
	if st[StSeats]+2000 > st[StStanding] {
		att += int64(r.Between(400, 700))
	}
	md, nmd := s.Matchday[l], MatchdaysPerLeague[l]
	var capacity int64
	if generic {
		t := aiCapacity[bucket(h)]
		capacity = int64(r.Between(t-t/3, t+t/3)) * 10
	} else {
		capacity = st[StSeats] + st[StStanding]
	}
	if imp > 3 {
		imp = 3
	}
	if l == 2 {
		imp = 0
	}
	if as < 90 && imp > 1 {
		imp--
	}
	if price < 22 {
		att = att * int64(imp+2) / 3
	}
	if md+5 > nmd && price < 22 {
		if md == nmd {
			att = capacity
		} else {
			k := boolInt(l != 0)
			if hp < 7-2*k {
				att += capacity / 4
				if ap < 7 {
					att += capacity / 5
				}
			}
			if hp+k+5 > TeamsPerLeague[l] {
				att += capacity / 4
			}
		}
	}
	if price > 10 {
		att += int64((10 - price) * 900)
	}
	limit := int64(2*lf-boolInt(lf == 4)) * 100000
	for att*int64(price) > limit {
		att -= 1000
	}
	for att <= 500 {
		att += 500
	}
	if att > capacity {
		att = capacity
	}
	return att
}

// riotDamage books the damage of a riot at the last home game and the
// comfort downgrade (stadium part of f_0cb5_13bd).
func (s *State) stadiumEvents(mi int) {
	m := s.Managers[mi]
	r := &s.Rand
	st := &m.Stadium
	if m.RiotPending {
		dmg := (st[StSeats]/2 + 4*st[StStanding]) * int64(r.Between(2, 5)) / 1000 * 1000
		m.Money -= dmg
		m.RiotPending = false
		s.notify(mi, msgRiot(dmg))
		if st[StCondition] > 1 {
			if m.Money > 2000000 && r.Between(0, 1) == 0 {
				st[StCondition]--
			} else if r.Between(0, 2) == 0 {
				st[StCondition]--
			}
		}
	}
	if r.Between(0, int(442-52*st[StComfort])) == 0 && st[StComfort] > 1 && m.League == 0 {
		st[StComfort]--
		s.notify(mi, msgComfort())
	}
}

// Training camps (f_112a_0145, f_112a_072d).
var (
	CampNames = [8]string{"Kickfelder Mühle", "Trlg. Hackstedt", "Hof Schweißperle", "Fitnesscenter Krone",
		"Scola Ludum Pilae", "Sportschule Träne", "Hotel Medium", "Luxushotel La Preux"}
	// CampRatings are E, F, P, R, V (Erholungswert, Fitnessausrüstung,
	// Platz-Qualität, Raumausstattung, Verpflegung), 1..5.
	CampRatings = [8][5]int{{1, 2, 1, 1, 1}, {1, 2, 3, 1, 1}, {1, 3, 2, 1, 1}, {2, 4, 3, 2, 2},
		{2, 1, 2, 4, 4}, {2, 4, 3, 2, 2}, {3, 2, 2, 3, 4}, {5, 3, 2, 4, 5}}
	CampRatingNames = [5]string{"Erholungswert", "Fitnessausrüstung", "Platz-Qualität", "Raumausstattung", "Verpflegung"}
	campFactor      = [8]int64{5, 6, 7, 10, 11, 10, 13, 15}
	campWeights     = [4][3]int{{20, 20, 40}, {60, 10, 20}, {20, 50, 20}, {10, 40, 30}}
)

// CampPrice returns the price of camp c per week. The original scales the
// prices once per program run with the squad of whoever opens the screen
// first; the port uses the squad of the manager asking.
func (s *State) CampPrice(mi, c int) int64 {
	return campFactor[c] * int64(s.SquadSize(mi)) * 2000
}

// CampOpen reports whether camp c is open.
func (s *State) CampOpen(c int) bool { return s.CampTimers[c] <= 0 }

// Camp errors.
var (
	ErrCampClosed  = CreditError("hat nicht geöffnet...")
	ErrCampMoney   = CreditError("Keine finanzielle Möglichkeit gegeben...")
	ErrCampTooSoon = CreditError("Einmal pro Woche reicht doch...")
)

// BookCamp sends the squad of manager mi to camp c.
func (s *State) BookCamp(mi, c int) error {
	m := s.Managers[mi]
	if m.CampDays != 0 {
		return ErrCampTooSoon
	}
	if !s.CampOpen(c) {
		return ErrCampClosed
	}
	price := s.CampPrice(mi, c)
	if m.Money < price {
		return ErrCampMoney
	}
	m.Money -= price
	s.campEffect(mi, c)
	return nil
}

func (s *State) campEffect(mi, c int) {
	r := &s.Rand
	var e [3]int
	for k := 0; k < 3; k++ {
		for j := 0; j < 4; j++ {
			e[k] += CampRatings[c][j] * campWeights[j][k]
		}
		e[k] /= 5
	}
	for i := 0; i < s.countLineup(mi, 0); i++ {
		sl := &s.Squads[mi][i]
		d := (e[0]+e[1]+e[2])/(-3) + (e[0]-e[2]+10)*2 + e[1]
		if d < 80 {
			d = 80
		}
		sl.Fatigue = clamp(sl.Fatigue+r.Between(d/9, d/6), 60, 150)
		t := 0
		for j := 0; j < 3; j++ {
			t = (r.Between(2*s.Level+87, 2*s.Level+117) + e[j] - 100) / 10
			a := r.Between(0, t*20/30)
			b := r.Between(0, t)
			t = t + b - a
			sl.Skill[j] += t >> 2
			if sl.Skill[j] > 99 {
				sl.Skill[j] = 99
			}
		}
		t += t * e[2] / 200
		if r.Between(0, 4) == 0 {
			sl.Tendency += r.Between(155, 300) * t / 800
		}
		s.initTendency(sl)
	}
	s.Managers[mi].CampDays = r.Between(7, 18)
}
