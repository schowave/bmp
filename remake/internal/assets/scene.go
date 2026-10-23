package assets

import (
	"bytes"
	"errors"
	"fmt"
	"strconv"
	"strings"
)

// Goal scenes ("Torszenen") of the match day: TORE\N.T (goal) and TORE\N.V
// (missed chance, "vergeben"), with the suffix E for a penalty and J for the
// rare joke scenes. They were made with the scene editor of the original
// (PIC\50.CP) and are played by f_0c7c_0004.
//
// File layout (f_14a4_05ec):
//
//	0   12  "BM-Ed1.3-WK\0"
//	12   4  frame numbers at which the sounds 2..5 start (-1: none)
//	16   1  number of frames - 1
//	17   n  frames, 164 bytes each:
//	          0 scroll x, 1 scroll y of the panorama PIC\26.VGA
//	          2 27 objects of 6 bytes: x (word), y (byte), 0, sprite (word)
//	..   1  length L of the builder's name, then L bytes (with the NUL)
const (
	SceneMagic      = "BM-Ed1.3-WK"
	SceneObjects    = 27
	sceneFrameBytes = 2 + SceneObjects*6
)

// Sprite ids of a scene object. Ids below SceneGoalLeft are cells of the
// sprite sheet PIC\27.VGA (24 cells of 12x11 per row); from SceneSmall on
// only the 4 pixels at x+4 of the cell are drawn (the ball). SceneGoalLeft
// draws the front of the left goal, larger ids the front of the right one
// (PIC\29.VGA), both at a fixed place.
const (
	SceneSmall    = 0x8e
	SceneGoalLeft = 1000
)

// SceneObject is a sprite of a frame; Y is relative to row 35 of the
// panorama (the top of the pitch).
type SceneObject struct {
	X, Y, ID int
}

// SceneFrame is one picture of a scene.
type SceneFrame struct {
	ScrollX, ScrollY int
	Objects          [SceneObjects]SceneObject
}

// Scene is a parsed goal scene.
type Scene struct {
	Sounds  [4]int // frame at which sound 2+i starts, -1 for none
	Frames  []SceneFrame
	Builder string // name stored by the editor (D:0x4bd8), in the font encoding of the game
}

// ParseScene parses a scene file.
func ParseScene(data []byte) (*Scene, error) {
	if len(data) < 17 || !bytes.Equal(data[:12], append([]byte(SceneMagic), 0)) {
		return nil, errors.New("not a goal scene (no BM-Ed1.3-WK header)")
	}
	s := &Scene{}
	for i := range s.Sounds {
		s.Sounds[i] = int(int8(data[12+i]))
	}
	n := int(int8(data[16])) + 1
	if n <= 0 {
		return nil, fmt.Errorf("goal scene: %d frames", n)
	}
	p := 17
	if p+n*sceneFrameBytes > len(data) {
		return nil, errors.New("goal scene: frames truncated")
	}
	s.Frames = make([]SceneFrame, n)
	for f := range s.Frames {
		b := data[p : p+sceneFrameBytes]
		fr := &s.Frames[f]
		fr.ScrollX, fr.ScrollY = int(b[0]), int(b[1])
		for i := range fr.Objects {
			o := b[2+i*6:]
			fr.Objects[i] = SceneObject{
				X:  int(o[0]) | int(o[1])<<8,
				Y:  int(o[2]),
				ID: int(o[4]) | int(o[5])<<8,
			}
		}
		p += sceneFrameBytes
	}
	if p < len(data) {
		l := int(data[p])
		name := data[p+1:]
		if l < len(name) {
			name = name[:l]
		}
		if i := bytes.IndexByte(name, 0); i >= 0 {
			name = name[:i]
		}
		s.Builder = string(name)
	}
	return s, nil
}

// Sprite id mirroring of f_14a4_0053 (constants D:0x52d6..0x52ee): the
// sprites 0..58 and 59..117 are the two running directions, in reverse
// order except the cycles 25..33 and 84..92; 118..141 are reversed in
// place (shifted by 21).
func mirrorSprite(id int) int {
	switch {
	case id < 59:
		if id < 25 || id > 33 {
			return 117 - id
		}
		return id + 59
	case id <= 117:
		if id < 84 || id > 92 {
			return 117 - id
		}
		return id - 59
	default:
		return 118 - id + 139
	}
}

// Prepared returns the scene as f_14a4_0053 prepares it after loading:
// when mirrored (a chance of the away team) the panorama scroll, the x
// positions and the sprites are flipped, and the x positions are always
// clamped to the panorama.
func (s *Scene) Prepared(mirrored bool) *Scene {
	out := &Scene{Sounds: s.Sounds, Builder: s.Builder, Frames: make([]SceneFrame, len(s.Frames))}
	copy(out.Frames, s.Frames)
	for f := range out.Frames {
		fr := &out.Frames[f]
		for i := range fr.Objects {
			o := &fr.Objects[i]
			x := uint16(o.X)
			if mirrored {
				w := 14
				if o.ID > 0x8d {
					w = 6
				}
				x = uint16(320 - w - int(x))
			}
			if x > 1000 {
				x = 0
			}
			if x > 0x134 && o.ID < 0x8e {
				x = 0x134
			}
			if x > 0x13c {
				x = 0x13c
			}
			o.X = int(x)
			if mirrored && o.ID < 0x8e {
				o.ID = mirrorSprite(o.ID)
			}
		}
		if mirrored {
			v := 0x89 - fr.ScrollX
			if v < 0 {
				v = 0
			}
			if v > 0x89 {
				v = 0x89
			}
			fr.ScrollX = v
		}
	}
	return out
}

// Scene reads and parses TORE\<name>, e.g. "10.T".
func (d *Dir) Scene(name string) (*Scene, error) {
	data, err := d.ReadFile("TORE/" + name)
	if err != nil {
		return nil, err
	}
	s, err := ParseScene(data)
	if err != nil {
		return nil, fmt.Errorf("TORE/%s: %w", name, err)
	}
	return s, nil
}

// SceneCounts are the highest numbers of the normal, penalty (E) and joke
// (J) scenes, from TORE\ANZAHL ("MAXSZENE:43|5|2", f_14a4_0419). Numbers
// start at 2.
type SceneCounts struct {
	Normal, Penalty, Joke int
}

// DefaultSceneCounts are the counts of the shipped TORE\ANZAHL.
var DefaultSceneCounts = SceneCounts{43, 5, 2}

// ParseSceneCounts parses TORE\ANZAHL. Like the original it reads the last
// three numbers of the file from its end.
func ParseSceneCounts(data []byte) (SceneCounts, error) {
	s := strings.TrimRight(string(data), "\x00\r\n ")
	if i := strings.IndexByte(s, ':'); i >= 0 {
		s = s[i+1:]
	}
	parts := strings.Split(s, "|")
	if len(parts) < 3 {
		return SceneCounts{}, fmt.Errorf("TORE/ANZAHL: %q", data)
	}
	var v [3]int
	for i, p := range parts[len(parts)-3:] {
		n, err := strconv.Atoi(strings.TrimSpace(p))
		if err != nil || n < 1 {
			return SceneCounts{}, fmt.Errorf("TORE/ANZAHL: %q", data)
		}
		v[i] = n
	}
	return SceneCounts{v[0], v[1], v[2]}, nil
}

// SceneCounts reads TORE\ANZAHL.
func (d *Dir) SceneCounts() (SceneCounts, error) {
	data, err := d.ReadFile("TORE/ANZAHL")
	if err != nil {
		return SceneCounts{}, err
	}
	return ParseSceneCounts(data)
}

// SceneName is the file name f_14a4_05ec builds: N.T or N.V, plus E for a
// penalty or J for a joke scene.
func SceneName(n int, goal, penalty, joke bool) string {
	s := strconv.Itoa(n)
	if goal {
		s += ".T"
	} else {
		s += ".V"
	}
	switch {
	case joke:
		s += "J"
	case penalty:
		s += "E"
	}
	return s
}
