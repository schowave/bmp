# The build context is the repository root:
#   docker build -f vnc/Dockerfile .

FROM debian:trixie-slim

ENV DEBIAN_FRONTEND=noninteractive \
    DEBCONF_NONINTERACTIVE_SEEN=true

# Layer 1: install packages (rarely changes)
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

# Debian 13 (trixie) replaces SDL 1.2 with sdl12-compat, a shim on top of SDL2.
# DOSBox 0.74 freezes with it after about a minute: the emulator thread no longer
# returns to the SDL event loop, mouse and keyboard stop responding and the picture
# stands still. So install the real SDL 1.2 from bookworm specifically. A
# Pin-Priority above 1000 allows the downgrade; the apt-mark hold keeps the shim
# from replacing it again later.
RUN echo 'deb http://deb.debian.org/debian bookworm main' > /etc/apt/sources.list.d/bookworm.list \
    && printf 'Package: libsdl1.2debian\nPin: release n=bookworm\nPin-Priority: 1001\n' > /etc/apt/preferences.d/real-sdl12 \
    && apt-get update \
    && apt-get install -y --no-install-recommends --allow-downgrades libsdl1.2debian \
    && apt-mark hold libsdl1.2debian \
    && rm -f /etc/apt/sources.list.d/bookworm.list \
    && apt-get clean \
    && rm -rf /var/lib/apt/lists/* /tmp/* /var/tmp/*

# Create an unprivileged user
RUN useradd -m -s /bin/sh bmp

# Layer 2: configure VNC + Ratpoison + DOSBox
RUN mkdir -p /home/bmp/.config/tigervnc /home/bmp/.dosbox \
    && touch /home/bmp/.Xauthority \
    && printf '#!/bin/sh\nexec ratpoison\n' > /home/bmp/.config/tigervnc/xstartup \
    && chmod +x /home/bmp/.config/tigervnc/xstartup \
    && printf 'set border 0\nset padding 0 0 0 0\nexec sh -c "while true; do dosbox -conf /home/bmp/.dosbox/dosbox.conf -c \\"MOUNT C: /dos\\" -c \\"MOUNT D: /savegame\\" -c \\"C:\\" -c \\"cd bmp\\" -c \\"bmmain.exe\\" -c \\"exit\\"; done"\n' > /home/bmp/.ratpoisonrc \
    && echo '<!DOCTYPE html><html><head><meta http-equiv="refresh" content="0;url=player.html"></head></html>' > /usr/share/novnc/index.html \
    && chown -R bmp:bmp /home/bmp

# Layer 3: copy game data (rarely changes)
COPY --chown=bmp:bmp bmp /dos/bmp

# Layer 4: configuration (changes occasionally)
COPY --chown=bmp:bmp vnc/dosbox.conf /home/bmp/.dosbox/dosbox.conf
COPY --chmod=755 vnc/start.sh /usr/local/bin/bmp-start
COPY vnc/player.html /usr/share/novnc/player.html
COPY web/hilfe.html /usr/share/novnc/hilfe.html
COPY web/favicon.png /usr/share/novnc/favicon.png
COPY VERSION /usr/share/novnc/version.txt

EXPOSE 8080

# websockify serves the page too; if it does not answer, nobody gets into the game.
# python3 is there anyway because websockify needs it, whereas curl and wget are not.
HEALTHCHECK --interval=30s --timeout=5s --start-period=20s --retries=3 \
    CMD python3 -c "import urllib.request; urllib.request.urlopen('http://localhost:8080/version.txt', timeout=4)"

USER bmp
CMD ["/usr/local/bin/bmp-start"]
