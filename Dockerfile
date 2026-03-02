FROM debian:bookworm-slim

ENV USER=root \
    DEBIAN_FRONTEND=noninteractive \
    DEBCONF_NONINTERACTIVE_SEEN=true

COPY bmp /dos/bmp

RUN set -ex \
    && apt-get update \
    && apt-get install -y --no-install-recommends \
        openssl \
        tightvncserver \
        xfonts-base \
        xfonts-75dpi \
        xfonts-100dpi \
        ratpoison \
        dosbox \
        novnc \
        websockify \
    && rm -rf /var/lib/apt/lists/* \
    && echo "tzdata tzdata/Areas select Europe" > /tmp/debconf.txt \
    && echo "tzdata tzdata/Zones/Europe select Berlin" >> /tmp/debconf.txt \
    && debconf-set-selections /tmp/debconf.txt \
    && mkdir -p /root/.vnc \
    && touch /root/.Xauthority \
    && echo "#!/bin/sh\nxsetroot -solid grey\nx-terminal-emulator -geometry 80x24+10+10 -ls -title \"\$VNCDESKTOP Desktop\" &\nratpoison &\nx-window-manager &" > /root/.vnc/xstartup \
    && echo "xset +fp /usr/share/fonts/X11/misc" >> /root/.vnc/xstartup \
    && echo "xset +fp /usr/share/fonts/X11/75dpi" >> /root/.vnc/xstartup \
    && echo "xset +fp /usr/share/fonts/X11/100dpi" >> /root/.vnc/xstartup \
    && chmod +x /root/.vnc/xstartup \
    && echo "exec dosbox -conf ~/.dosbox/dosbox.conf -fullscreen -c 'MOUNT C: /dos' -c 'MOUNT D: /savegame' -c 'C:' -c 'cd bmp' -c 'bmmain.exe'" >> /root/.ratpoisonrc \
    && export DOSCONF=$(dosbox -printconf) \
    && cp $DOSCONF /root/.dosbox/dosbox.conf \
    && sed -i 's/usescancodes=true/usescancodes=false/' /root/.dosbox/dosbox.conf \
    && openssl req -x509 -nodes -newkey rsa:2048 -keyout /root/novnc.pem -out /root/novnc.pem -days 3650 -subj "/C=DE/ST=B/L=B/O=B/OU=B/CN=B emailAddress=email@example.com" \
    && echo '<!DOCTYPE html><html><head><meta http-equiv="refresh" content="0;url=vnc_lite.html?autoconnect=true&resize=scale&password=bmp123"></head></html>' > /usr/share/novnc/index.html

EXPOSE 80

CMD ["sh", "-c", "vncserver :1 -geometry 1024x768 -depth 24 && websockify -D --web=/usr/share/novnc/ --cert=/root/novnc.pem 80 localhost:5901 && tail -f /dev/null"]
