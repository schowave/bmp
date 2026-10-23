"""Bounding box of the pixels of one colour.

Usage: ink.py shot.png x0 y0 x1 y1 r,g,b
"""
import sys
from PIL import Image
im=Image.open(sys.argv[1]).convert('RGB'); s=im.width//320
x0,y0,x1,y1=map(int,sys.argv[2:6]); c=tuple(map(int,sys.argv[6].split(',')))
pts=[(x,y) for y in range(y0,y1) for x in range(x0,x1) if im.getpixel((x*s,y*s))==c]
if pts: print('x %d-%d y %d-%d n=%d'%(min(p[0] for p in pts),max(p[0] for p in pts),min(p[1] for p in pts),max(p[1] for p in pts),len(pts)))
else: print('none')
