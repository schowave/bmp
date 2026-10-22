package game

import "strconv"

// trainingMatrix is D:0x292: effect of the training points (Kondition,
// Spiel, Schuss, Taktik) on Kondition, Technik and Form.
var trainingMatrix = [4][3]int{{80, 5, 15}, {30, 40, 30}, {20, 25, 55}, {5, 65, 30}}

// winterBreak reports the days of the winter break (window of the spec).
func (s *State) winterBreak() bool { return s.Day > 130 && s.Day < 207 }

// injuryCountdown counts down injuries once a week and clears the status
// when a player is fit again (f_0cb5_2b88).
func (s *State) injuryCountdown(slot *Slot) bool {
	changed := false
	if slot.Status&2 != 0 && slot.Out > 0 && s.Day%7 == 0 {
		slot.Out--
	}
	if slot.Status&3 != 0 && slot.Out == 0 {
		slot.Status &^= 3
		changed = true
	}
	return changed
}

// turn runs the twice-weekly processing of manager mi (f_0cb5_13bd).
func (s *State) turn(mi int) {
	if s.Day > 321 {
		return
	}
	m := s.Managers[mi]
	r := &s.Rand
	market := &s.Squads[MarketIndex]
	if mi == 0 {
		for i := 0; i < MarketSlots; i++ {
			sl := &market[i]
			if sl.Empty() {
				continue
			}
			if sl.Fatigue > 56 {
				sl.Fatigue -= r.Between(3, 9)
			}
			if r.Between(0, 3) == 0 && sl.Status > 0x1f && sl.Message {
				sl.Status &= 0x1f
				sl.Message = false
			}
		}
	}
	bit := 1 << mi
	for i := 0; i < MarketSlots; i++ {
		sl := &market[i]
		if sl.Empty() {
			continue
		}
		if sl.Rejected&0x7f == bit && r.Between(0, 2) == 0 {
			sl.Rejected ^= bit
		}
		if s.Persons[sl.Person].Owner != mi {
			continue
		}
		s.injuryCountdown(sl)
		if sl.Nego&0x80 == 0 {
			x := int(s.PlayerValue(MarketIndex, i, 0) / 10000)
			if r.Between(0, 180) < x && r.Between(0, 2) != 0 && !sl.Message {
				sl.Status |= 0x80
				sl.Buyer = s.chooseBuyer(x, false)
				sl.Message = true
				s.notify(mi, s.Teams[sl.Buyer].Name+" bietet für "+s.Persons[sl.Person].Name+".")
			}
		}
	}
	s.stadiumEvents(mi)
	noMatch := s.Events[s.Event] == 0
	relineup := false
	sq := &s.Squads[mi]
	for i := 0; i < s.countLineup(mi, 0); i++ {
		sl := &sq[i]
		p := &s.Persons[sl.Person]
		base := (boolInt(noMatch)*25 - sl.Fatigue + 160) * 2
		if base < 40 {
			base = 40
		}
		if sl.Out == 0 && sl.Status == 0 && r.Between(0, 30*s.Level+base+80) == 0 {
			s.injure(sl)
			s.notify(mi, "Verletzung im Training: "+p.Name+" ("+InjuryNames[sl.Injury]+")")
			relineup = true
		}
		e := s.retirementAge(sl)
		if rr := r.Between(32, 45); rr < e && sl.Nego&0x80 == 0 && sl.Nego < 100 {
			sl.Nego |= 0x80
			s.notify(mi, p.Name+" kündigt an, daß er seinen Vertrag nicht mehr verlängern wird.")
		}
		if sl.Out != 0 {
			sl.Lineup = 0
			relineup = true
		}
		if r.Between(0, 3) == 0 && sl.Status > 0x1f && sl.Message {
			sl.Status &= 0x1f
			sl.Message = false
		}
		q := sl.Nego
		if q > 99 && q&0x80 == 0 {
			if r.Between(0, 5) == 0 && sl.Message {
				sl.Nego = r.Between(9, 17)
				sl.Message = false
			}
		} else if q != 0 && q&0x80 == 0 {
			sl.Nego--
		}
		if sl.Loan == 0 {
			avg := sl.Strength()
			a := min(8, abs(25-p.Age))
			a = 100 - (100*a)/8
			g := min(5, (sl.Career[0]+sl.Career[1]+sl.Career[2])/36)
			x := (60*avg + 40*a) / 100
			x += (x * g * 10) / -100
			if r.Between(0, x) == 0 && r.Between(0, 3) == 0 && !sl.Message && sl.Nego&0x80 == 0 && sl.Years == 1 {
				rr := r.Between(0, 100)
				n := 2
				if rr > 90 {
					n = 4
				} else if rr > 70 {
					n = 3
				}
				sl.Nego = 100 + n
				sl.Message = true
				s.notify(mi, p.Name+" bietet an, von 1 auf "+strconv.Itoa(n)+" Jahre zu verlängern.")
			}
		}
	}
	if relineup {
		s.AutoLineup(mi)
	}
	for i := 0; i < s.countLineup(mi, 0); i++ {
		sl := &sq[i]
		if sl.Loan != 0 || sl.Nego&0x80 != 0 {
			continue
		}
		v := int(s.PlayerValue(mi, i, 0) / 10000)
		if r.Between(0, 200) < v && r.Between(0, 450) == 0 && !sl.Message {
			sl.Status |= 0x40
			foreign := v >= 91 && r.Between(0, 12) == 0
			sl.Buyer = s.chooseBuyer(v, foreign)
			sl.Message = true
			s.notify(mi, s.Teams[sl.Buyer].Name+" ist an "+s.Persons[sl.Person].Name+" interessiert.")
		}
	}
	s.develop(mi)
	_ = m
}

// retirementAge is f_14a4_2588: the "effective age" for the retirement
// announcement.
func (s *State) retirementAge(sl *Slot) int {
	p := &s.Persons[sl.Person]
	gk := 0
	if p.Position() == PosKeeper {
		gk = 15
	}
	d := (sl.Skill[1] - sl.Skill[0]) >> 1
	if d < 0 {
		d = 0
	}
	return p.Age * (100 - gk - d) / 100
}

// chooseBuyer picks an AI club whose strength fits the value x
// (f_14a4_24bf).
func (s *State) chooseBuyer(x int, foreign bool) int {
	lo, hi := 0, 63
	if foreign {
		lo, hi = 64, 199
	}
	for tries := 0; ; tries++ {
		t := s.Rand.Between(lo, hi)
		if s.ManagerOf(t) != nil {
			continue
		}
		tm := &s.Teams[t]
		q := 0
		for k := 0; k < 3; k++ {
			q += tm.Strength[k] + tm.Base[k] + 50
		}
		q /= 9
		if tries > 500 || (x < q+20 && x+2 > q) {
			return t
		}
	}
}

// develop is the training part of f_0cb5_13bd.
func (s *State) develop(mi int) {
	m := s.Managers[mi]
	r := &s.Rand
	var e [3]int
	for j := 0; j < 3; j++ {
		for i := 0; i < 4; i++ {
			e[j] += trainingMatrix[i][j] * 20 * m.Training[i]
		}
		e[j] /= 400
	}
	b := 37 - boolInt(s.Level == 5)
	window := s.winterBreak()
	sq := &s.Squads[mi]
	for i := 0; i < s.countLineup(mi, 0); i++ {
		sl := &sq[i]
		s.injuryCountdown(sl)
		in := m.Intensity
		if sl.Fatigue > 50 {
			sl.Fatigue -= r.Between(3, 7)
		}
		if window && in < 6 {
			in = 6
		}
		sl.Fatigue += (in - 5) * 2
		if sl.Fatigue > 130 && in < 6 {
			sl.Fatigue -= r.Between(1, 3)
		}
		w := 20 * in
		sl.Fatigue = clamp(sl.Fatigue, 60, 150)
		if sl.Fatigue > 135 {
			w -= r.Between(70-13*s.Level, 90-5*s.Level)
			if w < 10 {
				w = 10
			}
		}
		pos := s.Persons[sl.Person].Position()
		if pos == PosKeeper {
			w += 30
		}
		for j := 0; j < 3; j++ {
			d := e[j] * w / 100
			d = d * sl.Tendency / 50
			g := 10 * m.PosTraining[pos]
			d = d * (g + 75 - r.Between(0, 3)) / 100
			v := sl.Skill[j]
			r0 := boolInt(r.Between(0, 4) == 4)
			dir := 0
			var rr int
			if s.Level > 2 && window {
				if r.Between(0, 2) == 0 {
					r0++
				}
				rr = r.Between(b-r0-1, b+2)
				sl.Fatigue += r.Between(0, 2)
			} else {
				rr = r.Between(b-r0-2, b+2)
			}
			if s.Level < 2 && window {
				sl.Fatigue += r.Between(0, 2)
			}
			if d < rr-8 {
				if r.Between(0, 1) != 0 {
					dir = -1
				}
				rr = (rr - d + 8) / 11
			} else if d > rr+8 {
				if r.Between(0, 1) != 0 {
					dir = 1
				}
				rr = (d - rr + 8) / 11
			}
			if r.Between(0, 3) == 0 && dir != 0 {
				if rr > 2 {
					rr = r.Between(2, 3)
				}
				if rr != 0 {
					v += r.Between(1, rr) * dir
					if j < 2 {
						v = clamp(v, 1, 99)
					} else {
						v = clamp(v, 45, 55)
					}
					sl.Skill[j] = v
				}
			}
		}
		t := sl.Tendency
		if r.Between(0, 3) != 0 {
			step := (sl.Drift >> 4) & 7
			if sl.Drift&0x80 != 0 {
				t += step
			} else {
				t -= step
			}
		}
		n := (sl.Drift & 0x0f) - 1
		sl.Drift = (sl.Drift & 0xf0) + (n & 0x0f)
		if n == 0 {
			s.initTendency(sl)
		} else {
			sl.Tendency = clamp(t, 30, 68)
		}
	}
	s.AutoLineup(mi)
}

// TendencyClass returns 0 falling, 1 constant, 2 rising.
func TendencyClass(t int) int { return clamp((t-30)/13, 0, 2) }

// TendencyNames are the texts of the original.
var TendencyNames = [3]string{"FALLEND.", "KONSTANT.", "STEIGEND."}
