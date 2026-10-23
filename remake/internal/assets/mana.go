package assets

import (
	"errors"
	"fmt"
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

// Mana is the club database: MANA.DAT of the original, clubs.json of an
// extract.
type Mana struct {
	Clubs []ClubRecord
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

// Player name slots of a German club by position: goalkeepers 0-1,
// defenders 2-7, midfielders 8-14, forwards 15-19 (D:0x9432).
var rosterBlocks = [5]int{0, 2, 8, 15, 20}

type clubJSON struct {
	Name     string      `json:"name"`
	Logo     int         `json:"logo,omitempty"`
	Strength [3]int      `json:"strength"`
	Players  *rosterJSON `json:"players,omitempty"`
}

// rosterJSON lists the player names of a club by position; an empty name
// is an empty slot.
type rosterJSON struct {
	Goalkeepers []string `json:"goalkeepers"`
	Defenders   []string `json:"defenders"`
	Midfielders []string `json:"midfielders"`
	Forwards    []string `json:"forwards"`
}

func (r *rosterJSON) blocks() [4]*[]string {
	return [4]*[]string{&r.Goalkeepers, &r.Defenders, &r.Midfielders, &r.Forwards}
}

// EncodeClubs writes the club database as clubs.json: the 64 German clubs
// with logo (picture number) and player names, then the foreign clubs.
// Names are UTF-8.
func EncodeClubs(m *Mana) ([]byte, error) {
	out := make([]clubJSON, len(m.Clubs))
	for i, c := range m.Clubs {
		out[i] = clubJSON{Name: cp437ToUTF8(c.Name), Logo: c.Logo, Strength: c.Strength}
		if i >= NumGermanClubs {
			continue
		}
		if len(c.Players) != PlayersPerClub {
			return nil, fmt.Errorf("club %d: %d player names", i, len(c.Players))
		}
		r := &rosterJSON{}
		for k, b := range r.blocks() {
			for _, name := range c.Players[rosterBlocks[k]:rosterBlocks[k+1]] {
				*b = append(*b, cp437ToUTF8(name))
			}
		}
		out[i].Players = r
	}
	return marshalJSON(out)
}

// DecodeClubs reads clubs.json. It checks what the game relies on: the
// number of clubs, players per position, the length of the names and that
// they fit code page 437.
func DecodeClubs(data []byte) (*Mana, error) {
	var in []clubJSON
	if err := unmarshalJSON(data, &in); err != nil {
		return nil, err
	}
	if len(in) != NumClubs {
		return nil, fmt.Errorf("%d clubs, want %d", len(in), NumClubs)
	}
	m := &Mana{Clubs: make([]ClubRecord, NumClubs)}
	for i, cj := range in {
		c := &m.Clubs[i]
		name, err := manaName(cj.Name, clubNameLen)
		if err != nil {
			return nil, fmt.Errorf("club %d: %w", i, err)
		}
		c.Name, c.Logo, c.Strength = name, cj.Logo, cj.Strength
		for k, v := range cj.Strength {
			if v < 0 || v > 255 {
				return nil, fmt.Errorf("club %d (%s): strength %d is %d", i, cj.Name, k, v)
			}
		}
		if i >= NumGermanClubs {
			if cj.Players != nil || cj.Logo != 0 {
				return nil, fmt.Errorf("club %d (%s): foreign clubs have no players and no logo", i, cj.Name)
			}
			continue
		}
		if cj.Logo <= 0 {
			return nil, fmt.Errorf("club %d (%s): logo %d", i, cj.Name, cj.Logo)
		}
		if cj.Players == nil {
			return nil, fmt.Errorf("club %d (%s): no players", i, cj.Name)
		}
		for k, b := range cj.Players.blocks() {
			if want := rosterBlocks[k+1] - rosterBlocks[k]; len(*b) != want {
				return nil, fmt.Errorf("club %d (%s): %d players in position %d, want %d", i, cj.Name, len(*b), k, want)
			}
			for _, pn := range *b {
				name, err := manaName(pn, playerNameLen)
				if err != nil {
					return nil, fmt.Errorf("club %d (%s): %w", i, cj.Name, err)
				}
				c.Players = append(c.Players, name)
			}
		}
	}
	return m, nil
}

// manaName converts a name to CP437 and checks that it fits a slot of size
// bytes of MANA.DAT, with its NUL.
func manaName(s string, size int) (string, error) {
	b, err := utf8ToCP437(s)
	if err != nil {
		return "", err
	}
	if len(b) >= size {
		return "", fmt.Errorf("%q is longer than %d characters", s, size-1)
	}
	return b, nil
}
