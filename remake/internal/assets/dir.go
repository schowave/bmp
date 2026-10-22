package assets

import (
	"fmt"
	"io/fs"
	"strings"
	"sync"
)

// Dir gives access to an installed copy of the game. File names are matched
// case-insensitively, like DOS does.
type Dir struct {
	fsys  fs.FS
	names map[string]string // upper-case path -> real path

	mu   sync.Mutex
	pics map[string]*Pic
}

// OpenDir indexes the game directory fsys (the directory with BMMAIN.EXE).
func OpenDir(fsys fs.FS) (*Dir, error) {
	d := &Dir{fsys: fsys, names: map[string]string{}, pics: map[string]*Pic{}}
	err := fs.WalkDir(fsys, ".", func(path string, e fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !e.IsDir() {
			d.names[strings.ToUpper(path)] = path
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	for _, need := range []string{"BMMAIN.EXE", "MANA.DAT", "PIC/1.PAL"} {
		if _, ok := d.names[need]; !ok {
			return nil, fmt.Errorf("%s not found: is this the game directory?", need)
		}
	}
	return d, nil
}

// ReadFile reads a file of the game, e.g. "PIC/22.CP".
func (d *Dir) ReadFile(name string) ([]byte, error) {
	real, ok := d.names[strings.ToUpper(strings.ReplaceAll(name, "\\", "/"))]
	if !ok {
		return nil, fmt.Errorf("%s: %w", name, fs.ErrNotExist)
	}
	return fs.ReadFile(d.fsys, real)
}

// Pic returns picture n: PIC/N.VGA, or the full-screen PIC/N.CP when full.
func (d *Dir) Pic(n int, full bool) (*Pic, error) {
	ext := "VGA"
	if full {
		ext = "CP"
	}
	name := fmt.Sprintf("PIC/%d.%s", n, ext)
	d.mu.Lock()
	defer d.mu.Unlock()
	if p, ok := d.pics[name]; ok {
		return p, nil
	}
	data, err := d.ReadFile(name)
	if err != nil {
		return nil, err
	}
	p, err := DecodePic(data)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", name, err)
	}
	d.pics[name] = p
	return p, nil
}

// Palettes reads PIC/1.PAL.
func (d *Dir) Palettes() ([]Palette, error) {
	data, err := d.ReadFile("PIC/1.PAL")
	if err != nil {
		return nil, err
	}
	pals := DecodePalettes(data)
	if len(pals) == 0 {
		return nil, fmt.Errorf("PIC/1.PAL: no palette")
	}
	return pals, nil
}

// Mana reads MANA.DAT.
func (d *Dir) Mana() (*Mana, error) {
	data, err := d.ReadFile("MANA.DAT")
	if err != nil {
		return nil, err
	}
	return ParseMana(data)
}

// MainImage reads and unpacks BMMAIN.EXE.
func (d *Dir) MainImage() (*Image, error) {
	data, err := d.ReadFile("BMMAIN.EXE")
	if err != nil {
		return nil, err
	}
	return Unexepack(data)
}
