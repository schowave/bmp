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

// TestBuiltInSeasonsLoad checks every season built into the remake.
func TestBuiltInSeasonsLoad(t *testing.T) {
	for _, s := range List() {
		if _, err := Mana(s.ID); err != nil {
			t.Error(err)
		}
	}
}
