# BMP spec: players, squads, manager record

Scope: PERSON table, SQUAD slots, MANAGER record, new-game player creation, weekly
development (training, fatigue, tendency, injuries, suspensions), training camps,
youth work, lineup rules / auto-lineup / lineup strength, season-end player changes
(aging, retirement, contracts, youth promotion).

Conventions
- `rand(a,b)` = `f_076b_0cc7(a,b)` = `MSrand() % (b-a+1) + a` (inclusive). The ORDER
  of rand() calls is given wherever it matters for bit-exact reproduction.
- All divisions are C integer divisions (truncate toward zero) unless noted. Byte
  fields wrap modulo 256 when the original uses byte arithmetic (noted).
- `F:` = far data segment 0x4238, `D:` = DGROUP 0x4cb3.
- `m` = manager index (0..3), `m=4` = the TRANSFER LIST pseudo-squad (see 2.3).
- `L` = `D:0x4a28` = difficulty variable = `5 - LEVEL` (LEVEL 0..5 is what the options
  screen shows, f_262f_01b8). L=5 is the easiest, L=0 the hardest.
- `day` = `D:0x7dc` (long) = day of the season (0 at season start; season end is
  processed when `day > 322 + D:0x224c`). `D:0x224c` (byte, default 0) and `D:0x224d`
  (byte, default 0) are variant/edition constants; `D:0x224e`/`D:0x2252` (longs,
  default 1/1) are a currency display factor (displayed money = value*D:0x2252/D:0x224e).
  Treat all of them as their defaults unless you implement the variant.
- A "turn day" is a day with `(day - D:0x224c) % 7 == 0` or `(day - 4) % 7 == 0`
  (two per week, game loop f_1a58_3176 @1a58:361f..366f).

--------------------------------------------------------------------------------
## 1. Data layouts

### 1.1 PERSON record — `F:0x57dd + p*0x25`, p = 1..150 (37 bytes; p=0 unused)

The 150 persons are the complete pool of "individual" players in the game. They are
partitioned by position: p 1..20 goalkeepers, 21..63 defenders, 64..105 midfielders,
106..150 forwards (position never changes). Human squads, the transfer list and
CPU-club players (for top-scorer simulation) all reference persons.

| off | abs | size | meaning |
|---|---|---|---|
| 0x00 | 57dd | 26 | name, NUL-terminated (from MANA.DAT, 26-byte name slots) |
| 0x1a | 57f7 | 1 | age (years) |
| 0x1b | 57f8 | 1 | average of 0x1c..0x1e, only written at new game (f_322b_035c); never read. |
| 0x1c | 57f9 | 1 | stored Kondition (copied into slot +0x10 when the person joins a squad) |
| 0x1d | 57fa | 1 | stored Technik (→ slot +0x11) |
| 0x1e | 57fb | 1 | stored Form (→ slot +0x12); reduced by injuries (f_14a4_30cf) |
| 0x1f | 57fc | 1 | position code: `code/25` = position (0 TOR, 1 ABW, 2 MIT, 3 ANG), `code%25` 0..16 = depth inside the position (higher = more offensive). GK = 0 exactly. |
| 0x20 | 57fd | 1 | preferred side 0..6 (0 = far left .. 6 = far right). Displayed as "(L)" if <2... see 4.6 |
| 0x21 | 57fe | 1 | owner manager: 0..3 human manager, 5 = nobody (free/CPU). (4 not used as owner) |
| 0x22 | 57ff | 1 | league goals this season (top-scorer list); reset at season end / when joining a human squad |
| 0x23 | 5800 | 1 | league appearances this season; same resets |
| 0x24 | 5801 | 1 | club (team id 0..199), 0xff = no club |

Side display (f_0310_1c22): append " (" ; if side<5 append "L"; if side>1 { if side<5 append "+"; append "R" }; append ")".
So 0,1 → "(L)", 2..4 → "(L+R)", 5,6 → "(R)".

### 1.2 SQUAD slot — `F:0x774a + (m*0x19 + s)*0x34`, m = 0..4, s = 0..24 (52 bytes)

Each manager has 25 slots; only slots 0..23 are used (24 = overflow spare used while
shifting). m = 4 is the transfer list (only slots 0..11 used, max 12 entries).
Slots are kept compact (no gaps) and SORTED BY PERSON INDEX (hence by position GK, DEF,
MID, ATT). `count(m) = f_3091_1109(m,0)` = number of slots among the first N
(N = 24, or 12 for m=4) with person != 0.

| off | size | meaning | evidence |
|---|---|---|---|
| 0x00 | 1 | red cards this season | f_14a4_0906 row "ROTE KARTEN" |
| 0x01 | 1 | yellow cards this season | "GELBE KARTEN" |
| 0x02 | 1 | yellow-red cards this season | "GELB-ROTE K." |
| 0x03,04,05 | 3×1 | goals this season: Liga, DFB-Pokal, Europacup | f_1ecd_06af cols 9..12 |
| 0x06,07,08 | 3×1 | games this season: Liga, Pokal, EC | cols 3..6; f_1a58_20b2 |
| 0x09 | 1 | status: bits0-1 `1` = suspended, `2` = injured (count in +0x0d); bit 0x40 = CPU club interested in own player (offer msg pending), bit 0x80 = interest/bid on a transfer-list player; any of 0x20/0x40/0x80 = "offer pending" | 0cb5_13bd |
| 0x0a | 1 | shirt / lineup number: 0 = not nominated ("HÄNGT HALT SO RUM"), 1..11 = in the team, 12..15 = reserve (bench) | 1ecd_06af, 3360 |
| 0x0b | 1 | contract length remaining, in seasons | "VERTRAGSDAUER" |
| 0x0c | 1 | loan: 0 = own player; else player is on loan from team `(+0x0c & 0x7f)` (transfer screen writes team-0x80); value 0x63 has a special meaning in f_1ecd_37d8/f_1ecd_1560 (see 2.3) | 2277_2a8f "ist leider von $ nur ausgeliehen" |
| 0x0d | 1 | remaining suspension (league games) or injury (weeks) | |
| 0x0e | 1 | tendency raw 30..68; displayed class `(x-30)/13`: 0 FALLEND/"-", 1 KONSTANT/"O", 2 STEIGEND/"+" | f_1ecd_000e |
| 0x0f | 1 | person index (0 = empty slot) | |
| 0x10 | 1 | Kondition 1..99 | |
| 0x11 | 1 | Technik 1..99 | |
| 0x12 | 1 | Form (weekly development keeps it in 45..55) | |
| 0x13 | 1 | Erschöpfung (fatigue) 50/60..150; shown red when >130; normal ≈ 60..100 | |
| 0x14 | 1 | tendency drift: bits0-3 weeks left (1..13), bits4-6 step (1..3), bit7 direction (1 = up) | f_2277_000a |
| 0x15 | 1 | signed match rating of the last match (reset to 0 after each match day; +15 per goal, -8/-5 per goal conceded GK/field, ...) | f_1a58_0ca3, f_1a58_20b2 |
| 0x16 | 1 | team id of the club making an offer / interested | 0cb5_13bd |
| 0x17 | 1 | injury kind 0..17 (index into injury table 3.8) | f_14a4_30cf |
| 0x18 | 1 | contract talk state: 0 = neutral; 1..99 = weeks/turns of "not willing to negotiate" cooldown; 100+n (n=2..4) = player offered extension to n years (message pending); bit 0x80 = player announced he will NOT extend (retires at contract end) | 0cb5_13bd, 2277_2a8f |
| 0x19 | 1 | tactics board column 0..6 (left..right), -1 = not placed | f_1ecd_3360, 1266 |
| 0x1a | 1 | tactics board row 0..7 (7 = goal line, 0 = forward line) | |
| 0x1b | 1 | UNKNOWN (never seen referenced) | |
| 0x1c,1e,20 | 3×2 | career games: Liga, Pokal, EC (words) | f_14a4_0906 "SPIELE:" |
| 0x22,24,26 | 3×2 | career goals: Liga, Pokal, EC (words) | "TORE:" |
| 0x28 | 4 | monthly salary (long, DM) | col 17, f_14a4_2822 |
| 0x2c | 4 | UNKNOWN | |
| 0x30 | 4 | far pointer to the news message belonging to this slot's pending event (0 = none) | f_3091_0044 deletes it |

### 1.3 MANAGER record — `F:0x2242 + m*0x30a` (778 bytes), m = 0..3

The record starts at F:0x2242 (the name; `08bc:11d8 add ax,0x2242`, savegame block
0x2242..0x2e6a = 4×0x30a). Offsets below are relative to the record start
(absolute address for m=0 in brackets). Only player-relevant fields are certain;
others are listed with what was observed.

| off | abs | size | meaning |
|---|---|---|---|
| 0x000 | 2242 | 29 | manager name (string). Byte 0 = 0 → no name entered |
| 0x01d | 225f | 1 | UNCERTAIN (swapped between managers in f_08bc_2184; maybe face/picture) |
| 0x01e | 2260 | 1 | team id of the manager's club (100 = none during setup) |
| 0x039..0x03d | 227b..227f | 5 | UNCERTAIN (written by game loop) |
| 0x03e | 2280 | 50×4 | season history, entry k: +0 final place (0xff = unused), +1 league, +2,+3 written by game loop. Init: +0 = 0xff for k<50. Season k = D:0x7e2 (capped 49) |
| 0x109 | 234b | 1 | UNCERTAIN |
| 0x10a | 234c | 1 | UNCERTAIN (init `(scen?1:0 + 8)*2 - 3*league`) |
| 0x10b | 234d | 38 | UNCERTAIN, all init 10 |
| 0x131 | 2373 | 1 | lineup screen slider 0..34 (default 16) — "Einsatz"/aggression, UNCERTAIN name; used in strength calc 4.5 |
| 0x132 | 2374 | 5 | UNCERTAIN, init 30 each |
| 0x135 | 2377 | 1 | set to 1 in the 1964/1993 scenario |
| 0x138 | 237a | 1 | league 0=1.Liga, 1=2.Liga, 2=Amateur-Oberliga |
| 0x139 | 237b | 1 | training-camp cooldown (days); set rand(7,18) after a camp, -1 every day while != 0 (f_112a_0a6d, which runs daily per human manager, outside its month-end branch); camp menu disabled while != 0 |
| 0x13a,0x13b | 237c,237d | 2 | UNCERTAIN |
| 0x13d | 237f | 1 | match "effort" value 0..40 computed by f_0f9d_0002 (4.5); 100 = fewer than 8 players in lineup |
| 0x13e | 2380 | 1 | bit0 = riot damage pending (f_0cb5_13bd) |
| 0x13f | 2381 | 1 | Jugendarbeit (youth work) slider 0..34 (training screen, PIC 37); cost (v+1)*625 DM/month |
| 0x140 | 2382 | 1 | season-end flags: 1 promoted, 2 relegated, 4 licence refused, 8 restart with 500.000 |
| 0x141 | 2383 | 4 | training points: Kondition, Spiel, Schuss, Taktik (sum ≤ 20). Default 5,4,6,5 (D:0x286) |
| 0x145 | 2387 | 1 | training Intensität. Default 6 (D:0x28a) |
| 0x146 | 2388 | 4 | position training points: Tor, Abwehr, Mittelfeld, Angriff (sum ≤ 10). Default 0,1,3,3 (D:0x28c) |
| 0x14a | 238c | 1 | UNCERTAIN |
| 0x15e..0x191 | 23a0..23d2 | longs | stadium/finance values (0x23a0 = 3000*(2-lg)[+5000 scen], 0x23a8 = 4000*(5-lg)[+10000], 0x23b0 = 3500*(2-lg), 0x23b8, 0x23c0, 0x23c8 = 4+scen (reduced by riots), 0x23d0 = 3+scen (stadium comfort grade, reduced by events)). Not player topic |
| 0x1a4..0x1c0 | 23e6..2402 | 8×4 | written by game loop, UNCERTAIN |
| 0x1dc | 241e | 2 | UNCERTAIN (init 50-20*league; on the last day of a month, if < 95 and `rand(0,22) < D:0x68c[m]/2500` (long, D:0x68c + m*0x24): += rand(1,3); f_112a_0a6d) |
| 0x1de | 2420 | 2 | monthly cost component (f_14a4_2822), UNCERTAIN |
| 0x1e0 | 2422 | 2 | written by f_14a4_2636 (income) |
| 0x1e2 | 2424 | 2 | youth pool: += slider(0x2381)*4 on the last day of each month; used at season end (6.4) |
| 0x1e4..0x1ef | 2426..2430 | | longs; 0x242e init 99999 |
| 0x1f0 | 2432 | 4 | money (long). Init 1.500.000 (1.900.000 if L==4) |
| 0x1f4..0x1fb | 2436..243c | | written by f_1a58_20b2 |
| 0x1fc | 243e | 5×3×18 | credits (lender k=0..4, k=4 = bank, 3 each): +0 long amount, +4 long payment, +9 due day, +0xb month, +0x10 year. Ends exactly at 0x30a |

--------------------------------------------------------------------------------
## 2. Creating players and squads

### 2.1 New game: f_322b_035c (called from f_08bc_2184 on "new game")

Constants: `D:0x7a6[0..4] = {1,20,64,106,151}` (person block bounds; f_322b_035c reads
only D:0x7a7.., the add-manager pick in 2.4 also reads D:0x7a6[0] = 1),
`D:0x9438[k] = {2,5,8,5}` (squad players per position k), `D:0x943c[k] = {19,44,42,45}`
(running counters, never re-initialised), `D:0x9432[0..4] = {0,2,8,15,20}` (MANA.DAT
name slots per position: GK 0-1, ABW 2-7, MIT 8-14, ANG 15-19).

1. Load MANA.DAT into a buffer (64 team records of 23-byte name + 20 × 26-byte player
   names, record size 0x21f).
2. For every manager m=0..3, slot s=0..24: slot.person = 0.
3. For p = 1..150 (rand order exactly as listed):
   - `age = rand(18,33)`
   - `b = rand(30,89)`; if `b < 45` and `rand(0,1) != 0`: `b = rand(30,89)`
   - `T = b + 10 - rand(0,20)` → 0x1d (Technik)
   - `F = rand(45,55)` → 0x1e (Form)
   - `K = b + 10 - rand(0,20)` → 0x1c (Kondition)
   - 0x1b = (K+T+F)/3; `side = rand(0,6)` → 0x20; owner(0x21) = 5.
4. For every human manager m (0..nHumans-1), with `i` = 0..19 running across positions:
   for position k = 0..3, repeat D:0x9438[k] times:
   - `p = D:0x7a7[k] - D:0x943c[k] + 1`; `D:0x943c[k]--`
     (first manager gets GK 2,3; ABW 21..25; MIT 65..72; ANG 107..111; next manager continues)
   - name = MANA.DAT player name i of the manager's club (club = manager.team, found by
     comparing team names, f_322b_01d0). If that name is empty: random name, see 2.2
     with (k, p=0xff).
   - i++; owner = m; `age = rand(19,31)`;
     `f_1ecd_37d8(p, contract=2, loan=0, max=24)` (2.3); person.club = manager.team.
5. For p = 1..150 with an empty name (k = position group, incremented when
   `D:0x7a7[k] <= p`; NOTE off-by-one: p=20 already uses k=1): name = random name (2.2)
   with (k, p) (sets person.club); then if `F:0x513c > 2000` (normal game, see 2.4)
   person.club = 0xff (no club).
6. Position codes: p 1..20: 0; p 21..63: 25+rand(0,16); p 64..105: 50+rand(0,16);
   p 106..150: 75+rand(0,16) (in this order).

Afterwards f_08bc_2184 (per human manager, see 2.4) overwrites the squad skills.

### 2.2 Random name: f_322b_022e(buf, k, p)
loop:
1. `n = rand(D:0x9432[k], D:0x9432[k+1]-1)` (name slot)
2. repeat `t = rand(0, D:0x225e-1)` (D:0x225e = 18 = teams of league 0) until t is
   no human manager's team.
3. name = MANA.DAT name n of team t. Accept if non-empty and no person 1..150 already
   has this name (strcmp). Otherwise restart at 1.
4. if p != 0xff: person[p].club = t.

### 2.3 Adding a person to a squad: f_1ecd_37d8(p, contract, loan, ignored_long, max) → slot

Uses current manager `m = F:0x304a` (callers temporarily set it, m=4 = transfer list).
1. If `max == 12` and slot 11 of squad m is occupied → message "Schon 12 Mann auf dem
   Transfermarkt", return 0x7f.
2. If `m != 4` and `person.club != manager[m].team` and `f_112a_00b4(m) >= max` →
   "Schon 24 Mann im Team", return 0x7f.
   `f_112a_00b4(o)` = (number of persons in squad o) + (number of slots in every other
   existing squad (humans and m=4) whose person.owner == o). I.e. players a manager put
   on the transfer list still count.
3. Find insertion index s: s=0; while slot[s].person != 0 and slot[s].person < p and
   s < 24: s++. If slot[s] is occupied, shift slots 23..s down by one (slot i → i+1).
4. slot[s] := all zero bytes; `+0x0f = p`; `+0x13 = rand(80,120)`; `+0x0e = 0`;
   tendency init f_2277_000a (2.5); `+0x0b = contract`; `+0x0c = loan`;
   `+0x10/11/12 = person K/T/F`;
   `+0x28 = value(s, loan==0x63 ? 3 : 1)` (= salary, 2.6);
5. If m != 4 and person.club != manager.team: person +0x22 = +0x23 = 0,
   person.club = manager.team. (Owner is NOT set here; callers set person.owner.)
6. return s.

Removing: f_1ecd_10ee(s, force, n): if slot empty → 1. If person.owner != m and !force →
return 0. Delete the slot's pending message; copy slots s+1..n → s..n-1; slot[n].person=0.

### 2.4 Squad finishing in f_08bc_2184 (per human manager m, new game and "add manager")
- "Add manager in a running game" branch only: manager bytes 0x10b..0x130 = 10,
  0x132..0x136 = 30, history[k].place = 0xff; then for k = 0..3, D:0x9438[k] times:
  pick `p = rand(D:0x7a6[k], D:0x7a7[k])` until owner == 5 (≤1000 tries); owner = m;
  `f_1ecd_37d8(p, contract=1, 0, 24)`.
- `F:0x513c` ("scenario year"): 22222 normal game; 1964 or 1966 (option bits 2/4),
  +29 (→1993/1995) when F:0x56f4 = 1. `league = 2` (Amateur) if
  (0x513c > 1966 && !56f4) || (0x513c > 1995 && 56f4) (i.e. normal game), else 0.
- `lo = L/2 + 27` if league==2, `lo = L/2 + 68` if league==0; `lo += 12` if 0x513c is
  1964 or 1993.
- Skill reset (bug-faithful): for k = 0..3, for l = 0 .. count(manager k)-1
  (NOTE: loop bound uses squad *k*, writes squad *m*), for i = 0..2:
  `v = rand(lo, lo+5)`; slot[m][l].(0x10+i) = v; `f = rand(40,60)`; if i==2: slot.0x12 = f.
  Also writes person[X].0x1c/0x1d = v (i=0/1), 0x1e = f (i=2) for a STALE person index X
  (local [bp-4]; 0 in the add-manager branch, a UI loop counter in the new-game branch).
  Recommended: skip the person write (UNCERTAIN effect). Result: every human player has
  K,T ∈ [lo,lo+5], Form ∈ [40,60].
- Then for every slot: +0x28 = value(s, 1) (salary). Training defaults (1.3).
  `0x2373 = 16`, `0x241e = 50 - 20*league`, money as in 1.3.

### 2.5 Tendency init f_2277_000a(slot)
`+0x0e = rand(35,65)`; `+0x14 = rand(1,13)`; `+0x14 |= rand(1,3) << 4`;
`+0x14 |= rand(0,1) << 7`.

### 2.6 Value / salary formula f_2277_25de(s, flags) → long (slot of manager F:0x304a)
Coefficients `c[0..5]` = D:0x543e + 6*(flags&1):
- flags&1 == 0 (market value): `{58, 4, 4, 5, 4, 25}`
- flags&1 == 1 (monthly salary): `{70, 8, 7, 5, 8, 2}`

```
v  = ((K+T)/2) * c0 / 100                       (K=+0x10, T=+0x11)
t1 = ((g_liga+g_pokal)*30 + g_ec*40) / 100      season goals +3,+4,+5; clamp [0,20]
t2 = (50*G_liga + 20*G_pokal + 30*G_ec) / 100   career games +0x1c,+0x1e,+0x20; clamp [0,200]
v += c1*t1*5/100
v += (t2*100/200) * c2 / 100
v += c3 * tendency(+0x0e) / 100
t3 = 6*red(+0) + yellow(+1);  clamp [0,20];  v += c4*t3*5/100
t4 = 45 - age;  clamp [0,27];  v += (t4*100/27) * c5 / 100
if flags&1:                       // salary
    v = v*v*v / 1000
    x = K + T;  if x > 100: x = (x-100)/2 + 100
    if x > 135: v = v*(x-35)/100
    if v < 2: v = 2
    v = v*85
else:                             // market value
    v = v*10000;  if v > 110000: v -= 100000
    if flags&2: v = v/3
    if slot+9 & 0x80:            // bid/interest on transfer-list player
        v = (slot+0x16 > 63) ? v*130/100 : v*rand(95,100)/100
v = v*5/3
if flags&4 and !(flags&1): v += rand(v/20, v/10) - v/15     (rand bounds as 16-bit ints)
if flags&2 and !(flags&1): v = v*rand(75,95)/100
d = (flags&1) ? 100 : 1000;  return (v/d)*d
```
Usage: (s,1)/(s,3) salary at signing; (s,0) market value (transfer list, compensation,
interest checks); (s,5) salary basis in contract talks; youth: (s,1)*50/100.

--------------------------------------------------------------------------------
## 3. Weekly player development — f_0cb5_13bd

Called on every turn day (two per week) for each human manager m (game loop
1a58:367e). Whole function only runs while `day <= 321`. `B = 37 - (L==5)`.
Sequence per call (all for manager m):

### 3.1 Transfer-list housekeeping (only when m == 0) — brief
For each transfer-list slot t (m=4): if fatigue > 56: fatigue -= rand(3,9);
if rand(0,3)==0 and (flags 9 > 0x1f) and message: flags9 &= 0x1f, delete message.
Then message lifetimes F:0x1d34[k] (k = m*10..) count down; at 0 the message is deleted.
Only timed messages have a lifetime: those of `f_0000_239e` (format 2: injury in training,
riot damage, comfort downgrade, a finished construction `f_0000_20e1`) and the retirement
announcement (format 4), each 3. A manager has 10 places for them (`f_3091_0000`); with none
free the message is deleted at once and never shown. The loop counts the managers 0..2 only,
so the timed messages of manager 3 never expire (the remake counts all four). The messages a
squad or market slot points to (+0x30: an offer of a club, a request for a longer contract)
have no lifetime; they are deleted when the slot lets them go (offer expired or answered,
request withdrawn, season end).
(Followed for every m by CPU bids on transfer-list players owned by m — transfer topic:
uses value(t,0)/10000 vs rand(0,180), bit 0x80, team via f_14a4_24bf.)

### 3.2 Riot and stadium events (not player topic)

### 3.3 Per own squad slot s (in slot order) — events loop
`N = count(m)`; runs if day < 322 (always true here). For each s:
1. **Training injury.** `base = ((nomatch ? 25 : 0) - fatigue + 160) * 2`, min 40,
   where `nomatch = (D:0x2280[F:0x16e] == 0)` (no fixture of any kind today).
   If `slot+0x0d == 0` and `slot+0x09 == 0` and `rand(0, 30*L + base + 80) == 0`:
   create injury (3.8), message "Verletzung im Training:" + name, re-run auto lineup
   (f_1ecd_3360).
2. **Retirement announcement.** `e = f_14a4_2588(slot)` =
   `age * (100 - (GK?15:0) - max(0,(T-K)>>1)) / 100`. `r = rand(32,45)` (always drawn).
   If `r < e` and `!(slot+0x18 & 0x80)` and `slot+0x18 < 100`: set `+0x18 |= 0x80`,
   message type 4 "$ kündigt an, daß er seinen Vertrag nicht mehr verlängern wird."
   (stored in a free message slot with lifetime 3; if none free it is deleted at once).
3. If `slot+0x0d != 0` (suspended or injured): `slot+0x0a = 0`; re-run auto lineup.
4. If `rand(0,3) == 0` and `slot+9 > 0x1f` and message: offer expires: `slot+9 &= 0x1f`,
   delete message.
5. Contract state `q = slot+0x18`: if `q > 99 && !(q&0x80)`: if `rand(0,5)==0` and
   message: `q = rand(9,17)`, delete message (extension offer withdrawn).
   Else if `q != 0 && !(q&0x80)`: `q--`.
6. If `slot+0x0c == 0` (not on loan) — **extension offer by the player**:
   ```
   avg = (K+T+F)/3
   a   = min(8, |25 - age|);  a = 100 - (100*a)/8          (truncating)
   g   = min(5, (G_liga+G_pokal+G_ec)/36)                  (career games)
   x   = (60*avg + 40*a) / 100
   x  += (x*g*10) / -100
   if rand(0,x)==0 and rand(0,3)==0 and no message and !(q&0x80) and contract(+0x0b)==1:
       r = rand(0,100);  n = r>90 ? 4 : (r>70 ? 3 : 2)
       slot+0x18 = 100+n
       message type 0: "$ bietet an, von # auf # Jahre zu verlängern." (name, 1, n)
   ```
   (Accepting this is done in the contract screen; n is the offered length.)

### 3.4 CPU interest in own players (second loop over own slots, day < 322)
If person != 0, `+0x0c == 0`, `!(+0x18 & 0x80)`: `v = value(s,0)/10000`;
if `rand(0,200) < v` and `rand(0,450)==0` and no message: `+9 |= 0x40`;
`flag = (v >= 91 && rand(0,12)==0)`; `+0x16 = f_14a4_24bf(s, v, flag)` (club);
message type 1 "$ ist an $ interessiert." (Transfer analyst owns the rest.)

### 3.5 Training effect matrix
`P[i] = 20 * manager.train[i]` (i = Kondition, Spiel, Schuss, Taktik; 0x2383..).
`TM` = D:0x292 (signed bytes, row i, column j = K,T,F):

| training | →K | →T | →F |
|---|---|---|---|
| Kondition | 80 | 5 | 15 |
| Spiel | 30 | 40 | 30 |
| Schuss | 20 | 25 | 55 |
| Taktik | 5 | 65 | 30 |

`E[j] = (Σ_i TM[i][j] * P[i]) / 400` (int).

### 3.6 Per own slot s (all slots 0..N-1, in order) — development loop
```
f_0cb5_2b88(slot)                       // injury countdown, see 3.9
I = manager.intensity (0x2387)
if fat > 50: fat -= rand(3,7)                                  (fat = +0x13, byte)
if window and I < 6: I = 6
fat += (I-5)*2                                                 (byte add)
if fat > 130 and I < 6: fat -= rand(1,3)
W = 20*I
fat = clamp(fat, 60, 150)
if fat > 135: W -= rand(70-13L, 90-5L); if W < 10: W = 10
if position == GK: W += 30
for j in 0..2  (j=0 Kondition +0x10, 1 Technik +0x11, 2 Form +0x12):
    D = E[j]*W/100
    D = D*tendency(+0x0e)/50
    G = 10 * manager.postrain[position]          (0x2388 + pos)
    D = D*(G + 75 - rand(0,3))/100
    v = slot[0x10+j]
    R0 = (rand(0,4) == 4) ? 1 : 0;  dir = 0
    if L > 2 and window:   (window2 for this branch also includes day > 322, impossible here)
        if rand(0,2)==0: R0++
        R = rand(B-R0-1, B+2)
        fat += rand(0,2)
    else:
        R = rand(B-R0-2, B+2)
    if L < 2 and window: fat += rand(0,2)
    if D < R-8:   if rand(0,1)!=0: dir = -1; R = (R-D+8)/11
    elif D > R+8: if rand(0,1)!=0: dir = +1; R = (D-R+8)/11
    if rand(0,3)==0 and dir != 0:
        if R > 2: R = rand(2,3)
        if R != 0:
            v += rand(1,R)*dir
            v = clamp(v, 1, 99) for j<2;  clamp(v, 45, 55) for j==2 (Form)
            slot[0x10+j] = v
// tendency drift
T = +0x0e
if rand(0,3) != 0: T += (+0x14 bit7 ? +1 : -1) * ((+0x14 >> 4) & 7)
n = (+0x14 & 0x0f) - 1;  +0x14 = (+0x14 & 0xf0) + n   (byte)
if n == 0: f_2277_000a(slot)            (new random tendency; T discarded)
else:      +0x0e = clamp(T, 30, 68)
```
`window` = `130 + 7*D:0x224d < day < 207 - 14*D:0x224d` (default 131..206, the winter
break). After all slots: auto lineup f_1ecd_3360.

Interpretation: D ≈ 1.2×E for intensity 6; with B=37 a value must exceed ~45 to
improve and fall below ~29 to decline; each turn day changes a value by at most ±3
with probability ≤ 1/8.

### 3.7 Fatigue from matches — f_1a58_20b2 (after each match day, type 0 Liga, 1 Pokal, 2 EC)
For each participating human manager m: `D:0x664[m] = 11`. For each slot s < count:
- if `(+9 & 1)` (suspended) and `+0x0d > 0` and type == Liga: `+0x0d--` (suspensions
  only count league matches; transfer-list slots owned by m also count down).
- `+0x15 = 0`.
- if in lineup (+0x0a 1..11): `fat += 6 + rand(2,4) - (personK > personT ? 1 : 0)`
  (person +0x1c vs +0x1d); if Liga: person+0x23++; season games `+(6+type)++`,
  career games `word +(0x1c+2*type)++`.
- every slot: `fat = clamp(fat, 50, 150)`.

### 3.8 Injury creation f_14a4_30cf(slot)
```
if person.F(+0x1e) > 20: person.F -= rand(12,19)
k = rand(0,17)
if rand(0,1) != 0 and rand(2,4) < DUR[k]: k = rand(0,17)
slot+0x0d = DUR[k] + rand(0, DUR[k]/4 + 1)     (weeks)
slot+0x09 |= 2;  slot+0x17 = k;  slot+0x0a = 0
```
`DUR` = D:0x5bc = {1,2,1,2,2,4,2,1,1,6,12,5,7,10,11,9,20,14}; names D:0x23b4 table:
Oberschenkelzerrung, Prellung d. Knöchels, Platzwunde am Kopf, Stauchung d.
Fußgelenkes, Sprunggelenkprellung, Bänderdehnung, Bauchmuskelzerrung, Bluterguss am
Knie, Nasenbeinbruch, Außenbandanriß, Schienbeinbruch, Muskelfaserriß,
Meniskusverletzung, Innenbandabriß, Wadenbeinbruch, Kapselriß, Leistenbruch,
Kreuzbandriß. (Match injuries also call this, 1a58:1d9b.)

### 3.9 Injury/suspension countdown f_0cb5_2b88(slot)
If `(+9 & 2)` and `+0x0d > 0` and `(day - D:0x224c) % 7 == 0`: `+0x0d--` (one week).
Then if `(+9 & 3) != 0` and `+0x0d == 0`: `+9 &= 0xfc` (fit / suspension over), auto
lineup. Called from the development loop, for transfer-list slots owned by m, and daily
for all slots during the off-season (1a58:4434).

Availability rule everywhere (auto lineup, picker): status 0, or status 1 (suspended)
while `F:0x513e != 0` (set on DFB-Pokal / Europapokal days: league suspensions do not
apply to cup matches).

### 3.10 Training camp f_112a_072d(c), c = 0..7 (menu f_112a_0145)
Camps (D:0x2394): Kickfelder Mühle, Trlg. Hackstedt, Hof Schweißperle, Fitnesscenter
Krone, Scola Ludum Pilae, Sportschule Träne, Hotel Medium, Luxushotel La Preux.
Price `D:0x5258[c]` (long) = factor × count(m) × 2000 with factors {5,6,7,10,11,10,13,15}
(converted once when the first entry < 10). Availability: `D:0x620[c]` counter (≥1 =
closed); every call of f_112a_0a6d (daily, once per human manager, so nHumans steps per
day) each counter moves toward 0 by 1 and when it hits 0 is reset to `±rand(20,70)` (sign
flips). Initial {-80,-50,-100,-60,80,50,70,60}. Needs money ≥ price,
manager.0x237b == 0 (menu) and one camp per visit.
Params `CP` = D:0x527e + 5c (bytes 0..3 used): {1,2,1,1},{1,2,3,1},{1,3,2,1},{2,4,3,2},
{2,1,2,4},{2,4,3,2},{3,2,2,3},{5,3,2,4}; matrix `CM` = D:0x52a6 rows
{20,20,40},{60,10,20},{20,50,20},{10,40,30}.
```
C[j] = (Σ_{i=0..3} CP[i]*CM[i][j]) / 5            j = 0..2
for each slot s < count:
    F = (C0+C1+C2)/-3 + (C0-C2+10)*2 + C1;  if F < 80: F = 80
    fat += rand(F/9, F/6);  fat = clamp(fat, 60, 150)
    for j in 0..2:
        x = (rand(2L+87, 2L+117) + C[j] - 100) / 10
        a = rand(0, x*20/30);  b = rand(0, x);  x = x + b - a
        slot[0x10+j] += x >> 2  (arithmetic shift, byte add);  if > 99: = 99
    x += x*C2/200            (x of j=2; 112a:0995 imul [bp-0x16] = C[2])
    if rand(0,4)==0: +0x0e += rand(155,300)*x/800
    f_2277_000a(slot)        (re-rolls tendency; makes the line above ineffective)
manager.0x237b = rand(7,18)
```

### 3.11 Youth work slider
`manager.0x2381` (0..34) set in the training screen; costs (v+1)*625 DM per month
(f_14a4_2822); on the last day of every month `manager.0x2424 += v*4` (f_112a_0a6d).
Effect at season end: 6.4.

--------------------------------------------------------------------------------
## 4. Lineup

### 4.1 Lineup representation
In team: `+0x0a` 1..11; reserves 12..`F:0x56ee`; `F:0x56ee` = 15 normally
(13 if D:0x224e == 4; always 15 on Europapokal days) → max reserves = 0x56ee-11 (4).
Tactics board: `+0x19` column 0..6, `+0x1a` row 0..7. f_1ecd_24cd renumbers lineup
players 1..n in slot order (reserves keep their numbers).

### 4.2 Manual rules (f_1ecd_1560; `N` = 11, or the given number during a match)
- Taking a player OUT of the team while `count(in team) < N` → "Nicht weniger als
  (N-1) Spieler einsetzen!" (so the lineup may temporarily be N-1 while swapping).
- Putting a player INTO the team while `count(in team) >= N` → "Nicht mehr als N
  Spieler einsetzen!".
- Making a player reserve when the next reserve number > F:0x56ee → "Maximal
  (0x56ee-11) Reservespieler!".
- During a match: substitution only if `F:0x5396[m*2 + (GK?0:1)] != 0` else "Keine
  Auswechslung mehr möglich"; "Spieler ist nicht aufgestellt" for players not on the pitch.
- Any manual change while auto lineup is on (D:0x79e[2m] > 1) → message
  "Auto-Aufstellung ist deaktiviert !" and D:0x79e[2m] = 1 (f_1ecd_14c7).

### 4.3 Auto lineup f_1ecd_3360 (mode = D:0x79e[2m]; 1 = manual → nothing)
Formation f = mode-2, D:0x541c: f0 = {1,4,4,2}, f1 = {1,3,5,2}, f2 = {1,3,4,3}
(GK,ABW,MIT,ANG). Row per position D:0x542e = {7,6,3,0}; start column by count
D:0x5428[n] = {0,3,2,1,0,0} (n players in the line).
```
all slots 0..23: +0x0a = 0;  col[k] = 0;  shirt = 1
for k in 0..3: while need[f][k] > 0:
    s = pick(k); if s == -1: break
    +0x0a = shirt++;  need--;  +0x1a = ROW[k];  +0x19 = START[cnt_f_k] + col[k]
    if f == 0 and shirt == 3: row--, col++       (2nd placed player = 1st defender)
    if f == 0 and shirt == 6: row--, col--       (4th defender)
    col[k] += 2; if cnt_f_k == 5 and col[k] not in {2,6}: col[k]--
reserves:
  if F:0x56ee == 15: for k 0..3: s = pick(k); if -1 break; +0x0a = 12,13,14,15
  else: GK reserve 12 = pick(0); 13 = best remaining field player (the pick among
        positions 1..3 that raised the running maximum average)  — UNCERTAIN detail
f_0cb5_25d5 (swap lineup players to better match preferred side), renumber.
```
pick(k) = f_1ecd_3635: among slots 0..23 with person, `+0x0a == 0` and available
(3.9), choose the one with the highest (K+T+F)/3 among players of position k (ties →
later slot). If none, try the next position (k+1 cyclic; −1 cyclic for k==3, and for
k==2 when mode < 4). A goalkeeper offered for a field position gets score −1 (first GK) /
1 (further GKs). Loop end (likely): stop as soon as a slot is chosen; return −1 when a pass
finds no available player of any position; also stop when back at k after a goalkeeper
was seen in a field-position pass. It does NOT stop merely because it is back at k.

### 4.4 Ineffective / side helpers
`sideDiff(s) = |person.side − col|` (f_0310_1bd7).
`posDiff(s) = |code − ((7−row)*75/7 + 5)|` (f_0310_1cd3); expected code by row
7..0: 5,15,26,37,47,58,69,80.

### 4.5 Lineup strength f_0f9d_0002(m, detailed)
Called after lineup changes (detailed=0) and before matches (detailed=1). Lines:
`line = D:0x2a2[7-row]`, D:0x2a2 = {0,0,0,1,1,1,2,2} (rows 7..5 defence, 4..2 midfield,
1..0 attack). Arrays per line i: Ksum, Tsum (init 8), Fsum, N, Pen (side penalty);
balance `Bal = {-9,-8,-5}`; `wing = 0`; `gk = 1`; `kt = 0`; `cnt = 0`.
For each slot in team (+0x0a 1..11): cnt++; r = 7-row; i = line(r).
- if detailed:
  - col 1 or 5: wing += 1; col 0 or 6: wing += 2.
  - r < 3: Bal[0] += 2; if r==1 and col==3 and sideDiff<2 and K < T-10: Tsum[i] += 15;
    if r==0 and col==3: gk = (position==GK) ? 0 : 2.
  - r ≥ 3: r==3: Bal[0]++, Bal[1]++, and if 1<col<5 and sideDiff<2 and K < T-10:
    Tsum[i] += 15; r 4,5: Bal[1] += 2; r==6: Bal[1]++, Bal[2]++; r==7: Bal[2] += 2;
    then if sideDiff > 2: Pen[i] += rand(2,6); if posDiff > 25: Pen[i] += rand(2,6).
  - kt += K−T; Ksum[i] += games_liga/6 − fatigue/20 + 3; Tsum[i] += goals_liga − 1.
- always: Ksum[i] += K; Tsum[i] += T; Fsum[i] += F; N[i]++.
After loop:
```
Tsum[i] += (L-5)*Pen[i]*20/100
if cnt == 0: Tsum[0..2] = 0
manager.0x237f = (cnt < 8) ? 100 : 0
pad: while cnt != 0 and cnt < 11: N[j]++ (j = 0,1,2,0,...), cnt++
if detailed:
   if L == 0 and wing > 8: Tsum[2] -= 100*N[2]
   if gk != 0: g = gk-1
       Tsum[0] = Tsum[0]*rand(10g+5, 20g+5)/100
       Tsum[1] = Tsum[1]*rand(10,40)/100
       Ksum[0] = Ksum[0]*rand(5(g+2), 30g+20)/100
   clamp Tsum[0], Tsum[1], Ksum[0] to [0,32000]
   if Bal[1] < 1: { if Bal[0] > 0: Bal[2] += Bal[1]-1;  if Bal[2] > 0: Bal[0] += Bal[1]-1 }
   if Bal[0] < -1 and rand(7,10) > Bal[0]+10: Bal[1] -= rand(3,5)
   for i: Tsum[i] += L
          if Ksum[i] != 0: Ksum[i] += manager.0x2373 - 16
          if Tsum[i] != 0 and L < 4: Tsum[i] += (F:0x90c6[m]*3 + N[i]*Bal[i])*10
                                      Ksum[i] += N[i]*Bal[i]*5
          clamp Tsum[i], Ksum[i] ≥ 0
   kt = clamp(kt/9, -4, 4); if kt < 0: kt = 0; kt = clamp(kt + manager.0x2373, 0, 40)
   if manager.0x237f != 100: manager.0x237f = kt
team = manager.team
for i: if N[i] <= 1: team.str[i] (F:0x307e+t*0x22+i) = team.0x3081+i = team.0x3084+i = 0
       else: 0x307e+i = Ksum[i]/N[i]; 0x3081+i = Tsum[i]/N[i]; 0x3084+i = Fsum[i]/N[i]
```
So the three TEAM strength triples are, per line (defence, midfield, attack): average
Kondition, average Technik, average Form. CPU clubs get them from MANA.DAT. F:0x90c6[m]
is UNCERTAIN (per-manager byte).

### 4.6 Player info screen f_14a4_0906 (for completeness)
"Info über NAME": "NAME IST n JAHRE ALT, (label)"; label index = (age−18)/5 (signed),
if > 3 (unsigned) → 3: 0 "(GRÜNSCHNABEL)" ≤22, 1 "(NOCH GANZ FRISCH)" 23–27,
2 "(MITTELALTER)" 28–32, 3 "(FAST AM ENDE)" ≥33. Status texts; TORE/SPIELE season/career
per competition; rows Kondition, Technik, Form, red, yellow, yellow-red, side, salary,
contract; tendency bar `(x−30)*100/38` (cap 94); fatigue bar `(fat−50)*94/100`.
Squad list columns (f_1ecd_06af, widths D:0x7c4 = {11,18,57,13,12,12,14,36,12,14,12,12,
13,27,40,13,25,33}): number, position, name, games L/P/E/total, "K T F", avg, goals
L/P/E/total, cards "yellow red"(+1,+0), status, tendency, contract, salary.

--------------------------------------------------------------------------------
## 5. Contracts and negotiation

### 5.1 Contract talks f_2277_2a8f(s, ...) (interactive)
Refused when: `+0x0c` loan ("ist leider von $ nur ausgeliehen"), `+0x18 & 0x80`
("wird sich mit Ablauf des Vertrages zur Ruhe setzen"), `0 < +0x18 < 100` ("ist im
Moment nicht verhandlungsbereit"), years > 4 ("Wer wird sich denn SO lange
verpflichten?"). Displayed demand for Y years:
```
V  = value(s,5)                                  (salary formula)
f  = (-100*MD[league])/MDS[league] + 100         MD = D:0x225a[l] (current matchday, UNCERTAIN), MDS = D:0x2262 = {34,38,38}
g  = f + (Y-1)*100
h  = ((g/6 + 122) << 3) / 10
demand = V*(h+4)/100;  if day > 321: demand = demand*(152-age)/100
demand = max(demand, current salary)
```
Acceptance f_2277_2270(s, Y, S) (S = offered salary):
1. Y == current years and S > current salary → accept; S < current → reject.
2. Y < current years: if cur*120/100 < S → accept; if cur*105/100 ≥ S → reject.
3. Else: `h' = ((g/6 + 110) << 3)/10`; `D = V*rand(h'-6, h'+4)/100`;
   if day > 321: `D = D*rand(132-age, 152-age)/100`;
   accept iff `D < 100000` and `S > D`.
On accept: +0x0b = Y, +0x28 = S. After any completed talk: delete offer message if
`+0x18 > 99`; `+0x18 = rand(10,18)`. A reduced offer with longer years than now:
"So dumm ist $ leider nicht...". Rejection: "$ ist nicht an Ihrem Angebot interessiert."

--------------------------------------------------------------------------------
## 6. Season end — f_0cb5_0012 (game loop 1a58:43b5, after the season summary)

Order: for m = 0..nHumans-1 { summary screen & money events; 6.4 youth; if m==0: 6.1
aging; 6.2 retire/regenerate }. Then: 6.3 reset season stats; 6.5 contract expiry
dialogs. `D:0x5d4` = "Spiele automatisch weiter" (auto-play) flag.

### 6.1 Aging (once, during m==0) — for p = 1..150 in order
`age++`. If p is in any squad (f_3091_17b7 → holder h, slot s): delete its message;
`+9 &= 0x3f`; if D:0x5d4 == 0: `+0x0b--` (contract) else `age = rand(18,25)`,
`+0x18 = 0`. If `person.owner != h` (player on the transfer list, h=4): if owner < 4,
re-insert into the owner's squad (f_1ecd_37d8(p,1,0,max=25)), copy the whole old slot
over the new slot, owner unchanged; else owner = 5; remove from h
(f_1ecd_10ee(s, force, h==4 ? 12 : 24)); new slot `+0x18 = 0`, `+0x0c = 0`.

### 6.2 Retirement & regeneration (for each m) — for p = 1..150
- if m == nHumans-1: person +0x22 = +0x23 = 0.
- find slot (h, s). Only if not in a squad, or h == m:
  - `r = rand(32,34)`; retire if (not in a squad and age > r) or (in a squad and
    `+0x0b == 0` and `+0x18 & 0x80`). (Not in a squad and age ≤ r → the original reads
    garbage via a NULL pointer; treat as "no retirement" — UNCERTAIN.)
  - Retire: if in a squad: message type 3 "$ hängt den Fußballjob im Alter von #
    Jahren an den Nagel." (shown immediately), slot removed (force), owner = 5.
    Then regenerate the SAME person record (name, position, side kept):
    `age = rand(18,25)`; `b = rand(30,92)`; `K = rand(b-5,b+5)`; `rand(0,6)` is written
    to the side byte of a STALE person index (bug, [bp-0x7c]); `T = rand(b-5,b+5)`.
    Form/club unchanged. Free persons are rolled once per human manager.

### 6.3 Season stats reset
All human slots 0..23: +0..+2 (cards), +3..+5 (goals), +6..+8 (games) = 0.

### 6.4 Youth player (per m, inside the per-manager pass)
Only if `(F:0xa7a0 > 1966 && !F:0x56f4) || (F:0xa7a0 > 1995 && F:0x56f4)` (season year;
i.e. not in the first 3 seasons).
```
Y = manager.0x2424 / 12;  Y = Y + Y/-3;  manager.0x2424 = Y*12
if Y > 30 and rand(0,2)==0 and f_112a_00b4(m) < 23:
    Y >>= 1;  message "Ein Jugendspieler steigt in den Kader auf."
    repeat p = rand(1,150) until person.owner == 5
    Y = clamp(Y, 10, 90)
    side = rand(0,6); age = rand(17,19); F = rand(45,55); T = rand(Y-3,Y+3); K = rand(Y-3,Y+3)
    club = m (manager INDEX, bug, 0cb5:0505);  s = f_1ecd_37d8(p, contract=0, 0, 24);  owner = m
    (f_1ecd_37d8 then finds club != manager.team: resets person +0x22/+0x23, club = manager.team)
    +0x28 = value(s,1)*50/100;  +0x0b = rand(2,3)
```
(Position comes from the chosen person's block, so it is random weighted by block size.)

### 6.5 Expiring contracts (if D:0x5d4 == 0), per m
Show the squad list if any slot has `+0x0b == 0`. For each such slot (removal shifts
the list): contract talks f_2277_2a8f (5.1). If no new contract: "NAME kehrt Ihrem
Verein den Rücken..." and compensation `((value(s,0)/2)/1000)*1000` is added to money;
slot removed (owner stays as is — UNCERTAIN, f_1ecd_10ee does not reset owner). If the
contract was renewed: "bleibt Ihnen auch die nächste Saison erhalten."

### 6.6 Off-season
Daily until the new season: f_112a_0a6d per manager and f_0cb5_2b88 for every slot
(injuries heal at one week per 7 days).

--------------------------------------------------------------------------------
## 7. Function list

| function | meaning |
|---|---|
| f_322b_035c | new game: person table + human squads |
| f_322b_01d0 | MANA.DAT player name n of team t |
| f_322b_022e | random unused name from a non-human 1.Liga club |
| f_08bc_2184 | options/new game: manager init, squad skill reset, salaries, training defaults |
| f_1ecd_37d8 | add person to squad m (sorted insert, init slot) |
| f_1ecd_10ee | remove slot |
| f_112a_00b4 | number of players belonging to manager o (incl. on transfer list) |
| f_3091_1109 | count(m, mode 0 all / 1 in team / 2 reserve) |
| f_3091_17b7 | find slot of person p in any squad |
| f_2277_000a | random tendency |
| f_2277_25de | market value / salary |
| f_0cb5_13bd | twice-weekly events + development |
| f_0cb5_2b88 | injury countdown / status clear |
| f_14a4_2588 | "effective age" for retirement announcement |
| f_14a4_30cf | create injury |
| f_112a_274c | training screen (UI only: points, intensity, youth slider) |
| f_2277_01a9 | free training points (20 − Σ / 10 − Σ) |
| f_0f9d_1122 | squad K/T/fatigue averages for the training screen |
| f_112a_0145 / f_112a_072d | training camp menu / effect |
| f_112a_0a6d | daily upkeep per human manager (camp cooldown, camp availability; month end: youth pool, money) |
| f_1a58_20b2 | after match day: fatigue, games, suspension countdown |
| f_1ecd_1560 | lineup screen (rules) ; f_1ecd_14c7 disable auto lineup |
| f_1ecd_3360 / f_1ecd_3635 | auto lineup / best available player of position |
| f_0cb5_25d5 | swap lineup players to fit preferred sides (not decoded in detail) |
| f_1ecd_24cd | renumber lineup |
| f_0f9d_0002 | lineup → team strength triples |
| f_0310_1bd7 / f_0310_1cd3 | side mismatch / position mismatch |
| f_1ecd_000e / f_1ecd_06af / f_14a4_0906 / f_1ecd_2897 | squad table tooltip / squad table / player info / short info |
| f_2277_2a8f / f_2277_2270 | contract talks / acceptance |
| f_0cb5_0012 | season end players |
| f_3091_0190 | build news message from template D:0x267c (types 0..5) |
| f_262f_2551 | sponsor contract display ("VERTRAG"/"KÜNDBAR"), not players |
| f_262f_01b8 | options screen; cheats: 's' all tendency 68, fatigue 50; 'a' also K/T/F 99 and money 99.999.999; 'g' +200.000 |

## 8. Open questions / UNCERTAIN
- Meaning of manager bytes 0x01d, 0x10a, 0x10b..0x130, 0x132..0x136, 0x13a/b, 0x1dc,
  0x1de; F:0x90c6[m]; slot +0x1b, +0x2c.
- D:0x225a semantics (assumed current matchday per league) in contract formulas.
- Stale-index writes (2.4 person base skills, 6.2 side byte) and the NULL read in 6.2.
- Reserve #13 choice in auto lineup when F:0x56ee != 15.
- Match-time effects (cards, ratings, in-match injuries, Form changes from matches) are
  in the match engine and not covered here; Form is otherwise only moved by development
  (45..55), camps and new contracts.
