package ui

import (
	"slices"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/schowave/bmp/remake/internal/game"
)

// tsList is the player list of the original (f_1ecd_06af draws it,
// f_1ecd_000e finds the cell under the pointer and shows its hint). A list
// shows a selection of 18 columns, given as a bit mask.
type tsList struct {
	m     int    // squad: manager index or game.MarketIndex
	x     int    // left edge of the texts
	head  int    // baseline of the column headers
	right int    // right end of the header line
	foot  int    // the line below the rows
	mask  uint32 // columns shown
	mode  int    // 0 squad, 1 market (buy), 2 market (loan), 3 contracts
	title string // drawn centred above the headers

	// Pointer tests (f_1ecd_000e): the cell grid starts at hitX, the rows
	// at rowY; the yellow bar reaches from hitX to hitRight, the hint is
	// centred between hitX and hitRight on hintY.
	hitX, rowY, hitRight, hintY int
	// rowBase is the row index of the first slot (the market list counts
	// its rows from 1).
	rowBase int

	// prices are the market prices of the rows. The value of a listed
	// player with an offer draws a random number (f_2277_25de), which the
	// original does when it builds the list (f_1ecd_06af), not each frame.
	prices   []int64
	priceKey []int64
}

// updatePrices works out the market prices again when the rows or the
// mode changed.
func (l *tsList) updatePrices(s *game.State) {
	if l.mode != 1 && l.mode != 2 {
		l.prices, l.priceKey = nil, nil
		return
	}
	n := l.count(s)
	key := []int64{int64(l.mode)}
	for i := 0; i < n; i++ {
		sl := &s.Squads[l.m][i]
		key = append(key, int64(sl.Person), int64(sl.Status), int64(sl.Buyer), sl.Salary)
	}
	if slices.Equal(key, l.priceKey) {
		return
	}
	l.priceKey = key
	l.prices = l.prices[:0]
	for i := 0; i < n; i++ {
		l.prices = append(l.prices, s.MarketPrice(i, l.mode == 2))
	}
}

// price is the market price of row i, 0 before the first update.
func (l *tsList) price(i int) int64 {
	if i < len(l.prices) {
		return l.prices[i]
	}
	return 0
}

// Column masks of the lists of the original.
const (
	tsMaskContracts = 0x3d146 // ART NAME SP ST TO STATUS TD V.DAUER GEHALT
	tsMaskTransfer  = 0x090c6 // ART NAME SP STÄRKEN TO TD
	tsMaskMarket    = 0x20086 // ART NAME STÄRKEN WERT
)

// Column widths (D:0x7c4) and headers (D:0x4c48..).
var (
	tsWidths  = [18]int{11, 18, 57, 13, 12, 12, 14, 36, 12, 14, 12, 12, 13, 27, 40, 13, 25, 33}
	tsHeaders = [18]string{"NR ", "ART   ", "NAME                    ", "SP  ", "      ", "      ", "SP  ",
		"STÄRKEN  ", "ST  ", "TO  ", "      ", "      ", "TO  ", "GK RK  ", "STATUS      ", "TD ", "V.DAUER  ", ""}
	tsPosColor = [4]int{21, 26, 25, 24} // D:0x7b4: TOR, ABW, MIT, ANG
	tsPos      = [4]string{"TOR", "ABW", "MIT", "ANG"}
	tsTendency = [3]string{"-", "O", "+"}
	// Hints of the columns (D:0x2330).
	tsHints = [18]string{"RüCKENNUMMER DES SPIELERS", "ART DES SPIELERS: ", "NAME: ", "ANZAHL DER LIGASPIELE",
		"ANZAHL DER SPIELE IM DFB-POKAL", "ANZAHL DER SPIELE IM EUROPAPOKAL", "SPIELE GESAMT (LIGA U. POKAL)",
		"KONDITION,TECHNIK,FORM ", "DURCHSCHNTTS-STÄRKE", "ANZAHL DER TORE IN DER LIGA",
		"ANZAHL DER TORE IM DFB-POKAL", "ANZAHL DER TORE IM EUROPAPOKAL", "TORE GESAMT (LIGA U. POKAL)",
		"ANZAHL DER GELBEN UND ROTEN KARTEN", "MOMENTANER STATUS DES SPIELERS", "TENDENZ DES SPIELERS: ",
		"DAUER DES VERTRAGES MIT DEM SPIELER (IN SAISONS)", "MONATLICHES GEHALT DES SPIELERS"}
	tsPosNames = [4]string{"TORWART", "ABWEHRSPIELER", "MITTELFELDSPIELER", "ANGREIFER"}
)

// pad formats n like f_076b_0681 with the minimum width of D:0x79c: missing
// digits are '^', a blank of the width of a digit.
func tsPad(n int64, width int) string {
	s := itoa64(n)
	for len(s) < width {
		s = "^" + s
	}
	return s
}

func (l *tsList) has(c int) bool { return l.mask&(1<<c) != 0 }

// count is the number of slots of the squad.
func (l *tsList) count(s *game.State) int { return s.SquadSize(l.m) }

// header returns the line of the column headers.
func (l *tsList) header() string {
	h := ""
	for c := 0; c < 18; c++ {
		if !l.has(c) {
			continue
		}
		switch {
		case c == 2:
			h += tsHeaders[2]
			if l.mode != 0 {
				h = h[:len(h)-3]
			}
		case c == 17 && (l.mode == 1 || l.mode == 2):
			h += "     WERT"
		case c == 17:
			h += "  GEHALT"
		default:
			h += tsHeaders[c]
		}
	}
	return h
}

// draw draws title, headers, the rows and the line below them. hover is
// the slot under the pointer (-1 = none): its row gets the yellow bar.
func (l *tsList) draw(g *Gfx, dst *ebiten.Image, s *game.State, cur, hover int) {
	if l.title != "" {
		// Centred like f_06c7_087c: (x0+x1-width)/2.
		g.TextAt(dst, FontNormal, (l.x+l.right-g.TextWidth(FontNormal, l.title))/2, l.head-8, l.title, ColBeige, ColGrey7)
	}
	g.TextAt(dst, FontSmall, l.x, l.head, l.header(), ColBeige, NoShadow)
	FillRect(dst, l.x, l.head+2, l.right-l.x+1, 1, g.PalColor(ColBeige))
	for i := 0; i < l.count(s); i++ {
		l.drawRow(g, dst, s, cur, i, l.head+10+i*6, i == hover)
	}
	FillRect(dst, l.x, l.foot, l.right-l.x+1, 1, g.PalColor(ColBeige))
}

// rowColor is the colour of the texts of a row.
func (l *tsList) rowColor(s *game.State, sl *game.Slot) int {
	c := tsPosColor[s.Persons[sl.Person].Position()]
	if l.mode == 3 {
		c = contractColor(sl, c)
	}
	return c
}

// contractColor is the colour of a row of a contract list, c otherwise
// (f_1ecd_06af in mode 3): blue grey for a player not willing to talk for
// now, dark red for one who retires when his contract ends.
func contractColor(sl *game.Slot, c int) int {
	if sl.Nego > 0 && sl.Nego < 100 {
		c = ColGrey3
	}
	if sl.Nego&0x80 != 0 {
		c = ColRedDark
	}
	return c
}

// nameColor is the colour of the name of a row, base unless the name is
// marked light: an own player a club made an offer for (0x40 marks an offer
// for a squad player, 0x80 one for a listed player, so the rule covers the
// squad list as well as the market list), or in the contract list a player
// who asks for a longer contract.
func (l *tsList) nameColor(s *game.State, sl *game.Slot, cur, base int) int {
	if l.mode <= 2 && sl.HasOffer() && s.Persons[sl.Person].Owner == cur {
		return ColBeige
	}
	if l.mode == 3 && sl.Message && sl.AsksForContract() {
		return ColBeige
	}
	return base
}

func (l *tsList) drawRow(g *Gfx, dst *ebiten.Image, s *game.State, cur, i, y int, hot bool) {
	sl := &s.Squads[l.m][i]
	p := &s.Persons[sl.Person]
	base := l.rowColor(s, sl)
	// The bar is drawn with XOR 31 over the row (bar 8^31 = yellow).
	xor := 0
	if hot {
		xor = 31
		FillRect(dst, l.hitX, y-4, l.hitRight-l.hitX+1, 5, g.PalColor(ColBlue^31))
	}
	text := func(x int, t string, c int) { g.TextAt(dst, FontSmall, x, y, t, c^xor, NoShadow) }
	x := l.x
	for c := 0; c < 18; c++ {
		if !l.has(c) {
			continue
		}
		col := base
		switch c {
		case 0:
			if sl.Lineup > 11 {
				col = ColGrey4
			}
			if sl.Lineup != 0 {
				text(x, tsPad(int64(sl.Lineup), 2), col)
			}
		case 1:
			text(x, tsPos[p.Position()], col)
			if sl.Loan != 0 || (p.Owner == cur && (l.mode == 1 || l.mode == 2)) {
				FillRect(dst, x+tsWidths[1]-3, y-2, 1, 1, g.PalColor(col^xor))
			}
		case 2:
			text(x, p.Name, l.nameColor(s, sl, cur, col))
		case 3, 4, 5:
			text(x, tsPad(int64(sl.Games[c-3]), 2), col)
		case 6:
			if l.mask == tsMaskContracts {
				x -= 6
			}
			text(x, tsPad(int64(sl.Games[0]+sl.Games[1]+sl.Games[2]), 2), col)
		case 7:
			t := tsPad(int64(sl.Skill[0]), 2) + " " + tsPad(int64(sl.Skill[1]), 2) + " " + tsPad(int64(sl.Skill[2]), 2)
			if l.mode != 0 {
				x -= 6
			}
			text(x, t, col)
		case 8:
			text(x, tsPad(int64(sl.Strength()), 2), col)
		case 9, 10, 11:
			text(x, tsPad(int64(sl.Goals[c-9]), 2), col)
		case 12:
			text(x, tsPad(int64(sl.Goals[0]+sl.Goals[1]+sl.Goals[2]), 2), col)
		case 13:
			text(x, tsPad(int64(sl.Cards[1]), 2)+" "+tsPad(int64(sl.Cards[0]), 2), col)
		case 14:
			t, red := tsStatus(s, sl)
			if red {
				col = ColRed
			}
			text(x, t, col)
		case 15:
			if sl.Fatigue > 0x82 {
				col = ColRed
			}
			text(x, tsTendency[game.TendencyClass(sl.Tendency)], col)
		case 16:
			t := itoa(sl.Years) + " JAHR"
			if t[0] > '1' {
				t += "E"
			}
			text(x, t, col)
		case 17:
			var v int64
			if l.mode == 1 || l.mode == 2 {
				v = l.price(i)
			} else {
				v = sl.Salary
			}
			text(x, tsPad(v, 7)+" DM", col)
		}
		x += tsWidths[c]
	}
}

// tsStatus is the text of the STATUS column; red is set for suspended and
// injured players.
func tsStatus(s *game.State, sl *game.Slot) (string, bool) {
	st := sl.Status & 3
	switch {
	case sl.Lineup > 11:
		return "RESERVE", false
	case st == 0 || (st == 1 && s.CupDay):
		if sl.Lineup == 0 {
			return " ", false
		}
		return "IM TEAM", false
	case st == 1:
		return "GESP.(" + itoa(sl.Out) + ")", true
	}
	return "VERL.", true
}

// cell returns the slot and column under the pointer (f_1ecd_000e), or
// ok = false.
func (l *tsList) cell(s *game.State, mx, my int) (slot, col int, ok bool) {
	row := (my - l.rowY) / 6 // truncated towards zero like the original
	rel := mx - l.hitX
	col = 0
	sum := 0
	for ; col < 18; col++ {
		if l.has(col) {
			sum += tsWidths[col]
			if sum > rel {
				break
			}
		}
	}
	slot = row - l.rowBase
	if mx < 0 || mx > l.hitRight || slot < 0 || slot >= l.count(s) || col == 18 {
		return 0, 0, false
	}
	return slot, col, true
}

// hint is the text f_1ecd_000e shows for a cell.
func (l *tsList) hint(s *game.State, slot, col int) string {
	sl := &s.Squads[l.m][slot]
	p := &s.Persons[sl.Person]
	var t string
	switch col {
	case 1:
		t = tsPosNames[p.Position()]
		if sl.Loan != 0 {
			t += " (AUSGEL.)"
		}
	case 2:
		t = "NAME: " + p.Name + " (" + itoa(p.Age) + " JAHRE)" + game.SideText(p.Side)
	case 14:
		t = "STATUS: "
		st := sl.Status & 3
		switch {
		case sl.Lineup > 11:
			t += "RESERVE"
		case st != 0 && !(st == 1 && s.CupDay):
			if st == 1 {
				t += "NOCH " + itoa(sl.Out) + " SPIELE GESPERRT."
			} else {
				t += "VERLETZT (" + game.InjuryNames[sl.Injury%len(game.InjuryNames)] + ")"
			}
		case sl.Lineup == 0:
			t += "HÄNGT HALT SO RUM..."
		default:
			t += "IM TEAM"
		}
	case 15:
		t = tsHints[15] + [3]string{"FALLEND.", "KONSTANT.", "STEIGEND."}[game.TendencyClass(sl.Tendency)]
		if l.hitRight == 0xec {
			t += " (ERSCHöPFUNG:" + itoa(sl.Fatigue) + ")"
		}
	default:
		t = tsHints[col]
	}
	if col == 7 {
		t += "(" + itoa(sl.Strength()) + ")"
	}
	return t
}

// drawHint draws the hint of the cell under the pointer.
func (l *tsList) drawHint(g *Gfx, dst *ebiten.Image, s *game.State, mx, my int) {
	if slot, col, ok := l.cell(s, mx, my); ok {
		g.TextMid(dst, FontSmall, l.hitX, l.hitRight, l.hintY, l.hint(s, slot, col), ColLight, NoShadow)
	}
}
