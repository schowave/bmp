#!/bin/sh
# Startet VNC-Server und websockify und haelt DOSBox an, solange niemand spielt.
#
# DOSBox rechnet auch ohne verbundenen Browser ununterbrochen und belegt dabei etwa
# eine halbe Core. Der Container laeuft aber rund um die Uhr, gespielt wird selten.
# websockify baut seine Verbindung zu Port 5901 (hex 170D) nur auf, solange ein
# Browser verbunden ist. Gibt es dort eine ESTABLISHED-Verbindung, laeuft DOSBox
# weiter (SIGCONT), sonst wird es angehalten (SIGSTOP). Der Spielstand im Speicher
# bleibt dabei erhalten. Die PID wird jede Sekunde neu gesucht, weil ratpoison
# DOSBox nach dem Beenden des Spiels neu startet. In derselben Runde wird geprueft,
# ob Xvnc und websockify noch laufen; fehlt einer, beendet sich das Skript.

vncserver :1 -geometry 640x480 -depth 16 -SecurityTypes None \
    -xstartup /home/bmp/.config/tigervnc/xstartup || exit 1
websockify -D --web=/usr/share/novnc/ 8080 localhost:5901 || exit 1

# Als PID 1 ignoriert die Shell SIGTERM sonst, und podman stop wartet 10 s.
trap 'exit 0' TERM

while sleep 1; do
    if grep -qE '^ *[0-9]+: [0-9A-F]+:170D [0-9A-F]+:[0-9A-F]+ 01 ' /proc/net/tcp /proc/net/tcp6; then
        sig=CONT
    else
        sig=STOP
    fi
    xvnc= ws=
    for p in /proc/[0-9]*; do
        case "$(cat "$p/comm" 2>/dev/null)" in
            dosbox) kill -$sig "${p#/proc/}" 2>/dev/null ;;
            Xtigervnc) xvnc=1 ;;
            websockify) ws=1 ;;
        esac
    done
    # Ohne Xvnc oder websockify kommt kein Browser mehr ins Spiel, der Container liefe
    # aber still weiter. Also beenden, damit die Restart-Policy ihn neu startet.
    if [ -z "$xvnc" ] || [ -z "$ws" ]; then
        [ -z "$xvnc" ] && echo "bmp-start: Xtigervnc laeuft nicht mehr" >&2
        [ -z "$ws" ] && echo "bmp-start: websockify laeuft nicht mehr" >&2
        exit 1
    fi
done
