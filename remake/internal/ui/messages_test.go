package ui

import (
	"slices"
	"testing"
)

// A wrapped line must not overwrite the lines after it.
func TestMsgLinesWrap(t *testing.T) {
	app, _ := testApp(t)
	msg := make([]string, 0, 8) // spare capacity, like the slices of turn.go
	msg = append(msg, "11. April 1993", "In der neuen Komfortbewertung", "wird ihr Stadion um eine Note", "schlechter beurteilt. ")
	got := msgLines(app.Gfx, msg)
	want := []string{"11. April 1993", "In der neuen", "Komfortbewertung", "wird ihr Stadion um eine Note", "schlechter beurteilt. "}
	if !slices.Equal(got, want) {
		t.Fatalf("lines %q, want %q", got, want)
	}
}
