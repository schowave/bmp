"""Recursive-descent disassembler for an unpacked 16-bit MZ load image.

Usage: dis.py NAME DGROUP_SEG
Reads NAME.bin / NAME.json, writes NAME.lst (functions with annotations)
and NAME.funcs.json.
"""
import json
import re
import struct
import sys
from collections import defaultdict

from capstone import Cs, CS_ARCH_X86, CS_MODE_16, CS_GRP_JUMP, CS_GRP_CALL, CS_GRP_RET

name = sys.argv[1]
dgroup = int(sys.argv[2], 16)
img = open(name + ".bin", "rb").read()
meta = json.load(open(name + ".json"))
relocs = set(seg * 16 + off for seg, off in meta["relocs"])
md = Cs(CS_ARCH_X86, CS_MODE_16)
md.detail = True
DS = dgroup * 16

extra = []
if len(sys.argv) > 3:
    for a in sys.argv[3:]:
        s, o = a.split(":")
        extra.append((int(s, 16), int(o, 16)))


FS = 0x4238 * 16


def ds_string(off, base=None):
    if base is None:
        return ds_string(off, DS) or ds_string(off, FS)
    p = base + off
    if p >= len(img) or (base == FS and p >= DS):
        return None
    if off > 0 and img[p - 1] != 0:
        return None
    e = img.find(b"\0", p)
    if e < 0 or e - p < 2 or e - p > 200:
        return None
    s = img[p:e]
    if sum(32 <= c < 127 for c in s) < len(s) * 0.85:
        return None
    return s.decode("latin1")


funcs = {}  # linear -> (seg, off)
insns = {}  # linear -> insn info
work = [(meta["cs"], meta["ip"])] + extra
callers = defaultdict(set)


def lin(seg, off):
    return seg * 16 + off


def disasm_func(seg, off):
    todo = [off]
    seen = set()
    while todo:
        o = todo.pop()
        while True:
            la = lin(seg, o)
            if la in insns or o in seen or la >= len(img) or la >= DS:
                break
            seen.add(o)
            code = img[la:la + 16]
            ins = next(md.disasm(code, o), None)
            if ins is None:
                break
            text = ins.mnemonic + " " + ins.op_str
            insns[la] = (seg, o, ins.size, text, ins.bytes)
            mn = ins.mnemonic
            if mn == "lcall" or (mn == "call" and ins.bytes[0] == 0x9A):
                toff, tseg = struct.unpack_from("<HH", img, la + 1)
                work.append((tseg, toff))
                callers[lin(tseg, toff)].add(la)
            elif mn == "call" and ins.bytes[0] == 0xE8:
                t = int(ins.op_str, 16)
                work.append((seg, t))
                callers[lin(seg, t)].add(la)
            if mn in ("ret", "retf", "iret", "ljmp") or (mn == "jmp" and not ins.op_str.startswith("0x")):
                break
            if mn == "jmp":
                o = int(ins.op_str, 16)
                continue
            if mn.startswith("j") or mn.startswith("loop") or mn == "jcxz":
                if ins.op_str.startswith("0x"):
                    todo.append(int(ins.op_str, 16))
            o += ins.size


def drain():
  while work:
    seg, off = work.pop()
    la = lin(seg, off)
    if la in funcs:
        continue
    funcs[la] = (seg, off)
    disasm_func(seg, off)


drain()
# code segments seen so far; scan uncovered bytes for MSC prologues
segs = sorted(set(s for s, o in funcs.values()))
covered = bytearray(len(img))
for la, (seg, o, size, text, by) in insns.items():
    covered[la:la + size] = b"\1" * size
for k in range(0x10, DS - 3):
    if not covered[k] and img[k:k + 3] == b"\x55\x8b\xec":
        # pick the code segment with highest base <= k
        cand = [s for s in segs if s * 16 <= k]
        if not cand:
            continue
        seg = cand[-1]
        if k - seg * 16 > 0xffff:
            continue
        work.append((seg, k - seg * 16))
        drain()
        for la2, (sg, o2, size, t2, by2) in insns.items():
            pass
        for la2 in list(insns):
            sg, o2, size, t2, by2 = insns[la2]
            covered[la2:la2 + size] = b"\1" * size

names = {}
for la, (seg, off) in funcs.items():
    names[la] = "f_%04x_%04x" % (seg, off)

# load user-provided names
try:
    for line in open(name + ".names"):
        line = line.split("#")[0].strip()
        if not line:
            continue
        a, n = line.split()
        s, o = a.split(":")
        names[lin(int(s, 16), int(o, 16))] = n
except FileNotFoundError:
    pass
gnames = {}
try:
    for line in open(name + ".gnames"):
        line = line.split("#")[0].strip()
        if not line:
            continue
        a, n = line.split()
        if ":" in a:
            sg, of = a.split(":")
            gnames[(int(sg, 16), int(of, 16))] = n
        else:
            gnames[(dgroup, int(a, 16))] = n
except FileNotFoundError:
    pass

FSEGS = {}
for x in range(0, len(img) - DS - 1, 2):
    pass
def segval(slot):
    return struct.unpack_from("<H", img, DS + slot)[0]
SEGNAME = {dgroup: "D"}
for extra_seg in sys.argv[0:0]:
    pass
out = open(name + ".lst", "w")
cur_es = None
order = sorted(insns)
cur_func = None
for la in order:
    seg, o, size, text, by = insns[la]
    if la in funcs:
        out.write("\n;------------------------------------------------\n")
        cl = sorted(callers.get(la, []))
        out.write("%s:  ; %d callers\n" % (names[la], len(cl)))
    ann = []
    if la in funcs or (la in callers):
        cur_es = None
    mm = re.match(r"mov es, word ptr \[0x([0-9a-f]+)\]", text)
    if mm:
        v = segval(int(mm.group(1), 16))
        cur_es = v
        text = "mov es, SEG_%04x" % v
    elif text.startswith("les ") or "pop es" in text or text.startswith("mov es,"):
        cur_es = None
    me = re.search(r"es:\[([^\]]*?)0x([0-9a-f]+)\]", text)
    if me and cur_es is not None:
        v = int(me.group(2), 16)
        g = gnames.get((cur_es, v)) or gnames.get(cur_es * 16 + v)
        ann.append(g if g else "G%04x_%04x" % (cur_es, v))
    # far call
    if by[0] == 0x9A:
        toff, tseg = struct.unpack_from("<HH", img, la + 1)
        text = "call far " + names.get(lin(tseg, toff), "%04x:%04x" % (tseg, toff))
    elif by[0] == 0xE8:
        t = int(text.split()[1], 16)
        text = "call " + names.get(lin(seg, t), hex(t))
    for m in re.finditer(r"0x([0-9a-f]+)", text):
        v = int(m.group(1), 16)
        if "ptr" in text and "[" in text and m.start() > text.find("["):
            if (dgroup, v) in gnames and "es:" not in text:
                ann.append(gnames[(dgroup, v)])
        if v >= 0x10:
            s = ds_string(v)
            if s and ("push" in text or "mov" in text):
                ann.append(repr(s))
    for m in re.finditer(r"\[0x([0-9a-f]+)\]", text):
        v = int(m.group(1), 16)
        g = gnames.get((dgroup, v))
        if g and "es:" not in text:
            text = text.replace(m.group(0), "[%s]" % g)
    line = "  %04x:%04x  %-44s" % (seg, o, text)
    if ann:
        line += " ; " + " ".join(ann)
    out.write(line.rstrip() + "\n")
out.close()
json.dump({names[k]: [v[0], v[1]] for k, v in funcs.items()}, open(name + ".funcs.json", "w"))
print("functions:", len(funcs), "insns:", len(insns))
