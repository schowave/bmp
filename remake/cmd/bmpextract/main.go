// Command bmpextract writes the files the re-implementation needs from the
// original game into a directory, in open formats: the club database as
// JSON, the pictures as indexed PNGs, the palettes, the goal scenes and the
// fonts of BMMAIN.EXE. Every build embeds the default output directory.
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
	dir, err := assets.OpenOriginal(os.DirFS(*src))
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
