"""Closed-loop control of the original game in the reference container.

DOSBox only gets relative mouse motion from VNC, so the in-game pointer does
not follow absolute coordinates. We look for the pointer in screenshots and
move until it is where we want it.

usage:
  ctl.py shot FILE
  ctl.py click X Y [BUTTON]     (640x480 coordinates, button 1 left, 3 right)
  ctl.py type TEXT
  ctl.py key KEY
"""
import sys
import time

from PIL import Image
from vncdotool import api

RED = (180, 0, 32)
TIP = [(0, 0), (1, 1), (2, 2), (3, 5), (4, 6), (5, 9), (6, 12), (12, 12), (8, 14)]
SHOTS = "/private/tmp/claude-501/-Users-dsh-repos-privat-bmp/13af71b9-569b-4ec7-92a4-d5a193df8943/scratchpad/shots/"


def find_cursor(im):
    px = im.load()
    w, h = im.size
    for y in range(h):
        for x in range(w):
            if px[x, y][:3] != RED:
                continue
            ok = 0
            for dx, dy in TIP:
                if x + dx < w and y + dy < h and px[x + dx, y + dy][:3] == RED:
                    ok += 1
            if ok >= len(TIP) - 1:
                return x, y
    return None


class Ctl:
    def __init__(self):
        self.c = api.connect("127.0.0.1::5991")
        self.hx, self.hy = 320, 240
        self.c.mouseMove(self.hx, self.hy)

    def shot(self, path=None):
        self.c.refreshScreen()
        im = self.c.screen.convert("RGB")
        if path:
            im.save(path)
        return im

    def move(self, x, y):
        # DOSBox drops or clips big jumps; walk in small steps.
        while (self.hx, self.hy) != (x, y):
            dx = max(-30, min(30, x - self.hx))
            dy = max(-30, min(30, y - self.hy))
            self.hx += dx
            self.hy += dy
            self.c.mouseMove(self.hx, self.hy)
            time.sleep(0.01)

    def sync(self):
        # Drive the game pointer into the top-left corner: afterwards host and
        # game coordinates agree, because both move 1:1.
        self.move(639, 479)
        self.move(0, 0)
        time.sleep(0.15)

    def goto(self, x, y):
        self.sync()
        self.move(x, y)
        for _ in range(4):
            time.sleep(0.35)
            cur = find_cursor(self.shot())
            if cur is None:
                return False
            dx, dy = x - cur[0], y - cur[1]
            if abs(dx) <= 1 and abs(dy) <= 1:
                return True
            self.move(max(0, min(639, self.hx + dx)), max(0, min(479, self.hy + dy)))
        return False

    def click(self, x, y, button=1):
        ok = self.goto(x, y)
        self.c.mousePress(button)
        time.sleep(0.3)
        return ok


def main():
    cmd = sys.argv[1]
    k = Ctl()
    if cmd == "shot":
        k.shot(SHOTS + sys.argv[2])
    elif cmd == "click":
        b = int(sys.argv[4]) if len(sys.argv) > 4 else 1
        ok = k.click(int(sys.argv[2]), int(sys.argv[3]), b)
        time.sleep(1.2)
        print("on target" if ok else "MISSED")
    elif cmd == "type":
        for ch in sys.argv[2]:
            k.c.keyPress(ch)
    elif cmd == "key":
        k.c.keyPress(sys.argv[2])
    k.c.disconnect()


if __name__ == "__main__":
    main()
