package assets

import (
	"strings"
	"testing"
)

func sceneBytes(frames []SceneFrame, name string) []byte {
	b := append([]byte(SceneMagic), 0)
	b = append(b, 5, 0xff, 0xff, 0xff, byte(len(frames)-1))
	for _, f := range frames {
		b = append(b, byte(f.ScrollX), byte(f.ScrollY))
		for _, o := range f.Objects {
			b = append(b, byte(o.X), byte(o.X>>8), byte(o.Y), 0, byte(o.ID), byte(o.ID>>8))
		}
	}
	b = append(b, byte(len(name)+1))
	b = append(b, name...)
	return append(b, 0)
}

func TestParseScene(t *testing.T) {
	var f SceneFrame
	f.ScrollX, f.ScrollY = 70, 11
	f.Objects[0] = SceneObject{X: 300, Y: 6, ID: 111}
	f.Objects[26] = SceneObject{X: 0, Y: 0, ID: SceneGoalLeft}
	s, err := ParseScene(sceneBytes([]SceneFrame{f, f}, "J]RGEN KRAHE"))
	if err != nil {
		t.Fatal(err)
	}
	if len(s.Frames) != 2 || s.Sounds != [4]int{5, -1, -1, -1} || s.Builder != "J]RGEN KRAHE" {
		t.Fatalf("got %d frames, sounds %v, builder %q", len(s.Frames), s.Sounds, s.Builder)
	}
	if s.Frames[1] != f {
		t.Fatalf("frame: got %+v", s.Frames[1])
	}
	if _, err := ParseScene([]byte("nonsense, not a scene")); err == nil {
		t.Fatal("bad header accepted")
	}
}

func TestScenePrepared(t *testing.T) {
	var f SceneFrame
	f.ScrollX = 100
	f.Objects[0] = SceneObject{X: 10, ID: 0}     // runner, 0 -> 117
	f.Objects[1] = SceneObject{X: 20, ID: 30}    // cycle, 30 -> 89
	f.Objects[2] = SceneObject{X: 20, ID: 90}    // cycle, 90 -> 31
	f.Objects[3] = SceneObject{X: 20, ID: 120}   // 120 -> 137
	f.Objects[4] = SceneObject{X: 50, ID: 150}   // ball: not mirrored, x = 314-50
	f.Objects[5] = SceneObject{X: 310, ID: 5}    // off the left after mirroring -> 0
	f.Objects[6] = SceneObject{X: 0xfff0, ID: 5} // negative -> 0 (unmirrored)
	s := &Scene{Frames: []SceneFrame{f}}
	m := s.Prepared(true).Frames[0]
	want := []SceneObject{{296, 0, 117}, {286, 0, 89}, {286, 0, 31}, {286, 0, 137}, {264, 0, 150}, {0, 0, 112}, {0x134, 0, 112}}
	for i, w := range want {
		if m.Objects[i] != w {
			t.Errorf("object %d: got %+v, want %+v", i, m.Objects[i], w)
		}
	}
	if m.ScrollX != 37 {
		t.Errorf("scroll: got %d, want 37", m.ScrollX)
	}
	p := s.Prepared(false).Frames[0]
	if p.Objects[0] != f.Objects[0] || p.Objects[6].X != 0 || p.ScrollX != 100 {
		t.Errorf("unmirrored: got %+v", p)
	}
	if s.Frames[0] != f {
		t.Error("Prepared changed the original")
	}
}

func TestMirrorSpriteIsAnInvolution(t *testing.T) {
	for id := 0; id < SceneSmall; id++ {
		m := mirrorSprite(id)
		if id <= 117 && mirrorSprite(m) != id {
			t.Errorf("%d -> %d -> %d", id, m, mirrorSprite(m))
		}
	}
}

func TestParseSceneCounts(t *testing.T) {
	c, err := ParseSceneCounts([]byte("MAXSZENE:43|5|2\x00"))
	if err != nil || c != DefaultSceneCounts {
		t.Fatalf("got %+v, %v", c, err)
	}
}

func TestSceneName(t *testing.T) {
	for _, c := range []struct {
		n                   int
		goal, penalty, joke bool
		want                string
	}{{10, true, false, false, "10.T"}, {3, false, true, false, "3.VE"}, {2, true, false, true, "2.TJ"}} {
		if got := SceneName(c.n, c.goal, c.penalty, c.joke); got != c.want {
			t.Errorf("got %s, want %s", got, c.want)
		}
	}
}

func TestAllScenesParse(t *testing.T) {
	d := gameDir(t)
	c, err := d.SceneCounts()
	if err != nil {
		t.Fatal(err)
	}
	if c != DefaultSceneCounts {
		t.Errorf("counts %+v", c)
	}
	n := 0
	for upper := range d.names {
		if !strings.HasPrefix(upper, "TORE/") || strings.HasSuffix(upper, "ANZAHL") {
			continue
		}
		s, err := d.Scene(strings.TrimPrefix(upper, "TORE/"))
		if err != nil {
			t.Fatal(err)
		}
		if len(s.Frames) < 10 || s.Builder == "" {
			t.Errorf("%s: %d frames, builder %q", upper, len(s.Frames), s.Builder)
		}
		n++
	}
	if n != 94 {
		t.Errorf("%d scenes", n)
	}
	s, err := d.Scene("10.T")
	if err != nil {
		t.Fatal(err)
	}
	f := s.Frames[0]
	if len(s.Frames) != 54 || s.Builder != "J]RGEN KRAHE" || f.ScrollX != 70 || f.ScrollY != 11 ||
		f.Objects[0] != (SceneObject{177, 6, 0}) || f.Objects[2] != (SceneObject{81, 7, 111}) {
		t.Fatalf("10.T: %d frames, %q, %+v", len(s.Frames), s.Builder, f)
	}
}
