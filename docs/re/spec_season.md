# Spec: game setup, calendar, season structure and competitions (BMMAIN.EXE)

Notation: `F:x` = far data segment 0x4238, `D:x` = DGROUP 0x4cb3 (segment checked in BMMAIN.lst).
`M(m)+x` = manager record field: absolute address `F:(m*0x30a + x)`, m = 0..3 (the code writes
`m*0x30a + 0x2260` etc.; offsets below are given as those absolute bases, i.e. `M(m)+0x2260` = team id).
`rand(a,b)` = f_076b_0cc7 = `rand() % (b-a+1) + a` (MS C LCG). All "ids" are TEAM-table indexes (0..199).
"e" = event index (F:0x16e), "day" = day counter D:0x7dc (long).

---------------------------------------------------------------------------------------------------

## 0. Constant tables (dumped from BMMAIN.bin, D segment, initial values)

| Address | Value(s) | Meaning |
|---|---|---|
| D:0x224c | 0 (byte, never written) | day offset of the "Saturday" event inside a week. Weekend event of half-week pair = day ≡ D:224c (mod 7), midweek event = day ≡ 4 (mod 7). |
| D:0x224d | 0 (byte, never written) | build/variant flag. 0 in this (German) build. Used: winter-break window, table tie-break (see 4.2), money text suffix. Treat as constant 0. |
| D:0x224e (long), D:0x2252 (long) | 1, 1 | build/locale constants (currency factor numerator/denominator and language code: `D:2252==3` switches English ordinal suffixes "st/nd/rd" in f_0310_0fb6/f_0310_1fe7; `D:224e==4` shifts the winter-break warning date). In this build both are 1 ("DM, German"). The option "Histor. Start" is only offered when D:2252 == D:224e. Treat as constants. |
| D:0x2256 | 76000 (long) | DFB final attendance (see §6). |
| D:0x225a[3] | 1,1,1 | **current matchday per league** (1-based, = next matchday to be played). Reset to 1 at season end. |
| D:0x225e[3] | 18, 20, 20 | teams per league (1. Liga, 2. Liga, Amateur-Oberliga). |
| D:0x2262[3] | 34, 38, 38 | matchdays per league (= 2*(teams-1)). |
| D:0x2266[3] | 3, 4, 4 | regular number of relegated teams per league (for l=0 incl. the relegation-playoff place). D:0x2265+l (i.e. D:2266[l-1]) = number of promotion places shown in the table of league l. |
| D:0x226a[3] | 1, 0, 0 | relegation playoff enabled for league l (only 1. Liga vs 2. Liga). |
| D:0x226d | 4 | number of UEFA-Cup places from the 1. Liga table (also used for table colouring). |
| D:0x226e[3] | 0, 18, 38 | first team id of league l (team ids of league l = D:226e[l] .. D:226e[l]+D:225e[l]-1). Also used as "position offset" in the history/eternal table (overall rank = position + D:226e[league]). |
| D:0x226f, D:0x2270 | 18, 38 | f_17c2_1bb5(id): 1 if id<18, 2 if id<38, else 4 (league bitmask). |
| D:0x2271 | 2 | **points per win** (2-point rule). Draw = 1 point each (hard-coded). Table shows "x:y" plus/minus points when D:2271<3. |
| D:0x2272..0x2279 | (0,17),(18,37),(38,57),(58,255) | id range per league class (class 3 = non-league clubs 58..63). |
| D:0x227a..0x227f | 's','S','n','N','u','U' | form-string letters, see 4.1. |
| D:0x2280[93] | see §2.3 | event table, one byte per half-week event. |
| D:0x06fc + 4*k (k=1..17) | far ptrs | 18-team fixture strings (k=0 is an empty string). |
| D:0x0744 + 4*k (k=1..19) | far ptrs | 20-team fixture strings (k=0 empty). |
| D:0x07b8[12] | 31,28,31,30,31,30,31,31,30,31,30,31 | month lengths, index 0 = January (no leap years). |
| D:0x23fc + 4*m | far ptrs | month names "Januar".."Dezember" (m = 0..11). |
| D:0x242c + 4*w | far ptrs | weekday names, w=0 "Montag" .. 6 "Sonntag". |
| D:0x07e0 | 0x07ab = 1963 (word) | year in which the current season started (set to 1992 unless "Histor. Start"), +1 per season. |
| D:0x07e2 | 0 (word) | number of completed seasons. |
| D:0x07ac..0x07b0 | 7,0,0,0,0 | cup-holder references (team id + 1, 0 = none), see §6. |
| D:0x53d0[3] / D:0x53d4[3] | days 24,19,12 / months 12,11,4 | special dates: 24 Dec (Christmas), 19 Nov, 12 Apr (birthdays of the two programmers). |
| D:0x4a28 | 2 | Spiel-Level (internal value, see 1.2). |
| D:0x05fe..0x060c | 1,1,1,1,1,1,1,1,1,1,1,1,1,1,0 | display toggles (see 1.3). |
| D:0x063a | 40 (word) | Spielgeschwindigkeit slider value, 0..70. |

Run-time state:

| Address | Meaning |
|---|---|
| D:0x7dc (long) | day counter within the season. 0 = first event day of the season (29 July). Reset to D:224c (=0) at season start. |
| F:0x2e8e (long) / F:0x2eba (long) | current month (0=Jan..11=Dec) / day of month (1..31). Recomputed from the day counter by f_0310_1190(0). |
| F:0xa7a0 (long) | current calendar year (computed, see 2.1). |
| F:0x016e (byte) | current event index e (0..92) into D:2280. |
| F:0x513c (word) | year in which the game ends (see 1.1). |
| F:0x56f4 (byte) | 1 = modern start (1992), 0 = historic start (1963). |
| F:0x535a + l*0x14 + i | league table of league l: team id at position i (0 = top). |
| F:0x4b5e + (l*10+i)*2 | fixture buffer: [home id, away id] of pair i of the matchday last computed for league l (f_2a41_10e3). Byte 0x46 ('F') as home id = end-of-list marker. |
| F:0x6dc8 + ((l*0x26 + md)*10 + i)*2 | result of league l, matchday md (1-based), pair i: [home goals, away goals]. Equivalent: F:0x6ddc + ((l*38 + md-1)*10+i)*2; whole block F:0x6ddc, 0x8e8 bytes = [3][38][10][2]. Home byte 0x1e = "game postponed / not yet counted". Zeroed at season start. |
| F:0x5714 + r*5 (r=0..19) | postponed-game records: +1 event slot (0 = free), +2 matchday, +3 league, +4 pair index. |
| F:0x0ecc + t*0x36 (t=0..63) | per-club season statistics, see 4.1. |
| M(m)+0x237a | league of manager m's club (0,1,2). |
| M(m)+0x234d + k (k=0..37) | table position (0-based) of the manager's club after matchday k+1 (written by f_2a41_2d33). Initialised to 10 for new managers. |

---------------------------------------------------------------------------------------------------

## 1. New game setup

### 1.1 Program start / new game sequence (f_08bc_0418)

1. Graphics init (f_08bc_0004 — NOT fixture related, it loads the palette), MANA.DAT load f_262f_36ec:
   - team logo ids, strengths (TEAM+0x18..0x1a and base copy +0x1b..0x1d), random TEAM+0x1e..0x20
     = rand(45,55) − (id > 64 ? 5 : 0) (note: id **> 64**, so team 64 is not reduced), TEAM+0x17 = rand(45,55)
     (per team: 3 calls for the 3 values then 1 call for +0x17, in id order 0..199);
   - the 320-byte table in MANA.DAT is read and ignored;
   - **initial league composition**: `F:535a[l][i] = D:226e[l] + i`, i.e. ids 0..17 = 1. Liga,
     18..37 = 2. Liga, 38..57 = Amateur-Oberliga, ids 58..63 = non-league reserve clubs
     (MANA.DAT order; see list in Appendix A).
2. D:7ab = 0; f_08bc_01fb (clears F:9336 head-to-head table, sets M(m)+0x2376..0x2378 = 0x1e,
   M(m)+0x225f = m+1); **f_0310_09c5 (id shuffle, 1.3)**; f_17c2_0ef2 (European qualification, §6);
   f_17c2_1b6d; f_17c2_09e0(c,0) for c=1..3 (European draws).
3. f_08bc_2184(…) = new-game dialog (or load). Returns 0 for a new game.
4. New game: D:225a = 1; f_14a4_19fb(); f_17c2_09e0(0,0) (DFB draw); f_17c2_09e0(c,1) c=1..3;
   day = D:224c (=0); f_2277_1e38(); f_14a4_0419(); then the main loop f_1a58_3176.

### 1.2 New-game dialog f_08bc_2184

Number of managers D:7ab = 1..4 (entered names, M(m)+0x2242 = name, 0x13 chars). Unused manager
slots get team id 100. Team choice: the player scrolls through the 64 German club logos (ids 0..63,
any league) and clicks one for each manager; a team cannot be taken twice (M(m)+0x2260 = chosen id).
Managers can swap order (f_08bc_1e96).

Option box (only for a fresh game, local_38 == 0; f_262f_0004 checkboxes):

* **Spiel-Level**: 6 buttons in 3 rows × 2 columns showing the numbers 0..5; button index
  k = 2*row + (x > 0xcd). Default k = 2 (D:4a28 initial value). On OK: **D:4a28 = 5 − k**
  (internal level L). Displayed level ("LEVEL n" in the in-game options) = 5 − L = k. So
  **internal L=5 is displayed as level 0 (easiest?)**, L=0 = displayed level 5. (Which end is
  "easy" must be confirmed from the formulas: e.g. start money 1.9M only for L==4, players' start
  skill +L/2; UNCERTAIN.)
* **Game length** radio bits (local bit0..2): Endlosspiel (bit0, default), 1-Jahres-Spiel (bit1),
  3-Jahres-Spiel (bit2); checkbox **Histor. Start** (bit3, default off; row only shown if D:2252==D:224e).
  Results:
  - `F:513c = 22222 (0x56ce)` endless; `1964 (0x7ac)` 1-year; `1966 (0x7ae)` 3-year.
  - `F:56f4 = (Histor. Start ? 0 : 1)`. If F:56f4 == 1: `F:513c += 29` (→ 22251 / 1993 / 1995)
    and **D:7e0 = 1992**; otherwise D:7e0 stays **1963**.
  - The game ends at the season end at which the calendar year (F:a7a0, i.e. the year of June of the
    season) == F:513c → 1 season (1-year) or 3 seasons (3-year). Endless never matches.
* Then D:4a28 = 5 − k, f_322b_035c() (persons init).

Per new manager m (f_08bc_2184, loop over managers):

1. Starting league `L0`: **2 (Amateur-Oberliga) for the endless game**, i.e. if
   (F:513c > 1966 and F:56f4==0) or (F:513c > 1995 and F:56f4==1); **0 (1. Liga) for 1- and 3-year games**.
2. If the 1-year game: D:0x68c+m*0x24 = 30000 (long), D:0x66c+m*0x24 = 180000 (long),
   D:0x66c+(m*9+k)*4 = 45000 for k=1..6 (instead of the defaults 50000 / 6000×6 / 0 / 5000; these are
   stadium/infrastructure values used by the finance code — see finance spec, UNCERTAIN meaning).
3. (If manager 0's name is empty — UNCERTAIN purpose — M(m)+0x2260 = m*11.)
4. Money M(m)+0x2432 (long) = 1,500,000; if L == 4: 1,900,000. M(m)+0x242e (long) = 99999 (0x1869f).
5. Club placement (the chosen club is moved into a slot of league L0 by identity swap, see 3.4):
   - if L0 == 0, or the chosen id is not in league L0 (f_17c2_1bb5(id)>>1 != L0), or id > D:2277 (57):
     repeat { x = rand(D:226e[L0], D:226e[L0]+D:225e[L0]−1) } until x is not the team of an earlier
     manager (m' < m) **and** f_08bc_1df3(x,7) == 0 (x is not in any European-cup draw, cups 1..3);
     then f_0310_0b24(chosen, x, 0) (swap identities) and f_08bc_1fc4(chosen, x) (swap the 0x36-byte
     season-stat records back, so stats stay with the slot).
     Effect: for a 1./3.-year game the chosen club always gets a random 1. Liga slot that is not in Europe;
     for the endless game a non-Oberliga club is moved into a random Oberliga slot.
   - 1-year game only (and fresh game): M(m)+0x2377 = 1 and the club is moved again into a random
     slot x of league L0 with f_08bc_1df3(x,4) != 0, i.e. **a slot that is drawn in the UEFA-Pokal** —
     the human starts the 1-year game as a UEFA-Cup participant.
6. Initial squad: for each of the manager's squad slots, 3 skill values = rand(B, B+5) with
   B = L/2 + 68 (L0==0) or L/2 + 27 (L0==2), +12 in the 1-year game; third value also rand(40,60)
   (slot+0x12). (Details in the squad spec.)
7. Other start values (consumers in finance/other specs): M+0x2383..0x2387 = D:286 = 5,4,6,5,6;
   M+0x2388..0x238b = D:28c = 1,3,3,3; M+0x237a = L0; M+0x234c = (1-year?9:8)*2 − 3*L0
   (16/18 in Liga 1, 10 in Oberliga; UNCERTAIN: probably ticket price);
   M+0x23a8 = (5−L0)*4000, M+0x23a0 = (2−L0)*3000, (+10000 / +5000 in the 1-year game),
   M+0x23b0 = (2−L0)*3500, M+0x23c8 = 4 (+1 1-year), M+0x23d0 = 3 (+1 1-year); if not endless:
   M+0x23c0 = 3 − 2*(3-year), M+0x23b8 = 3 − (3-year); M+0x2373 = 0x10;
   **M+0x241e = 50 − 20*L0** (board/target value, see §5); f_08bc_0a63(m); f_14a4_2cb4().
8. Managers added later into a running game get: M+0x234d..+0x2372 = 10, M+0x2374..+0x2378 = 0x1e,
   history M+0x2280+4k = 0xff (k<50) and 4 random youth/free players per position.

### 1.3 In-game options f_262f_01b8 (display toggles, speed, cheats)

15 checkboxes D:5fe..D:60c (value 1 = on):

| Flag | Meaning (as used) |
|---|---|
| D:5fe + 3l | "Halbzeitstände" for league l (shown at minute 45, f_0310_2303) |
| D:5ff + 3l | "Ergebnisse" for league l (results screen at minute 90) |
| D:600 + 3l | "Tabelle" for league l (table after the matchday, f_0310_15db → f_2a41_214c) |
| D:607 | Torszenen |
| D:608 | DfB-Pokal screens |
| D:609 | Europapokale screens |
| D:60a | Nachholspiele results screen |
| D:60b | Zeitung |
| D:60c | Blenden (default 0) |

**All toggles only control what is displayed; competitions are always played** (cups/Nachholspiele
run regardless; checked in f_0310_15db / f_0310_2303 / f_17c2_*). Spielgeschwindigkeit D:63a = 0..70
(slider x−0xca), display speed only. The header shows "V2.0 - ENDE: <F:513c or ' NIE.'> (LEVEL 5−L)".

Hidden keys in this screen (faithful but optional): '0'..'5' → D:4a28 = 5 − digit; 's' → own squad
slots with person set: slot+0x0e = 0x44, slot+0x13 = 0x32; 'a' = same plus slot+0x10..0x12 = 99 and
money = 99,999,999; 'g' → money += 200,000; 'd' → clears D:5ad, D:5ae and a 50-entry table at F:0xa6d8;
'p' → D:5f5 = 1.

### 1.4 Level L (D:4a28) — where it is used (formulas belong to the respective specs)

f_08bc_2184 (start money, start skills), f_0310_2ee5 (match events: rand(0,(L+5)*50+x*70),
rand(0, L*5+2x+37), rand(0, 55x+60L+200)), f_0cb5_13bd (half-weekly processing: L==5, L*30+…+80,
rand(70−13L, (18−L)*5), L<3, L<2), f_0f9d_0002 (AI line-up strength ±(L−5)*v*20/100, +L), f_112a_072d
(rand(2L+87, 2L+117)), f_14a4_2cb4 (bi-weekly finances: L*800 + … + 0x4074), f_112a_000a,
f_322b_21c4 (high score stores '5'−L), savegame.

### 1.5 Initial id shuffle f_0310_09c5 (also at every season start)

```
for l in 0..2:
    lo = D:226e[l]; hi = lo + D:225e[l] - 1
    repeat 55 times:
        a = rand(lo, hi); b = rand(lo, hi)
        f_0310_0b24(a, b, 1)              # swap the identities of slots a and b (see 3.4)
        i = first index with table[l][i] == a; j = first index with table[l][j] == b
        if both found: swap(table[l][i], table[l][j])
```
Because both the club data and the table entries are swapped, the visible table order is unchanged,
but every club gets a random id inside its league range → the fixture letter (and so the schedule) of
each club is randomised every season.

---------------------------------------------------------------------------------------------------

## 2. Calendar and main loop

### 2.1 Date computation f_0310_1190(p)

```
if p == 0: d = day                                   # D:7dc
else:      if p == 0x78: p = 0
           d = (p >> 1)*7 + (p odd ? 4 : D:224c)     # date of event index p (used for "(N)" dates)
year = D:7e0; doy = d + 210
while doy > 365: doy -= 365; year += 1               # no leap years
month = 0
while doy > monthLen[month]: doy -= monthLen[month]; month = (month+1) % 12
F:2eba = doy (1-based day of month); F:2e8e = month (0=Jan); F:a7a0 = year
```
day 0 = **29 July**; weekday shown in the calendar (f_0310_0fb6) = name[(day+5) % 7] (Mon=0) → day 0
is a Saturday. (f_1a58_2a06 uses name[(day+4)%7] — inconsistent, display only.)
Event e happens on day `(e>>1)*7 + (e odd ? 4 : 0)`: even e = Saturday, odd e = Wednesday.

### 2.2 Event table D:2280 (93 entries, e = 0..92)

Bits: bit0/1/2 = league 0/1/2 plays its next matchday; bit3 = DFB-Pokal round; bits 4..6:
value 0x70 = European cup round (all three cups, f_0310_0290(0xe)); **value 0x10 alone =
Relegation match** (f_0310_02bf); bit7 (0x80) = Nachholspiele (only set at run time). Entry value 9
is treated as 0 (never occurs). Weekday/date for a season starting in 1992:

| e | day | date | val | content |
|---|---|---|---|---|
| 0 | 0 | Sa 29.Jul 1992 | 0x07 | L1:MD1 L2:MD1 L3:MD1 |
| 1 | 4 | Mi  2.Aug 1992 | 0x70 | EC#1 |
| 2 | 7 | Sa  5.Aug 1992 | 0x07 | L1:MD2 L2:MD2 L3:MD2 |
| 3 | 11 | Mi  9.Aug 1992 | 0x00 |  |
| 4 | 14 | Sa 12.Aug 1992 | 0x07 | L1:MD3 L2:MD3 L3:MD3 |
| 5 | 18 | Mi 16.Aug 1992 | 0x70 | EC#2 |
| 6 | 21 | Sa 19.Aug 1992 | 0x07 | L1:MD4 L2:MD4 L3:MD4 |
| 7 | 25 | Mi 23.Aug 1992 | 0x06 | L2:MD5 L3:MD5 |
| 8 | 28 | Sa 26.Aug 1992 | 0x08 | DFB#1 |
| 9 | 32 | Mi 30.Aug 1992 | 0x00 |  |
| 10 | 35 | Sa  2.Sep 1992 | 0x07 | L1:MD5 L2:MD6 L3:MD6 |
| 11 | 39 | Mi  6.Sep 1992 | 0x00 |  |
| 12 | 42 | Sa  9.Sep 1992 | 0x07 | L1:MD6 L2:MD7 L3:MD7 |
| 13 | 46 | Mi 13.Sep 1992 | 0x06 | L2:MD8 L3:MD8 |
| 14 | 49 | Sa 16.Sep 1992 | 0x07 | L1:MD7 L2:MD9 L3:MD9 |
| 15 | 53 | Mi 20.Sep 1992 | 0x70 | EC#3 |
| 16 | 56 | Sa 23.Sep 1992 | 0x07 | L1:MD8 L2:MD10 L3:MD10 |
| 17 | 60 | Mi 27.Sep 1992 | 0x07 | L1:MD9 L2:MD11 L3:MD11 |
| 18 | 63 | Sa 30.Sep 1992 | 0x07 | L1:MD10 L2:MD12 L3:MD12 |
| 19 | 67 | Mi  4.Oct 1992 | 0x70 | EC#4 |
| 20 | 70 | Sa  7.Oct 1992 | 0x07 | L1:MD11 L2:MD13 L3:MD13 |
| 21 | 74 | Mi 11.Oct 1992 | 0x00 |  |
| 22 | 77 | Sa 14.Oct 1992 | 0x08 | DFB#2 |
| 23 | 81 | Mi 18.Oct 1992 | 0x01 | L1:MD12 |
| 24 | 84 | Sa 21.Oct 1992 | 0x07 | L1:MD13 L2:MD14 L3:MD14 |
| 25 | 88 | Mi 25.Oct 1992 | 0x06 | L2:MD15 L3:MD15 |
| 26 | 91 | Sa 28.Oct 1992 | 0x07 | L1:MD14 L2:MD16 L3:MD16 |
| 27 | 95 | Mi  1.Nov 1992 | 0x00 |  |
| 28 | 98 | Sa  4.Nov 1992 | 0x07 | L1:MD15 L2:MD17 L3:MD17 |
| 29 | 102 | Mi  8.Nov 1992 | 0x06 | L2:MD18 L3:MD18 |
| 30 | 105 | Sa 11.Nov 1992 | 0x08 | DFB#3 |
| 31 | 109 | Mi 15.Nov 1992 | 0x70 | EC#5 |
| 32 | 112 | Sa 18.Nov 1992 | 0x07 | L1:MD16 L2:MD19 L3:MD19 |
| 33 | 116 | Mi 22.Nov 1992 | 0x00 |  |
| 34 | 119 | Sa 25.Nov 1992 | 0x07 | L1:MD17 L2:MD20 L3:MD20 |
| 35 | 123 | Mi 29.Nov 1992 | 0x70 | EC#6 |
| 36 | 126 | Sa  2.Dec 1992 | 0x07 | L1:MD18 L2:MD21 L3:MD21 |
| 37 | 130 | Mi  6.Dec 1992 | 0x00 |  |
| 38 | 133 | Sa  9.Dec 1992 | 0x00 |  |
| 39 | 137 | Mi 13.Dec 1992 | 0x00 |  |
| 40 | 140 | Sa 16.Dec 1992 | 0x00 |  |
| 41 | 144 | Mi 20.Dec 1992 | 0x00 |  |
| 42 | 147 | Sa 23.Dec 1992 | 0x00 |  |
| 43 | 151 | Mi 27.Dec 1992 | 0x00 |  |
| 44 | 154 | Sa 30.Dec 1992 | 0x00 |  |
| 45 | 158 | Mi  3.Jan 1993 | 0x00 |  |
| 46 | 161 | Sa  6.Jan 1993 | 0x00 |  |
| 47 | 165 | Mi 10.Jan 1993 | 0x00 |  |
| 48 | 168 | Sa 13.Jan 1993 | 0x00 |  |
| 49 | 172 | Mi 17.Jan 1993 | 0x00 |  |
| 50 | 175 | Sa 20.Jan 1993 | 0x00 |  |
| 51 | 179 | Mi 24.Jan 1993 | 0x00 |  |
| 52 | 182 | Sa 27.Jan 1993 | 0x00 |  |
| 53 | 186 | Mi 31.Jan 1993 | 0x00 |  |
| 54 | 189 | Sa  3.Feb 1993 | 0x00 |  |
| 55 | 193 | Mi  7.Feb 1993 | 0x00 |  |
| 56 | 196 | Sa 10.Feb 1993 | 0x00 |  |
| 57 | 200 | Mi 14.Feb 1993 | 0x00 |  |
| 58 | 203 | Sa 17.Feb 1993 | 0x00 |  |
| 59 | 207 | Mi 21.Feb 1993 | 0x07 | L1:MD19 L2:MD22 L3:MD22 |
| 60 | 210 | Sa 24.Feb 1993 | 0x07 | L1:MD20 L2:MD23 L3:MD23 |
| 61 | 214 | Mi 28.Feb 1993 | 0x00 |  |
| 62 | 217 | Sa  3.Mar 1993 | 0x07 | L1:MD21 L2:MD24 L3:MD24 |
| 63 | 221 | Mi  7.Mar 1993 | 0x70 | EC#7 |
| 64 | 224 | Sa 10.Mar 1993 | 0x08 | DFB#4 |
| 65 | 228 | Mi 14.Mar 1993 | 0x07 | L1:MD22 L2:MD25 L3:MD25 |
| 66 | 231 | Sa 17.Mar 1993 | 0x07 | L1:MD23 L2:MD26 L3:MD26 |
| 67 | 235 | Mi 21.Mar 1993 | 0x70 | EC#8 |
| 68 | 238 | Sa 24.Mar 1993 | 0x07 | L1:MD24 L2:MD27 L3:MD27 |
| 69 | 242 | Mi 28.Mar 1993 | 0x06 | L2:MD28 L3:MD28 |
| 70 | 245 | Sa 31.Mar 1993 | 0x07 | L1:MD25 L2:MD29 L3:MD29 |
| 71 | 249 | Mi  4.Apr 1993 | 0x00 |  |
| 72 | 252 | Sa  7.Apr 1993 | 0x07 | L1:MD26 L2:MD30 L3:MD30 |
| 73 | 256 | Mi 11.Apr 1993 | 0x00 |  |
| 74 | 259 | Sa 14.Apr 1993 | 0x07 | L1:MD27 L2:MD31 L3:MD31 |
| 75 | 263 | Mi 18.Apr 1993 | 0x00 |  |
| 76 | 266 | Sa 21.Apr 1993 | 0x07 | L1:MD28 L2:MD32 L3:MD32 |
| 77 | 270 | Mi 25.Apr 1993 | 0x70 | EC#9 |
| 78 | 273 | Sa 28.Apr 1993 | 0x07 | L1:MD29 L2:MD33 L3:MD33 |
| 79 | 277 | Mi  2.May 1993 | 0x00 |  |
| 80 | 280 | Sa  5.May 1993 | 0x07 | L1:MD30 L2:MD34 L3:MD34 |
| 81 | 284 | Mi  9.May 1993 | 0x70 | EC#10 |
| 82 | 287 | Sa 12.May 1993 | 0x07 | L1:MD31 L2:MD35 L3:MD35 |
| 83 | 291 | Mi 16.May 1993 | 0x00 |  |
| 84 | 294 | Sa 19.May 1993 | 0x07 | L1:MD32 L2:MD36 L3:MD36 |
| 85 | 298 | Mi 23.May 1993 | 0x00 |  |
| 86 | 301 | Sa 26.May 1993 | 0x08 | DFB#5 |
| 87 | 305 | Mi 30.May 1993 | 0x00 |  |
| 88 | 308 | Sa  2.Jun 1993 | 0x07 | L1:MD33 L2:MD37 L3:MD37 |
| 89 | 312 | Mi  6.Jun 1993 | 0x00 |  |
| 90 | 315 | Sa  9.Jun 1993 | 0x07 | L1:MD34 L2:MD38 L3:MD38 |
| 91 | 319 | Mi 13.Jun 1993 | 0x10 | RELEGATION |
| 92 | 322 | Sa 16.Jun 1993 | 0x10 | RELEGATION |

Totals: 34 / 38 / 38 league matchdays, 5 DFB rounds, 10 European-cup dates (5 two-legged rounds,
see §6), 2 relegation legs. Winter break: e = 37..58 (6 Dec – 17 Feb) are empty; last matchday before
the break = e 36 (Sat 2 Dec), first after = e 59 (Wed 21 Feb). Season ends after e = 92 (16 June).

### 2.3 Main loop f_1a58_3176 (exact order)

```
loop forever:                                      # one iteration per event index e = F:16e
  f_0310_1190(0); F:4bce = month
  ev = D:2280[e]; if ev == 9: ev = 0
  for m in managers: f_112a_0a6d()                 # daily processing of the event day
  f_0f9d_0697(1)
  if ev != 0:                                      # managers only get a turn on event days with an event
      for m: D:79e[2m] = D:79f[2m]; f_1ecd_3360(); f_0f9d_0002(m,0)
      for m (starting with the saved manager when resuming a savegame):
          F:5358 = m; f_076b_0e27()
          if rand(0, D:7ab+3) == 0: f_2277_1e38()  # random event
          f_08bc_0b5f(e)                           # the manager's turn (menus, autosave)
      f_076b_0e27()
      for m: f_0f9d_0002(m,1); D:79f[2m]=D:79e[2m]; D:79e[2m]=1; F:1d14[m]=0; F:90c6[m]=0
      F:304a = 0x1e                                # "no current manager" during matches
      if ev & 7:
          if ev & 1: f_0310_0463(e, 0)             # weather postponements (2.5)
          f_0310_0463(e, 1); f_0310_0463(e, 2)     # (called even if that league has no game!)
          f_0310_0995(ev)                          # play league matchdays: f_0310_15db(ev, 0)
      if ev & 8: f_0310_0261(1)                    # DFB-Pokal round
      if (ev & 0x70) == 0x10: f_0310_02bf()        # relegation leg
      elif ev & 0x70:  f_0310_0290(0xe)            # European cups 1..3
      if ev & 0x80 and f_0310_000a(e): f_0310_07a2(e)   # Nachholspiele
  # LAB_3397 (also for ev == 0)
  for l in 0..2: if D:2280[e] bit l: D:225a[l] += 1          # matchday counters
  if 130 + 7*D:224d < day <= 207 - 14*D:224d: show WINTERPAUSE picture (display only)
  for m: for s in 0..23: if squad[m][s]+0x0d != 0: squad[m][s]+0x0a = 0   # suspended players leave the line-up
  day += 1; check_special_dates()                  # uses the date still set for the previous day
  do:
      f_0310_1190(0)
      for m: F:5358 = m; f_112a_0a6d()             # daily processing
      day += 1; check_special_dates()
      if month == 11 and D:5da: exit(99)           # demo version stops in December
  while (day - D:224c) % 7 != 0 and (day - 4) % 7 != 0
  for m: f_0cb5_13bd(); if (day - D:224c) % 14 == 0: f_14a4_2cb4()   # per event / bi-weekly
  if day <= D:224c + 322: e += 1; continue
  -> season end (2.6)
```
`check_special_dates()`: for k=0..2: if F:2eba == D:53d0[k] and F:2e8e == D:53d4[k]−1:
f_1a58_2a06(k): with probability 1/4 (rand(0,3)==0): k=0 Christmas card "Frohe Weihnachten !" —
then with 50% the players get Christmas packages: amount = rand(15,85)*10000; every manager pays
amount/(league+1) (money −=). k=1/2: birthday card of a programmer (needs a key press; in this
build only shown when D:2252 == D:224e). Display + money only.

Daily processing f_112a_0a6d and the per-event f_0cb5_13bd / bi-weekly f_14a4_2cb4 belong to
other specs. f_112a_314d (called from the turn) shows "Achtung ! Dies ist der letzte Spieltag vor
der Winterpause." when the date is 2 December (3 Dec if D:224e==4), and the relegation notice on
13 June (see §5/§6).

### 2.4 Turn at season end

When `day > 322` after e = 92: for m: D:79e/79f copy, f_1ecd_3360, f_0f9d_0002(m,0); then each manager
gets a final turn f_08bc_0b5f(0xff) followed by the history update for that manager (§5); then the
season-end processing 2.6.

### 2.5 Weather postponements f_0310_0463(e, l) and Nachholspiele

Only for 25 ≤ e ≤ 69 (late autumn → spring):
```
d = |e - 44|
n = 9 - d/3;  if n != 0 and rand(0,1) != 0: n = 8 - d/3       # integer division
n = rand(0, n)
picked = []                                                     # pair indexes already postponed now
loop:
    if n == 0: return
    n = min(n, 8); before = n
    for r in 0..19:
        if record[r].slot != 0: continue
        p = rand(0, D:225e[l]/2 - 1)
        if p in picked or result[l][D:225a[l]][p].home == 0x1e: continue   # try next record r
        if f_0310_05f1(e, l, p, r): n -= 1; picked.append(p); break
    if n == before: return          # nothing could be scheduled in a full pass
```
f_0310_05f1(e, l, p, r) finds the new date:
```
slot = e
repeat:
    slot += 1
    if slot > 89: return 0                         # cannot be rescheduled -> not postponed
    if 36 <= slot <= 58: slot = 59                 # never in the winter break
    if D:2280[slot] not in (0x00, 0x80): continue  # only on otherwise free dates
    conflict = any record q with q.slot == slot whose pair (league q.league, matchday q.md, pair q.pair)
               shares a team with pair p of league l, matchday D:225a[l]
until not conflict
record[r] = {slot, md = D:225a[l], league = l, pair = p}
result[l][D:225a[l]][p].home = 0x1e                # skipped when the matchday is played
D:2280[slot] = 0x80
return 1
```
The postponed game is therefore not played on its matchday (f_0310_15db skips pairs whose home result
byte is 0x1e) and not counted in the table until played.

Nachholspiele, event with bit 0x80 (f_0310_000a / f_0310_07a2):
1. Collect all records with slot == e and md != 0 (in record order); for each, recompute the
   fixture of (league, md) with f_2a41_10e3 and copy the pair into F:4b5e[0..k−1]; if k<10 put 0x46 at
   F:4b5e[2k].
2. f_0310_15db(0,0): plays these pairs as "league 0" (the results go to the scratch row
   result[0][D:225a[0]], i.e. the not-yet-played next 1. Liga matchday), and the table update
   f_2a41_2d33(0,…) adds them to the per-club statistics (clubs of any league) and re-sorts league 0
   (other leagues are re-sorted at their next matchday).
3. Copy results from the scratch row positions (2j, 2j+1) to result[league][md][pair] for each record,
   free the record (slot = 0), zero the first 19 bytes of the scratch row (bug: 20 would be needed) and
   set D:2280[e] = 0.
(f_0310_000a returns garbage-nonzero when there are no records; 07a2 then plays nothing — harmless.)

Forfeits: if a human club's status M+0x237f == 'd' (set by f_0f9d_0002 when fewer than 8 players are
available, see §5.5) and the date is not on/after 10 June, f_1a58_20b2 returns 1/2: the game is not simulated, the manager is
fined 200,000 DM ("Ihr Spiel wird mit 0:2 gewertet. Sie zahlen 200.000 DM Strafe") and the result is
set to 0:2 (home forfeits) or 2:0 (away forfeits) via F:549c/F:4bd0/F:57a6 lists, f_0310_152e.

### 2.6 Season end (rest of f_1a58_3176)

0. (Already during the final turns, per manager: history record + lifetime balance, §5.2 step 1.)
1. For each manager: remember the club's eternal points `before[m]` = long +0x32 of the club record
   (F:0xefe+t*0x36), F:4bec[m]=0, v[m] = f_14a4_1ad5(1).
2. f_1a58_4597() — season summary, adds this season's eternal-table points to +0x32 of every club (§5.2).
3. Rating update: n = min(D:7e2+1, 5); for each manager (one shared variable r, see quirk in §5.2):
   `r = M+0x241e + (int16)((uint32)((eternal_new − before[m])*100) / 108 / n) − (uint16)M+0x241e / n`
   (unsigned 32-bit divisions; the difference is never negative), clamp r to 0..100 (f_3091_1009).
4. Zero all season statistics of clubs 0..63 (fields +0x00,+0x16,+0x1a,+0x1e,+0x26,+0x2a, both
   home and away bytes) and set both form strings to "@@@@@@@@".
5. **Promotion / relegation** (4.4).
6. f_0f9d_0697(10) (AI strength drift, §5.5); then for each manager m (same shared r — all managers
   get the value computed for the LAST manager, original bug): if r < 30 and league(m) < 2: r = 30;
   if r < 61 and league(m) == 0: r = 61; M(m)+0x241e = r. (League AFTER promotion/relegation.)
7. If F:513c == F:a7a0 (end year reached): game end — fixture strings are deliberately corrupted
   (D:704[0], D:70c[4], D:718[8], D:720[8] = 'a'), high score / final screens (§5), program exit.
8. Else for each manager: f_322b_2366(2); f_14a4_2cb4().
9. f_0310_09c5() (id shuffle 1.5); f_17c2_0ef2() (European qualification for next season);
   f_0cb5_0012(v[]); D:7e2 += 1.
10. Summer: day += 1; repeat { F:4bce = month; f_0310_1190(0); for m: f_112a_0a6d(); for each squad
    slot s < count: f_0cb5_2b88(slot); day += 1 } until the date is **28 July** (F:2eba==28 and month==6).
11. f_0310_1190(0); e = F:16e = 0; day = D:224c (0); **D:7e0 += 1**; results block F:6ddc (0x8e8 bytes)
    zeroed; f_08bc_0a63(m) for each manager; f_17c2_1b6d(); f_08bc_02ef(3); f_17c2_09e0(c,0) for c=0..3
    (draws), f_17c2_09e0(c,1) c=1..3; f_08bc_034d(3); continue with the main loop.

---------------------------------------------------------------------------------------------------

## 3. Fixtures

### 3.1 Fixture strings (D:6fc / D:744)

18 teams (string k = round k, k = 1..17), 9 pairs per string, pair i = (s[2i], s[2i+1]) = (home, away);
letter − 'a' = index within the league:

```
 1 abcdefghijklmnopqr      10 rbnflhjqigkemcoapd
 2 ronklijghefcdabqpm      11 amckeijlhnfpdrboqg
 3 afchejglinkpmrbdqo      12 rfnjlqgeickambodph
 4 rknglejchafbdqompi      13 aicglnjphrfodmbkqe
 5 ajclengpirkodfbhqm      14 rjnqecgaibkdmfohpl
 6 rgnclajbhdfqmkoipe      15 aecqlrjohmfkdibgpn
 7 fhancpergoimdjblqk      16 rncaebgdifkhmjolqp
 8 rcnbldjfhqkimgoepa      17 aqnolmjkhifgdebcpr
 9 arcoemgkhjfldnbpqi
```
20 teams (k = 1..19, 10 pairs):
```
 1 abcdefghijklmnopqrst     11 mqhdcoientjbksgaprlf
 2 dahcnklibetqropmjgfs     12 sitpegdjrcomahfnblqk
 3 caglinkpqfsbedhjmrot     13 hecmnbgsiqkopfldrtja
 4 dstmlhbqrkpingjcaefo     14 mktcejbpoiqgaldnshfr
 5 gpceobsairktmfqdjlhn     15 nagoimrbjsckpdhqtfle
 6 phaqdotirgnjesfklcbm     16 slendrkiapfcqjbtmgoh
 7 oahrcsgtifkbqelnjpmd     17 hmpetdnsgkralqicjofb
 8 bithrjampleofgdksqnc     18 mjdfqnatcbkholspiger
 9 mecqlrgbkaosjtnpidhf     19 lmbdrsnojkfahigctepq
10 tlaibhqofjdgrnpcsmek
```
Verified: each set is a complete single round robin (153 / 190 distinct pairs, every team once per
round). Home counts per Hinrunde are 8/9 (18) and 9/10 (20).

### 3.2 f_2a41_10e3(l, md) — fixture of league l, matchday md (1-based)

```
off = 0 (l=0), D:225e[0] = 18 (l=1), D:225e[0]+D:225e[1] = 38 (l=2)
half = D:2262[l] / 2                         # 17 or 19
swap = md > half;  if swap: md -= half       # Rückrunde = same round, home/away swapped
md = min(md, half)
s = (D:225e[l] == 18) ? str18[md] : str20[md]
for i in 0 .. D:225e[l]/2 - 1:
    h = s[2i] - 'a'; a = s[2i+1] - 'a'
    if swap: h, a = a, h
    F:4b5e[(l*10+i)*2]   = h + off           # team id
    F:4b5e[(l*10+i)*2+1] = a + off
```
So the fixture is a pure function of (league, matchday); the letter → club mapping is the club's id
within the league range, which is randomised by f_0310_09c5. The third league (Amateur-Oberliga,
20 teams) uses the 20-team strings exactly like the 2. Liga, with id offset 38. **Note: the code
indexes the result/fixture arrays by team id directly, which requires that league l always consists of
exactly the ids D:226e[l]..+D:225e[l]−1; promotion/relegation therefore swaps club identities (3.4).**

### 3.3 Playing a matchday f_0310_15db(leagueMask, 0, …)

For l = 0..2 (only l=0 when leagueMask == 0, used by Nachholspiele): if leagueMask != 0 compute
fixtures f_2a41_10e3(l, D:225a[l]); f_2a41_2d33(l, 1, 1) (snapshot, 4.3); then for pairs i in order:
stop at home id 0x46; skip if result[l][D:225a[l]][i].home == 0x1e; else play f_1a58_20b2(home, away,
0, l, i, leagueMask) (match engine, other spec). NOTE: all three leagues' games of an event are played
"simultaneously" minute by minute (f_0310_2303 called for minutes 1..45 and 46..90). At minute 90 for
each league in the mask: f_2a41_2d33(l, 0, 1) (table update) and f_14a4_1662(l).
After the matchday: if D:600+3l: show table f_2a41_214c(l, 0).

### 3.4 Identity swap f_0310_0b24(a, b, flag) (a, b < 64)

Swaps everything that belongs to a club between ids a and b: person team field (+0x24 of persons 1..150),
manager team ids M+0x2260, (if flag) the cup-holder references D:7ac..D:7b0 (stored as id+1), TEAM
record (0x22 bytes), season record F:0xecc (0x36 bytes), series record F:0x9d36 (0x15 bytes),
record tables F:0xa2ca/F:0xa4ca (8 bytes each) and opponent references in F:0xa4ca (f_0310_0f44), and
the per-manager head-to-head table F:0x9336 (10 bytes per team per manager). After a swap the club
formerly at id a is at id b and vice versa; table arrays F:535a are NOT touched (they hold ids).

---------------------------------------------------------------------------------------------------

## 4. League tables

### 4.1 Per-club season record F:0x0ecc + t*0x36 (t = 0..63)

Each statistic is stored as [home part, away part] (+ a 2-byte snapshot copy where noted):

| Off | Abs | Meaning |
|---|---|---|
| +0x00/+0x01 | 0xecc/0xecd | points earned at home / away (snapshot at +0x02/+0x03) |
| +0x04, +0x0d | 0xed0, 0xed9 | form strings (8 chars + NUL each, current and snapshot), init "@@@@@@@@"; new char appended at the end, older ones shift left (f_2a41_1fb0) |
| +0x16/+0x17 | 0xee2/0xee3 | goals scored home / away (snapshot +0x18/+0x19) |
| +0x1a/+0x1b | 0xee6/0xee7 | goals conceded home / away (snapshot +0x1c/+0x1d) |
| +0x1e/+0x1f | 0xeea/0xeeb | games played home / away (no snapshot) |
| +0x26/+0x27 | 0xef2/0xef3 | wins home / away (snapshot +0x28/+0x29) |
| +0x2a/+0x2b | 0xef6/0xef7 | losses home / away (snapshot +0x2c/+0x2d) |
| +0x2e | 0xefa | current table position (0-based) |
| +0x32 | 0xefe | long, eternal-table points of the club (accumulated over seasons by f_1a58_4597, NOT reset) |

Draws are not stored: draws = games − wins − losses. Minus points (display) = 2*games − points.

### 4.2 Table update f_2a41_2d33(l, mode, which)

Precondition: md = D:225a[l] − 1 (0-based row) ≤ D:2262[l] − 1, l < 3.

mode 1 (snapshot, before the matchday): for every pair (h, a) of the current fixture buffer (stop at
0x46, skip postponed 0x1e): f_2a41_1ef4(t, 0, 2) for both teams = copy current fields (points, goals,
conceded, wins, losses, form string) to the snapshot slots.

mode 0 (result, at minute 90; also called from the live table): for each played pair with result
(g_h, g_a) (skip 0x1e):
```
restore h and a from their snapshot (f_2a41_1ef4(t,2,0))   # makes the update idempotent
h.goalsHome += g_h; h.concHome += g_a; a.goalsAway += g_a; a.concAway += g_h
h.gamesHome += 1; a.gamesAway += 1          # (not snapshotted: incremented once per call)
if g_h > g_a:  h.ptsHome += D:2271 (2); h.winsHome += 1; a.lossesAway += 1; form h 's', a 'N'
elif g_h == g_a: h.ptsHome += 1; a.ptsAway += 1; form h 'u', a 'U'
else: a.ptsAway += D:2271; a.winsAway += 1; h.lossesHome += 1; form h 'n', a 'S'
```
plus series/record bookkeeping (F:9d36, F:a2ca via f_2a41_37cf, f_2a41_2073, and per-manager
maxima F:a276) and TEAM+0x17 adjustment f_2a41_1fec: `TEAM[t]+0x17 += ±rand(0, 2*f_3091_10d5(diff))`
(sign of the goal difference from t's view; result clamped to 47..53 by f_3091_1009) — UNCERTAIN
details of f_3091_10d5 (probably |x|).

Sorting (always, both modes), `which`: 0 = home only, 1 = total, 2 = away only (always 1 from the
game code). Value(x) = home part (if which<2) + away part (if which≠0). Exchange sort:
```
for i in 0 .. n-2:
  for j in i+1 .. n-1:
    A = table[l][i]; B = table[l][j]
    if pts(B) > pts(A): swap
    elif pts(B) == pts(A):
        if D:224d == 0 and games(B) < games(A): swap            # fewer games first (=fewer minus points)
        elif games equal (or D:224d != 0):
            if gd(A) < gd(B): swap                               # gd = goals − conceded
            elif gd equal and goals(A) < goals(B): swap
```
(swap = exchange table[l][i] and table[l][j]; ties after all criteria keep the current order.)
Then `team[table[l][i]].pos(+0x2e) = i` and for every manager `M+0x234d+md = pos of his club`.

### 4.3 Table screen

f_2a41_214c(l, …): columns Pl., club, form string, Sp., S, U, N,
"Punkte" (x:y with y = 2*games − x while D:2271 < 3), "Tore" goals:conceded and difference; human
clubs highlighted. Zones (colours): league 0: place 1 (champion), places 2..1+D:226d (UEFA), bottom
D:2266[l] places; leagues 1/2: top D:2266[l−1] places (promotion) and bottom D:2266[l].

---------------------------------------------------------------------------------------------------

### 4.4 Promotion / relegation (f_1a58_3176, 1a58:3d99..40e2)

Run after the statistics reset (2.6 step 4). `forced[3] = {0,0,0}`.

Step A — licence withdrawal (bankruptcy): for each manager m: M+0x2382 = 0; if money M+0x2432 <
−2,000,000 (long compare: < 0xFFE17B80) and league L = M+0x237a < 2:
M+0x2382 = 4; remove his club from table[L] (shift following entries up) and put it on the last
place; forced[L] += 1; M+0x237a += 1 (relegated immediately).

Step B — for l = 0, 1, 2:
```
if l == 0 and F:3060 == 0: forced[0] += 1      # F:3060: 0 = 1.Liga club won the relegation tie (§6)
forced[l] = forced[l] > D:2266[l] ? forced[l] - D:2266[l] : 0
D:225a[l] = 1
cnt = forced[l] + D:2266[l]
for k in 0 .. cnt-1:
    rel = table[l][D:225e[l] - 1 - k]            # k-th from the bottom
    if l < 2: prom = table[l+1][k]               # k-th from the top of the lower league
    else:                                        # Amateur-Oberliga: replaced by reserve clubs
        base = D:225e[0]+D:225e[1]+D:225e[2]     # = 58
        prom = (k == 0 and base < 60) ? rand(base+3, base+4) : base + 3 - k   # 61|62, 60, 59, 58
    if l == 0 and cnt - k == 1 and F:3060 == 0 and D:226a[0] != 0: continue   # 16th stays up
    mr = manager owning rel
    if mr and M(mr)+0x2382 == 0:
        if l < 2: M(mr)+0x2382 = 2; M(mr)+0x237a += 1                  # relegated
        else:     M(mr)+0x2382 = 8; money(mr) = 500,000; f_0310_0b24(rel, prom, 1)
    mp = manager owning prom (looked up AFTER the possible swap above)
    if mp and M(mp)+0x2382 == 0: M(mp)+0x237a -= 1; M(mp)+0x2382 = 1  # promoted
    f_0310_0b24(rel, prom, 1)                    # promoted club takes the slot id of the relegated one
```
Results with default constants:
* 1. Liga: places 18 and 17 down, 16 down only if it lost the relegation tie vs. 2. Liga 3rd
  (F:3060 != 0). 2. Liga 1st, 2nd (and 3rd if the tie was won) go up.
* 2. Liga: bottom 4 down, Amateur-Oberliga top 4 up.
* Amateur-Oberliga: bottom 4 replaced by the reserve clubs (ids 61 or 62, 60, 59, 58 — id 63 and the
  other of 61/62 never come up in that season).
* A human club relegated from the Amateur-Oberliga is NOT removed: the two swaps cancel out (faithful
  quirk); the manager gets status 8 and his money set to 500,000 DM (the reserve club stays out).
* Bankrupt clubs (step A) count towards the relegation places; more bankrupt clubs than places →
  more relegations.
Table arrays still contain the same ids after this; the promoted club now sits at the bottom of the
higher table (same order as before), the relegated one at the top of the lower table.
M+0x2382 (season status: 1 promoted, 2 relegated, 4 licence withdrawn, 8 dropped out of the Oberliga)
is evaluated in the season summary (§5). F:3060 is the result of the relegation tie (§6.7).

---------------------------------------------------------------------------------------------------
## 5. Season end, goals, firing, job offers, high score, history

Notation: `M` = manager record base `F:0x2242 + m*0x30a` (fields given as absolute
offsets for m=0, e.g. `M.0x2260` = team id). `T(t)` = TEAM record `F:0x3066 + t*0x22`,
`X(t)` = extended German-club record `F:0x0ecc + t*0x36`. Team ids 0..63 are *league
slots*: ids 0..17 = 1. Bundesliga, 18..37 = 2. Liga, 38..57 = Amateur-Oberliga,
58..63 = clubs outside the three leagues (see f_17c2_1bb5 and D:0x226f=18, D:0x2270=38,
D:0x2277=57). Promotion/relegation swaps the *contents* of two slots
(f_0310_0b24 swaps T, X, player team ids and manager team ids), so every per-club
statistic in X (including the eternal points) travels with the club.

### 5.0 Summary of what does NOT exist

* **No firing, no manager job offers.** No "entlassen"/job-offer strings exist; the
  "Angebot/Vertrag" strings belong to the transfer market and player contracts. A
  manager keeps their club for the whole game. The closest mechanics are:
  * licence refusal = forced relegation when money < -2,000,000 DM (5.2),
  * a human club that would be relegated *out of* the Amateur-Oberliga stays where it is
    and its money is reset to 500,000 DM (5.2).
* **No success/fail game over.** The game ends only when the chosen duration has
  elapsed (5.6). The "letzte Chance verspielt ..." messages are information only (5.1).

### 5.1 Goal messages: f_112a_314d

Called once per manager turn (from f_08bc_0b5f @08bc:1415, the first time the main
menu is drawn in that turn). It has no effect on the game except setting flag bits and
showing message boxes. Let `l = M.0x237a` (league), `n = D:0x225e[l]` (teams),
`me = M.0x2260`, `pts(t) = X(t)+0x00 + X(t)+0x01` (home+away points, 2-point rule,
see the league-table section), `games(t) = X(t)+0x1e + X(t)+0x1f`,
`maxpts(t) = pts(t) + (D:0x2262[l] - games(t)) * D:0x2271` (D:0x2271 = 2).

1. **Relegation-match notice**: if D:0x226a != 0 AND date is 13 June (F:0x2e8e==5,
   F:0x2eba==13) AND `D:0x225a[l] - D:0x2262[l] == 1` (all matchdays played) AND
   ((l==0 and position after the last matchday == index 15, i.e. 16th) or
   (l==1 and position index 2, i.e. 3rd)) -> message "Ihre Mannschaft bestreitet das
   Relegationsspiel. Viel Glück." and return. (Position read from M.0x234c+matchday.)
2. Otherwise: if month==June and day > 13 -> return (no messages after 13 June).
3. If date == 2 December (day = 2 + (D:0x224e==4 ? 1 : 0); D:0x224e is 1 in this
   version, so day 2) -> message "Achtung! Dies ist der letzte Spieltag vor der
   Winterpause."
4. Thresholds (table index into `F:0x535a + l*0x14`, 0-based), with `v = (D:0x224e==4)`
   (always 0 in the German version, a currency/version constant never written):

   | league | k=0 (Meister) | k=1 (UEFA-Cup / Aufstieg) | k=2 (Klassenerhalt) |
   |---|---|---|---|
   | 0 (BL)  | 0 | 4 | n-4-v = 14 |
   | 1 (2.L) | 0 | 1+2v = 1 | n-5 = 15 |
   | 2 (AOL) | 0 | 3 | n-5 = 15 |

   `ref_k = table[l][idx_k]`.
5. For k = 0..2:
   * **lost**: if `maxpts(me) < pts(ref_k)` and bit k of F:0x4bec[m] is clear: set it,
     show "<club> hat / die letzte Chance verspielt, / MEISTER zu werden." (k=0),
     "den UEFA-CUP zu erreichen." (k=1, l==0) / "AUFZUSTEIGEN." (k=1, l>0),
     "DIE KLASSE ZU ERHALTEN." (k=2).
   * **secured**: `best = max(0, max over table positions idx_k+1 .. n-1 of maxpts)`;
     if `best < pts(me)` and bit k+3 clear: set it, show "<club> hat / die
     MEISTERSCHAFT | den UEFA-CUP-PLATZ (l==0) | den AUFSTIEG (l>0) | den
     KLASSENERHALT / auf jeden Fall gesichert."
   * Message boxes are only shown if `l == 0 || k != 0` (no "Meister" messages below
     the Bundesliga); the flag bits are set anyway.
   Note: comparisons are strict; `me` itself may be `ref_k` (then "lost" is impossible).
6. F:0x4bec[m] is cleared at season end (f_1a58_3176 @1a58:3c13 and f_1a58_4597).

### 5.2 Season-end sequence (inside f_1a58_3176, after the last event day)

When day counter D:0x7dc > D:0x224c + 0x142 (322), after the last manager turn
(see calendar section) the loop does, in this order:

1. **Per manager m: history + eternal balance** (label LAB_2a58_3763, @1a58:3763):
   with `t = M.0x2260`, for h in {0=home, 1=away} (int arrays of 2, index h):
   * `M.0x23e6[h] += X(t)+0x00+h` (points)
   * `M.0x23f2[h] += 2*games_h - points_h` (minus points), games_h = X(t)+0x1e+h
   * `M.0x23ea[h] += X(t)+0x16+h` (goals for), `M.0x23ee[h] += X(t)+0x1a+h` (against)
   * `M.0x23f6[h] += games_h`, `M.0x23fa[h] += X(t)+0x26+h` (wins),
     `M.0x23fe[h] += X(t)+0x2a+h` (losses), `M.0x2402[h] += games_h - losses - wins`
     (draws).
   * reset attendance statistics: `M.0x2426/0x2428` (long total) = 0,
     `M.0x242a/0x242c` (long max) = 0, `M.0x237c` (home-game count) = 0,
     `M.0x242e/0x2430` (long min) = 99999 (0x1869f).
   * **history record** (4 bytes) at `M.0x2280 + k*4`, `k = D:0x7e2` (season index,
     0-based). If k > 49: shift records 1..49 down to 0..48 (memcpy 4 bytes each),
     `M.0x237d++` (number of dropped seasons), k = 49.
     * byte0 = `M.0x234c[D:0x2262[l]] + 1 + D:0x226e[l]` = final 1-based position
       plus league offset (0/18/38) = overall rank 1..58.
       (M.0x234d+i = position index after matchday i+1, written by f_2a41_2d33.)
       If final position index == 0 and l == 0: `M.0x227b++` (championships) and
       f_17c2_274d(m,0) (trophy screen "<club> gewinnt die Deutsche Meisterschaft").
     * byte1 = l (league during that season).
     * byte2 = `M.0x2374 + 1` (DFB-Pokal round reached; 0x1e = not participating) ,
       `+0x80` if D:0x7ac == team+1 (DFB-Pokal winner); then `M.0x227c++`.
     * byte3: for c = 1..3 (1 = Pokal der Landesmeister, 2 = Pokal der Pokalsieger,
       3 = UEFA-Pokal): if `M.0x2374[c]` not in {0, 0x1e}: byte3 = `c*8 + M.0x2374[c]`;
       if the cup winner byte (D:0x7ad / 0x7ae / 0x7af for c = 1/2/3) == team+1:
       byte3 += 0x80 and `M.0x227c[c]++` (0x227d / 0x227e / 0x227f). Only the last
       participating cup survives (a club is in at most one).
     History records are initialised to 0xff (all 50) for managers added at setup
     (f_08bc_2184); byte0 == 0xff means "EXISTIERTEN SIE NOCH GAR NICHT ALS MANAGER".
   Title counters: 0x227b Meisterschaften, 0x227c DFB-Pokale, 0x227d Landesmeister-,
   0x227e Pokalsieger-, 0x227f UEFA-Pokale (labels D-slots 0x4a6c..0x4a7c).
2. For each manager: remember `before[m] = X(team)+0x32` (long, eternal points),
   F:0x4bec[m] = 0, `topscorer[m] = f_14a4_1ad5(1)` (1 if the league's top scorer
   plays for the manager's club, else 0; f_14a4_1ad5 is the "Die Besten der Liga"
   scorer list).
3. **f_1a58_4597 "SAISONENDE"** - eternal-table points for every German club
   t = 0..63, `pos = X(t)+0x2e` (final 0-based table position):
   * `base = ((t<18) + (t<38))*25 - pos + 19`; if t > 57 base = 0
     -> BL: 69-pos (69..52), 2.Liga: 44-pos (44..25), AOL: 19-pos (19..0).
   * cup winners (byte = winner team id + 1): DFB-Pokal D:0x7ac +20,
     UEFA-Pokal D:0x7af +30, Pokalsieger D:0x7ae +40, Landesmeister D:0x7ad +50.
   * bonus: t<18: pos==0 -> +12, pos 1..4 -> +3; t>=18 and t<=57: pos==0 -> +3.
   * `X(t)+0x32 (long) += base + cups + bonus`.
   (Also clears F:0x4bec[0..3].) Displays only the "SAISONENDE" title picture.
4. **Manager rating `M.0x241e`** (int, 0..100; name UNCERTAIN, behaves like
   board confidence/"Ansehen"; initial value 50, or 10 when starting in the
   Amateur-Oberliga - f_08bc_2184 sets `50 - 20*startLeagueFlag` where the flag is 2
   for Endlosspiel):
   `K = min(D:0x7e2 + 1, 5)`;
   `delta = X(team)+0x32 - before[m]` (this season's eternal points);
   `r = old - old/K + ((delta*100)/108)/K` (32-bit, then 16-bit; `old/K` unsigned);
   clamp to [0,100] (f_3091_1009).
   **Quirk (original bug):** the value lives in one stack variable; the loop over
   managers overwrites it, and a later loop (@1a58:40f1) assigns the *last* computed
   value to every manager after applying floors cumulatively:
   `if r < 30 and league(m) < 2: r = 30; if r < 61 and league(m) == 0: r = 61;`
   `M.0x241e = r` (leagues after promotion/relegation). With one human manager this
   is simply the formula above.
   Uses of 0x241e: f_08bc_0a63 (season budget `M.0x2420` and D:0x688+m*0x24
   = `((2-league)*30 + r)*1000`), f_14a4_0906 (`+ r/5 + 25` in some income),
   f_112a_0a6d/f_14a4_2636 (if r < 95: with probability depending on D:0x68c+m*0x24/2500
   vs rand(0,22): `r += rand(1,3)`). Finance details belong to the finance spec.
5. **Licence refusal** (@1a58:3e32): for each manager: `M.0x2382 = 0`; if money
   `M.0x2432` (long) < -2,000,000 and league < 2: `M.0x2382 = 4` ("DFB verweigert die
   Lizenz. Sie steigen ab."), the club is moved to the last place of its table
   (entries after it shift up), `forced[l]++`, `M.0x237a++`.
   The number of extra relegation places is `max(0, forced[l] (+1 for l==0 if
   F:0x3060==0) - D:0x2266[l])`, i.e. a forced-relegated club normally takes one of the
   regular relegation slots. (Promotion/relegation itself: see the league section.)
6. **Promotion/relegation status** set in the same loop (f_0310_32b1 finds the
   manager of a team; `M.0x2382`/`M.0x237a` only changed if 0x2382 still 0):
   * relegated from BL/2.Liga: `M.0x2382 = 2`, `M.0x237a++`;
   * relegated out of the Amateur-Oberliga: `M.0x2382 = 8`, money `M.0x2432 = 500,000`
     and f_0310_0b24(down, up, 1) is called once extra - together with the regular swap
     this cancels out, so **the human club stays in the Oberliga** ("Sie beginnen neu
     mit 500.000 DM."). The replacement club `up` comes from slots 58..63
     (k=0: rand(61,62), then 60, 59, 58).
   * promoted: `M.0x2382 = 1`, `M.0x237a--`.
7. f_0f9d_0697(10): AI strength drift for the new season (5.5).
8. Rating assignment (step 4 second half).
9. If end year reached: game end (5.6). Then for every manager f_322b_2366(2)
   (update high-score file, 5.4) and f_14a4_2cb4 (finance).
10. Redraw, f_0310_09c5 (new random table order), f_17c2_0ef2 (cups),
    **f_0cb5_0012(topscorer[])** = season-end report per manager (below), D:0x7e2++,
    then days run until 28 July, D:0x7e0++ (calendar section).

**f_0cb5_0012 season report** per manager ("Sie beendeten Ihre Saison / <n>. Platz
der <league>", position from history record `byte0 - D:0x226e[byte1]`):
* status bit 1 (promoted): "+1,600,000 DM" if now in BL (`0x186a00`), else
  "+800,000 DM" (`0xc3500`); also advertising contracts handling (finance spec).
* bit 2: "Ihr Abstieg erschüttert Millionen Fans." bit 4: licence text. bit 8:
  "Sie beginnen neu mit 500.000 DM."
* topscorer: "Torschützenkönig aus Ihrem Team (250.000 DM)" money += 250,000.
* f_17c2_11e7(team) (team in list F:0x56fe[0..F:0x2e74-1]): "Sie sind für den
  Europapokal qualifiziert."
* youth player / player ageing / contract expiry follow (player spec).
* "Ihre Spieler werden älter und schwächer... War nur ein SCHERZ!!!" joke box with
  probability 1/5 (only in DM version, D:0x2252==D:0x224e).

### 5.3 Per-manager eternal balance display ("Ewige Tabelle/Bilanz")

f_262f_18e7:
* **Ewige Tabelle**: the 64 German clubs sorted by `X(t)+0x32` (unsigned long)
  descending with an exchange sort: for i in 0..62, for j in i..63: if
  `val[list[i]] < val[list[j]]` swap list[i], list[j] (strict, so ties keep the
  lower team id first). Shown: top 23 rows plus any rows of human clubs; points
  printed as `value % 100000` (f_3a01_1c70, UNCERTAIN: unsigned long mod).
  Initial eternal points: 0 for every club (F segment zero; the 320-byte block in
  MANA.DAT is read into a temporary buffer in f_262f_36ec and discarded - UNCERTAIN
  whether anything else seeds it; nothing writes 0xefe except f_1a58_4597 and the
  slot swaps).
* **Ewige Bilanz** of the current manager, columns GESAMT/HEIM/AUSWÄRTS
  (home+away, [0], [1]): PUN. = `0x23e6 : 0x23f2` (points : minus points; the
  ": minus" part suppressed if D:0x224d != 0, never in this version), TORE =
  `0x23ea : 0x23ee`, SIEGE 0x23fa, NIED. 0x23fe, UNEN. 0x2402.
* Title counters 0x227b..0x227f with labels MEISTERSCHAFTEN, DFB-POKALE,
  LANDESMEISTER-POKALE, POKALSIEGER-POKALE, UEFA-POKALE (0 -> " KOMMT VIELLEICHT
  NOCH...").

f_262f_390e (manager history "Sais. Pl. Liga DfB-Pokal Europapokal" +
"Gesamtentwicklung" bar graph), for i < min(D:0x7e2, 50):
* season label number = `1963 + 28*modern + M.0x237d + i`, where
  modern = (F:0xa7a0 - D:0x7e2 >= 1971) (i.e. non-historic start). UNCERTAIN exact
  printed format.
* "Pl." = `byte0 - D:0x226e[byte1]` + league name (D-slots 0x2388.. "Bundesliga",
  "Zweite Liga", "Am.-Oberliga"); highlighted if byte0==1 and byte1==0.
* DFB: byte2&0x80 -> "SIEGER", else round text slot `0x24c0 + (byte2&7)*4`
  (0x24c8 "1.Runde ", 0x24cc "Achtelfinale ", 0x24d0 "Viertelfinale ",
  0x24d4 "Halbfinale ", 0x24d8 "Endspiel "; 0x24c4 is empty).
* Europe: byte3 & 0xf8 == 0 -> "nicht im wettbewerb"; else cup name slot
  `0x24dc + ((byte3>>3)&3)*4` (1 Landesmeister, 2 Pokalsieger, 3 UEFA) + ("SIEGER "
  if 0x80 else round text `0x24c4 + (byte3&7)*4`).
* Bar graph: height from byte0 (overall rank), colour by league band (rank <= 18,
  <= 38, else).

### 5.4 High score (HIGH.xy) - f_322b_21c4, f_322b_2366, f_322b_2a2a, f_322b_2864

**File**: name built from D:0x949c "HIGH.00": char 5 = '0' + type where type =
1 (1-Jahres-Spiel, F:0x513c in {1964,1993}), 3 (3-Jahres-Spiel, {1966,1995}) else 0
(Endlosspiel); char 6 = '5' - F:0x4a28 (the Spiel-Level digit as chosen). One file per
(duration, level). Content: exactly 0x488 bytes = 20 entries x 0x3a, read/written
directly from/to F:0x1d52. f_322b_21c4(0) loads (if the file is missing it checks
that MANA.DAT is accessible, then creates a zeroed table and writes it);
f_322b_21c4(1) writes. Returns type.

**Entry layout (0x3a bytes, built at F:0x4cfe by f_322b_2a2a)**:
| off | size | meaning |
|---|---|---|
| 0x00 | 0x1a | manager name (copy of M.0x2242) |
| 0x1a | 0x17 | club name (T(team) name) |
| 0x31 | 1 | championships (M.0x227b) |
| 0x32 | 1 | DFB-Pokale (M.0x227c) |
| 0x33 | 1 | European cups (0x227d+0x227e+0x227f) |
| 0x34 | 2 | score (int) |
| 0x36 | 4 | game id = D:0x668 (long; UNCERTAIN where it is set - no direct write found, probably part of new-game/savegame block) |
Displayed as "PUNKTE: <score>" and "(M/D/E)" from bytes 0x31..0x33.

**Score** `f_322b_2a2a(m, flag)`:
1. `S = f_322b_2864(m)` (performance, below).
2. `assets = Σ_{s<squad size} f_2277_25de(s,0)` (player market values)
   `+ f_14a4_2bba() * 135000` (UNCERTAIN: stadium value units)
   `+ M.0x2432` (money, long) `- f_112a_0f8d(0,'c')` (UNCERTAIN: outstanding loans).
3. `S += assets / 80000` (signed long division, truncated; low 16 bits added).
4. `S += 40*M.0x227b + 20*M.0x227c - (500 - 60*(0x227d+0x227e+0x227f))`.
5. if flag == 0: `S += 300`. (flag = game type: 0 only for Endlosspiel, so endless
   games get +300; at real game end the call is made with flag 1.)
6. if S < 1: S = 1.

**Performance** `f_322b_2864(m)` (32-bit long `P`):
1. `v = 57 - X(team)+0x2e - D:0x226e[M.0x237a]`; if v < 0: v = 0. `P = v/2`.
   Note: at season end this runs after the slot swap, so pos is the final position in
   the *old* league but the offset is of the *new* league (quirk, reproduce as is).
2. for c = 0..3: `b = M.0x2374[c]`; if 1 <= b <= 6: `P += 2*b` (current cup rounds).
3. `n = min(D:0x7e2, 50)`; for i < n with record r = M.0x2280+4i:
   * `P += trunc((58 - r[0]) / 2)` (signed, toward zero)
   * if r[2] & 0x80 (DFB winner): `P += 20` and then Europe is evaluated:
     if r[3] & 0x80: `P += 25` else `P += 3*(r[3]&7)`;
     else: `P += 2*(r[2]&7)` and **Europe is NOT evaluated** (exact original behaviour:
     the Europe term is only reached when the DFB byte has bit 7 set).
4. if n != 0: `P /= (n+1)` (signed long). Return `(P*10)` truncated to 16 bits.

**f_322b_2366(mode)** (UI + table maintenance):
* always: zero F:0x1d52 table, f_322b_21c4(0) load, `type` = return value.
* mode != 0: candidate = f_322b_2a2a(current manager F:0x304a, type).
* mode 2 (called for every manager at every season end): search i = 0..19 for the
  first entry with same manager name AND club name (strcmp), or i == 19 with
  `entry[19].score < cand.score`; copy candidate there (+0x36 = D:0x668), then sort
  all 20 (for a in 0..18, for b in a+1..19: if score[b] > score[a] swap); write file.
  So a manager's entry is overwritten every season even if the score dropped.
* mode 1 (menu, @08bc:1bcc): display list; candidate inserted visually before the
  first entry with score <= cand.score (not saved); for Endlosspiel it then calls
  mode 2 (saved).
* mode 0 (final display at game end): list only; it also overwrites the fixture
  permutation strings D:0x6fc[i] with 'a' (anti-continuation; irrelevant for a
  re-implementation).
* rows with score 0 are not drawn; highlight = inserted row (timed games) or rows
  whose +0x36 == D:0x668 (Endlosspiel).

### 5.5 AI club strength between seasons: f_0f9d_0697(p)

Called with p=1 once per event (half-week) at the top of the game loop, and with
p=10 at season end (after promotion) and once at new game (f_08bc_0418). For each
German club t = 0..63 (foreign clubs are never changed):
1. `lg = f_17c2_1bb5(t) >> 1` (0 for t<18, 1 for t<38, 2 otherwise).
2. `r = rand(0, 2*p)` **then**, if p != 1: clear bit 7 of T(t)+0x21 and set p = 3.
   (So with p=10 the first club uses r = rand(0,20) and all later clubs r = rand(0,6) -
   reproduce exactly.)
3. `s = (p == 3) ? 3 : 1`; clamp range `[lo,hi] = (p==3) ? (D:0x5ce[lg*2],
   D:0x5cf[lg*2]) : (1, 99)` with D:0x5ce = **70,93 / 40,74 / 15,45** (BL / 2.Liga /
   AOL+others).
4. for i in 0..2: `T+0x18+i = clamp(v + rand(0, 2*(s+r)) - s - p)` (current strength).
5. for i in 0..2: `v = T+0x1b+i`; if rand(0,2)==0 or p==3:
   `v += rand(0, 2*(s+r)) - p - s`; `T+0x1b+i = clamp(v)` (clamp applied always).
6. for i in 0..2: `T+0x1e+i = clamp_[45,55](T+0x1e+i + rand(0,6) - 3)`.
Order of rand() calls per club: rand(0,2p); then 3x step 4; then for each i of step
5 rand(0,2) [and rand(...) if taken]; then 3x rand(0,6).
Human clubs: f_0f9d_0002(m, flag) (called per manager before each event) recomputes
T(team)+0x18..0x20 from the squad, overwriting the drift (player spec). It also sets
`M.0x237f = 'd'` when fewer than 8 players are available -> every match of that club
is forfeited 0:2 with a 200,000 DM fine (f_1a58_2051), except from 10 June on.
Team form T+0x17 changes after each league match in f_2a41_1fec (match spec).

### 5.6 Game end

* F:0x513c = end year: 22222 (+29 if non-historic) for Endlosspiel (never reached),
  1964/1966 (historic 1-/3-year), 1993/1995 (modern 1-/3-year).
* Checked once per season end (@1a58:4161): if `F:0x513c == F:0xa7a0` (current
  calendar year, long) the game ends:
  1. corrupt fixture strings (D:0x704[0], D:0x70c[4], D:0x718[8], D:0x720[8] = 'a');
  2. f_322b_21c4(0) load high scores; for each manager: f_322b_2a2a(m, 1), insert by
     name/club match or by beating entry 19, bubble sort (5.4);
  3. f_322b_21c4(1) save; f_322b_2366(0) show; f_1a58_000c(0) shows "ENDE";
     f_076b_114c() exits the program.
* There is no bankruptcy game over (money < -2M only triggers licence refusal).

### 5.7 Function list

| function | meaning |
|---|---|
| f_112a_314d | per-turn goal messages (lost/secured chance, relegation match, winter break warning) |
| f_1a58_3176 @3763..4130 | season-end history, rating, licence refusal, promotion status |
| f_1a58_4597 | eternal-table points per club |
| f_0cb5_0012 | season-end report per manager (bonuses, player changes) |
| f_14a4_1ad5(1) | top scorer belongs to current manager? |
| f_17c2_274d(m,c) | trophy screen (c=0 Meisterschaft, else cup name) |
| f_17c2_11e7(t) | t qualified for Europe (in F:0x56fe list) |
| f_08bc_0a63(m) | new-season budget from rating 0x241e |
| f_0f9d_0697(p) | AI strength drift (p=1 per event, 10 per season) |
| f_0f9d_0002(m,f) | human club strength from squad; 'd' status |
| f_262f_18e7 | Ewige Tabelle / Ewige Bilanz screen |
| f_262f_390e | manager history screen |
| f_322b_21c4(w) | HIGH.xy load (w=0) / save (w=1) |
| f_322b_2366(mode) | high score maintenance/display |
| f_322b_2a2a(m,f) | high score of manager |
| f_322b_2864(m) | performance part of the score |
| f_1a58_000c | "ENDE" screen |
| f_0310_0b24(a,b,f) | swap two club slots (used for promotion/relegation) |
| f_0310_32b1(a,b) | manager owning team a (returns m+1) or b (returns m+0x81), else 0 |
## 6. DFB-Pokal, Europapokal, Relegation

Notation: F:x = far segment 0x4238, D:x = DGROUP 0x4cb3 (checked against BMMAIN.lst). "Team id" = index into
the TEAM table F:0x3066 (0..63 German, 64..199 foreign). German team ids are league slots (see D:0x2272 below).
Cup index `c`: 0 = DFB-Pokal, 1 = Pokal der Landesmeister, 2 = Pokal der Pokalsieger, 3 = UEFA-Pokal.

### 6.1 Data

| Address | Size | Meaning |
|---|---|---|
| F:0x0008 + c | 1 byte x4 | round counter of cup c (1..5). Set to 0, then to 1 by the draw f_17c2_09e0; incremented by f_17c2_13a2. **f_0310_15db and f_17c2_16dc read only F:0x8 (DFB) and F:0x9 (Europe), so the three European cups must stay in step.** f_0310_02bf (relegation) sets F:0x9 = 5. |
| D:0x07d6[r] | 6 bytes | pairs per round r: `[32,16,8,4,2,1]`. Round r uses D:0x7d6[r]. Since r starts at 1, round 1 has 16 pairs (32 teams), then 8, 4, 2, 1 pairs. Index 0 (32 pairs) is never used. |
| F:0x76ca + c*0x20 + i | 32 bytes per cup | bracket: team ids. Pair k = slots (2k, 2k+1) with slot 2k at home. After a round the winner is moved to slot 2k (see 6.5). |
| F:0x4c7e + c*0x20 + i | 32 bytes per cup | result of the current leg: [2k] home goals, [2k+1] away goals. Markers on the home byte: +10 = after extra time ("n.V."), +20 = after penalties ("n.E.", the stored score is then the shootout score). The reader strips them with `while (x > 9) x -= 10`. Cleared by f_17c2_1b6d (all 4 cups). |
| F:0x21c2 + c*0x20 + i | 32 bytes, c=1..3 (abs. F:0x21e2..0x2241) | first-leg result, stored PER SLOT after the slot swap (see 6.5). Cleared when cup 1 is drawn (f_17c2_09e0 with c=1 clears all three cups) and in f_17c2_13a2 for c != 0. |
| F:0x1d17 + c | 1 byte, c=1..3 | leg flag: 0 = the next game is a first leg, 1 = the next game is a second leg. F:0x1d18 (c=1) is also the leg flag of the relegation tie. |
| F:0x2374 + m*0x30a + c | 1 byte | cup status of manager m's team: 0x1e = not taking part, 1 = drawn into round 1, (round+1) after winning a round. It is not updated while relegation is played (F:0x57dc != 0). |
| D:0x07ac | 1 byte | DFB winner + 1 (0 = none). Initial value **7**, i.e. team id 6 holds the cup at game start (UNCERTAIN which club that is after the start shuffle; MANA.DAT id 6 = 1.FC KÖLN). |
| D:0x07b0 | 1 byte | losing DFB finalist + 1. Initial value 0. |
| D:0x07ad / 0x07ae / 0x07af | 1 byte each | winner + 1 of LM / PS / UEFA if the winner is German (id < 0x40), else 0. Initial value 0. |
| F:0x57a4 + c*6 + k (c=1..3; abs. F:0x57aa/0x57b0/0x57b6) | 6 bytes per cup | German qualifiers for European cup c, terminated by 0x80. |
| F:0x56fe[F:0x2e74] | list | all German teams already qualified for Europe (used to avoid doubles). |
| D:0x2277 | 1 byte | 57: the highest team id that can be drawn for the DFB-Pokal. |
| D:0x226d | 1 byte | 4: number of UEFA places from the league table. |
| D:0x226f/0x2270 | bytes | 18 / 38. f_17c2_1bb5(id) returns 1 if id < 18 (1. Liga), 2 if id < 38 (2. Liga), otherwise 4 (Amateur-Oberliga). |
| D:0x2272..0x2279 | 4 pairs | team-id ranges per league class: (0,17), (18,37), (38,57), (58,255). |
| D:0x2256 | long | 76000: attendance used for the DFB final. |
| D:0x537c / 0x537d | bytes | 87 / 81: last event index of the DFB / Europe calendar. Used only by the "next round" text in f_17c2_1ccb. |
| D:0x0608 / 0x0609 | flags | the "DfB-Pokal" / "Europapokale" display toggles. They only switch the draw and result screens on or off (f_17c2_0006, f_0310_15db, f_0310_2303); the cups are played either way. |
| F:0x57dc | flag | 1 while the relegation tie is played (set and cleared in f_0310_02bf). |
| F:0x3060 | byte | relegation outcome (6.7). |

Names: D:0x24c4 + r*4 = round name (1 "1.Runde", 2 "Achtelfinale", 3 "Viertelfinale", 4 "Halbfinale",
5 "Endspiel"); D:0x24dc + c*4 = cup name ("DfB-Pokal", "Pokal der Landesmeister", "Pokal der Pokalsieger",
"UEFA-Pokal").

### 6.2 Calendar (event table D:0x2280; date = event index i -> day (i>>1)*7 + (i odd ? 4 : 0), day 0 = 29 July)

Main loop (f_1a58_3176): `if (ev & 8) f_0310_0261(1)` (DFB); `if ((ev & 0x70) == 0x10) f_0310_02bf()` (relegation);
`else if (ev & 0x70) f_0310_0290(0x0e)` (all three European cups together). Bits 4..6 are therefore one combined
flag; the value 0x10 on its own means Relegation.

| Competition | Event idx (date) |
|---|---|
| DFB round 1 (16 pairs) | 8 (26.08.) |
| DFB round 2 Achtelfinale (8) | 22 (14.10.) |
| DFB round 3 Viertelfinale (4) | 30 (11.11.) |
| DFB round 4 Halbfinale (2) | 64 (10.03.) |
| DFB round 5 Endspiel (1) | 86 (26.05.) |
| Europe round 1 (16 pairs), legs 1/2 | 1 (02.08.) / 5 (16.08.) |
| Europe round 2 Achtelfinale | 15 (20.09.) / 19 (04.10.) |
| Europe round 3 Viertelfinale | 31 (15.11.) / 35 (29.11.) |
| Europe round 4 Halbfinale | 63 (07.03.) / 67 (21.03.) |
| Europe round 5 Endspiel (two legs as well) | 77 (25.04.) / 81 (09.05.) |
| Relegation, legs 1/2 | 91 (13.06.) / 92 (16.06.) |

All cup ties are single games except Europe and Relegation, which are two-legged. The European final is also two-legged.

### 6.3 Qualification for Europe — f_17c2_0ef2 (season end, after the league reshuffle, and once at new game)

1. Clear all three lists F:0x57aa.. (3x6 bytes) to 0x80 and set the used list to empty (F:0x2e74 = 0).
2. **Landesmeister**: list[0] = F:0x535a[0] (1. Liga champion), which is added to the used list. If D:0x7ad != 0
   (a German club is LM cup holder): if holder = D:0x7ad-1 is not yet used, list[1] = holder; otherwise
   (the holder is the champion) list[1] = F:0x535b (1. Liga runner-up). Mark it used.
3. **Pokalsieger** (index j = 0): if DFB winner D:0x7ac-1 is not used, take it; else, if DFB finalist D:0x7b0-1 is
   not used, take that one; else nobody. Then, if D:0x7ae != 0 (German PS holder): take the holder if it is not used,
   else the DFB finalist if not used, else nothing.
4. **UEFA**: places = D:0x226d (4). If D:0x7af != 0: take the holder if it is not used (as an extra place);
   if it is already used, places++. Then walk the 1. Liga table from position index 1 (F:0x535a[1], [2], ...),
   skipping used teams, and append until `places` teams have been added.
5. Quirks that must be kept: a holder that is also the champion gives the LM place to the runner-up, so up to 2
   German LM entrants. The UEFA list can reach 5 entries (6 slots exist, there is no bound check).
6. Uses the final table order F:0x535a of the season that just ended. UNCERTAIN: whether the promotion/relegation
   code run just before has already disturbed the top positions (it only moves the bottom teams, so probably not).

### 6.4 Draw for round 1 — f_17c2_09e0(c, showOnly)

Season-end order in f_1a58_3176: `f_17c2_1b6d(); for c=0..3: f_17c2_09e0(c,0); for c=1..3: f_17c2_09e0(c,1)`.
The second pass only shows the draw. New game: f_08bc_0418 draws c=1..3 before the options screen and c=0 after it.

1. F:0x8+c = 0. The presentation (f_17c2_0006: "Möchten Sie ... Auslosung", needs D:0x608/0x609) is ignored for the logic.
2. If c == 0 or showOnly == 0:
   - If c == 1: clear F:0x21e2[3][32].
   - For every manager m: F:0x2374 + m*0x30a + c = 0x1e.
   - **DFB (c=0):** every manager's team is placed first in a distinct random slot `rand(0,31)` (re-roll on
     collision), with status 1. Every other slot s = 0..31 (skipping manager slots) gets `rand(0, D:0x2277=57)`.
     It is re-rolled while it equals a team already in DFB slots 0..s-1 or any manager's team. So 32 entrants
     are drawn at random from the 58 teams of all three leagues (ids 58..63 never play), and human clubs always take part.
   - **Europe (c=1..3):** each slot s gets `rand(64,199)` (foreign). It is re-rolled while it equals a team
     in this cup's slots 0..s-1 or in ANY slot of the other three cup arrays. Cups not yet redrawn still hold last season's
     teams, so those count too (keep this quirk; minor), and it is also re-rolled if it equals a manager team.
     Then F:0x1d17+c = 0, and for each German qualifier q in list c (until 0x80): pick `r = rand(1,31)`, re-roll
     while slot r already holds a German team (< 0x40), and set slot[r] = q. Slot 0 is never replaced.
     A manager whose team is q gets status 1.
   - The random calls run in this exact order: slot by slot, with re-rolls.
3. **DFB home rule** (only c == 0): for i = 0..31, f_17c2_1187(slot[i], slot[i-1], i). If i is odd, slot[i]
   is an Amateur-Oberliga club (class 4), slot[i-1] is class 1 or 2, and the round F:0x8 < 4, then swap the two,
   so the amateur plays at home. This applies in rounds 1-3 only (also in f_17c2_13a2).
   For i = 0 it reads F:0x76c9, the byte before the array; with i even no swap happens anyway.
4. F:0x8+c = 1.

There is no seeding and no country protection, and German clubs can meet each other.

### 6.5 Playing a cup event — f_0310_15db(0, mask) and f_17c2_1226 / f_17c2_16dc

mask: 1 = DFB (from f_0310_0261), 0x0e = the three European cups (from f_0310_0290), 2 = relegation (from f_0310_02bf,
which uses cup slot 1).
- Teams in the round: n = D:0x7d6[F:0x8] * 2 for mask 1, else n = D:0x7d6[F:0x9] * 2.
- For every cup c whose bit is set and every pair i = 0, 2, ..., n-2: the match is slot[i] (home) against slot[i+1]
  (f_1a58_20b2(home, away, mask, c, i, 0)). Minutes 1-45 and 46-90 are played (f_0310_2303).
- **After 90 min**: f_17c2_1226(mask, n, 0) returns nonzero if any tie needs extra time:
  - DFB: if home == away goals, add +10 to the home goals (marks extra time).
  - Europe/relegation: only on a second leg (F:0x1d17+c != 0) and only when f_17c2_15e8(c,i) == 0x1e.
    Then +10 to the home goals.
  If nonzero, minutes 91-105 and 106-120 are played and f_17c2_1226(mask, n, 1) is called. A tie that is still
  undecided (DFB: `home - away == 10`; Europe: f_17c2_15e8 == 0x1e again) gets +10 and goes to penalties:
  f_0310_356d(c, i, mask).
- **Penalties f_0310_356d**: with no human team involved, home = rand(2,5) and away = rand(2,5) are re-rolled until they
  differ, and the stored home value is home+20. With a human team the shootout is played interactively (5 rounds, then
  sudden death). The final shootout score (+20 marker) REPLACES the match score in F:0x4c7e.
- **Tie decision f_17c2_15e8(c, i)** (first-leg data must be stored per slot, see below):
  `h2 = F:4c7e[i] mod 10, a2 = F:4c7f[i], f1 = F:21c2[c][i], f2 = F:21c2[c][i+1]`.
  slot i = team Q (home in leg 2, away in leg 1), slot i+1 = team P.
  Q_total = f1 + h2, P_total = f2 + a2; away goals: Q = f1, P = a2.
  ```
  if (day < 316 /*0x13c*/ || Q_total != P_total) {
      if (Q_total < P_total || (Q_total == P_total && f1 < a2)) return 1;   // Q is out
      if (!(Q_total == P_total && f1 == a2))                     return 0;   // Q goes through
  }
  return 0x1e;                                                               // still level
  ```
  Day >= 316 only applies to the relegation legs (days 319/322), so the relegation has NO away-goals rule.
  **Faithful bug**: after penalties h2 is the shootout score, so the check compares f1+penH with f2+penA. That
  can eliminate the team that won the shootout (for example legs 1:3 and 3:1, shootout 5:4 -> 1+5 = 6 < 3+4 = 7, Q out),
  and a level result returns 0x1e, which f_17c2_16dc treats as "Q out".
- **Advancing f_17c2_16dc(mask-1)**:
  - DFB: per pair, strip the +10 markers from home and set winner = home unless home < away (then slot[i] = away).
    Managers whose team is now in slot i get status = F:0x8+1. Final (n == 2): D:0x7ac = winner+1,
    D:0x7b0 = loser+1. If the winner belongs to a manager, the trophy screen f_17c2_274d(m,1) is shown.
    The final pair is then restored for display.
  - Europe (c = 1..3; only c = 1 in relegation mode):
    - First leg (F:0x1d17+c == 0): set the flag to 1. Per pair: F:21c2[c][i] = away goals, F:21c2[c][i+1] = home goals,
      and swap slot[i] and slot[i+1] (leg 2 has the other club at home; the first-leg data now belong to the new slots).
    - Second leg: set the flag to 0. Per pair: if f_17c2_15e8 != 0, swap, so the winner is in slot i. Managers in slot i
      get status F:0x8+c+1 (not in relegation mode). In the final (n == 2, not relegation): D:0x7ad/7ae/7af =
      winner+1 if the winner id < 0x40, else 0. A human winner gets f_17c2_274d(m, c+1).
  - Next round: DFB: if not relegation and F:0x8 <= 4, call f_17c2_1b6d (clears ALL cup results) and f_17c2_13a2(0).
    Europe: if not relegation and F:0x1d18 == 0 (after a second leg) and F:0x9 <= 4, call f_17c2_1b6d and then
    f_17c2_13a2(c) for c = 1..3.
- **Next-round draw f_17c2_13a2(c)** (does nothing if F:0x8+c == 5):
  1. c != 0: clear F:0x21e2[3][32].
  2. n = D:0x7d6[old round] (number of winners); F:0x8+c++.
  3. Compaction: for k = 0..n-1 (k < 16): slot[k] = slot[2k].
  4. Shuffle: repeat 20 times: `a = rand(1,n)-1; b = rand(1,n)-1; swap(slot[a], slot[b])`.
  5. DFB only: apply the amateur home rule f_17c2_1187 for i = 0..31 (with the new F:0x8).
- An eliminated human club keeps the round it reached in F:0x2374 and takes no further part. There are no other
  consequences in these functions.

### 6.6 Money (only as far as the scheduling code touches it; the rest is in f_1a58_20b2)

- There is no prize money per round in the cup code; income comes from gate receipts in f_1a58_20b2.
  DFB ties (mask != 0, cup 0): the gate is divided by 2 and both clubs get their half (local_26 = 2).
  European ties: only the home club takes the gate.
- DFB final (cup 1 with F:0x8 > 4): attendance is fixed at D:0x2256 = 76000, and each manager in the final gets
  +532,000 DM (0x81e20) flat.
- If a team cannot play (manager flag +0x237f == 'd', outside 10.-30.06), the game is forfeited 0:2 and the club
  pays 200,000 DM (f_1a58_2051, f_0310_152e). That also applies to cup games.

### 6.7 Relegation — f_0310_02bf (events 91/92, value 0x10)

1. F:0x9 = 5 (1 pair), F:0x57dc = 1, show "Relegationsspiel".
2. Save cup-1 slots F:0x76ea/0x76eb, results F:0x4c9e/0x4c9f and first-leg F:0x21e2/0x21e3; set F:0x4c9e/9f = 0.
3. Leg 1 (F:0x1d18 == 0): first-leg data = 0; home F:0x76ea = F:0x5370 (2. Liga, table position index 2 = 3rd),
   away F:0x76eb = F:0x5369 (1. Liga position index 15 = 16th = 3rd from bottom).
   Leg 2: first-leg data = stash F:0x2e6e/0x2e6f; home = 1. Liga 16th, away = 2. Liga 3rd.
4. f_0310_15db(0, 2) plays the leg (extra time and penalties as in 6.5, using f_17c2_15e8 without away goals), then
   f_17c2_16dc(1) handles leg storage and swap.
5. After leg 1 (the flag is now 1): stash F:0x2e6e = F:0x21e2 (1. Liga club's leg-1 goals), F:0x2e6f = F:0x21e3 (2. Liga club's).
   After leg 2 (the flag is now 0): **F:0x3060 = f_17c2_15e8(1,0)**: 0 = the Bundesliga club survived, 1 = it lost.
   0x1e is possible in the penalty edge case of 6.5. The season-end code in f_1a58_3176 uses F:0x3060 together with D:0x226a.
6. Restore the saved Europe cup-1 state and set F:0x57dc = 0. The relegation updates no manager cup status and draws no rounds.

### 6.8 Function list

| Function | Meaning |
|---|---|
| f_17c2_0006(c) | asks whether the draw should be shown and loads its graphics; sets F:0x57c8. UI only |
| f_17c2_0378 | animation of one drawn ball. UI only |
| f_17c2_089a | closes the draw screen and shows the fixtures (f_17c2_1ccb) |
| f_17c2_09e0(c, showOnly) | first-round draw of cup c (6.4) |
| f_17c2_0ef2 | European qualifiers (6.3) |
| f_17c2_1187(a, b, i) | DFB amateur home rule |
| f_17c2_11e7(t) | is t in the qualified list F:0x56fe? |
| f_17c2_1226(mask, n, pass) | extra time / penalties needed? (6.5) |
| f_17c2_13a2(c) | next-round draw: compact the winners, 20 random swaps (6.5) |
| f_17c2_15e8(c, i) | decides a two-legged tie: 1 = slot i out, 0 = slot i through, 0x1e = level |
| f_17c2_16dc(c') | handles winners, first-leg storage, cup winners and next round (6.5) |
| f_17c2_1b6d | clears F:0x4c7e[4][32] |
| f_17c2_1bb5(id) | league class 1/2/4 by team id |
| f_17c2_1bf0 | text "LIGA gegen LIGA". UI only |
| f_17c2_1ccb(c, minute) | fixtures and results screen ("n.V."/"n.E." from the +10/+20 markers, next-round date). UI only |
| f_17c2_274d(m, k) | trophy screen (k: 0 = Meisterschaft, 1 = DFB, 2..4 = LM/PS/UEFA). No state changes |
| f_0310_0261 / f_0310_0290 / f_0310_02bf | DFB / Europe / Relegation event handlers |
| f_0310_356d(c, i, mask) | penalty shootout; overwrites the result with the shootout score + 20 |

---------------------------------------------------------------------------------------------------

## 7. Function index (topic-relevant)

| Function | Meaning |
|---|---|
| f_08bc_0418 | main: init, MANA.DAT, id shuffle, cup draws, new-game dialog, starts the loop |
| f_08bc_0004 | graphics/palette init (NOT fixtures) |
| f_08bc_01fb | clears head-to-head F:9336, cup status 0x1e, M+0x225f |
| f_08bc_2184 | new-game dialog: managers, team choice, level, game length, historic start, start values |
| f_08bc_1df3(id,mask) | 1 if id is drawn in a European cup c∈{1,2,3} with bit c−1 of mask (current round F:9) |
| f_08bc_1fc4(a,b) | swap the 0x36-byte season records of a and b |
| f_17c2_1bb5(id) | league bitmask 1/2/4 from id |
| f_262f_36ec | MANA.DAT loader; initial leagues (ids 0..17/18..37/38..57) |
| f_262f_01b8 | in-game options (display toggles, speed, level cheat keys) |
| f_1a58_3176 | main game loop (events, days, turns, matches, season end, promotion/relegation) |
| f_1a58_2a06(k) | Christmas / birthday cards (25%), Christmas bonus payment |
| f_1a58_4597 | season end summary, eternal points (§5) |
| f_0310_1190(p) | date from day counter (p=0) or from event index p |
| f_0310_0fb6 | calendar display (weekday = (day+5)%7) |
| f_0310_0463(e,l) | random weather postponements (e 25..69) |
| f_0310_05f1 | find a free date for a postponed game, write record F:5714 |
| f_0310_000a / f_0310_07a2 | Nachholspiele check / play |
| f_0310_0995 / 0261 / 0290 / 02bf | play league matchday / DFB round / European round / relegation leg (all via f_0310_15db) |
| f_0310_15db(lmask,cmask,…) | generic event player (fixtures, snapshot, match engine per pair, minute loops, tables) |
| f_0310_2303 | minute loop 1–45 / 46–90, half-time/result screens, table update at minute 90 |
| f_0310_09c5 | random id shuffle within each league (55 swaps per league) |
| f_0310_0b24(a,b,f) | swap club identities a↔b (all per-club data + references) |
| f_0310_0f44 | swap opponent ids inside the record table F:a4ca |
| f_0310_32b1(a,b) | manager owning team a (returns m+1) or b (m+0x81), 0 = none |
| f_0310_152e | write forfeit results |
| f_1a58_2051 | forfeit: message + 200,000 DM fine |
| f_2a41_10e3(l,md) | fixture list of a matchday from the permutation strings |
| f_2a41_2d33(l,mode,which) | table snapshot/update + sort |
| f_2a41_1ef4 | copy stat fields current↔snapshot |
| f_2a41_1fb0 | append a form-string char |
| f_2a41_1fec | TEAM+0x17 adjustment after a league game |
| f_2a41_214c | table screen |
| f_2a41_120a | fixtures/results screen ("SPIELTAG", "NACHHOLSPIELE", "(N)" with new date) |
| f_112a_314d | per-turn goal/"letzte Chance" messages, winter-break and relegation notices (§5.1) |
| f_17c2_* | cups (§6) |
| f_322b_21c4 / f_322b_2366 / f_322b_2a2a / f_322b_2864 | high score (§5.4) |

## 8. Open questions / UNCERTAIN (summary)

1. Direction of the level scale: internal L = 5 − displayed number; whether displayed 0 or 5 is the
   easiest level must be decided from the level formulas (start money 1.9M only for L==4 is odd).
2. D:224c / D:224d / D:224e / D:2252 are treated as build constants (0, 0, 1, 1); their alternative
   values (other language/currency builds) are not specified.
3. Meaning of the start values M+0x234c, M+0x23a0..0x23d0, D:66c/D:68c (finance spec).
4. The Oberliga-relegation double swap (human club stays, money := 500,000) looks like an original
   bug but is specified literally.
5. f_0310_0463 postpones games of leagues 1 and 2 also on events where only league 0 plays (e=23,
   value 0x01) — literal behaviour, probably unintended.
6. Nachholspiele are evaluated as "league 0" and only league 0's table is re-sorted immediately;
   the scratch-row clear is 19 instead of 20 bytes.
7. Initial cup holder D:7ac = 7 (id 6) refers to a slot, which is shuffled before the first season.
8. See also the UNCERTAIN marks in §5 and §6.
