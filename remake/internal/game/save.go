package game

import (
	"encoding/json"
	"errors"

	"github.com/schowave/bmp/remake/internal/assets"
)

// saveVersion changes whenever the save format changes incompatibly.
const saveVersion = 1

type saveFile struct {
	Version int
	State   *State
}

// Save serialises the game. Games are saved during a manager's turn.
func (s *State) Save() ([]byte, error) {
	if s.Loop.Phase != PhaseTurn && s.Loop.Phase != PhaseContracts {
		return nil, errors.New("Spielstände können nur im Hauptmenü gespeichert werden")
	}
	return json.Marshal(saveFile{Version: saveVersion, State: s})
}

// Load restores a game saved with Save.
func Load(data []byte, mana *assets.Mana) (*State, error) {
	var f saveFile
	if err := json.Unmarshal(data, &f); err != nil {
		return nil, err
	}
	if f.Version != saveVersion || f.State == nil {
		return nil, errors.New("Spielstand aus falscher Version !")
	}
	f.State.mana = mana
	return f.State, nil
}

// MatchdayDay returns the season day on which matchday md of league l is
// scheduled.
func (s *State) MatchdayDay(l, md int) int {
	n := 0
	for e := 0; e < NumEvents; e++ {
		if seasonEvents[e]&(1<<l) != 0 {
			n++
			if n == md {
				return EventDay(e)
			}
		}
	}
	return 0
}
