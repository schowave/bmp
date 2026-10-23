package game

import "github.com/schowave/bmp/remake/internal/assets"

// Team is a club of the TEAM table (F:0x3066 + id*0x22). German clubs are
// ids 0..63, foreign clubs 64..199.
type Team struct {
	Name string
	// Mood (+0x17) moves with results, 45..55 at start, 47..53 later.
	Mood int
	// Strength (+0x18..0x1a) are the three current strength values, Base
	// (+0x1b..0x1d) the long-term values they drift around. For clubs of
	// human managers they are recomputed from the squad.
	Strength [3]int
	Base     [3]int
	// Extra (+0x1e..0x20) are three more values in 45..55.
	Extra [3]int
	// Logo (+0x21) is the picture number of the club's logo.
	Logo int
	// Ticker is bit 7 of +0x21 in the original: the goals of the club are
	// announced (ANZEIGEN in the club info). The remake keeps picture
	// numbers up to 143 in Logo, so the flag has a field of its own.
	Ticker bool `json:",omitempty"`
}

// Credit is a loan a manager has taken (M+0x243e + slot*0x12).
type Credit struct {
	Amount                          int64
	Interest                        int64 // per month
	StartDay, StartMonth, StartYear int
	DueDay, DueMonth, DueYear       int
	Rate                            int
}

// Stadium item indexes of Manager.Stadium.
const (
	StSeats    = iota // Sitzplätze
	StStanding        // Stehplätze
	StCovered         // überdachte Plätze
	StFloodlight
	StScoreboard
	StCondition // Zustand / Renovierung
	StComfort
	NumStadiumItems
)

// Manager is the record of a human manager (F:0x2242 + m*0x30a).
type Manager struct {
	Name   string // +0x2242
	Number int    // +0x225f
	Face   int    // picture of the manager (part of PIC/0.VGA)
	Team   int    // +0x2260

	// History (+0x2280): per season overall rank, league, DFB round, Europe.
	History        [][4]byte
	DroppedSeasons int // +0x237d

	Titles [5]int // +0x227b: Meisterschaften, DFB, Landesmeister, Pokalsieger, UEFA

	TicketPrice   int     // +0x234c
	PosAfter      [38]int // +0x234d: table position after each matchday
	EffortSetting int     `json:"Einsatz"` // +0x2373: the EINSATZ slider of the lineup screen, 0..34, default 16
	CupStatus     [4]int  // +0x2374: 0x1e = not taking part, else round reached
	League        int     // +0x237a
	CampDays      int     // +0x237b
	HomeGames     int     // +0x237c
	Effort        int     // +0x237f: 0..40 from the lineup; 100 = fewer than 8 players
	RiotPending   bool    // +0x2380 bit 0
	Youth         int     // +0x2381 youth work level 0..34
	SeasonFlags   int     // +0x2382: 1 promoted, 2 relegated, 4 licence, 8 restart
	Training      [4]int  // +0x2383: points for Kondition, Spiel, Schuss, Taktik (sum <= 20)
	Intensity     int     // +0x2387: training intensity
	PosTraining   [4]int  // +0x2388: points for Tor, Abwehr, Mittelfeld, Angriff (sum <= 10)

	Attendance [20]int // +0x238c: attendance/1000 of each home league game

	Stadium      [NumStadiumItems]int64 // +0x23a0, current values
	StadiumBuild [NumStadiumItems]int64 // +0x23a4, values after construction
	BuildTimer   [NumStadiumItems]int   // +0x23d8, days until completion

	// Eternal balance (+0x23e6..): [0] home, [1] away.
	EtPoints, EtMinus, EtGoals, EtAgainst, EtGames, EtWins, EtLosses, EtDraws [2]int

	Popularity     int   // +0x241e, 0..100
	CostItem       int   // +0x2420 (uint16)
	BaseIncome     int   // +0x2422
	YouthPoints    int   // +0x2424
	SeasonAudience int64 // +0x2426
	RecordAudience int64 // +0x242a
	MinAudience    int64 // +0x242e
	Money          int64 // +0x2432
	RecordOpponent int   // +0x2436
	MinOpponent    int   // +0x243a

	Credits [5][3]Credit // +0x243e: lender k (0..3 managers, 4 bank), slot j

	// Advertising (D:0x066c + m*0x24): [0] Trikot, [1..6] boards, [7] TV,
	// [8] Werbeausgaben.
	Ads          [9]int64
	TrikotMonths int    // F:0x0000 + m*2
	TrikotIndex  int    // F:0x0001 + m*2
	BoardMonths  [6]int // F:0x4b9a + (m*6+b)*2
	BoardIndex   [6]int
	Offers       [2][10]int64 // F:0x002e: offers per type (Trikot, Bande)
	OfferYears   [2][10]int   // F:0x2ec2

	BalanceSum int64 // D:0x0644: sum of daily balances this month

	GoalFlags int // F:0x4bec: "letzte Chance" messages already shown

	// AutoLineup is D:0x79e[2m]: 1 manual, 2..4 automatic with formation
	// 4-4-2, 3-5-2, 3-4-3.
	AutoLineup int
	Subs       int // F:0x90c6: substitutions made in the current match

	Inbox       []Message // newest first
	NewMessages bool      // F:0x56e2: open the message panel at the turn
}

// ForfeitEffort marks a team with fewer than 8 players in the lineup.
const ForfeitEffort = 100

// State is the complete state of a game.
type State struct {
	Rand Rand

	Teams [NumTeams]Team
	Clubs [NumGerman]ClubStats
	// Tables holds the club ids of each league in table order (F:0x535a).
	Tables [NumLeagues][20]int
	// Results of every league matchday (F:0x6ddc): [league][md-1][pair].
	Results [NumLeagues][38][10]Score
	// Matchday is the next matchday of each league (D:0x225a, 1-based).
	Matchday [NumLeagues]int

	Managers []*Manager
	Persons  [NumPersons]Person
	// Squads holds the slots of the human managers 0..3 and of the
	// transfer market (index 4).
	Squads [MaxManagers + 1][SquadSlots]Slot

	Day        int // D:0x7dc, day of the season, 0 = 29 July
	Event      int // F:0x16e
	Events     [NumEvents]byte
	StartYear  int // D:0x7e0, year the current season started
	Seasons    int // D:0x7e2, completed seasons
	EndYear    int // F:0x513c
	Modern     bool
	Level      int // D:0x4a28, internal level (5 - displayed level)
	Postponed  [20]PostponedGame
	BankBase   int    // D:0x0630
	BankRates  [6]int // D:0x060e
	CampTimers [8]int // D:0x0620
	// BuildLockout is F:0x5780: days without a construction company per
	// stadium item.
	BuildLockout [NumStadiumItems]int

	Cups Cups

	CupDay bool // F:0x513e: league suspensions do not apply today

	// SubsidyOffered is set when a sponsor subsidy of PendingSubsidy DM,
	// possibly 0, is offered after a Purchase.
	SubsidyOffered bool
	PendingSubsidy int64
	Purchase       PendingPurchase

	Loop           Loop
	RelegationLeg2 bool

	Display Display
	// Draws lists the cups whose new draw the screens still have to
	// present (f_17c2_0006), in the order of the original.
	Draws []Cup
	// Highscores are the entries of the managers at the last season end
	// that the screens still have to put into the high-score table.
	Highscores []HighscoreEntry
	mana       *assets.Mana `json:"-"`
}

// Display holds the toggles of the options screen (D:0x05fe..0x060c) and
// the speed of the match days (D:0x063a). They only change what is shown.
type Display struct {
	HalfTime   [NumLeagues]bool // Halbzeitstände
	Results    [NumLeagues]bool // Ergebnisse
	Table      [NumLeagues]bool // Tabelle
	GoalScenes bool             // Torszenen
	DFB        bool             // DfB-Pokal
	Europe     bool             // Europapokale
	Replays    bool             // Nachholspiele
	Newspaper  bool             // Zeitung
	Fades      bool             // Blenden
	Speed      int              // Spielgeschwindigkeit, 0..70
}

// DefaultDisplay is the setting of a new game: everything on but the fades.
func DefaultDisplay() Display {
	on := [NumLeagues]bool{true, true, true}
	return Display{HalfTime: on, Results: on, Table: on, GoalScenes: true, DFB: true, Europe: true,
		Replays: true, Newspaper: true, Speed: 40}
}

// TakeDraw returns the next cup draw to present, or false.
func (s *State) TakeDraw() (Cup, bool) {
	if len(s.Draws) == 0 {
		return 0, false
	}
	c := s.Draws[0]
	s.Draws = s.Draws[1:]
	return c, true
}

// PostponedGame is a league game moved to a later date (F:0x5714).
type PostponedGame struct {
	Slot     int // event index of the new date, 0 = free record
	Matchday int
	League   int
	Pair     int
}

// Date returns today's date.
func (s *State) Date() Date { return DateOf(s.StartYear, s.Day) }

// ManagerOf returns the manager whose club is id, or nil.
func (s *State) ManagerOf(id int) *Manager {
	for _, m := range s.Managers {
		if m.Team == id {
			return m
		}
	}
	return nil
}
