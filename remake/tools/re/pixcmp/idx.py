"""Reads screenshots as palette indices of the UI palette (PIC/1.PAL, as
DOSBox shows it), for dumping rows and columns:

    from idx import Img
    o = Img("shot.png"); o(x, y); o.runs(y); o.cruns(x)
"""
import sys
from PIL import Image
P=[(0,0,0),(162,162,195),(130,130,162),(113,113,146),(97,97,130),(81,81,113),(65,65,97),(48,48,81),(0,0,113),(146,113,81),(178,162,130),(211,195,178),(81,97,16),(97,113,32),(97,130,48),(113,146,65),(97,0,16),(146,0,16),(178,0,32),(195,113,32),(162,97,32),(113,65,16),(48,65,195),(243,243,0),(178,162,113),(146,130,81),(130,97,65),(97,65,48),(81,48,32),(243,243,243)]
M={c:i for i,c in enumerate(P)}
class Img:
    def __init__(s,f):
        s.im=Image.open(f).convert('RGB'); s.s=s.im.width//320
    def __call__(s,x,y): return M.get(s.im.getpixel((x*s.s,y*s.s)),-1)
    def runs(s,y,x0=0,x1=320):
        out=[];cur=None
        for x in range(x0,x1):
            v=s(x,y)
            if cur and cur[0]==v: cur[2]=x
            else:
                if cur: out.append(tuple(cur))
                cur=[v,x,x]
        out.append(tuple(cur)); return out
    def cruns(s,x,y0=0,y1=240):
        out=[];cur=None
        for y in range(y0,y1):
            v=s(x,y)
            if cur and cur[0]==v: cur[2]=y
            else:
                if cur: out.append(tuple(cur))
                cur=[v,y,y]
        out.append(tuple(cur)); return out
