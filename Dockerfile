FROM debian:bookworm-slim

ENV USER=root \
    DEBIAN_FRONTEND=noninteractive \
    DEBCONF_NONINTERACTIVE_SEEN=true

RUN set -ex \
    && printf 'path-exclude /usr/share/doc/*\npath-exclude /usr/share/man/*\npath-exclude /usr/share/info/*\n' > /etc/dpkg/dpkg.cfg.d/excludes \
    && apt-get update \
    && apt-get install -y --no-install-recommends \
        tightvncserver \
        xfonts-base \
        ratpoison \
        dosbox \
        novnc \
        websockify \
    && apt-get clean \
    && rm -rf /var/lib/apt/lists/* /tmp/* /var/tmp/* \
    && mkdir -p /root/.vnc \
    && touch /root/.Xauthority \
    && printf '#!/bin/sh\nxsetroot -solid black\nratpoison &\n' > /root/.vnc/xstartup \
    && chmod +x /root/.vnc/xstartup \
    && printf 'set border 0\nset padding 0 0 0 0\nexec dosbox -conf ~/.dosbox/dosbox.conf -fullscreen -c "MOUNT C: /dos" -c "MOUNT D: /savegame" -c "C:" -c "cd bmp" -c "bmmain.exe"\n' > /root/.ratpoisonrc \
    && export DOSCONF=$(dosbox -printconf) \
    && cp "$DOSCONF" /root/.dosbox/dosbox.conf \
    && sed -i 's/usescancodes=true/usescancodes=false/' /root/.dosbox/dosbox.conf \
    && sed -i 's/output=surface/output=opengl/' /root/.dosbox/dosbox.conf \
    && sed -i 's/aspect=true/aspect=false/' /root/.dosbox/dosbox.conf \
    && echo '<!DOCTYPE html><html><head><meta http-equiv="refresh" content="0;url=vnc_lite.html?autoconnect=true&resize=scale"></head></html>' > /usr/share/novnc/index.html

COPY bmp /dos/bmp

EXPOSE 80

CMD ["sh", "-c", "vncserver :1 -geometry 1920x1080 -depth 24 -SecurityTypes None && websockify -D --web=/usr/share/novnc/ 80 localhost:5901 && tail -f /dev/null"]
