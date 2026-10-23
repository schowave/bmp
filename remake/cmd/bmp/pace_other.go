//go:build !darwin

package main

import "github.com/schowave/bmp/remake/internal/ui"

// pace keeps the frames of Ebitengine: only macOS needs a ticker (see
// pace_darwin.go).
func pace(*ui.App) {}
