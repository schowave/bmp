package ui

import "testing"

// TestMidX checks the centring of f_06c7_087c for both parities of the
// span and the width, and the clamp at the edges.
func TestMidX(t *testing.T) {
	cases := []struct{ x0, x1, w, want int }{
		{5, 313, 100, 109},  // even span, even width (cup round title)
		{5, 313, 101, 109},  // even span, odd width
		{0, 319, 100, 109},  // odd span, even width
		{0, 319, 101, 109},  // odd span, odd width
		{0, 0x140, 60, 130}, // the draw title
		{0, 319, 330, 0},    // wider than the screen: starts at the left
		{300, 319, 60, 0},   // would leave the screen on the right
	}
	for _, c := range cases {
		if got := midX(c.x0, c.x1, c.w); got != c.want {
			t.Errorf("midX(%d, %d, %d) = %d, want %d", c.x0, c.x1, c.w, got, c.want)
		}
	}
}
