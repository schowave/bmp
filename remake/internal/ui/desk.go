package ui

import (
	"sort"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/schowave/bmp/remake/internal/game"
)

// simpleScreen draws with a function and closes on HAUPTMENÜ.
type simpleScreen struct {
	draw   func(app *App, dst *ebiten.Image)
	update func(app *App) bool // returns true when it handled the input
}

func (s *simpleScreen) Update(app *App) error {
	if s.update != nil && s.update(app) {
		return nil
	}
	if MenuButtonHit(&app.Input) {
		app.Pop()
	}
	return nil
}

func (s *simpleScreen) Draw(app *App, dst *ebiten.Image) {
	app.Gfx.Background(dst)
	s.draw(app, dst)
	app.Gfx.DrawMenuButton(dst)
}

// NewFinanceScreen shows attendance and finance statistics (f_262f_0af7).
func NewFinanceScreen(ss *Session) Scene {
	return &simpleScreen{draw: func(app *App, dst *ebiten.Image) {
		g := app.Gfx
		s := ss.S
		mi := ss.Mgr()
		m := ss.M()
		g.Panel(dst, 4, 4, 196, 186)
		g.Heading(dst, 10, 8, "ZUSCHAUER")
		y := 22
		row := func(label, value string) {
			g.Text(dst, FontSmall, 10, y, label, g.C(ColLight))
			g.TextRight(dst, FontSmall, 194, y, value, g.C(ColWhite))
			y += 9
		}
		if m.HomeGames == 0 {
			row("BESUCHERZAHL", "(NOCH KEINE...)")
			row("SCHNITT/SPIEL", "(NOCH KEINER...)")
		} else {
			row("BESUCHERZAHL", formatMoney(m.SeasonAudience))
			row("SCHNITT/SPIEL", formatMoney(m.SeasonAudience/int64(m.HomeGames)))
		}
		row("BENÖTIGTER SCHNITT", formatMoney(s.RequiredAverage(mi)))
		if m.RecordAudience > 0 {
			row("ZUSCHAUERREKORD", formatMoney(m.RecordAudience)+" ("+s.Teams[m.RecordOpponent].Name+")")
		}
		if m.MinAudience != 99999 {
			row("MINUSKULISSE", formatMoney(m.MinAudience)+" ("+s.Teams[m.MinOpponent].Name+")")
		}
		y += 6
		g.Heading(dst, 10, y, "FINANZEN")
		y += 14
		row("KONTOSTAND", Money(m.Money))
		row("SCHULDEN GESAMT", Money(m.Debts()))
		row("ZINSEN/MONAT", Money(m.InterestPerMonth()))
		y += 6
		g.Heading(dst, 10, y, "EINNAHMEN+AUSGABEN / MONAT")
		y += 14
		row("EINNAHMEN (SPENDEN, SPONSOREN)", Money(s.MonthlyIncome(mi)))
		row("AUSGABEN (GEHÄLTER, REISEKOSTEN)", Money(s.MonthlyExpenses(mi)))
		// attendance graph
		g.Panel(dst, 204, 4, 112, 100)
		g.Text(dst, FontSmall, 208, 7, "ZUSCHAUER/1000", g.C(ColBeige))
		maxA := 1
		for i := 0; i < m.HomeGames && i < len(m.Attendance); i++ {
			maxA = max(maxA, m.Attendance[i])
		}
		for i := 0; i < m.HomeGames && i < len(m.Attendance); i++ {
			h := m.Attendance[i] * 80 / maxA
			FillRect(dst, 208+i*5, 98-h, 4, h, g.C(ColRed))
		}
		g.Panel(dst, 204, 108, 112, 82)
		g.Text(dst, FontSmall, 208, 111, "KONTO", g.C(ColBeige))
		bar := int(clamp64(m.Money/50000+50, 0, 100))
		for x := 0; x < 100; x++ {
			c := ColRed
			if x > 33 {
				c = ColOrange
			}
			if x > 66 {
				c = 14
			}
			FillRect(dst, 210+x, 130, 1, 12, g.C(c))
		}
		FillRect(dst, 210+bar, 126, 1, 20, g.C(ColWhite))
	}}
}

func clamp64(v, lo, hi int64) int64 {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}

// NewEternalScreen shows the eternal table and the eternal balance of the
// manager (f_262f_18e7).
func NewEternalScreen(ss *Session) Scene {
	s := ss.S
	ids := make([]int, game.NumGerman)
	for i := range ids {
		ids[i] = i
	}
	sort.SliceStable(ids, func(a, b int) bool { return s.Clubs[ids[a]].Eternal > s.Clubs[ids[b]].Eternal })
	cols := []Column{{"", 2, 14, Right}, {"", 20, 100, Left}, {"", 122, 24, Right}}
	t := NewTableView(4, 4, 150, 188, cols)
	t.NoHeader = true
	for i, id := range ids {
		row := Row{Cells: []string{itoa(i+1) + ".", s.Teams[id].Name, itoa64(s.Clubs[id].Eternal % 100000)}}
		if s.ManagerOf(id) != nil {
			row.Color = ColYellow
		}
		t.Rows = append(t.Rows, row)
	}
	return &simpleScreen{
		update: func(app *App) bool { t.Update(&app.Input); return false },
		draw: func(app *App, dst *ebiten.Image) {
			g := app.Gfx
			m := ss.M()
			t.Draw(g, dst)
			g.Panel(dst, 158, 4, 158, 188)
			g.TextCentered(dst, FontNormal, 158, 7, 158, "Ewige Bilanz", g.C(ColBeige))
			titles := []string{"MEISTERSCHAFTEN", "DFB-POKALE", "LANDESMEISTER-POKALE", "POKALSIEGER-POKALE", "UEFA-POKALE"}
			for i, tt := range titles {
				g.Text(dst, FontSmall, 164, 22+i*12, tt, g.C(ColLight))
				v := itoa(m.Titles[i])
				if m.Titles[i] == 0 {
					v = "KOMMT VIELLEICHT NOCH..."
				}
				g.TextRight(dst, FontSmall, 312, 28+i*12, v, g.C(ColWhite))
			}
			y := 92
			g.Text(dst, FontSmall, 164, y, "        GESAMT   HEIM   AUSW.", g.C(ColBeige))
			line := func(label string, v [2]int, minus *[2]int) {
				y += 9
				tot := v[0] + v[1]
				f := func(a int, i int) string {
					if minus != nil {
						return itoa(a) + ":" + itoa(i)
					}
					return itoa(a)
				}
				var mt, m0, m1 int
				if minus != nil {
					mt, m0, m1 = minus[0]+minus[1], minus[0], minus[1]
				}
				g.Text(dst, FontSmall, 164, y, label, g.C(ColLight))
				g.TextRight(dst, FontSmall, 236, y, f(tot, mt), g.C(ColWhite))
				g.TextRight(dst, FontSmall, 270, y, f(v[0], m0), g.C(ColWhite))
				g.TextRight(dst, FontSmall, 310, y, f(v[1], m1), g.C(ColWhite))
			}
			line("PUN.", m.EtPoints, &m.EtMinus)
			line("TORE", m.EtGoals, &m.EtAgainst)
			line("SIEGE", m.EtWins, nil)
			line("NIED.", m.EtLosses, nil)
			line("UNEN.", m.EtDraws, nil)
		}}
}

// NewHistoryScreen shows the manager's seasons (f_262f_390e).
func NewHistoryScreen(ss *Session) Scene {
	s := ss.S
	m := ss.M()
	cols := []Column{{"SAIS.", 2, 30, Left}, {"PL.", 34, 16, Right}, {"LIGA", 54, 70, Left}, {"DFB-POKAL", 126, 70, Left}, {"EUROPAPOKAL", 198, 112, Left}}
	t := NewTableView(4, 24, 312, 168, cols)
	t.RowH = 8
	first := s.StartYear - s.Seasons
	for i, h := range m.History {
		year := first + m.DroppedSeasons + i
		pos := int(h[0]) - game.FirstID[h[1]]
		dfb := ""
		switch {
		case h[2]&0x80 != 0:
			dfb = "SIEGER"
		case h[2]&7 > 0 && h[2]&7 <= 5:
			dfb = game.RoundNames[h[2]&7]
		}
		eu := "nicht im wettbewerb"
		if h[3]&0xf8 != 0 {
			eu = game.CupNames[(h[3]>>3)&3]
			if h[3]&0x80 != 0 {
				eu += " SIEGER"
			} else if r := h[3] & 7; r <= 5 {
				eu += " " + game.RoundNames[r]
			}
		}
		row := Row{Cells: []string{itoa(year%100) + "/" + pad2((year+1)%100), itoa(pos) + ".", game.LeagueNames[h[1]], dfb, eu}}
		if pos == 1 && h[1] == 0 {
			row.Color = ColYellow
		}
		t.Rows = append(t.Rows, row)
	}
	if len(t.Rows) == 0 {
		t.Rows = append(t.Rows, Row{Cells: []string{"", "", "NOCH HABEN SIE KEINE HISTORISCH SONDERLICH WICHTIGEN ERFOLGE ERZIELT..."}})
	}
	return &ListScreen{Title: "Gesamtentwicklung - " + m.Name, Table: t}
}

// StadiumScreen shows and extends the stadium (f_0000_0602).
type StadiumScreen struct {
	ss    *Session
	hover int
}

// NewStadiumScreen creates the stadium screen.
func NewStadiumScreen(ss *Session) *StadiumScreen { return &StadiumScreen{ss: ss, hover: -1} }

var stadiumRows = []string{"GESAMTKAPAZITÄT", "SITZPLÄTZE", "STEHPLÄTZE", "ÜBERDACHTE PLÄTZE", "FLUTLICHT",
	"ANZEIGETAFEL", "ZUSTAND", "KOMFORT", "EINTRITTSPREIS"}

func (st *StadiumScreen) value(k int, after bool) string {
	m := st.ss.M()
	v := func(i int) int64 {
		if after {
			return m.Stadium[i] + m.StadiumBuild[i]
		}
		return m.Stadium[i]
	}
	switch k {
	case 0:
		return formatMoney(v(game.StSeats) + v(game.StStanding))
	case 1, 2, 3:
		return formatMoney(v(k - 1))
	case 4, 5:
		return game.SizeNames[clampInt(int(v(k-1)), 0, 3)]
	case 6, 7:
		return game.GradeNames[clampInt(int(v(k-1)), 0, 6)]
	}
	return itoa(m.TicketPrice) + " DM"
}

func clampInt(v, lo, hi int) int { return max(lo, min(hi, v)) }

func (st *StadiumScreen) Update(app *App) error {
	in := &app.Input
	st.hover = -1
	for k := 1; k < len(stadiumRows); k++ {
		y := 108 + k*9
		if in.In(8, y, 300, 9) {
			st.hover = k
			if in.Left {
				st.order(app, k-1)
				return nil
			}
		}
	}
	if MenuButtonHit(in) {
		app.Pop()
	}
	return nil
}

// order runs the construction dialog for item k (0..6) or the ticket price
// (7).
func (st *StadiumScreen) order(app *App, k int) {
	s := st.ss.S
	mi := st.ss.Mgr()
	name := st.ss.M().Name + ":"
	if k == 7 {
		app.Push(NewNumberInput(game.BuildTitles[7], "EINTRITTSPREIS (5 BIS 25 DM)", int64(st.ss.M().TicketPrice),
			func(app *App, v int64, ok bool) {
				if ok {
					s.SetTicketPrice(mi, int(v))
				}
			}))
		return
	}
	lim := s.StadiumLimit(mi, k)
	switch {
	case lim.Locked:
		app.Push(NewMessage(name, "Im Moment keine Baufirma aufzutreiben."))
		return
	case lim.NoMoney && k <= game.StCovered:
		app.Push(NewMessage(name, "Kein Geld für neue Plätze da..."))
		return
	case lim.NoMoney:
		app.Push(NewMessage(name, "Da müssen Sie erst noch sparen..."))
		return
	case k > game.StCovered && lim.Now >= lim.Top:
		app.Push(NewMessage(name, "Was denn...? NOCH besser ???"))
		return
	}
	confirm := func(app *App, amount int64) {
		o := s.QuoteBuild(mi, k, amount)
		what := "AUSBAU UM " + formatMoney(amount) + " PLÄTZE"
		if k > game.StCovered {
			names := game.SizeNames[:]
			if k >= game.StCondition {
				names = game.GradeNames[:]
			}
			what = "NEUER STATUS: " + names[clampInt(int(amount), 0, len(names)-1)]
		}
		weeks := o.Days/7 + 1
		bau := "BAUZEIT: CIRKA "
		if o.Again {
			bau = "BAUZEIT: INSGESAMT CA. "
		}
		app.Push(NewQuestion(game.BuildTitles[k], what+", KOSTEN: "+Money(o.Cost)+". "+bau+itoa(weeks)+" WOCHEN", "NA KLAR !", "ACH NEE...",
			func(app *App, yes bool) {
				if !yes {
					s.DeclineBuild(k)
					return
				}
				if err := s.ConfirmBuild(mi, o); err != nil {
					app.Push(NewMessage(name, err.Error()))
				}
			}))
	}
	if k <= game.StCovered {
		app.Push(NewNumberInput(game.BuildTitles[k], "KOSTET "+formatMoney(game.BuildCost[k])+" DM PRO 1000, MAX. "+formatMoney(lim.Max),
			lim.Max, func(app *App, v int64, ok bool) {
				v = v / 1000 * 1000
				if ok && v > 0 && v <= lim.Max {
					confirm(app, v)
				}
			}))
		return
	}
	app.Push(NewNumberInput(game.BuildTitles[k], "NEUE STUFE ("+itoa(int(lim.Now)+1)+" BIS "+itoa(int(lim.Max))+"), "+
		formatMoney(game.BuildCost[k])+" DM PRO PUNKT", lim.Max, func(app *App, v int64, ok bool) {
		if ok && v > lim.Now && v <= lim.Max {
			confirm(app, v)
		}
	}))
}

func (st *StadiumScreen) Draw(app *App, dst *ebiten.Image) {
	g := app.Gfx
	m := st.ss.M()
	g.Background(dst)
	pic := game.StadiumPicture(m)
	g.Frame(dst, 7, 15, 139, 82)
	Draw(dst, g.Pic(pic, false), 7, 15)
	g.Frame(dst, 152, 15, 138, 82)
	g.Panel(dst, 152, 15, 138, 82)
	y := 20
	for k := 0; k < game.NumStadiumItems; k++ {
		if m.BuildTimer[k] > 0 {
			g.Text(dst, FontSmall, 156, y, game.BuildTitles[k], g.C(ColLight))
			g.TextRight(dst, FontSmall, 286, y, itoa(m.BuildTimer[k]/7+1)+" WOCHE(N)", g.C(ColYellow))
			y += 8
		}
	}
	g.Text(dst, FontSmall, 156, 88, "IHR KONTOSTAND: "+Money(m.Money), g.C(ColWhite))
	g.Panel(dst, 4, 104, 312, 92)
	g.Text(dst, FontSmall, 120, 107, "MOMENTAN", g.C(ColBeige))
	g.Text(dst, FontSmall, 220, 107, "NACH AUSBAU", g.C(ColBeige))
	for k, label := range stadiumRows {
		yy := 108 + k*9
		col := ColLight
		if k == st.hover {
			col = ColYellow
		}
		if k > 0 {
			yy += 0
		}
		g.Text(dst, FontSmall, 8, yy+8, label, g.C(col))
		g.Text(dst, FontSmall, 120, yy+8, st.value(k, false), g.C(ColWhite))
		g.Text(dst, FontSmall, 220, yy+8, st.value(k, true), g.C(ColGrey3))
	}
	g.Text(dst, FontSmall, 8, 200, "Zeile anklicken: ausbauen bzw. Preis ändern", g.C(ColLight))
	g.DrawMenuButton(dst)
}

// BankScreen takes credits (f_112a_167f).
type BankScreen struct{ ss *Session }

// NewBankScreen creates the bank screen.
func NewBankScreen(ss *Session) *BankScreen { return &BankScreen{ss: ss} }

func (b *BankScreen) Update(app *App) error {
	in := &app.Input
	s := b.ss.S
	mi := b.ss.Mgr()
	name := b.ss.M().Name + ":"
	for row := 0; row < 6; row++ {
		if in.Clicked(152, 26+row*11, 96, 11) {
			app.Push(NewNumberInput("Kredit", "Wie hoch soll der Kredit sein ?", 0, func(app *App, v int64, ok bool) {
				if !ok || v <= 0 {
					return
				}
				if err := s.TakeBankCredit(mi, v, row); err != nil {
					app.Push(NewMessage(name, err.Error()))
					return
				}
				app.Push(NewMessage(name, "Kredit von "+Money(v)+" zu "+itoa(s.BankRates[row])+"% aufgenommen."))
			}))
			return nil
		}
	}
	for li := range s.Managers {
		if li == mi {
			continue
		}
		if in.Clicked(8, 200+li*9, 140, 9) {
			lender := li
			app.Push(NewNumberInput("Kredit von "+s.Managers[li].Name, "Wie hoch soll der Kredit sein ?", 0, func(app *App, v int64, ok bool) {
				if !ok || v <= 0 {
					return
				}
				app.Push(NewNumberInput("Kredit", "Laufzeit des Kredites (Mon.) ?", 12, func(app *App, months int64, ok bool) {
					if !ok || months <= 0 {
						return
					}
					app.Push(NewNumberInput("Kredit", "Prozentsatz des Kredites ?", 5, func(app *App, rate int64, ok bool) {
						if !ok {
							return
						}
						if err := s.TakeManagerCredit(mi, lender, v, int(months), int(rate)); err != nil {
							app.Push(NewMessage(name, err.Error()))
						}
					}))
				}))
			}))
			return nil
		}
	}
	if MenuButtonHit(in) {
		app.Pop()
	}
	return nil
}

func (b *BankScreen) Draw(app *App, dst *ebiten.Image) {
	g := app.Gfx
	s := b.ss.S
	m := b.ss.M()
	g.Background(dst)
	Draw(dst, g.Sub(7, rectWH(0, 0, 139, 82)), 7, 15)
	g.Panel(dst, 150, 10, 100, 92)
	g.Text(dst, FontNormal, 156, 14, "Dauer Zinssatz", g.C(ColBeige))
	for row := 0; row < 6; row++ {
		y := 27 + row*11
		col := ColLight
		if app.Input.In(152, 26+row*11, 96, 11) {
			col = ColYellow
		}
		g.TextRight(dst, FontNormal, 186, y, itoa((row+1)*3)+" Mon.", g.C(col))
		g.TextRight(dst, FontNormal, 240, y, itoa(s.BankRates[row])+"%", g.C(col))
	}
	g.Panel(dst, 150, 106, 166, 84)
	g.TextCentered(dst, FontNormal, 150, 110, 166, "Kontostand:", g.C(ColBeige))
	g.TextCentered(dst, FontNormal, 150, 121, 166, Money(m.Money), g.C(ColWhite))
	g.TextCentered(dst, FontNormal, 150, 136, 166, "Gesamtschulden:", g.C(ColBeige))
	g.TextCentered(dst, FontNormal, 150, 147, 166, Money(m.Debts()), g.C(ColWhite))
	g.TextCentered(dst, FontNormal, 150, 162, 166, "Zinsen/Monat:", g.C(ColBeige))
	g.TextCentered(dst, FontNormal, 150, 173, 166, Money(m.InterestPerMonth()), g.C(ColWhite))
	g.Panel(dst, 4, 104, 142, 86)
	y := 107
	g.Text(dst, FontSmall, 8, y, "IHRE KREDITE:", g.C(ColBeige))
	for k := range m.Credits {
		for _, c := range m.Credits[k] {
			if c.Amount == 0 {
				continue
			}
			y += 8
			g.Text(dst, FontSmall, 8, y, formatMoney(c.Amount)+" ("+itoa(c.Rate)+"%) FÄLLIG "+itoa(c.DueDay)+"."+itoa(c.DueMonth+1)+"."+itoa(c.DueYear), g.C(ColLight))
		}
	}
	g.Text(dst, FontSmall, 8, 192, "Zeile der Tabelle anklicken: Kredit von der Bank", g.C(ColLight))
	for li, o := range s.Managers {
		if li != b.ss.Mgr() {
			g.Text(dst, FontSmall, 8, 200+li*9, "Kredit von "+o.Name, g.C(ColYellow))
		}
	}
	g.DrawMenuButton(dst)
}

// SponsorScreen shows advertising income and offers (f_262f_2be8, PIC 46).
type SponsorScreen struct {
	ss    *Session
	board int // 0 = Trikot, 1..6 = board
}

// NewSponsorScreen creates the advertising screen.
func NewSponsorScreen(ss *Session) *SponsorScreen { return &SponsorScreen{ss: ss} }

func (sp *SponsorScreen) Update(app *App) error {
	in := &app.Input
	s := sp.ss.S
	mi := sp.ss.Mgr()
	m := sp.ss.M()
	name := m.Name + ":"
	for b := 0; b < 7; b++ {
		if in.Clicked(4+b*44, 4, 42, 12) {
			sp.board = b
		}
	}
	t := 0
	if sp.board > 0 {
		t = 1
	}
	for k := 0; k < 10; k++ {
		if in.Clicked(116, 22+k*9, 200, 9) && m.Offers[t][k] != 0 {
			var err error
			if t == 0 {
				err = s.AcceptTrikot(mi, k)
			} else {
				err = s.AcceptBoard(mi, sp.board-1, k)
			}
			if err != nil {
				app.Push(NewMessage(name, err.Error()))
			}
			return nil
		}
	}
	if in.Clicked(270, 158, 12, 10) {
		s.ChangeAdSpending(mi, false)
	}
	if in.Clicked(284, 158, 12, 10) {
		s.ChangeAdSpending(mi, true)
	}
	if MenuButtonHit(in) {
		app.Pop()
	}
	return nil
}

func (sp *SponsorScreen) Draw(app *App, dst *ebiten.Image) {
	g := app.Gfx
	m := sp.ss.M()
	g.Background(dst)
	labels := []string{"TRIKOT", "BANDE 1", "BANDE 2", "BANDE 3", "BANDE 4", "BANDE 5", "BANDE 6"}
	for b, l := range labels {
		bt := Button{X: 4 + b*44, Y: 4, W: 42, H: 12, Label: l}
		bt.Draw(g, dst, b == sp.board)
	}
	Draw(dst, g.SubFull(46, rectWH(6, 58, 104, 172)), 4, 20)
	g.Panel(dst, 112, 20, 204, 96)
	t, months, cur := 0, m.TrikotMonths, m.Ads[0]
	if sp.board > 0 {
		t, months, cur = 1, m.BoardMonths[sp.board-1], m.Ads[sp.board]
	}
	for k := 0; k < 10; k++ {
		y := 23 + k*9
		o := m.Offers[t][k]
		col := ColLight
		if app.Input.In(116, 22+k*9, 200, 9) {
			col = ColYellow
		}
		g.Text(dst, FontSmall, 116, y, game.SponsorNames[k], g.C(col))
		if o == 0 {
			g.Text(dst, FontSmall, 200, y, "KEIN INTERESSE...", g.C(ColGrey4))
		} else {
			g.TextRight(dst, FontSmall, 270, y, formatMoney(o), g.C(col))
			g.Text(dst, FontSmall, 276, y, itoa(m.OfferYears[t][k])+" J.", g.C(col))
		}
	}
	g.Panel(dst, 112, 120, 204, 70)
	contract := "KÜNDBAR"
	if months > 0 {
		contract = "VERTRAG: " + itoa(months) + " MONAT(E)"
	}
	g.Text(dst, FontSmall, 116, 123, contract+"   "+Money(cur)+"/MONAT", g.C(ColWhite))
	var boards int64
	for b := 1; b <= 6; b++ {
		boards += m.Ads[b]
	}
	g.Text(dst, FontSmall, 116, 134, "Trikotwerbung", g.C(ColLight))
	g.TextRight(dst, FontSmall, 312, 134, Money(m.Ads[0]), g.C(ColWhite))
	g.Text(dst, FontSmall, 116, 143, "Bandenwerbung", g.C(ColLight))
	g.TextRight(dst, FontSmall, 312, 143, Money(boards), g.C(ColWhite))
	g.Text(dst, FontSmall, 116, 152, "TV-Übertragung", g.C(ColLight))
	g.TextRight(dst, FontSmall, 312, 152, Money(m.Ads[7]), g.C(ColWhite))
	g.Text(dst, FontSmall, 116, 161, "Werbeausgaben", g.C(ColLight))
	g.Text(dst, FontNormal, 272, 158, "-", g.C(ColYellow))
	g.Text(dst, FontNormal, 286, 158, "+", g.C(ColYellow))
	g.TextRight(dst, FontSmall, 266, 161, Money(m.Ads[8]), g.C(ColWhite))
	g.Text(dst, FontSmall, 116, 176, "Summe", g.C(ColBeige))
	g.TextRight(dst, FontSmall, 312, 176, Money(m.Ads[0]+boards+m.Ads[7]-m.Ads[8]), g.C(ColYellow))
	g.DrawMenuButton(dst)
}
