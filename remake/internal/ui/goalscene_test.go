package ui

import (
	"strings"
	"testing"

	"github.com/schowave/bmp/remake/internal/assets"
	"github.com/schowave/bmp/remake/internal/game"
)

// updatesUntil runs the scene until done and returns the number of updates
// and the update at which each frame was first shown.
func updatesUntil(s *GoalScene) (int, map[int]int) {
	first := map[int]int{s.Frame(): 0}
	n := 0
	for !s.Update() {
		n++
		if _, ok := first[s.Frame()]; !ok {
			first[s.Frame()] = n
		}
		if n > 100000 {
			break
		}
	}
	return n + 1, first
}

func TestGoalSceneTiming(t *testing.T) {
	sc := &assets.Scene{Frames: make([]assets.SceneFrame, 40), Builder: "J]RGEN KRAHE"}
	s := NewGoalScene(nil, sc, false)
	if s.Builder() != "JÜRGEN KRAHE" {
		t.Errorf("builder %q", s.Builder())
	}
	tick := 1 / biosTicksPerFrame // updates per BIOS tick
	n, first := updatesUntil(s)
	// Measured in the original: frame 0 for 12 ticks, then 2 ticks each,
	// the last frame 7 ticks longer.
	if want := (12 + 2*39 + 7) * tick; float64(n) < want-2 || float64(n) > want+2 {
		t.Errorf("play 0: %d updates, want %.0f", n, want)
	}
	if want := 12 * tick; float64(first[1]) < want-1 || float64(first[1]) > want+1 {
		t.Errorf("frame 1 at %d, want %.0f", first[1], want)
	}
	if !s.Replay() || s.Frame() != 21 || s.Plays() != 2 {
		t.Fatalf("replay 1 starts at frame %d", s.Frame())
	}
	n, _ = updatesUntil(s)
	if want := (5*19 + 7) * tick; float64(n) < want-2 || float64(n) > want+2 {
		t.Errorf("replay 1: %d updates, want %.0f", n, want)
	}
	if !s.Replay() || s.Frame() != 0 {
		t.Fatal("replay 2")
	}
	n, _ = updatesUntil(s)
	if want := (15 + 5*39 + 7) * tick; float64(n) < want-2 || float64(n) > want+2 {
		t.Errorf("replay 2: %d updates, want %.0f", n, want)
	}
	if s.Replay() {
		t.Error("a third replay")
	}
}

func TestPickScene(t *testing.T) {
	app, _ := testApp(t)
	r := &game.Rand{Seed: 1}
	pens := 0
	for i := 0; i < 2000; i++ {
		goal := i%2 == 0
		sc, name, pen, err := PickScene(app.Gfx.dir, r, goal, false)
		if err != nil {
			t.Fatal(err)
		}
		if len(sc.Frames) == 0 || strings.Contains(name, ".T") != goal || strings.HasSuffix(name, "E") != pen {
			t.Fatalf("%s goal %v pen %v", name, goal, pen)
		}
		if pen {
			pens++
		}
	}
	// 1/16 of the goals and 1/26 of the misses become penalties: ~101.
	if pens < 60 || pens > 150 {
		t.Errorf("%d penalty scenes of 2000", pens)
	}
	_, name, pen, _ := PickScene(app.Gfx.dir, r, false, true)
	if !pen || !strings.HasSuffix(name, ".VE") {
		t.Errorf("penalty miss: %s", name)
	}
}

// f_3930_01e7(n) waits n/3 ticks of the 18.2 Hz timer: a minute of a match
// day at speed 40 lasts 6 ticks, about a third of a second, as measured on
// the original in the browser.
func TestWaitFrames(t *testing.T) {
	for _, c := range []struct{ n, want int }{{6, 7}, {0x14, 20}, {0x1e, 33}, {0x26 - 40/2, 20}, {0x46, 76}, {0x78 - 40, 86}, {0x96, 165}} {
		if got := waitFrames(c.n); got != c.want {
			t.Errorf("waitFrames(%#x) = %d, want %d", c.n, got, c.want)
		}
	}
}
