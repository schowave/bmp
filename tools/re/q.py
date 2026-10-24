"""Queries the reverse-engineering work directory (tools/re/work, made by
run.sh / `mise run re:build`).

    q.py str PATTERN            strings of slots.json containing PATTERN, deduplicated
    q.py fn NAME [--full]       a function of BMMAIN.ann.c (declarations left out
                                unless --full); the listing when the decompiler failed
    q.py lst NAME               the function in the disassembly BMMAIN.lst
    q.py calls NAME             the functions that call NAME
    q.py seg D|F OFF FMT        constants of a data segment: FMT is a struct format
                                ('<5h'), 'str' (C string) or 'ptrs:N' (N far
                                pointers to strings, as the string tables use)

D is DGROUP (0x4cb3), F the far data segment (0x4238); offsets in hex.
WORK=<dir> uses another work directory.
"""
import json
import os
import re
import struct
import sys

WORK = os.environ.get("WORK", os.path.join(os.path.dirname(os.path.abspath(__file__)), "work"))
SEG = {"D": 0x4CB3, "F": 0x4238}
DECL = re.compile(r"^  (undefined\d?|int|uint|char|byte|bool|ulong|long|short)\b.*;$")


def path(name):
    p = os.path.join(WORK, name)
    if not os.path.exists(p):
        sys.exit(f"{p} missing: run `mise run re:build` (GHIDRA set for the decompilation)")
    return p


def functions(name):
    """Yields (function name, lines) of BMMAIN.ann.c."""
    cur, lines = None, []
    with open(path(name), errors="replace") as f:
        for line in f:
            m = re.match(r"^// ==== (f_[0-9a-f]+_[0-9a-f]+)", line)
            if m:
                if cur:
                    yield cur, lines
                cur, lines = m.group(1), []
            lines.append(line.rstrip("\n"))
    if cur:
        yield cur, lines


def listing(fn):
    out, on = [], False
    with open(path("BMMAIN.lst"), errors="replace") as f:
        for line in f:
            if re.match(r"^f_[0-9a-f]+_[0-9a-f]+:", line):
                if on:
                    break
                on = line.startswith(fn + ":")
            if on:
                out.append(line.rstrip("\n"))
    return out


def cmd_str(pat):
    seen = {}
    for k, v in json.load(open(path("slots.json"))).items():
        if pat.lower() in v.lower():
            seen.setdefault(v, []).append(k)
    for v, keys in seen.items():
        print(f"{v!r}  slots {','.join(hex(int(k)) for k in keys[:5])}{' ...' if len(keys) > 5 else ''}")


def cmd_fn(fn, full):
    for name, lines in functions("BMMAIN.ann.c"):
        if name == fn:
            if any("decompile failed" in l for l in lines[:4]):
                print("\n".join(lines[:4]) + "\n(decompiler failed, listing:)")
                print("\n".join(listing(fn)))
                return
            print("\n".join(l for l in lines if full or not DECL.match(l)))
            return
    sys.exit(f"{fn} not found")


def cmd_calls(fn):
    for name, lines in functions("BMMAIN.ann.c"):
        hits = [i for i, l in enumerate(lines) if fn + "(" in l and not l.startswith("// ====")]
        if name != fn and hits:
            print(f"{name}: {len(hits)} call(s)")
    with open(path("BMMAIN.lst"), errors="replace") as f:
        cur = None
        for line in f:
            if re.match(r"^f_[0-9a-f]+_[0-9a-f]+:", line):
                cur = line.split(":")[0]
            elif "call" in line and fn in line:
                print(f"  lst {cur}: {line.strip()}")


def cmd_seg(seg, off, fmt):
    d = open(path("BMMAIN.bin"), "rb").read()
    base = SEG[seg.upper()] * 16 + int(off, 16)

    def cstr(a):
        return d[a:d.index(b"\0", a)]

    if fmt == "str":
        print(repr(cstr(base)))
    elif fmt.startswith("ptrs:"):
        for i in range(int(fmt[5:])):
            o, s = struct.unpack_from("<HH", d, base + 4 * i)
            print(i, f"{s:04x}:{o:04x}", repr(cstr(s * 16 + o)))
    else:
        print(struct.unpack_from(fmt, d, base))


def main(a):
    if len(a) < 2:
        sys.exit(__doc__)
    if a[0] == "str":
        cmd_str(a[1])
    elif a[0] == "fn":
        cmd_fn(a[1], "--full" in a)
    elif a[0] == "lst":
        print("\n".join(listing(a[1])))
    elif a[0] == "calls":
        cmd_calls(a[1])
    elif a[0] == "seg" and len(a) == 4:
        cmd_seg(a[1], a[2], a[3])
    else:
        sys.exit(__doc__)


if __name__ == "__main__":
    main(sys.argv[1:])
