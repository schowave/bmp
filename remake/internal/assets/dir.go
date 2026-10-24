package assets

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"path"
	"sync"
)

// Files of an extract (Extract). Pictures are indexed PNGs whose pixels
// are the palette indices of the original; everything else is JSON.
const (
	clubsFile    = "clubs.json"    // MANA.DAT
	palettesFile = "palettes.json" // PIC/1.PAL
	fontFile     = "fonts.json"    // the fonts inside BMMAIN.EXE
	picDir       = "pic"           // pic/N.png = PIC/N.VGA, pic/full/N.png = PIC/N.CP
	sceneDir     = "scenes"        // scenes/N.T.json = TORE/N.T, scenes/counts.json = TORE/ANZAHL
	countsFile   = "counts.json"
)

// Dir gives access to an extract of the original game: the files built
// into the remake, or a directory written by Extract.
type Dir struct {
	fsys fs.FS
	more []fs.FS // further pictures (AddPics), searched after fsys

	mu   sync.Mutex
	pics map[string]*Pic
}

// OpenDir opens the extract in fsys.
func OpenDir(fsys fs.FS) (*Dir, error) {
	for _, need := range []string{clubsFile, palettesFile, fontFile} {
		if _, err := fs.Stat(fsys, need); err == nil {
			continue
		}
		if _, err := fs.Stat(fsys, "BMMAIN.EXE"); err == nil {
			return nil, errors.New("this is the original game: extract it first (go run ./cmd/bmpextract -data DIR -out OUT)")
		}
		return nil, fmt.Errorf("%s not found: is this an extract of the game?", need)
	}
	return &Dir{fsys: fsys, pics: map[string]*Pic{}}, nil
}

// AddPics adds the pictures of fsys (pic/N.png) to those of the extract:
// the logos of the clubs of real seasons. The extract wins where both have
// a picture.
func (d *Dir) AddPics(fsys fs.FS) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.more = append(d.more, fsys)
}

// ReadFile reads a file of the extract, e.g. "clubs.json".
func (d *Dir) ReadFile(name string) ([]byte, error) {
	return fs.ReadFile(d.fsys, name)
}

// picPath is the file of picture n in an extract.
func picPath(n int, full bool) string {
	if full {
		return fmt.Sprintf("%s/full/%d.png", picDir, n)
	}
	return fmt.Sprintf("%s/%d.png", picDir, n)
}

// Pic returns picture n: PIC/N.VGA of the original, or the full-screen
// PIC/N.CP when full.
func (d *Dir) Pic(n int, full bool) (*Pic, error) {
	name := picPath(n, full)
	d.mu.Lock()
	defer d.mu.Unlock()
	if p, ok := d.pics[name]; ok {
		return p, nil
	}
	data, err := d.ReadFile(name)
	for _, f := range d.more {
		if !errors.Is(err, fs.ErrNotExist) {
			break
		}
		data, err = fs.ReadFile(f, name)
	}
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

// Palettes reads the palettes of PIC/1.PAL.
func (d *Dir) Palettes() ([]Palette, error) {
	data, err := d.ReadFile(palettesFile)
	if err != nil {
		return nil, err
	}
	pals, err := DecodePalettes(data)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", palettesFile, err)
	}
	return pals, nil
}

// Mana reads the club database.
func (d *Dir) Mana() (*Mana, error) {
	data, err := d.ReadFile(clubsFile)
	if err != nil {
		return nil, err
	}
	m, err := DecodeClubs(data)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", clubsFile, err)
	}
	return m, nil
}

// Fonts returns the bitmap fonts.
func (d *Dir) Fonts() (map[string]*Font, error) {
	data, err := d.ReadFile(fontFile)
	if err != nil {
		return nil, err
	}
	return DecodeFonts(data)
}

// Scene reads goal scene name, e.g. "10.T".
func (d *Dir) Scene(name string) (*Scene, error) {
	file := path.Join(sceneDir, name+".json")
	data, err := d.ReadFile(file)
	if err != nil {
		return nil, err
	}
	s, err := DecodeScene(data)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", file, err)
	}
	return s, nil
}

// SceneCounts reads the numbers of goal scenes.
func (d *Dir) SceneCounts() (SceneCounts, error) {
	data, err := d.ReadFile(path.Join(sceneDir, countsFile))
	if err != nil {
		return SceneCounts{}, err
	}
	return DecodeSceneCounts(data)
}

// marshalJSON encodes v as indented JSON without escaping <, > and &.
func marshalJSON(v any) ([]byte, error) {
	var b bytes.Buffer
	enc := json.NewEncoder(&b)
	enc.SetEscapeHTML(false)
	enc.SetIndent("", "\t")
	if err := enc.Encode(v); err != nil {
		return nil, err
	}
	return b.Bytes(), nil
}

// unmarshalJSON decodes data into v and rejects unknown fields, so a typo
// in a hand-edited file is an error instead of a missing value.
func unmarshalJSON(data []byte, v any) error {
	dec := json.NewDecoder(bytes.NewReader(data))
	dec.DisallowUnknownFields()
	return dec.Decode(v)
}
