package ui

import (
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/schowave/bmp/remake/internal/assets"
	"github.com/schowave/bmp/remake/internal/game"
	"github.com/schowave/bmp/remake/internal/seasons"
)

// FileScreen is the screen to load or save a game (f_322b_2bbe, called by
// f_322b_120c "Spielstand Laden" and f_322b_07fe "Spielstand Speichern"):
// the directory in two columns of 18 names, the directory, the file name
// and the page, and buttons for the drives and LÖSCHEN. The remake lists
// the save directory; a game is saved as JSON in NAME.MAN.
type FileScreen struct {
	ss      *Session
	save    bool
	entries []fileEntry
	page    int    // index of the first entry shown (steps of 36)
	name    string // the text of the FILENAME field
	typing  bool
	buttons [8]Button
}

type fileEntry struct {
	file string // name in the directory
	show string // name as shown (upper case like DOS)
	dir  bool
}

// lastFileName is the file name of the last load or save (D:0x54a2); the
// screen starts with it.
var lastFileName string

// Geometry of the screen.
const (
	fileRows     = 18 // names per column
	filePerPage  = 2 * fileRows
	fileListY    = 41 // baseline of the first row
	fileRowH     = 7
	fileNameX    = 63
	fileNameMaxW = 140
)

// NewFileScreen creates the save or load screen.
func NewFileScreen(ss *Session, save bool) *FileScreen {
	f := &FileScreen{ss: ss, save: save, name: lastFileName}
	labels := []string{"A:", "B:", "C:", "D:", "E:", "F:", "G:", "LöSCHEN"}
	for i := range f.buttons {
		f.buttons[i] = Button{X: (i%4)*59 + 10, Y: (i/4)*13 + 187, W: 57, H: 12, Label: labels[i]}
	}
	f.scan()
	return f
}

// scan reads the directory: "." and "..", the directories, then the files.
func (f *FileScreen) scan() {
	f.entries = []fileEntry{{".", ".", true}, {"..", "..", true}}
	des, _ := os.ReadDir(f.ss.SaveDir)
	var dirs, files []fileEntry
	for _, e := range des {
		if strings.HasPrefix(e.Name(), ".") {
			continue
		}
		fe := fileEntry{file: e.Name(), show: strings.ToUpper(e.Name()), dir: e.IsDir()}
		if fe.dir {
			dirs = append(dirs, fe)
		} else {
			files = append(files, fe)
		}
	}
	byName := func(l []fileEntry) {
		sort.Slice(l, func(a, b int) bool { return l[a].show < l[b].show })
	}
	byName(dirs)
	byName(files)
	f.entries = append(append(f.entries, dirs...), files...)
	if f.page >= len(f.entries) {
		f.page = 0
	}
}

// isSave reports whether a name ends with .MAN (shown orange).
func isSave(show string) bool {
	return strings.HasSuffix(show, ".MAN")
}

// path returns the file of the name in the FILENAME field: an entry of
// the directory with that name, else the name itself.
func (f *FileScreen) path() string {
	for _, e := range f.entries {
		if !e.dir && e.show == f.name {
			return filepath.Join(f.ss.SaveDir, e.file)
		}
	}
	return filepath.Join(f.ss.SaveDir, filepath.Base(f.name))
}

// hover returns the column and row of the list under the pointer.
func hoverCell(in *Input) (col, row int, ok bool) {
	if in.X < 11 || in.X > 245 || in.Y < 37 || in.Y > 159 {
		return 0, 0, false
	}
	col = 0
	if in.X > 130 {
		col = 1
	}
	return col, (in.Y - 37) / fileRowH, true
}

func (f *FileScreen) Update(app *App) error {
	in := &app.Input
	if f.typing {
		f.typeName(app)
		return nil
	}
	// A right click above the lines turns the page.
	if in.Right && in.Y < 165 && len(f.entries) > filePerPage {
		f.page += filePerPage
		if f.page > len(f.entries) {
			f.page = 0
		}
		return nil
	}
	if col, row, ok := hoverCell(in); ok {
		if in.Left {
			if i := f.page + col*fileRows + row; i < len(f.entries) && !f.entries[i].dir {
				// A click on the name in the field confirms it.
				if f.entries[i].show == f.name {
					f.confirm(app)
					return nil
				}
				f.name = f.entries[i].show
			}
			return nil
		}
	}
	// The FILENAME row: left click types a new name, right click takes the
	// one in the field.
	if in.X > 10 && in.X < 246 && in.Y > 173 && in.Y < 183 {
		if in.Left {
			f.typing = true
			f.name = ""
			return nil
		}
		if in.Right && f.name != "" {
			f.confirm(app)
			return nil
		}
	}
	if MenuButtonHit(in) {
		app.Pop()
		return nil
	}
	// The drive buttons do nothing: the remake has only its save directory.
	if f.buttons[7].Hit(in) && f.name != "" {
		app.Push(NewQuestion(f.asker(), "FILE\n"+f.name+"\nWIRKLICH LöSCHEN ?", "MACH SCHON!", "OH NEIN!",
			func(app *App, yes bool) {
				if !yes {
					return
				}
				if err := os.Remove(f.path()); err != nil {
					app.Push(diskError())
					return
				}
				f.name = ""
				f.scan()
			}))
	}
	return nil
}

// typeName edits the FILENAME field (f_076b_07d7): it starts empty and
// ends with Enter.
func (f *FileScreen) typeName(app *App) {
	in := &app.Input
	g := app.Gfx
	for _, c := range in.Chars {
		// no path separators: saves stay in the save directory
		if c <= 0x20 && c != ' ' || c >= 0x7f || c == '/' || c == '\\' || c == ':' {
			continue
		}
		n := f.name + strings.ToUpper(string(c))
		if g.TextWidth(FontSmall, n) <= fileNameMaxW {
			f.name = n
		}
	}
	if in.Backspace && f.name != "" {
		f.name = f.name[:len(f.name)-1]
	}
	if in.Escape {
		f.typing = false
		f.name = ""
	}
	if !in.Enter {
		return
	}
	f.typing = false
	if f.name == "" {
		return
	}
	f.name = strings.ReplaceAll(f.name, " ", "_")
	if !strings.HasSuffix(f.name, ".MAN") {
		f.name += ".MAN"
	}
	f.confirm(app)
}

// confirm loads or saves the file in the FILENAME field. The screen closes
// afterwards; a question or an error is shown over it first.
func (f *FileScreen) confirm(app *App) {
	path := f.path()
	if f.save {
		if _, err := os.Stat(path); err == nil {
			app.Push(NewQuestion(f.ss.M().Name, "FILE EXISTIERT\nBEREITS.\nÜBERSCHREIBEN ?", "ACH BITTE !", "BLOß NICHT !",
				func(app *App, yes bool) {
					if yes {
						f.write(app, path)
					} else {
						app.Pop()
					}
				}))
			return
		}
		f.write(app, path)
		return
	}
	data, err := os.ReadFile(path)
	if err != nil {
		f.fail(app, diskError())
		return
	}
	s, err := LoadGame(data, f.ss.Mana)
	if err != nil {
		f.fail(app, NewAlert("Kein", "Bundesliga Manager Prof.", "Spielstand.", "!!!"))
		return
	}
	lastFileName = f.name
	f.ss.S = s
	app.Reset(NewDriver(f.ss))
}

// LoadGame restores a saved game with the clubs it was started with: those
// of the original (orig) or of its real season.
func LoadGame(data []byte, orig *assets.Mana) (*game.State, error) {
	id, err := game.SaveSeason(data)
	if err != nil {
		return nil, err
	}
	m, err := seasons.ManaOr(orig, id)
	if err != nil {
		return nil, err
	}
	return game.Load(data, m)
}

// asker is the name over a question: the manager whose turn it is, none
// before a game runs (the load screen of the setup).
func (f *FileScreen) asker() string {
	if f.ss.S == nil {
		return ""
	}
	return f.ss.M().Name
}

// fail shows an alert over the screen and closes both together.
func (f *FileScreen) fail(app *App, a *AlertBox) {
	a.Done = func(app *App) { app.Pop() }
	app.Push(a)
}

func (f *FileScreen) write(app *App, path string) {
	data, err := f.ss.S.Save()
	if err == nil {
		err = os.MkdirAll(filepath.Dir(path), 0o755)
	}
	if err == nil {
		err = os.WriteFile(path, data, 0o644)
	}
	if err != nil {
		f.fail(app, diskError())
		return
	}
	lastFileName = f.name
	app.Pop()
}

func diskError() *AlertBox {
	return NewAlert("Achtung !", "Fehler beim", "Diskettenzugriff !", "Vorgang abgebrochen.")
}

func (f *FileScreen) Draw(app *App, dst *ebiten.Image) {
	g := app.Gfx
	in := &app.Input
	g.Background(dst)
	title := "SPIELSTAND LADEN"
	if f.save {
		title = "SPIELSTAND SPEICHERN"
	}
	g.TextMid(dst, FontBig, 0, 320, 20, title, ColBeige, ColBlack)
	g.Window(dst, 6, 33, 247, 214, ColBlue)
	g.DrawMenuButton(dst)
	for y := 163; y < 184; y += 10 {
		FillRect(dst, 7, y, 240, 1, g.PalColor(ColBeige))
	}
	// The bar under the pointer is drawn with XOR (f_3930_052d mode 2):
	// blue becomes yellow and every colour c becomes c^31.
	hc, hr, hover := hoverCell(in)
	if f.typing {
		hover = false
	}
	if hover {
		FillRect(dst, hc*120+10, hr*fileRowH+37, 116, 5, g.PalColor(ColBlue^31))
	}
	for k := 0; k < filePerPage && f.page+k < len(f.entries); k++ {
		e := f.entries[f.page+k]
		col, row := k/fileRows, k%fileRows
		c := ColLight
		switch {
		case e.dir:
			c = ColGrey3
		case isSave(e.show):
			c = ColOrange
		}
		if hover && col == hc && row == hr {
			c ^= 31
		}
		g.TextAt(dst, FontSmall, 10+col*120, fileListY+row*fileRowH, g.Fit(FontSmall, e.show, 115), c, NoShadow)
	}
	dir := strings.ToUpper(f.ss.SaveDir)
	g.TextAt(dst, FontSmall, 10, 170, g.Fit(FontSmall, "VERZEICHNIS: "+dir, 230), ColLight, NoShadow)
	g.TextAt(dst, FontSmall, 10, 180, "FILENAME      :", ColLight, NoShadow)
	g.TextAt(dst, FontSmall, fileNameX, 180, f.name, ColLight, NoShadow)
	if f.typing {
		FillRect(dst, fileNameX+g.TextWidth(FontSmall, f.name)+2, 176, 3, 5, g.PalColor(ColLight))
	}
	g.TextAt(dst, FontSmall, 210, 180, "SEITE: "+itoa(f.page/filePerPage+1), ColGrey2, NoShadow)
	for i := range f.buttons {
		drawDialogButton(app, dst, &f.buttons[i])
	}
}
