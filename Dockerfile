FROM debian:bookworm-slim

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

# Unprivilegierten User anlegen
RUN useradd -m -s /bin/sh bmp

# Layer 2: VNC + Ratpoison + DOSBox konfigurieren
RUN mkdir -p /home/bmp/.vnc /home/bmp/.dosbox \
    && touch /home/bmp/.Xauthority \
    && printf '#!/bin/sh\nexec ratpoison\n' > /home/bmp/.vnc/xstartup \
    && chmod +x /home/bmp/.vnc/xstartup \
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

EXPOSE 8080

USER bmp
CMD ["sh", "-c", "vncserver :1 -geometry 640x480 -depth 16 -SecurityTypes None -xstartup /home/bmp/.vnc/xstartup && websockify -D --web=/usr/share/novnc/ 8080 localhost:5901 && tail -f /dev/null"]
