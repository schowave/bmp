package game

import (
	"encoding/binary"
	"errors"
	"slices"
	"sort"
)

// Layout of the high-score file HIGH.xy (f_322b_21c4): 20 entries of 0x3a
// bytes, written as they are in memory.
const (
	HighscoreRows   = 20
	highscoreLen    = 0x3a
	highscoreName   = 0x1a // manager name, NUL-terminated
	highscoreClub   = 0x17 // club name from 0x1a, NUL-terminated
	highscoreTitles = 0x31 // three bytes: championships, DFB, Europe
	highscoreScore  = 0x34 // int16
	highscoreGameID = 0x36 // uint32
)

// ErrHighscoreFile is returned for a high-score file of the wrong size.
var ErrHighscoreFile = errors.New("high-score file has the wrong size")

// HighscoreEntry is a line of the high-score table.
type HighscoreEntry struct {
	Manager string
	Club    string
	Titles  [3]int // Meisterschaften, DFB-Pokale, Europapokale
	Score   int
	GameID  uint32 // D:0x668, which the original never sets
}

// HighscoreEntry returns the entry of manager mi as the file stores it.
func (s *State) HighscoreEntry(mi int) HighscoreEntry {
	m := s.Managers[mi]
	return storedHighscore(HighscoreEntry{Manager: m.Name, Club: s.Teams[m.Team].Name,
		Titles: [3]int{m.Titles[0], m.Titles[1], m.Titles[2] + m.Titles[3] + m.Titles[4]}, Score: s.Score(mi)})
}

// HighscoreFile returns the name of the high-score table of the game:
// "HIGH." with the digit of the game length and the level as shown
// ("HIGH.02").
func (s *State) HighscoreFile() string {
	t := 0
	switch s.Length() {
	case OneYear:
		t = 1
	case ThreeYears:
		t = 3
	}
	return "HIGH." + itoa(t) + itoa(5-s.Level)
}

// DecodeHighscores reads a high-score file. Rows without points are empty
// and left out.
func DecodeHighscores(data []byte) ([]HighscoreEntry, error) {
	if len(data) != HighscoreRows*highscoreLen {
		return nil, ErrHighscoreFile
	}
	var hs []HighscoreEntry
	for i := 0; i < HighscoreRows; i++ {
		b := data[i*highscoreLen : (i+1)*highscoreLen]
		e := HighscoreEntry{
			Manager: FromCP437(cString(b[:highscoreName])),
			Club:    FromCP437(cString(b[highscoreName : highscoreName+highscoreClub])),
			Titles:  [3]int{int(b[highscoreTitles]), int(b[highscoreTitles+1]), int(b[highscoreTitles+2])},
			Score:   int(int16(binary.LittleEndian.Uint16(b[highscoreScore:]))),
			GameID:  binary.LittleEndian.Uint32(b[highscoreGameID:]),
		}
		if e.Score != 0 {
			hs = append(hs, e)
		}
	}
	return hs, nil
}

// EncodeHighscores writes a high-score file; missing rows are zeros.
func EncodeHighscores(hs []HighscoreEntry) []byte {
	data := make([]byte, HighscoreRows*highscoreLen)
	for i, e := range hs[:min(len(hs), HighscoreRows)] {
		e = storedHighscore(e)
		b := data[i*highscoreLen : (i+1)*highscoreLen]
		copy(b[:highscoreName-1], ToCP437(e.Manager))
		copy(b[highscoreName:highscoreName+highscoreClub-1], ToCP437(e.Club))
		for k, n := range e.Titles {
			b[highscoreTitles+k] = byte(n)
		}
		binary.LittleEndian.PutUint16(b[highscoreScore:], uint16(int16(e.Score)))
		binary.LittleEndian.PutUint32(b[highscoreGameID:], e.GameID)
	}
	return data
}

// storedHighscore returns e as EncodeHighscores writes it: the names in
// CP437 and cut to their fields, titles and score in their bytes.
func storedHighscore(e HighscoreEntry) HighscoreEntry {
	field := func(s string, n int) string {
		b := ToCP437(s)
		return FromCP437(cString(b[:min(len(b), n-1)]))
	}
	e.Manager = field(e.Manager, highscoreName)
	e.Club = field(e.Club, highscoreClub)
	for k := range e.Titles {
		e.Titles[k] = clamp(e.Titles[k], 0, 0xff)
	}
	e.Score = clamp(e.Score, -0x8000, 0x7fff)
	return e
}

func cString(b []byte) string {
	if i := slices.Index(b, 0); i >= 0 {
		b = b[:i]
	}
	return string(b)
}

// EnterHighscore puts e into the table like f_322b_2366(2): it replaces
// the entry of the same manager and club, or else the last of 20 when e
// has more points, and sorts the table. It returns the table and the row
// of e (-1 when it did not get in).
func EnterHighscore(hs []HighscoreEntry, e HighscoreEntry) ([]HighscoreEntry, int) {
	// Compare and store e as it reads back from the file, or a name the
	// file cannot hold would never find its row again.
	e = storedHighscore(e)
	hs = append([]HighscoreEntry(nil), hs...)
	at := -1
	for i := range hs {
		if hs[i].Manager == e.Manager && hs[i].Club == e.Club {
			at = i
			break
		}
	}
	switch {
	case at >= 0:
		hs[at] = e
	case len(hs) < HighscoreRows:
		hs = append(hs, e)
	case hs[HighscoreRows-1].Score < e.Score:
		hs[HighscoreRows-1] = e
	default:
		return hs, -1
	}
	// The original swaps an entry down only when the other has more
	// points: a stable sort by descending points.
	sort.SliceStable(hs, func(a, b int) bool { return hs[a].Score > hs[b].Score })
	if len(hs) > HighscoreRows {
		hs = hs[:HighscoreRows]
	}
	for i := range hs {
		if hs[i] == e {
			return hs, i
		}
	}
	return hs, -1
}

// TakeHighscores returns the entries of the managers the season end made
// for the high-score table (f_322b_2366(2)) and clears them.
func (s *State) TakeHighscores() []HighscoreEntry {
	hs := s.Highscores
	s.Highscores = nil
	return hs
}
