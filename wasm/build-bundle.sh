#!/usr/bin/env bash
# Build the web root of the WASM image for BMP (Bundesliga Manager Professional):
# the .jsdos bundle plus the files the page serves next to it.
#
# The .jsdos format is a ZIP archive containing:
#   .jsdos/dosbox.conf  — DOSBox configuration
#   <game files>        — everything from bmp/
#
# Usage: wasm/build-bundle.sh <output dir>
# The Docker build writes to /public, `mise run wasm:build` to wasm/dist.

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"
GAME_DIR="$PROJECT_ROOT/bmp"
OUT_DIR="${1:?Usage: $0 <output dir>}"
mkdir -p "$OUT_DIR"
OUT_DIR="$(cd "$OUT_DIR" && pwd)"
OUTPUT="$OUT_DIR/bmp.jsdos"
rm -f "$OUTPUT"
TMPDIR_BUILD="$(mktemp -d)"

trap 'rm -rf "$TMPDIR_BUILD"' EXIT

echo "Building .jsdos bundle..."

# Copy game files
cp -r "$GAME_DIR/"* "$TMPDIR_BUILD/"

# Target directory for saves, which dosbox.conf mounts as D:. It has to be in the
# bundle, otherwise the mount fails. Zip does not drop empty directories, so a
# .keep is not needed, but it makes the purpose visible.
mkdir -p "$TMPDIR_BUILD/SAVES"
echo "Spielstaende gehoeren hierher, gemountet als Laufwerk D:." > "$TMPDIR_BUILD/SAVES/LIESMICH.TXT"

# Start script with a restart loop. When the player quit BMP, they used to land at
# the DOS prompt; now the game restarts. The marker line goes over the DOS console
# to the page, which then saves right away instead of waiting for its interval -
# on exit the game state has just been written, after all.
# CRLF because it is a DOS batch file.
printf '@ECHO OFF\r\n:TOP\r\nBMMAIN.EXE\r\nECHO ---BMP-BEENDET---\r\nGOTO TOP\r\n' > "$TMPDIR_BUILD/START.BAT"

# Create .jsdos config directory and copy dosbox.conf
mkdir -p "$TMPDIR_BUILD/.jsdos"
cp "$SCRIPT_DIR/dosbox.conf" "$TMPDIR_BUILD/.jsdos/dosbox.conf"

# Create ZIP with .jsdos extension
(cd "$TMPDIR_BUILD" && zip -q -r -9 "$OUTPUT" .)

# The page and what it serves next to the bundle. The help page and favicon live
# in web/ because both images serve them.
cp "$SCRIPT_DIR/index.html" "$OUT_DIR/index.html"
cp "$PROJECT_ROOT/VERSION" "$OUT_DIR/version.txt"
cp "$PROJECT_ROOT/web/hilfe.html" "$OUT_DIR/hilfe.html"
cp "$PROJECT_ROOT/web/favicon.png" "$OUT_DIR/favicon.png"

echo "Created $OUTPUT ($(du -h "$OUTPUT" | cut -f1))"
