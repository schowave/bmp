package game

import (
	"strings"
	"testing"
)

// The human manager whose club wins the DFB final gets the title screen
// "gewinnt den DfB-Pokal" (f_17c2_16dc); the loser and a final won by another
// club show none.
func TestDFBFinalShowsTrophy(t *testing.T) {
	for _, tc := range []struct {
		name       string
		home, away int // 0 is the club of the manager
		goals      [2]int
		wantTrophy bool
	}{
		{"won at home", 0, 1, [2]int{2, 1}, true},
		{"won away", 1, 0, [2]int{1, 2}, true},
		{"lost", 0, 1, [2]int{0, 1}, false},
		{"other clubs", 1, 2, [2]int{1, 0}, false},
	} {
		s := lineupState(t)
		s.Notices = nil
		own := s.Managers[0].Team
		club := func(k int) int {
			if k == 0 {
				return own
			}
			return (own + k) % NumGerman
		}
		c := &s.Cups
		c.Round[CupDFB] = 5
		c.Slots[CupDFB][0], c.Slots[CupDFB][1] = club(tc.home), club(tc.away)
		c.Result[CupDFB][0], c.Result[CupDFB][1] = tc.goals[0], tc.goals[1]
		s.advanceCup(CupDFB)
		var got *Notice
		for _, n := range s.Notices {
			if n.Kind == NoticeTrophy {
				got = &n
			}
		}
		if (got != nil) != tc.wantTrophy {
			t.Errorf("%s: trophy notice %v, want %v", tc.name, got != nil, tc.wantTrophy)
			continue
		}
		if got != nil && (got.Manager != 0 || got.Cup != 1 || !strings.Contains(strings.Join(got.Lines, " "), "gewinnt den")) {
			t.Errorf("%s: notice %+v", tc.name, *got)
		}
	}
}
