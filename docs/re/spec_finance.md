# BMMAIN.EXE – Finances & club economy (spec)

Scope: account balance, all regular income/expense, monthly bookkeeping, ticket income /
attendance, sponsors (Trikot/Banden/TV/advertising spend), bank & player credits, training
camps, Christmas, riots, promotion/top-scorer bonuses, licence refusal, starting money,
finance statistics. Stadium construction and transfer fees are only referenced.

Conventions: `F:` = segment 0x4238, `D:` = DGROUP 0x4cb3. `M(m)` = manager record base
`F:0x2242 + m*0x30a` (m = 0..3; `F:0x304a` = current manager). Offsets below written as
absolute addresses for m = 0 (add `m*0x30a`). "long" = signed 32-bit, little endian.
`rand(a,b)` = `f_076b_0cc7` = `rand() % (b-a+1) + a` (MS C LCG). Integer division is C
truncation toward zero (`f_3a01_1a4c` signed long div, `f_3a01_1ae6` long mul,
`f_3a01_1c10` unsigned long div, `f_3a01_1b18` long mod, `f_3a01_1a08`/`1bd0` = `*p /= x`,
`f_3a01_1a2a`/`1bf0` = `*p *= x`, `f_3a01_1bb8`/`1bc4` = long shl/sar by CL,
`f_3a01_3156` = float→long with truncation). `f_3091_1009(&v,lo,hi)` = clamp.
`f_3091_1109(m,0)` = number of players in manager m's squad (squad size).
Level `L5 = D:0x4a28` (0..5): the UI shows "LEVEL (5 - D:0x4a28)"; at new game
`D:0x4a28 = 5 - selection`.

Currency display: every DM amount shown is `amount * D:0x2252(long) / D:0x224e(long)`,
inputs are converted back with `input * D:0x224e / D:0x2252`; both are 1 in the shipped
data (pure DM). `D:0x224d` (byte, 0) selects an optional prefix string (empty). Ignore for
a DM-only re-implementation.

---------------------------------------------------------------------------------------
## 1. Data layout

### 1.1 Manager record fields (F, per manager, stride 0x30a)

| addr (m=0) | type | meaning |
|---|---|---|
| 0x2242 | char[] | manager name |
| 0x2260 | byte | team id |
| 0x234c | byte | ticket price (DM per spectator) – user range 5..25 (slider, f_0000_0602 @0000:0d64..0eb0) |
| 0x2373 | byte | init 16 (used by team-strength calc, not finance) |
| 0x237a | byte | league L: 0 = Bundesliga, 1 = 2. Liga, 2 = Amateur-Oberliga |
| 0x237b | byte | training-camp days remaining (set rand(7,18) by a camp, -1 per day) |
| 0x237c | byte | number of home league games this season (index into 0x238c) |
| 0x237f | byte | 100 ('d') = fewer than 8 fielded players (set by f_0f9d_0002) → forfeit |
| 0x2380 | byte | bit0 = "riot happened at last home game" (pending damage) |
| 0x2381 | byte | youth work level 0..34 (slider in f_112a_274c) |
| 0x2382 | byte | season-end flags: 1 promoted, 2 relegated, 4 licence refused, 8 restart with 500.000 |
| 0x238c+i | byte[] | attendance/1000 of home league game i |
| 0x23a0 | 7 × {long cur, long building} | stadium: [0] 0x23a0 Sitzplätze, [1] 0x23a8 Stehplätze, [2] 0x23b0 Überdachung, [3] 0x23b8 Flutlicht, [4] 0x23c0 Anzeigetafel, [5] 0x23c8 Renovierung/Zustand, [6] 0x23d0 Komfort-Note (names from f_0000_20e1; [0]/[1] meaning UNCERTAIN but consistent with all uses) |
| 0x241e | word | popularity / "Beliebtheit" P, 0..100 (unsigned) |
| 0x2420 | word | monthly fixed cost item (see 3.2 term E3; likely "Reisekosten", UNCERTAIN) |
| 0x2422 | word | last computed base income ("Spenden" part, low word) |
| 0x2424 | word | youth points (+level*4 per month) |
| 0x2426 | long | season total home attendance (league) |
| 0x242a | long | attendance record (max) ; 0x2436 word = opponent team id |
| 0x242e | long | "Minuskulisse" (min) ; init 99999 ; 0x243a word = opponent |
| **0x2432** | **long** | **Kontostand (account balance), signed DM** |
| 0x243e | 15 × 0x12 | credits: slot (lender k*3 + j), k = 0..3 human manager, 4 = bank; j = 0..2 |

Credit slot (0x12 bytes) at `0x243e + (k*3+j)*0x12` (written by f_112a_1039):

| off | type | meaning |
|---|---|---|
| +0x00 | long | amount (0 = slot free) |
| +0x04 | long | interest per month = amount * rate / 100 |
| +0x08 | byte | start day (1-based) |
| +0x09 | byte | due day (day-of-month on which interest is paid) |
| +0x0a | byte | start month (0-based) |
| +0x0b | byte | due month (0-based) |
| +0x0c | byte | rate in % |
| +0x0e | word | start year |
| +0x10 | word | due year |

### 1.2 Sponsor / advertising globals

| addr | type | meaning |
|---|---|---|
| D:0x066c + (m*9+i)*4 | long[9] per mgr | i=0 Trikotwerbung income/month; i=1..6 Bandenwerbung board 1..6; i=7 TV-Übertragung; i=8 Werbeausgaben (expense) |
| F:0x0000 + m*2 | byte | Trikot contract months remaining |
| F:0x0001 + m*2 | byte | Trikot sponsor index (0..9) |
| F:0x4b9a + (m*6+b)*2 | byte | board b months remaining; +1 = sponsor index |
| F:0x002e + ((m*2+t)*10+k)*4 | long | offer k (0..9) of type t (0 Trikot, 1 Bande); 0 = no offer |
| F:0x2ec2 + (m*2+t)*10+k | byte | offer duration in years (1..3) |

### 1.3 Other globals

| addr | meaning |
|---|---|
| D:0x0644 + m*4 (long) | sum of daily balances in current month (NOT saved; reset monthly) |
| D:0x060e [6] (byte) | bank interest table (% for 3,6,9,12,15,18 months); init {7,6,5,4,3,2}; saved |
| D:0x0630 (word) | bank base rate, init 5 (not saved) |
| D:0x0620 [8] (word) | training-camp open/closed counters; init {-80,-50,-100,-60,80,50,70,60}; saved |
| D:0x5258 [8] (long) | training-camp prices; init {5,6,7,10,11,10,13,15} (multipliers, see §7); not saved |
| D:0x07b8 [12] (byte) | days per month {31,28,31,30,31,30,31,31,30,31,30,31} (no leap years) |
| F:0x2eba (word) day 1..31, F:0x2e8e (word) month 0..11, F:0xa7a0 (word) year |
| D:0x07dc (long) | day counter of the season; season games while ≤ 0x141 |
| D:0x07e2 (word) | number of completed seasons |
| D:0x225e [3] | teams per league {18,20,20}; D:0x2262 [3] matchdays {34,38,38}; D:0x226e [3] first team id {0,18,38}; D:0x2272 [4][2] team-id ranges {0-17},{18-37},{38-57},{58-255}; D:0x225a [3] current matchday per league |
| D:0x5390 [4] (word) | default ticket price of non-human home team per league bucket {16,14,10,8} |
| D:0x02c8 [4] (long) | base capacity/10 for non-human home stadium {4500,2200,1200,600} |
| D:0x2256 (long) | DFB-Pokal final attendance 76000 |
| D:0x53d0 / D:0x53d4 | special days: day {24,12,19}, month(1-based) {12,11,4} → Christmas, two programmer birthdays |

---------------------------------------------------------------------------------------
## 2. Time structure (who calls what) – f_1a58_3176

* Every simulated day, for every human manager m (in order 0..n-1): `f_0310_1190` (calendar)
  then **f_112a_0a6d** (daily bank/booking, §3). Also during the summer break until
  28 July (day 28, month 6).
* On each special date (`day == D:0x53d0[i] && month == D:0x53d4[i]-1`) `f_1a58_2a06(i)`
  (§8; i=0 Christmas). Checked every day.
* The day loop stops at "turn days": `(day - D:0x224c) % 7 == 0 || (day - 4) % 7 == 0`
  (D:0x224c = 0) → for every manager **f_0cb5_13bd** (weekly events incl. riots, §9), and if
  `(day - D:0x224c) % 14 == 0` **f_14a4_2cb4** (new sponsor offers, §5.2).
* Matches: f_1a58_20b2 (gate money, §4) per fixture involving a human team.
* Season end (after last matchday): popularity update (§6.2), licence check (§10),
  promotion/relegation, `f_14a4_2cb4` per manager, `f_0cb5_0012` (season summary with
  bonuses, §10), D:0x7e2++, summer days, then `f_08bc_0a63(m)` for all managers (TV money
  and cost item 0x2420 for the new league, §5.3).

---------------------------------------------------------------------------------------
## 3. Daily & monthly bookkeeping – f_112a_0a6d (per manager per day)

Steps, exactly in this order:

1. `f_0000_20e1()` – stadium construction progress (not in scope).
2. Training-camp availability (D:0x620[i], i = 0..7) — runs once per *manager* per day:
   `s = (x<0) - (x>0); x += s; if x == 0: x = rand(20,70) * s`.
   (x>0 = closed for x days, x<0 = open for -x days; at 0 it flips to the other state
   with a new random length 20..70.)
3. `if rand(0,60) == 0: f_112a_000a()` (bank rate drift, §7.1).
4. `D:0x644[m] += money` (balance accumulator, signed long).
5. `if F:0x237b != 0: F:0x237b--` (camp days).
6. **Month end** – if `day == D:0x7b8[month]`:
   1. if `month % 4 == 0`: set autosave flag D:0x5256 = 1.
   2. Trikot: if `F:(m*2) != 0`: decrement; if it became 0 → `D:0x66c[m][0] /= 10`
      (income drops to 10 % and stays until a new contract).
   3. Boards b=0..5: same with `F:0x4b9a[m][b]` and `D:0x66c[m][1+b]`.
   4. `0x2424 += 0x2381 * 4` (youth points).
   5. `money += f_14a4_2636()` (income, §3.1).
   6. `money -= f_14a4_2822()` (expenses, §3.2).
   7. `D:0x644[m] = 0`.
   8. Popularity from advertising: if `P < 95` and
      `D:0x68c[m] / 2500 > rand(0,22)` → `P += rand(1,3)`.
7. **Credits** (§7.3) for lender k = 0..4, slot j = 0..2.

### 3.1 Monthly income – f_14a4_2636 (screen label "EINNAHMEN (SPENDEN, SPONSOREN)")

```
team = M.0x2260
S = Σ_{k=0..2} (F:0x307e[team*0x22+k] + F:0x3081[team*0x22+k])   // 6 team strength bytes
base = S*15 * squadSize / 3 + P/5 + 25          // "Spenden"; low word stored to 0x2422
inc  = base + Σ_{i=0..7} D:0x66c[m][i]           // Trikot + 6 boards + TV
for every other human manager o != m:
    inc += total monthly interest of ALL of o's credits (f_112a_0f8d(1,'c') with
           current manager temporarily = o)       // see note
if D:0x644[m] > 0: inc += (D:0x644[m] / day) / 75 // ~1.33 %/month on avg positive balance
return max(inc, 0)
```
Note (faithful bug): with ≥2 humans each manager receives the interest the *other*
managers pay on all their credits (also bank credits); the interest a borrower pays is
otherwise never credited to the lending manager. `day` is the current day (= days of month).

### 3.2 Monthly expenses – f_14a4_2822 ("AUSGABEN (GEHÄLTER, REISEKOSTEN)")

```
E  = 150000                                              // fixed base
E += Σ squad slots s of m: slot.long@+0x28 (monthly salary, F:0x7772 + (m*25+s)*0x34)
E += Σ slots s of pool 4 (F:0x774a+(100+s)*0x34) whose person's owner (+0x21) == m: salary
     (players lent out still paid)
St = 4*((3-L)*43750 + (Flutlicht+Anzeigetafel)*1375 + Komfort*3000 + Sitz) + 2*Steh + Überdachung
E += St                                                  // stadium/operation costs
E += (uint16) M.0x2420                                   // E3, see §5.3
E += ((M.0x2381 + 1) * 10000) >> 4                       // youth: (lvl+1)*625
E += D:0x68c[m]                                          // Werbeausgaben
E += Σ monthly interest of all own credits (f_112a_0f8d(1,'c'))
if D:0x644[m] < 0: E += (D:0x644[m] / day) / -10        // 10 %/month on avg overdraft
if L != 2 and money > 2,000,000:
    t = ((money - 2,000,000) / 500,000) * 30,000 + 20,000
    if money >= 4,000,000: t = t*3/2
    E += t                                               // "wealth" charge
return max(E, 0)
```
(stadium values = the "cur" longs of §1.1.)

### 3.3 Break-even attendance ("BENÖTIGTER SCHNITT") – f_14a4_2b15
`max(0, (12*expense - 12*income) / (D:0x225e[L] - 1) / ticketPrice)`.

---------------------------------------------------------------------------------------
## 4. Match day gate money & attendance

### 4.1 f_1a58_20b2(home, away, comp, p4, p5, leagueMask)
comp 0 = league, 1 = DFB-Pokal, ≥2 = European cups. For each human manager m involved:

* `split = 2` if comp != 0 and p4 == 0 (one-off cup tie: gate split between both clubs),
  else 1. (p4 semantics UNCERTAIN – second leg/replay flag.)
* **DFB-Pokal final** (comp == 1 and F:0x0008 > 4): each human finalist gets a flat
  **+532,000 DM** (0x81e20); attendance = 76000; no other gate money.
* Home manager:
  * Forfeit: if `M.0x237f == 100` (<8 players) and not (month == 5 and day ≥ 10):
    `f_1a58_2051`: "Ihr Spiel wird mit 0:2 gewertet. Sie zahlen 200.000 DM Strafe",
    **money -= 200,000**, return. (Same check for the away manager.)
  * `att = f_0f9d_11e0(m, home, away, c, split, 0, 0)` with c = 1 for comp ≤ 1, c = 2 for
    European cups, c = 3 if additionally F:0x0009 > 4.
  * League (comp 0): record `M.0x238c[M.0x237c] = att/1000`, `0x2426 += att`, update max
    record (≥) and min record (≤) with opponent id, `0x237c++`.
  * Cup: let b = league bucket of the away team (D:0x2272 ranges). If `b < L` (opponent
    from a higher league): `att += rand(att / (8 - 3*(L-b)), att)` (16-bit rand bounds),
    then `att = min(att, Sitz + Steh)`.
  * **money += att * M.0x234c / split**.
* Away manager in a split cup tie: if the home side is human its `att` is reused and the
  away manager gets `att * homePrice / 2`; otherwise
  `price = D:0x5390[bucket(home)] + rand(0,1)`,
  `att = f_0f9d_11e0(awayMgr, home, away, 1, 2, 1, price)` and money += `price*att/2`.
* Riot roll (only for the home manager, after the game, in the replay/result block):
  `x = clamp((int16)(Steh.lo - Sitz.lo) / 1000, 0, 30)`;
  `if rand(0, (30-x)*4 + 15) == 0: M.0x2380 |= 1`. Damage is booked in f_0cb5_13bd (§9).

### 4.2 Attendance – f_0f9d_11e0(m, h, a, c, split, capFlag, priceOverride)
(uses 8087 emulation; decoded with patched disassembly)
```
Sd = stadium of manager m (§1.1 "cur" values)
posH = h<64 ? F:0xefa[h*0x36] : 5      // league table position, 0-based
posA = a<64 ? F:0xefa[a*0x36] : 5
strH = (f_0310_1468(h,1,0)+f_0310_1468(h,0,0)+f_0310_1468(h,2,0))*2/15   // team strength
strA = same for a
price = priceOverride != 0 ? priceOverride : M.0x234c
L = M.league ; lf = 1 << (2-L)
if lf == 4 and posH > 14: posH = 14
if strH == 0 and strA == 0: return 1
X = ((1-(posA-1)*0.02)*strA + (1-(posH-1)*0.02)*strH) * (10/43) * 1150.0      (double)
r = rand((price-10)*1050, price*1550 - 14400)            // int16 arithmetic for bounds
att = trunc(X - r + 1000.0)
q = (strH + strA) * 100 / 180
att += Komfort    * rand(5q, 9q)  - 20q                  // 5q,9q,4q,12q are int16
att += Renovierung* rand(4q, 12q) - 31q
att += (Flutlicht*10 - 8) * q
att += (Anzeigetafel*10 - 5) * q
att += (Überdachung/1000 - 8) * q
if P > 50: att += (uint16)(30*P - 1500)
att -= 2000*L
if D:0x4a28 != 0: att += 1200
if Sitz + 2000 > Steh: att += rand(400,700)
md = D:0x225a[L]; total = D:0x2262[L]
cap = capFlag ? rand(T - T/3, T + T/3)*10  (T = D:0x2c8[bucket(h)]) : Sitz + Steh
c = min(c,3); if L == 2: c = 0; if strA < 90 and c > 1: c--
if price < 22: att = att*(c+2)/3
if md + 5 > total and price < 22:          // last 4 matchdays
    if md == total: att = cap               // last matchday sold out
    else:
        if posH < 7 - 2*(L!=0): att += cap/4; if posA < 7: att += cap/5
        if posH + (L!=0) + 5 > D:0x225e[L]: att += cap/4
if price > 10: att += (10 - price)*900
lim = (2*lf - (lf==4)) * 100000            // 700k / 400k / 200k DM gate cap
while att*price > lim: att -= 1000
while att <= 500: att += 500
return min(att, cap)
```
RNG call order: r, rand(5q,9q), rand(4q,12q), [rand(400,700)], [cap rand].

---------------------------------------------------------------------------------------
## 5. Sponsors, TV, advertising

### 5.1 Advertising screen – f_262f_2be8 / f_262f_29a8 (PIC 46.VGA)
Shows Trikotwerbung (D:0x66c[m][0]), Bandenwerbung (Σ boards), TV-Übertragung
(D:0x688[m]), Werbeausgaben (D:0x68c[m]) and total = Trikot + Banden + TV − Werbung.
* Accept Trikot offer k: only if `F:(m*2) == 0` (else "Sie stehen noch unter Vertrag");
  `F:(m*2) = dur*12`, `F:(m*2+1) = k`, `D:0x66c[m][0] = offer[0][k]` (offer stays listed).
* Accept board offer k for board b: only if board counter == 0; counter = dur*12,
  index = k, `D:0x66c[m][1+b] = offer[1][k]`, then `offer[1][k] = 0`.
* Werbeausgaben buttons: ±2500; after change: if value > 50000 → 2500; if value < 2500 →
  50000 (wrap-around; legal values 2500..50000).
* Contracts cannot be cancelled by the player.

### 5.2 Offer generation – f_14a4_2cb4 (current manager), uses f_14a4_2bba
```
f_14a4_2bba (stadium attractiveness 0..100):
  v = Überdachung/2 + Steh + Sitz ; v = min(v, 80000)
  v += Anzeigetafel*1900 + Flutlicht*1500 + (2*Komfort + Renovierung)*1000
  v = min(v, 100000) ; return v/1000
for t in {0 (Trikot), 1 (Bande)}: for k in 0..9:
  V = f_14a4_2bba(); if V < 50 and P > 29: V += 20
  r = rand(15,20); A = V/2 + 50; T = V/2 + r
  qq = A/5 + 2k ; offer = rand(11qq + 58, 12qq + 62)
  if t == 0: A *= 5
  Pk = ((7k)/10 + 3) * 3 ; cval = 10k + rand(3,20)
  if cval >= T and rand(0,Pk) != 0: offer = 0; continue      // no offer in this slot
  mlt = rand(P/7 + 100, P/5 + 100)
  offer = offer * A * mlt / 100
  years = rand(0,2); dur[t][k] = years + 1
  lg = 2 - L
  if L == 1: offer = offer*107/100
  offer = offer * rand((years+9)*10, years*15 + 92) / 100
  offer = offer * rand((lg+5)*20, 23*lg + 102) / 100 + D:0x4a28*800 + 16500
  if t == 0: offer += D:0x4a28*2000 - 11000
```
Called: new game, every 14 days, season end.

### 5.3 TV money and cost item – f_08bc_0a63(m) (new game + start of every season)
```
lg = 2 - L
w = uint16( lg*(rand(15,20)+40) + uint16((lg*(110-P)) & 0xFFFE) * 500 )   // 16-bit wrap
M.0x2420 = uint16( w * rand((lg+10)*10, (lg+4)*25) / 100 )               // 16-bit wrap
D:0x688[m] = (30*lg + P) * 1000            // TV-Übertragung per month
```

### 5.4 Sponsor subsidy on a purchase – f_0000_272d(fee) (called by transfer f_2277_04a5
after paying `fee` for a non-human, non-loan transfer)
`if rand(0,6) == 0: s = ((fee/100)*rand(20,65)/10000)*10000`; dialog "NEHMEN SIE EINEN
SPONSOR-ZUSCHUSS VON s DM AN ?" ("ABER IMMER!" / "LIEBER NICHT."); yes → money += s.

---------------------------------------------------------------------------------------
## 6. Popularity P (M.0x241e) – drives TV, offers, base income, attendance
* New game: `P = 50 - 20*L`.
* Monthly: advertising growth (§3 step 6.8).
* Season end (f_1a58_3176 @1a58:3bd5..3d1e, 40f1..4147): prestige snapshot
  `S0 = F:0xefe[team]` before f_1a58_4597 adds the season's prestige; then
  `k = min(D:0x7e2+1, 5)`, `tmp = P + ((Δprestige*100)/108)/k - P/k` (unsigned divs),
  `tmp = clamp(tmp,0,100)`. Faithful bug: tmp is a single variable – after computing it
  for all managers, the *last* manager's tmp is used for everyone, then for each manager in
  order: `if tmp < 30 and L < 2: tmp = 30; if tmp < 61 and L == 0: tmp = 61; P = tmp`
  (floors accumulate into tmp).
* Prestige (f_1a58_4597, per German team t): `+(((t<D:0x226f)+(t<D:0x2270))*25 - pos + 19)`
  (0 if t > D:0x2277), +20/30/40/50 if t+1 equals D:0x7ac/0x7af/0x7ae/0x7ad (cup winners),
  +12 champion of league 0, +3 for pos 1..4 in league 0 or pos 0 elsewhere.

---------------------------------------------------------------------------------------
## 7. Bank & credits (f_112a_*)

### 7.1 Bank rate table – f_112a_000a (new game; then rand(0,60)==0 per manager per day)
```
h = (D:0x4a28 > 2)
base = clamp(base + rand(0, 2+h) + h - 1, 2, 7)          // D:0x630
r = base ; step = 5 - base
for i in 0..5: tbl[i] = r ; r += step ; if step > 1: step-- ; r = clamp(r, 2, 7)
```
tbl = D:0x60e; row i = duration (i+1)*3 months, rate tbl[i] %.

### 7.2 Bank screen – f_112a_167f (lenders: other human managers + "der Bank")
Shows "Dauer Zinssatz" table, Kontostand, Gesamtschulden (Σ amounts), Zinsen/Monat
(Σ interest) (f_112a_23cb); per lender list (f_112a_119b): "d.m.yyyy <amount> DM (r%)",
"FÄLLIG AM d.m.yyyy", "ZINSEN PRO MONAT: x DM"; SCHULDEN/ZINSEN HIER.
Taking a credit:
* amount: numeric input ("Wie hoch soll der Kredit sein ?").
* **Bank**: months = (row+1)*3, rate = tbl[row]. Refused if
  `Σ bank amounts + amount > 4,000,000` ("Sie können nur maximal 4 Mio. DM Kredit von der
  Bank aufnehmen.").
* **Other manager o**: refused if `money(o) < amount` ("So viel Geld besitzt <o> gar
  nicht..."), or `Σ amounts owed to o + amount > 1,000,000` ("maximal 1 Mio. DM ... von
  einem Mitspieler"). Then months input: > 24 → "Maximal 24 Monate." (refused), 0 → abort.
  Rate input: > 13 → "Wegen Wucher ... 1.000.000 DM ... Okay... War nur ein Scherz" (refused,
  NO fine); < 3 → "Mindest-Prozentsatz: 3 Prozent" (refused). Accepted range 3..13 %, 1..24 months.
* f_112a_1039(amount, rate, lender, months): first free slot j (amount==0) of that lender;
  none → "Schon 3 Kredite aufgenommen." (max 3 credits per lender, i.e. 3 at the bank).
  Fill slot: interest = amount*rate/100; start = today;
  `dm = month + months; while dm > 11: dm -= 12, dueYear++`;
  `dueDay = (today <= D:0x7b8[dm]) ? today : D:0x7b8[dm]`.
* On success: money += amount; lender manager's money -= amount (bank: nothing).

### 7.3 Interest & maturity (daily, f_112a_0a6d step 7)
For each used slot:
```
pay = (slot.dueDay == today) or
      (D:0x7b8[slot.dueMonth] > D:0x7b8[month] and today == D:0x7b8[month])
if pay:
    money -= slot.interest                      // interest goes to nobody
    if slot.dueYear == year and slot.dueMonth == month:
        msg "Ihr Kredit von <lender> wurde heute fällig."
        money -= slot.amount ; if lender is a manager: lender.money += slot.amount
        slot.amount = 0
```
Faithful quirk: the second condition compares month lengths, so a credit can pay interest
twice in a short month (on dueDay and on the month's last day). No early repayment exists.
Credits make balances negative freely (no checks) – only §10 punishes.

---------------------------------------------------------------------------------------
## 8. Training camps – f_112a_0145 (PIC 5 + 0x2B) and f_112a_072d

8 camps, index i = column*4 + row; names D-slots 0x2394+i*4:
0 Kickfelder Mühle, 1 Trlg. Hackstedt, 2 Hof Schweißperle, 3 Fitnesscenter Krone,
4 Scola Ludum Pilae, 5 Sportschule Träne, 6 Hotel Medium, 7 Luxushotel La Preux.
* Price: on the first visit in a program run (`D:0x5258[0] < 10`) all 8 entries are
  multiplied by `squadSize(current manager) * 2000` (once, not saved, so after reloading the
  base multipliers apply again with the then-current manager's squad):
  base {5,6,7,10,11,10,13,15} → e.g. 22 players: 220k,264k,308k,440k,484k,440k,572k,660k DM.
* Click: if `D:0x620[i] > 0` → "<name> hat nicht geöffnet..."; else if `money < price` →
  "Keine finanzielle Möglichkeit gegeben..."; else money -= price, apply camp, leave
  screen. ("Einmal pro Woche reicht doch..." is effectively unreachable.)
  The main menu (f_08bc_0b5f @08bc:1624/17a8) disables the camp entry while M.0x237b != 0
  (UNCERTAIN which item, very likely this one).
* Effect f_112a_072d(i): ratings R[i][0..4] = D:0x527e (E,F,P,R,V = Erholungswert,
  Fitnessausrüstung, Platz-Qualität, Raumausstattung, Verpflegung):
  {1,2,1,1,1},{1,2,3,1,1},{1,3,2,1,1},{2,4,3,2,2},{2,1,2,4,4},{2,4,3,2,2},{3,2,2,3,4},{5,3,2,4,5}.
  Weights W[k][j] = D:0x52a6 (k = E,F,P,R; V unused): {20,20,40},{60,10,20},{20,50,20},{10,40,30}.
  `B[j] = Σ_k R[i][k]*W[k][j] / 5` → camps 0..7: {34,26,26},{42,46,34},{50,38,34},
  {72,62,56},{36,62,52},{72,62,56},{50,60,58},{72,78,84}.
  Per squad player:
  ```
  x = (B0+B1+B2)/(-3) + ((B0-B2)+10)*2 + B1 ; x = max(x,80)   // always 80 with this data
  ex(+0x13) += rand(x/9, x/6) (byte add) ; clamp ex to [60,150]
  for j in 0..2 (attr +0x10,+0x11,+0x12):
      t = (rand(2*L5+87, 2*L5+117) + B[j] - 100) / 10
      a = rand(0, t*20/30) ; c = rand(0, t) ; t += c - a
      attr[j] += t >> 2 (arith) ; if (unsigned) attr[j] > 99: attr[j] = 99
  t += t*B2/200                       // t of j=2
  if rand(0,4) == 0: tendency(+0x0e) += rand(155,300)*t/800
  f_2277_000a(player)
  ```
  then `M.0x237b = rand(7,18)`.

---------------------------------------------------------------------------------------
## 9. Weekly events – f_0cb5_13bd (only while D:0x7dc ≤ 0x141)
* **Riot damage** (if M.0x2380 bit0): `dmg = ((Sitz/2 + 4*Steh) * rand(2,5)) / 1000 * 1000`;
  money -= dmg; clear bit; then if `Renovierung > 1`: decrement it with probability
  (money > 2,000,000 ? 1/2 + 1/2·1/3 : 1/3) (rand(0,1)==0, else rand(0,2)==0).
  Message "Randalierer im Stadion / richteten einen Sachschaden / von <dmg> DM an."
* Komfort downgrade: `if rand(0, 442 - 52*Komfort) == 0 and Komfort > 1 and L == 0`:
  Komfort-- ("In der neuen Komfortbewertung wird ihr Stadion um eine Note schlechter
  beurteilt.").

---------------------------------------------------------------------------------------
## 10. Season end – licence, promotion, bonuses (f_1a58_3176 + f_0cb5_0012)
* Licence (before promotion/relegation): for each manager `flags = 0`; if
  `money < -2,000,000` and L < 2: flags = 4 ("DFB verweigert die Lizenz. Sie steigen ab."),
  the team is moved to the last table place of its league and the league gets one extra
  relegation slot; L++. No effect in the Amateur-Oberliga.
* Relegation/promotion (f_0310_32b1 per affected team): relegated from L<2 → flags 2, L++;
  promoted → flags 1, L--; relegated from the Amateur-Oberliga → flags 8, **money = 500,000**
  ("Sie beginnen neu mit 500.000 DM.") and the manager is given another team.
* f_0cb5_0012 (summary "Sie beendeten Ihre Saison"/"Ereignisse"), per manager:
  * promoted (flag 1): new L == 0 → **+1,600,000 DM**, else **+800,000 DM**. Then Trikot:
    if counter == 0 → D:0x66c[m][0] /= 10, else counter := 0 + message "Ihre Werbepartner
    erlauben es, die Verträge jederzeit zu kündigen. Die Einnahmen bleiben solange
    erhalten."; boards: if counter == 0 → income /= 10; counter := 0 for all 6.
  * top scorer of the league in own team (f_14a4_1ad5(1) result): **+250,000 DM**.
  * players leaving at contract end (f_2277_2a8f == 0): "Sie erhalten eine Ablösesumme von"
    `(marketValue(f_2277_25de)/2)/1000*1000` DM (transfer analyst).
  * youth player (uses 0x2424), "Ihre Spieler werden älter..." joke – not finance.

## 11. Christmas – f_1a58_2a06(0) (24 Dec; birthdays i=1,2 cost nothing)
`if rand(0,3) != 0: return` (screen only 25 %); for Christmas: "Frohe Weihnachten !";
`if rand(0,1) != 0: X = rand(15,85)*10000`, text "BUNDESLIGA: X DM, 2.LIGA: X/2 DM,
AMATEUR-OBERLIGA: X/3 DM (INCL. MWST.)", and every human manager pays `X / (L+1)`.

## 12. Starting state (f_08bc_2184, new game)
* money = **1,500,000 DM**; if `D:0x4a28 == 4` (displayed LEVEL 1) **1,900,000 DM**.
* Normal (endless) game: start in Amateur-Oberliga (L=2), random team; short modes
  (F:0x513c == 0x7ac/0x7c9 = end after 1 season, 0x7ae/0x7cb = 3 seasons) start in L=0.
* ticket price = (short1 + 8)*2 - 3L (normal: 10 DM); Steh = (5-L)*4000; Sitz = (2-L)*3000
  (+5000 Sitz/+10000 Steh in 1-season mode); Überdachung = (2-L)*3500; Renovierung = 4(+1);
  Komfort = 3(+1); Flutlicht/Anzeigetafel = 3/3 (1-season) or 2/1 (3-season), else 0.
* P = 50 - 20L; f_08bc_0a63; f_14a4_2cb4; min attendance record 99999.
* 1-season mode only: Trikot income 180,000, boards 1..6 45,000 each, Werbeausgaben 30,000
  (no contract counters) – the advertising menu appears disabled in this mode (UNCERTAIN).
* f_112a_000a once (rate table).
* Debug keys in the options screen (f_262f_01b8): 'g' +200,000 DM; 'a' money = 99,999,999
  and squad values 99; '0'..'5' set level.

## 13. Finance statistics – f_262f_0af7 (section "!ZUSCHAUER", "!FINANZEN")
BESUCHERZAHL = 0x2426; SCHNITT/SPIEL = 0x2426/0x237c; BENÖTIGTER SCHNITT = f_14a4_2b15;
ZUSCHAUERREKORD = 0x242a + opponent; MINUSKULISSE = 0x242e (hidden while 99999);
KONTOSTAND = money; SCHULDEN GESAMT = −Σ credit amounts; EINNAHMEN = f_14a4_2636();
AUSGABEN = f_14a4_2822() (both recomputed live, same side effects: 0x2422 is rewritten).

## 14. Function index
| function | meaning |
|---|---|
| f_112a_0a6d | daily per-manager booking: camps, rate drift, balance sum, month end, credits |
| f_14a4_2636 | monthly income |
| f_14a4_2822 | monthly expenses |
| f_14a4_2b15 | required average attendance |
| f_112a_000a | bank interest table drift |
| f_112a_0f8d(sel,lender) | Σ amounts (sel=0) / interest (sel≠0) of current mgr's credits, lender 'c' = all |
| f_112a_1039 | create credit slot |
| f_112a_119b / 23cb / 167f | credit list, bank summary, bank screen |
| f_112a_0145 / 072d | training-camp screen / effect |
| f_1a58_20b2 / 2051 | gate money / 0:2 forfeit + 200,000 fine |
| f_0f9d_11e0 | attendance (FP) |
| f_262f_2be8 / 29a8 | advertising screen |
| f_14a4_2cb4 / 2bba | sponsor offers / stadium attractiveness |
| f_08bc_0a63 | TV income + cost item 0x2420 |
| f_0000_272d | sponsor subsidy after transfer |
| f_0cb5_13bd | weekly events (riot damage, comfort downgrade, injuries…) |
| f_0cb5_0012 | season summary: promotion bonus, top-scorer bonus, sponsor resets |
| f_1a58_2a06 | Christmas packages / birthday screens |
| f_1a58_4597 | season prestige points (0xefe) |
| f_262f_0af7 | statistics screen |

## 15. Open questions / UNCERTAIN
* Exact meaning of M.0x2420 (computed like a travel/organisation cost; may wrap at 16 bit).
* Labels of stadium slots 0 and 1 (Sitz vs Steh) inferred from usage; slot 5 meaning.
* p4 of f_1a58_20b2 (gate split) and F:0x0008/0x0009 cup-round semantics.
* f_0310_1468 (team strength parts) and f_14a4_1ad5 (top-scorer) are only referenced.
* Which menu entry is gated by M.0x237b and by the 1-season mode.
