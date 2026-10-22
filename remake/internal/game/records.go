package game

// Records holds per-club series and records shown on the statistics
// screen (F:0x9d36, F:0xa2ca, F:0xa4ca) and the head-to-head table F:0x9336.
// They are display-only and travel with a club on identity swaps.
type Records struct{}

func (r *Records) swap(a, b int) {}
