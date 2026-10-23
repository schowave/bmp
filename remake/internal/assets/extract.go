package assets

import (
	"encoding/json"
	"fmt"
	"maps"
	"os"
	"path"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
)

type fontJSON struct {
	Name    string
	Height  int
	Spacing int
	Remap   map[string]string // CP437 code -> glyph character, as decimal strings
	Glyphs  []glyphJSON
}

type glyphJSON struct {
	W    int
	Rows []string // '#' = pixel set
}

// EncodeFonts serialises fonts for FONTS.JSON, sorted by name so that an
// extract does not change from run to run.
func EncodeFonts(fonts map[string]*Font) ([]byte, error) {
	var out []fontJSON
	names := slices.Sorted(maps.Keys(fonts))
	for _, name := range names {
		f := fonts[name]
		fj := fontJSON{Name: f.Name, Height: f.Height, Spacing: f.Spacing, Remap: map[string]string{}}
		for k, v := range f.remap {
			fj.Remap[fmt.Sprint(k)] = fmt.Sprint(v)
		}
		for _, g := range f.Glyphs {
			gj := glyphJSON{W: g.W}
			for _, row := range g.Rows {
				var b strings.Builder
				for _, on := range row {
					if on {
						b.WriteByte('#')
					} else {
						b.WriteByte('.')
					}
				}
				gj.Rows = append(gj.Rows, b.String())
			}
			fj.Glyphs = append(fj.Glyphs, gj)
		}
		out = append(out, fj)
	}
	return json.MarshalIndent(out, "", "\t")
}

// DecodeFonts reads FONTS.JSON.
func DecodeFonts(data []byte) (map[string]*Font, error) {
	var in []fontJSON
	if err := json.Unmarshal(data, &in); err != nil {
		return nil, err
	}
	fonts := map[string]*Font{}
	for _, fj := range in {
		if len(fj.Glyphs) != glyphCount {
			return nil, fmt.Errorf("font %q: %d glyphs", fj.Name, len(fj.Glyphs))
		}
		f := &Font{Name: fj.Name, Height: fj.Height, Spacing: fj.Spacing, remap: map[byte]byte{}}
		for k, v := range fj.Remap {
			var kb, vb byte
			if _, err := fmt.Sscan(k, &kb); err != nil {
				return nil, fmt.Errorf("font %q: remap key %q: %w", fj.Name, k, err)
			}
			if _, err := fmt.Sscan(v, &vb); err != nil {
				return nil, fmt.Errorf("font %q: remap value %q: %w", fj.Name, v, err)
			}
			f.remap[kb] = vb
		}
		for i, gj := range fj.Glyphs {
			g := Glyph{W: gj.W}
			for _, row := range gj.Rows {
				r := make([]bool, len(row))
				for x := range row {
					r[x] = row[x] == '#'
				}
				g.Rows = append(g.Rows, r)
			}
			f.Glyphs[i] = g
		}
		fonts[fj.Name] = f
	}
	return fonts, nil
}

// Extract writes what the re-implementation needs from the original game
// src into dst, in open formats (see the files of Dir): the club database,
// the pictures as indexed PNGs, the palettes, the goal scenes and the fonts
// of BMMAIN.EXE.
func Extract(src *Original, dst string) error {
	write := func(name string, data []byte) error {
		out := filepath.Join(dst, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(out), 0o755); err != nil {
			return err
		}
		return os.WriteFile(out, data, 0o644)
	}
	mana, err := src.Mana()
	if err != nil {
		return err
	}
	data, err := EncodeClubs(mana)
	if err != nil {
		return err
	}
	if err := write(clubsFile, data); err != nil {
		return err
	}
	pals, err := src.Palettes()
	if err != nil {
		return err
	}
	if err := write(palettesFile, EncodePalettes(pals)); err != nil {
		return err
	}
	for _, upper := range src.files("PIC", ".VGA", ".CP") {
		base := strings.TrimPrefix(upper, "PIC/")
		ext := filepath.Ext(base)
		n, err := strconv.Atoi(strings.TrimSuffix(base, ext))
		if err != nil {
			return fmt.Errorf("%s: not a numbered picture", upper)
		}
		full := ext == ".CP"
		p, err := src.Pic(n, full)
		if err != nil {
			return err
		}
		data, err := EncodePic(p, &pals[0])
		if err != nil {
			return err
		}
		if err := write(picPath(n, full), data); err != nil {
			return err
		}
	}
	for _, upper := range src.files("TORE", ".T", ".V", ".TE", ".VE", ".TJ", ".VJ") {
		name := strings.TrimPrefix(upper, "TORE/")
		sc, err := src.Scene(name)
		if err != nil {
			return err
		}
		data, err := EncodeScene(sc)
		if err != nil {
			return err
		}
		if err := write(path.Join(sceneDir, name+".json"), data); err != nil {
			return err
		}
	}
	counts, err := src.SceneCounts()
	if err != nil {
		return err
	}
	if data, err = EncodeSceneCounts(counts); err != nil {
		return err
	}
	if err := write(path.Join(sceneDir, countsFile), data); err != nil {
		return err
	}
	fonts, err := src.Fonts()
	if err != nil {
		return err
	}
	if data, err = EncodeFonts(fonts); err != nil {
		return err
	}
	return write(fontFile, data)
}
