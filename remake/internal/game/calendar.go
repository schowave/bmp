package game

// Leagues: 0 = Bundesliga, 1 = 2. Liga, 2 = Amateur-Oberliga. A league is a
// fixed range of club ids; promotion and relegation swap clubs between ids.
const (
	NumLeagues    = 3
	NumGerman     = 64  // club ids 0..63 are German clubs
	NumTeams      = 200 // ids 64..199 are foreign clubs for the European cups
	MaxManagers   = 4
	NumEvents     = 93 // half-week dates of a season
	LastSeasonDay = 322
)

var (
	// TeamsPerLeague is D:0x225e.
	TeamsPerLeague = [NumLeagues]int{18, 20, 20}
	// MatchdaysPerLeague is D:0x2262.
	MatchdaysPerLeague = [NumLeagues]int{34, 38, 38}
	// Relegated is D:0x2266: regular relegation places per league.
	Relegated = [NumLeagues]int{3, 4, 4}
	// RelegationPlayoff is D:0x226a: the 16th of the Bundesliga plays the
	// 3rd of the 2. Liga.
	RelegationPlayoff = [NumLeagues]bool{true, false, false}
	// FirstID is D:0x226e: first club id of each league.
	FirstID = [NumLeagues]int{0, 18, 38}
)

// UEFAPlaces is D:0x226d.
const UEFAPlaces = 4

// PointsPerWin is D:0x2271: 2 in the original (2-point rule). A game of a
// real season gives 3 from 1995/96 on, when the German leagues took up the
// 3-point rule.
func (s *State) PointsPerWin() int {
	if s.Season != "" && s.StartYear >= 1995 {
		return 3
	}
	return 2
}

// firstReserveClub is the first of the ids 58..63, reserve clubs outside
// the leagues (LeagueOf counts them to league 2).
const firstReserveClub = 58

// LeagueOf returns the league of a club id (f_17c2_1bb5 >> 1). Ids 58..63
// are reserve clubs outside the leagues and count as league 2 there.
func LeagueOf(id int) int {
	switch {
	case id < 18:
		return 0
	case id < 38:
		return 1
	default:
		return 2
	}
}

// LeagueMask returns 1, 2 or 4 like f_17c2_1bb5.
func LeagueMask(id int) int { return 1 << LeagueOf(id) }

// Event bits of the season calendar D:0x2280.
const (
	EvLeague0    = 0x01
	EvLeague1    = 0x02
	EvLeague2    = 0x04
	EvDFB        = 0x08
	EvRelegation = 0x10 // 0x10 alone
	EvEurope     = 0x70
	EvReplay     = 0x80 // Nachholspiele, set at run time
)

// seasonEvents is the event table D:0x2280: one byte per half-week event
// (even index Saturday, odd index Wednesday).
var seasonEvents = [NumEvents]byte{
	0x07, 0x70, 0x07, 0x00, 0x07, 0x70, 0x07, 0x06, 0x08, 0x00,
	0x07, 0x00, 0x07, 0x06, 0x07, 0x70, 0x07, 0x07, 0x07, 0x70,
	0x07, 0x00, 0x08, 0x01, 0x07, 0x06, 0x07, 0x00, 0x07, 0x06,
	0x08, 0x70, 0x07, 0x00, 0x07, 0x70, 0x07, 0x00, 0x00, 0x00,
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x07,
	0x07, 0x00, 0x07, 0x70, 0x08, 0x07, 0x07, 0x70, 0x07, 0x06,
	0x07, 0x00, 0x07, 0x00, 0x07, 0x00, 0x07, 0x70, 0x07, 0x00,
	0x07, 0x70, 0x07, 0x00, 0x07, 0x00, 0x08, 0x00, 0x07, 0x00,
	0x07, 0x10, 0x10,
}

// EventDay returns the day of event e: day 0 is Saturday 29 July.
func EventDay(e int) int {
	d := (e >> 1) * 7
	if e&1 != 0 {
		d += 4
	}
	return d
}

var monthLen = [12]int{31, 28, 31, 30, 31, 30, 31, 31, 30, 31, 30, 31}

// MonthNames and WeekdayNames are the texts of the original.
var (
	MonthNames   = [12]string{"Januar", "Februar", "März", "April", "Mai", "Juni", "Juli", "August", "September", "Oktober", "November", "Dezember"}
	WeekdayNames = [7]string{"Montag", "Dienstag", "Mittwoch", "Donnerstag", "Freitag", "Samstag", "Sonntag"}
)

// Date is a calendar date of the game. Month is 0-based.
type Date struct {
	Day, Month, Year int
}

// DateOf converts a season day into a date (f_0310_1190). There are no leap
// years.
func DateOf(seasonStartYear, day int) Date {
	year := seasonStartYear
	doy := day + 210
	for doy > 365 {
		doy -= 365
		year++
	}
	month := 0
	for doy > monthLen[month] {
		doy -= monthLen[month]
		month = (month + 1) % 12
	}
	return Date{Day: doy, Month: month, Year: year}
}

// Weekday returns the weekday index (0 = Montag) of a season day, as shown
// in the calendar (f_0310_0fb6).
func Weekday(day int) int { return (day + 5) % 7 }
