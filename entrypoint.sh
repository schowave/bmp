#!/bin/sh
# /savegame ist ein Bind-Mount vom Host, der Besitzer kommt also von dort und ist
# meist root. Das Spiel laeuft unprivilegiert als bmp und kann dort dann nichts
# ablegen: im Spiel laesst sich Laufwerk D: nicht beschreiben, und ein Spielstand
# auf C: liegt nur im Container und ist beim naechsten Update verloren.
#
# Deshalb startet der Container als root, richtet einmalig die Rechte und gibt sie
# sofort wieder ab. Das ist der Preis dafuer, dass es auf jedem Host funktioniert,
# ohne dass dort erst jemand von Hand ein chown absetzen muss.
set -e

if [ "$(id -u)" = 0 ]; then
    if [ -d /savegame ]; then
        # chown ist der saubere Weg. Wo er nicht durchgeht, etwa bei manchen
        # Netzlaufwerken, reichen weit offene Rechte immer noch fuer den Zweck.
        chown -R bmp:bmp /savegame 2>/dev/null \
            || chmod -R a+rwX /savegame 2>/dev/null \
            || echo "entrypoint: /savegame bleibt schreibgeschuetzt, Spielstaende gehen verloren" >&2
    fi
    exec setpriv --reuid=1000 --regid=1000 --init-groups "$@"
fi

# Bereits unprivilegiert gestartet, etwa mit docker run --user: dann bleibt es dabei.
exec "$@"
