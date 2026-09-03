#!/usr/bin/env bash
# Build a .jsdos bundle for BMP (Bundesliga Manager Professional)
#
# The .jsdos format is a ZIP archive containing:
#   .jsdos/dosbox.conf  — DOSBox configuration
#   <game files>        — everything from bmp/
#
# Usage: cd wasm && ./build-bundle.sh

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"
GAME_DIR="$PROJECT_ROOT/bmp"
OUTPUT="$SCRIPT_DIR/bmp.jsdos"
TMPDIR_BUILD="$(mktemp -d)"

trap 'rm -rf "$TMPDIR_BUILD"' EXIT

echo "Building .jsdos bundle..."

# Copy game files
cp -r "$GAME_DIR/"* "$TMPDIR_BUILD/"

# Zielverzeichnis fuer Spielstaende, das die dosbox.conf als D: mountet. Muss im
# Bundle liegen, sonst schlaegt der Mount fehl. Zip verwirft leere Verzeichnisse
# nicht, ein .keep ist also nicht noetig, aber es macht den Zweck sichtbar.
mkdir -p "$TMPDIR_BUILD/SAVES"
echo "Spielstaende gehoeren hierher, gemountet als Laufwerk D:." > "$TMPDIR_BUILD/SAVES/LIESMICH.TXT"

# Create .jsdos config directory and copy dosbox.conf
mkdir -p "$TMPDIR_BUILD/.jsdos"
cp "$SCRIPT_DIR/dosbox.conf" "$TMPDIR_BUILD/.jsdos/dosbox.conf"

# Create ZIP with .jsdos extension
(cd "$TMPDIR_BUILD" && zip -r -9 "$OUTPUT" .)

# Copy VERSION file for the web UI
cp "$PROJECT_ROOT/VERSION" "$SCRIPT_DIR/version.txt"

echo "Created $OUTPUT ($(du -h "$OUTPUT" | cut -f1))"
