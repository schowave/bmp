package game

// OnPitch returns the number of players of manager m on the pitch
// (f_3091_1109(m, 1)).
func (s *State) OnPitch(m int) int { return s.countLineup(m, 1) }

// PeriodEnded reports the last minute of the period the last Step finished,
// or 0 when the Step did not end a period.
func (md *MatchDay) PeriodEnded() int { return md.ended }

// PeriodStart returns the first minute of the period being played.
func (md *MatchDay) PeriodStart() int {
	if md.period < len(md.periods) {
		return md.periods[md.period][0]
	}
	return 0
}

// Cups returns the cups played on this match day as a mask (bit c).
func (md *MatchDay) Cups() int { return md.cupMask }

// Leagues returns the leagues played on this match day as a mask.
func (md *MatchDay) Leagues() int { return md.leagueMask }

// Replays reports a day of replays (Nachholspiele).
func (md *MatchDay) Replays() bool { return md.replay }

// Over reports whether every period is played, extra time included: the
// next Step finishes the match day.
func (md *MatchDay) Over() bool { return md.period >= len(md.periods) }

// Silent reports extra time without a match of a human team: the match
// screen plays it without showing it (F:0x2e99 = 0).
func (md *MatchDay) Silent() bool { return md.quietExtra && md.period >= 2 }

// Relegation reports the relegation tie between the 1. and the 2. Liga.
func (md *MatchDay) Relegation() bool { return md.relegation }
