#!/bin/sh
# montage.sh out.png in1.png in2.png ...: the shots in two labelled columns
# (ImageMagick), to look at several states at once.
out=$1; shift
montage -font /System/Library/Fonts/Supplemental/Arial.ttf -label '%t' -pointsize 14 "$@" \
  -tile 2x -geometry 640x480+4+4 -filter point "$out"
