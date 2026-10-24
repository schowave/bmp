#!/bin/sh
# Updates real seasons of the remake from transfermarkt.de: fetches their
# tables and squads into $BMP_TM_DIR (default ~/tm-daten, with the cache)
# and writes internal/seasons/data from them. Run in remake/ by
# mise run remake:seasons-update [FROM [TO]]; without years the running
# season. The seasons are fetched anew each time.
set -eu
# The running season starts in July.
running=$(date +%Y)
[ "$(date +%m)" -lt 7 ] && running=$((running - 1))
from=${1:-$running}
to=${2:-$from}
dir=${BMP_TM_DIR:-$HOME/tm-daten}
uv run --quiet --group transfermarkt python tools/transfermarkt/tm_json.py \
	--out "$dir" --from "$from" --to "$to" --refresh
go run ./cmd/bmpseasons update -tm "$dir" -from "$from" -to "$to"
