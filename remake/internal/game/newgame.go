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
	Historic bool // start in historicYear instead of modernYear
	// Season is the real season whose clubs mana holds ("1995-96"), empty
	// for the clubs of the original; Year is its first year.
	Season string
	Year   int
	Seed   uint32
}

// NewManager is a human manager of a new game.
type NewManager struct {
	Name string
	// Team is the id of the chosen German club in MANA.DAT order (0..63).
	// NewGame moves the club to the id and league the start year gives it:
	// read Manager.Team and Manager.League after NewGame.
	Team int
	Face int
}

// Start years of the original (D:0x7e0 at a new game): the historic and the
// modern one; lastEndYear is above the end year of every game that ends.
const (
	historicYear = 1963
	modernYear   = 1992
	lastEndYear  = 9999
)

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
		tm.Name = FromCP437(c.Name)
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
	for c := CupLandesmeister; c < NumCups; c++ {
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
	s.StartYear = historicYear
	if s.Modern {
		s.EndYear += 29
		s.StartYear = modernYear
	}
	if opt.Season != "" {
		// A real season starts in its own year; the end year keeps its
		// distance to the start.
		s.Season, s.Modern = opt.Season, true
		s.EndYear += opt.Year - s.StartYear
		s.StartYear = opt.Year
	}
	s.FirstYear = s.StartYear
	s.initPersons()
	for mi := range s.Managers {
		s.initManager(mi, opt.Length, false)
	}
	s.driftBankRates()
	s.driftStrengths(10)
	s.distributePersons()
	s.drawCup(CupDFB)
	for c := CupDFB; c < NumCups; c++ {
		s.queueDraw(c)
	}
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
func (s *State) placeClub(mi, l0 int, short, late bool) {
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
	// Only a new 1-year game puts the club into the UEFA-Pokal.
	if short && !late {
		m.CupStatus[CupUEFA] = 1
		x := pick(true)
		y := m.Team
		s.swapIdentity(y, x, false)
		s.Clubs[y], s.Clubs[x] = s.Clubs[x], s.Clubs[y]
	}
}

// initManager gives manager mi of a new game, or one added later with NEU
// (late), the start values of f_08bc_2184: club placement, stadium, squad,
// money.
func (s *State) initManager(mi int, length GameLength, late bool) {
	m := s.Managers[mi]
	if late {
		s.lateSquad(mi)
	}
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
	s.placeClub(mi, l0, short, late)
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
	m.EffortSetting = 16
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
	switch {
	case s.Endless():
		return Endless
	case s.EndYear-s.FirstYear == 1:
		return OneYear
	}
	return ThreeYears
}

// Endless reports whether the game has no end. The original encodes it as
// the end year 22222 (22251 in a modern game) and tests for a year after
// 2000; with real seasons one- and three-year games end after 2000 too, so
// the remake tests for a year no game reaches.
func (s *State) Endless() bool { return s.EndYear > lastEndYear }

// IsReal reports whether the game plays a real season (Season) instead of
// the clubs of the original.
func (s *State) IsReal() bool { return s.Season != "" }

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
	s.initManager(mi, s.Length(), true)
	s.AutoLineup(mi)
	s.LineupStrength(mi, false)
	return nil
}

var (
	latePlayers = [4]int{2, 5, 8, 5}          // D:0x9438, players per position
	latePersons = [5]int{1, 20, 64, 106, 151} // D:0x7a6, person ranges per position
)

// lateSquad gives a manager added with NEU his players: per position free
// persons of its range (f_08bc_2184). The ranges end on the first person of
// the next position, the last one on 151, behind the person table; the
// remake takes that one as not free.
func (s *State) lateSquad(mi int) {
	for k := 0; k < 4; k++ {
		for n := 0; n < latePlayers[k]; n++ {
			p := 0
			for try := 0; try < 1000; try++ {
				p = s.Rand.Between(latePersons[k], latePersons[k+1])
				if p < NumPersons && s.Persons[p].Owner == OwnerFree {
					break
				}
			}
			if p >= NumPersons {
				continue
			}
			s.Persons[p].Owner = mi
			_, _ = s.addPerson(mi, p, 1, 0, SquadMax)
		}
	}
}

// inCupDraw reports whether a club is drawn in cup c.
func (s *State) inCupDraw(id int, c Cup) bool {
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
	for c := CupLandesmeister; c < NumCups; c++ {
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
	return FromCP437(s.mana.Clubs[t].Players[n])
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
		if FromCP437(s.mana.Clubs[i].Name) == s.Teams[t].Name {
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
				_, _ = s.addPerson(mi, p, 2, 0, SquadMax)
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
			if s.Endless() {
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

// distributePersons ends the setup of a new game (f_14a4_19fb): after a
// reshuffle every person without a club joins a club of each league with
// a share left, so the last of them keeps him. The original also does this
// for persons with a club once the year (F:0xa7a0) is set, which it is not
// yet in a new game.
func (s *State) distributePersons() {
	share := s.reshufflePersons()
	for p := 1; p < NumPersons; p++ {
		if s.Persons[p].Team != 0xff {
			continue
		}
		for k := range share {
			if share[k] != 0 {
				s.Persons[p].Team = s.leagueTeam(k)
				share[k]--
			}
		}
	}
}

// reshufflePersons lets the skills of all persons drift and moves some
// foreign players to German clubs (f_14a4_1798). Each of the n rounds draws
// persons until one is neither moved by the draw nor at a human club while
// at most 4 foreign players are left, and moves him too; so the first round
// brings home all foreign players beyond 4. It returns how many persons
// each league should get: its share of the human squads, applied to the
// persons neither in a human squad nor abroad.
func (s *State) reshufflePersons() [NumLeagues]int {
	r := &s.Rand
	var squads [NumLeagues]int
	for mi, m := range s.Managers {
		if m.League >= 0 && m.League < NumLeagues {
			squads[m.League] += s.countLineup(mi, 0)
		}
	}
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
	// The original passes the person index as a slot index to f_2277_25de
	// and so reads some slot; the remake takes the person's value.
	move := func(p int) {
		s.Persons[p].Team = s.chooseBuyer(int(s.personValue(p)/10000), false)
	}
	for i := 0; i < n; i++ {
		for tries := 0; tries < 10000; tries++ {
			p := r.Between(1, 150)
			t0 := s.Persons[p].Team
			if foreign > 4 && foreignTeam(t0) {
				foreign--
				move(p)
				continue
			}
			human := t0 != 0xff && t0 < NumTeams && s.ManagerOf(t0) != nil
			if !human && foreign <= 4 {
				move(p)
				break
			}
		}
	}
	// Bytes, like the original: taken counts the human squads, then the
	// foreign persons.
	taken := 0
	for l := range squads {
		squads[l] &= 0xff
		taken += squads[l]
	}
	for p := 1; p < NumPersons; p++ {
		if foreignTeam(s.Persons[p].Team) {
			taken++
		}
	}
	taken &= 0xff
	rest := (150 - taken) & 0xff
	taken = max(taken, 1)
	var share [NumLeagues]int
	for l := range share {
		share[l] = squads[l] * 100 / taken * rest / 100
	}
	return share
}
