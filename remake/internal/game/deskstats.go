package game

// Queries for the statistics screen of the desk (f_262f_0af7).

// BestSeries returns the longest series of club t per kind and venue, like
// the per-manager maxima F:0xa276 that the original raises after every
// league game to the running series of the manager's club (f_2a41_2d33).
// The remake derives them from the league games of the current season, so
// they start again with every season.
func (s *State) BestSeries(t int) [NumSeries][3]int {
	var cur, best [NumSeries][3]int
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
		for k := range cur {
			for _, v := range []int{VenueTotal, venue} {
				if hit[k] {
					cur[k][v]++
				} else {
					cur[k][v] = 0
				}
				best[k][v] = max(best[k][v], cur[k][v])
			}
		}
	}
	return best
}
