// Package embedded holds the files of the original game that the
// re-implementation needs, built into every binary: the pictures, MANA.DAT,
// the goal scenes and the fonts of BMMAIN.EXE. cmd/bmpextract extracts them
// from bmp/ into internal/embedded/data (mise run remake:extract).
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
