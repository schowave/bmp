package assets

import (
	"encoding/json"
	"fmt"
	"maps"
	"os"
	"path/filepath"
	"slices"
	"strings"
)

// fontFile holds the fonts of BMMAIN.EXE in an extract.
const fontFile = "FONTS.JSON"

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
			fmt.Sscan(k, &kb)
			fmt.Sscan(v, &vb)
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

// Extract copies what the re-implementation needs from the game directory
// src into dst: the pictures, MANA.DAT and the fonts of BMMAIN.EXE as
// FONTS.JSON.
func Extract(src *Dir, dst string) error {
	for upper, real := range src.names {
		keep := upper == "MANA.DAT" || strings.HasPrefix(upper, "TORE/") || (strings.HasPrefix(upper, "PIC/") &&
			(strings.HasSuffix(upper, ".VGA") || strings.HasSuffix(upper, ".CP") || strings.HasSuffix(upper, ".PAL")))
		if !keep {
			continue
		}
		data, err := src.ReadFile(real)
		if err != nil {
			return err
		}
		out := filepath.Join(dst, filepath.FromSlash(upper))
		if err := os.MkdirAll(filepath.Dir(out), 0o755); err != nil {
			return err
		}
		if err := os.WriteFile(out, data, 0o644); err != nil {
			return err
		}
	}
	fonts, err := src.Fonts()
	if err != nil {
		return err
	}
	data, err := EncodeFonts(fonts)
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(dst, fontFile), data, 0o644)
}
