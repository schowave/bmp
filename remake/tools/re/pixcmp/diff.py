"""Pixel diff of a screenshot of the original and one of the remake.

Usage: diff.py original.png remake.png out.png
Both may be 320x240 or a multiple (the driver saves 640x480). Prints the
number of differing pixels and writes original | remake | differences (red)
side by side, scaled 2x.
"""
import sys
from PIL import Image
o=Image.open(sys.argv[1]).convert('RGB'); r=Image.open(sys.argv[2]).convert('RGB')
so=o.width//320; sr=r.width//320
out=Image.new('RGB',(960,240))
n=0; rows={}
for y in range(240):
    for x in range(320):
        a=o.getpixel((x*so,y*so)); b=r.getpixel((x*sr,y*sr))
        out.putpixel((x,y),a); out.putpixel((x+320,y),b)
        if a!=b:
            n+=1; out.putpixel((x+640,y),(255,0,0)); rows.setdefault(y,[]).append(x)
        else:
            out.putpixel((x+640,y),tuple(v//3 for v in a))
out=out.resize((1920,480),Image.NEAREST); out.save(sys.argv[3])
print('diff',n)
