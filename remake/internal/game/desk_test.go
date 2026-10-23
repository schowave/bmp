package game

import "testing"

func TestBestSeries(t *testing.T) {
	s := &State{}
	s.Matchday[0] = 3
	// club 0: home 3:0, away 1:2
	s.Results[0][0][0] = Score{3, 0}
	s.Results[0][1][6] = Score{2, 1}
	best := s.BestSeries(0)
	if best[SeriesWon][VenueTotal] != 1 || best[SeriesWon][VenueHome] != 1 || best[SeriesWon][VenueAway] != 0 ||
		best[SeriesCleanSheet][VenueHome] != 1 || best[SeriesLost][VenueAway] != 1 || best[SeriesNotLost][VenueTotal] != 1 {
		t.Fatalf("best %v", best)
	}
	if cur := s.TeamSeries(0); cur[SeriesWon][VenueTotal] != 0 {
		t.Fatalf("current %v", cur)
	}
}

func TestCheckCreditAmount(t *testing.T) {
	s := &State{Managers: []*Manager{{}, {Money: 500000}}}
	if err := s.CheckCreditAmount(0, LenderBank, 4000001); err != ErrBankLimit {
		t.Errorf("bank limit: %v", err)
	}
	if err := s.CheckCreditAmount(0, LenderBank, 4000000); err != nil {
		t.Errorf("bank: %v", err)
	}
	if err := s.CheckCreditAmount(0, 1, 600000); err != ErrLenderMoney {
		t.Errorf("lender money: %v", err)
	}
	s.Managers[1].Money = 2000000
	s.Managers[0].Credits[1][0] = Credit{Amount: 600000, Interest: 30000}
	if err := s.CheckCreditAmount(0, 1, 500000); err != ErrLenderLimit {
		t.Errorf("lender limit: %v", err)
	}
	if v := s.CreditSum(0, 1, true); v != 30000 {
		t.Errorf("interest %d", v)
	}
}
