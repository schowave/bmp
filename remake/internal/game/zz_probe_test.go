package game

import "testing"

func TestZZProbe(t *testing.T) {
	m := mana(t)
	bad, awayOrder := 0, 0
	for seed := uint32(1); seed <= 20; seed++ {
		opt := Options{Seed: seed, Level: 5}
		for i := 0; i < 4; i++ { opt.Managers = append(opt.Managers, NewManager{Name: "M" + itoa(i), Team: (int(seed)*7 + i*13) % 64}) }
		s, err := NewGame(m, opt)
		if err != nil { t.Fatal(err) }
		s.Start()
		for s.Seasons < 2 {
			switch s.Loop.Phase {
			case PhaseTurn:
				for mi := range s.Managers { s.TakeInbox(mi) }
				s.EndTurn()
			case PhaseMatches:
				for md := s.NextMatchDay(); md != nil; md = s.NextMatchDay() {
					// record ET marks before finish clears them
					et := map[int]bool{}
					for md.Step() {
						if md.Minute > 91 && md.Minute < 120 {
							for i := range md.Fixtures { if md.ExtraTime(i) { et[i] = true } }
						}
					}
					for _, e := range md.Events {
						if e.Minute > 90 && e.Minute < 121 && e.Manager >= 0 && (e.Kind == EvInjury) {
							f := md.Fixtures[e.Fixture]
							_ = f
							if et[e.Fixture] { continue }
							bad++
							t.Logf("seed %d %s: injury minute %d fixture %d", seed, md.Title, e.Minute, e.Fixture)
						}
					}
				}
			case PhaseContracts:
				for _, si := range s.ExpiringContracts() { _ = s.NegotiateContract(s.Loop.Current, si, 2, s.ContractHint(s.Loop.Current, si, 2)*2, 0) }
				s.ContractsDone()
			case PhaseGameOver:
				return
			}
		}
	}
	t.Logf("bad=%d %d", bad, awayOrder)
}
