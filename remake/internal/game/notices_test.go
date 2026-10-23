package game

import (
	"slices"
	"testing"
)

func noticeGame(t *testing.T) *State {
	t.Helper()
	s, err := NewGame(mana(t), Options{Managers: []NewManager{{Name: "TEST", Team: 9}}, Level: 2, Seed: 1})
	if err != nil {
		t.Fatal(err)
	}
	return s
}

func TestNoticeQueue(t *testing.T) {
	s := noticeGame(t)
	if s.HasNotices() || s.TakeNotices() != nil {
		t.Fatal("new game has notices")
	}
	s.alert(0, BackdropMenu, msgWinterBreak)
	s.seasonScreen(0, 3, 1)
	s.seasonLine(0, msgPromoted2)
	s.seasonDialog(0, msgAdsFree)
	s.seasonLine(0, msgEurope)
	s.seasonDone(0)
	s.seasonPopup(0, msgRetires("TRAUTNER", 34))
	if !s.HasNotices() {
		t.Fatal("no notices")
	}
	q := s.TakeNotices()
	if s.HasNotices() || len(q) != 2 {
		t.Fatalf("queue %+v", q)
	}
	if a := q[0]; a.Kind != NoticeAlert || !slices.Equal(a.Lines, []string{"Achtung !", "Dies ist der letzte", "Spieltag vor der", "Winterpause."}) {
		t.Errorf("alert %+v", a)
	}
	se := q[1]
	if se.Kind != NoticeSeasonEnd || se.Title != "Sie beendeten Ihre Saison" || se.Heading != "Ereignisse" ||
		!slices.Equal(se.Lines, []string{"3.Platz der ZWEITE LIGA", msgPromoted2, msgEurope}) {
		t.Errorf("season %+v", se)
	}
	if len(se.Dialogs) != 1 || se.Dialogs[0].At != 2 || len(se.Dialogs[0].Lines) != 4 {
		t.Errorf("season dialogs %+v", se.Dialogs)
	}
	if len(se.Popups) != 1 || se.Popups[0].Lines[0] != "TRAUTNER hängt den" {
		t.Errorf("season popups %+v", se.Popups)
	}
	s.seasonScreen(0, 1, 0)
	s.seasonDone(0)
	if q := s.TakeNotices(); len(q[0].Lines) != 2 || q[0].Lines[1] != "Keine besonderen Ereignisse..." {
		t.Errorf("without events %+v", q)
	}
}

func TestSpecialDateCards(t *testing.T) {
	s := noticeGame(t)
	s.Start()
	s.TakeNotices()
	cards, packages := 0, 0
	for i := 0; i < 400; i++ {
		money := s.Managers[0].Money
		s.specialDate(0)
		for _, n := range s.TakeNotices() {
			cards++
			if n.Kind != NoticeCard || n.Title != "Frohe Weihnachten !" || n.Birthday {
				t.Fatalf("card %+v", n)
			}
			if len(n.Lines) > 0 {
				packages++
				if len(n.Lines) != 5 || len(n.Small) != 2 || s.Managers[0].Money >= money {
					t.Fatalf("card with packages %+v", n)
				}
			} else if s.Managers[0].Money != money {
				t.Fatal("card without packages costs money")
			}
		}
	}
	// About 1/4 of the dates show the card, half of them with packages.
	if cards < 60 || cards > 140 || packages < cards/4 || packages > cards*3/4 {
		t.Errorf("%d cards, %d with packages", cards, packages)
	}
	for i := 0; i < 40; i++ {
		s.specialDate(2)
	}
	q := s.TakeNotices()
	if len(q) == 0 || !q[0].Birthday || q[0].Lines[3] != "glorreichen Programmierer: WERNER KRAHE" || len(q[0].Small) != 2 {
		t.Errorf("birthday %+v", q)
	}
}

func TestCompletionPopupWindow(t *testing.T) {
	s := noticeGame(t)
	for _, c := range []struct {
		day   int
		popup bool
	}{{100, false}, {128, false}, {129, true}, {206, true}, {207, false}, {322, false}, {323, true}} {
		s.Day = c.day
		if s.popupWindow() != c.popup {
			t.Errorf("day %d: popup %v", c.day, !c.popup)
		}
	}
}

func TestSeasonNoticesInGame(t *testing.T) {
	s := noticeGame(t)
	s.Start()
	seen := false
	for steps := 0; !seen && steps < 100000; steps++ {
		for _, n := range s.TakeNotices() {
			if n.Kind == NoticeSeasonEnd && n.Manager == 0 && len(n.Lines) >= 2 {
				seen = true
			}
		}
		switch s.Loop.Phase {
		case PhaseTurn:
			s.TakeInbox(0)
			s.EndTurn()
		case PhaseMatches:
			for md := s.NextMatchDay(); md != nil; md = s.NextMatchDay() {
				md.Run()
			}
		case PhaseContracts:
			s.ContractsDone()
		default:
			t.Fatalf("phase %d", s.Loop.Phase)
		}
	}
	if !seen {
		t.Fatal("no season report")
	}
	for _, msg := range s.Messages(0) {
		if msg.Text == msgEurope || msg.Text == msgYouthPlayer {
			t.Errorf("season event in the panel: %q", msg.Text)
		}
	}
}
