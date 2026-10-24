import re,sys
img=open('BMMAIN.bin','rb').read()
DS=0x4cb3*16; FS=0x4238*16
def s_at(base,off):
    p=base+off
    if p>=len(img) or (base==FS and p>=DS): return None
    if off>0 and img[p-1]!=0: return None
    e=img.find(b'\0',p)
    if e<0 or e-p<3 or e-p>300: return None
    s=img[p:e]
    if sum(32<=c<127 for c in s)<len(s)*0.9: return None
    return s.decode('latin1')
cur=None; out={}
for line in open('BMMAIN.c'):
    m=re.match(r'// ==== (\S+)',line)
    if m: cur=m.group(1); out[cur]=[]; continue
    if not cur: continue
    # far pointers 0x5238xxxx / 0x5cb3xxxx
    for m in re.finditer(r'0x(5238|5cb3)([0-9a-f]{4})\b',line):
        base=FS if m.group(1)=='5238' else DS
        s=s_at(base,int(m.group(2),16))
        if s: out[cur].append(s)
    for m in re.finditer(r'CONCAT\d+\((?:unaff_DS|[^,]*0x5cb3[^,]*),\s*\(char \*\)0x([0-9a-f]+)\)',line):
        s=s_at(DS,int(m.group(1),16))
        if s: out[cur].append(s)
    for m in re.finditer(r'CONCAT12\(\(char\)unaff_DS,0x([0-9a-f]+)\)',line):
        s=s_at(DS,int(m.group(1),16))
        if s: out[cur].append(s)
    for m in re.finditer(r'CONCAT22\(0x5238,\(char \*\)0x([0-9a-f]+)\)',line):
        s=s_at(FS,int(m.group(1),16))
        if s: out[cur].append(s)
for k,v in out.items():
    if v:
        seen=[]; [seen.append(x) for x in v if x not in seen]
        print(k, '|', ' || '.join(x[:60] for x in seen[:12]))
