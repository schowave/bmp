package game

// Queries for the club information popup (f_2a41_000e) and its remaining
// fixtures (f_0000_28c4).
//
// The original keeps running series (F:0x9d36), records (F:0xa2ca with the
// opponents in F:0xa4ca) and the last five head-to-head results per human
// manager (F:0x9336) as extra state, updated after every league game
// (f_2a41_2d33, f_2a41_37cf, f_2a41_2073). The remake does not track them
// yet; these functions derive them from the league results of the current
// season instead, which gives the same values within a season.

// Series kinds of the popup (D:0x4adc).
const (
	SeriesWon = iota
	SeriesLost
	SeriesDrawn
	SeriesNotWon
	SeriesNotLost
	SeriesCleanSheet // OHNE GEGENTOR
	SeriesNoGoal     // OHNE TORERFOLG
	NumSeries
)

// Venues of a series: total, home, away (index of F:0x9d36).
const (
	VenueTotal = iota
	VenueHome
	VenueAway
)

// Record kinds of the popup: 0..3 home, 4..7 away (F:0xa2ca + t*8).
const (
	RecordWin      = iota // HÖCHSTER SIEG
	RecordDefeat          // HÖCHSTE NIEDERL.
	RecordScored          // ERZIELTE TORE
	RecordConceded        // KASSIERTE TORE
	NumRecords     = 8
)

// Record is one record of a club like the byte of F:0xa2ca: two goal
// counts in the high and the low nibble and the opponent (F:0xa4ca).
// Raw == 0 means "NOCH KEINE EINTRAGUNG.".
//
// f_2a41_2d33 stores the score of the game with the home team's goals in
// the high nibble for wins and defeats (so "3:0" for a home win, "0:3" for
// an away win), the scored goals in the high nibble at home and in the low
// nibble away, the conceded goals in the low nibble at home and in the high
// nibble away.
type Record struct {
	Raw int
	Opp int
}

// Hi and Lo are the two nibbles of a record.
func (r Record) Hi() int { return r.Raw >> 4 }
func (r Record) Lo() int { return r.Raw & 15 }

// update is f_2a41_37cf(.., goals g1, g2, mode, a, b, opp): the record is
// replaced when a*g1 - b*g2 is larger (mode 0) or smaller (mode 1) than
// the stored value.
func (r *Record) update(g1, g2, mode, a, b, opp int) {
	hi, lo := r.Hi(), r.Lo()
	stored, game := hi*a-lo*b, a*g1-b*g2
	if (mode == 0 && stored < game) || (mode != 0 && game < stored) {
		if b < 0 {
			b = 1
		}
		v := (a*g1*16 + b*g2) & 0xff
		r.Raw, r.Opp = v, opp
	}
}

// PlayedGame is a league game of a club from its own view.
type PlayedGame struct {
	Opp          int
	Home         bool
	For, Against int
}

// InfoStrength is the strength shown in the popup: the mean of the three
// current, base and extra values (f_2a41_000e, f_2a41_1d18).
func (s *State) InfoStrength(t int) int {
	tm := &s.Teams[t]
	sum := 0
	for i := 0; i < 3; i++ {
		sum += tm.Strength[i] + tm.Base[i] + tm.Extra[i]
	}
	return sum / 9
}

// LeagueGames returns the league games club t has played this season in
// matchday order.
func (s *State) LeagueGames(t int) []PlayedGame {
	if t < 0 || t >= NumGerman {
		return nil
	}
	l := LeagueOf(t)
	var out []PlayedGame
	for md := 1; md < s.Matchday[l] && md <= MatchdaysPerLeague[l]; md++ {
		for i, p := range Fixtures(l, md) {
			if p.Home != t && p.Away != t {
				continue
			}
			r := s.Results[l][md-1][i]
			if r.Home == Postponed {
				continue
			}
			if p.Home == t {
				out = append(out, PlayedGame{Opp: p.Away, Home: true, For: r.Home, Against: r.Away})
			} else {
				out = append(out, PlayedGame{Opp: p.Home, Home: false, For: r.Away, Against: r.Home})
			}
		}
	}
	return out
}

// TeamSeries returns the running series of club t per kind and venue, like
// F:0x9d36: a series counts the games in a row up to now and is reset by
// a game that breaks it.
func (s *State) TeamSeries(t int) [NumSeries][3]int {
	var ser [NumSeries][3]int
	for _, g := range s.LeagueGames(t) {
		venue := VenueAway
		if g.Home {
			venue = VenueHome
		}
		hit := [NumSeries]bool{
			SeriesWon:        g.For > g.Against,
			SeriesLost:       g.For < g.Against,
			SeriesDrawn:      g.For == g.Against,
			SeriesNotWon:     g.For <= g.Against,
			SeriesNotLost:    g.For >= g.Against,
			SeriesCleanSheet: g.Against == 0,
			SeriesNoGoal:     g.For == 0,
		}
		for k := range ser {
			for _, v := range []int{VenueTotal, venue} {
				if hit[k] {
					ser[k][v]++
				} else {
					ser[k][v] = 0
				}
			}
		}
	}
	return ser
}

// TeamRecords returns the records of club t: 0..3 at home, 4..7 away,
// updated after each game like f_2a41_2d33 does.
func (s *State) TeamRecords(t int) [NumRecords]Record {
	var rec [NumRecords]Record
	for _, g := range s.LeagueGames(t) {
		if g.Home {
			hg, ag := g.For, g.Against
			rec[RecordScored].update(hg, ag, 0, 1, 0, g.Opp)
			rec[RecordConceded].update(hg, ag, 0, 0, -1, g.Opp)
			switch {
			case hg > ag:
				rec[RecordWin].update(hg, ag, 0, 1, 1, g.Opp)
			case hg < ag:
				rec[RecordDefeat].update(hg, ag, 1, 1, 1, g.Opp)
			}
		} else {
			hg, ag := g.Against, g.For
			rec[4+RecordConceded].update(hg, ag, 0, 1, 0, g.Opp)
			rec[4+RecordScored].update(hg, ag, 0, 0, -1, g.Opp)
			switch {
			case hg > ag:
				rec[4+RecordDefeat].update(hg, ag, 0, 1, 1, g.Opp)
			case hg < ag:
				rec[4+RecordWin].update(hg, ag, 1, 1, 1, g.Opp)
			}
		}
	}
	return rec
}

// HeadToHead returns the league results of club t against club own from
// t's view, at t's home and away, oldest first, at most five each like
// F:0x9336.
func (s *State) HeadToHead(t, own int) (home, away []PlayedGame) {
	for _, g := range s.LeagueGames(t) {
		if g.Opp != own {
			continue
		}
		if g.Home {
			home = append(home, g)
		} else {
			away = append(away, g)
		}
	}
	if len(home) > 5 {
		home = home[len(home)-5:]
	}
	if len(away) > 5 {
		away = away[len(away)-5:]
	}
	return home, away
}

// Ticker reports the "ANZEIGEN" flag of club t (TEAM+0x21 bit 7): the
// goals of the club are announced.
func (s *State) Ticker(t int) bool { return s.Teams[t].Ticker }

// ToggleTicker switches the "ANZEIGEN" flag of club t.
func (s *State) ToggleTicker(t int) { s.Teams[t].Ticker = !s.Teams[t].Ticker }

// MatchdayEvent returns the event index of matchday md of league l
// (f_0000_2895): the md-th event in which the league plays.
func (s *State) MatchdayEvent(l, md int) int {
	n := 0
	for e := 0; e < NumEvents; e++ {
		if seasonEvents[e]&(1<<l) != 0 {
			n++
			if n == md {
				return e
			}
		}
	}
	return NumEvents - 1
}

// GameEvent returns the event index of the game of pair p on matchday md
// of league l: the date of the matchday, or the new date of a postponed
// game that has not been played yet (F:0x5714).
func (s *State) GameEvent(l, md, p int) int {
	for _, q := range s.Postponed {
		if q.Slot != 0 && q.Matchday == md && q.League == l && q.Pair == p {
			return q.Slot
		}
	}
	return s.MatchdayEvent(l, md)
}
