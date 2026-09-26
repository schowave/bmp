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

# Zielverzeichnis fuer Spielstaende, das die dosbox.conf als D: mountet. Muss im
# Bundle liegen, sonst schlaegt der Mount fehl. Zip verwirft leere Verzeichnisse
# nicht, ein .keep ist also nicht noetig, aber es macht den Zweck sichtbar.
mkdir -p "$TMPDIR_BUILD/SAVES"
echo "Spielstaende gehoeren hierher, gemountet als Laufwerk D:." > "$TMPDIR_BUILD/SAVES/LIESMICH.TXT"

# Startskript mit Neustart-Schleife. Beendet der Spieler BMP, landete er vorher auf
# dem DOS-Prompt; jetzt startet das Spiel neu. Die Marker-Zeile geht ueber die
# DOS-Konsole an die Seite, die daraufhin sofort speichert, statt auf ihr Intervall zu
# warten - beim Beenden ist der Spielstand ja gerade frisch geschrieben.
# CRLF, weil es eine DOS-Batchdatei ist.
printf '@ECHO OFF\r\n:TOP\r\nBMMAIN.EXE\r\nECHO ---BMP-BEENDET---\r\nGOTO TOP\r\n' > "$TMPDIR_BUILD/START.BAT"

# Create .jsdos config directory and copy dosbox.conf
mkdir -p "$TMPDIR_BUILD/.jsdos"
cp "$SCRIPT_DIR/dosbox.conf" "$TMPDIR_BUILD/.jsdos/dosbox.conf"

# Create ZIP with .jsdos extension
(cd "$TMPDIR_BUILD" && zip -q -r -9 "$OUTPUT" .)

# Die Seite und was sie neben dem Bundle ausliefert. Hilfeseite und Favicon liegen
# in web/, weil beide Images sie ausliefern.
cp "$SCRIPT_DIR/index.html" "$OUT_DIR/index.html"
cp "$PROJECT_ROOT/VERSION" "$OUT_DIR/version.txt"
cp "$PROJECT_ROOT/web/hilfe.html" "$OUT_DIR/hilfe.html"
cp "$PROJECT_ROOT/web/favicon.png" "$OUT_DIR/favicon.png"

echo "Created $OUTPUT ($(du -h "$OUTPUT" | cut -f1))"
