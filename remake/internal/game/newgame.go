package game

import (
	"errors"

	"github.com/schowave/bmp/remake/internal/assets"
)

// GameLength is the option of the new-game dialog.
type GameLength int

// Game lengths.
const (
	Endless GameLength = iota
	OneYear
	ThreeYears
)

// Options of a new game.
type Options struct {
	Managers []NewManager
	// Level as shown in the dialog, 0..5 (internally 5 - Level).
	Level    int
	Length   GameLength
	Historic bool // start in 1963 instead of 1992
	Seed     uint32
}

// NewManager is a human manager of a new game.
type NewManager struct {
	Name string
	Team int // id of the chosen German club (0..63, MANA.DAT order)
	Face int
}

// cp437 converts a CP437 string of MANA.DAT to UTF-8.
func cp437(s string) string {
	m := map[byte]rune{0x8e: 'Ä', 0x99: 'Ö', 0x9a: 'Ü', 0x84: 'ä', 0x94: 'ö', 0x81: 'ü', 0xe1: 'ß'}
	var r []rune
	for i := 0; i < len(s); i++ {
		if u, ok := m[s[i]]; ok {
			r = append(r, u)
		} else {
			r = append(r, rune(s[i]))
		}
	}
	return string(r)
}

// Constants of the person table (f_322b_035c).
var (
	personBlocks   = [5]int{0, 20, 64, 106, 151} // D:0x7a6
	squadPerPos    = [4]int{2, 5, 8, 5}          // D:0x9438
	manaNameBlocks = [5]int{0, 2, 8, 15, 20}     // D:0x9432
	adDefaults     = [9]int64{50000, 6000, 6000, 6000, 6000, 6000, 6000, 0, 5000}
	campStart      = [8]int{-80, -50, -100, -60, 80, 50, 70, 60}
)

// NewGame sets up a new game from MANA.DAT (f_08bc_0418, f_262f_36ec,
// f_08bc_2184, f_322b_035c).
func NewGame(mana *assets.Mana, opt Options) (*State, error) {
	if len(opt.Managers) == 0 || len(opt.Managers) > MaxManagers {
		return nil, errors.New("1 bis 4 Manager")
	}
	used := map[int]bool{}
	for _, nm := range opt.Managers {
		if nm.Team < 0 || nm.Team >= NumGerman || used[nm.Team] {
			return nil, errors.New("jeder Manager braucht einen eigenen deutschen Verein")
		}
		used[nm.Team] = true
	}
	s := &State{Rand: Rand{Seed: opt.Seed}}
	s.mana = mana
	r := &s.Rand
	for t := 0; t < NumTeams; t++ {
		c := mana.Clubs[t]
		tm := &s.Teams[t]
		tm.Name = cp437(c.Name)
		tm.Logo = c.Logo
		for k := 0; k < 3; k++ {
			tm.Strength[k] = c.Strength[k]
			tm.Base[k] = c.Strength[k]
			tm.Extra[k] = r.Between(45, 55) - 5*boolInt(t > 64)
		}
		tm.Mood = r.Between(45, 55)
	}
	for l := 0; l < NumLeagues; l++ {
		for i := 0; i < TeamsPerLeague[l]; i++ {
			s.Tables[l][i] = FirstID[l] + i
		}
	}
	for t := range s.Clubs {
		s.Clubs[t].Form, s.Clubs[t].FormSnap = emptyForm, emptyForm
	}
	s.Events = seasonEvents
	s.Matchday = [3]int{1, 1, 1}
	s.CampTimers = campStart
	s.BankBase = 5
	s.BankRates = [6]int{7, 6, 5, 4, 3, 2}
	s.Cups.Holder[CupDFB] = 7
	for i, nm := range opt.Managers {
		m := &Manager{Name: nm.Name, Number: i + 1, Team: nm.Team, Face: nm.Face, AutoLineup: 2}
		m.CupStatus = [4]int{NotInCup, NotInCup, NotInCup, NotInCup}
		s.Managers = append(s.Managers, m)
	}
	// Remember the chosen clubs by name: the id shuffle moves them.
	chosen := make([]string, len(opt.Managers))
	for i, nm := range opt.Managers {
		chosen[i] = s.Teams[nm.Team].Name
	}
	s.shuffleIDs()
	for i := range s.Managers {
		s.Managers[i].Team = s.teamByName(chosen[i])
	}
	s.qualifyEurope()
	s.clearCupResults()
	for c := 1; c < NumCups; c++ {
		s.drawCup(c)
	}
	// The original draws before the managers exist.
	for _, m := range s.Managers {
		m.CupStatus = [4]int{NotInCup, NotInCup, NotInCup, NotInCup}
	}

	// Options.
	s.Level = 5 - clamp(opt.Level, 0, 5)
	switch opt.Length {
	case OneYear:
		s.EndYear = 1964
	case ThreeYears:
		s.EndYear = 1966
	default:
		s.EndYear = 22222
	}
	s.Modern = !opt.Historic
	s.StartYear = 1963
	if s.Modern {
		s.EndYear += 29
		s.StartYear = 1992
	}
	s.initPersons()
	for mi := range s.Managers {
		s.initManager(mi, opt.Length)
	}
	s.driftBankRates()
	s.driftStrengths(10)
	s.reshufflePersons()
	s.drawCup(CupDFB)
	s.Draws = []int{CupDFB, CupLandesmeister, CupPokalsieger, CupUEFA}
	s.Display = DefaultDisplay()
	s.Day = 0
	// The table places before the first matchday are the table order.
	for l := 0; l < NumLeagues; l++ {
		for i, id := range s.Tables[l][:TeamsPerLeague[l]] {
			s.Clubs[id].Pos = i
		}
	}
	s.refreshMarket()
	for mi := range s.Managers {
		s.AutoLineup(mi)
		s.LineupStrength(mi, false)
	}
	return s, nil
}

func (s *State) teamByName(name string) int {
	for t := 0; t < NumGerman; t++ {
		if s.Teams[t].Name == name {
			return t
		}
	}
	return -1
}

// placeClub moves the club of a manager into a slot of the start league
// (f_08bc_2184 step 5).
func (s *State) placeClub(mi, l0 int, short bool) {
	r := &s.Rand
	m := s.Managers[mi]
	id := m.Team
	pick := func(inUEFA bool) int {
		for {
			x := r.Between(FirstID[l0], FirstID[l0]+TeamsPerLeague[l0]-1)
			taken := false
			for _, o := range s.Managers[:mi] {
				if o.Team == x {
					taken = true
				}
			}
			if taken {
				continue
			}
			if inUEFA && !s.inCupDraw(x, CupUEFA) || !inUEFA && s.inEuropeDraw(x) {
				continue
			}
			return x
		}
	}
	if l0 == 0 || LeagueOf(id) != l0 || id > 57 {
		x := pick(false)
		s.swapIdentity(id, x, false)
		s.Clubs[id], s.Clubs[x] = s.Clubs[x], s.Clubs[id]
	}
	if short {
		m.CupStatus[CupUEFA] = 1
		x := pick(true)
		y := m.Team
		s.swapIdentity(y, x, false)
		s.Clubs[y], s.Clubs[x] = s.Clubs[x], s.Clubs[y]
	}
}

// initManager gives manager mi of a new game, or one added later with NEU,
// the start values of f_08bc_2184: club placement, stadium, squad, money.
func (s *State) initManager(mi int, length GameLength) {
	m := s.Managers[mi]
	endless := length == Endless
	l0 := 0
	if endless {
		l0 = 2
	}
	short := length == OneYear
	if short {
		m.Ads = [9]int64{180000, 45000, 45000, 45000, 45000, 45000, 45000, 0, 30000}
	} else {
		m.Ads = adDefaults
	}
	m.Money = 1500000
	if s.Level == 4 {
		m.Money = 1900000
	}
	m.MinAudience = 99999
	s.placeClub(mi, l0, short)
	m.League = l0
	m.TicketPrice = (boolInt(short)+8)*2 - 3*l0
	m.Stadium[StStanding] = int64((5 - l0) * 4000)
	m.Stadium[StSeats] = int64((2 - l0) * 3000)
	if short {
		m.Stadium[StSeats] += 5000
		m.Stadium[StStanding] += 10000
	}
	m.Stadium[StCovered] = int64((2 - l0) * 3500)
	m.Stadium[StCondition] = int64(4 + boolInt(short))
	m.Stadium[StComfort] = int64(3 + boolInt(short))
	if !endless {
		three := boolInt(length == ThreeYears)
		m.Stadium[StScoreboard] = int64(3 - 2*three)
		m.Stadium[StFloodlight] = int64(3 - three)
	}
	m.Training = [4]int{5, 4, 6, 5}
	m.Intensity = 6
	m.PosTraining = [4]int{1, 3, 3, 3}
	m.Einsatz = 16
	for k := range m.PosAfter {
		m.PosAfter[k] = 10
	}
	m.Popularity = 50 - 20*l0
	s.startSquad(mi, l0, short)
	s.seasonBudget(mi)
	s.sponsorOffers(mi)
}

// Length returns the game length of the running game, from its end year.
func (s *State) Length() GameLength {
	switch s.EndYear {
	case 1964, 1993:
		return OneYear
	case 1966, 1995:
		return ThreeYears
	}
	return Endless
}

// AddManager adds a manager to a running game (NEU in the disk menu): he
// gets club team (an id of the current table) and the start values of a new
// game.
func (s *State) AddManager(name string, team, face int) error {
	if len(s.Managers) >= MaxManagers {
		return errors.New("Es können höchstens vier Manager spielen.")
	}
	for _, m := range s.Managers {
		if m.Team == team {
			return errors.New("Der Verein hat schon einen Manager.")
		}
	}
	mi := len(s.Managers)
	m := &Manager{Name: name, Number: mi + 1, Team: team, Face: face, AutoLineup: 2}
	m.CupStatus = [4]int{NotInCup, NotInCup, NotInCup, NotInCup}
	s.Managers = append(s.Managers, m)
	s.initManager(mi, s.Length())
	s.AutoLineup(mi)
	s.LineupStrength(mi, false)
	return nil
}

// inCupDraw reports whether a club is drawn in cup c.
func (s *State) inCupDraw(id, c int) bool {
	for _, x := range s.Cups.Slots[c] {
		if x == id {
			return true
		}
	}
	return false
}

// inEuropeDraw reports whether a club is drawn in a European cup
// (f_08bc_1df3).
func (s *State) inEuropeDraw(id int) bool {
	for c := 1; c < NumCups; c++ {
		for _, x := range s.Cups.Slots[c] {
			if x == id {
				return true
			}
		}
	}
	return false
}

// manaName returns name n of the roster of MANA.DAT club t.
func (s *State) manaName(t, n int) string {
	if s.mana == nil || t < 0 || t >= assets.NumGermanClubs || n >= len(s.mana.Clubs[t].Players) {
		return ""
	}
	return cp437(s.mana.Clubs[t].Players[n])
}

// randomName picks an unused name of a non-human Bundesliga club
// (f_322b_022e).
func (s *State) randomName(k, p int) string {
	r := &s.Rand
	for {
		n := r.Between(manaNameBlocks[k], manaNameBlocks[k+1]-1)
		var t int
		for {
			t = r.Between(0, TeamsPerLeague[0]-1)
			if s.ManagerOf(t) == nil {
				break
			}
		}
		name := s.manaName(s.manaIndex(t), n)
		if name == "" {
			continue
		}
		used := false
		for q := 1; q < NumPersons; q++ {
			if s.Persons[q].Name == name {
				used = true
				break
			}
		}
		if used {
			continue
		}
		if p != 0xff {
			s.Persons[p].Team = t
		}
		return name
	}
}

// manaIndex returns the MANA.DAT record of the club now in slot t.
func (s *State) manaIndex(t int) int {
	for i := 0; i < assets.NumGermanClubs; i++ {
		if cp437(s.mana.Clubs[i].Name) == s.Teams[t].Name {
			return i
		}
	}
	return t
}

// initPersons creates the person table and the squads of the managers
// (f_322b_035c).
func (s *State) initPersons() {
	r := &s.Rand
	for p := 1; p < NumPersons; p++ {
		ps := &s.Persons[p]
		ps.Age = r.Between(18, 33)
		b := r.Between(30, 89)
		if b < 45 && r.Between(0, 1) != 0 {
			b = r.Between(30, 89)
		}
		ps.Skill[1] = b + 10 - r.Between(0, 20)
		ps.Skill[2] = r.Between(45, 55)
		ps.Skill[0] = b + 10 - r.Between(0, 20)
		ps.Avg = (ps.Skill[0] + ps.Skill[1] + ps.Skill[2]) / 3
		ps.Side = r.Between(0, 6)
		ps.Owner = OwnerFree
		ps.Team = 0xff
	}
	counter := [4]int{19, 44, 42, 45} // D:0x943c
	for mi, m := range s.Managers {
		s.Squads[mi] = [SquadSlots]Slot{}
		i := 0
		for k := 0; k < 4; k++ {
			for c := 0; c < squadPerPos[k]; c++ {
				p := personBlocks[k+1] - counter[k] + 1
				counter[k]--
				name := s.manaName(s.manaIndex(m.Team), i)
				if name == "" {
					name = s.randomName(k, 0xff)
				}
				ps := &s.Persons[p]
				ps.Name = name
				i++
				ps.Owner = mi
				ps.Age = r.Between(19, 31)
				ps.Team = m.Team
				s.addPerson(mi, p, 2, 0, SquadMax)
				ps.Team = m.Team
			}
		}
	}
	k := 0
	for p := 1; p < NumPersons; p++ {
		if personBlocks[k+1] <= p {
			k++
		}
		if s.Persons[p].Name == "" {
			s.Persons[p].Name = s.randomName(k, p)
			if s.EndYear > 2000 {
				s.Persons[p].Team = 0xff
			}
		}
	}
	for p := 1; p < NumPersons; p++ {
		ps := &s.Persons[p]
		switch {
		case p <= 20:
			ps.Pos = 0
		case p <= 63:
			ps.Pos = 25 + r.Between(0, 16)
		case p <= 105:
			ps.Pos = 50 + r.Between(0, 16)
		default:
			ps.Pos = 75 + r.Between(0, 16)
		}
	}
}

// startSquad sets the skills and salaries of a new manager's squad
// (f_08bc_2184).
func (s *State) startSquad(mi, l0 int, short bool) {
	r := &s.Rand
	lo := s.Level/2 + 68
	if l0 == 2 {
		lo = s.Level/2 + 27
	}
	if short {
		lo += 12
	}
	// Like the original the loop runs over the squad sizes of all managers
	// but always writes this manager's squad.
	for k := range s.Managers {
		for i := 0; i < s.countLineup(k, 0); i++ {
			sl := &s.Squads[mi][i]
			for j := 0; j < 3; j++ {
				v := r.Between(lo, lo+5)
				sl.Skill[j] = v
				f := r.Between(40, 60)
				if j == 2 {
					sl.Skill[2] = f
				}
			}
		}
	}
	for i := 0; i < s.countLineup(mi, 0); i++ {
		s.Squads[mi][i].Salary = s.PlayerValue(mi, i, ValueSalary)
	}
}

// reshufflePersons lets the skills of all persons drift and moves some
// foreign players to German clubs (f_14a4_1798).
func (s *State) reshufflePersons() {
	r := &s.Rand
	foreign := 0
	for p := 1; p < NumPersons; p++ {
		ps := &s.Persons[p]
		ps.Skill[0] = clamp(ps.Skill[0]+r.Between(1, 7)-4, 30, 99)
		ps.Skill[1] = clamp(ps.Skill[1]+r.Between(1, 7)-4, 30, 99)
		if ps.Team >= NumGerman && ps.Team != 0xff {
			foreign++
		}
	}
	n := r.Between(5, 15)
	foreignTeam := func(t int) bool { return t >= NumGerman && t != 0xff }
	move := func(p int) {
		ps := &s.Persons[p]
		x := (ps.Skill[0] + ps.Skill[1]) / 2 * 58 / 100
		ps.Team = s.chooseBuyer(x, false)
	}
	for i := 0; i < n; i++ {
		for tries := 0; tries < 10000; tries++ {
			p := r.Between(1, 150)
			t0 := s.Persons[p].Team
			done := false
			if foreign > 4 && foreignTeam(t0) {
				foreign--
				move(p)
				done = true
			}
			human := t0 != 0xff && t0 < NumTeams && s.ManagerOf(t0) != nil
			if !human && (done || foreign <= 4) {
				if !done {
					move(p)
				}
				break
			}
		}
	}
}
