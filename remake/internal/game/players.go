package game

// Person is a player of the PERSON table (F:0x57dd + p*0x25, p = 1..150).
// AI clubs have no rosters: a person belongs to a club only via Team.
type Person struct {
	Name  string
	Age   int    // +0x1a
	Avg   int    // +0x1b, average of the three skills at creation
	Skill [3]int // +0x1c..0x1e
	// Pos (+0x1f) is the position code: Pos/25 = 0 Torwart, 1 Abwehr,
	// 2 Mittelfeld, 3 Angriff.
	Pos  int
	Side int // +0x20: preferred side 0 (far left) .. 6 (far right)
	// Owner (+0x21): 0..3 human manager, 4 transfer market, 5 free or at an
	// AI club.
	Owner       int
	LeagueGoals int // +0x22
	LeagueGames int // +0x23
	Team        int // +0x24, 0xff = none
	Origin      int // team of the MANA.DAT roster the name came from
}

// Owner values of Person.Owner besides manager indexes.
const (
	OwnerMarket = 4
	OwnerFree   = 5
)

// Positions.
const (
	PosKeeper = iota
	PosDefence
	PosMidfield
	PosAttack
)

// PositionNames are the texts of the original.
var PositionNames = [4]string{"TORWART", "ABWEHRSPIELER", "MITTELFELDSPIELER", "ANGREIFER"}

// Position returns the position of a person.
func (p *Person) Position() int { return p.Pos / 25 }

// NumPersons is the size of the person table; index 0 is unused.
const NumPersons = 151

// Squad sizes: 25 slots per human manager (24 used), the transfer market is
// pseudo manager 4 with 12 slots.
const (
	SquadSlots  = 25
	SquadMax    = 24
	MarketSlots = 12
	MarketIndex = 4
)

// Slot is a player in a squad or on the market (F:0x774a + (m*25+s)*0x34).
type Slot struct {
	// Cards (+0x00..0x02): red, yellow and yellow-red cards this season.
	Cards [3]int
	Goals [3]int // +0x03: goals this season (league, DFB, Europe)
	Games [3]int // +0x06: games this season (league, DFB, Europe)
	// Rejected is +0x03 of a market slot: bit m = manager m's bid was
	// rejected, 0x80 = some rejection active.
	Rejected int
	// Status (+0x09): &3 1 = suspended, 2 = injured; 0x40 AI offer for
	// this squad player; 0x80 AI offer for a listed player.
	Status      int
	Lineup      int    // +0x0a: 1..11 team, 0 none, >= 12 reserve
	Years       int    // +0x0b: contract seasons left
	Loan        int    // +0x0c: 0 own player, 0x80|team = loaned in
	Out         int    // +0x0d: games left for suspension or injury
	Tendency    int    // +0x0e: 30..68, class (x-30)/13: falling, constant, rising
	Person      int    // +0x0f, 0 = empty slot
	Skill       [3]int // +0x10..0x12: Kondition, Technik, Form
	Fatigue     int    // +0x13: Erschöpfung 50..150, normal 60..100
	Drift       int    // +0x14: tendency drift: weeks (bits 0-3), step (4-6), up (7)
	Rating      int    // +0x15: match rating accumulator
	Buyer       int    // +0x16: AI club making an offer
	Injury      int    // +0x17: injury kind
	Nego        int    // +0x18: negotiation state
	Col         int    // +0x19: column on the tactics board 0..6 (left..right)
	Row         int    // +0x1a: row on the tactics board 0 (forward line)..7 (goal)
	Career      [3]int // +0x1c, +0x1e, +0x20: career appearances per competition
	CareerGoals [3]int // +0x22..0x26
	Salary      int64  // +0x28: monthly salary, asking price on the market
	Message     bool   // +0x30: a news message belongs to this slot
}

// Empty reports whether the slot holds no player.
func (s *Slot) Empty() bool { return s.Person == 0 }

// Strength is the strength shown for a player: the average of his skills.
func (s *Slot) Strength() int { return (s.Skill[0] + s.Skill[1] + s.Skill[2]) / 3 }

// squad returns the slots of manager m (or the market for m == 4).
func (s *State) squad(m int) []Slot {
	if m == MarketIndex {
		return s.Squads[m][:MarketSlots]
	}
	return s.Squads[m][:SquadSlots]
}

// SquadSize returns the number of players of manager m (f_3091_1109).
func (s *State) SquadSize(m int) int {
	limit := SquadMax
	if m == MarketIndex {
		limit = MarketSlots
	}
	n := 0
	for i := 0; i < limit; i++ {
		if !s.Squads[m][i].Empty() {
			n++
		}
	}
	return n
}

// managerIndex returns the index of a manager.
func (s *State) managerIndex(m *Manager) int {
	for i, x := range s.Managers {
		if x == m {
			return i
		}
	}
	return -1
}
