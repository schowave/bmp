package assets

import (
	"bytes"
	"io/fs"
	"os"
	"path/filepath"
	"testing"

	"github.com/schowave/bmp/remake/internal/embedded"
)

// dataDir returns the files of the original built into the remake.
func dataDir(t *testing.T) *Dir {
	t.Helper()
	d, err := OpenDir(embedded.FS())
	if err != nil {
		t.Fatal(err)
	}
	return d
}

// gameDir returns the original game directory of the repository (../bmp), or
// skips the test when it is not there. Only tests of BMMAIN.EXE and of the
// extraction need it.
func gameDir(t *testing.T) *Dir {
	t.Helper()
	path := filepath.Join("..", "..", "..", "bmp")
	if _, err := os.Stat(filepath.Join(path, "BMMAIN.EXE")); err != nil {
		t.Skip("original game files not found in", path)
	}
	d, err := OpenDir(os.DirFS(path))
	if err != nil {
		t.Fatal(err)
	}
	return d
}

func TestDecodePicRuns(t *testing.T) {
	// 4x2 picture: literal of 3 bytes, run of 5 x 7, end marker.
	data := []byte{4, 0, 2, 0, 0, 0, 0x02, 1, 2, 3, 0x84, 7, 0x80}
	p, err := DecodePic(data)
	if err != nil {
		t.Fatal(err)
	}
	want := []byte{1, 2, 3, 7, 7, 7, 7, 7}
	if string(p.Pix) != string(want) {
		t.Fatalf("got %v, want %v", p.Pix, want)
	}
}

func TestAllPicturesDecodeToTheirSize(t *testing.T) {
	d := dataDir(t)
	n := 0
	for upper := range d.names {
		ext := filepath.Ext(upper)
		if ext != ".VGA" && ext != ".CP" {
			continue
		}
		data, err := d.ReadFile(upper)
		if err != nil {
			t.Fatal(err)
		}
		p, err := DecodePic(data)
		if err != nil {
			t.Fatalf("%s: %v", upper, err)
		}
		if len(p.Pix) != p.W*p.H || p.W == 0 || p.H == 0 {
			t.Fatalf("%s: %dx%d with %d pixels", upper, p.W, p.H, len(p.Pix))
		}
		n++
	}
	if n < 150 {
		t.Fatalf("only %d pictures found", n)
	}
}

func TestFullScreenPictureIs320x240(t *testing.T) {
	d := dataDir(t)
	p, err := d.Pic(22, true)
	if err != nil {
		t.Fatal(err)
	}
	if p.W != 320 || p.H != 240 {
		t.Fatalf("got %dx%d", p.W, p.H)
	}
}

func TestMana(t *testing.T) {
	d := dataDir(t)
	m, err := d.Mana()
	if err != nil {
		t.Fatal(err)
	}
	c := m.Clubs[0]
	if c.Name != "1.FC KAISERSLAUTERN" || c.Logo != 88 || len(c.Players) != 20 || c.Players[0] != "EHRMANN" {
		t.Fatalf("club 0: %+v", c)
	}
	if c.Strength != [3]int{94, 94, 95} {
		t.Fatalf("club 0 strength %v", c.Strength)
	}
	if m.Clubs[64].Name == "" || m.Clubs[199].Name == "" || m.Clubs[64].Players != nil {
		t.Fatalf("foreign clubs: %+v %+v", m.Clubs[64], m.Clubs[199])
	}
}

func TestFonts(t *testing.T) {
	d := gameDir(t)
	img, err := d.MainImage()
	if err != nil {
		t.Fatal(err)
	}
	if img.CS != 0x3a01 || img.IP != 0x16 {
		t.Fatalf("entry %04x:%04x", img.CS, img.IP)
	}
	fonts, err := Fonts(img)
	if err != nil {
		t.Fatal(err)
	}
	for name, h := range map[string]int{"flow.font": 8, "micro4.font": 6, "newbold.font": 16} {
		f := fonts[name]
		if f == nil || f.Height != h {
			t.Fatalf("%s: %+v", name, f)
		}
	}
	a := fonts["flow.font"].Glyph('A')
	if a.W != 7 || !a.Rows[0][1] || a.Rows[0][0] {
		t.Fatalf("flow A: %+v", a)
	}
	// CP437 Ä is drawn with the glyph of '['.
	if fonts["flow.font"].Glyph(0x8e) != fonts["flow.font"].Glyph('[') {
		t.Fatal("umlaut remap missing")
	}
}

func TestExtractWorksWithoutTheExecutable(t *testing.T) {
	d := gameDir(t)
	out := t.TempDir()
	if err := Extract(d, out); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(out, "BMMAIN.EXE")); err == nil {
		t.Fatal("extract contains the executable")
	}
	e, err := OpenDir(os.DirFS(out))
	if err != nil {
		t.Fatal(err)
	}
	want, err := d.Fonts()
	if err != nil {
		t.Fatal(err)
	}
	got, err := e.Fonts()
	if err != nil {
		t.Fatal(err)
	}
	for name, f := range want {
		g := got[name]
		if g == nil || g.Height != f.Height || g.Glyph('A').W != f.Glyph('A').W || g.Glyph(0x8e) == g.Glyph(' ') {
			t.Fatalf("font %s differs after the round trip", name)
		}
		for c := 0x20; c < 0x7f; c++ {
			a, b := f.Glyph(byte(c)), g.Glyph(byte(c))
			for y := range a.Rows {
				for x := range a.Rows[y] {
					if a.Rows[y][x] != b.Rows[y][x] {
						t.Fatalf("font %s glyph %q differs", name, rune(c))
					}
				}
			}
		}
	}
	if _, err := e.Pic(22, true); err != nil {
		t.Fatal(err)
	}
	if _, err := e.Mana(); err != nil {
		t.Fatal(err)
	}
}

func TestEmbeddedFilesMatchTheOriginal(t *testing.T) {
	d := gameDir(t)
	out := t.TempDir()
	if err := Extract(d, out); err != nil {
		t.Fatal(err)
	}
	want, got := os.DirFS(out), embedded.FS()
	n := 0
	err := fs.WalkDir(want, ".", func(path string, e fs.DirEntry, err error) error {
		if err != nil || e.IsDir() {
			return err
		}
		a, _ := fs.ReadFile(want, path)
		b, err := fs.ReadFile(got, path)
		if err != nil || !bytes.Equal(a, b) {
			t.Errorf("%s differs from the extract of %s, run mise run remake:extract", path, "../bmp")
		}
		n++
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	m := 0
	err = fs.WalkDir(got, ".", func(path string, e fs.DirEntry, err error) error {
		if err == nil && !e.IsDir() {
			m++
		}
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
	if m != n {
		t.Errorf("embedded %d files, the extract has %d", m, n)
	}
}

func TestEncodeFontsIsStable(t *testing.T) {
	fonts, err := dataDir(t).Fonts()
	if err != nil {
		t.Fatal(err)
	}
	first, err := EncodeFonts(fonts)
	if err != nil {
		t.Fatal(err)
	}
	for range 5 {
		again, err := EncodeFonts(fonts)
		if err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(first, again) {
			t.Fatal("EncodeFonts gives different bytes for the same fonts")
		}
	}
}
