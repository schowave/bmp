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

# Layer 2: VNC + Ratpoison + DOSBox konfigurieren (ändert sich gelegentlich)
RUN mkdir -p /root/.vnc \
    && touch /root/.Xauthority \
    && printf '#!/bin/sh\nexec ratpoison\n' > /root/.vnc/xstartup \
    && chmod +x /root/.vnc/xstartup \
    && printf 'set border 0\nset padding 0 0 0 0\nexec dosbox -conf ~/.dosbox/dosbox.conf -c "MOUNT C: /dos" -c "MOUNT D: /savegame" -c "C:" -c "cd bmp" -c "bmmain.exe"\n' > /root/.ratpoisonrc \
    && export DOSCONF=$(dosbox -printconf) \
    && cp "$DOSCONF" /root/.dosbox/dosbox.conf \
    && sed -i 's/usescancodes=true/usescancodes=false/' /root/.dosbox/dosbox.conf \
    && sed -i 's/^fullscreen=false/fullscreen=true/' /root/.dosbox/dosbox.conf \
    && sed -i 's/^fulldouble=false/fulldouble=true/' /root/.dosbox/dosbox.conf \
    && sed -i 's/^fullresolution=original/fullresolution=1440x1080/' /root/.dosbox/dosbox.conf \
    && sed -i 's/^windowresolution=original/windowresolution=1440x1080/' /root/.dosbox/dosbox.conf \
    && sed -i 's/^output=surface/output=overlay/' /root/.dosbox/dosbox.conf \
    && sed -i 's/^aspect=false/aspect=true/' /root/.dosbox/dosbox.conf \
    && echo '<!DOCTYPE html><html><head><meta http-equiv="refresh" content="0;url=vnc_lite.html?autoconnect=true&resize=scale"></head></html>' > /usr/share/novnc/index.html

# Layer 3: Spieldaten kopieren (ändert sich selten, aber unabhängig von Config)
COPY bmp /dos/bmp

EXPOSE 80

CMD ["sh", "-c", "vncserver :1 -geometry 1440x1080 -depth 24 -SecurityTypes None -xstartup /root/.vnc/xstartup && websockify -D --web=/usr/share/novnc/ 80 localhost:5901 && tail -f /dev/null"]
