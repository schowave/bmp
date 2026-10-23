package game

import (
	"strings"
	"testing"
)

func newsWidth(s string) int { return 4 * len([]rune(s)) }

func TestNewsExpand(t *testing.T) {
	s := &State{}
	s.Teams[1].Name = "VFB STUTTGART"
	s.Teams[2].Name = "VFL WOLFSBURG"
	s.Managers = []*Manager{{Name: "TEST"}}
	md := &MatchDay{s: s}
	rep := &newsReport{own: 1, opp: 2, gf: 0, ga: 3, state: 2, attendance: 12000}
	rep.sel = [6]int{0, 1, 4, 3, 1, 2}
	ex := &newsExpander{md: md, rep: rep}
	l1, p := ex.expand(newsHeadlines[60], 0)
	l2, _ := ex.expand(newsHeadlines[60], p)
	if l1 != "VFB STUTTGART OHNE CHANCE IM SPIEL" || l2 != "GEGEN VFL WOLFSBURG" {
		t.Errorf("headline %q / %q", l1, l2)
	}
	got, _ := ex.expand(newsParagraphs[7], 0)
	want := "DIE WIRKLICH ANSPRECHENDE PARTIE ZWISCHEN VFB STUTTGART GEGEN VFL WOLFSBURG ENDETE 0:3 GEGEN VFB STUTTGART."
	if got != want {
		t.Errorf("paragraph %q", got)
	}
	got, _ = ex.expand(newsParagraphs[12], 0) // nested choices
	if want := "DIE 12000 FANS IM STADION BETRAUERTEN DIE UNGLÜCKLICHE NIEDERLAGE IHRER MANNSCHAFT."; got != want {
		t.Errorf("nested %q", got)
	}
	rep.sel[5] = 2
	got, _ = ex.expand(newsParagraphs[107], 0)
	if !strings.Contains(got, "BESSEREN MANNSCHAFT") || strings.Contains(got, "%") || strings.Contains(got, "#") {
		t.Errorf("random choices %q", got)
	}
}

func TestNewsWriterJustifies(t *testing.T) {
	w := &newsWriter{width: newsWidth}
	w.start(40, 0, 10)
	for _, s := range strings.Fields("AA BB CC DD EE FF") {
		w.word(s, false)
	}
	w.flush(false)
	if len(w.lines) != 2 || w.lines[0].Text != "AA  BB  CC" || w.lines[1].Text != "DD EE FF" || w.lines[1].Y != 17 {
		t.Errorf("lines %+v", w.lines)
	}
}

func TestNewspaperAfterMatchday(t *testing.T) {
	m := mana(t)
	s, err := NewGame(m, Options{Managers: []NewManager{{Name: "TEST", Team: 9}}, Level: 2, Seed: 5})
	if err != nil {
		t.Fatal(err)
	}
	s.Start()
	papers := 0
	for steps := 0; steps < 400 && papers < 12; steps++ {
		switch s.Loop.Phase {
		case PhaseTurn:
			s.EndTurn()
		case PhaseMatches:
			for md := s.NextMatchDay(); md != nil; md = s.NextMatchDay() {
				md.Run()
				np := md.Newspaper(0, newsWidth)
				if (np != nil) != (md.cupMask == 0 && md.humans[0].plays) {
					t.Fatalf("newspaper %v for cup mask %d", np != nil, md.cupMask)
				}
				if np == nil {
					continue
				}
				papers++
				if np.Photo < 200 || np.Photo > 229 || np.Headline[0] == "" || np.Headline[1] == "" || len(np.Article) < 5 {
					t.Fatalf("newspaper %+v", np)
				}
				for _, l := range append(np.Article, np.Box...) {
					if strings.ContainsAny(l.Text, "%#") {
						t.Errorf("unexpanded template: %q", l.Text)
					}
				}
				if len(np.Grades) == 0 || len(np.Grades) > 11 {
					t.Errorf("%d grades", len(np.Grades))
				}
				for _, g := range np.Grades {
					if g.Grade < 1 || g.Grade > 6 {
						t.Errorf("grade %+v", g)
					}
				}
				if papers == 1 {
					t.Logf("%s / %s", np.Headline[0], np.Headline[1])
					for _, l := range append(np.Article, np.Box...) {
						t.Logf("%3d %v %s", l.Y, l.Split, l.Text)
					}
				}
			}
		}
	}
	if papers == 0 {
		t.Fatal("no newspaper")
	}
	s.Display.Newspaper = false
}
