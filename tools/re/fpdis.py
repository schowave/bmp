"""Disassemble a range of BMMAIN.bin with the 8087 emulator interrupts of
the MS C runtime (INT 34h..3Dh) turned back into FPU instructions.

usage: fpdis.py BMMAIN.bin SEG OFF END   (hex)
"""
import sys
from capstone import Cs, CS_ARCH_X86, CS_MODE_16
img=open(sys.argv[1],'rb').read()
seg=int(sys.argv[2],16); off=int(sys.argv[3],16); end=int(sys.argv[4],16)
b=bytearray(img[seg*16+off:seg*16+end])
i=0
segpfx={0:0x26,1:0x2e,2:0x36,3:0x3e}
while i<len(b)-1:
    if b[i]==0xCD and 0x34<=b[i+1]<=0x3B:
        b[i]=0x9B; b[i+1]=0xD8+(b[i+1]-0x34); i+=3; continue
    if b[i]==0xCD and b[i+1]==0x3C:
        # next byte: ESC with seg encoded in bits 7..6 (MS C convention)
        x=b[i+2]; s=(x>>6)&3
        b[i]=0x9B; b[i+1]={0:0x26,1:0x2e,2:0x36,3:0x3e}[3-s] if False else segpfx[s]; b[i+2]=(x|0xC0)&0xDF|0xD8
        i+=4; continue
    if b[i]==0xCD and b[i+1]==0x3D:
        b[i]=0x90; b[i+1]=0x9B; i+=2; continue
    i+=1
md=Cs(CS_ARCH_X86,CS_MODE_16)
for ins in md.disasm(bytes(b),off):
    print('%04x:%04x  %-8s %s'%(seg,ins.address,ins.mnemonic,ins.op_str))
