# Spec: Stadium, Attendance, Training Camps, Youth (BMMAIN.EXE)

Notation: `F:x` = far data segment 0x4238, `D:x` = DGROUP 0x4cb3. `M = F:0x304a*0x30a` = offset of the
current human manager's record (all stadium data is **per human manager**, AI clubs have no stadium
record). `rand(a,b)` = `f_076b_0cc7(a,b)` = `a + rand() % (b-a+1)` (C semantics, 16-bit int args).
"long" = 32-bit signed, `/` = C truncating division. Function names are `f_SSSS_OOOO`.

Currency: every displayed DM value is `x * D:0x2252(long) / D:0x224e(long)` with optional prefix string
D:0x4f78 when byte `D:0x224d != 0`. Defaults in the image: 0x224d=0, 0x224e=1, 0x2252=1 (identity).
Internally all money is DM. Ignore the conversion for game logic.

---------------------------------------------------------------------------------------------------
## 1. Data layout

### 1.1 Stadium block (in the manager record, persisted with F:0x2242..0x2e6a in the savegame)

Seven construction items `k = 0..6` (dialog item number `p = k+1`), each 8 bytes:

| offset (F:M+) | type | item k | meaning |
|---|---|---|---|
| 0x23a0 / 0x23a4 | long cur / long pending | 0 | Sitzplätze (seats) |
| 0x23a8 / 0x23ac | long / long | 1 | Stehplätze (standing) |
| 0x23b0 / 0x23b4 | long / long | 2 | Überdachte Plätze (covered places) |
| 0x23b8 / 0x23bc | long / long | 3 | Flutlicht level 0..3 |
| 0x23c0 / 0x23c4 | long / long | 4 | Anzeigetafel level 0..3 |
| 0x23c8 / 0x23cc | long / long | 5 | Zustand (condition) grade 1..6 |
| 0x23d0 / 0x23d4 | long / long | 6 | Komfort grade 1..6 |
| 0x23d8 + 2k | uint16 | 0..6 | remaining construction time of item k in **days** (0 = idle) |

`pending` is the amount that will be added to `cur` when the timer reaches 0 (for level items it is the
level delta).

Other stadium/attendance fields in the manager record:

| F:M+ | type | meaning |
|---|---|---|
| 0x234c | byte | Eintrittspreis (ticket price, DM), range 5..25 |
| 0x234d + i | byte | league table position (0-based) after matchday i (history, not stadium) |
| 0x237a | byte | manager's league L (0 = 1.Liga, 1 = 2.Liga, 2 = Amateur-Oberliga) |
| 0x237b | byte | training-camp cooldown (days), see §6 |
| 0x237c | byte | number of league home games played this season |
| 0x2380 | byte | bit0 = "riot pending" flag (set after a home game, processed in the next turn) |
| 0x2381 | byte | youth-department investment level 0..34 (see §7) |
| 0x238c + i | byte | attendance/1000 of league home game i (history for the statistics graph) |
| 0x241e | uint16 | fan popularity ("Beliebtheit", name UNCERTAIN), see §3.4 |
| 0x2424 | uint16 | youth points accumulator (see §7) |
| 0x2426 | long | sum of league home attendances this season ("BESUCHERZAHL") |
| 0x242a | long | season attendance record ("ZUSCHAUERREKORD"), 0x2436 (word) = opponent team id |
| 0x242e | long | season minimum ("MINUSKULISSE"), init 99999, 0x243a (word) = opponent team id |
| 0x2432 | long | bank balance (Kontostand) |

Global stadium-related state:

| addr | type | meaning |
|---|---|---|
| F:0x577f + p (p=1..7) = F:0x5780 + k | byte | "no construction company" lockout per item, days; global (not per manager!), saved (F:0x5780, 7 bytes) |
| D:0x620 + 2i (i=0..7) | int16 | training camp open/closed timer (saved, 16 bytes) |
| D:0x5258 + 4i | long | training camp price (NOT saved) |

### 1.2 Grade / size texts

* Level items 3,4 (Flutlicht, Anzeigetafel): value 0..3 -> `D:0x2858+4v`: 0 "IS NICH", 1 "KLEIN", 2 "MITTEL", 3 "GROß".
* Grade items 5,6 (Zustand, Komfort): value 1..6 -> `D:0x283c+4v`: 1 "UNGENUEGEND", 2 "MANGELHAFT",
  3 "AUSREICHEND", 4 "BEFRIEDIGEND", 5 "GUT", 6 "SEHR GUT" (0 = empty/NULL).
* Row labels (`D:0x2868+4r`, r=0..8): GESAMTKAPAZITÄT, SITZPLÄTZE, STEHPLÄTZE, ÜBERDACHTE PLÄTZE,
  FLUTLICHT, ANZEIGETAFEL, ZUSTAND, KOMFORT, EINTRITTSPREIS.
* Dialog titles (`D:0x2818+4p`, p=1..8): SITZPLÄTZE AUSBAUEN, STEHPLÄTZE AUSBAUEN, PLÄTZE ÜBERDACHEN,
  FLUTLICHTMASTEN AUSBAUEN, ANZEIGETAFEL ERRICHTEN, STADION RENOVIEREN, KOMFORT VERBESSERN,
  EINTRITTSPREIS ÄNDERN.

### 1.3 Constant tables (dumped from BMMAIN.bin)

```
D:0x44 + 4p  (long) cost per unit, p = 1..7:
   p1 Sitzplätze   150000 DM per 1000 seats      ("KOSTET x DM PRO 1000")
   p2 Stehplätze   100000 DM per 1000
   p3 Überdachen   160000 DM per 1000
   p4 Flutlicht    380000 DM per level           ("DM PRO PUNKT")
   p5 Anzeigetafel 440000 DM per level
   p6 Renovieren   550000 DM per grade
   p7 Komfort      650000 DM per grade
D:0x63 + p   (byte) base construction time in WEEKS, p = 1..7: 15, 15, 20, 10, 10, 8, 4
D:0x2c8 + 4j (long) generic capacity seed by league index j: 4500, 2200, 1200, 600
D:0x5390 + 2j (word) generic AI ticket price by league index j: 16, 14, 10, 8
D:0x2256     (long) DFB-Pokal final attendance: 76000
D:0x2272     team-id ranges per league index j: [0..17] [18..37] [38..57] [58..255]
D:0x225a[l]  current matchday of league l (runtime, init 1,1,1)
D:0x225e[l]  teams per league: 18, 20, 20
D:0x2262[l]  matchdays per league: 34, 38, 38
D:0x8e       strength weight table W[k][i] (3x3 bytes): k0 [80,20,0], k1 [25,50,25], k2 [0,20,80]
FP constants (D): 0x9b4a 1.0, 0x9b52 2.0, 0x9b5a 0.01, 0x9b62 0.23255813953488372 (=10/43),
                  0x9b6a 1150.0, 0x9b72 1000.0
```

---------------------------------------------------------------------------------------------------
## 2. Initial stadium at game start (f_08bc_2184, new game, per human manager)

Inputs: `L0` = league the manager starts in (0 normally; 2 = Amateur-Oberliga in the endless mode, see
below), `S` = "short game" flag (end-year option F:0x513c == 1964 or 1993), `E66` = end-year == 1966 or
1995. (F:0x513c = game end year: 22222 = "NIE" (never), 1964 or 1966; +29 for the 1992 start variant,
flag F:0x56f4. `L0 = 2` when the end year is > 1966 (> 1995 in the modern variant), i.e. only in the
endless game; otherwise L0 = 0 — but the team chosen may still decide, see f_08bc_2184.)

```
price   (0x234c) = (S + 8)*2 - 3*L0          -> 16 (1.Liga), 18 (short), 10 (Oberliga)
Steh    (0x23a8) = (5 - L0) * 4000           -> 20000 / 12000
Sitz    (0x23a0) = (2 - L0) * 3000           -> 6000 / 0
if S:   Sitz += 5000; Steh += 10000
Überd.  (0x23b0) = (2 - L0) * 3500           -> 7000 / 0
Zustand (0x23c8) = 4 + S                      (BEFRIEDIGEND / GUT)
Komfort (0x23d0) = 3 + S                      (AUSREICHEND / BEFRIEDIGEND)
if end-year < 2000 (all limited games):
    Anzeigetafel (0x23c0) = 3 - 2*E66        -> GROß (3) or KLEIN (1)
    Flutlicht    (0x23b8) = 3 - E66          -> GROß (3) or MITTEL (2)
else: both stay 0 (IS NICH) (record zero-initialised)
pending fields and timers = 0
popularity (0x241e) = 50 - 20*L0;  0x2373 = 16
money (0x2432) = 1,500,000 (1,900,000 if D:0x4a28 == 4); min-attendance 0x242e = 99999
```

---------------------------------------------------------------------------------------------------
## 3. Attendance (Zuschauer) and ticket income

### 3.1 Where it is computed — f_1a58_20b2(home, away, comp, leg, ?, leagueMask)

Called from the match processing f_0310_15db for every match. `comp` (param_3): 0 = league,
1 = DFB-Pokal, >=2 = European cups (UNCERTAIN naming; F:0x8 = DFB round, F:0x9 = Euro round).
`leg` (param_4) and `local_26 = gateDiv = 2 if comp != 0 && leg == 0 else 1`.
For each human manager m (0..D:0x7ab-1) taking part (league games only if the manager's league bit is
set in param_6):

1. `cupFinal = (comp == 1 && F:0x8 > 4)` -> attendance = D:0x2256 = 76000 and every human participant
   gets a flat **+532,000 DM** (0x81e20); no other income, no records.
2. If m's team is **home** (and not the special "fired/bankrupt" case 0x237f == 'd' -> f_1a58_2051):
   * `P4 = 1`; if `comp > 1` then `P4 = 2`, and if `F:0x9 > 4` then `P4 = 3`.
   * `att = Attendance(m, home, away, P4, P5=gateDiv, generic=0, priceOverride=0)` (§3.2).
   * League game (`comp == 0`): history byte `F:M+0x238c+count = att/1000`; `0x2426 += att`;
     if `att >= record` -> record = att, opponent = away; if `att <= minimum` -> minimum = att,
     opponent = away; `count (0x237c)++`.
   * Cup game (`comp != 0`): let `b` = league index (D:0x2272 ranges) of the **away** team; if
     `b < L` (opponent from a higher German league): `att += rand(att/(8 - 3*(L-b)), att)` (both
     arguments truncated to int16! — overflow possible above 32767), then `att = min(att, Sitz+Steh)`.
   * Income: `money += price * att / gateDiv` (long arithmetic, price = F:M+0x234c).
3. If m's team is **away** and `gateDiv == 2` (cup, first/only leg):
   * if no attendance has been computed yet in this call (home is AI):
     `priceAI = D:0x5390[j(home)] + rand(0,1)`;
     `att = Attendance(m, home, away, P4=1, P5=2, generic=1, priceOverride=priceAI)`;
     `money += priceAI * att / 2`.
   * else (home is also human): `money += homePrice * att / 2` using the home manager's price and the
     home attendance.
   League away games give no gate money.
4. Match-report bookkeeping (only for the human's displayed match, UNCERTAIN gating via
   f_0310_32b1): for a human home match, report flag "sold out" (report+0x0c) is set when
   `Sitz+Steh == F:M+0x2426` (compares with the season **sum** — effectively only true on the first
   home game; looks like an original bug), then the **riot roll** (§4.3).

### 3.2 Attendance formula — f_0f9d_11e0(m, h, a, P4, P5, generic, priceOverride)

(Ghidra could not decompile it — 8087-emulator code; decoded from the raw bytes.)
`St` = stadium of manager m (F:m*0x30a+0x23a0) — **always m's own stadium**, also when m plays away.

```
pos(t)  = t < 64 ? F:0x0efa + t*0x36 (league table position, 0-based) : 5
hp = pos(h); ap = pos(a)
str(t)  = (S(t,1) + S(t,0) + S(t,2)) * 2 / 15        (int, truncating)
  where S(t,k) = f_0310_1468(t,k,0)
              = Σ_{i=0..2} (T[t].base[i] + T[t].cur[i] + T[t].rnd[i]) * W[k][i] / 100   (each term /100 separately)
                + (T[t].b17 - 44)
    T[t] = team record F:0x3066+t*0x22: cur = +0x18..0x1a, base = +0x1b..0x1d, rnd = +0x1e..0x20, b17 = +0x17
hs = str(h); as = str(a)
price = priceOverride != 0 ? priceOverride : St.price (F:m+0x234c)
L = F:m+0x237a (manager's league);  k = 1 << (2 - L)   (4, 2, 1)
if k == 4 and hp > 14: hp = 14
if hs == 0 and as == 0: return 1

// floating point part (double)
A = (1.0 - (ap - 1) * 2.0 * 0.01) * as
B = (1.0 - (hp - 1) * 2.0 * 0.01) * hs
X = (A + B) * (10/43) * 1150.0
r = rand((price - 10) * 1050, price * 1550 - 14400)       // int16 args, see note
att = (long) trunc(X - r + 1000.0)                         // f_3a01_3156 = _ftol

Q = (hs + as) * 100 / 180                                  // long
att += rand(5Q, 9Q) * Komfort   - 20Q
att += rand(4Q, 12Q) * Zustand  - 31Q
att += (10*Flutlicht    - 8) * Q
att += (10*Anzeigetafel - 5) * Q
att += (Überdacht/1000  - 8) * Q
if popularity(F:m+0x241e) > 50: att += (uint16)(30*popularity - 1500)
att += -2000 * L
if D:0x4a28 != 0: att += 1200                              // difficulty byte
if Sitz + 2000 > Steh: att += rand(400, 700)

if generic:  j = league index of h (D:0x2272 ranges); Tj = D:0x2c8[j]
             cap = rand(Tj - Tj/3, Tj + Tj/3) * 10      // 30-60k, 14.7-29.3k, 8-16k, 4-8k
else:        cap = Sitz + Steh                          // current values, pending not counted

if P4 > 3: P4 = 3
if L == 2: P4 = 0
if as < 90 and P4 > 1: P4 -= 1
if price < 22: att = att * (P4 + 2) / 3                  // P4: 0 -> x2/3, 1 -> x1, 2 -> x4/3, 3 -> x5/3

md = D:0x225a[L]; nmd = D:0x2262[L]; nt = D:0x225e[L]; t = (L != 0)
if md + 5 > nmd and price < 22:                           // last 5 matchdays
    if md == nmd: att = cap                               // last matchday: always sold out
    else:
        if hp < 7 - 2t: att += cap/4 ; if ap < 7: att += cap/5   // title race (top 7 / top 5)
        if hp + t + 5 > nt: att += cap/4                          // relegation battle
if price > 10: att += (10 - price) * 900

limit = (2k - (k == 4 ? 1 : 0)) * 100000                  // 700000 / 400000 / 200000 DM
while att * price > limit: att -= 1000                    // gate revenue cap per match
while att <= 500: att += 500
if att > cap: att = cap
return att
```

Notes:
* RNG call order: r, rand(5Q,9Q), rand(4Q,12Q), [rand(400,700)], [cap rand]. P5 is unused.
* `r` arguments are 16-bit ints; for price <= 7 the range is "negative" (b < a); the original then
  computes `rand() % (b-a+1)` with a negative divisor (C: result in [0, |b-a+1|-1]) — reproduce with
  C-style truncating modulo.
* Higher price lowers `X - r` by ~1300/DM plus 900/DM above 10 DM; prices >= 22 lose the cup and
  end-of-season bonuses.
* The "generic" capacity and AI price are only used for a human's away cup games.

### 3.3 Statistics screen (f_262f_0af7, rows 0x13..0x18 of the stats table D:0x2448+4*row)

* BESUCHERZAHL = F:M+0x2426 (or text D:0x4a46 when no home game yet).
* SCHNITT/SPIEL = 0x2426 / 0x237c.
* BENÖTIGTER SCHNITT = f_14a4_2b15():
  `max(0, (12*monthlyCosts(f_14a4_2822) - 12*monthlyIncome(f_14a4_2636)) / (D:0x225e[L] - 1) / price)`
  i.e. the average attendance per home game needed to cover a year's deficit.
* ZUSCHAUERREKORD = 0x242a + " (" + team name(0x2436) + ")"; MINUSKULISSE = 0x242e (shown as missing
  while 99999) + opponent 0x243a.
* Season reset (f_1a58_3176 @1a58:3839): 0x2426 = 0, 0x242a = 0, 0x237c = 0, 0x242e = 99999.
* Graph: bars of 0x238c[i] scaled between min and max.

### 3.4 Popularity F:M+0x241e (feeds the attendance and sponsor formulas)

* init 50 - 20*L0; monthly (f_112a_0a6d @ month end, also f_14a4_0906): if `< 95` and
  `rand(0,22) < D:(m*0x24+0x68c)(long)/2500` then `+= rand(1,3)` (D:0x68c = advertising budget,
  UNCERTAIN). Season end (f_1a58_3176 ~line 18110): recomputed; floored to 30 for L < 2 and to 61 for
  L == 0 (base value comes from season result — see season-end spec).

---------------------------------------------------------------------------------------------------
## 4. Periodic stadium processing

### 4.1 Daily: construction progress — f_0000_20e1 (first call of f_112a_0a6d, which runs once per
day per human manager, f_1a58_3176)

```
for k in 0..6:
    if F:0x5780+k != 0: F:0x5780+k -= 1            // lockout (global! decremented once per manager per day)
    if timer[k] != 0:
        timer[k] -= 1
        if timer[k] == 0:
            cur[k] += pending[k]; pending[k] = 0
            message: line1 = D:0x28d8+4k, then (k in 3,4,6) name line, then "ist abgeschlossen."
```
Messages (D:0x28d8+4k): "Der Ausbau der Sitzplätze" / "Der Ausbau der Stehplätze" /
"Die Überdachung des Stadions" / "Die Errichtung der neuen" + "Flutlichtmasten" /
"Die Errichtung der neuen" + "Anzeigetafel" / "Die Stadionrenovierung" /
"Die Verbesserung des" + "Stadion-Komforts", each followed by "ist abgeschlossen.".
Delivery: popup immediately if `0x80+7*D:0x224d < day < 0xcf-14*D:0x224d` or `day > 0x142`,
otherwise queued into the mailbox (f_0000_239e) — UI only.

### 4.2 Per turn: f_0cb5_13bd (called for each human after the day loop; the calendar stops when
`day % 7 == D:0x224c (0)` or `day % 7 == 4`, i.e. twice a week). Whole function is skipped when
day counter D:0x7dc >= 322.

### 4.3 Riots ("Randalierer im Stadion")

Trigger (f_1a58_20b2 @1a58:291a, after a human **home** match):
```
x = (int16)(Steh_lo16 - Sitz_lo16) / 1000      // 16-bit arithmetic on the low words
x = clamp(x, 0, 30)
if rand(0, (30 - x)*4 + 15) == 0: F:M+0x2380 |= 1
```
Effect, next turn in f_0cb5_13bd (if bit0 set):
```
damage = ((4*Steh + Sitz/2) * rand(2,5) / 1000) * 1000          // long
money -= damage; F:M+0x2380 ^= 1
message "Randalierer im Stadion" / "richteten einen Sachschaden" / "von <damage> DM an."
if Zustand > 1:
    if money > 2,000,000 and rand(0,1) == 0: Zustand -= 1          // money AFTER the damage
    elif rand(0,2) == 0:                     Zustand -= 1          // (rich: P=2/3, else 1/3)
```
This is the **only** way Zustand decreases (no time-based decay).

### 4.4 Comfort downgrade

Every turn (f_0cb5_13bd), always consuming one random number:
```
if rand(0, 442 - 52*Komfort) == 0 and Komfort > 1 and L == 0 (1. Bundesliga only):
    Komfort -= 1; message "In der neuen Komfortbewertung" / "wird ihr Stadion um eine Note" /
                          "schlechter beurteilt."
```
(Probability 1/131 at SEHR GUT, 1/391 at MANGELHAFT per turn.)

### 4.5 Monthly upkeep (part of f_14a4_2822 = monthly expenses, paid on the last day of each month,
D:0x7b8 = month lengths)

Stadium/operations block of the monthly cost (other terms = salaries etc., see finance spec):
```
175000*(3 - L) + 5500*(Flutlicht + Anzeigetafel) + 12000*Komfort + 4*Sitz + 2*Steh + Überdacht
```
(exact evaluation order: `(((3-L)*43750 + (F+A)*1375 + K*3000 + Sitz)*2 + Steh)*2 + Überdacht`.)
Whether the (3-L) term is meant as stadium cost is UNCERTAIN; it is in the same expression.

### 4.6 Stadium attractiveness for sponsor offers — f_14a4_2bba

`v = min(Sitz + Steh + Überdacht/2, 80000) + 1900*Anzeigetafel + 1500*Flutlicht + 1000*(2*Komfort +
Zustand); v = min(v, 100000) / 1000` (0..100). Used by f_14a4_2cb4 (advertising/sponsor offers): if
v < 50 and popularity > 29 then v += 20; base = v/2 + 50 ... (see finance spec).

---------------------------------------------------------------------------------------------------
## 5. Construction dialog (menu entry -> f_0000_0602, confirm f_0000_0000)

Overview (f_0000_1a49): "IHR KONTOSTAND: x DM"; table rows r=0..8 with columns "MOMENTAN" and
"NACH AUSBAU": row 0 GESAMTKAPAZITÄT = Sitz+Steh (+pending when timer[0] or timer[1] running);
rows 1-3 numbers, rows 4-5 size texts, rows 6-7 grade texts, row 8 price + " DM". Rows with a running
timer show "x WOCHE(N)" = `timer/7 + 1`. Stadium picture (f_0000_16e3):
`size = (total>14000) + (total>34000) + (total>54000)` (total = Sitz+Steh current),
picture `PIC (11 + 3*size - (Zustand-1)/2)`, plus light-mast / scoreboard overlays when
Flutlicht/Anzeigetafel != 0.

Clicking row p (1..8) in f_0000_0602:
0. If lockout `F:0x577f+p != 0`: "Im Moment keine Baufirma aufzutreiben." (p=8 never locked: byte
   F:0x5787 is never set).
1. Show "KOSTET <cost[p]> DM PRO 1000" (p<4) / "DM PRO PUNKT" (p 4..7).

**Places (p = 1 Sitz, 2 Steh, 3 Überdachen):**
```
capMax(p=1,2) = 130000 - (Sitz + SitzPend + Steh + StehPend)
capMax(p=3)   = Sitz + Steh - Überd - ÜberdPend           // current seats/standing only
afford = max(0, money / cost[p] * 1000)                    // long; in units of places
if afford == 0: "Kein Geld für neue Plätze da..."; stop
max = min(capMax, afford)      -> "MAX. GRÖßE: max", " KOSTEN: (max/1000)*cost DM."
amount = number input (0..max, step 1000)                  // f_3091_0ba5
if amount accepted: Confirm(p, amount, cost = (amount/1000)*cost[p])
```
**Levels (p = 4 Flutlicht, 5 Anzeigetafel, 6 Zustand, 7 Komfort):**
```
top = (p >= 6) ? 6 : 3;  now = cur + pending (low words)
if now == top: "Was denn...? NOCH besser ???"; stop
afford = money / cost[p]          // levels affordable
if afford <= 0: "Da müssen Sie erst noch sparen..."; stop
maxLevel = min(top, now + min(top, afford))
list levels (p4/5: 1..3, p6/7: 2..6), price for L > now = (L - now)*cost[p]; selectable now < L <= maxLevel
"MAX. STATUS: <text>  (KOSTEN: (maxLevel-now)*cost DM)"
Confirm(p, L, (L - now)*cost[p])
```
**Ticket price (p = 8)** (no confirm, no cost, immediate):
`newPrice = input(min 5, max 25, step 1)` (in the alternate currency mode, D:0x224e >= 2, the input is
scaled ×10 with step 5); `F:M+0x234c = newPrice`.

**Confirm — f_0000_0000(p, amount, cost):**
```
if money - cost < 0: "Zu wenig Kleingeld vorhanden..."; return -1
old  = timer[p-1] * rand(20, 80) / 100                     // long, RNG call 1
days = (old + 7 * D:0x63[p]) * rand(80, 120) / 100         // RNG call 2
show "AUSBAU UM <amount> AUF <cur+pending+amount> <row label>" (p<4)
     or "NEUE GRÖßE:/NEUER STATUS: <text(amount)>" (p>=4); " KOSTEN: cost DM"
     "BAUZEIT: CIRKA" (old == 0) / "BAUZEIT: INSGESAMT CA." (old != 0) + (days/7 + 1) + " WOCHEN"
buttons "NA KLAR !" / "ACH NEE..."
if "ACH NEE...": F:0x577f+p = rand(15, 55)   // no company for 15-55 days for this item!
                 return -1
money -= cost
pending += amount; if p > 3: pending -= (cur + oldPending)    // => pending = newLevel - cur
timer[p-1] = days (low 16 bits)
return 1
```
Notes: a second order while one is running adds to `pending` and restarts the timer with
`old` = 20-80 % of the remaining days plus a new full base time. Maximum stadium: 130,000
seats+standing; covered places cannot exceed current seats+standing.

---------------------------------------------------------------------------------------------------
## 6. Training camps ("Trainingslager", PIC 5) — f_112a_0145, effect f_112a_072d

Eight camps i = 0..7; names D:0x2394+4i; five rating bars (letters D:0x29a4 "EFPRV",
hover texts D:0x28f8..: Erholungswert, Fitnessausrüstung, Platz-Qualität, Raumausstattung,
Verpflegung), ratings `R[i][j]` = D:0x527e + 5i + j (displayed 1..5):

| i | camp | E | F | P | R | V | price factor D:0x5258+4i | E[0] E[1] E[2] |
|---|---|---|---|---|---|---|---|---|
| 0 | Kickfelder Mühle | 1 | 2 | 1 | 1 | 1 | 5 | 34 26 26 |
| 1 | Trlg. Hackstedt | 1 | 2 | 3 | 1 | 1 | 6 | 42 46 34 |
| 2 | Hof Schweißperle | 1 | 3 | 2 | 1 | 1 | 7 | 50 38 34 |
| 3 | Fitnesscenter Krone | 2 | 4 | 3 | 2 | 2 | 10 | 72 62 56 |
| 4 | Scola Ludum Pilae | 2 | 1 | 2 | 4 | 4 | 11 | 36 62 52 |
| 5 | Sportschule Träne | 2 | 4 | 3 | 2 | 2 | 10 | 72 62 56 |
| 6 | Hotel Medium | 3 | 2 | 2 | 3 | 4 | 13 | 50 60 58 |
| 7 | Luxushotel La Preux | 5 | 3 | 2 | 4 | 5 | 15 | 72 78 84 |

Price: on entering the screen, if `price[0] < 10` (still unscaled) every `price[i] *= squadSize*2000`
(squad size of the manager opening it: f_3091_1109). Shown on hover as "<name>: <price> DM/Woche.".
Prices are global and not saved (re-scaled after loading a game).

Open/closed: `D:0x620[i]` > 0 = closed ("<name> hat nicht geöffnet..."), <= 0 open. Initial values
-80,-50,-100,-60,+80,+50,+70,+60. Daily (f_112a_0a6d, once per human manager per day):
`s = sign; v -= sign(v); if v == 0: v = rand(20,70) * (old>0 ? -1 : +1)` (open/closed periods of
20-70 days alternate; a 0 value would stay 0).

Booking: menu entry disabled while `F:M+0x237b != 0`. Click camp i: closed -> message; second
booking in the same screen -> "Einmal pro Woche reicht doch..."; money < price -> "Keine finanzielle
Möglichkeit gegeben..."; else money -= price, run effect, close screen.

Effect f_112a_072d(i) (W = D:0x52a6, rows j = E,F,P,R; V has no effect):
```
W = [[20,20,40],[60,10,20],[20,50,20],[10,40,30]]
E[k] = (Σ_{j=0..3} R[i][j]*W[j][k]) / 5         (k: 0 -> +0x10 Kondition, 1 -> +0x11 Technik, 2 -> +0x12 Form)
for each squad player s (F:0x774a+(m*25+s)*0x34):
    d = (E0+E1+E2)/(-3) + (E0-E2+10)*2 + E1; d = max(d, 80)      // = 80 for all camps
    s[+0x13] += rand(d/9, d/6); clamp to [60,150]               // rand(8,13); field = "Erschöpfung" per CONTEXT (direction UNCERTAIN)
    for k in 0..2:
        t = (rand(2D+87, 2D+117) + E[k] - 100) / 10            // D = D:0x4a28
        a = rand(0, 20*t/30); b = rand(0, t); t = t + b - a
        s[+0x10+k] += t >> 2 (arith.); if (unsigned) > 99: = 99
    t += t * E[2] / 200                                          // t of k = 2
    if rand(0,4) == 0: s[+0x0e] += (rand(155,300) * t) / 800    // tendency raw
    f_2277_000a(s)                                               // recompute derived values
F:M+0x237b = rand(7,18)          // cooldown days, decremented daily in f_112a_0a6d
```

---------------------------------------------------------------------------------------------------
## 7. Youth department (Jugend) — slider on the training screen f_112a_274c (PIC 37 is just the
button/icon sheet used by f_06c7_00a7, not a separate youth screen)

* `F:M+0x2381` = youth investment 0..34 (slider, clamped by f_3091_1009(.,0,0x22)).
* Monthly (f_112a_0a6d month end): `0x2424 += 0x2381 * 4`; monthly cost in f_14a4_2822:
  `((0x2381 + 1) * 10000) >> 4` = 625*(level+1) DM (plus D:0x68c budget term, finance).
* Season end (f_0cb5_0012 @0cb5:033d): `avg = 0x2424 / 12` (unsigned). Only if
  (year F:0xa7a0 > 1966 and not modern variant) or (year > 1995 and modern variant):
  `avg = avg + avg/(-3)` (C truncation); `0x2424 = avg*12`; if `avg > 30` and `rand(0,2)==0` and squad
  size (f_112a_00b4) < 23: `avg >>= 1`, message "Ein Jugendspieler steigt in den Kader auf.", pick a
  random free person (owner F:+0x57fe == 5, rand(1,150) until free), `avg = clamp(avg,10,90)`,
  +0x20 = rand(0,6), age = rand(17,19), skill3(+0x1e) = rand(45,55), skill2(+0x1d) = rand(avg-3,avg+3),
  skill1(+0x1c) = rand(avg-3,avg+3), team = manager; added to squad (f_1ecd_37d8); salary
  = value(f_2277_25de)*50/100; contract = rand(2,3).

---------------------------------------------------------------------------------------------------
## 8. Function list

| function | meaning |
|---|---|
| f_0000_0602 | stadium screen main loop (row selection, limits, price input) |
| f_0000_0000 | confirm order: money check, construction time, apply/lockout |
| f_0000_1a49 | stadium overview table / Kontostand |
| f_0000_16e3 | stadium picture + overlays |
| f_0000_20e1 | daily: lockout and construction timers, completion messages |
| f_0000_239e | enqueue message into mailbox (generic) |
| f_0000_243f | status/hover line (also used by bank and camp screens) |
| f_0f9d_11e0 | attendance formula (FP emulated) |
| f_1a58_20b2 | per-match gate income, records, cup sharing, riot trigger |
| f_0cb5_13bd | per-turn processing: riots damage, comfort downgrade (+ squad stuff) |
| f_112a_0a6d | daily per manager: construction, camp timers, month-end finance, popularity |
| f_112a_0145 / f_112a_072d | training camp screen / camp effect |
| f_14a4_2822 | monthly expenses (incl. stadium upkeep) |
| f_14a4_2b15 | "BENÖTIGTER SCHNITT" |
| f_14a4_2bba | stadium attractiveness 0..100 for sponsor offers |
| f_08bc_2184 | new game: initial stadium |
| f_262f_0af7 | statistics screen (attendance rows) |

## 9. Open questions / UNCERTAIN

* Exact meaning of `comp`/`leg` params of f_1a58_20b2 (gate split rule for cups) and of F:0x8/F:0x9.
* Popularity F:M+0x241e naming and its season-end recomputation formula (finance/season spec).
* Training-camp field +0x13: whether "+rand(8,13)" is good (freshness) or bad (exhaustion).
* "Sold out" report flag compares capacity with the season attendance sum (likely bug, kept as is).
* Integer-overflow quirks (16-bit rand args in the cup boost and the riot formula) should be kept
  for faithfulness but are rarely hit.
