package ui

import (
	"fmt"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/schowave/bmp/remake/internal/assets"
	"github.com/schowave/bmp/remake/internal/game"
)

// Goal scene geometry: the stadium view PIC/8.VGA (188x102) is the frame,
// the scene itself is a 182x96 window of the panorama PIC/26.VGA at +3,+3
// (f_1a58_0ca3 blits 8.VGA at 0x42,y+2, f_0c7c_0004 the scene at 0x45,y+5).
const (
	GoalSceneW = 0xbc
	GoalSceneH = 0x66
	sceneInset = 3
	sceneViewW = 0xb6
	sceneViewH = 0x60
	sceneTop   = 0x23 // object y 0 is panorama row 35
)

// biosTicksPerFrame converts the BIOS timer of the original (18.2 Hz, the
// unit of f_35ac_307a) to Ebitengine updates at 60 per second.
const biosTicksPerFrame = 1193182.0 / 65536 / 60

// GoalScene plays a scene like f_0c7c_0004. Play 0 shows all frames at 2
// ticks each (the first frame 10 ticks longer); the replays (Replay) are
// slow motion at 5 ticks: the first one only the last 19 frames, the second
// one all. After the last frame the picture stays for 7 ticks.
type GoalScene struct {
	g     *Gfx
	sc    *assets.Scene
	play  int
	frame int
	t     float64 // ticks the current frame has been shown
	hold  bool    // the last frame is held
	done  bool
	buf   *ebiten.Image
	drawn int // frame in buf, -1 for none
}

// NewGoalScene prepares a scene; mirrored for a chance of the away team
// (f_14a4_0053 with param_9 of f_1a58_0ca3).
func NewGoalScene(g *Gfx, data *assets.Scene, mirrored bool) *GoalScene {
	s := &GoalScene{g: g, sc: data.Prepared(mirrored), drawn: -1}
	s.start()
	return s
}

// Builder is the name of the scene's author, for "Der Erbauer der letzten
// gespielten Torszene war:" (right click on the match screen).
func (s *GoalScene) Builder() string { return fontToUTF8(s.sc.Builder) }

// Plays is how often the scene has been started (1..3, D:0x549a + 1).
func (s *GoalScene) Plays() int { return s.play + 1 }

// Frame is the index of the frame shown.
func (s *GoalScene) Frame() int { return s.frame }

// Replay starts the next slow-motion replay. The original allows two
// (while 'r' or 'w' is pressed after a play); it returns false when both
// have been shown.
func (s *GoalScene) Replay() bool {
	if s.play >= 2 {
		return false
	}
	s.play++
	s.start()
	return true
}

func (s *GoalScene) start() {
	s.frame, s.t, s.hold, s.done = 0, 0, false, false
	if s.play == 1 {
		s.frame = len(s.sc.Frames) - 1 - 0x12
		if s.frame < 0 {
			s.frame = 0
		}
	}
}

// ticks is how long frame f is shown in the current play.
func (s *GoalScene) ticks(f int) float64 {
	n := 2.0
	if s.play > 0 {
		n = 5
	}
	if f == 0 {
		n += 10
	}
	return n
}

// Update advances the scene by one 1/60 s; it reports when the play is over.
func (s *GoalScene) Update() (done bool) {
	if s.done {
		return true
	}
	s.t += biosTicksPerFrame
	if s.hold {
		if s.t >= 7 {
			s.done = true
		}
		return s.done
	}
	if d := s.ticks(s.frame); s.t >= d {
		s.t -= d
		if s.frame+1 < len(s.sc.Frames) {
			s.frame++
		} else {
			s.hold = true
		}
	}
	return false
}

// Draw draws the stadium frame and the current picture into the 188x102
// region with its top-left corner at x, y.
func (s *GoalScene) Draw(dst *ebiten.Image, x, y int) {
	g := s.g
	Draw(dst, g.Sub(8, rectWH(0, 0, GoalSceneW, GoalSceneH)), x, y)
	if s.buf == nil {
		s.buf = ebiten.NewImage(320, 112)
	}
	if s.drawn != s.frame {
		s.render(s.frame)
	}
	fr := &s.sc.Frames[s.frame]
	Draw(dst, s.buf.SubImage(rectWH(fr.ScrollX, fr.ScrollY, sceneViewW, sceneViewH)).(*ebiten.Image),
		x+sceneInset, y+sceneInset)
}

// render composes frame f on the panorama: the sprites of PIC/27.VGA and
// the fronts of the goals of PIC/29.VGA in the order of the objects.
func (s *GoalScene) render(f int) {
	g := s.g
	s.buf.Clear()
	Draw(s.buf, g.Pic(26, false), 0, 0)
	for _, o := range s.sc.Frames[f].Objects {
		switch {
		case o.ID < assets.SceneGoalLeft:
			sx, w := (o.ID%24)*12, 12
			if o.ID >= assets.SceneSmall {
				sx, w = sx+4, 4
			}
			sy := (o.ID / 24) * 11
			Draw(s.buf, g.SubKeyed(27, 0, rectWH(sx, sy, w, 11)), o.X, o.Y+sceneTop)
		case o.ID == assets.SceneGoalLeft:
			Draw(s.buf, g.SubKeyed(29, 0, rectWH(0, 0, 24, 20)), 0, 0x36)
		default:
			// The original blits 23 columns from x 24 of the 46 pixels
			// wide picture: its last column is column 0 of the next row.
			Draw(s.buf, g.SubKeyed(29, 0, rectWH(24, 0, 22, 20)), 0x128, 0x36)
			Draw(s.buf, g.SubKeyed(29, 0, rectWH(0, 1, 1, 19)), 0x128+22, 0x36)
		}
	}
	s.drawn = f
}

// PickScene chooses and loads a scene like f_14a4_05ec: a random number
// 2..43, then a 1 in 16 (goal) or 1 in 26 (miss) chance that the chance is
// shown as a penalty, then for a penalty a number 2..5 of the E scenes,
// else a 1 in 401 chance of the joke scene 2.TJ / 2.VJ. It returns whether
// the scene is a penalty scene (the original returns that flag to
// f_1a58_0ca3, which passes it to the replays and uses it as penalty).
// The random numbers come from r, the generator of the game, like in the
// original.
func PickScene(dir *assets.Dir, r *game.Rand, goal, penalty bool) (sc *assets.Scene, name string, pen bool, err error) {
	counts, err := dir.SceneCounts()
	if err != nil {
		counts = assets.DefaultSceneCounts
	}
	odds := 25
	if goal {
		odds = 15
	}
	for try := 0; try < 50; try++ {
		pen = penalty
		n := r.Between(2, counts.Normal)
		if r.Between(0, odds) == 0 {
			pen = true
		}
		joke := false
		if pen {
			n = r.Between(2, counts.Penalty)
		} else if r.Between(0, 400) == 0 {
			joke = true
			n = r.Between(2, counts.Joke)
		}
		name = assets.SceneName(n, goal, pen, joke)
		sc, err = dir.Scene(name)
		if err == nil {
			return sc, name, pen, nil
		}
	}
	return nil, "", penalty, fmt.Errorf("no goal scene: %w", err)
}

// fontToUTF8 turns a text in the font encoding of the game ('[' Ä, '|' ö,
// ']' Ü, '{' ä, '}' ü, '~' ß, or CP437) into a Go string.
func fontToUTF8(s string) string {
	m := map[byte]rune{'[': 'Ä', '|': 'ö', ']': 'Ü', '{': 'ä', '}': 'ü', '~': 'ß',
		0x8e: 'Ä', 0x99: 'Ö', 0x9a: 'Ü', 0x84: 'ä', 0x94: 'ö', 0x81: 'ü', 0xe1: 'ß'}
	out := make([]rune, 0, len(s))
	for i := 0; i < len(s); i++ {
		if r, ok := m[s[i]]; ok {
			out = append(out, r)
		} else {
			out = append(out, rune(s[i]))
		}
	}
	return string(out)
}
