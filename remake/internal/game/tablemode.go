package game

// Table modes of the league table screen (f_2a41_214c, local_70): the
// buttons HEIM, GESAMT and AUSWÄRTS.
const (
	TableHome = iota
	TableTotal
	TableAway
)

// TableRow is the record of a club in a home, total or away table.
type TableRow struct {
	Club                   int
	Games, Wins, Losses    int
	Points, Goals, Against int
	Form                   string
}

// Draws returns the number of draws.
func (r TableRow) Draws() int { return r.Games - r.Wins - r.Losses }

// MinusPoints is shown next to the points under the 2-point rule.
func (r TableRow) MinusPoints() int { return 2*r.Games - r.Points }

// tableRow sums the home part (modes HEIM and GESAMT) and the away part
// (GESAMT and AUSWÄRTS) of a club's record, like every column of
// f_2a41_214c: "mode < 2" adds home, "mode != 0" adds away.
func (s *State) tableRow(id, mode int) TableRow {
	c := &s.Clubs[id]
	r := TableRow{Club: id, Form: c.Form}
	for v := 0; v < 2; v++ {
		if (v == 0 && mode < 2) || (v == 1 && mode != 0) {
			r.Games += c.Games[v]
			r.Wins += c.Wins[v]
			r.Losses += c.Losses[v]
			r.Points += c.Points[v]
			r.Goals += c.Goals[v]
			r.Against += c.Against[v]
		}
	}
	return r
}

// LeagueTable returns the table of league l in mode TableHome, TableTotal
// or TableAway, best first. It sorts like f_2a41_2d33 with param_3 = mode:
// an exchange sort over the current table order by points, fewer games,
// goal difference and goals. The state is not changed (the original sorts
// the table in place and sorts it back by total when the screen closes).
func (s *State) LeagueTable(l, mode int) []TableRow {
	n := TeamsPerLeague[l]
	rows := make([]TableRow, n)
	for i := 0; i < n; i++ {
		rows[i] = s.tableRow(s.Tables[l][i], mode)
	}
	for i := 0; i < n-1; i++ {
		for j := i + 1; j < n; j++ {
			a, b := &rows[i], &rows[j]
			swap := false
			switch {
			case b.Points > a.Points:
				swap = true
			case b.Points == a.Points:
				if b.Games < a.Games {
					swap = true
				} else if a.Games == b.Games {
					da, db := a.Goals-a.Against, b.Goals-b.Against
					if da < db || (da == db && a.Goals < b.Goals) {
						swap = true
					}
				}
			}
			if swap {
				rows[i], rows[j] = rows[j], rows[i]
			}
		}
	}
	return rows
}
