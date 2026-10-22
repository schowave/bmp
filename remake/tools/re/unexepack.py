"""Unpack a Microsoft EXEPACK-compressed MZ executable.

Writes the unpacked image as a flat binary (load image) plus a JSON with
entry point, stack and relocations, and also a rebuilt MZ file.
"""
import json
import struct
import sys


def unpack(path, out):
    d = open(path, "rb").read()
    (magic, cblp, cp, crlc, cparhdr, minalloc, maxalloc, ss, sp, csum, ip, cs,
     lfarlc, ovno) = struct.unpack_from("<2s13H", d, 0)
    assert magic == b"MZ"
    size = cp * 512 - ((512 - cblp) if cblp else 0)
    hdr = cparhdr * 16
    image = bytearray(d[hdr:size])
    epos = cs * 16
    # EXEPACK header
    if ip == 0x12:
        fields = struct.unpack_from("<8H", image, epos)
        real_ip, real_cs, _mem, exe_size, real_sp, real_ss, dest_len, skip_len = fields
        sig = image[epos + 16:epos + 18]
    else:
        fields = struct.unpack_from("<7H", image, epos)
        real_ip, real_cs, _mem, exe_size, real_sp, real_ss, dest_len = fields
        skip_len = 1
        sig = image[epos + 14:epos + 16]
    assert sig == b"RB", sig
    exepack = image[epos:epos + exe_size]
    comp = bytearray(image[:epos - (skip_len - 1) * 16])
    dst = bytearray(dest_len * 16)
    s = len(comp)
    while s > 0 and comp[s - 1] == 0xFF:
        s -= 1
    dp = len(dst)
    while True:
        cmd = comp[s - 1]; s -= 1
        length = comp[s - 2] | (comp[s - 1] << 8); s -= 2
        op = cmd & 0xFE
        if op == 0xB0:
            val = comp[s - 1]; s -= 1
            dst[dp - length:dp] = bytes([val]) * length
            dp -= length
        elif op == 0xB2:
            dst[dp - length:dp] = comp[s - length:s]
            dp -= length
            s -= length
        else:
            raise ValueError("bad cmd %02x at %d" % (cmd, s))
        if cmd & 1:
            break
    # leading uncompressed part: dst[:dp] should equal comp[:s]
    dst[:s] = comp[:s]
    # relocations: after the stub code and "Packed file is corrupt" message
    idx = exepack.find(b"Packed file is corrupt")
    rp = idx + len(b"Packed file is corrupt")
    relocs = []
    for seg in range(16):
        n = struct.unpack_from("<H", exepack, rp)[0]; rp += 2
        for _ in range(n):
            off = struct.unpack_from("<H", exepack, rp)[0]; rp += 2
            relocs.append((seg * 0x1000, off))
    meta = dict(cs=real_cs, ip=real_ip, ss=real_ss, sp=real_sp,
                minalloc=minalloc, relocs=relocs, size=len(dst))
    open(out + ".bin", "wb").write(dst)
    json.dump(meta, open(out + ".json", "w"))
    print(path, "-> unpacked %d bytes, %d relocs, entry %04x:%04x ss:sp %04x:%04x"
          % (len(dst), len(relocs), real_cs, real_ip, real_ss, real_sp))


if __name__ == "__main__":
    unpack(sys.argv[1], sys.argv[2])
