#!/bin/sh
# Renders every demo screen of the remake to PNG files (320x240), to look at
# a change of the screens or compare them with the original (docs/re/spec_ui.md).
# Usage: remake/tools/shots.sh [outdir]   (default: remake/shots, ignored by git)
# Needs a display: on macOS the screen awake (caffeinate -u), on Linux xvfb-run.
set -eu
cd "$(dirname "$0")/.."
out=${1:-shots}
mkdir -p "$out"
go build -o "$out/bmp" ./cmd/bmp
for d in $("$out/bmp" -demos); do
    "$out/bmp" -scale 1 -demo "$d" -shot "$out/$d.png" -shot-frame 10
done
rm "$out/bmp"
ls "$out"
