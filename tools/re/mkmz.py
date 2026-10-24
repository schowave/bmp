import json,struct,sys
n=sys.argv[1]; img=open(n+'.bin','rb').read(); m=json.load(open(n+'.json'))
rel=m['relocs']; hdrsize=0x1c+4*len(rel); hdrpar=(hdrsize+15)//16
total=hdrpar*16+len(img)
cp=(total+511)//512; cblp=total%512
h=struct.pack('<2s13H',b'MZ',cblp,cp,len(rel),hdrpar,m['minalloc'],0xffff,m['ss'],m['sp'],0,m['ip'],m['cs'],0x1c,0)
h+=b''.join(struct.pack('<HH',o,s) for s,o in rel)
h+=b'\0'*(hdrpar*16-len(h))
open(n+'_unp.exe','wb').write(h+img)
