"""Annotate Ghidra C output: string pointer slots, function names."""
import json, re, struct, sys
img = open('BMMAIN.bin', 'rb').read(); m = json.load(open('BMMAIN.json'))
DS = 0x4cb3 * 16; FS = 0x4238 * 16
def s_at(base, off):
    p = base + off
    e = img.find(b'\0', p)
    return img[p:e].decode('latin1')
slots = {}
for s, o in m['relocs']:
    x = s * 16 + o
    if x < DS: continue
    seg = struct.unpack_from('<H', img, x)[0]; off = struct.unpack_from('<H', img, x - 2)[0]
    if seg in (0x4238, 0x4cb3):
        st = s_at(seg * 16, off)
        if st: slots[x - 2 - DS] = st[:70]
names = {}
try:
    for line in open('BMMAIN.names'):
        line = line.split('#')[0].split()
        if len(line) >= 2: names[line[0]] = line[1]
except FileNotFoundError: pass
out = open('BMMAIN.ann.c', 'w')
for line in open('BMMAIN.c'):
    ann = []
    for mm in re.finditer(r'\*\(\w+ \*\)0x([0-9a-f]{3,4})\b', line):
        v = int(mm.group(1), 16)
        if v in slots: ann.append('"%s"' % slots[v])
    for mm in re.finditer(r'\b0x([0-9a-f]{3,4})\b', line):
        v = int(mm.group(1), 16)
        if v in slots and '"%s"' % slots[v] not in ann and ('0x5cb3' in line or 'unaff_DS' in line or 'CONCAT' in line):
            ann.append('~"%s"' % slots[v])
    line = re.sub(r'\bf_([0-9a-f]{4})_([0-9a-f]{4})\b', lambda mm: names.get(mm.group(0), mm.group(0)), line)
    if ann:
        line = line.rstrip('\n') + '  /* ' + ' '.join(ann) + ' */\n'
    out.write(line)
json.dump({str(k): v for k, v in slots.items()}, open('slots.json', 'w'))
print(len(slots), 'slots')
