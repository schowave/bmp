package game

import (
	"encoding/json"
	"testing"
)

// TestLoadDisplay keeps an options setting with everything off and gives
// a game saved before the options existed the defaults.
func TestLoadDisplay(t *testing.T) {
	data, _ := json.Marshal(saveFile{Version: saveVersion, State: &State{}})
	s, err := Load(data, nil)
	if err != nil {
		t.Fatal(err)
	}
	if s.Display != (Display{}) {
		t.Fatalf("%+v", s.Display)
	}
	s, err = Load([]byte(`{"Version":1,"State":{}}`), nil)
	if err != nil {
		t.Fatal(err)
	}
	if s.Display != DefaultDisplay() {
		t.Fatalf("%+v", s.Display)
	}
	if _, err := Load([]byte(`{"Version":1}`), nil); err != ErrSaveVersion {
		t.Fatal(err)
	}
}
