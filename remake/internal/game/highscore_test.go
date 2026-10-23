package game

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

func TestEnterHighscore(t *testing.T) {
	var hs []HighscoreEntry
	for i := 0; i < 20; i++ {
		hs = append(hs, HighscoreEntry{Manager: string(rune('A' + i)), Club: "X", Score: 100 - i})
	}
	// The same manager and club is replaced, even with fewer points.
	got, row := EnterHighscore(hs, HighscoreEntry{Manager: "B", Club: "X", Score: 50})
	if row != 19 || got[19].Manager != "B" || got[1].Manager != "C" || hs[1].Manager != "B" {
		t.Fatalf("row %d %+v", row, got)
	}
	// A new one only replaces the last when it has more points.
	if _, row := EnterHighscore(hs, HighscoreEntry{Manager: "Z", Club: "Y", Score: 81}); row != -1 {
		t.Fatalf("row %d", row)
	}
	got, row = EnterHighscore(hs, HighscoreEntry{Manager: "Z", Club: "Y", Score: 95})
	if row != 6 || len(got) != 20 || got[19].Manager != "S" {
		t.Fatalf("row %d %+v", row, got)
	}
}

// TestHighscoreFile reads a high-score table of the original and writes
// it back unchanged.
func TestHighscoreFile(t *testing.T) {
	path := filepath.Join("..", "..", "..", "bmp", "HIGH.02")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Skip("original game files not found:", err)
	}
	hs, err := DecodeHighscores(data)
	if err != nil {
		t.Fatal(err)
	}
	if len(hs) == 0 || hs[0].Manager != "MICHEL" || hs[0].Club != "BORUSSIA DORTMUND" || hs[0].Score != 125 {
		t.Fatalf("%+v", hs)
	}
	if got := EncodeHighscores(hs); !bytes.Equal(got, data) {
		t.Fatal("the table is not written back as it was read")
	}
	if _, err := DecodeHighscores(data[1:]); err == nil {
		t.Fatal("a short file was accepted")
	}
}

func TestHighscoreUmlauts(t *testing.T) {
	hs := []HighscoreEntry{{Manager: "JÜRGEN", Club: "FC KÖLN", Titles: [3]int{1, 2, 3}, Score: -7}}
	got, err := DecodeHighscores(EncodeHighscores(hs))
	if err != nil || len(got) != 1 || got[0] != hs[0] {
		t.Fatalf("%+v %v", got, err)
	}
}

// TestSeasonEndHighscores checks that every season end hands the entries
// of all managers to the high-score table (f_322b_2366(2)).
func TestSeasonEndHighscores(t *testing.T) {
	s, err := NewGame(mana(t), Options{Managers: []NewManager{{Name: "A", Team: 0}, {Name: "B", Team: 1}}, Seed: 3})
	if err != nil {
		t.Fatal(err)
	}
	s.Start()
	for s.Seasons == 0 {
		if len(s.Highscores) != 0 {
			t.Fatal("high scores before the season end")
		}
		autoplay(t, s, 1)
	}
	hs := s.TakeHighscores()
	if len(hs) != 2 || hs[0].Manager != "A" || hs[1].Manager != "B" || len(s.Highscores) != 0 {
		t.Fatalf("%+v", hs)
	}
}
