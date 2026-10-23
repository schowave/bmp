package assets

import (
	"fmt"
	"io/fs"
	"slices"
	"strings"
)

// Original gives access to an installed copy of the original game, the
// source of an extract (Extract). File names are matched case-insensitively,
// like DOS does.
type Original struct {
	fsys  fs.FS
	names map[string]string // upper-case path -> real path
}

// OpenOriginal indexes the directory fsys of the original game.
func OpenOriginal(fsys fs.FS) (*Original, error) {
	o := &Original{fsys: fsys, names: map[string]string{}}
	err := fs.WalkDir(fsys, ".", func(path string, e fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !e.IsDir() {
			o.names[strings.ToUpper(path)] = path
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	for _, need := range []string{"MANA.DAT", "PIC/1.PAL", "BMMAIN.EXE"} {
		if _, ok := o.names[need]; !ok {
			return nil, fmt.Errorf("%s not found: is this the game directory?", need)
		}
	}
	return o, nil
}

// ReadFile reads a file of the game, e.g. "PIC/22.CP".
func (o *Original) ReadFile(name string) ([]byte, error) {
	real, ok := o.names[strings.ToUpper(strings.ReplaceAll(name, "\\", "/"))]
	if !ok {
		return nil, fmt.Errorf("%s: %w", name, fs.ErrNotExist)
	}
	return fs.ReadFile(o.fsys, real)
}

// files returns the upper-case paths of the files in directory dir with
// one of the extensions exts, sorted.
func (o *Original) files(dir string, exts ...string) []string {
	var out []string
	for upper := range o.names {
		if !strings.HasPrefix(upper, dir+"/") {
			continue
		}
		for _, ext := range exts {
			if strings.HasSuffix(upper, ext) {
				out = append(out, upper)
				break
			}
		}
	}
	slices.Sort(out)
	return out
}

// Pic decodes picture n: PIC/N.VGA, or the full-screen PIC/N.CP when full.
func (o *Original) Pic(n int, full bool) (*Pic, error) {
	ext := "VGA"
	if full {
		ext = "CP"
	}
	name := fmt.Sprintf("PIC/%d.%s", n, ext)
	data, err := o.ReadFile(name)
	if err != nil {
		return nil, err
	}
	p, err := ParsePic(data)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", name, err)
	}
	return p, nil
}

// Palettes reads PIC/1.PAL.
func (o *Original) Palettes() ([]Palette, error) {
	data, err := o.ReadFile("PIC/1.PAL")
	if err != nil {
		return nil, err
	}
	pals := ParsePalettes(data)
	if len(pals) == 0 {
		return nil, fmt.Errorf("PIC/1.PAL: no palette")
	}
	return pals, nil
}

// Mana reads MANA.DAT.
func (o *Original) Mana() (*Mana, error) {
	data, err := o.ReadFile("MANA.DAT")
	if err != nil {
		return nil, err
	}
	return ParseMana(data)
}

// MainImage reads and unpacks BMMAIN.EXE.
func (o *Original) MainImage() (*Image, error) {
	data, err := o.ReadFile("BMMAIN.EXE")
	if err != nil {
		return nil, err
	}
	return Unexepack(data)
}

// Fonts returns the bitmap fonts unpacked from BMMAIN.EXE.
func (o *Original) Fonts() (map[string]*Font, error) {
	img, err := o.MainImage()
	if err != nil {
		return nil, err
	}
	return Fonts(img)
}

// Scene reads and parses TORE\<name>, e.g. "10.T".
func (o *Original) Scene(name string) (*Scene, error) {
	data, err := o.ReadFile("TORE/" + name)
	if err != nil {
		return nil, err
	}
	s, err := ParseScene(data)
	if err != nil {
		return nil, fmt.Errorf("TORE/%s: %w", name, err)
	}
	return s, nil
}

// SceneCounts reads TORE\ANZAHL.
func (o *Original) SceneCounts() (SceneCounts, error) {
	data, err := o.ReadFile("TORE/ANZAHL")
	if err != nil {
		return SceneCounts{}, err
	}
	return ParseSceneCounts(data)
}
