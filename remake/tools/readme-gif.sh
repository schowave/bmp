#!/bin/sh
# Renders a tour of the remake's screens into the animated GIF shown in the
# top README (docs/screens.gif), at twice the size of the 320x240 canvas.
# Usage: remake/tools/readme-gif.sh [out]   (default: ../docs/screens.gif)
# The manager plays 1.FC Kaiserslautern (-demo-team 0).
# Needs ImageMagick and a display, as tools/shots.sh does.
set -eu
cd "$(dirname "$0")/.."
out=${1:-../docs/screens.gif}
tmp=$(mktemp -d)
trap 'rm -rf "$tmp"' EXIT
go build -o "$tmp/bmp" ./cmd/bmp
frames=""
for d in title setup seasons menu squad training stadium sponsor transfer flag match newspaper table finance; do
    "$tmp/bmp" -scale 1 -demo "$d" -demo-team 0 -shot "$tmp/$d.png" -shot-frame 10
    frames="$frames $tmp/$d.png"
done
# shellcheck disable=SC2086 # frames is a list of paths without spaces
magick -delay 80 -loop 0 $frames -filter point -resize 200% \
    -layers Optimize "$out"
ls -l "$out"
