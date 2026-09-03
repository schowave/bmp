FROM debian:trixie-slim

ENV DEBIAN_FRONTEND=noninteractive \
    DEBCONF_NONINTERACTIVE_SEEN=true

# Layer 1: Pakete installieren (ändert sich selten)
RUN printf 'path-exclude /usr/share/doc/*\npath-exclude /usr/share/man/*\npath-exclude /usr/share/info/*\n' > /etc/dpkg/dpkg.cfg.d/excludes \
    && apt-get update \
    && apt-get install -y --no-install-recommends \
        tigervnc-standalone-server \
        ratpoison \
        dosbox \
        novnc \
        websockify \
    && apt-get clean \
    && rm -rf /var/lib/apt/lists/* /tmp/* /var/tmp/*

# Debian 13 (trixie) ersetzt SDL 1.2 durch sdl12-compat, einen Shim auf SDL2.
# DOSBox 0.74 friert damit nach etwa einer Minute ein: der Emulator-Thread kehrt
# nicht mehr in den SDL-Event-Loop zurueck, Maus und Tastatur reagieren nicht
# mehr und das Bild bleibt stehen. Deshalb gezielt das echte SDL 1.2 aus
# bookworm installieren. Pin-Priority ueber 1000 erlaubt den Downgrade, das
# apt-mark hold verhindert, dass es spaeter wieder durch den Shim ersetzt wird.
RUN echo 'deb http://deb.debian.org/debian bookworm main' > /etc/apt/sources.list.d/bookworm.list \
    && printf 'Package: libsdl1.2debian\nPin: release n=bookworm\nPin-Priority: 1001\n' > /etc/apt/preferences.d/real-sdl12 \
    && apt-get update \
    && apt-get install -y --no-install-recommends --allow-downgrades libsdl1.2debian \
    && apt-mark hold libsdl1.2debian \
    && rm -f /etc/apt/sources.list.d/bookworm.list \
    && apt-get clean \
    && rm -rf /var/lib/apt/lists/* /tmp/* /var/tmp/*

# Unprivilegierten User anlegen
RUN useradd -m -s /bin/sh bmp

# Layer 2: VNC + Ratpoison + DOSBox konfigurieren
RUN mkdir -p /home/bmp/.config/tigervnc /home/bmp/.dosbox \
    && touch /home/bmp/.Xauthority \
    && printf '#!/bin/sh\nexec ratpoison\n' > /home/bmp/.config/tigervnc/xstartup \
    && chmod +x /home/bmp/.config/tigervnc/xstartup \
    && printf 'set border 0\nset padding 0 0 0 0\nexec sh -c "while true; do dosbox -conf /home/bmp/.dosbox/dosbox.conf -c \\"MOUNT C: /dos\\" -c \\"MOUNT D: /savegame\\" -c \\"C:\\" -c \\"cd bmp\\" -c \\"bmmain.exe\\" -c \\"exit\\"; done"\n' > /home/bmp/.ratpoisonrc \
    && echo '<!DOCTYPE html><html><head><meta http-equiv="refresh" content="0;url=player.html"></head></html>' > /usr/share/novnc/index.html \
    && chown -R bmp:bmp /home/bmp

# Layer 3: Spieldaten kopieren (ändert sich selten)
COPY --chown=bmp:bmp bmp /dos/bmp

# Layer 4: Konfiguration (ändert sich gelegentlich)
COPY --chown=bmp:bmp dosbox.conf /home/bmp/.dosbox/dosbox.conf
COPY player.html /usr/share/novnc/player.html
COPY images/favicon.png /usr/share/novnc/favicon.png
COPY VERSION /usr/share/novnc/version.txt

COPY entrypoint.sh /entrypoint.sh

EXPOSE 8080

# Ohne USER bmp startet der Container als root, damit entrypoint.sh die Rechte auf
# /savegame richten kann; es wechselt danach selbst auf bmp. HOME muss dabei gesetzt
# sein, sonst landet die vncserver-Konfiguration unter /root statt /home/bmp.
ENV HOME=/home/bmp
ENTRYPOINT ["/entrypoint.sh"]
CMD ["sh", "-c", "vncserver :1 -geometry 640x480 -depth 16 -SecurityTypes None -xstartup /home/bmp/.config/tigervnc/xstartup && websockify -D --web=/usr/share/novnc/ 8080 localhost:5901 && tail -f /dev/null"]
