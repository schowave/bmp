#!/bin/sh
# Drives the remake for a playtest through its control mode (bmp -control).
# Every command prints what the game shows afterwards (ui.App.Describe):
# the scene on top, its text, date, place, money, messages.
#
#   playtest.sh start [BMP FLAGS]  build and start, e.g. -load SAVE; TPS=600 default
#   playtest.sh send < SCRIPT      play a script (internal/ui/script.go)
#   playtest.sh do 'CMD' ...       play the given script lines
#   playtest.sh advance [N]        click through a match day to the main menu
#   playtest.sh turn [N]           end N turns, each with its match days
#   playtest.sh status             what the game shows
#   playtest.sh stop
#
# advance stops with exit status 1 at a scene it does not know how to leave.
# State, saves and the log are in $BMP_PLAYTEST (default $TMPDIR/bmp-playtest).
# Needs a display: on macOS the screen awake (start runs caffeinate).
set -eu
cd "$(dirname "$0")/.."
dir=${BMP_PLAYTEST:-${TMPDIR:-/tmp}/bmp-playtest}
ctl=$dir/ctl

send() {
	cat >"$ctl/cmd.tmp"
	rm -f "$ctl/done"
	mv "$ctl/cmd.tmp" "$ctl/cmd"
	i=0
	while [ ! -f "$ctl/done" ]; do
		sleep 0.05
		i=$((i + 1))
		if [ $i -gt 1200 ]; then
			echo "error: no answer after 60 s (see $dir/log)" >&2
			exit 2
		fi
	done
	cat "$ctl/done"
}

do_lines() { printf '%s\n' "$@" | send; }

scene() { sed -n 's/^scene: //p'; }

advance() {
	n=${1:-100}
	st=$(do_lines "wait 1")
	while [ "$n" -gt 0 ]; do
		n=$((n - 1))
		case $(echo "$st" | scene) in
		*MainMenu) echo "$st"; return 0 ;;
		# A click plays the rest of the period at once.
		*MatchScreen) st=$(do_lines "click 160 6" "wait 30") ;;
		*CupScreen | *ResultsScreen | *TableScreen | *ListScreen) st=$(do_lines "click button.menu" "wait 30") ;;
		*NewspaperScene) st=$(do_lines "click 160 120" "wait 30") ;;
		# Substitutions after a card or an injury: keep the team.
		*SquadScreen) st=$(do_lines "click lineup.exit" "wait 30") ;;
		*AlertBox) echo "$st" | sed -n 's/^text: /alert: /p'; st=$(do_lines "click box.0" "wait 30") ;;
		*MessageBox)
			case $st in
			*AUSLOSUNG*) st=$(do_lines "click box.1" "wait 60") ;; # KEIN GEDANKE
			*) echo "$st"; return 1 ;;
			esac ;;
		*) echo "$st"; return 1 ;;
		esac
	done
	echo "$st"
	return 1
}

case ${1:-} in
start)
	shift
	mkdir -p "$ctl" "$dir/saves"
	rm -f "$ctl/cmd" "$ctl/done"
	go build -o "$dir/bmp" ./cmd/bmp
	"$dir/bmp" -scale 2 -control "$ctl" -tps "${TPS:-600}" -saves "$dir/saves" "$@" >"$dir/log" 2>&1 &
	echo $! >"$dir/pid"
	if command -v caffeinate >/dev/null; then caffeinate -u -w "$(cat "$dir/pid")" & fi
	do_lines "wait 30"
	;;
send) send ;;
do) shift; do_lines "$@" ;;
status) do_lines "wait 1" ;;
advance) advance "${2:-100}" ;;
turn)
	k=${2:-1}
	while [ "$k" -gt 0 ]; do
		k=$((k - 1))
		case $(do_lines "wait 1") in
		*"messages open"*) do_lines "click messages.close" "wait 10" >/dev/null ;;
		esac
		do_lines "click menu.calendar" "wait 60" >/dev/null
		advance 200 || exit 1
	done
	;;
stop) kill "$(cat "$dir/pid")" 2>/dev/null || true ;;
*) sed -n '2,17s/^# \{0,1\}//p' "$0"; exit 2 ;;
esac
