#!/bin/sh
# Orders the squads of the seasons without market values (1995/96 to
# 2003/04) by their games on transfermarkt.de: fetches the performance data
# of the clubs the seasons took a Transfermarkt squad for into $BMP_TM_DIR
# (default ~/tm-daten, with the cache of update.sh) and gives the seasons
# those squads. Run in remake/ by mise run remake:seasons-games [FROM [TO]].
set -eu
from=${1:-1995}
to=${2:-2003}
dir=${BMP_TM_DIR:-$HOME/tm-daten}
uv run --quiet --group transfermarkt python tools/transfermarkt/tm_games.py \
	--out "$dir" --seasons internal/seasons/data --from "$from" --to "$to"
go run ./cmd/bmpseasons squads -tm "$dir" -from "$from" -to "$to"
