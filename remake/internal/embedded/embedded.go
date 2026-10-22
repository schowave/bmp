// Package embedded holds the files of the original game when the binary
// is built with the tag "embeddata" (see the mise task remake:build). The
// files are extracted from bmp/ by cmd/bmpextract into
// internal/embedded/data, which git ignores.
package embedded

import "io/fs"

// FS returns the embedded game files, or nil if the binary has none.
func FS() fs.FS { return files }
