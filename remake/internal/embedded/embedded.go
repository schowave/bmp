// Package embedded holds the files of the original game that the
// re-implementation needs, built into every binary, in open formats: the
// club database (clubs.json), the pictures as indexed PNGs (pic/), the
// palettes, the goal scenes (scenes/) and the fonts of BMMAIN.EXE.
// cmd/bmpextract extracts them from bmp/ into internal/embedded/data (mise
// run remake:extract); assets.Dir reads them.
package embedded

import (
	"embed"
	"io/fs"
)

//go:embed data
var data embed.FS

var files = mustSub(data)

// FS returns the embedded game files.
func FS() fs.FS { return files }

func mustSub(f embed.FS) fs.FS {
	sub, err := fs.Sub(f, "data")
	if err != nil {
		panic(err)
	}
	return sub
}
