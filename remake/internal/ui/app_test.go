package ui

import (
	"testing"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
)

// recorder is a scene that keeps the input of every update.
type recorder struct{ got []Input }

func (r *recorder) Update(app *App) error    { r.got = append(r.got, app.Input); return nil }
func (r *recorder) Draw(*App, *ebiten.Image) {}

// A paced app runs the scenes at 60 ticks a second however often Ebitengine
// calls Update; a click between two ticks reaches the next one at the
// place it was made, even when the mouse has moved on since.
func TestPacedClickKeepsItsPlace(t *testing.T) {
	r := &recorder{}
	a := NewApp(nil, r)
	a.Paced = true
	t0 := time.Unix(0, 0)
	at := func(ms int) time.Time { return t0.Add(time.Duration(ms) * time.Millisecond) }
	update := func(in Input, ms int) {
		t.Helper()
		if err := a.update(in, at(ms)); err != nil {
			t.Fatal(err)
		}
	}
	update(Input{X: 10, Y: 10}, 0)
	// The click on the player and the mouse moving away come between two
	// ticks, as input events.
	update(Input{X: 248, Y: 203, Left: true, LeftHeld: true}, 2)
	update(Input{X: 200, Y: 160}, 4)
	update(Input{X: 148, Y: 103, Chars: []rune("a")}, 6)
	// The next tick.
	update(Input{X: 140, Y: 100}, 17)
	if len(r.got) != 2 {
		t.Fatalf("%d ticks, want 2", len(r.got))
	}
	in := r.got[1]
	if !in.Left || in.X != 248 || in.Y != 203 || string(in.Chars) != "a" {
		t.Fatalf("tick input %+v, want the click at 248,203 and the typed a", in)
	}
	// The click is not repeated.
	update(Input{X: 140, Y: 100}, 34)
	if in := r.got[2]; in.Left || in.X != 140 || len(in.Chars) != 0 {
		t.Fatalf("next tick %+v", in)
	}
	// Over a second of frames from a 60 Hz ticker and events in between,
	// the scenes get 60 ticks.
	n := len(r.got)
	for ms := 0; ms < 1000; ms++ {
		if ms%17 == 0 || ms%5 == 0 {
			update(Input{}, 1000+ms)
		}
	}
	if d := len(r.got) - n; d < 59 || d > 61 {
		t.Fatalf("%d ticks in a second, want 60", d)
	}
}
