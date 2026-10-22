//go:build embeddata

package embedded

import (
	"embed"
	"io/fs"
)

//go:embed data
var data embed.FS

var files = mustSub(data)

func mustSub(f embed.FS) fs.FS {
	sub, err := fs.Sub(f, "data")
	if err != nil {
		panic(err)
	}
	return sub
}
