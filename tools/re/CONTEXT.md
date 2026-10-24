# BMMAIN.EXE reverse-engineering context (shared)

Game: Bundesliga Manager Professional (Software 2000 / Kron, DOS, ~1993/94, "V2.0").
Goal of the overall project: a faithful re-implementation of the game logic in Go.
Your job as an analyst: read the decompiled code and write a precise spec (formulas,
constants, data layouts, control flow) so that a programmer can re-implement the
behaviour WITHOUT reading the original code. Cite function addresses for every claim.

## Files (all in the work directory created by tools/re/run.sh)

- `BMMAIN.ann.c`  Ghidra decompilation of all functions (C-like), annotated: string
  pointer-table reads get a trailing comment `/* "TEXT" */` (exact slot) or `~"TEXT"`
  (base of a table indexed at runtime, i.e. the table STARTS near that text).
  Functions are delimited by `// ==== f_SSSS_OOOO @ ...`.
- `BMMAIN.lst`    Plain disassembly (capstone) of the same code, one function per block,
  delimited by `f_SSSS_OOOO:`. This listing KNOWS WHICH DATA SEGMENT is used:
  `mov es, SEG_4238` then `es:[0x1234]` is annotated `G4238_1234` (= F segment) and
  `G4cb3_xxxx` (= D segment). Use it whenever the C is ambiguous about the segment.
- `slots.json`    D-segment offset -> string, for far-pointer string tables.
- `BMMAIN.bin`    The unpacked load image (raw bytes). Linear address = seg*16+off.
  F segment base = 0x42380, D segment base = 0x4cb30. You may read initialised data
  (tables of constants!) from it with python, e.g.
  `python3 -c "d=open('BMMAIN.bin','rb').read(); print(list(d[0x42380+0x225e:0x42380+0x2262]))"`
- `funcs_summary.txt` per function: #insns, #callees, some strings.
- `strings` of game texts: umlauts are encoded as `[`=Ä `|`=Ö `]`=Ü `{`=ä `}`=ü `~`=ß
  in UI text (fonts), and as CP437 (0x8e Ä, 0x9a Ü, 0x81 ü, 0x84 ä, 0x94 ö, 0xe1 ß) in data.

## Segments / addresses

- Function names `f_SSSS_OOOO` use the segment of the unpacked image (load base 0).
  In Ghidra's C output, addresses are shown with +0x1000 on the segment
  (`f_1a58_3176 @ 2a58:3176`), and far pointers to data contain `0x5238` (= F segment
  0x4238) or `0x5cb3` (= D segment / DGROUP 0x4cb3).
- Two data segments matter: **F = 0x4238** (far data, most game state) and **D = 0x4cb3**
  (DGROUP, near data, string tables). In the Ghidra C, `*(char *)0x304a` does NOT say
  which one: the compiler loaded ES from a slot like `[0x9da0]` first. Check BMMAIN.lst.
  Rule of thumb: game state structs are in F.
- Large memory model, MS C 6. All functions are far cdecl, args pushed right-to-left.

## Library / helper functions (already identified)

- `f_3a01_02c6` = stack check (ignore). `f_3a01_0e18` = strcpy, `f_3a01_0dd2` = strcat,
  `f_3a01_0e4e` = strcmp, `f_3a01_0e78` = strlen, `f_3a01_17be` = rand() (MS C LCG:
  seed = seed*214013+2531011; return (seed>>16)&0x7fff), `f_3a01_1b18`/`1bd0`/`1ae6`/`1c10`
  = 32-bit div/mod/mul helpers (long arithmetic), `f_35ac_304a(src,dst,n)` = memcpy.
- `f_076b_0cc7(a,b)` = random integer in [a,b] inclusive: rand() % (b-a+1) + a.
- `f_076b_0681(lo,hi)` = long -> decimal string (with thousands dots).
- `f_3930_0023` open, `f_3930_0088` read, `f_3930_00d6` write, `f_3930_000c` close,
  `f_3930_010c` farmalloc, `f_3930_0188` farfree.
- `f_076b_0356` = load picture (pic\N.vga), `f_06c7_*`, `f_3930_05xx..0cxx` = graphics,
  `f_06c7_087c`/`07d5` = draw text. `f_3091_146c` = dialog/message box, `f_3091_105d` = text
  formatting (umlaut translate). UI code is not interesting except where it applies rules.

## Global state (from the savegame writer f_322b_07fe — complete list of persisted state)

F-segment unless noted:
- `F:0x304a` (1 byte) current manager index (0..3); `D:0x7ab` number of human managers.
- `F:0x5358` also current manager (copy).
- `F:0x3066 + t*0x22`, t=0..199: TEAM table, 200 teams, 34 bytes each.
    +0x00 name (23 bytes, from MANA.DAT)
    +0x17 (0x307d abs) random 45..55 at load
    +0x18..+0x1a (0x307e..0x3080) 3 strength values (current), from MANA.DAT
    +0x1b..+0x1d (0x3081..0x3083) 3 strength values (base copy)
    +0x1e..+0x20 (0x3084..) random 45..55 (minus 5 for teams > 64)
    +0x21 (0x3087) logo picture number (teams 0..63, 80..143 = PIC\N.VGA)
  Teams 0..63 are German clubs (with 20 player names each in MANA.DAT); 64..199 are
  foreign clubs (European cups) with names + strengths only.
- `F:0x0ecc + t*0x36`, t=0..63: per German club extended record (54 bytes),
  includes season stats (fields at +0x00/+0x16/+0x1a/+0x1e/+0x26/+0x2a per home/away,
  +0x32/+0x34 long at 0xefe/0xf00, string at +4 (0xed0) = "@@@@@@@@"..).
- `F:0x2242..` (0xc28 bytes) = 4 x 0x30a MANAGER records starting at F:0x2242?;
  the code addresses them as `m*0x30a + 0x2260` (+0x2260 = team id of manager m),
  `+0x237a` = league of manager (0=1.Liga,1=2.Liga,2=Amateur-Oberliga),
  `+0x2374..` , `+0x2280 + k*4` history, `+0x2432/0x2434` long (money?), `+0x241e`...
- `F:0x57dd + i*0x25`, i=1..150: PERSON table (players), 37 bytes:
    +0x00 name, +0x1a (0x57f7) age, +0x1b (0x57f8) avg of the next 3,
    +0x1c/+0x1d/+0x1e (0x57f9..0x57fb) 3 skill values, +0x1f (0x57fc) position code:
    value/25 = position (0 Torwart, 1 Abwehr, 2 Mittelfeld, 3 Angriff),
    +0x20 (0x57fd) 0..6, +0x21 (0x57fe) owner manager (5 = none?), +0x24 (0x5801) team.
  New-game init: f_322b_035c.
- `F:0x774a + (m*0x19 + s)*0x34`: SQUAD slots, 25 per manager, 52 bytes:
    +0x09 status bits (&3: 1 = suspended with +0x0d games left, 2 = injured, +0x17 injury kind)
    +0x0a (<0x0c: in team (shirt no?), 0 = "hängt halt so rum", >=0x0c reserve)
    +0x0c on loan, +0x0e tendency raw ((x-0x1e)/0xd -> falling/const/rising),
    +0x0f person index (into PERSON table), +0x10/+0x11/+0x12 Kondition/Technik/Form,
    +0x13 Erschöpfung.
- `F:0x535a + l*0x14`: team ids of league l (l=0,1,2), `D?/F:0x225e[l]` teams per league.
- `D:0x7dc/0x7de` (long) day counter, `F:0x2e8e`, `F:0x2eba` calendar values,
  `F:0x16e` index into `D:0x2280` (event flags per date: bit0..2 league l plays,
  bit3 DFB-Pokal, bits4..6 Europapokal, bit7 Nachholspiele).
- `F:0x9336` (0x1394 bytes), `F:0x6ddc` (0x8e8), `F:0x4b5e`, `F:0x5714`, `F:0x76ca`,
  `F:0x21e2`, `F:0x4c7e`, `F:0x2ec2`, `F:0x002e`, `D:0x66c`... are unknown yet.

## Main structure (already known)

- main = f_08bc_0418 -> new game / load (f_08bc_2184 options) -> f_1a58_3176 = the game
  loop (days, matchdays, season end). Per manager per week: f_08bc_0b5f (manager's
  turn / main menu, also autosave). Season end summary: f_1a58_4597 ("SAISONENDE").
- MANA.DAT loader f_262f_36ec: 64 bytes logo ids, 200x3 strengths, 320 bytes table,
  then 64x(23 name + 20x26 player names) + 136x23 names.

## Output

Write your findings as Markdown to the file named in your task. Structure:
data layouts (offset, size, meaning), algorithms as numbered steps or pseudo-code with
exact constants, function address list with one-line meaning. Mark anything uncertain
as UNCERTAIN. Prefer exactness over breadth, but cover the whole topic. Tables of
constants in the data segment: dump them (python on BMMAIN.bin) and include the values.
Do not modify any file other than your output file (and scratch files of your own).
