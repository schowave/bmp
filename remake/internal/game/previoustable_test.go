package game

import "testing"

// A real season qualifies the clubs of the previous final table, not the
// clubs the id shuffle put on the first places.
func TestNewGameQualifiesPreviousTable(t *testing.T) {
	s, err := NewGame(mana(t), Options{Managers: []NewManager{{Name: "TEST", Team: 0}}, Level: 2, Seed: 5})
	if err != nil {
		t.Fatal(err)
	}
	names := []string{s.Teams[3].Name, s.Teams[5].Name, s.Teams[7].Name}
	ids := s.previousTable(append(names, "GIBT ES NICHT"))
	if len(ids) != 3 || ids[0] != 3 || ids[2] != 7 {
		t.Fatalf("previousTable = %v", ids)
	}
	s.qualifyEurope(ids)
	if q := s.Cups.Qualified[CupLandesmeister]; len(q) == 0 || q[0] != 3 {
		t.Errorf("Landesmeister = %v, want club 3 first", q)
	}
}
