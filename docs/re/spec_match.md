# BMMAIN.EXE – Match engine specification

Scope: how results are computed (AI vs AI and matches with a human manager), the
minute-by-minute event generation, cards / injuries / substitutions, extra time and
penalty shoot-out, attendance and gate money, and what is updated after a match.
All addresses are image-relative `f_SSSS_OOOO`. `F:` = far data segment 0x4238, `D:` =
DGROUP 0x4cb3. `rnd(a,b)` = `f_076b_0cc7(a,b)` = `rand() % (b-a+1) + a` (MS C LCG).
Integer division is C (truncation toward zero) unless noted; 16-bit `int` arithmetic.
`m` = human manager index (0..3), `MGR(m)+x` means `F:x + m*0x30a` (the code always
writes the absolute address of manager 0, e.g. `MGR+0x2260` = team id).

---------------------------------------------------------------------------------
## 0. Overview / call graph for one match day

Game loop `f_1a58_3176`, per day with match flags `b = D:0x2280[F:0x16e]`
(bit0..2 league 0..2, bit3 DFB-Pokal, bits4..6 Europapokal, bit7 Nachholspiele):

1. For each human manager m: `f_0f9d_0002(m,1)` (lineup -> team strengths, see §3),
   `F:0x90c6[m]=0` (substitutions made), `F:0x1d14[m]=0` (plays today flag).
   `F:0x304a = 0x1e` (no "current manager" during matches!).
2. if `b&7`: if `b&1` `f_0310_0463(day,0)` (postpone winter matches, §14),
   `f_0310_0463(day,1)`, `f_0310_0463(day,2)`, then `f_0310_0995(b)` = Ligaspiel
   -> `f_0310_15db(leagueMask=b, cupMask=0, ..)`.
3. if `b&8`: `f_0310_0261(1)` = DFB-Pokal -> `f_0310_15db(0, 1, ..)`.
4. if `(b&0x70)==0x10`: `f_0310_02bf()` Relegation (§15); elif `b&0x70`: `f_0310_0290(0xe)` =
   Europapokal -> `f_0310_15db(0, 0x0e, ..)` (cup slots c=1..3).
5. if `b&0x80` and `f_0310_000a(day)` (a postponed match exists for that day):
   `f_0310_07a2(day)` (Nachholspiele -> `f_0310_15db(0,0,..)`).

Daily (independent of matches): `f_0f9d_0697(1)` (AI strength random walk, §4).

`f_0310_15db(leagueMask, cupMask, ., fast)` = one "Spieltag" of one competition:

```
setup: for league: for lg in 0..(leagueMask==0 ? 0 : 2):
          if leagueMask: f_2a41_10e3(lg, D:0x225a[lg])     // build pair list F:0x4b5e
          f_2a41_2d33(lg, 1, 1)                             // table backup (see §13.1)
          for idx = 0,2,4.. while pair list[lg][idx] != 'F'(0x46) && idx < teams:
             if R[lg][md][idx] != 30: r = f_1a58_20b2(home,away,0,lg,idx,leagueMask)
                 if r: forfeit -> R:=30 now, remember (r,lg,md-1,idx) in list F:0x549c..
       for cups: for c in 0..3 with bit c in cupMask: for idx < pairs*2 step 2:
             f_1a58_20b2(home,away,cupMask,c,idx,0)
period(1,45); period(46,90)         // f_0310_2303, see §8
F:0x2e99 = 0
if cup and f_17c2_1226(cupMask, n, 0):           // draws -> extra time (§12)
    period(91,105); period(106,120)
    if f_17c2_1226(cupMask, n, 1): ...           // remaining draws -> shoot-out
    (+ cup result screens f_17c2_1ccb)
if league: per league in mask with option D:0x600[lg*3]: table screen f_2a41_214c
if cup:    f_17c2_16dc(cupMask-1)                // winners advance (§12.4)
f_2e3a_23aa(cupMask)                             // player grades, newspaper (§13.4)
```

Forfeit results (`f_1a58_2051`, §6.1) are re-written by `f_0310_152e`: at the start of each
period every remembered forfeit result is set to 30:30 (skip) and at the end of each period
to `R_home = (r==2)?2:0`, `R_away = (r==1)?2:0` (r=1 home team forfeits -> 0:2).

---------------------------------------------------------------------------------
## 1. Data structures used by the match engine

### 1.1 Result tables
* League results `R(lg,md,idx)` = byte `F:0x6dc8 + (lg*0x26 + md)*0x14 + idx`
  (idx even = home goals, idx+1 = away goals of pair idx/2), `md = D:0x225a[lg]`
  (current match day, 1-based, incremented by the game loop before the matches).
  Value 30 (0x1e) = match postponed / not to be played. `'F'` (0x46) in the pair list
  terminates the list.
* League pair list: `F:0x4b5e + lg*0x14 + idx` = home, `+idx+1` = away team id.
  Built by `f_2a41_10e3(lg, md)` from schedule tables `D:0x6fc` (18 teams) / `D:0x744`
  (20 teams) (letters - 'a' + league team offset). (Scheduling = other topic.)
* Cup pair list: `F:0x76ca + c*0x20 + idx` (home), `+1` (away), c = 0 DFB-Pokal,
  c = 1..3 the three European cups (c = 1 also used for the Relegation).
  Pairs this round: `D:0x7d6[round]` = [32,16,8,4,2,1]; round = `F:0x8` (DFB) or `F:0x9` (EC).
* Cup results: `F:0x4c7e + c*0x20 + idx` (home), `+1` (away). Encoding of the HOME
  byte: `<10` normal, `10..19` = extra time was played (+10), `>=20` = shoot-out result
  (+20). Display / evaluation always uses `value mod 10` (loop `while v>9: v-=10`).
* EC first-leg result: `F:0x21c2 + c*0x20 + idx` = goals the *second-leg home team*
  scored in the first leg (it was away), `+1` (0x21c3) = goals of the second-leg away
  team in the first leg (stored by `f_17c2_16dc`, then the pair is swapped).
* Chance minutes ("Torchancen" schedule), 8 slots per side:
  league `F:0x515c + (lg*0x14 + idx)*8 + j` (home side, idx even) and `+8` (away side);
  cups `F:0x4d3c + (c*0x20 + idx)*8 + j` / `+8`. Value = minute (0 = empty).
  (Written by `f_0310_12fe` with the formula base `F:0x433c + (lg*0x20+idx)*8` for lg>=10.)

### 1.2 TEAM strength fields (TEAM t, `F:0x3066 + t*0x22`)
| off | abs | meaning in the engine |
|---|---|---|
| +0x17 | 0x307d | "form" F_t, 45..55 at load, changed after league matches (47..53, §4.2) |
| +0x18..+0x1a | 0x307e.. | `cur[i]` i=0 Abwehr,1 Mittelfeld,2 Angriff |
| +0x1b..+0x1d | 0x3081.. | `base[i]` (decays during the match) |
| +0x1e..+0x20 | 0x3084.. | `rnd[i]` 45..55 (random walk, §4.1) |
| +0x21 bit7 | 0x3087 | "ticker" flag (user-selected team, goals are announced) |
For human-managed teams cur/base/rnd are **overwritten** from the lineup by `f_0f9d_0002` (§3).
For AI teams cur and base start equal (MANA.DAT values).

### 1.3 Squad slot fields used (slot s of manager m, `F:0x774a + (m*25+s)*0x34`)
| off | meaning |
|---|---|
| +0x00 | red cards (count) |
| +0x01 | yellow cards (count; yellow-red: -1) |
| +0x02 | yellow-red cards (count) |
| +0x03..+0x05 | goals this season per competition (0 league, 1 DFB, 2 EC) |
| +0x06..+0x08 | appearances this season per competition |
| +0x09 | status: bit0 suspended, bit1 injured (counter in +0x0d), bits 5..7 other |
| +0x0a | lineup shirt 1..11 = on the pitch, 0 = not in team, >=12 bench |
| +0x0d | games/weeks left of suspension / injury |
| +0x0f | PERSON index |
| +0x10, +0x11, +0x12 | the three player values (CONTEXT: Kondition/Technik/Form – names UNCERTAIN; called K, T, P below) |
| +0x13 | "E" (Erschöpfung per CONTEXT), clamped 50..150 by the match setup |
| +0x15 | match rating accumulator (signed, reset to 0 before each match) |
| +0x17 | injury kind 0..17 |
| +0x19 | tactical role 0..6 (3 = neutral; compared to PERSON+0x20) |
| +0x1a | formation position code; `pos = 7 - (+0x1a)` (0 = goal ... 7 = forward) |
| +0x1c..+0x20 | word: career appearances per competition |
| +0x22..+0x26 | word: career goals per competition |
PERSON (`F:0x57dd + p*0x25`): +0x1c/+0x1d/+0x1e skills, +0x1f (0x57fc) position code
(0..99, /25 = 0 TW,1 AB,2 MF,3 ST), +0x20 (0x57fd) preferred role 0..6, +0x21 owner,
+0x22 (0x57ff) league goals, +0x23 (0x5800) league games, +0x24 (0x5801) team.

### 1.4 Manager record fields used (`MGR(m)+x`)
`0x2260` team; `0x234c` ticket price (DM); `0x234d+k` table position copies;
`0x2373` training "Intensität" 0..34 (default 16, slider in `f_1ecd_1560`);
`0x2374+c` cup round reached; `0x237a` league (0,1,2); `0x237c` home games counted;
`0x237f` discipline value 0..40, **100 = fewer than 8 players in lineup (forfeit)**;
`0x2380` bit0 "crowd" event flag (§6.3); `0x238c+k` attendance/1000 per home game;
`0x23a0`/`0x23a8` (long) stadium capacity parts (capacity = sum);
`0x23b0`,`0x23b8`,`0x23c0`,`0x23c8`,`0x23d0` (long) stadium/club attributes used by the
attendance formula (meaning UNCERTAIN – stadium module); `0x241e` (word) used by
attendance; `0x2426` (long) season attendance sum; `0x242a`/`0x2436` max attendance +
opponent; `0x242e`/`0x243a` min attendance (init 99999) + opponent; `0x2432` (long) cash.

### 1.5 Per-human match state (indexed by m)
`F:0x1d14[m]` team playing now; `F:0x0178[m*2]` remaining yellow-card budget (6 in
league, 0 in cups); `F:0x0179[m*2]` red card still possible (1 league, 0 cups);
`F:0x1d1c[m*6+k]` booked players (slot, 0x7f empty); `F:0x5396[m*2]` keeper subs left (1),
`F:0x5397[m*2]` field subs left (2); `F:0x90c6[m]` subs made; `D:0x664[m]` players on
the pitch (11, -1 per red card); `F:0x21d8+(m+1)*2` planned chances (home,away) of the
current period; `F:0x5710[m]`/`F:0x5778[m]` chances (home/away team) so far in the
period; `F:0x5140+m*6`: [0] yellow, [1] red, [2] injuries, [4] home chances,
[5] away chances (display); watched-match list `F:0x3052[k]` (manager+1),
`F:0x305c[k]` lg/c, `F:0x0028[k]` idx, count `F:0x4bf0`.

### 1.6 Match report record (newspaper) `F:0x90ca + m*0x9a` (written by `f_2e3a_223e`, `f_1a58_20b2`)
+0..+3 scores at the moment of minimum/maximum goal difference; +4 goals for; +5 goals
against; +6 number of chances; +7 table position (opponent); +9 current goal diff;
+0xa min diff; +0xb max diff; +0xe own team; +0xf opponent; +0x11 state (0 draw,1 lead,
2 behind); +0x12 "sold out" flag; +0x13+i minute / +0x27+i scorer slot ('d'=100 =
opponent goal) / +0x63+i assister of goal i, count +0x8b; +0x3b+i / +0x4f+i / +0x77+i
minute/shooter/assister of missed chance i, count +0x8c; +0x8d+i booked players, count
+0x94; +0x93 sent-off player+1; +0x96 (long) attendance.

### 1.7 Constant tables (dumped from BMMAIN.bin)
| address | values | use |
|---|---|---|
| D:0x008e | `[80,20,0, 25,50,25, 0,20,80]` | weights W[kind][i] for S(), kind 0 defence, 1 midfield, 2 attack |
| D:0x02a2 | `[0,0,0,1,1,1,2,2]` | line of formation position pos (0 def,1 mid,2 att) |
| D:0x02aa | `[1,2,2,2,2,2,3,3,3,3,3,3,3,3,4,4,4,4,4,4]` | chance spread T[d], d=0..19 (d>=20: d/5) |
| D:0x05bc | `[1,2,1,2,2,4,2,1,1,6,12,5,7,10,11,9,20,14]` | injury base duration (weeks) per kind |
| D:0x23b4.. | 18 names: Oberschenkelzerrung, Prellung d. Knöchels, Platzwunde am Kopf, Stauchung d. Fußgelenkes, Sprunggelenkprellung, Bänderdehnung, Bauchmuskelzerrung, Bluterguss am Knie, Nasenbeinbruch, Außenbandanriß, Schienbeinbruch, Muskelfaserriß, Meniskusverletzung, Innenbandabriß, Wadenbeinbruch, Kapselriß, Leistenbruch, Kreuzbandriß | injury kinds 0..17 |
| D:0x05ce | `[70,93, 40,74, 15,45]` | clamp [lo,hi] of AI strengths per level (Liga1, Liga2, Amateur) at season start |
| D:0x02c8 | long `[4500,2200,1200,600]` | cup capacity estimate ×10 per bracket |
| D:0x5390 | word `[16,14,10,8]` | ticket price for DFB away games per home-team bracket |
| D:0x5331 | `[0,2,3,4,5,6,7,8,9,10]` | AI goal attribution (§13.3) |
| D:0x2256 | long 76000 | fixed attendance DFB final |
| D:0x225a | `[1,1,1]` | current match day per league (initial) |
| D:0x225e | `[18,20,20]` | teams per league |
| D:0x2262 | `[34,38,38]` | match days per league |
| D:0x226e | `[0,18,38]` | rank offset per league |
| D:0x226f/0x2270 | 18 / 38 | first team id of league 1 / 2 (f_17c2_1bb5) |
| D:0x2271 | 2 | points for a win (2-point rule) |
| D:0x2272 | `[0,17, 18,37, 38,57, 58,255]` | team-id brackets (league 0,1,2, others) |
| D:0x227a | `"sSnNuU"` | form chars: home win s / away win S / home loss n / away loss N / draw u (home) U (away) |
| D:0x7d6 | `[32,16,8,4,2,1]` | pairs per cup round |
| D:0x7b8 | `[31,28,31,30,31,30,31,31,30,31,30,31]` | month lengths (calendar) |
| D:0x9b4a.. | doubles 1.0, 2.0, 0.01, 10/43, 1150.0, 1000.0 | attendance formula |

Global settings: `G = D:0x4a28` = 5 - LEVEL (0..5; G=5 easiest, 0 hardest).
`D:0x5d4` = no delays/animations; `D:0x607` = 3D goal scenes; `D:0x63a` speed;
`D:0x5fe/5ff/600 + k*3`, `D:0x608/0x609` = "show results/table" options.

---------------------------------------------------------------------------------
## 2. Team strength function S(t, kind, minute) – `f_0310_1468`

```
S = 0
for i in 0..2:
    b = base[t][i]                                  // F:0x3081+t*0x22+i (unsigned byte)
    v = b + (10*minute) / (-1 - b)                  // = b - trunc(10*minute/(b+1))
    if v < 0: v = 0
    S += ((v + cur[t][i] + rnd[t][i]) * W[kind][i]) / 100     // long division, per term
return S + (form[t] - 44)                           // form = F:0x307d+t*0x22
```
kind 0 = defence (80/20/0), 1 = midfield/overall (25/50/25), 2 = attack (0/20/80).
The `base` part wears off during the match; a team with low base loses more
(base 40: -11 at minute 46, -21 at minute 90; base 70: -6 / -12).

---------------------------------------------------------------------------------
## 3. Strength of a human team from its lineup – `f_0f9d_0002(m, full)`

Called with full=1 before each match day (game loop), after every red card
(`f_1a58_0ca3`), and by several screens; full=0 is a simplified display version.
All arrays are int[3] indexed by line L = `D:0x2a2[pos]`, `pos = 7 - (signed)slot+0x1a`.

```
C = [-9,-8,-5]   // line balance
X = [0,0,0]; N = [0,0,0]; A = [0,0,0]; B = [8,8,8]; Z = [0,0,0]
R = 0; D = 0; keeper = 1; n = 0
for s in 0 .. count(m,0)-1:            // count(m,0) = slots with a person (f_3091_1109)
    p = slot(m,s); if not (1 <= p+0x0a <= 11): continue
    n++; pos = 7 - p+0x1a; L = line[pos]
    if full:
        role = p+0x19
        if role in (1,5): R += 1;  if role in (0,6): R += 2
        rm = |PERSON(p).0x20 - role|                       // f_0310_1bd7
        pm = |PERSON(p).poscode - ((7-p+0x1a)*75/7 + 5)|   // f_0310_1cd3 (expression as coded:
                                                           //  ((p+0x1a - 7) * -75)/7 + 5)
        if pos < 3:
            C[0] += 2
            if pos==1 && role==3 && rm<2 && (T-10) > K: B[L] += 15
            if pos==0 && role==3: keeper = (PERSON.poscode/25 == 0) ? 0 : 2
        else:
            if   pos==3: C[0]++; C[1]++; if 1<role<5 && rm<2 && (T-10) > K: B[L] += 15
            elif pos<6:  C[1] += 2
            elif pos==6: C[1]++; C[2]++
            else:        C[2] += 2
            if rm > 2:  X[L] += rnd(2,6)
            if pm > 25: X[L] += rnd(2,6)
        D    += K - T
        A[L] += E6/6 - E/20 + 3        // E6 = slot+0x06 (league apps), E = slot+0x13
        B[L] += slot+0x03 - 1          // league goals this season
    A[L] += K; B[L] += T; Z[L] += P; N[L]++          // K=+0x10, T=+0x11, P=+0x12
for i: B[i] += ((G-5) * X[i] * 20) / 100
if n == 0: B = [0,0,0]
MGR(m)+0x237f = (n < 8) ? 100 : 0
k = 0; while n != 0 and n < 11: N[k]++; n++; k = (k+1)%3      // missing players count as 0
if full:
    if G == 0 and R > 8: B[2] -= 100*N[2]
    if keeper != 0:                                // 1 = no keeper, 2 = field player in goal
        q = keeper - 1
        B[0] = B[0] * rnd(10q+5, 20q+5) / 100
        B[1] = B[1] * rnd(10, 40) / 100
        A[0] = A[0] * rnd(5q+10, 30q+20) / 100
    clamp B[0], B[1], A[0] to [0, 32000]
    if C[1] < 1:
        if C[0] > 0: C[2] += C[1]-1
        if C[2] > 0: C[0] += C[1]-1
    if C[0] < -1 and rnd(7,10) > C[0]+10: C[1] -= rnd(3,5)
    for i in 0..2:
        B[i] += G
        if A[i] != 0: A[i] += MGR+0x2373 - 16              // Intensität
        if B[i] != 0 and G < 4:
            B[i] += (F:0x90c6[m]*3 + N[i]*C[i]) * 10        // subs made + line balance
            A[i] += N[i]*C[i]*5
        B[i] = max(B[i],0); A[i] = max(A[i],0)
    D = clamp(D/9, -4, 4); D = max(D,0); D = clamp(D + MGR+0x2373, 0, 40)
    if MGR+0x237f != 100: MGR+0x237f = D
team = MGR+0x2260
for i: if N[i] <= 1: cur=base=rnd=0
       else: cur[i] = A[i]/N[i]; base[i] = B[i]/N[i]; rnd[i] = Z[i]/N[i]   (stored as byte, &0xff)
```
The random factors (keeper penalty, X bonuses) are re-drawn on every call.

---------------------------------------------------------------------------------
## 4. AI strength changes

### 4.1 `f_0f9d_0697(p)` – random walk, German clubs 0..63 only
Called with p=1 daily (game loop) and p=10 at new game / new season.
```
for t in 0..63:
    lvl = f_17c2_1bb5(t) >> 1           // 0 if t<18, 1 if t<38, else 2
    r = rnd(0, 2p)                      // drawn with the CURRENT p (10 only for team 0!)
    if p != 1: TEAM[t].0x3087 &= 0x7f; p = 3   // sticky for the remaining teams
    k = (p==3) ? 3 : 1
    (lo,hi) = (p==3) ? D:0x5ce[lvl] : (1,99)
    for i: cur[i]  = clamp(cur[i] + rnd(0, 2(k+r)) - k - p, lo, hi)
    for i: if rnd(0,2)==0 or p==3: base[i] += rnd(0, 2(k+r)) - p - k
           base[i] = clamp(base[i], lo, hi)
    for i: rnd[i] = clamp(rnd[i] + rnd(0,6) - 3, 45, 55)
```
(Daily: mean change 0; season start: re-randomised and forced into the league band
70..93 / 40..74 / 15..45.) Foreign teams 64..199 never change. For human teams the
values are overwritten by §3 anyway.

### 4.2 Form after league matches – `f_2a41_1fec(t, goaldiff)`
`form[t] += (gd > 0 ? +1 : -1) * rnd(0, 2|gd|)`, then clamp 47..53 (draw: rnd(0,0) = 0).
Called from the table update for both teams (§13.1). Cup matches do not change form.

---------------------------------------------------------------------------------
## 5. Results are "chances" scheduled per period

The engine is two-stage:
1. At the start of every period (1-45, 46-90, 91-105, 106-120) `f_0f9d_08e9` computes
   the number of **chances** per side and `f_0310_12fe` assigns them to random minutes.
2. In each of those minutes `f_0f9d_0c3b` decides whether the chance is a goal.

### 5.1 Number of chances – `f_0f9d_08e9(lg, idx, home, away, from, to, schedule)`
```
if R(lg,md(lg),idx) == 30: return       // (for cups lg>=10 this reads an unrelated byte – quirk)
h = 0; a = 0
h += rnd(0,1)                                  // home advantage
sH = S(home,1,from); sA = S(away,1,from)
d  = sH/3 - sA/3                               // computed as sA/-3 + sH/3
if d > 0:
    n = (d <= 19) ? T[d] : d/5                 // T = D:0x2aa
    x = rnd(1,n); y = rnd(1,n)
    if rnd(0,1) != 0 and d > 8: h += pick(x,y) else: h = x      // NOTE: '=' drops home bonus
    if d < 25: a = rnd(0,2)
else:
    e = -d
    n = (e <= 19) ? T[e] : e/5
    if e < 25: h += rnd(0,2)
    x = rnd(1,n); y = rnd(1,n)
    if rnd(0,2) == 0 and e > 8: a += pick(x,y) else: a = x
a += rnd(0,2); h += rnd(0,3)
while h > 8 or a > 8: h--; a--
if h > 4 and rnd(0,1): h--
if a > 4 and rnd(0,1): a--
if (h > 2 or a > 2) and rnd(0,1): h--; a--
h = max(h,0); a = max(a,0)
if sH == 0: h = 0; if sA > 20: a = 8
if sA == 0: a = 0; if sH > 20: h = 8
h = ((to-from)*h)/45; a = ((to-from)*a)/45     // regular halves: 44/45 -> 1 planned chance = 0!
if from > 90: h++; a++                        // extra time: 14/45 + 1
F:0x5788 (long) += h + a                      // statistics
if schedule: f_0310_12fe(lg, idx, h, from, to); f_0310_12fe(lg, idx+1, a, from, to)
             for each human manager k of home/away: F:0x21d8+(k+1)*2 = (h, a)
else:        F:0x6dbc = h; F:0x6dbd = a      // used by re-planning (§10.5)
```
`pick(x,y)` = `f_0310_12d3(x,y)`: `rnd(0,1)==0 ? y : max(x,y)` (exactly: returns x if the
coin is 1 and x>y, else y).

### 5.2 Minute assignment – `f_0310_12fe(lg, slot, cnt, from, to)`
Clear the 8 minute bytes of the side; `cnt = min(cnt,8)`; for i in 0..cnt-1: up to 100
tries of `mm = rnd(from,to)`; for league tables (lg<10) a try is rejected if an earlier
slot j<i already has minute mm; after 100 failed tries the slot stays 0 (lost chance).
Cups (lg>=10): the collision test is broken (it zeroes slot i instead) so duplicates are
accepted on the first try; two chances in the same minute are both played.

### 5.3 Per-minute order inside a period – `f_0310_2303(lgMask, cupMask, ., from, to, show)`
```
reset F:0x5710/0x5778 (period chance counters)
league: f_0310_152e(30) (forfeits), then pre-plan ALL 3 leagues (all pairs up
        to 'F') with f_0f9d_08e9(...,1), whatever lgMask says: a league that does
        not play is planned (draws) but not played (lgMask==0 -> only league 0 =
        Nachholspiele)
cups:   pre-plan pairs of every c with bit c in cupMask: f_0f9d_08e9(c+10, idx, ..., 1)
for minute = from .. to:
   (UI, key ' ' -> options menu f_262f_01b8)
   for lg in leagues of lgMask (cups: DFB -> c=0 only; EC -> c=1,2,3 regardless of mask):
     for each pair idx (until 'F'):
        r = f_0310_2ee5(lg, idx, home, away, isCup)     // human events §10
        if r: f_0310_347f(..., minute, from, to, r-1)   // re-plan chances for that manager
        if mouse clicked and minute != to: f_0310_3314(...)   // substitution screen + re-plan
        for j in 0..7:
           if homeSlot[j] == minute: f_0f9d_0c3b(lg, idx, home, away, 0, minute, 0)
           if awaySlot[j] == minute: f_0f9d_0c3b(lg, idx, home, away, 1, minute, 0)
           (cup slots are cleared to 0 after use)
end: f_0310_152e(2) (forfeit results); if league and to==90:
     for lg in mask: f_2a41_2d33(lg,0,1) (table, §13.1); f_14a4_1662(lg) (AI scorers, §13.3)
     result screens f_2a41_120a if options D:0x5fe[k*3] (half) / D:0x5ff[k*3] (full)
```
Human events of a minute are generated **before** the chances of that minute.

### 5.4 Chance -> goal – `f_0f9d_0c3b(lg, idx, home, away, side, minute, .)`
```
F:0x6db0 (long)++                                   // chances statistic
dA = (S(away,2,minute) - S(home,0,minute)) / 3      // away attack vs home defence
dH = (S(home,2,minute) - S(away,0,minute)) / 3
(hs, as) = current score (league R / cup F:0x4c7e)
if minute > 90 and hs < 10: return 0                // extra time only for marked matches
own = (side==0) ? hs : as; dd = (side==0) ? dH : dA
if own == 9 or own >= 19: goal = 0                  // max 9 goals (19 in ET encoding)
else:
   pass = (side==0) ? (rnd(0,4) >= 2) : (rnd(0,1) != 0)    // 3/5 home, 1/2 away
   if not pass: goal = 0
   elif dd > 0: goal = rnd(0, (dd>>2) + 2) != 0
   else:        k = (-dd) >> 2;  goal = (rnd(0,k) == 0) or (rnd(0,6) == 0)
                // second rnd only drawn when the first is non-zero
if goal: F:0x5496 (long)++; score[side]++  (R or cup table)
if match is a watched human match (in F:0x3052 list):
     f_1a58_0ca3(goal, lg, idx, home, away, side, minute, comp, side, 0)   // §11
     comp = 0 league (lg<10), 1 DFB (lg==10), 2 EC (lg>10)
elif goal and F:0x2e99 and (home or away has TEAM+0x21 bit7):
     ticker line "Home - Away  h:a" (scores mod 10)
return goal
```
Resulting probability per chance: home `0.6 * (1 - 1/((dH>>2)+3))` if dH>0, else
`0.6 * (1/(k+1) + k/(k+1)/7)`; away the same with 0.5.

AI-vs-AI matches consist only of §5.1–5.4: no cards, injuries or substitutions.

---------------------------------------------------------------------------------
## 6. Pre-match setup – `f_1a58_20b2(home, away, comp, lg, idx, lgMask)`
Returns 0, or 1/2 when the home/away human team forfeits. `split = 2` if comp!=0 and
lg==0 (DFB-Pokal: gate split between both clubs), else 1. If comp==1 and DFB round
`F:0x8 > 4` (final): attendance A = `D:0x2256` (76000) and each human finalist gets
+532000 DM (0x81e20) instead of gate money (`fixed`).
For each human manager m whose league bit is in lgMask (or comp != 0 / lgMask==0):

### 6.1 Forfeit (`f_1a58_2051`)
If m's team is home (resp. away) and `MGR+0x237f == 100` (fewer than 8 lineup players at
the last strength computation) and NOT (`F:0x2e8e == 5` and day `F:0x2eba >= 10`)
[calendar month 5 = June, i.e. Relegation games are exempt – UNCERTAIN], then message
"Ihr Spiel wird mit 0:2 gewertet. Sie zahlen 200.000 DM Strafe", cash -= 200000, return
1 (home) / 2 (away). The match is not simulated; result becomes 0:2 / 2:0.

### 6.2 Attendance & gate money (home human)
`F:0x1d14[m] = 1`; importance `imp` = 1 (league or DFB), 2 (EC, comp>1), 3 (EC and
`F:0x9 > 4`). A = `f_0f9d_11e0(m, home, away, imp, split, 0, 0)` (§7) unless fixed.
* league (comp==0): `MGR+0x238c[MGR+0x237c] = A/1000`; season sum += A; update max/min
  (with opponent); `MGR+0x237c++`.
* cup and opponent from a higher bracket (bracket(away) < MGR+0x237a, not fixed):
  `A += rnd(A / (8 - 3*(league - bracket)), A)`, capped by capacity (0x23a0+0x23a8).
* money: fixed -> +532000; else `cash += A * MGR+0x234c / split` (long).
Away human in the DFB-Pokal (split 2): if the home team was not human (A==0):
`price = D:0x5390[bracket(home)] + rnd(0,1)`, `A = f_0f9d_11e0(m, home, away, 1, 2, 1, price)`,
`cash += A*price/2`; else `cash += A * homeMgr.ticketPrice / 2`. (fixed -> +532000.)

### 6.3 Squad updates before kick-off (for every human manager m whose team plays in
this match; the watched-match registration below happens only once per match, for the
first human found by `f_0310_32b1(home,away)`)
```
D:0x664[m] = 11
for loaned players (manager-4 slots 100..111 whose PERSON.owner == m):
    if status&1 and +0x0d and comp==0: +0x0d--              // suspension served
for s < count(m,0):
    if status&1 and +0x0d and comp==0: +0x0d--             // suspensions only count in league games
    +0x15 = 0
    if in lineup (1..11):
        k = (PERSON.skill0(0x57f9) > PERSON.skill1(0x57fa)) ? 1 : 0
        +0x13 += rnd(2,4) - k + 6                              // +7..+10
        if comp==0: PERSON+0x23 (league games)++
        ci = (comp>=1) + (comp>1); slot+0x06+ci ++; word slot+0x1c+2ci ++
    clamp +0x13 to [50,150]
if count(m,0) < 6 or comp != 0: yellowBudget=0, redPossible=0   // NO cards in cup matches
else: yellowBudget = 6, redPossible = 1
keeperSubs = 1; fieldSubs = 2; booked list = 0x7f*6; F:0x2e99 = 1
register watched match (F:0x305c/0x28/0x4bf0)
report record (§1.6) = fresh copy; home human: +0x96 = A, +0x12 = (capacity == season sum),
   crowd flag: x = clamp((cap23a8 - cap23a0)/1000, 0, 30); if rnd(0,(30-x)*4+15)==0: MGR+0x2380 |= 1
away human: +0x96 = A + 200000 if split==2 (UNCERTAIN purpose)
+0xe = own team, +0xf = opponent, +0x7 = opponent table position
```

---------------------------------------------------------------------------------
## 7. Attendance – `f_0f9d_11e0(m, home, away, imp, split, cupFlag, priceOverride)`
(floating point emulator code; decoded with an FPU-aware disassembler)
```
posH = home<64 ? EXT[home].pos(0xefa) : 5;  posA = away<64 ? EXT[away].pos : 5   (0-based ranks)
sH = (S(home,1,0)+S(home,0,0)+S(home,2,0))*2/15;  sA likewise
P  = priceOverride ? priceOverride : MGR+0x234c
L  = MGR+0x237a;  lf = 1 << (2-L)        // 4,2,1
if lf == 4 and posH > 14: posH = 14
if sH == 0 and sA == 0: return 1
X = ((1-(posA-1)*0.02)*sA + (1-(posH-1)*0.02)*sH) * (10/43) * 1150     (double)
r = rnd((P-10)*1050, P*1550 - 14400)      // 16-bit int products (overflow wraps!)
A = (long)(X - r + 1000)                  // C float->long truncation
Q = (sH+sA)*100/180
A += rnd(5Q, 9Q) * L30 - 20Q;   A += rnd(4Q,12Q) * L28 - 31Q
A += (L18*10 - 8)*Q;  A += (L20*10 - 5)*Q;  A += (L10/1000 - 8)*Q
     (Lxx = long at MGR+0x23a0+xx; meanings belong to the stadium/club module)
if word MGR+0x241e > 50: A += (30*w - 1500) (16-bit unsigned)
A += -2000*L
if G != 0: A += 1200
if L00 + 2000 > L08: A += rnd(400,700)
md = D:0x225a[L]; nmd = D:0x2262[L]
CAP = cupFlag ? rnd(T - T/3, T + T/3)*10 (T = low word of D:0x2c8[bracket(home)]) : L00 + L08
imp = min(imp,3); if L == 2: imp = 0; if sA < 90 and imp > 1: imp--
if P < 22: A = A*(imp+2)/3
if md+5 > nmd and P < 22:                 // last 4 match days
    if md == nmd: A = CAP                 // last match day: sold out
    else: k = (L != 0); if 7-2k > posH: A += CAP/4; if posA < 7: A += CAP/5  (only inside previous if)
          if posH + k + 5 > D:0x225e[L]: A += CAP/4       // relegation battle
if P > 10: A += (10-P)*900
limit = 100000 * (2*lf - (lf==4))         // 700000 / 400000 / 200000
while A*P > limit: A -= 1000
while A <= 500: A += 500
return min(A, CAP)
```
(The exact nesting of the "+CAP/5" test: it is evaluated only when `7-2k > posH`.)

---------------------------------------------------------------------------------
## 8. (see §5.3) Periods and match flow
* Regular: periods (1,45) and (46,90). Strength decay uses `from` for planning and the
  current minute for goal decisions.
* Display parameter `show` = F:0x2e99 (some human match in this competition today).
  Without human matches nothing is displayed and no delays happen.

---------------------------------------------------------------------------------
## 9. Watched match handling
Each human match is put into the list `F:0x3052/0x305c/0x0028` (max 4). Only chances
in these matches produce event screens (`f_1a58_0ca3`); other matches with a ticker team
produce one-line ticker messages.

---------------------------------------------------------------------------------
## 10. Human-team events per minute – `f_0310_2ee5(lg, idx, home, away, isCup)`
Only if `F:0x2e99`. For each human m with `F:0x1d14[m]` whose team is home or away:
```
if MGR(F:0x304a)+0x237f == 100: return 0     // BUG: F:0x304a is 0x1e during matches, so this
                                             // reads F:0x7eeb (a squad byte) – treat as "never"
if count(m,1) < 4: return 0                  // fewer than 4 on the pitch: no events
H = clamp(40 - MGR+0x237f, 0, 40)            // high Intensität -> small H -> more events
// red card
if redPossible[m] and rnd(0, (G+5)*50 + 70*H) == 0:
     redPossible[m] = 0; p = f_0310_1d45(m); ret = m+1
     f_1a58_0ca3(2, lg, idx, home, away, ?, p, m, team==away, 0)      // "Rote Karte"
// yellow card
if yellowBudget[m] and rnd(0, 5G + 2H + 37) == 0:
     yellowBudget[m]--
     repeat p = f_0310_1d45(m); booked = p in bookedList[m]
     until redPossible[m] != 0 or not booked      // after a red, 2nd yellows are avoided
     if booked: redPossible[m] = 0; type 2 with p10=1 ("Gelb-Rote Karte")
     else: add p to bookedList; type 3 ("Gelbe Karte")
// injury
if rnd(0, 55H + 60G + 200) == 0:
     p = f_0310_1d45(m); ret = m+1; f_1a58_0ca3(4, ..., p, m, team==away, 0)   // "Verletzt"
return ret (0 or m+1 of the last manager with red/injury; yellow-red does not set it)
```
Per-minute probabilities e.g. G=2, Intensität 16 (H≈24): yellow 1/96, red 1/2031,
injury 1/1641. AI teams never get cards or injuries.

### 10.1 Victim choice – `f_0310_1d45(m)`
repeat `s = rnd(0, count(m,0)-1)` until slot is on the pitch, `status&3 == 0` and
(`d = T - K`; d>0: accept if `d > rnd(0,3)`; else accept if `rnd(0,10)==0`).

### 10.2 Red / yellow-red effects (`f_1a58_0ca3` type 2)
red: slot+0 ++; yellow-red: slot+2 ++, slot+1 --. `+0x0a = 0` (off the pitch),
`status |= 1`, `+0x0d = rnd(1, 7 - 6*p10)` (red 1..7 games, yellow-red 1 game),
`+0x15 -= 10`, report +0x93 = player+1, `D:0x664[m]--`, `f_0f9d_0002(m,1)` (team weaker).
### 10.3 Yellow card (type 3)
`+0x15 -= 10`, `slot+1 ++`, text "Gelbe Karte: NAME (n.)", report booking list.
No accumulation suspension was found in the match code (OPEN QUESTION; `f_2e3a_23aa`
converts `+0x0d == 100` into a 1-game suspension but no writer of 100 was found).
### 10.4 Injury (`f_14a4_30cf`)
```
if PERSON.skill2(0x57fb) > 20: skill2 -= rnd(12,19)
k = rnd(0,17); if rnd(0,1) and rnd(2,4) < DUR[k]: k = rnd(0,17)   // DUR = D:0x5bc
slot+0x0d = DUR[k] + rnd(0, DUR[k]/4 + 1); status |= 2; slot+0x17 = k; slot+0x0a = 0
text "Verletzt: NAME (injury name)"
```
Duration unit = weeks: `f_0cb5_2b88` decrements +0x0d of injured players when
`(day - D:0x224c) % 7 == 0` and clears status bits when it reaches 0.
After the injury screen the substitution screen `f_1ecd_1560` is opened (`f_1a58_1f75`).

### 10.5 Re-planning after red card / injury / user substitution – `f_0310_347f`
```
f_0f9d_08e9(lg, idx, home, away, from, to, 0)       // new totals for the whole period -> N[0..1]
for side i in 0,1:
    P = planned[m][i]                                // F:0x21da+m*2+i
    if P <= N[i]: N[i] -= P
    P -= (i==0 ? F:0x5710[m] : F:0x5778[m])          // chances already played
    P += N[i]; if P < 0: P = 0
    f_0310_12fe(lg, idx+i, P, currentMinute, to)     // reschedule the rest of the period
```
(Literal; it can increase the remaining chances.) `f_0310_3314` is the mouse-click hook:
click on a team logo of a human match -> `f_1a58_1f75` (lineup/substitution screen
`f_1ecd_1560`; substitutions cost keeperSubs/fieldSubs, add `rnd(2,4)+4` to the incoming
player's +0x13, increment appearances and `F:0x90c6[m]`) and then `f_0310_347f`.

---------------------------------------------------------------------------------
## 11. Chance / goal event – `f_1a58_0ca3(type, lg, idx, home, away, side, minute, comp, side, pen)`
type 0 missed chance, 1 goal, 2 red (pen=1 yellow-red), 3 yellow, 4 injury; pen=1 for
shoot-out kicks ("Elfmeter für" instead of "Chance für"). For type 0/1:
1. Count chances: per human of home/away -> F:0x5140+m*6 [4]/[5], F:0x5710/0x5778.
2. Text "Chance für TEAM !" (or 3D scene `f_14a4_05ec` when option D:0x607).
3. For the attacking team if human (manager h):
   `sc = f_0310_2c9a(h, 0, type)`; if count(h,1) > 1: repeat `as = f_0310_2c9a(h,1,1)` until `as != sc`.
   Report record: goal/miss with minute, scorer, assister; for every human whose team
   is the defender: report entry with scorer 'd' (100).
   * goal (type 1) and pen==0: if comp==0: PERSON(sc)+0x22 ++ (league goals);
     word slot+0x22+2comp ++; slot+0x03+comp ++; `+0x15 += 15`;
     text "Torschütze: NAME (total)" where total = +3 + +4 + +5.
   * missed: "Chance vergeben: NAME", shooter `+0x15 -= 10`.
   * "nach Vorlage von NAME" shown if sc != 0xFE, scene returned 0 and pen==0;
     the assister gets `+0x15 += 10` (also for missed chances).
   * `sc == 0xFE` ("Eigentor !") only when the team has no lineup players (degenerate).
   * Shoot-out kicks (pen=1) are not counted in player statistics.
4. If the attacking team is not human but the defending team is: for every player on
   the pitch: goal against: keeper (poscode==0) -8, others -5; saved chance: keeper +8,
   others -3 (`+0x15`).

### 11.1 Scorer / assister choice – `f_0310_2c9a(m, kind, type)` (kind 0 scorer, 1 assist)
```
if count(m,1) == 0: return 0xFE
loop:
  s = rnd(0, count(m,0)-1); require 1<=+0x0a<=11
  if kind != 0 or type != 0: W = (3T)/(5-3kind) + K/5 + P/5
  else:                      W = rnd(120,170)                 // shooter of a missed chance
  W += (poscode/25) * (6-kind) * 7
  W += 46 * slot+0x03                                          // league goals so far
  W += (kind==0)*300 + (7 - |slot+0x1a|)*200
  if slot+0x1a < 2: W += 1200
  W += (|slot+0x19 - 3| - 2*(kind==0)) * 5
  if kind != 0 or type != 0: W += (5 - rm)*5 + (35 - pm)       // rm/pm see §3
  else:                      W += rm*5 + pm
  ok = rnd(0,3500) < W
  if poscode == 0 and rnd(0,20) != 0: ok = false               // goalkeepers rarely
until ok
```

---------------------------------------------------------------------------------
## 12. Cups: extra time, penalties, aggregate, advancement

### 12.1 Draw detection – `f_17c2_1226(cupMask, n, shootout)`
Clears `F:0x1d14[*]`. For each c in mask and pair idx<n:
* DFB (mask==1): tie if `H == A` or `H - A == 10`.
* EC (other masks): only if second leg (`F:0x1d17[c] != 0`) and `f_17c2_15e8(c,idx) == 30`.
On tie: `H += 10`; if a human plays: `F:0x1d14[m] = 1`, `F:0x2e99 = 1`; if shootout:
`f_0310_356d(c, idx, mask, human)`. Returns whether any tie was found.
First call after 90 min (shootout=0) -> periods 91-105 and 106-120; second call
(shootout=1) after 120 min handles still-tied matches.

### 12.2 Shoot-out – `f_0310_356d(c, idx, mask)`
* No human: repeat `H=rnd(2,5); A=rnd(2,5)` until H != A; `H += 20`.
* Human: screen "Elfmeterschiessen"; H=A=0; `first = rnd(0,1)` (drawn first);
  `side = first; rounds = 0`; while `rounds < 5 or H == A or side != first`:
  `goal = min(rnd(0,2),1)` (2/3); `f_1a58_0ca3(goal, c+10, idx, home, away, side, 120, comp, side, 1)`;
  score[side] += goal; side ^= 1; if side == first: rounds++.
  Always 5 full rounds, then sudden death per round. Finally `H += 20`, "Endergebnis ... n.E.".

### 12.3 Two-leg evaluation – `f_17c2_15e8(c, idx)` (second leg)
```
h = H mod 10 (current home), a = A, f1 = F:0x21c2[...] (first-leg goals of current home team),
f2 = F:0x21c3[...] (first-leg goals of current away team)
if day >= 316 (D:0x7dc long) and f1 + h == f2 + a: return 30        // late (final): no away-goals rule
if f1 + h < f2 + a: return 1                                         // away team advances
if f1 + h == f2 + a and f1 < a: return 1                             // away goals rule
if f1 + h == f2 + a and f1 == a: return 30                           // undecided -> ET / penalties
return 0                                                              // home team advances
```
QUIRK: after a shoot-out H/A hold the penalty score, so 15e8 then compares
`f1 + penH` vs `f2 + penA` (can contradict the shoot-out unless f1 == f2... ).

### 12.4 Advancement – `f_17c2_16dc(cupMask-1)`
* DFB: for each pair: H mod 10 (strip ET/penalty marks); `H < A` -> away team wins;
  winner stored into the pair's home slot `F:0x76ca+idx`; human's `MGR+0x2374 = round+1`.
  Final (1 pair): `D:0x7ac = winner+1`, `D:0x7b0 = loser+1`, trophy `f_17c2_274d(m,1)`.
* EC: first leg (`F:0x1d17[c]==0`): flag=1, `0x21c2 = A`, `0x21c3 = H`, swap home/away.
  Second leg: flag=0; `if f_17c2_15e8 != 0: swap` (so 1 and 30 both make the current
  away team the winner – with a shoot-out the result decides via the quirk above);
  winner = home slot; final: `D:0x7ad+c-1 = winner+1`, trophy `f_17c2_274d(m, c+1)`.
  Then the next draw (`f_17c2_1b6d`, `f_17c2_13a2`) – other topic.

---------------------------------------------------------------------------------
## 13. After the match

### 13.1 League table – `f_2a41_2d33(lg, mode, sort)` (EXT record `F:0x0ecc + t*0x36`)
EXT layout: +0/+1 points home/away (+2/+3 backup), +4 form string (8 chars + NUL;
backup at +0x0d), +0x16/+0x17 goals for home/away (+0x18/+0x19 backup), +0x1a/+0x1b
goals against home/away (+0x1c/+0x1d backup), +0x1e/+0x1f games home/away,
+0x26/+0x27 wins home/away, +0x2a/+0x2b losses home/away, +0x2e rank (0-based).
* mode 1 (before the matches): `f_2a41_1ef4(t,0,2)` backup (points, GF, GA, home wins,
  home losses, form string) for all teams of non-postponed pairs.
* mode 0 (after 90 min): restore backup (`f_2a41_1ef4(t,2,0)`), then for each played pair
  (h:a goals): GF/GA/games; `f_2a41_1fec` form for both (§4.2); record/streak tables
  (`F:0x9d36 + t*0x15`, best values per human at `F:0xa276`, `f_2a41_37cf` records);
  win: winner +`D:0x2271` (=2) points, wins+1, loser losses+1; draw: +1 each; form chars
  appended (`f_2a41_1fb0` shifts the 8-char string left): home win 's'/away 'N', draw
  'u'/'U', away win 'n'/'S'; human managers' history `f_2a41_2073`.
* sort (exchange sort over the league's team list `F:0x535a+lg*0x14`, sort 0 home table,
  1 total, 2 away table): points desc; then games played ASC (fewer games first; skipped
  if `D:0x224d`); then goal difference desc; then goals scored desc. Writes EXT+0x2e rank
  and `MGR+0x234d+sort`.

### 13.2 Cup results – see §12.4. No table, no form change.

### 13.3 AI goal scorers – `f_14a4_1662(lg)` -> `f_14a4_14d4(team, goals)` (teams without human manager)
```
cand = []
for p in 1..150 with PERSON[p].team == team:
    if rnd(0,100) < 94: PERSON[p].games++
        if rnd(10,99) < rnd(5,25) + PERSON[p].poscode: cand += p
if len(cand) > 1:
  for g in 1..goals:
    L = (len<10) ? D:0x5331[len]-1 : 10; if L > 2: L--
    if rnd(0,4) != 0:
       do: if rnd(0,L) != 0: PERSON[cand[rnd(0,len-1)]].goals++ ; break
       while rnd(0,1) + 2 < len
```
(Only the 150 PERSON entries are tracked; many AI goals are not attributed.)

### 13.4 Player grades / newspaper – `f_2e3a_23aa(cupMask)`
For each human m that played and each squad slot: if `+0x0d == 100`: `+0x0d = 1`,
status |= 1. For lineup players:
```
r = clamp((signed)+0x15, -12, 36)
grade = ( ((K+T)/(-33) - |30 - P|/6 + 12)*2 + ((-(r+12))/6 + 6)*4 + E/33 + pm/5 + rm ) / 11
grade = clamp(grade, 1, 6)        // stored in F:0x4d3c + (m*0x20+s)*8 (reused buffer)
```
League only (cupMask==0): newspaper report `f_2e3a_0ea3(m,0)`.

### 13.5 Other persistent updates summary
| what | where |
|---|---|
| appearances (+6.., words +0x1c..), PERSON league games | §6.3 (before kick-off) |
| E (+0x13) +7..+10, clamp 50..150 | §6.3 |
| goals (slot +3.., words +0x22.., PERSON +0x22) | §11 |
| cards, suspensions (+0x0d games, only league matches count down, §6.3) | §10.2 |
| injuries (+0x0d weeks, kind +0x17, PERSON skill2 −12..19) | §10.4 |
| team table, form F_t | §13.1, §4.2 |
| AI cur/base/rnd | daily §4.1 (not match-dependent) |
| gate money, attendance stats, report | §6.2, §7 |
Recovery of E, form/tendency (+0x0e) and training effects happen in the daily manager
update `f_0cb5_13bd` (not part of the match engine; also creates training injuries
"Verletzung im Training" with `rnd(0, 30G + max(40, 2*(160 + 25*(noMatchToday) - E)) + 80) == 0`).

---------------------------------------------------------------------------------
## 14. Postponements and Nachholspiele
* `f_0310_0463(day, lg)` for date slots 25 < day < 70 (winter): `k = |day-44|`,
  `n = 9 - k/3`, if n != 0 and rnd(0,1): `n = 8 - k/3`; `n = rnd(0,n)` (max 8 per pass).
  Repeatedly picks a free Nachhol entry (20 entries of 5 bytes at `F:0x5715`:
  day, matchday, league, pairIdx) and a random pair `rnd(0, teams/2-1)` not yet postponed,
  and calls `f_0310_05f1`: next date slot d' > day (max 89, slots 36..58 skipped) whose
  flags are 0 or 0x80 and where no other postponed match on d' shares a team; on success
  `R = 30` (postponed) and `D:0x2280[d'] = 0x80`.
  (The local "already chosen" list is clobbered by constants 120,54,107,23 – harmless.)
* `f_0310_07a2(day)`: collects all entries for the day into league 0's pair list
  (terminated by 'F'), plays them with `f_0310_15db(0,0)` (results in league-0 row of the
  current match day), copies each result back to its original `R(lg,md,idx)`, clears the
  entry, zeroes 19 bytes of league-0's current row (QUIRK: overwrites the last
  displayed league-0 results) and clears the date flag. The table update for these
  matches is done by `f_2a41_2d33(0,0,1)` (team-id based, so it works for all leagues;
  only league 0 gets re-sorted).

## 15. Relegation – `f_0310_02bf`
Uses EC slot c=1 (mask 2) temporarily (saves/restores `F:0x76ea`, `0x4c9e`, `0x21e2`),
sets `F:0x9 = 5`, `F:0x57dc = 1`. First leg (`F:0x1d18 == 0`): home = Liga2 3rd
(`F:0x5370`), away = Liga1 16th (`F:0x5369`); second leg swapped with first-leg score
restored from `F:0x2e6e`. Played as a two-leg cup tie (§12; day >= 316 so no away-goals
rule: a tie goes to extra time/penalties). After the 2nd leg `F:0x3060 = f_17c2_15e8(1,0)`.

## 16. Report text templates (brief) – `f_2e3a_0964(m, src, dst)`
Copies text until '#' or NUL. `%` escapes refer to the report record (§1.6):
`%0` goals for, `%1` goals against, `%2..%5` record bytes 0..3, `%9` attendance (long),
`%a` own team name, `%b` opponent name, `%c`/`%d` player name of slot (+0xc/+0xd, 1-based),
`%t` manager name, `%e` end. `%xDN alt1#alt2#...#%`: choose alternative k, with
`k = rnd(1,N)` if D=='0', else `k = F:0x579b[D]` (selector set by the report logic);
alternatives may nest. Example "%x02KLAGEN#SCHWEREN#%". Line breaking: `f_2e3a_06df`.

## 17. RNG order notes (for bit-exact reproduction)
Per period: planning for all leagues/pairs (league order 0..2, pair order) with the
draw sequence of §5.1 (incl. the rnd inside pick only when called), then minute loop:
per minute, per league, per pair: 2ee5 draws (red test, yellow test, injury test, in this
order, only for human teams), then chance slots j=0..7 home then away. Strength
recomputation `f_0f9d_0002` consumes rnd only in full mode (X bonuses, keeper, line
balance). Display/animation code also calls rand() (scenes, ticker) – exact
reproduction with display enabled is not realistic.

## 18. Function list
| function | meaning |
|---|---|
| f_1a58_3176 | game loop; calls match days |
| f_0310_0995 / 0261 / 0290 / 02bf / 07a2 | Ligaspiel / DFB-Pokal / Europapokal / Relegation / Nachholspiele |
| f_0310_01f2, f_0310_00fa | title screen of a match day |
| f_0310_15db | match-day driver (setup, periods, ET, shoot-out, post) |
| f_1a58_20b2 | per-match pre-processing for humans (forfeit, gate, squad) |
| f_1a58_2051 | forfeit 0:2 + 200000 DM |
| f_0f9d_11e0 | attendance |
| f_0310_2303 | one period, minute loop |
| f_0f9d_08e9 | number of chances per side for a period |
| f_0310_12d3 | pick(x,y) helper |
| f_0310_12fe | assign chance minutes |
| f_0f9d_0c3b | resolve a chance (goal?) |
| f_0310_1468 | team strength S(t,kind,minute) |
| f_0f9d_0002 | lineup -> team strengths, discipline value |
| f_0f9d_0697 | AI strength random walk |
| f_2a41_1fec | form change after league match |
| f_0310_2ee5 | human cards/injuries per minute |
| f_0310_1d45 | pick victim |
| f_0310_1bd7 / 1cd3 | role / position mismatch of a player |
| f_0310_2c9a | pick scorer / assister |
| f_1a58_0ca3 | event screen + statistics (chance, goal, cards, injury) |
| f_14a4_30cf | injury kind and duration |
| f_0310_347f, f_0310_3314, f_1a58_1f75 | re-planning, click hook, substitution screen |
| f_0310_152e | forfeit result placeholder handling |
| f_0310_322c, f_0310_1def, f_0310_2086, f_1a58_042f, f_1a58_009d | display |
| f_17c2_1226 | cup tie detection (ET / shoot-out trigger) |
| f_0310_356d | penalty shoot-out |
| f_17c2_15e8 | two-leg aggregate |
| f_17c2_16dc | cup winners advance |
| f_17c2_1ccb, f_2a41_120a, f_2a41_214c | result / table screens |
| f_2a41_2d33, f_2a41_1ef4, f_2a41_1fb0 | league table update / backup / form string |
| f_14a4_1662, f_14a4_14d4 | AI goal scorer statistics |
| f_2e3a_223e | report record events |
| f_2e3a_23aa | player grades, newspaper trigger |
| f_2e3a_0964 | report template expansion |
| f_0310_0463, f_0310_05f1, f_0310_000a | postponements |
| f_0cb5_2b88 | weekly injury countdown |
| f_3091_1109(m,mode) | count slots: 0 with person, 1 on pitch, 2 bench (12 slots for m=4) |
| f_3091_1009(p,lo,hi) | clamp; f_3091_10d5 = abs difference |
| f_0310_32b1(a,b) | manager+1 of team a (or +0x81 if team b), 0 if none |

## 19. Open questions / uncertainties
* Semantics of slot fields +0x10/+0x11/+0x12/+0x13 (K,T,P,E) taken from CONTEXT; the
  engine uses +0x11 as decaying "base", +0x10 as "cur", +0x12 as "rnd" part, and the
  grade formula treats +0x12 as ideal at 30 – naming UNCERTAIN.
* Who sets slot+0x0d = 100 (pending 1-game suspension) – not found; yellow-card
  accumulation rule unknown (maybe absent).
* Forfeit exemption "month 5, day >= 10" interpretation (June / relegation) UNCERTAIN.
* `f_0310_2ee5` early-exit reads `MGR(F:0x304a)` with F:0x304a = 0x1e (garbage byte
  F:0x7eeb) – presumably intended for the current manager; harmless in practice.
* Attendance inputs `MGR+0x23b0..0x23d0`, `0x241e` meaning (stadium/club module).
* Away-human DFB report attendance +200000 (encoding flag?).
* 3D goal scene `f_14a4_05ec` return value suppresses the "nach Vorlage" line; not
  analysed (no influence on results).
