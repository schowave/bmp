#!/bin/sh
# Regenerates the reverse-engineering work directory from the original
# BMMAIN.EXE: unpacked image, disassembly listing and (with Ghidra) the
# annotated decompilation the specs in docs/re were written from.
#
# usage: tools/re/run.sh [workdir]      (default: tools/re/work, ignored by git)
# needs: python3 with capstone; optional GHIDRA=<ghidra install dir>
set -eu
here=$(cd "$(dirname "$0")" && pwd)
root=$(cd "$here/../../.." && pwd)
work=${1:-$here/work}
mkdir -p "$work"
work=$(cd "$work" && pwd)

python3 "$here/unexepack.py" "$root/bmp/BMMAIN.EXE" "$work/BMMAIN"
cd "$work"
python3 "$here/disasm16.py" BMMAIN 4cb3
python3 "$here/mkmz.py" BMMAIN
python3 -c "
import json
f=json.load(open('BMMAIN.funcs.json'))
open('BMMAIN.starts','w').write('\n'.join('%04x:%04x'%(s,o) for s,o in f.values()))"

if [ -n "${GHIDRA:-}" ]; then
    mkdir -p ghidra-project
    "$GHIDRA/support/analyzeHeadless" "$work/ghidra-project" bmmain \
        -import "$work/BMMAIN_unp.exe" -overwrite -scriptPath "$here/ghidra" \
        -preScript PreSetup.java "$work/BMMAIN.starts" \
        -postScript DecompAll.java "$work/BMMAIN.starts" "$work/BMMAIN.c" > ghidra.log 2>&1
    python3 "$here/annotate.py"
    echo "decompilation: $work/BMMAIN.ann.c"
else
    echo "GHIDRA not set: skipped the decompilation"
fi
echo "listing: $work/BMMAIN.lst"
