// Command bmpextract copies the files the re-implementation needs from the
// original game into a directory: the pictures, MANA.DAT and the fonts of
// BMMAIN.EXE. Building with -tags embeddata embeds that directory.
package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/schowave/bmp/remake/internal/assets"
)

func main() {
	src := flag.String("data", "../bmp", "directory of the original game")
	dst := flag.String("out", "internal/embedded/data", "output directory")
	flag.Parse()
	dir, err := assets.OpenDir(os.DirFS(*src))
	if err == nil {
		err = os.RemoveAll(*dst)
	}
	if err == nil {
		err = assets.Extract(dir, *dst)
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, "bmpextract:", err)
		os.Exit(1)
	}
}
