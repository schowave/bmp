FROM debian:bookworm-slim

ENV USER=root \
    DEBIAN_FRONTEND=noninteractive \
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

# Layer 2: VNC + Ratpoison + DOSBox konfigurieren
RUN mkdir -p /root/.vnc /root/.dosbox \
    && touch /root/.Xauthority \
    && printf '#!/bin/sh\nexec ratpoison\n' > /root/.vnc/xstartup \
    && chmod +x /root/.vnc/xstartup \
    && printf 'set border 0\nset padding 0 0 0 0\nexec sh -c "while true; do dosbox -conf /root/.dosbox/dosbox.conf -c \\"MOUNT C: /dos\\" -c \\"MOUNT D: /savegame\\" -c \\"C:\\" -c \\"cd bmp\\" -c \\"bmmain.exe\\"; done"\n' > /root/.ratpoisonrc \
    && echo '<!DOCTYPE html><html><head><meta http-equiv="refresh" content="0;url=player.html"></head></html>' > /usr/share/novnc/index.html

COPY player.html /usr/share/novnc/player.html
COPY images/favicon.png /usr/share/novnc/favicon.png

COPY dosbox.conf /root/.dosbox/dosbox.conf

# Layer 3: Spieldaten kopieren
COPY bmp /dos/bmp

EXPOSE 80

CMD ["sh", "-c", "vncserver :1 -geometry 640x480 -depth 16 -SecurityTypes None -xstartup /root/.vnc/xstartup && websockify -D --web=/usr/share/novnc/ 80 localhost:5901 && tail -f /dev/null"]
