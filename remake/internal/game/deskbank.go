package game

// Checks of the bank screen (f_112a_167f) before the duration and the rate
// of a credit are asked.

// CheckCreditAmount reports the refusal of a credit of amount from lender
// (a manager index or LenderBank) before more is asked: the lender lacks
// the money, or the limit of 1 Mio. DM per fellow manager or 4 Mio. DM at
// the bank would be exceeded.
func (s *State) CheckCreditAmount(mi, lender int, amount int64) error {
	m := s.Managers[mi]
	var sum int64
	for _, c := range m.Credits[lender] {
		sum += c.Amount
	}
	if lender == LenderBank {
		if sum+amount > 4000000 {
			return ErrBankLimit
		}
		return nil
	}
	if s.Managers[lender].Money < amount {
		return ErrLenderMoney
	}
	if sum+amount > 1000000 {
		return ErrLenderLimit
	}
	return nil
}

// CreditSum returns the amounts (or the monthly interest) of the credits
// of manager mi from lender (f_112a_0f8d).
func (s *State) CreditSum(mi, lender int, interest bool) int64 {
	var v int64
	for _, c := range s.Managers[mi].Credits[lender] {
		if c.Amount == 0 {
			continue
		}
		if interest {
			v += c.Interest
		} else {
			v += c.Amount
		}
	}
	return v
}
