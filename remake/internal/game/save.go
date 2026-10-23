package game

import (
	"encoding/json"
	"errors"

	"github.com/schowave/bmp/remake/internal/assets"
)

// saveVersion changes whenever the save format changes incompatibly.
const saveVersion = 1

// Errors of Save and Load. The screens show the original's own alerts.
var (
	ErrNotSavable  = errors.New("games can only be saved during a turn")
	ErrSaveVersion = errors.New("save game of another version")
)

type saveFile struct {
	Version int
	State   *State
}

// Save serialises the game. Games are saved during a manager's turn.
func (s *State) Save() ([]byte, error) {
	if s.Loop.Phase != PhaseTurn && s.Loop.Phase != PhaseContracts {
		return nil, ErrNotSavable
	}
	return json.Marshal(saveFile{Version: saveVersion, State: s})
}

// Load restores a game saved with Save.
func Load(data []byte, mana *assets.Mana) (*State, error) {
	var f struct {
		Version int
		State   json.RawMessage
	}
	if err := json.Unmarshal(data, &f); err != nil {
		return nil, err
	}
	if f.Version != saveVersion || len(f.State) == 0 || string(f.State) == "null" {
		return nil, ErrSaveVersion
	}
	// A game saved before the options existed keeps the defaults.
	s := &State{Display: DefaultDisplay()}
	if err := json.Unmarshal(f.State, s); err != nil {
		return nil, err
	}
	s.mana = mana
	return s, nil
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
