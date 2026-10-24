"""Decodes a picture of PIC/ (N.VGA or N.CP) to width, height and palette
indices, without the game: from pic import dec; w, h, px = dec(path)
"""
import struct,sys
def dec(path):
    d=open(path,'rb').read()
    w,h,n=struct.unpack_from('<HHH',d,0); p=6; out=bytearray()
    while p<len(d) and len(out)<w*h:
        b=d[p];p+=1
        if b<0x80: out+=d[p:p+b+1]; p+=b+1
        else:
            b&=0x7f
            if b==0: break
            out+=bytes([d[p]])*(b+1); p+=1
    out+=bytes(max(0,w*h-len(out)))
    return w,h,out
