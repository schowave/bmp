package game

import (
	"math/rand"
	"testing"
)

// TestStressRandomManagers plays many seasons with four managers that take
// random actions, to find crashes and broken invariants.
func TestStressRandomManagers(t *testing.T) {
	if testing.Short() {
		t.Skip("long")
	}
	m := mana(t)
	for seed := uint32(1); seed <= 6; seed++ {
		rnd := rand.New(rand.NewSource(int64(seed)))
		opt := Options{Seed: seed, Level: int(seed % 6), Historic: seed%2 == 0}
		for i := 0; i < 4; i++ {
			opt.Managers = append(opt.Managers, NewManager{Name: "M" + itoa(i), Team: (int(seed)*7 + i*13) % 64})
		}
		s, err := NewGame(m, opt)
		if err != nil {
			t.Fatal(err)
		}
		s.Start()
		for steps := 0; s.Seasons < 8 && steps < 200000; steps++ {
			switch s.Loop.Phase {
			case PhaseTurn:
				randomActions(s, s.Loop.Current, rnd)
				s.EndTurn()
			case PhaseMatches:
				for md := s.NextMatchDay(); md != nil; md = s.NextMatchDay() {
					for md.Step() {
						if rnd.Intn(200) == 0 {
							mi := rnd.Intn(len(s.Managers))
							_ = md.Substitute(mi, rnd.Intn(SquadMax), rnd.Intn(SquadMax))
						}
					}
				}
			case PhaseContracts:
				for _, si := range s.ExpiringContracts() {
					if rnd.Intn(2) == 0 {
						_ = s.NegotiateContract(s.Loop.Current, si, 1+rnd.Intn(3), s.ContractHint(s.Loop.Current, si, 2), 0)
					}
				}
				s.ContractsDone()
			case PhaseGameOver:
				t.Fatal("endless game ended")
			}
			checkInvariants(t, s)
		}
		t.Logf("seed %d: %d seasons, leagues %d %d %d %d", seed, s.Seasons,
			s.Managers[0].League, s.Managers[1].League, s.Managers[2].League, s.Managers[3].League)
	}
}

func randomActions(s *State, mi int, rnd *rand.Rand) {
	for k := 0; k < rnd.Intn(4); k++ {
		switch rnd.Intn(9) {
		case 0:
			if n := s.SquadSize(MarketIndex); n > 0 {
				i := rnd.Intn(n)
				bid := s.MarketPrice(i, false) * int64(80+rnd.Intn(60)) / 100
				res, err := s.Bid(mi, i, bid, rnd.Intn(4) == 0)
				if err == nil && res == BidNeedsContract {
					si, err := s.BeginPurchase(mi, i, bid)
					if err == nil {
						err = s.NegotiateContract(mi, si, 2, s.ContractHint(mi, si, 2)*11/10, 0)
						_ = s.FinishPurchase(err == nil)
					}
				}
				if err == nil && res == BidAskOtherManager {
					res, err = s.AnswerManagerBid(mi, i, bid, false, rnd.Intn(2) == 0)
					if err == nil && res == BidNeedsContract {
						if si, err := s.BeginPurchase(mi, i, bid); err == nil {
							err = s.NegotiateContract(mi, si, 2, s.ContractHint(mi, si, 2)*11/10, 0)
							_ = s.FinishPurchase(err == nil)
						}
					}
				}
				s.AcceptSubsidy(mi, true)
			}
		case 1:
			if n := s.SquadSize(mi); n > 15 {
				_ = s.ListPlayer(mi, rnd.Intn(n))
			}
		case 2:
			k := rnd.Intn(NumStadiumItems)
			lim := s.StadiumLimit(mi, k)
			if !lim.Locked && !lim.NoMoney && lim.Max > 0 {
				amount := lim.Max
				if k <= StCovered {
					amount = (lim.Max / 2) / 1000 * 1000
				}
				if amount > 0 && (k <= StCovered || amount > lim.Now) {
					o := s.QuoteBuild(mi, k, amount)
					_ = s.ConfirmBuild(mi, o)
				}
			}
		case 3:
			_ = s.TakeBankCredit(mi, int64(rnd.Intn(5)+1)*200000, rnd.Intn(6))
		case 4:
			_ = s.BookCamp(mi, rnd.Intn(8))
		case 5:
			if n := s.SquadSize(mi); n > 0 {
				_ = s.ToggleLineup(mi, rnd.Intn(n))
				s.Managers[mi].AutoLineup = 2 + rnd.Intn(3)
				s.AutoLineup(mi)
			}
		case 6:
			for si := 0; si < s.SquadSize(mi); si++ {
				if o, ok := s.SquadOffer(mi, si); ok {
					s.AnswerSquadOffer(mi, si, o, rnd.Intn(2) == 0)
					break
				}
			}
		case 7:
			k := rnd.Intn(10)
			if s.Managers[mi].Offers[0][k] != 0 {
				_ = s.AcceptTrikot(mi, k)
			}
			_ = s.AcceptBoard(mi, rnd.Intn(6), rnd.Intn(10))
		case 8:
			s.SetTicketPrice(mi, 5+rnd.Intn(21))
		}
	}
}

func checkInvariants(t *testing.T, s *State) {
	t.Helper()
	seen := map[int]bool{}
	for l := 0; l < NumLeagues; l++ {
		for i := 0; i < TeamsPerLeague[l]; i++ {
			id := s.Tables[l][i]
			if seen[id] || LeagueOf(id) != l || id > 57 {
				t.Fatalf("table %d broken at %d: id %d", l, i, id)
			}
			seen[id] = true
		}
	}
	for mi, m := range s.Managers {
		if LeagueOf(m.Team) != m.League {
			t.Fatalf("manager %d: league %d but team %d", mi, m.League, m.Team)
		}
		for i := 0; i < SquadSlots; i++ {
			sl := &s.Squads[mi][i]
			if sl.Empty() {
				continue
			}
			if i > 0 && !s.Squads[mi][i-1].Empty() && s.Squads[mi][i-1].Person >= sl.Person {
				t.Fatalf("manager %d squad not sorted at %d", mi, i)
			}
		}
		if s.SquadSize(mi) > SquadMax {
			t.Fatalf("manager %d: squad %d", mi, s.SquadSize(mi))
		}
	}
	persons := map[int]int{}
	for h := 0; h <= MarketIndex; h++ {
		if h < MarketIndex && h >= len(s.Managers) {
			continue
		}
		for _, sl := range s.squad(h) {
			if !sl.Empty() {
				persons[sl.Person]++
				if persons[sl.Person] > 1 {
					t.Fatalf("person %d in two squads", sl.Person)
				}
			}
		}
	}
}
