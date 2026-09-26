# Container command without the idle pause of vnc/start.sh, as the image ran before:
#   node bench.mjs --label no-pause --cmd "$(cat variants/no-pause.sh)"
vncserver :1 -geometry 640x480 -depth 16 -SecurityTypes None -xstartup /home/bmp/.config/tigervnc/xstartup \
  && websockify -D --web=/usr/share/novnc/ 8080 localhost:5901 \
  && tail -f /dev/null
