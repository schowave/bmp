#!/usr/bin/env bash
# Smoke test for the WASM image: starts it, checks that the page and the bundle are
# served, that a save survives a PUT and a GET byte for byte, and that the server
# rejects what it should. Needs no browser; the game itself is not started.
#
# Usage: wasm/smoke.sh [image]   (default bmp-wasm, what `mise run wasm:docker` builds)
# Runs podman, or the command in CONTAINER_CLI (CI sets docker).

set -euo pipefail

IMAGE="${1:-bmp-wasm}"
CLI="${CONTAINER_CLI:-podman}"
NAME=bmp-wasm-smoke
PORT=18091
BASE="http://localhost:$PORT"
TMP="$(mktemp -d)"

cleanup() {
    "$CLI" rm -f "$NAME" >/dev/null 2>&1 || true
    rm -rf "$TMP"
}
trap cleanup EXIT

failed=0
fail() { echo "FAILED: $*" >&2; failed=1; }

# Status code of a request; curl exits 0 on any HTTP status, so -f is left out.
status() { curl -s -o /dev/null -w '%{http_code}' "$@"; }

expect() {
    local want="$1" what="$2"; shift 2
    local got
    got="$(status "$@")"
    if [ "$got" = "$want" ]; then echo "ok   $what ($got)"; else fail "$what: HTTP $got, expected $want"; fi
}

"$CLI" rm -f "$NAME" >/dev/null 2>&1 || true
# No bind mount: the anonymous volume from the image's VOLUME /data belongs to the
# unprivileged user, as it has to for saves to work.
"$CLI" run -d --name "$NAME" -p "$PORT:8080" "$IMAGE" >/dev/null

for _ in $(seq 1 30); do
    [ "$(status "$BASE/version.txt")" = 200 ] && break
    sleep 1
done

expect 200 "page"          "$BASE/"
expect 200 "bundle"        "$BASE/bmp.jsdos"
expect 200 "version"       "$BASE/version.txt"
expect 200 "help page"     "$BASE/hilfe"
expect 404 "unknown slot"  "$BASE/api/saves/smoke-test"
expect 400 "invalid name"  "$BASE/api/saves/..%2Fetc"

head -c 100000 /dev/urandom > "$TMP/save"
expect 204 "save"          -X PUT --data-binary "@$TMP/save" "$BASE/api/saves/smoke-test"
curl -s "$BASE/api/saves/smoke-test" -o "$TMP/back"
if cmp -s "$TMP/save" "$TMP/back"; then echo "ok   save read back identically"; else fail "save read back differs"; fi

head -c $((4 * 1024 * 1024 + 1)) /dev/zero > "$TMP/big"
expect 413 "save over 4 MB" -X PUT --data-binary "@$TMP/big" "$BASE/api/saves/smoke-test"

# The command HEALTHCHECK runs, so a missing wget in a new base image shows up here.
if "$CLI" exec "$NAME" wget -q -O /dev/null http://localhost:8080/version.txt; then
    echo "ok   healthcheck command"
else
    fail "healthcheck command"
fi

if [ "$("$CLI" exec "$NAME" id -u)" != 0 ]; then echo "ok   server runs unprivileged"; else fail "server runs as root"; fi

if [ "$failed" != 0 ]; then
    "$CLI" logs "$NAME" >&2 || true
    exit 1
fi
echo "All checks passed"
