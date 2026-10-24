#!/bin/sh
# Starts the VNC server and websockify, and stops DOSBox while nobody is playing.
#
# DOSBox keeps computing without a browser connected and takes about half a core.
# The container runs around the clock, though, and is rarely played. websockify only
# opens its connection to port 5901 (hex 170D) while a browser is connected. With an
# ESTABLISHED connection there DOSBox runs (SIGCONT), otherwise it is stopped
# (SIGSTOP). The game state in memory survives that. The PID is looked up every
# second because ratpoison restarts DOSBox after the game exits. The same pass checks
# that Xvnc and websockify are still running; if one is gone, the script exits.

vncserver :1 -geometry 640x480 -depth 16 -SecurityTypes None \
    -xstartup /home/bmp/.config/tigervnc/xstartup || exit 1
websockify -D --web=/usr/share/novnc/ 8080 localhost:5901 || exit 1

# As PID 1 the shell would ignore SIGTERM, and podman stop would wait 10 s.
trap 'exit 0' TERM

# What was last sent to which DOSBox. A signal only goes out when the state or the
# process changes, so a new DOSBox after a restart gets one right away.
last=

while sleep 1; do
    if grep -qE '^ *[0-9]+: [0-9A-F]+:170D [0-9A-F]+:[0-9A-F]+ 01 ' /proc/net/tcp /proc/net/tcp6; then
        sig=CONT
    else
        sig=STOP
    fi
    xvnc='' ws='' dosbox=''
    for p in /proc/[0-9]*; do
        # read is a builtin, so this pass forks no process per PID.
        { read -r comm < "$p/comm"; } 2>/dev/null || continue
        case "$comm" in
            dosbox) dosbox="$dosbox ${p#/proc/}" ;;
            Xtigervnc) xvnc=1 ;;
            websockify) ws=1 ;;
        esac
    done
    if [ -n "$dosbox" ] && [ "$sig$dosbox" != "$last" ]; then
        # $dosbox is a list of PIDs and meant to split.
        # shellcheck disable=SC2086
        kill -$sig $dosbox 2>/dev/null
        last="$sig$dosbox"
    fi
    # Without Xvnc or websockify no browser gets into the game, yet the container
    # would keep running quietly. Exit instead, so the restart policy restarts it.
    if [ -z "$xvnc" ] || [ -z "$ws" ]; then
        [ -z "$xvnc" ] && echo "bmp-start: Xtigervnc is no longer running" >&2
        [ -z "$ws" ] && echo "bmp-start: websockify is no longer running" >&2
        exit 1
    fi
done
