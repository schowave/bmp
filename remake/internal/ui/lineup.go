package ui

import (
	"image"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/schowave/bmp/remake/internal/game"
)

// The lineup screen "IHRE MANNSCHAFT" (f_1ecd_1560): the squad list in a
// window at the left (f_1ecd_06af), the EINSATZ meter and a column of icons
// at the right. The pitch icon opens the tactics board (f_1ecd_26e6), on
// which the players of the team are moved with the mouse (f_1ecd_29c6),
// with the three formations of the automatic lineup below it. The
// contracts icon shows the contract list in the same window (f_2277_2a8f).

// Views of the lineup screen; the icon of the view is shown pressed
// (local_b8).
const (
	lineupList = iota
	lineupPitch
	lineupContracts
)

// Columns of the squad lists (D:0x7c4 widths, D:0x4c48.. headers). A list
// shows the columns of a bit mask.
var lineupColW = [18]int{11, 18, 57, 13, 12, 12, 14, 36, 12, 14, 12, 12, 13, 27, 40, 13, 25, 33}

var lineupColHead = [18]string{"NR ", "ART   ", "NAME                    ", "SP  ", "      ", "      ",
	"SP  ", "STÄRKEN  ", "ST  ", "TO  ", "      ", "      ", "TO  ", "GK RK  ", "STATUS      ", "TD ",
	"V.DAUER  ", "  GEHALT"}

// Hints in the footer for the column under the mouse (D:0x2330).
var lineupColHint = [18]string{"RÜCKENNUMMER DES SPIELERS", "ART DES SPIELERS: ", "NAME: ",
	"ANZAHL DER LIGASPIELE", "ANZAHL DER SPIELE IM DFB-POKAL", "ANZAHL DER SPIELE IM EUROPAPOKAL",
	"SPIELE GESAMT (LIGA U. POKAL)", "KONDITION,TECHNIK,FORM ", "DURCHSCHNTTS-STÄRKE",
	"ANZAHL DER TORE IN DER LIGA", "ANZAHL DER TORE IM DFB-POKAL", "ANZAHL DER TORE IM EUROPAPOKAL",
	"TORE GESAMT (LIGA U. POKAL)", "ANZAHL DER GELBEN UND ROTEN KARTEN", "MOMENTANER STATUS DES SPIELERS",
	"TENDENZ DES SPIELERS: ", "DAUER DES VERTRAGES MIT DEM SPIELER (IN SAISONS)", "MONATLICHES GEHALT DES SPIELERS"}

const (
	lnMaskSquad     = 0xe28f  // NR ART NAME SP STÄRKEN TO GK RK STATUS TD
	lnMaskContracts = 0x3d146 // ART NAME SP ST TO STATUS TD V.DAUER GEHALT
)

// Row colours of the positions (D:0x7b4) and of the number of a reserve
// player.
var lnPosColor = [4]int{21, 26, 25, 24}

const lnReserveColor = ColGrey4

// Positions of the names of the positions (D:0x2378) and of the tendency.
var (
	lnPositionLong = [4]string{"TORWART", "ABWEHRSPIELER", "MITTELFELDSPIELER", "ANGREIFER"}
	lnTendencyLong = [3]string{"FALLEND.", "KONSTANT.", "STEIGEND."}
	lnTendencyMark = [3]string{"-", "O", "+"}
)

// Geometry of the screen.
const (
	lnListX0   = 6    // left edge of the list
	lnListX1   = 0xec // right edge of the squad list
	lnContX1   = 0xef // right edge of the contract list
	lnPitchX1  = 0xaf // right edge of list hover and footer beside the board
	lnTitleY   = 16   // baseline of the title
	lnHeadY    = 24   // baseline of the header
	lnRow0Y    = 34   // baseline of the first row; rows are 6 pixels apart
	lnBottomY  = 0xb0 // line above the footer
	lnFooterY  = 0xb8 // baseline of the footer
	lnIconX    = 0x10c
	lnMeterX   = 0x10b
	lnMeterY   = 5
	lnBoardX   = 0xb5 // window of the tactics board
	lnBoardY   = 0x56
	lnFieldX   = 0xbc // the pitch picture in it
	lnFieldY   = 0x5d
	lnCellX    = 0xc2 // top left of the marker of cell 0,0
	lnCellY    = 99
	lnFormY    = 0xc0 // formation buttons
	lnMarkerSz = 13
)

// lnIconY are the icon buttons at the right: the list, the tactics board,
// the contracts and HAUPT MENU.
var lnIconY = [4]int{0x29, 0x56, 0x83, 0xb0}

var lnIcons = [3]Icon{IconLineup, IconPitch, IconContracts} // D:0x5410

// SquadScreen is the lineup screen.
type SquadScreen struct {
	ss     *Session
	view   int
	vac    game.BoardCell // the cell the last player taken out left
	picked int            // slot picked on the tactics board, or -1
	// cursor is the cell under the mouse on the board, col < 0 for none.
	cursorCol, cursorRow int

	// During a match (f_1a58_1f75: f_1ecd_1560 with N players) the screen
	// substitutes: md is the match day, matchMi the manager and out the
	// player taken out of the team, or -1.
	md      *game.MatchDay
	matchMi int
	out     int

	// note is what the key o did, in the footer while nothing else is
	// shown there: as many of its parts as fit.
	note []string
}

// NewSquadScreen creates the lineup screen (team group, first item).
func NewSquadScreen(ss *Session) *SquadScreen {
	return &SquadScreen{ss: ss, picked: -1, cursorCol: -1, out: -1}
}

// NewMatchLineup opens the lineup of manager mi during a match: a click on
// a player of the team takes him out, a click on a reserve player then
// brings that one in.
func NewMatchLineup(ss *Session, md *game.MatchDay, mi int) *SquadScreen {
	q := NewSquadScreen(ss)
	q.md, q.matchMi = md, mi
	return q
}

// NewSquadContracts opens the lineup screen with the contract list
// (team group, second item: f_1ecd_1560 with 'c').
func NewSquadContracts(ss *Session) *SquadScreen {
	q := NewSquadScreen(ss)
	q.view = lineupContracts
	return q
}

func (q *SquadScreen) mi() int {
	if q.md != nil {
		return q.matchMi
	}
	return q.ss.Mgr()
}

func (q *SquadScreen) mask() int {
	if q.view == lineupContracts {
		return lnMaskContracts
	}
	return lnMaskSquad
}

// x1 is the right edge of the list for the hover and the footer.
func (q *SquadScreen) x1() int {
	switch q.view {
	case lineupPitch:
		return lnPitchX1
	case lineupContracts:
		return lnContX1
	}
	return lnListX1
}

// hover returns the slot and the column of the list under the mouse
// (f_1ecd_000e), or -1.
func (q *SquadScreen) hover(in *Input) (slot, col int) {
	if in.Y < lnRow0Y-4 || in.X < lnListX0 || in.X > q.x1() {
		return -1, -1
	}
	i := (in.Y - (lnRow0Y - 4)) / 6
	if i >= game.SquadMax || q.ss.S.Squads[q.mi()][i].Empty() {
		return -1, -1
	}
	x := in.X - lnListX0
	sum := 0
	for c := 0; c < 18; c++ {
		if q.mask()&(1<<c) == 0 {
			continue
		}
		sum += lineupColW[c]
		if x < sum {
			return i, c
		}
	}
	return -1, -1
}

// boardCell returns the cell of the tactics board under the mouse
// (f_1ecd_29c6), or col -1.
func (q *SquadScreen) boardCell(in *Input) (col, row int) {
	if q.view != lineupPitch || in.X < 0xc3 || in.X > 0x133 || in.Y < 100 || in.Y > 0xd4 {
		return -1, -1
	}
	return min((in.X-lnCellX)/16, game.BoardCols-1), min((in.Y-lnCellY)/14, game.BoardRows-1)
}

func lnIconHit(in *Input, x, y int) bool { return in.Clicked(x, y, ButtonW, ButtonH) }

func (q *SquadScreen) Update(app *App) error {
	s := q.ss.S
	mi := q.mi()
	m := s.Managers[mi]
	in := &app.Input
	q.cursorCol, q.cursorRow = q.boardCell(in)

	// The EINSATZ meter follows the mouse while the button is held.
	if (in.Left || in.LeftHeld) && in.X > 0x10a && in.X < 0x13f && in.Y > 4 && in.Y < 0x24 {
		m.EffortSetting = lnClamp(in.X-0x113, 0, 0x22)
		return nil
	}
	if in.Escape {
		q.exit(app)
		return nil
	}
	// The icons; while the board is open it covers all but the first.
	if lnIconHit(in, lnIconX, lnIconY[0]) {
		q.setView(lineupList)
		return nil
	}
	if q.view != lineupPitch {
		switch {
		case lnIconHit(in, lnIconX, lnIconY[1]):
			q.setView(lineupPitch)
			return nil
		case lnIconHit(in, lnIconX, lnIconY[2]):
			q.setView(lineupContracts)
			return nil
		case lnIconHit(in, lnIconX, lnIconY[3]):
			q.exit(app)
			return nil
		}
	}
	if q.view == lineupPitch {
		if q.updateBoard(app) {
			return nil
		}
		// A hidden key of the remake, not of the original.
		for _, c := range in.Chars {
			if c == 'o' && q.md == nil {
				q.optimize(app)
				return nil
			}
		}
	}
	if !in.Left && !in.Right {
		return nil
	}
	slot, col := q.hover(in)
	if slot < 0 {
		return nil
	}
	if q.view == lineupContracts {
		if in.Left && q.md == nil {
			(&ContractScreen{ss: q.ss}).negotiate(app, slot)
		}
		return nil
	}
	// The number and the position take the clicks; during a match the whole
	// row does, so that a click on the name substitutes too (the remake's).
	if col > 1 && q.md == nil {
		return nil
	}
	if q.md != nil {
		if in.Left {
			q.substitute(app, slot)
		}
		return nil
	}
	var err error
	if in.Left {
		err = s.LineupLeft(mi, slot, &q.vac)
	} else {
		err = s.LineupRight(mi, slot)
	}
	if le, ok := err.(game.LineupError); ok {
		app.Push(newLineupToast(le))
	}
	return nil
}

// updateBoard handles the tactics board and the formation buttons; it
// reports whether the input was used.
func (q *SquadScreen) updateBoard(app *App) bool {
	s := q.ss.S
	mi := q.mi()
	in := &app.Input
	if !in.Left {
		return false
	}
	for f := 0; f < 3 && q.formationButtons(); f++ {
		if lnIconHit(in, f*60+4, lnFormY) {
			s.SetFormation(mi, f)
			q.picked = -1
			q.vac = game.BoardCell{}
			return true
		}
	}
	if q.cursorCol < 0 {
		return false
	}
	if s.ManualLineup(mi) {
		app.Push(newLineupToast(game.ErrAutoDisabled))
	}
	at := s.PlayerAt(mi, q.cursorCol, q.cursorRow)
	if at == q.out && at >= 0 {
		at = -1 // he is not on the board any more
	}
	switch {
	case q.picked < 0:
		q.picked = at
	default:
		s.MoveOnBoard(mi, q.picked, q.cursorCol, q.cursorRow)
		q.picked = -1
	}
	return true
}

// optimize sets the best lineup (game.State.OptimizeLineup) and notes in the
// footer what it is worth, as expected points per match: the new lineup and
// the automatic lineups of the three formations, against the next opponent,
// who is named, or else against the clubs of the league.
func (q *SquadScreen) optimize(app *App) {
	s := q.ss.S
	res := s.OptimizeLineup(q.mi())
	q.picked = -1
	q.vac = game.BoardCell{}
	q.note = []string{"OPTIMAL " + pointsText(res.After),
		"STAND. " + pointsText(res.Auto[0]) + "/" + pointsText(res.Auto[1]) + "/" + pointsText(res.Auto[2])}
	if res.Opponent >= 0 {
		q.note = append(q.note, opponentNote(s.Teams[res.Opponent].Name, res.Opponent, res.Home)...)
	}
}

// pointsText writes n hundredths of a point with a decimal comma, "1,35".
func pointsText(n int) string { return decimalText(n, 2) }

// signedDecimal is decimalText with a plus for a gain: "+0,21", "-0,05".
func signedDecimal(n, places int) string {
	if n >= 0 {
		return "+" + decimalText(n, places)
	}
	return decimalText(n, places)
}

// decimalText writes n with places digits after a decimal comma:
// decimalText(135, 2) is "1,35", decimalText(410, 1) "41,0".
func decimalText(n, places int) string {
	pre := ""
	if n < 0 {
		pre, n = "-", -n
	}
	div := 1
	for range places {
		div *= 10
	}
	frac := itoa(n % div)
	for len(frac) < places {
		frac = "0" + frac
	}
	return pre + itoa(n/div) + "," + frac
}

func (q *SquadScreen) setView(v int) {
	q.view = v
	q.picked = -1
}

// substitute handles a click on slot i during a match: a player of the
// team is taken out, a reserve player then comes in for him.
func (q *SquadScreen) substitute(app *App, i int) {
	s := q.ss.S
	mi := q.mi()
	err := q.md.CanSubstitute(mi, i)
	if err == nil {
		switch sl := &s.Squads[mi][i]; {
		case sl.Lineup <= 11:
			q.out = i
		case q.out < 0:
			// Nobody taken out: after an injury the team has a place free.
			err = q.md.SubstituteIn(mi, i)
		default:
			err = q.md.Substitute(mi, q.out, i)
			q.out = -1
		}
	}
	if le, ok := err.(game.LineupError); ok {
		app.Push(newLineupToast(le))
	}
}

func (q *SquadScreen) exit(app *App) {
	// During a match the strength gets the tactical bonuses (f_0f9d_0002
	// with 1).
	if q.md != nil {
		q.md.LineupClosed(q.mi())
	} else {
		q.ss.S.LineupStrength(q.mi(), false)
	}
	app.Pop()
}

func lnClamp(v, lo, hi int) int { return max(lo, min(hi, v)) }

func (q *SquadScreen) Draw(app *App, dst *ebiten.Image) {
	g := app.Gfx
	s := q.ss.S
	m := s.Managers[q.mi()]
	in := &app.Input
	g.Background(dst)

	// The list window.
	g.Window(dst, 2, 7, 2+0xee, 7+0xb5, ColBlue)
	g.TextMid(dst, FontNormal, lnListX0, lnListX1, lnTitleY, "IHRE MANNSCHAFT", ColBeige, ColGrey7)
	head := ""
	for c := 0; c < 18; c++ {
		if q.mask()&(1<<c) != 0 {
			h := lineupColHead[c]
			if c == 2 && q.view == lineupContracts {
				h = h[:len(h)-3]
			}
			head += h
		}
	}
	g.TextAt(dst, FontSmall, lnListX0, lnHeadY, head, ColBeige, NoShadow)
	lineIncl(dst, lnListX0, lnHeadY+2, lnListX1, lnHeadY+2, g.PalColor(ColBeige))
	lineIncl(dst, lnListX0, lnBottomY, lnListX1, lnBottomY, g.PalColor(ColBeige))

	// The row under the mouse, or the one of the player under the mouse
	// on the board, is shown inverted (XOR 31: a yellow bar).
	bar, barX0, barX1 := -1, lnListX0, q.x1()
	footer := ""
	slot, col := q.hover(in)
	onBoard := -1
	if app.stack[len(app.stack)-1] != Scene(q) {
		// A message is shown: no highlight below it.
		slot = -1
	} else if q.cursorCol >= 0 {
		onBoard = s.PlayerAt(q.mi(), q.cursorCol, q.cursorRow)
	}
	switch {
	case onBoard >= 0:
		bar, barX0, barX1 = onBoard, 4, 0xb3
		footer = q.boardHint(onBoard)
	case slot >= 0:
		bar = slot
		footer = q.hint(slot, col)
	}
	for i := 0; i < game.SquadMax; i++ {
		if s.Squads[q.mi()][i].Empty() {
			continue
		}
		y := lnRow0Y + 6*i
		if i == bar {
			fillIncl(dst, barX0, y-4, barX1, y, g.PalColor(ColYellow))
		}
		q.drawRow(g, dst, i, y, i == bar, barX0, barX1)
	}
	if footer == "" {
		footer = fitParts(g, FontSmall, q.x1()-lnListX0, q.note...)
	}
	g.TextMid(dst, FontSmall, lnListX0, q.x1(), lnFooterY, footer, ColLight, NoShadow)

	// EINSATZ meter (f_1ecd_1435).
	Draw(dst, g.Sub(37, image.Rect(0, 0, 0x33, 0x1f)), lnMeterX, lnMeterY)
	lineIncl(dst, 0x113, 0x19, 0x113+m.EffortSetting, 0x19, g.PalColor(ColWhite))
	lineIncl(dst, 0x113, 0x1a, 0x113+m.EffortSetting, 0x1a, g.PalColor(ColWhite))

	for i, ic := range lnIcons {
		g.drawIconButton(dst, ic, lnIconX, lnIconY[i], q.pressed() == i)
	}
	exit := IconMainMenu
	if q.md != nil {
		// During a match WEITER, and the club below the list.
		exit = IconWeiter
		g.TextMid(dst, FontBig, 2, 0xf0, 0xe4, s.Teams[m.Team].Name, ColBeige, ColBlack)
	}
	g.DrawIconButton(dst, exit, lnIconX, lnIconY[3])
	if q.view == lineupPitch {
		q.drawBoard(g, dst)
	}
}

// drawRow draws the cells of the player of slot i (f_1ecd_06af). Inside
// the bar the colours are inverted like the XOR bar of the original.
func (q *SquadScreen) drawRow(g *Gfx, dst *ebiten.Image, i, y int, inBar bool, barX0, barX1 int) {
	s := q.ss.S
	sl := &s.Squads[q.mi()][i]
	p := &s.Persons[sl.Person]
	pc := lnPosColor[p.Position()]
	contracts := q.view == lineupContracts
	x := lnListX0
	for c := 0; c < 18; c++ {
		if q.mask()&(1<<c) == 0 {
			continue
		}
		col := pc
		txt := ""
		switch c {
		case 0:
			// The player taken out for a substitution has no number.
			if sl.Lineup != 0 && i != q.out {
				txt = numText(int64(sl.Lineup), 2, true)
				if sl.Lineup > 11 {
					col = lnReserveColor
				}
			}
		case 1:
			txt = posShort[p.Position()]
		case 2:
			txt = p.Name
			if contracts && sl.Message && sl.Nego&0x7f > 0x63 {
				col = ColBeige
			}
		case 3, 4, 5:
			txt = numText(int64(sl.Games[c-3]), 2, true)
		case 6:
			if contracts {
				x -= 6 // the following columns move left, too
			}
			txt = numText(int64(sl.Games[0]+sl.Games[1]+sl.Games[2]), 2, true)
		case 7:
			txt = numText(int64(sl.Skill[0]), 2, true) + " " + numText(int64(sl.Skill[1]), 2, true) + " " +
				numText(int64(sl.Skill[2]), 2, true)
		case 8:
			txt = numText(int64(sl.Strength()), 2, true)
		case 9, 10, 11:
			txt = numText(int64(sl.Goals[c-9]), 2, true)
		case 12:
			txt = numText(int64(sl.Goals[0]+sl.Goals[1]+sl.Goals[2]), 2, true)
		case 13:
			txt = numText(int64(sl.Cards[1]), 2, true) + " " + numText(int64(sl.Cards[0]), 2, true)
		case 14:
			txt, col = q.statusCell(sl, col)
		case 15:
			if sl.Fatigue > 0x82 {
				col = ColRed
			}
			txt = lnTendencyMark[game.TendencyClass(sl.Tendency)]
		case 16:
			txt = itoa(sl.Years) + " JAHR"
			if txt[0] > '1' {
				txt += "E"
			}
		case 17:
			txt = numText(sl.Salary, 7, true) + " DM"
		}
		if inBar {
			col ^= 31
		}
		g.TextAt(dst, FontSmall, x, y, txt, col, NoShadow)
		x += lineupColW[c]
	}
}

// statusCell is the STATUS column: RESERVE, IM TEAM, or GESP./VERL. in
// red with the games of the suspension.
func (q *SquadScreen) statusCell(sl *game.Slot, col int) (string, int) {
	st := sl.Status & 3
	switch {
	case sl.Lineup > 11:
		return "RESERVE", col
	case st == 0 || (st == 1 && q.ss.S.CupDay):
		if sl.Lineup != 0 {
			return "IM TEAM", col
		}
		return " ", col
	case st == 1:
		return "GESP.(" + itoa(sl.Out) + ")", ColRed
	}
	return "VERL.", ColRed
}

// hint is the footer for the column c of slot i (f_1ecd_000e).
func (q *SquadScreen) hint(i, c int) string {
	s := q.ss.S
	sl := &s.Squads[q.mi()][i]
	p := &s.Persons[sl.Person]
	switch c {
	case 1:
		t := lnPositionLong[p.Position()]
		if sl.Loan != 0 {
			t += " (AUSGEL.)"
		}
		return t
	case 2:
		return "NAME: " + p.Name + " (" + itoa(p.Age) + " JAHRE)" + game.SideText(p.Side)
	case 7:
		return lineupColHint[7] + "(" + itoa(sl.Strength()) + ")"
	case 14:
		t := "STATUS: "
		st := sl.Status & 3
		switch {
		case sl.Lineup > 11:
			t += "RESERVE"
		case st == 1 && !s.CupDay:
			t += "NOCH " + itoa(sl.Out) + " SPIELE GESPERRT."
		case st == 2:
			t += "VERLETZT  (" + game.InjuryNames[sl.Injury] + ")"
		case sl.Lineup == 0:
			t += "HÄNGT HALT SO RUM..."
		default:
			t += "IM TEAM"
		}
		return t
	case 15:
		t := lineupColHint[15] + lnTendencyLong[game.TendencyClass(sl.Tendency)]
		if q.view == lineupList {
			t += " (ERSCHÖPFUNG:" + itoa(sl.Fatigue) + ")"
		}
		return t
	}
	return lineupColHint[c]
}

// boardHint is the footer for the player under the mouse on the board
// (f_1ecd_2897).
func (q *SquadScreen) boardHint(i int) string {
	s := q.ss.S
	sl := &s.Squads[q.mi()][i]
	p := &s.Persons[sl.Person]
	return "STÄRKE:" + itoa(sl.Strength()) + ", " + itoa(p.Age) + " JAHRE" + game.SideText(p.Side)
}

// Parts of PIC/6.VGA: the pitch, the formations and the markers.
var (
	lnFieldRect  = image.Rect(0, 0, 120, 120)
	lnMarkerRect = image.Rect(13, 121, 26, 134) // a clean marker
	lnPickRect   = image.Rect(26, 121, 39, 134) // ring of the picked player
	lnCursorRect = image.Rect(52, 121, 65, 134) // dashed circle under the mouse
)

func lnFormRect(f int) image.Rectangle { return image.Rect(0x79, f*0x18, 0x79+0x20, f*0x18+0x17) }

// formationButtons tells whether the formation buttons are under the board:
// during a match the formations do not apply, so the remake leaves them out
// (the original draws them there too).
func (q *SquadScreen) formationButtons() bool { return q.md == nil }

// boardSlots are the slots of the players on the tactics board: the team
// with a cell, but not the player taken out for a substitution, who has left
// the board.
func (q *SquadScreen) boardSlots() []int {
	var out []int
	for i := 0; i < game.SquadMax; i++ {
		sl := &q.ss.S.Squads[q.mi()][i]
		if sl.Empty() || sl.Lineup < 1 || sl.Lineup > 11 || sl.Col < 0 || i == q.out {
			continue
		}
		out = append(out, i)
	}
	return out
}

// drawBoard draws the tactics board with the markers of the team and the
// formation buttons (f_1ecd_26e6, f_1ecd_132a).
func (q *SquadScreen) drawBoard(g *Gfx, dst *ebiten.Image) {
	s := q.ss.S
	mi := q.mi()
	m := s.Managers[mi]
	for f := 0; f < 3 && q.formationButtons(); f++ {
		x := f*60 + 4
		Draw(dst, g.Pic(24+lnB2i(m.AutoLineup-f == 2), false), x, lnFormY)
		Draw(dst, g.Sub(6, lnFormRect(f)), x+9, lnFormY+9)
	}
	g.Window(dst, lnBoardX, lnBoardY, lnBoardX+0x84, lnBoardY+0x84, ColBlue)
	Draw(dst, g.Sub(6, lnFieldRect), lnFieldX, lnFieldY)
	for _, i := range q.boardSlots() {
		sl := &s.Squads[mi][i]
		x, y := sl.Col*16+lnCellX, sl.Row*14+lnCellY
		Draw(dst, g.SubKeyed(6, 0, lnMarkerRect), x, y)
		g.TextMid(dst, FontSmall, x+1, x+lnMarkerSz, y+8, itoa(sl.Lineup), ColBeige, NoShadow)
		if i == q.picked {
			Draw(dst, g.SubKeyed(6, 0, lnPickRect), x, y)
		}
	}
	if q.cursorCol >= 0 {
		Draw(dst, g.SubKeyed(6, 0, lnCursorRect), q.cursorCol*16+lnCellX, q.cursorRow*14+lnCellY)
	}
}

func lnB2i(b bool) int {
	if b {
		return 1
	}
	return 0
}

// lineupToast is the message box of the lineup screen (f_076b_0e57): a
// dark red box at the top right that grows, stays a moment and shrinks
// again; messages about the automatic lineup appear at once and stay half
// as long. It takes no input.
type lineupToast struct {
	lines [2]string
	auto  bool
	frame int
}

func newLineupToast(e game.LineupError) *lineupToast {
	return &lineupToast{lines: e, auto: e[0] == "Auto-Aufstellung"}
}

// lnToastLineH is the distance of the lines of the message box.
const lnToastLineH = 13

const lnToastSteps = 16

func (t *lineupToast) total() int {
	if t.auto {
		return 30
	}
	return 2*lnToastSteps + 60
}

func (t *lineupToast) Update(app *App) error {
	t.frame++
	if t.frame >= t.total() {
		app.Pop()
	}
	return nil
}

func (t *lineupToast) Draw(app *App, dst *ebiten.Image) {
	g := app.Gfx
	// The full box is 0xa4,0xc .. 0x141,0x2d; it grows from 0xb3,0x1b by
	// one pixel up and left, 10 wider and 2 higher per step.
	step := lnToastSteps - 1
	if !t.auto {
		switch {
		case t.frame < lnToastSteps:
			step = t.frame
		case t.frame >= t.total()-lnToastSteps:
			step = t.total() - 1 - t.frame
		}
	}
	x, y := 0xb3-step, 0x1b-step
	w, h := 7+10*step, 3+2*step
	if step < lnToastSteps-1 {
		fillIncl(dst, x+1, y+1, x+w-1, y+h-1, g.PalColor(ColDarkRed))
		return
	}
	lay := t.layout()
	fillIncl(dst, 165, 13, 319, lay.bottom, g.PalColor(ColOrange))
	fillIncl(dst, 166, 14, 318, lay.bottom-1, g.PalColor(ColBlack))
	fillIncl(dst, 167, 15, 317, lay.bottom-2, g.PalColor(ColDarkRed))
	for i, l := range lay.lines {
		g.TextMid(dst, FontNormal, lay.x0, lay.x1, lay.ys[i], l, ColWhite, ColBlack)
	}
}

// toastLayout is where the full message box puts its texts.
type toastLayout struct {
	lines  []string
	ys     []int // baselines
	x0, x1 int   // the band the lines are centred in
	bottom int   // last row of the box
}

// layout places the two lines of the full box, centred in 0xb9..0x12f.
func (t *lineupToast) layout() toastLayout {
	lay := toastLayout{x0: 0xb9, x1: 0x12f, bottom: 44}
	for i, l := range t.lines {
		lay.lines = append(lay.lines, l)
		lay.ys = append(lay.ys, 0x1a+i*lnToastLineH)
	}
	return lay
}

// pressed is the icon shown pressed: the list one also while the board is
// open.
func (q *SquadScreen) pressed() int {
	if q.view == lineupContracts {
		return 2
	}
	return 0
}
