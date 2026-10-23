package ui

import (
	"fmt"
	"os"
	"strings"

	"github.com/schowave/bmp/remake/internal/game"
)

// Development support for playtests (-control in cmd/bmp): named click
// targets for scripts, a description of what the game shows and a dump of
// the running game.

// session returns the running game, or nil before one is started.
func (a *App) session() *Session {
	for _, s := range a.stack {
		if d, ok := s.(*Driver); ok {
			return d.ss
		}
	}
	return nil
}

// target returns the centre of the named click target of a script:
//
//	menu.desk menu.leagues menu.team    the groups at the left of the main menu
//	menu.cups menu.disk menu.quit       the groups at the right
//	menu.item.1 .. menu.item.9          the icons of an open group, column by column
//	menu.calendar menu.messages         end the turn, open the messages
//	messages.up messages.close messages.down
//	button.menu                         HAUPT MENU / WEITER at the bottom right
//	lineup.exit                         HAUPT MENU / WEITER of the lineup screen
//	box.N                               button N (from 0) of the dialog on top
func (a *App) target(name string) (x, y int, ok bool) {
	centre := func(x, y, w, h int) (int, int, bool) { return x + w/2, y + h/2, true }
	groups := []string{"desk", "leagues", "team", "cups", "disk", "quit"}
	for i, g := range groups {
		if name == "menu."+g {
			x := menuLeftX
			if i >= 3 {
				x = menuRightX
			}
			return centre(x, menuRowsY[i%3], ButtonW, ButtonH)
		}
	}
	var k int
	if n, _ := fmt.Sscanf(name, "menu.item.%d", &k); n == 1 && k >= 1 && k <= 9 {
		return centre(submenuCols[(k-1)/3], submenuRowsY[(k-1)%3], ButtonW, ButtonH)
	}
	for i, b := range []string{"up", "close", "down"} {
		if name == "messages."+b {
			return centre(msgButtonX, msgPanelY+i*msgFieldPitch, msgButtonW, msgButtonH)
		}
	}
	switch name {
	case "menu.calendar":
		return centre(calendarRect())
	case "menu.messages":
		return 0x90, 0xb0, true
	case "button.menu":
		return centre(MenuButton.X, MenuButton.Y, MenuButton.W, MenuButton.H)
	case "lineup.exit":
		return centre(lnIconX, lnIconY[3], ButtonW, ButtonH)
	}
	if n, _ := fmt.Sscanf(name, "box.%d", &k); n == 1 && k >= 0 {
		switch b := a.Top().(type) {
		case *MessageBox:
			b.geometry(a.Gfx)
			if k < len(b.buttons) {
				bt := b.buttons[k]
				return centre(bt.X, bt.Y, bt.W, bt.H)
			}
		case *AlertBox:
			if k == 0 {
				return centre(b.ok.X, b.ok.Y, b.ok.W, b.ok.H)
			}
		case *tsMessage:
			if k == 0 {
				return centre(0xb2, 0xa0, 57, 12)
			}
		}
	}
	return 0, 0, false
}

// Describe tells a playtest driver what the game shows: the scene on top
// with its text, then the state of the running game, one item per line.
func (a *App) Describe() string {
	var b strings.Builder
	fmt.Fprintf(&b, "scene: %T\n", a.Top())
	text := ""
	switch t := a.Top().(type) {
	case *MessageBox:
		text = t.Title + " | " + strings.Join(t.Lines, " ") + " | [" + strings.Join(t.Buttons, "] [") + "]"
	case *AlertBox:
		text = strings.Join(t.Lines[:], " ")
	case *tsMessage:
		text = strings.Join(t.lines[:], " ")
	case *tsToast:
		text = t.l0 + " " + t.l1
	case *DeskToast:
		text = t.Line1 + " " + t.Line2
	case *lineupToast:
		text = t.lines[0] + " " + t.lines[1]
	case *MainMenu:
		if t.msgOpen {
			text = "messages open"
		}
	}
	if text = strings.Join(strings.Fields(text), " "); text != "" {
		fmt.Fprintf(&b, "text: %s\n", text)
	}
	if a.ctlErr != "" {
		fmt.Fprintf(&b, "error: %s\n", a.ctlErr)
	}
	ss := a.session()
	if ss == nil {
		return b.String()
	}
	s := ss.S
	m := ss.M()
	d := game.DateOf(s.StartYear, s.Day)
	fmt.Fprintf(&b, "date: %d.%d.%d (day %d)\n", d.Day, d.Month+1, d.Year, s.Day)
	fmt.Fprintf(&b, "manager: %s, %s, league %d, place %d, next matchday %d\n",
		m.Name, s.Teams[m.Team].Name, m.League+1, s.Clubs[m.Team].Pos+1, s.Matchday[m.League])
	fmt.Fprintf(&b, "money: %d\n", m.Money)
	if n := len(m.Inbox); n > 0 {
		fmt.Fprintf(&b, "messages: %d, latest %q\n", n, strings.ReplaceAll(strings.TrimSpace(m.Inbox[0].Text), "\n", " / "))
	}
	return b.String()
}

// dump writes the running game to a save file.
func (a *App) dump(path string) error {
	ss := a.session()
	if ss == nil {
		return fmt.Errorf("dump %s: no game running", path)
	}
	data, err := ss.S.Save()
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o644)
}
