package ui

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/schowave/bmp/remake/internal/game"
)

// NewCupScreen shows the current round of a cup (f_17c2_1ccb).
func NewCupScreen(ss *Session, c int) Scene {
	s := ss.S
	cols := []Column{{"", 4, 130, Right}, {"", 136, 10, Center}, {"", 148, 120, Left}, {"", 270, 38, Center}}
	t := NewTableView(4, 24, 312, 168, cols)
	t.NoHeader = true
	t.RowH = 10
	round := s.Cups.Round[game.CupDFB]
	if c != game.CupDFB {
		round = s.Cups.Round[game.CupLandesmeister]
	}
	for i, p := range s.CupPairs(c) {
		h, a := s.Cups.Result[c][2*i], s.Cups.Result[c][2*i+1]
		res := ""
		if h != 0 || a != 0 {
			res = itoa(game.ResultOf(h)) + ":" + itoa(a)
			if h >= 20 {
				res += " n.E."
			} else if h >= 10 {
				res += " n.V."
			}
		}
		if c != game.CupDFB && (s.Cups.First[c][2*i] != 0 || s.Cups.First[c][2*i+1] != 0) {
			res = "(" + itoa(s.Cups.First[c][2*i]) + ":" + itoa(s.Cups.First[c][2*i+1]) + ")"
		}
		row := Row{Cells: []string{teamWithLeague(s, p.Home), "-", teamWithLeague(s, p.Away), res}}
		if s.ManagerOf(p.Home) != nil || s.ManagerOf(p.Away) != nil {
			row.Color = ColYellow
		}
		t.Rows = append(t.Rows, row)
	}
	sub := ""
	if round >= 1 && round <= 5 {
		sub = game.RoundNames[round]
	}
	if w := s.Cups.Holder[c]; w != 0 {
		sub += "   POKALVERTEIDIGER: " + s.Teams[w-1].Name
	}
	return &ListScreen{Title: game.CupNames[c], Sub: sub, Table: t}
}

func teamWithLeague(s *game.State, t int) string {
	if t >= game.NumGerman {
		return s.Teams[t].Name
	}
	return s.Teams[t].Name + " (" + itoa(game.LeagueOf(t)+1) + ")"
}

// FileScreen saves or loads a game.
type FileScreen struct {
	ss    *Session
	save  bool
	table *TableView
	files []string
}

// NewFileScreen creates the save or load screen.
func NewFileScreen(ss *Session, save bool) *FileScreen {
	cols := []Column{{"FILENAME", 4, 180, Left}, {"", 190, 110, Left}}
	f := &FileScreen{ss: ss, save: save, table: NewTableView(4, 24, 312, 168, cols)}
	f.scan()
	return f
}

func (f *FileScreen) scan() {
	f.files = nil
	f.table.Rows = nil
	entries, _ := os.ReadDir(f.ss.SaveDir)
	for _, e := range entries {
		if strings.HasSuffix(e.Name(), ".man.json") {
			name := strings.TrimSuffix(e.Name(), ".man.json")
			f.files = append(f.files, name)
			info, _ := e.Info()
			when := ""
			if info != nil {
				when = info.ModTime().Format("02.01.2006 15:04")
			}
			f.table.Rows = append(f.table.Rows, Row{Cells: []string{name, when}})
		}
	}
	if f.save {
		f.table.Rows = append(f.table.Rows, Row{Cells: []string{"< NEUER SPIELSTAND >"}, Color: ColYellow})
	}
}

func (f *FileScreen) path(name string) string {
	return filepath.Join(f.ss.SaveDir, name+".man.json")
}

func (f *FileScreen) Update(app *App) error {
	in := &app.Input
	left, _ := f.table.Update(in)
	name := f.ss.M().Name + ":"
	if left >= 0 {
		if f.save {
			def := "spielstand"
			if left < len(f.files) {
				def = f.files[left]
			}
			app.Push(NewTextInput("Spielstand Speichern", "FILENAME", def, 20, func(app *App, v string, ok bool) {
				v = strings.TrimSpace(v)
				if !ok || v == "" {
					return
				}
				if err := f.write(v); err != nil {
					app.Push(NewMessage(name, "Fehler beim Speichern: "+err.Error()))
					return
				}
				f.scan()
				app.Push(NewMessage(name, "Spielstand gespeichert."))
			}))
			return nil
		}
		data, err := os.ReadFile(f.path(f.files[left]))
		if err == nil {
			var s *game.State
			s, err = game.Load(data, f.ss.Mana)
			if err == nil {
				f.ss.S = s
				app.Reset(NewDriver(f.ss))
				return nil
			}
		}
		app.Push(NewMessage(name, "Kein Bundesliga Manager Prof. Spielstand. "+err.Error()))
		return nil
	}
	if MenuButtonHit(in) {
		app.Pop()
	}
	return nil
}

func (f *FileScreen) write(name string) error {
	data, err := f.ss.S.Save()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(f.ss.SaveDir, 0o755); err != nil {
		return err
	}
	return os.WriteFile(f.path(name), data, 0o644)
}

func (f *FileScreen) Draw(app *App, dst *ebiten.Image) {
	g := app.Gfx
	g.Background(dst)
	g.Panel(dst, 4, 3, 312, 17)
	title := "Spielstand Laden"
	if f.save {
		title = "Spielstand Speichern"
	}
	g.TextCentered(dst, FontNormal, 4, 7, 312, title, g.C(ColBeige))
	f.table.Draw(g, dst)
	g.Text(dst, FontSmall, 4, 196, "VERZEICHNIS: "+f.ss.SaveDir, g.C(ColLight))
	g.DrawMenuButton(dst)
}

// highscores reads and writes the high-score table of a game kind.
func highscores(dir, kind string) []game.HighscoreEntry {
	var hs []game.HighscoreEntry
	data, err := os.ReadFile(filepath.Join(dir, "high."+kind+".json"))
	if err == nil {
		_ = json.Unmarshal(data, &hs)
	}
	return hs
}

func saveHighscores(dir, kind string, hs []game.HighscoreEntry) {
	data, err := json.MarshalIndent(hs, "", " ")
	if err != nil {
		return
	}
	_ = os.MkdirAll(dir, 0o755)
	_ = os.WriteFile(filepath.Join(dir, "high."+kind+".json"), data, 0o644)
}

// recordHighscores enters every manager into the table (f_322b_2366(2)).
func recordHighscores(ss *Session) []game.HighscoreEntry {
	kind := ss.S.HighscoreKind()
	hs := highscores(ss.SaveDir, kind)
	for mi := range ss.S.Managers {
		e := ss.S.HighscoreEntry(mi)
		replaced := false
		for i := range hs {
			if hs[i].Manager == e.Manager && hs[i].Club == e.Club {
				hs[i] = e
				replaced = true
			}
		}
		if !replaced {
			hs = append(hs, e)
		}
	}
	sort.SliceStable(hs, func(a, b int) bool { return hs[a].Score > hs[b].Score })
	if len(hs) > 20 {
		hs = hs[:20]
	}
	saveHighscores(ss.SaveDir, kind, hs)
	return hs
}

// NewHighscoreScreen shows the high-score table.
func NewHighscoreScreen(ss *Session) Scene {
	hs := highscores(ss.SaveDir, ss.S.HighscoreKind())
	cur := ss.S.HighscoreEntry(ss.Mgr())
	return newHighscoreList(ss, hs, &cur)
}

func newHighscoreList(ss *Session, hs []game.HighscoreEntry, cur *game.HighscoreEntry) Scene {
	cols := []Column{{"", 4, 16, Right}, {"MANAGER", 24, 90, Left}, {"VEREIN", 118, 110, Left}, {"(M/D/E)", 230, 40, Center}, {"PUNKTE", 272, 34, Right}}
	t := NewTableView(4, 24, 312, 168, cols)
	t.RowH = 8
	shown := false
	add := func(e game.HighscoreEntry, color int) {
		t.Rows = append(t.Rows, Row{Cells: []string{itoa(len(t.Rows)+1) + ".", e.Manager, e.Club,
			itoa(e.Titles[0]) + "/" + itoa(e.Titles[1]) + "/" + itoa(e.Titles[2]), itoa(e.Score)}, Color: color})
	}
	for _, e := range hs {
		if cur != nil && !shown && cur.Score >= e.Score {
			add(*cur, ColYellow)
			shown = true
		}
		add(e, 0)
	}
	if cur != nil && !shown {
		add(*cur, ColYellow)
	}
	return &ListScreen{Title: "BUNDESLIGA-MANAGER HIGHSCORES", Sub: "LEVEL " + itoa(5-ss.S.Level), Table: t}
}

// OptionsScreen shows the game settings.
func NewOptionsScreen(ss *Session) Scene {
	return &simpleScreen{draw: func(app *App, dst *ebiten.Image) {
		g := app.Gfx
		s := ss.S
		g.Panel(dst, 20, 20, 280, 160)
		ende := " NIE."
		if s.EndYear < 3000 {
			ende = itoa(s.EndYear)
		}
		g.Heading(dst, 30, 28, "V2.0 - ENDE: "+ende+" (LEVEL "+itoa(5-s.Level)+")")
		lines := []string{
			"Spiel-Level " + itoa(5-s.Level) + " (0 = leicht, 5 = schwer)",
			"Manager: " + itoa(len(s.Managers)),
			"Saisons gespielt: " + itoa(s.Seasons),
			"",
			"Tempo der Spieltage: TEMPO-Knopf im Spieltag.",
			"Rechte Maustaste im Spieltag: sofort zum Ergebnis.",
		}
		for i, l := range lines {
			g.Text(dst, FontNormal, 30, 46+i*12, l, g.C(ColLight))
		}
	}}
}

// GameOver is shown when the chosen game length is over.
type GameOver struct {
	ss *Session
	hs []game.HighscoreEntry
}

// NewGameOver records the high scores and shows them.
func NewGameOver(ss *Session) Scene {
	return &GameOver{ss: ss, hs: recordHighscores(ss)}
}

func (o *GameOver) Update(app *App) error {
	if app.Input.Left || app.Input.Right || app.Input.Enter || app.Input.Escape {
		app.Reset(NewTitle().OnDone(func() Scene { return NewSetup(o.ss.Mana, o.ss.SaveDir) }))
	}
	return nil
}

func (o *GameOver) Draw(app *App, dst *ebiten.Image) {
	g := app.Gfx
	newHighscoreList(o.ss, o.hs, nil).Draw(app, dst)
	FillRect(dst, 100, 200, 120, 20, g.C(ColDarkRed))
	g.TextCentered(dst, FontBig, 100, 202, 120, "ENDE", g.C(ColBeige))
}
