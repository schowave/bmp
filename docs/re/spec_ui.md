# Spec: the screens of BMMAIN.EXE

What the screens of the original look like and how they behave, collected
while the remake's screens were rebuilt against the running original. The
other specs cover the rules; this one is for whoever fixes a screen or a
screen test. Addresses are of the unpacked image (see
`remake/tools/re/CONTEXT.md`).

## 1. Comparing a screen with the original

The original runs in the browser through the WASM image (js-dos). The tools
are in `remake/tools/re/refgame/wasm` and `remake/tools/re/pixcmp`.

```sh
cd remake/tools/re/refgame/wasm && npm install && npx playwright install chromium
remake/tools/re/refgame/wasm/serve.sh --driver   # build wasm/dist, expose window.ci, serve :8000, driver on :9333
./cmd.sh '[["newgame","TEST"],["shot","/abs/menu.png"]]'
```

Without `--driver` serve.sh only serves :8000 (then `node driver.mjs` by
hand); the driver port is `BMP_DRIVER_PORT` (default 9333), for a second
driver. The driver says "ci ready" once BMMAIN runs: screenshots while DOSBox
boots, or while the game builds the season after the start, crash the js-dos
backend ("memory access out of bounds"), then restart serve.sh.

Driver commands (one JSON array per request, run in order): `["click",x,y]`,
`["rclick",x,y]`, `["move",x,y]` in game coordinates (320×240), `["type","TEXT"]`,
`["press","Enter"]`, `["wait",ms]`, `["shot","/abs/file.png"]` (the canvas of
the game, 2× = 640×480), `["eval","js"]` (e.g. `ci.sendMouseButton(0,true)` to
hold the left button). The mouse goes through the js-dos command interface
and lands exactly. `["state"]` names the screen from a few pixels (title,
setup, options, question, menu or unknown), `["waitfor","question/menu",ms]`
waits for one of them, `["newgame","NAME"]` goes from the title or the setup
to the main menu of a new game (the steps below), `["quit"]` stops the
driver and, with `--driver`, the server.

Into a game: wait ~15 s, click 160,120 (title), click the name field 40,229,
type a name, Enter, click 40,135 (club in the middle of the strip), right
click twice (options, start), then "KEIN GEDANKE" at 190,168 four times for the
cup draws. Main menu groups: x=30 at y=130/173/216 on the left, x=290 at
y=130/173 on the right; submenu items x=107/161/215, y=132/175/217. HAUPT MENU
or WEITER at 295,213. The turn ends on the calendar sheet at 275,75.

The remake side: `bmp -scale 1 -demo NAME -shot f.png` or
`-script file` (`internal/ui/script.go`: `click X Y`, `rclick X Y`, `move X Y`,
`type TEXT`, `key enter`, `wait N` frames, `shot FILE`). Demo names are in
`internal/ui/demo.go` (`start` runs a new game from the first cup draw,
`menu` the main menu after some events, `dfb0`/`uefa0` the first cup round).
The demo game is another random game, so compare layout, not data.

Compare with `pixcmp/diff.py original.png remake.png out.png` (pixel diff
picture) and `pixcmp/find.py original.png remake.png x0 y0 x1 y1 [r]` (where
a region of the remake lies in the original: the way to measure offsets when
the data differ). `idx.py` dumps palette indices row or column wise.

The decompilation is in `remake/tools/re/work/` (`mise run re:build`);
`remake/tools/re/q.py` finds strings (`str`), prints a function (`fn`, with the
listing when Ghidra failed, as for the main menu f_08bc_0b5f), its callers
(`calls`) and constants of the data segments (`seg D 24c4 ptrs:7`).

`mise run remake:shot DEMO OUT [SCRIPT]` takes one shot, `remake/tools/shots.sh`
renders every demo screen (CI uploads them as the artifact remake-shots).
Ebitengine needs a monitor: with the display asleep `bmp` fails with "no
monitor was found"; `caffeinate -u -t 2` wakes it (the mise task does this),
`caffeinate -d` keeps it awake. On Linux run under `xvfb-run`.

## 2. Drawing primitives

| Original | Meaning | Remake |
|---|---|---|
| f_06c7_00a7(img, x, y, w, h, buf, 0, fill) | window from x,y to x+w,y+h **inclusive**: bevel (left and top-left light, top colour 2, right 7, bottom 6) and a black shadow of 2 pixels right and below; w==0 draws icon button h | `g.Window(dst, x0, y0, x1, y1, fill)` |
| f_076b_0c91(img, n) | select font: 1 flow.font (8 px), 2 micro4.font (6 px), 3 newbold.font (16 px) | `FontNormal`, `FontSmall`, `FontBig` |
| f_06c7_087c(img, s, x0, x1, y, c, sh) | text centred at x0+(x1−x0)/2−w/2, y = baseline (bottom row of the capitals), shadow colour sh one pixel right (−1: none) | `g.TextMid` |
| f_06c7_07d5(img, s, x, y, c) | left-aligned text, baseline y | `g.TextAt`, `g.TextEnd` |
| f_06c7_09b0(img, x0, y0, x1, y1, c) | line | `FillRect` / `drawLine` |
| f_06c7_083c(img, x0, y0, x1, y1, c) | filled box, inclusive | `FillRect` |
| f_06c7_0753(img, x0, y0, x1, y1, c) | rectangle outline | `FrameRect` |
| f_3930_0991(src, sx, sy, dst, dx, dy, w−1, h−1) | blit | `Draw(dst, g.Sub(...))` |
| f_3091_146c | dialog with title, lines and two buttons | `NewQuestion`, `NewMessage` |
| f_3091_0e3a | 4-line alert without title, OKAY at 178,160 | `NewAlert`, `NewAlertText` |

Most light texts have a 1-pixel shadow to the right: colour 7 for light
text, colour 0 for grey text. Colours are indices of the UI palette:
0 black, 1 light grey-blue, 2..7 greys from light to dark, 8 blue (window
fill), 10 sand (dialog text), 11 beige, 16 dark red (dialogs), 17 red
(button text), 18 red, 19 orange, 23 yellow, 29 white.

Text in the original uses `[`=Ä, `]`=Ü, `{`=ä, `}`=ü, `~`=ß and `|`=**ö**:
there is no capital Ö, so capital words show the ö glyph ("H|CHSTER",
"M|CHTEN"). In Go the UI strings are written that way ("HöCHSTER"). Club and
player names come from MANA.DAT in CP437 and do have a capital Ö (cell `\`).

Icon buttons: frame PIC/24.VGA (PIC/25.VGA pressed), its window filled blue
inside a black line (x+9..x+40, y+9..y+31). The icon sheet PIC/3.VGA is a grid
of 6 columns and 24-pixel rows; a cell of 23 rows is drawn 9 pixels below the
top of the frame, horizontally centred. The cobweb (unavailable item) is the
only cell on black and is drawn keyed on blue. The HAUPT MENU / WEITER button
is at 268,193.

Dialog f_3091_146c: 156 pixels wide at x=75, centred on y=143, height
42 + 8·lines, title (flow, colour 11) at baseline y+9, line under it at y+11,
lines (micro, colour 10) from baseline y+21 every 8, line at h−21, buttons
57×12 at x+14 and x+84 (one button centred), text colour 17, 1 under the
pointer. Texts are centred over box and shadow (x0=75+2, x1=75+157).

## 3. Mouse

The left button selects. The right button does **not** leave screens: they
close with HAUPT MENU or WEITER only. The main menu does not react to
hovering (s4, s5). Clicked on the original (shots in
`remake/testdata/original`, see its README; – = not tried):

| Element | Left | Right |
|---|---|---|
| setup: club strip (arrows, logos) | scrolls, middle logo puts the club into the next slot (o07–a4, b4) | options box, anywhere in the setup (c1) |
| setup: name field | activates it, block cursor (b1) | – |
| setup: options box | AN/AUS buttons | starts the game (d1) |
| cup draw question | ABER KLAR / KEIN GEDANKE | – |
| cup draw running | skips to the complete draw (d6) | – |
| main menu: group (left and right column) | opens it (g1–g4, k0, u0) | closes the open group (w1) |
| main menu: manager tearing his hair | nothing | quit question, red face (m1) |
| main menu: calendar sheet 275,75 | ends the turn (the only way) | – |
| main menu: middle area | message panel (q5, q6), X closes, arrows page (v3–v7) | – |
| statistics, load screen | items | nothing (h1b, k3b) |
| sponsor screen | items (i1) | back to the main menu (i1b) |
| NEU | setup with the managers of the game (s2) | as in the setup (s3) |
| match day | plays the rest of the half at once | author of the last goal scene (n6–n8, u1–u3) |

During a match day the space bar opens the options (u4).

## 4. Screens

### Title, setup (f_08bc_2184)
- Title 47.CP with palette 2 of PIC/1.PAL; any click.
- Banner PIC/4.VGA at y=−3, drawn after the strip; strip PIC/34.VGA (309×65)
  at 5,40 with the logos at 5+12+41k, 47 (the frame of the middle cell is
  part of the picture); club name centred at baseline ≈ 92.
- Slots PIC/35.VGA at x=12,92,172,252, y=105; logo of the chosen club at +10,+9;
  face cell i*41+1 (39 wide) at +9,+65; name field: text left-aligned at
  slot+8, baseline 230, colour 19, block cursor 3×5 two pixels after the text.
- Strip: a left click left of x=160 scrolls one club back, right of it one on,
  holding keeps scrolling. The ids are shuffled within the leagues before the
  setup (f_0310_09c5), so the order is random.
- Right click: option box, window 78,69–242,211. Labels flow colour 1 shadow 0,
  centred in [83,182] at baseline button-y+12. Level buttons 24×17 at x=181/207,
  y=75+18·row: the AN cell of PIC/33.VGA with its text area filled with colour 6
  and the digit (colour 4, chosen 19). AN/AUS switches are PIC/33.VGA cells
  24×17 at x=100 (AN off), 124 (AUS off), 148 (AN on), 172 (AUS on), rows at
  y=134, 152, 170, 188. Right click again starts.
- A left click at y ≤ 35 (the banner) opens the load screen "Spielstand
  Laden" (f_322b_120c); a loaded game starts, else the setup comes back.
  Only at the start, not for NEU.
- NEU in the disk menu opens the same screen with the managers of the game;
  a right click goes back (no option box).

### Cup draws (f_17c2_0006, f_17c2_0378, f_17c2_089a)
New game: DFB, Landesmeister, Pokalsieger, UEFA in this order; season end
the same; every new round once per cup. Marble, title (big font, colour 11,
shadow 0, centred in [0,320] at baseline 20), the question "MöCHTEN SIE DEM
UNÜBERTROFFENEM / SCHAUSPIEL EINER DFB-POKAL- (EUROPA-POKAL-) / AUSLOSUNG
BEIWOHNEN ?" with ABER KLAR / KEIN GEDANKE. DFB semi-final and final: no
question, only the round. Board PIC/47.VGA at 19,26; plates 113×7 at x=37/167,
y=38+10·row, beige, name (DFB: "NAME (league)") in colour 0/7/26 for league
1/2/3; then "TASTE DRÜCKEN" (micro, white, shadow 0, centred [90,237] at 217),
click, the round. The options DfB-Pokal and Europapokale switch it off.

### Cup round (f_17c2_1ccb)
Lines = pairs+1 (+1 per empty DFB group; −6 for Europe); rowsH = 7·lines+58;
mid = (233−rowsH)/2; window 5,mid−15 to 313,mid−15+rowsH−8; line at y=mid−5
from 7 to 311; title (round name + cup name + "    D.M." of the next cup event,
or "    N.Minute") flow colour 11 shadow 7 in [5,313] at mid−7. Rows every 7
pixels from mid−20+16 (+8 for the DFB group headers, micro colour 2 shadow 0
centred in [0,299]). Home centred in [0,130], "GEGEN" colour 3 shadow 0 in
[0,280], away in [140,270], clubs of managers (DFB) or German clubs (Europe)
in beige; result "(h:a)" at 268 (Europe 249, first leg at 285). A click on a
club opens its information.

### Main menu (f_08bc_0b5f; the decompiler failed on it, read BMMAIN.lst)
- Header PIC/23.VGA at y=41; face at 15,59; matchday text centred in [66,172]
  at top 52; manager at top 92.
- Calendar sheet: weekday, month, year colour 4 with shadow 7, the day in the
  big font colour 6 with shadow 8, lines at y=54, 65, 84, 94.
- Box under the matchday: the table places of the manager's league after each
  matchday (manager +0x10c, written by f_2a41_2d33), y = 65 + place, from
  63,74 to x=177 in colour 11; before the second day a dark line at y=74.
- Left groups x=5, right x=266, rows y=110/153/196; submenu columns
  82/136/190, rows 112/154/196. The group clicked last keeps frame 25.
- Disk menu: LADEN, SPEICHERN, HIGHSCORE | OPTIONEN, NEU.
- The manager tearing his hair: a **right** click asks "MöCHTEN SIE / DAS SPIEL
  WIRKLICH / BEENDEN ?" LEIDER JA / KEIN GEDANKE (the icon shows the screaming
  manager of PIC/37.VGA, 0,31, 32×23, while it is open; LEIDER JA ends the
  program); a left click does nothing. The turn ends on the calendar sheet only (not
  while the message panel is open).
- Message panel f_3091_05c4: fill 0x3b,0x72–0xf5,0xea colour 4, outline
  0x39,0x70–0x106,0xea colour 7, three raised fields from y=0x71 every 0x29,
  buttons PIC/2.VGA (18×122, pressed column at x=0x13) at 0xf5,0x71. Messages
  (20 per manager, newest first, `^` separates lines) from baseline 0x7a: the
  date line colour 25 then +10, the other lines colour 24 then +8, +7 between
  messages; "Keine Nachricht vorhanden." colour 1 at 0xb1. Up/down scroll by one
  message, X closes. It opens by itself when new messages came (F:0x56e2) and a
  click in the free middle opens it.

### Flag with a word (f_0310_01f2)

The German flag of `f_0310_00fa` over the whole screen (beige border; black
1..0x4f, red 0x50..0xa0, gold 0xa1..0xee), the word in newbold centred on
baseline 0x80, colour 0x1d with shadow 0; then `f_3930_01e7(0x32)` (not with
the auto play D:0x5d4), no click needed. Shown before the matches of every
competition of a match day: "Ligaspiel" (`f_0310_0995`), "DFB-Pokal"
(`f_0310_0261`), "Europapokal" (`f_0310_0290`), "Relegationsspiel"
(`f_0310_02bf`), "Nachholspiele" (`f_0310_07a2`); in the loop `f_1a58_3176`
after the matches of an event whose day is inside the winter break (130 < day <
207), "WINTERPAUSE"; at the start of the season end (`f_1a58_4597`),
"SAISONENDE". The newbold font has capitals only, so "Ligaspiel" reads
LIGASPIEL. Compared with the original after the first turn: only the pointer
differs. The winter guard also reads the pixel at 1,1 (`f_3930_043a`, skip if
it is colour 8) and leaves colour 8 there (`f_3930_04fd`, `f_3930_03ce`); the
remake shows WINTERPAUSE once for a run of days without a turn (unverified in
the running original).

### Match day (f_0310_15db, f_0310_2303, f_1a58_042f, f_1a58_0ca3)
- One scoreboard per match of a human team at (0,0), (161,0), (0,115),
  (161,115): PIC/38.VGA 155×114 at x+3,y; names micro colour 1 centred in
  [x+5,x+0x4b] and [x+0x55,x+0x99] at y+12 (wider than 60: split at the first
  space, lines at y+7 and y+13); logos 38×38 at x+0x16/x+0x65, y+0x23; LED digits
  of PIC/38.VGA (d·14, 114, 11×15) at x+0x24/x+0x72, y+0x10; attendance centred in
  [x+9,x+0x33] at y+99, colour 17 when sold out, "(AUSW.)" for an away manager;
  the manager's counters (yellow, red, injured, players on the pitch, chances)
  at x+0x7e (+16 when away), y+0x52+7i on colour 12.
- Bottom: bars x 36..125 and 194..286 at y 235/236 in a bevel 35..126/194..287,
  234..237; the first half fills the left bar 2 px per minute, the second half
  the right bar from the right; "N.Minute" flow white, shadow 4, centred in
  [4,320] at 239. Minute delay f_3930_01e7(0x26 − speed/2). The pause
  f_3930_01e7(n) waits n/3 ticks of the BIOS timer (18.2 Hz): at speed 40 a
  minute lasts 6 ticks, about 0.33 s (0.35 s measured in the browser).
- Scene window for a chance, card or injury of a human team, in the other half
  (y = 113 for panels 0/1, 0 for 2/3): black, outline colour 1, logos at 0x14 and
  0x105, faces of managers below them, clocks of PIC/8.VGA at 115 and 193 at
  y+0x6c, minute at y+0x77, PIC/8.VGA (188×102) at 0x42,y+2. The goal scene of
  TORE/ plays in that frame (format and timing: `internal/assets/scene.go`,
  `internal/ui/goalscene.go`), then "Torschütze: N" / "nach Vorlage von M" or
  "Chance vergeben: N" (flow colour 1), the TOR icon of PIC/8.VGA at the
  scorer's side; cards "Gelbe Karte: N" (colour 23), "Rote Karte"/"Gelb-Rote
  Karte"/"Verletzt" (colour 18, shadow 16) with their icons. With Torszenen off
  "Chance für X !" (colour 11) blinks four times (f_3930_01e7(6) dark, then
  lit). The result stays f_3930_01e7(0x78 − speed), a card or an injury
  0x82 − speed; the ticker line f_3930_01e7(0x46).
- A goal of a club marked ANZEIGEN: "HOME - AWAY   h:a" white, centred, at 237.
- After each period: the results of every league whose Halbzeitstände (45) or
  Ergebnisse (90) option is on (f_2a41_120a), on cup days the cup round with
  the minute. After the match day the tables (f_2a41_214c, option Tabelle),
  then the newspaper of each manager (f_2e3a_23aa, option Zeitung).

### Other screens
The league screens (f_2a41_214c, f_2a41_120a, f_2a41_3891), desk screens,
team screens, lineup (f_1ecd_1560), load/save (f_322b_2bbe), high scores
(f_322b_2366), options (f_262f_01b8), club information (f_2a41_000e),
newspaper and the notices are described in the comments of their files in
`remake/internal/ui`; their layouts were compared pixel by pixel.

## 5. Reference screenshots

The shots of the original taken for this work are listed in
`remake/testdata/original/README.md`; they are not part of the repository
(they show the game's graphics) and can be taken again with the driver.

## 6. Known differences

- Messages: the original dates panel messages today minus rand(0,3) days; the
  remake dates them today. (Their expiry and their deletion with the slot they
  belong to are in the remake: spec_players §3.1.)
- The best series of the statistics screen start again every season.
- The Christmas/birthday cards, credit dialogs and the season report were
  built from the decompilation only; reaching them in the original takes many
  turns.
- The club information opened from the results screen sits a few pixels
  further right in the original (param_3 = 1).
- Contract list: players who will not negotiate are not greyed out (colour 3),
  retiring players not drawn in colour 17; the dot after a player on loan is
  missing.
