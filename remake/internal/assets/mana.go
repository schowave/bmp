package assets

import (
	"errors"
)

// MANA.DAT layout, as read by the loader of BMMAIN.EXE (f_262f_36ec):
//
//	64 bytes      picture number of the logo of each German club
//	200 x 3 bytes strength values of every club
//	320 bytes     a table the loader skips
//	64 x (23 + 20*26) bytes  German clubs: name, then 20 player names
//	136 x 23 bytes           foreign clubs: name only
const (
	NumGermanClubs  = 64
	NumClubs        = 200
	PlayersPerClub  = 20
	clubNameLen     = 23
	playerNameLen   = 26
	manaSkipped     = 320
	manaGermanBlock = clubNameLen + PlayersPerClub*playerNameLen
	manaSize        = NumGermanClubs + NumClubs*3 + manaSkipped +
		NumGermanClubs*manaGermanBlock + (NumClubs-NumGermanClubs)*clubNameLen
)

// ClubRecord is one club of MANA.DAT.
type ClubRecord struct {
	Name     string
	Logo     int    // picture number, 0 for foreign clubs
	Strength [3]int // the three strength values of the club
	Players  []string
}

// Mana is the parsed MANA.DAT.
type Mana struct {
	Clubs   []ClubRecord
	Skipped []byte // the 320-byte table the loader does not use
}

// ParseMana parses MANA.DAT. Names keep the game's CP437 encoding.
func ParseMana(data []byte) (*Mana, error) {
	if len(data) != manaSize {
		return nil, errors.New("MANA.DAT has an unexpected size")
	}
	m := &Mana{Clubs: make([]ClubRecord, NumClubs)}
	p := 0
	for i := 0; i < NumGermanClubs; i++ {
		m.Clubs[i].Logo = int(data[p])
		p++
	}
	for i := 0; i < NumClubs; i++ {
		for k := 0; k < 3; k++ {
			m.Clubs[i].Strength[k] = int(data[p])
			p++
		}
	}
	m.Skipped = append([]byte(nil), data[p:p+manaSkipped]...)
	p += manaSkipped
	for i := 0; i < NumClubs; i++ {
		m.Clubs[i].Name = cString(data[p:p+clubNameLen], 0)
		p += clubNameLen
		if i >= NumGermanClubs {
			continue
		}
		for k := 0; k < PlayersPerClub; k++ {
			name := cString(data[p:p+playerNameLen], 0)
			p += playerNameLen
			m.Clubs[i].Players = append(m.Clubs[i].Players, name)
		}
	}
	return m, nil
}
