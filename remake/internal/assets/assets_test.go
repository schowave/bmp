package assets

import (
	"bytes"
	"fmt"
	"image"
	"image/png"
	"io/fs"
	"os"
	"path/filepath"
	"reflect"
	"strings"
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
func gameDir(t *testing.T) *Original {
	t.Helper()
	path := filepath.Join("..", "..", "..", "bmp")
	if _, err := os.Stat(filepath.Join(path, "BMMAIN.EXE")); err != nil {
		t.Skip("original game files not found in", path)
	}
	d, err := OpenOriginal(os.DirFS(path))
	if err != nil {
		t.Fatal(err)
	}
	return d
}

func TestParsePicRuns(t *testing.T) {
	// 4x2 picture: literal of 3 bytes, run of 5 x 7, end marker.
	data := []byte{4, 0, 2, 0, 0, 0, 0x02, 1, 2, 3, 0x84, 7, 0x80}
	p, err := ParsePic(data)
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
	err := fs.WalkDir(d.fsys, picDir, func(path string, e fs.DirEntry, err error) error {
		if err != nil || e.IsDir() {
			return err
		}
		data, err := d.ReadFile(path)
		if err != nil {
			return err
		}
		p, err := DecodePic(data)
		if err != nil {
			return fmt.Errorf("%s: %w", path, err)
		}
		if len(p.Pix) != p.W*p.H || p.W == 0 || p.H == 0 {
			return fmt.Errorf("%s: %dx%d with %d pixels", path, p.W, p.H, len(p.Pix))
		}
		n++
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if n < 150 {
		t.Fatalf("only %d pictures found", n)
	}
}

func TestPicPNGRoundTrip(t *testing.T) {
	p := &Pic{W: 3, H: 2, Pix: []byte{0, 1, 255, 7, 7, 128}}
	var pal Palette
	data, err := EncodePic(p, &pal)
	if err != nil {
		t.Fatal(err)
	}
	got, err := DecodePic(data)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(got, p) {
		t.Fatalf("got %+v, want %+v", got, p)
	}
	var b bytes.Buffer
	if err := png.Encode(&b, image.NewRGBA(image.Rect(0, 0, 2, 2))); err != nil {
		t.Fatal(err)
	}
	if _, err := DecodePic(b.Bytes()); err == nil {
		t.Fatal("a PNG without palette indices was accepted")
	}
}

func TestPalettesJSONRoundTrip(t *testing.T) {
	pals, err := dataDir(t).Palettes()
	if err != nil {
		t.Fatal(err)
	}
	if len(pals) != 3 {
		t.Fatalf("%d palettes", len(pals))
	}
	got, err := DecodePalettes(EncodePalettes(pals))
	if err != nil || !reflect.DeepEqual(got, pals) {
		t.Fatalf("round trip: %v", err)
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
	if m.Clubs[1].Name != "FC BAYERN M\x9aNCHEN" {
		t.Fatalf("club 1 is not in CP437: %q", m.Clubs[1].Name)
	}
}

func TestClubsJSON(t *testing.T) {
	m, err := dataDir(t).Mana()
	if err != nil {
		t.Fatal(err)
	}
	data, err := EncodeClubs(m)
	if err != nil {
		t.Fatal(err)
	}
	if got, err := DecodeClubs(data); err != nil || !reflect.DeepEqual(got, m) {
		t.Fatalf("round trip: %v", err)
	}
	for _, want := range []string{`"FC BAYERN MÜNCHEN"`, `"STANDARD L▄TTICH"`} {
		if !strings.Contains(string(data), want) {
			t.Errorf("clubs.json lacks %s", want)
		}
	}
	file := string(data)
	for name, broken := range map[string]string{
		"long name":      strings.Replace(file, `"FC BAYERN MÜNCHEN"`, `"FC BAYERN MÜNCHEN E.V. 1900"`, 1),
		"not CP437":      strings.Replace(file, `"FC BAYERN MÜNCHEN"`, `"FC BAYERN MÜNCHEN ✓"`, 1),
		"one keeper":     strings.Replace(file, "\"goalkeepers\": [\n\t\t\t\t\"EHRMANN\",", `"goalkeepers": [`, 1),
		"typo":           strings.Replace(file, `"strength"`, `"strenght"`, 1),
		"too few":        "[]",
		"foreign roster": strings.Replace(file, `"name": "FC LIVERPOOL"`, `"name": "FC LIVERPOOL", "players": {}`, 1),
	} {
		if broken == file {
			t.Fatalf("%s: the test did not change the file", name)
		}
		if _, err := DecodeClubs([]byte(broken)); err == nil {
			t.Errorf("%s: accepted", name)
		}
	}
}

func TestCP437RoundTrip(t *testing.T) {
	var all []byte
	for c := 1; c < 256; c++ {
		all = append(all, byte(c))
	}
	got, err := ToCP437(FromCP437(string(all)))
	if err != nil || got != string(all) {
		t.Fatalf("round trip: %v", err)
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

// TestExtractIsLossless reads everything of an extract back and compares it
// with the original files.
func TestExtractIsLossless(t *testing.T) {
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
	pics := d.files("PIC", ".VGA", ".CP")
	if len(pics) < 150 {
		t.Fatalf("%d pictures in the original", len(pics))
	}
	for _, upper := range pics {
		ext := filepath.Ext(upper)
		var n int
		if _, err := fmt.Sscan(strings.TrimSuffix(strings.TrimPrefix(upper, "PIC/"), ext), &n); err != nil {
			t.Fatal(upper, err)
		}
		a, err := d.Pic(n, ext == ".CP")
		if err != nil {
			t.Fatal(err)
		}
		if b, err := e.Pic(n, ext == ".CP"); err != nil || !reflect.DeepEqual(a, b) {
			t.Fatalf("%s differs after the round trip: %v", upper, err)
		}
	}
	pa, err := d.Palettes()
	if err != nil {
		t.Fatal(err)
	}
	if pb, err := e.Palettes(); err != nil || !reflect.DeepEqual(pa, pb) {
		t.Fatalf("palettes differ after the round trip: %v", err)
	}
	ma, err := d.Mana()
	if err != nil {
		t.Fatal(err)
	}
	if mb, err := e.Mana(); err != nil || !reflect.DeepEqual(ma, mb) {
		t.Fatalf("club database differs after the round trip: %v", err)
	}
	scenes := d.files("TORE", ".T", ".V", ".TE", ".VE", ".TJ", ".VJ")
	if len(scenes) != 94 {
		t.Fatalf("%d scenes in the original", len(scenes))
	}
	for _, upper := range scenes {
		name := strings.TrimPrefix(upper, "TORE/")
		a, err := d.Scene(name)
		if err != nil {
			t.Fatal(err)
		}
		if b, err := e.Scene(name); err != nil || !reflect.DeepEqual(a, b) {
			t.Fatalf("%s differs after the round trip: %v", upper, err)
		}
	}
	ca, err := d.SceneCounts()
	if err != nil {
		t.Fatal(err)
	}
	if cb, err := e.SceneCounts(); err != nil || ca != cb {
		t.Fatalf("scene counts differ: %+v, %v", cb, err)
	}
}

func TestOpenDirExplainsTheOriginal(t *testing.T) {
	d := gameDir(t)
	_, err := OpenDir(d.fsys)
	if err == nil || !strings.Contains(err.Error(), "extract it first") {
		t.Fatalf("got %v", err)
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
