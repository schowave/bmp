package seasons

import (
	"testing"
	"testing/fstest"

	"github.com/schowave/bmp/remake/internal/assets"
	"github.com/schowave/bmp/remake/internal/embedded"
)

func TestParseID(t *testing.T) {
	s, err := ParseID("1999-00")
	if err != nil || s.Year != 1999 || s.Label() != "1999/00" {
		t.Fatalf("got %+v, %v", s, err)
	}
	for _, bad := range []string{"1999-01", "99-00", "1999/00", "1999", "abcd-ef"} {
		if _, err := ParseID(bad); err == nil {
			t.Errorf("%q accepted", bad)
		}
	}
}

// testSeason is a season file made from the clubs of the original.
func testSeason(t *testing.T) []byte {
	t.Helper()
	dir, err := assets.OpenDir(embedded.FS())
	if err != nil {
		t.Fatal(err)
	}
	m, err := dir.Mana()
	if err != nil {
		t.Fatal(err)
	}
	b, err := assets.EncodeClubs(m)
	if err != nil {
		t.Fatal(err)
	}
	return b
}

func TestListAndMana(t *testing.T) {
	b := testSeason(t)
	fsys := fstest.MapFS{
		"2008-09.json": {Data: b},
		"1995-96.json": {Data: b},
		"README.md":    {Data: []byte("x")},
		"pic/1000.png": {Data: []byte("x")},
	}
	got := list(fsys)
	if len(got) != 2 || got[0].ID != "1995-96" || got[1].Year != 2008 {
		t.Fatalf("got %+v", got)
	}
	m, err := mana(fsys, "1995-96")
	if err != nil || len(m.Clubs) != assets.NumClubs {
		t.Fatalf("got %v", err)
	}
	if _, err := mana(fsys, "2001-02"); err == nil {
		t.Fatal("missing season loaded")
	}
}

// TestBuiltInSeasonsLoad checks every season built into the remake: 64
// German clubs with a logo and 136 foreign ones without, every one named
// once.
func TestBuiltInSeasonsLoad(t *testing.T) {
	for _, s := range List() {
		m, err := Mana(s.ID)
		if err != nil {
			t.Error(err)
			continue
		}
		names := map[string]bool{}
		for i, c := range m.Clubs {
			if c.Name == "" || names[c.Name] {
				t.Errorf("%s club %d: name %q missing or twice", s.ID, i, c.Name)
			}
			names[c.Name] = true
			german := i < assets.NumGermanClubs
			if german != (c.Logo > 0) {
				t.Errorf("%s club %d (%s): logo %d", s.ID, i, c.Name, c.Logo)
			}
		}
	}
	// Every season from 1990/91 on, without a gap, at least to 2026/27.
	l := List()
	for i, s := range l {
		if s.Year != 1990+i {
			t.Fatalf("season %d is %s, want %s", i, s.ID, ForYear(1990+i).ID)
		}
	}
	if len(l) == 0 || l[len(l)-1].Year < 2026 {
		t.Errorf("%d seasons, want 1990/91..2026/27 at least", len(l))
	}
}
