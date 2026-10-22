# Reverse-engineering tools

These scripts took the original `BMMAIN.EXE` apart. The specifications that
came out of it are in `docs/re/`; the Go engine in `remake/internal/game`
implements them.

| Script | What it does |
|---|---|
| `run.sh` | Runs the whole pipeline into a work directory (see below). |
| `unexepack.py` | Unpacks the Microsoft EXEPACK compression of the executables. |
| `disasm16.py` | Recursive-descent disassembly with capstone. Marks which data segment an access goes to (`G4238_xxxx` = far data, `G4cb3_xxxx` = DGROUP) and annotates strings. |
| `mkmz.py` | Rebuilds an unpacked MZ file with relocations, for Ghidra. |
| `ghidra/PreSetup.java`, `ghidra/DecompAll.java` | Headless Ghidra: creates the functions found by `disasm16.py`, enables parameter ID, decompiles everything. |
| `annotate.py` | Adds the texts of the far-pointer string tables to the decompiled C. |
| `fpdis.py` | Disassembles code that uses the 8087 emulator interrupts (INT 34h–3Dh), e.g. the attendance formula. |
| `strmap.py`, `topng.py` | Strings per function; pictures as PNG. |
| `refgame/ctl.py`, `refgame/tcpproxy.py` | Remote control of the original game in the VNC container for reference screenshots (see below). |
| `CONTEXT.md` | The shared brief for the analysis: segments, helpers, data structures. |

## Pipeline

```sh
python3 -m venv .venv && .venv/bin/pip install capstone
PATH=.venv/bin:$PATH GHIDRA=/path/to/ghidra_12.x remake/tools/re/run.sh /tmp/re-work
```

Ghidra's decompiler has no macOS arm64 binary in the release; build it
with `make ghidra_opt` in `Ghidra/Features/Decompiler/src/decompile/cpp`
(an empty `bfd.h` on the include path is enough) and copy the result to
`Ghidra/Features/Decompiler/os/mac_arm_64/decompile`.

## Reference game

The VNC image of this repository runs the original. DOSBox only gets
relative mouse motion, so `ctl.py` looks for the pointer in screenshots
and moves it in small steps. The VNC server listens inside the container
only; `tcpproxy.py` forwards port 5902 to it:

```sh
podman run -d --name bmp-ref -p 5991:5902 bmp
podman cp remake/tools/re/refgame/tcpproxy.py bmp-ref:/tmp/
podman exec -d bmp-ref python3 /tmp/tcpproxy.py
pip install vncdotool pillow
BMP_SHOTS=shots python3 remake/tools/re/refgame/ctl.py click 320 300 3   # right click
BMP_SHOTS=shots python3 remake/tools/re/refgame/ctl.py shot menu.png
```

Coordinates are in the 640×480 VNC picture (twice the 320×240 of the game).
