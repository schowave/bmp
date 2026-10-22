package game

import "testing"

// The reference game (endless game, Borussia Dortmund, level 2) shows on
// its statistics screen at the start: income 115.916 DM, expenses
// 427.225 DM per month and a required average of 19.661. The squad skills
// are random, so the port only has to land close to these values.
func TestStartFinancesMatchReference(t *testing.T) {
	m := mana(t)
	for seed := uint32(1); seed < 6; seed++ {
		s, err := NewGame(m, Options{Managers: []NewManager{{Name: "TEST", Team: 9}}, Level: 2, Seed: seed})
		if err != nil {
			t.Fatal(err)
		}
		inc, exp, req := s.MonthlyIncome(0), s.MonthlyExpenses(0), s.RequiredAverage(0)
		if inc < 113000 || inc > 119000 || exp < 424000 || exp > 431000 || req < 19400 || req > 19900 {
			t.Errorf("seed %d: income %d expenses %d required %d", seed, inc, exp, req)
		}
	}
}
