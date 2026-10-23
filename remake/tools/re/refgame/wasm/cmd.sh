#!/bin/sh
# Sends commands to the driver: cmd.sh '[["click",160,120],["wait",1000],["shot","/abs/x.png"]]'
# PORT selects the driver (default 9333).
curl -s -X POST --data-binary "$1" "http://localhost:${PORT:-9333}/"
