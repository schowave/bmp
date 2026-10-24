import struct,zlib,sys,os,glob
def png(path,w,h,px,pal):
    raw=b''.join(b'\0'+bytes(px[y*w:(y+1)*w]) for y in range(h))
    def ch(t,d): c=struct.pack('>I',len(d))+t+d; return c+struct.pack('>I',zlib.crc32(t+d))
    open(path,'wb').write(b'\x89PNG\r\n\x1a\n'+ch(b'IHDR',struct.pack('>IIBBBBB',w,h,8,3,0,0,0))+ch(b'PLTE',pal)+ch(b'IDAT',zlib.compress(raw))+ch(b'IEND',b''))
def dec(d):
    w,h,n=struct.unpack_from('<HHH',d,0); p=6; out=bytearray()
    while p<len(d):
        b=d[p];p+=1
        if b<0x80: out+=d[p:p+b+1]; p+=b+1
        else:
            b&=0x7f
            if b==0: break
            out+=bytes([d[p]])*(b+1); p+=1
    return w,h,out
src,dst,palidx=sys.argv[1],sys.argv[2],int(sys.argv[3])
P=open(os.path.join(src,'1.PAL'),'rb').read()
pal=bytes(min(255,c*255//63) for c in P[palidx*768:(palidx+1)*768])
for f in glob.glob(os.path.join(src,'*.VGA'))+glob.glob(os.path.join(src,'*.CP')):
    w,h,px=dec(open(f,'rb').read())
    if len(px)<w*h: print('short',f,len(px),w*h); px+=bytes(w*h-len(px))
    elif len(px)>w*h: print('long',f,len(px),w*h)
    png(os.path.join(dst,os.path.basename(f)+'.png'),w,h,px,pal)
