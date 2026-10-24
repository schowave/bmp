#!/bin/sh
# Sends commands to the driver: cmd.sh '[["click",160,120],["wait",1000],["shot","/abs/x.png"]]'
# BMP_DRIVER_PORT selects the driver (default 9333).
curl -s -X POST --data-binary "$1" "http://localhost:${BMP_DRIVER_PORT:-9333}/"
