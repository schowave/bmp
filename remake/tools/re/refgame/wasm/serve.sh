#!/usr/bin/env bash
# Builds the WASM web root, exposes the js-dos command interface as
# window.ci (the driver needs it) and serves it on port 8000.
# Usage: remake/tools/re/refgame/wasm/serve.sh   (from the repository root)
set -euo pipefail
ROOT="$(cd "$(dirname "$0")/../../../../.." && pwd)"
"$ROOT/wasm/build-bundle.sh" "$ROOT/wasm/dist"
sed -i.bak 's/if (event === "ci-ready") {/if (event === "ci-ready") { window.ci = ci;/' "$ROOT/wasm/dist/index.html"
rm -f "$ROOT/wasm/dist/index.html.bak"
exec python3 -m http.server 8000 -d "$ROOT/wasm/dist"
