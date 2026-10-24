"""Where does a region of the remake shot appear in the original shot?

Usage: find.py original.png remake.png x0 y0 x1 y1 [radius]
Tries every offset up to radius (default 8) and prints the best two as
(differing pixels, dx, dy): the original has the region at x0+dx, y0+dy.
The tool to measure positions without identical game data elsewhere.
"""
import sys
import numpy as np
from PIL import Image
o=np.array(Image.open(sys.argv[1]).convert('RGB'))
r=np.array(Image.open(sys.argv[2]).convert('RGB'))
so=o.shape[1]//320; sr=r.shape[1]//320
o=o[::so,::so].astype(int); r=r[::sr,::sr].astype(int)
a=list(map(int," ".join(sys.argv[3:]).split()))
x0,y0,x1,y1=a[:4]; R=a[4] if len(a)>4 else 8
t=r[y0:y1,x0:x1]
best=[]
for dy in range(-R,R+1):
    for dx in range(-R,R+1):
        if y0+dy<0 or x0+dx<0 or y1+dy>240 or x1+dx>320: continue
        w=o[y0+dy:y1+dy,x0+dx:x1+dx]
        n=int((np.abs(w-t).sum(axis=2)>0).sum())
        best.append((n,dx,dy))
best.sort(); print(best[:2],'of',t.shape[0]*t.shape[1])
