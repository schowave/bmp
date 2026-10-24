package ui

import (
	"testing"

	"github.com/schowave/bmp/remake/internal/game"
)

// The driver shows the queued notices in the order of the queue: a trophy
// queued before a flag comes before it, and a switched-off draw is skipped.
func TestDriverShowsNoticesInOrder(t *testing.T) {
	app, ss := leagueSession(t)
	s := ss.S
	s.TakeNotices()
	s.Display.DFB = false
	s.AddNotice(game.Notice{Kind: game.NoticeTrophy, Manager: 0, Lines: []string{"A", "B", "C"}})
	s.AddNotice(game.Notice{Kind: game.NoticeDraw, Manager: -1, Cup: int(game.CupDFB)})
	s.AddNotice(game.Notice{Kind: game.NoticeFlag, Manager: -1, Title: "SAISONENDE"})
	step(t, app, Input{})
	if _, ok := top(app).(*trophyScreen); !ok {
		t.Fatalf("first %T, want the trophy", top(app))
	}
	step(t, app, click(160, 120))
	if f, ok := top(app).(*flagScreen); !ok || f.text != "SAISONENDE" {
		t.Fatalf("then %T, want the flag SAISONENDE", top(app))
	}
}
