package game

import (
	"bytes"
	"encoding/binary"
	"os"
	"path/filepath"
	"testing"

	"github.com/schowave/bmp/remake/internal/assets"
)

// The tests in this file compare constant tables of the port with the
// unpacked BMMAIN.EXE of the original, when it is available in ../bmp.

const (
	segF = 0x4238 * 16
	segD = 0x4cb3 * 16
)

type origImage struct{ b []byte }

func original(t *testing.T) *origImage {
	t.Helper()
	path := filepath.Join("..", "..", "..", "bmp", "BMMAIN.EXE")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Skip("original BMMAIN.EXE not found:", path)
	}
	img, err := assets.Unexepack(data)
	if err != nil {
		t.Fatal(err)
	}
	return &origImage{img.Data}
}

func (o *origImage) dBytes(off, n int) []byte { return o.b[segD+off : segD+off+n] }

// dFarString follows a far pointer stored at D:off.
func (o *origImage) dFarString(off int) string {
	p := int(binary.LittleEndian.Uint16(o.b[segD+off:]))
	seg := int(binary.LittleEndian.Uint16(o.b[segD+off+2:]))
	s := o.b[seg*16+p:]
	return string(s[:bytes.IndexByte(s, 0)])
}

func TestEventTableMatchesOriginal(t *testing.T) {
	o := original(t)
	if got := o.dBytes(0x2280, NumEvents); !bytes.Equal(got, seasonEvents[:]) {
		t.Fatalf("event table differs:\n%x\n%x", got, seasonEvents)
	}
}

func TestFixtureStringsMatchOriginal(t *testing.T) {
	o := original(t)
	for k := range fixtures18 {
		if s := o.dFarString(0x6fc + 4*(k+1)); s != fixtures18[k] {
			t.Errorf("18-club round %d: %q", k+1, s)
		}
	}
	for k := range fixtures20 {
		if s := o.dFarString(0x744 + 4*(k+1)); s != fixtures20[k] {
			t.Errorf("20-club round %d: %q", k+1, s)
		}
	}
}

func TestLeagueConstantsMatchOriginal(t *testing.T) {
	o := original(t)
	b := o.dBytes(0x225e, 0x10)
	for l := 0; l < NumLeagues; l++ {
		if int(b[l]) != TeamsPerLeague[l] || int(b[4+l]) != MatchdaysPerLeague[l] ||
			int(b[8+l]) != Relegated[l] || int(b[0xc+l]) != boolInt(RelegationPlayoff[l]) {
			t.Errorf("league %d: % x", l, b)
		}
	}
}

func TestFixturesAreRoundRobins(t *testing.T) {
	for l := 0; l < NumLeagues; l++ {
		met := map[[2]int]int{}
		for md := 1; md <= MatchdaysPerLeague[l]; md++ {
			seen := map[int]bool{}
			for _, p := range Fixtures(l, md) {
				if seen[p.Home] || seen[p.Away] {
					t.Fatalf("league %d md %d: club twice", l, md)
				}
				seen[p.Home], seen[p.Away] = true, true
				met[[2]int{p.Home, p.Away}]++
			}
		}
		n := TeamsPerLeague[l]
		if len(met) != n*(n-1) {
			t.Fatalf("league %d: %d distinct home/away pairs", l, len(met))
		}
	}
}

func TestDateOf(t *testing.T) {
	if d := DateOf(1992, 0); d != (Date{29, 6, 1992}) {
		t.Fatal(d)
	}
	if d := DateOf(1992, EventDay(92)); d != (Date{16, 5, 1993}) {
		t.Fatal(d)
	}
	if WeekdayNames[Weekday(0)] != "Samstag" {
		t.Fatal(Weekday(0))
	}
}

func TestRandMatchesMSC(t *testing.T) {
	r := Rand{Seed: 1}
	// First values of rand() after srand(1) in Microsoft C.
	want := []int{41, 18467, 6334, 26500, 19169}
	for i, w := range want {
		if got := r.Next(); got != w {
			t.Fatalf("rand %d: got %d want %d", i, got, w)
		}
	}
}
