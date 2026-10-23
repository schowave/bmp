package game

import "strconv"

// clublessStrength replaces the strength of the club for persons without a
// club (team 0xff): the original reads a byte behind the TEAM table there.
// The value is chosen so that the market looks like the one of the
// reference game (Kondition about 22..44, Technik 20..51).
const clublessStrength = 34

// refreshMarket replaces AI players on the transfer market by free persons
// (f_2277_1e38). Players listed by human managers stay.
func (s *State) refreshMarket() {
	r := &s.Rand
	mk := &s.Squads[MarketIndex]
	kept := 0
	for i := 0; i < MarketSlots; i++ {
		p := mk[i].Person
		if s.removeSlot(MarketIndex, i, false, MarketSlots) {
			if p != 0 {
				s.Persons[p].Owner = OwnerFree
			}
		} else {
			kept++
		}
	}
	n := MarketSlots - kept
	if n == 0 {
		return
	}
	n -= r.Between(0, n)
	for j := 0; j < n; j++ {
		k := r.Between(0, MarketSlots-1)
		p := mk[k].Person
		if p != 0 && s.Persons[p].Owner != OwnerMarket {
			continue
		}
		if p != 0 {
			s.removeSlot(MarketIndex, k, true, MarketSlots)
			s.Persons[p].Owner = OwnerFree
		}
		q := 0
		for try := 0; try < 1500; try++ {
			q = r.Between(1, 150)
			if s.Persons[q].Owner == OwnerFree {
				break
			}
		}
		ps := &s.Persons[q]
		ps.Skill[2] = r.Between(45, 55)
		t := ps.Team
		if t >= 0 && t < NumTeams && s.ManagerOf(t) != nil {
			l := 2
			for ll := 0; ll < NumLeagues; ll++ {
				if t >= FirstID[ll] && t < FirstID[ll]+TeamsPerLeague[ll] {
					l = ll
					break
				}
			}
			for {
				t = r.Between(leagueRange[l][0], leagueRange[l][1])
				if s.ManagerOf(t) == nil {
					break
				}
			}
			// Original quirk: the league search reuses the loop counter.
			j = l
		}
		cur, base := clublessStrength, clublessStrength
		if t >= 0 && t < NumTeams {
			cur, base = s.Teams[t].Strength[1], s.Teams[t].Base[1]
		}
		ps.Skill[0] = clamp(r.Between(0, 14)+cur-10, 10, 99)
		s2 := r.Between(0, 14) + base - 10
		if r.Between(0, 4) == 0 {
			s2 += r.Between(12, 20)
		}
		ps.Skill[1] = clamp(s2, 10, 99)
		slot, err := s.addPerson(MarketIndex, q, 0, 0, MarketSlots)
		if err != nil {
			return
		}
		ps.Owner = OwnerMarket
		ps.Team = t
		mk[slot].Salary = s.PlayerValue(MarketIndex, slot, ValueNoise)
	}
}

// MarketPrice returns the price of market slot i as shown in the list: the
// asking price of AI players, the value of players listed by managers.
// Loans cost a third.
func (s *State) MarketPrice(i int, loan bool) int64 {
	sl := &s.Squads[MarketIndex][i]
	var v int64
	if s.Persons[sl.Person].Owner == OwnerMarket {
		v = sl.Salary
	} else {
		v = s.PlayerValue(MarketIndex, i, 0)
	}
	if loan {
		v /= 3
	}
	return v
}

// aiAccepts decides whether the AI accepts a bid (f_2277_2171).
func (s *State) aiAccepts(i int, bid int64, loan bool) bool {
	r := &s.Rand
	v := s.Squads[MarketIndex][i].Salary
	if loan {
		v /= 3
	}
	if int64(r.Between(75, 85))*v/100 > bid {
		return false
	}
	if int64(r.Between(120, 130))*v/100 < bid {
		return true
	}
	// Only a bid of 0 for a player worth 0 gets here, where the original
	// divides by zero.
	if v == 0 {
		return true
	}
	return bid*100/v > int64(r.Between(80, 120))
}

// TransferError is a refusal shown as a message box.
type TransferError = CreditError

// ListPlayer puts squad slot si of manager mi on the transfer market.
func (s *State) ListPlayer(mi, si int) error {
	sl := &s.Squads[mi][si]
	if sl.Nego&0x80 != 0 {
		return TransferError("Dieser Spieler hört am Saisonende auf.")
	}
	n := 0
	for _, x := range s.squad(MarketIndex) {
		if !x.Empty() && s.Persons[x.Person].Owner == mi {
			n++
		}
	}
	if n >= 3 {
		return TransferError("Schon 3 Spieler auf dem Transfermarkt")
	}
	p := sl.Person
	if s.Persons[p].Owner != mi {
		return TransferError("Dieser Spieler ist nur ausgeliehen")
	}
	ms, err := s.addPerson(MarketIndex, p, 1, 0, MarketSlots)
	if err != nil {
		return err
	}
	sl.Lineup = 0
	s.Squads[MarketIndex][ms] = *sl
	s.removeSlot(mi, si, true, SquadMax)
	s.AutoLineup(mi)
	return nil
}

// Offer is a sale offer of an AI club for a player of a manager.
type Offer struct {
	Club     int
	Amount   int64 // "Angebot"
	Reduce   int64 // "Ablöse": deduction for the remaining contract
	Credited int64
}

// SquadOffer returns the offer of the AI club interested in squad slot si
// (f_2277_04a5 / f_2277_1b5f). ok is false without an offer.
func (s *State) SquadOffer(mi, si int) (Offer, bool) {
	sl := &s.Squads[mi][si]
	if sl.Status&0x40 == 0 {
		return Offer{}, false
	}
	v := s.PlayerValue(mi, si, 0)
	o := int64(s.Rand.Between(85, 150)) * v / 10000 * 100
	d := o / 7 * int64(sl.Years)
	return Offer{Club: sl.Buyer, Amount: o, Reduce: d, Credited: o - d}, true
}

// AnswerSquadOffer accepts (sell) or rejects an offer from SquadOffer.
func (s *State) AnswerSquadOffer(mi, si int, o Offer, sell bool) {
	sl := &s.Squads[mi][si]
	sl.Status &^= 0x40
	s.endMessage(sl)
	if !sell {
		return
	}
	p := sl.Person
	s.Managers[mi].Money += o.Credited
	avg := sl.Strength()
	s.removeSlot(mi, si, true, SquadMax)
	s.Persons[p].Team = o.Club
	s.strengthenBuyer(o.Club, avg)
	s.Persons[p].Owner = OwnerFree
	s.AutoLineup(mi)
}

// ListedOffer returns the AI offer for a player manager mi listed on the
// market (slot i).
func (s *State) ListedOffer(mi, i int) (Offer, bool) {
	sl := &s.Squads[MarketIndex][i]
	if s.Persons[sl.Person].Owner != mi || sl.Status&0x80 == 0 {
		return Offer{}, false
	}
	v := s.PlayerValue(MarketIndex, i, 0)
	o := int64(s.Rand.Between(67, 110)) * v / 10000 * 100
	d := o / 7 * int64(sl.Years)
	return Offer{Club: sl.Buyer, Amount: o, Reduce: d, Credited: o - d}, true
}

// AnswerListedOffer accepts or rejects an offer from ListedOffer.
func (s *State) AnswerListedOffer(mi, i int, o Offer, sell bool) {
	sl := &s.Squads[MarketIndex][i]
	sl.Status &^= 0x80
	s.endMessage(sl)
	if !sell {
		return
	}
	p := sl.Person
	s.Persons[p].Team = sl.Buyer
	s.strengthenBuyer(sl.Buyer, sl.Strength())
	s.Managers[mi].Money += o.Credited
	s.removeSlot(MarketIndex, i, true, MarketSlots)
	s.Persons[p].Owner = OwnerFree
}

// strengthenBuyer raises the strength of the club that bought a player
// (f_14a4_23da, with the intended sum instead of the uninitialised index).
func (s *State) strengthenBuyer(t, avg int) {
	if t < 0 || t >= NumTeams {
		return
	}
	r := &s.Rand
	tm := &s.Teams[t]
	sum := 0
	for k := 0; k < 3; k++ {
		sum += tm.Strength[k] + tm.Base[k]
	}
	d := clamp((avg-sum/6+10)/3, 1, 10)
	k := r.Between(0, 2)
	tm.Strength[k] = (tm.Strength[k] + r.Between(d-1, d+1)) & 0xff
	tm.Base[k] = (tm.Base[k] + r.Between(d-1, d+1)) & 0xff
}

// BuyResult tells the UI what happens next after a bid.
type BuyResult int

// Results of Bid.
const (
	BidRejected BuyResult = iota
	BidNeedsContract
	BidDone
	BidAskOtherManager // the owning human has to accept the bid
)

// Bid is the bid of manager mi for market slot i. It returns whether the
// seller accepted and, for purchases, that a contract has to be
// negotiated with NegotiateContract before CompletePurchase.
func (s *State) Bid(mi, i int, bid int64, loan bool) (BuyResult, error) {
	ms := &s.Squads[MarketIndex][i]
	p := ms.Person
	owner := s.Persons[p].Owner
	if ms.Rejected&0x80 != 0 && ms.Rejected&(1<<mi) != 0 {
		return BidRejected, TransferError("Angebot wurde bereits abgelehnt")
	}
	if owner == mi {
		return BidDone, s.takeBack(mi, i)
	}
	if s.Managers[mi].Money < bid {
		return BidRejected, TransferError("Sie haben nicht genug Geld")
	}
	if owner != OwnerMarket {
		v := s.PlayerValue(MarketIndex, i, 0)
		if loan {
			v /= 3
		}
		if bid < v*60/100 || bid > v*140/100 {
			return BidRejected, TransferError("DIESER BETRAG LIEGT AUSSERHALB DES ERLAUBTEN !")
		}
		return BidAskOtherManager, nil
	}
	if !s.aiAccepts(i, bid, loan) {
		ms.Rejected |= 1<<mi | 0x80
		return BidRejected, TransferError("Ihr Angebot wurde abgelehnt")
	}
	if loan {
		return BidDone, s.completeTransfer(mi, i, bid, true, false)
	}
	return BidNeedsContract, nil
}

// takeBack takes an own listed player back into the squad.
func (s *State) takeBack(mi, i int) error {
	ms := s.Squads[MarketIndex][i]
	ns, err := s.addPerson(mi, ms.Person, 1, 0, SquadMax+1)
	if err != nil {
		return err
	}
	s.Squads[mi][ns] = ms
	s.removeSlot(MarketIndex, i, true, MarketSlots)
	s.AutoLineup(mi)
	return nil
}

// AnswerManagerBid is the answer of the owning human manager to a bid
// (BidAskOtherManager). A purchase (not a loan) still needs a contract:
// then it returns BidNeedsContract and the purchase continues with
// BeginPurchase like a purchase from the AI.
func (s *State) AnswerManagerBid(mi, i int, bid int64, loan, accept bool) (BuyResult, error) {
	ms := &s.Squads[MarketIndex][i]
	if !accept {
		ms.Rejected |= 1<<mi | 0x80
		return BidRejected, TransferError("Ihr Angebot wurde abgelehnt")
	}
	// The person takes over the current skills of the listed player.
	s.Persons[ms.Person].Skill = ms.Skill
	if loan {
		return BidDone, s.completeTransfer(mi, i, bid, true, true)
	}
	return BidNeedsContract, nil
}

// subsidy offers a sponsor subsidy after a deal with the AI
// (f_0000_272d), also one of 0 DM for a cheap player.
func (s *State) subsidy(bid int64) {
	if s.Rand.Between(0, 6) == 0 {
		s.SubsidyOffered = true
		s.PendingSubsidy = (bid / 100) * int64(s.Rand.Between(20, 65)) / 10000 * 10000
	}
}

// PendingPurchase is a Purchase from the AI waiting for the contract.
type PendingPurchase struct {
	Active  bool
	Manager int
	Market  int
	Bid     int64
	Person  int
	OldTeam int
	Slot    int
	H2H     bool // bought from another human manager
}

// BeginPurchase puts the player of market slot i into the squad of manager
// mi so that his contract can be negotiated (NegotiateContract on the
// returned slot), then FinishPurchase completes or cancels the transfer.
func (s *State) BeginPurchase(mi, i int, bid int64) (int, error) {
	p := s.Squads[MarketIndex][i].Person
	old := s.Persons[p].Team
	h2h := s.Persons[p].Owner != OwnerMarket && s.Persons[p].Owner != mi
	ns, err := s.addPerson(mi, p, 1, 0, SquadMax)
	if err != nil {
		return -1, err
	}
	s.Purchase = PendingPurchase{true, mi, i, bid, p, old, ns, h2h}
	return ns, nil
}

// FinishPurchase completes the Purchase begun with BeginPurchase when the
// contract was agreed, or undoes it.
func (s *State) FinishPurchase(agreed bool) error {
	pp := s.Purchase
	s.Purchase = PendingPurchase{}
	if !pp.Active {
		return nil
	}
	mi := pp.Manager
	if !agreed {
		s.removeSlot(mi, pp.Slot, true, SquadMax)
		// Like the original only the club comes back: the league goals and
		// games addPerson cleared stay at zero.
		s.Persons[pp.Person].Team = pp.OldTeam
		if !pp.H2H {
			s.Squads[MarketIndex][pp.Market].Rejected |= 1<<mi | 0x80
		}
		return TransferError("Der Transfer findet nicht statt")
	}
	ps := &s.Persons[pp.Person]
	if pp.H2H {
		ms := &s.Squads[MarketIndex][pp.Market]
		sl := &s.Squads[mi][pp.Slot]
		sl.Status, sl.Injury, sl.Out, sl.Cards = ms.Status, ms.Injury, ms.Out, ms.Cards
		s.Managers[ps.Owner].Money += pp.Bid
		ps.Team = s.Managers[mi].Team
	}
	s.removeSlot(MarketIndex, pp.Market, true, MarketSlots)
	ps.Owner = mi
	s.Managers[mi].Money -= pp.Bid
	if !pp.H2H {
		s.subsidy(pp.Bid)
	}
	s.AutoLineup(mi)
	return nil
}

// completeTransfer moves market slot i into the squad of manager mi.
func (s *State) completeTransfer(mi, i int, bid int64, loan, h2h bool) error {
	ms := s.Squads[MarketIndex][i]
	p := ms.Person
	ps := &s.Persons[p]
	oldTeam := ps.Team
	seller := ps.Owner
	loanVal := 0
	if loan {
		loanVal = 0x63
	}
	if h2h {
		ps.Skill = ms.Skill
	}
	ns, err := s.addPerson(mi, p, 1, loanVal, SquadMax)
	if err != nil {
		return err
	}
	saved := s.Squads[mi][ns].Salary
	sl := &s.Squads[mi][ns]
	if h2h {
		sl.Status, sl.Injury, sl.Out = ms.Status, ms.Injury, ms.Out
		sl.Cards = ms.Cards
	}
	if loan {
		*sl = ms
		sl.Years = 1
		sl.Loan = 0x80 + oldTeam
		sl.Salary = saved
		sl.Goals = [3]int{}
		sl.Lineup = 0
	}
	s.removeSlot(MarketIndex, i, true, MarketSlots)
	if h2h {
		s.Managers[seller].Money += bid
		ps.Team = s.Managers[mi].Team
	}
	if !loan {
		ps.Owner = mi
	}
	s.Managers[mi].Money -= bid
	if !h2h {
		s.subsidy(bid)
	}
	s.AutoLineup(mi)
	return nil
}

// AcceptSubsidy answers "NEHMEN SIE EINEN SPONSOR-ZUSCHUSS VON ... AN ?".
func (s *State) AcceptSubsidy(mi int, yes bool) {
	if yes {
		s.Managers[mi].Money += s.PendingSubsidy
	}
	s.SubsidyOffered, s.PendingSubsidy = false, 0
}

// ContractHint is the salary the player would like for y years
// (f_2277_2a8f), shown before negotiating.
func (s *State) ContractHint(m, si, y int) int64 {
	sl := &s.Squads[m][si]
	v := s.PlayerValue(m, si, 5)
	yp := s.contractYp(m, y)
	k := int64((yp/6 + 122) * 8 / 10)
	hint := v * (k + 4) / 100
	if s.Day > 321 {
		hint = hint * int64(152-s.Persons[sl.Person].Age) / 100
	}
	if hint < sl.Salary {
		hint = sl.Salary
	}
	return hint
}

func (s *State) contractYp(m, y int) int {
	lg := 0
	if m < len(s.Managers) {
		lg = s.Managers[m].League
	}
	return (y-1)*100 + (-100*s.Matchday[lg])/MatchdaysPerLeague[lg] + 100
}

// playerAccepts decides whether a player accepts a contract (f_2277_2270).
func (s *State) playerAccepts(m, si, y int, sal int64) bool {
	sl := &s.Squads[m][si]
	cy, cs := sl.Years, sl.Salary
	if y == cy && sal > cs {
		return true
	}
	if y == cy && sal < cs {
		return false
	}
	if cy > y && cs*120/100 < sal {
		return true
	}
	if cy > y && cs*105/100 >= sal {
		return false
	}
	yp := s.contractYp(m, y)
	v := s.PlayerValue(m, si, 5)
	k := (yp/6 + 110) * 8 / 10
	d := v * int64(s.Rand.Between(k-6, k+4)) / 100
	if s.Day > 321 {
		age := s.Persons[sl.Person].Age
		d = d * int64(s.Rand.Between(132-age, 152-age)) / 100
	}
	return d < 100000 && sal > d
}

// NegotiateCheck returns why the talks with squad slot si cannot start,
// and the years the player asks for himself (0 = free choice).
func (s *State) NegotiateCheck(mi, si int) (forced int, err error) {
	sl := &s.Squads[mi][si]
	name := s.Persons[sl.Person].Name
	switch {
	case sl.Loan != 0:
		return 0, TransferError(name + " ist leider von " + s.Teams[sl.Loan&0x7f].Name + " nur ausgeliehen.")
	case sl.Nego&0x80 != 0:
		return 0, TransferError(name + " wird sich mit Ablauf des Vertrages zur Ruhe setzen.")
	case sl.Nego >= 100:
		return sl.Nego - 100, nil
	case sl.Nego > 0:
		return 0, TransferError(name + " ist im Moment nicht verhandlungsbereit.")
	}
	return 0, nil
}

// NegotiateContract offers squad slot si of squad m a contract of y years
// for salary sal per month. forced are the years the player asked for.
func (s *State) NegotiateContract(m, si, y int, sal int64, forced int) error {
	sl := &s.Squads[m][si]
	name := s.Persons[sl.Person].Name
	defer func() {
		if sl.Nego > 99 {
			s.endMessage(sl)
		}
		sl.Nego = s.Rand.Between(10, 18)
	}()
	if y > 4 {
		return TransferError("Wer wird sich denn SO lange verpflichten ? (Mal abgesehen von Bundi's)")
	}
	if y == 0 {
		y = sl.Years
		if forced != 0 {
			y = forced
		}
	}
	if sal == 0 {
		sal = sl.Salary
	}
	if y > sl.Years && sal < sl.Salary {
		return TransferError("So dumm ist " + name + " leider nicht...")
	}
	if forced != 0 {
		if sal < sl.Salary || y != forced {
			return TransferError(name + " ist nicht an Ihrem Angebot interessiert.")
		}
	} else if !s.playerAccepts(m, si, y, sal) {
		return TransferError(name + " ist nicht an Ihrem Angebot interessiert.")
	}
	sl.Years, sl.Salary = y, sal
	return nil
}

// AgeLabel returns the age text of the player info screen.
func AgeLabel(age int) string {
	i := (age - 18) / 5
	if i < 0 || i > 3 {
		i = 3
		if age < 18 {
			i = 0
		}
	}
	return [4]string{"(GRÜNSCHNABEL)", "(NOCH GANZ FRISCH)", "(MITTELALTER)", "(FAST AM ENDE)"}[i]
}

// SideText returns the preferred side like f_0310_1c22: "(L)", "(L+R)",
// "(R)".
func SideText(side int) string {
	t := " ("
	if side < 5 {
		t += "L"
	}
	if side > 1 {
		if side < 5 {
			t += "+"
		}
		t += "R"
	}
	return t + ")"
}

func itoa(n int) string { return strconv.Itoa(n) }
