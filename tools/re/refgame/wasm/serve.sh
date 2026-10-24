#!/usr/bin/env bash
# Builds the WASM web root, exposes the js-dos command interface as
# window.ci (the driver needs it) and serves it on port 8000.
# With --driver it also runs driver.mjs and stops both together (Ctrl-C, or
# the driver's ["quit"] command).
# Usage: tools/re/refgame/wasm/serve.sh [--driver]   (from anywhere)
set -euo pipefail
HERE="$(cd "$(dirname "$0")" && pwd)"
ROOT="$(cd "$HERE/../../../.." && pwd)"
"$ROOT/images/wasm/build-bundle.sh" "$ROOT/images/wasm/dist"
sed -i.bak 's/if (event === "ci-ready") {/if (event === "ci-ready") { window.ci = ci;/' "$ROOT/images/wasm/dist/index.html"
rm -f "$ROOT/images/wasm/dist/index.html.bak"
if [ "${1:-}" != "--driver" ]; then
    exec python3 -m http.server 8000 -d "$ROOT/images/wasm/dist"
fi
python3 -m http.server 8000 -d "$ROOT/images/wasm/dist" >/dev/null 2>&1 &
server=$!
trap 'kill $server 2>/dev/null' EXIT
cd "$HERE" && node driver.mjs
