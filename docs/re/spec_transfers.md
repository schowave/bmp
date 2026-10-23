# Spec: Transfer market, player value, offers, contracts, loans (BMMAIN.EXE)

All claims cite functions. "C" = Ghidra output (often wrong for 32-bit math in this segment),
all formulas below were re-derived from the disassembly (`BMMAIN.lst`) unless marked.
Integer semantics: `/` is C truncating division (operands are non-negative almost everywhere,
so floor). `rand(a,b)` = `f_076b_0cc7(a,b)` = `rand()%(b-a+1)+a` (MS C LCG). The ORDER of rand
calls is given where it matters for bit-exact RNG reproduction.

Long helpers (verified): `f_3a01_1ae6` = 32-bit mul, `f_3a01_1a4c` = signed 32-bit div,
`f_3a01_1c10` = unsigned 32-bit div, `f_3a01_1b18` = signed 32-bit mod, `f_3a01_1bd0(p,x)` = `*p /= x`,
`f_3a01_1bf0(p,x)` = `*p *= x`, `f_3a01_1bb8` = shl dx:ax by cl, `f_3a01_1bc4` = sar dx:ax by cl,
`f_3091_1009(p,lo,hi)` = clamp int `*p` to [lo,hi], `f_3091_10d5(a,b)` = |a-b| (long),
`f_3091_0044(ptr)` = delete a news/inbox message (far ptr), returns 0.

---------------------------------------------------------------------------------------------
## 1. Data model

### 1.1 "Squads" — including the transfer market as pseudo manager 4
`SLOT(m,s) = F:0x774a + (m*25 + s)*0x34`, m = 0..3 human managers, **m = 4 is the transfer
market** (`F:0x8b9a + s*0x34`, s = 0..11). Squads use slots 0..23 (24 players), the market 0..11.
Many routines temporarily set `F:0x304a := 4` to operate on the market and restore it from
`F:0x5358` afterwards.

`f_3091_1109(m,0)` = number of occupied slots (`+0x0f != 0`) among the first 24 (12 for m=4).
Slots are kept **sorted by person index** (`f_1ecd_37d8` inserts in order) and compacted on removal
(`f_1ecd_10ee`).

Squad slot fields relevant here (offset, size):

| off | sz | meaning |
|---|---|---|
| +0x00..+0x02 | 3×1 | per-season counters "A" (3 competitions: league/DFB-Pokal/Europa). Reset at season end (f_0cb5_0012). Used in value as `6*A0 + A1`. UNCERTAIN meaning (goals?) |
| +0x03..+0x05 | 3×1 | per-season counters "B" (same 3 competitions). Reset at season end. UNCERTAIN |
| +0x03 (market slot only) | 1 | **rejection mask**: bit m (1<<manager) = manager m's bid was rejected; bit 0x80 = "some rejection active" |
| +0x06..+0x08 | 3×1 | per-season games (league = +0x06, shown as "LIGASPIELE" in f_2277_2a8f) |
| +0x09 | 1 | status bits: &3 injury/suspension (see CONTEXT), **0x40 = AI club offer for this squad player pending**, **0x80 = AI club offer for a player the human listed on the market** (in the market slot). (In f_2277_25de, 0x80 on a squad slot is read as "on loan" – see §3) |
| +0x0a | 1 | lineup position (1..11 team, 0 none, >=12 reserve) |
| +0x0b | 1 | **contract length remaining (seasons)**. 0 = expired (renegotiate at season end) |
| +0x0c | 1 | **loan flag**: 0 = own player; `0x80 | originTeam` = loaned in (originTeam = team id he came from). 0x63 is used as a *parameter* value meaning "loan" in f_1ecd_37d8 |
| +0x0d | 1 | games left for suspension/injury |
| +0x0e | 1 | rand(35..65) at creation (f_2277_000a); re-rolled every 1..13 weeks; enters value |
| +0x0f | 1 | person index (1..150) into PERSON table |
| +0x10,+0x11,+0x12 | 3×1 | skills copied from person +0x1c,+0x1d,+0x1e (`F:0x57f9/57fa/57fb`). Strength shown = (s10+s11+s12)/3 |
| +0x13 | 1 | rand(80..120) at creation (fitness/exhaustion-like, not used by transfers) |
| +0x14 | 1 | f_2277_000a: `rand(1,13) | rand(1,3)<<4 | rand(0,1)<<7` (low nibble = weeks until +0x0e re-roll) |
| +0x16 | 1 | team id of the AI club making an offer (set with +9 0x40/0x80) |
| +0x17 | 1 | injury kind |
| +0x18 | 1 | **negotiation state**: 0 = willing; 1..99 = weeks until willing (decremented weekly); 100+N (N=2..4) = player *asks* for an extension of exactly N seasons; bit 0x80 = retires when contract ends |
| +0x1c,+0x1e,+0x20 | 3×2 | word counters (not reset at season end). UNCERTAIN meaning (career stats). Enter value |
| +0x28 | 4 | **long: monthly salary** for squad players; **asking price (Ablöse)** for AI-owned market players; for players a human listed on the market it still holds the salary (copied slot) |
| +0x30 | 4 | far ptr to the news message belonging to a pending offer/request (0 = none) |

### 1.2 Person table (`F:0x57dd + p*0x25`, p = 1..150)
`+0x1a (57f7)` age, `+0x1c/+0x1d/+0x1e (57f9/57fa/57fb)` skills, `+0x21 (57fe)` **owner**:
0..3 = human manager, **4 = transfer market (AI-offered)**, **5 = free / at an AI club**.
`+0x24 (5801)` team id (0..63 German, 64..199 foreign; 0xff = none).

### 1.3 Manager record (`F:0x2242 + m*0x30a`)
`+0x1e (0x2260)` team id; `+0x138 (0x237a)` league 0..2; `+0x1f0/+0x1f2 (0x2432/0x2434)`
**signed long account balance (DM)**.

### 1.4 Currency (D segment)
`D:0x224d` flag "currency prefix" (string at `D:0x4f78`), `D:0x224e` (long) and `D:0x2252` (long)
conversion factors, default 1 and 1 (bytes `01 00 00 00 01 00 00 00`).
Display value = `DM * [0x2252] / [0x224e]` (signed div; salaries use unsigned div `1c10`).
Entered value → DM = `input * [0x224e] / [0x2252]`. All internal amounts are DM.

### 1.5 Constant tables (dumped from BMMAIN.bin)
* `D:0x543e` value weights, signed bytes, 2 rows × 6:
  * row 0 (transfer value): **W = [58, 4, 4, 5, 4, 25]**
  * row 1 (salary):         **W = [70, 8, 7, 5, 8, 2]**
* `D:0x225a[3]` = current matchday per league (init `01 01 01`). (UNCERTAIN name, consistent with use)
* `D:0x2262[3]` = matchdays per league **[34, 38, 38]**.
* `D:0x225e[3]` teams per league **[18, 20, 20]**; `D:0x226e[3]` first team id **[0, 18, 38]**;
  `D:0x2272[2l],[2l+1]` team id range **(0,17), (18,37), (38,57)**.
* `D:0x224c` = 0 (day offset; day counter `D:0x7dc` long). "Season-end phase" in this code is
  `day > 0x141` (321); the game loop runs the season end when `day >= D:0x224c + 0x142`.

---------------------------------------------------------------------------------------------
## 2. Who is where (player pool)
* AI clubs have no rosters: a person "belongs" to a club only via `person.team`.
* Free pool = persons with `owner == 5`. Market = owner 4 (and in a market slot). Human squads =
  owner m. A player a human listed keeps owner m while sitting in a market slot.
* Loaned-in players from the market keep owner 4 (loan from another human: keep that owner).

---------------------------------------------------------------------------------------------
## 3. Player value / salary function `f_2277_25de(slot s, flags)` → long DM

Operates on `SLOT(F:0x304a, s)`. `mode = flags & 1` (0 = transfer value, 1 = salary), W = table row
`mode`. Steps (all exact):

1. `A = ((s10 + s11)/2 * W[0]) / 100`                         (long)
2. `B = ((s03 + s04)*30 + s05*40) / 100`; clamp B to [0,20]
3. `C = (w1e*20 + w20*30 + w1c*50) / 100` (unsigned 16×16→32); clamp C to [0,200]
4. `A += (W[1]*B) * 5 / 100`
5. `A += ((C*100/200) * W[2]) / 100`
6. `A += (s0e * W[3]) / 100`
7. `D = 6*s00 + s01`; clamp [0,20]; `A += (W[4]*D) * 5 / 100`
8. `E = 45 - age`; clamp [0,27]; `A += ((E*100/27) * W[5]) / 100`
   (A is now a 0..~100 "rating")
9. **mode 1 (salary):**
   * `A = A*A*A / 1000`
   * `T = s10 + s11`; if `T > 100`: `T = (T-100)/2 + 100` (arith. shift); if `T > 135`: `A = A*(T-35)/100`
   * if `A < 2`: `A = 2`; `A = A*85`
   **mode 0 (value):**
   * `A = A*10000`; if `A > 110000`: `A -= 100000`
   * if `flags & 2`: `A /= 3`   (never used with mode 0 by any caller)
   * if `s09 & 0x80`: if `s16 >= 0x40`: `A = A*130/100` else `A = A*rand(95,100)/100`
     (intended "loaned from a foreign club"; on a market slot 0x80 means "AI offer pending" and s16 is
     the bidding team — the rule is applied anyway)
10. `A = A*5/3`
11. mode 0 and `flags & 4`: `A += rand(A/20, A/10) - A/15`  (rand bounds are ints of the long)
12. mode 0 and `flags & 2`: `A = A*rand(75,95)/100` (unused)
13. Round down: `A = (A/1000)*1000` (mode 0) or `(A/100)*100` (mode 1). Return A.

Callers / flag values: 0 = current value (no noise); 4 = asking price for new market players;
1 or 3 = salary at squad insertion (f_1ecd_37d8); 5 = salary demand base (contract talks).
Example: s10=s11=70, age 25, s0e=50, counters 0 → value 833 000 DM, salary 19 800 DM/month.

Other users: f_322b_2a2a sums mode-0 values of the squad for the manager score (§13);
f_14a4_1798 calls it with a *person* index as slot (bug, see §4.4).

---------------------------------------------------------------------------------------------
## 4. Filling / refreshing the market

### 4.1 When
* New game: `f_08bc_0418` sets `day = D:0x224c` and calls `f_2277_1e38` once.
* Game loop `f_1a58_3176` (1a58:3b65): in each human manager's turn (the branch that calls
  `f_08bc_0b5f(F:0x16e)`), **if `rand(0, H+3) == 0`** (H = number of human managers `D:0x7ab`)
  the market is refreshed with `f_2277_1e38` before the manager's menu.

### 4.2 `f_2277_1e38` refresh (as manager 4)
1. `kept = 0`. For `i = 0..11`: `p = market[i].person`; `r = f_1ecd_10ee(i, 0, 12)`;
   if r != 0: `person[p].owner = 5` else `kept++`.
   * `f_1ecd_10ee(i,force,limit)`: empty slot → returns 1; person owned by someone else than the
     current manager (4) and `force==0` → returns 0 (kept); otherwise delete message `+0x30`,
     shift slots i+1..limit-1 down by one, clear `slot[limit].person`, return 1.
   * Effect: AI players are removed, human-listed players are kept. **Faithful quirk:** because
     i increments after a shift, the player shifted into i is skipped → roughly every 2nd AI player
     survives.
2. `N = 12 - kept`; if N == 0 stop. `N -= rand(0, N)`.
3. `j = 0; while j < N:`
   1. `k = rand(0,11)`, `p = market[k].person`.
   2. If `p != 0` and `owner(p) != 4`: `j++`, continue (human-listed slot hit; iteration wasted).
   3. If `p != 0` (an AI player): `f_1ecd_10ee(k,1,12)` (remove); `owner(p) = 5`.
   4. Pick newcomer: repeat `q = rand(1,150)` until `owner(q) == 5` (max 1500 tries; after that
      the last q is used regardless).
   5. `person[q].s3 (57fb) = rand(45,55)`; `t = person[q].team`.
   6. If t is a human manager's team (`f_0310_32b1(t,-1) != 0`): find league `l` with
      `226e[l] <= t < 226e[l]+225e[l]` (l = 0..2; none → l = 2); then repeat
      `t = rand(2272[2l], 2273[2l])` until t is not a human team. **Quirk:** the league search
      reuses the loop counter: afterwards `j = l` (then step 11 increments it).
   7. `s1 = rand(0,14) + team[t].str2cur (F:0x307f+t*0x22) - 10`, clamp [10,99] → `person.57f9`.
   8. `s2 = rand(0,14) + team[t].str2base (F:0x3082+t*0x22) - 10`; if `rand(0,4)==0`:
      `s2 += rand(12,20)`; clamp [10,99] → `person.57fa`.
   9. `slot = f_1ecd_37d8(q, years=0, loan=0, 0, 0, limit=12)` (inserts into the market, see §5.6);
      `owner(q) = 4`; `person[q].team = t` (shown as "VON <club>").
   10. `market[slot].+0x28 = f_2277_25de(slot, 4)` (asking price with ±noise).
   11. `j++`.
   If the market is full, f_1ecd_37d8 shows "Schon 12 Mann auf dem Transfermarkt" and returns
   0x7f; the code still writes owner/team/price (out-of-range slot) — edge case, avoid.

### 4.3 Weekly market maintenance (f_0cb5_13bd, see §7) and season end (§9.4)

### 4.4 AI-club reshuffle `f_14a4_1798` (season end via f_0cb5_2756, and f_14a4_19fb)
1. For p = 1..150: `s1 += rand(1,7)-4`, `s2 += rand(1,7)-4`, each clamped [30,99] (f_14a4_174e,
   order: s1 then s2). Count `F` = persons with `64 <= team != 0xff`.
2. `n = rand(5,15)`; repeat n times: loop { `p = rand(1,150)`, `t0 = team(p)`;
   if `F > 4` and t0 foreign: `F--`, `team(p) = f_14a4_24bf(p, f_2277_25de(p,0)/10000, 0)`, mark done }
   until t0 not a human team and p not done and F <= 4 (listing 14a4:18ee-1912); then
   `team(p) = f_14a4_24bf(...)` likewise. So the first round moves every foreign person beyond 4
   to a German club, and each round moves one more non-human person.
   (f_2277_25de is called with the person index as slot index of the current manager – original bug;
   value is effectively garbage-ish.)
3. Writes a 3-int result `out[k] = cnt_k*100/T*G/100` to the caller's buffer (bytes): cnt_k = squad
   sizes of the humans in league k (counted before step 1), T = sum of cnt_k + #foreign persons after
   step 2 (min 1), G = 150-T.
4. f_0cb5_2756 (end of f_0cb5_0012, before the summer days) calls f_14a4_1798, then per range k of
   D:0x2272 ({0,17},{18,37},{38,57},{58,255}; out[3] = 0) counts the persons with a team in the range
   (each lowers out[k] while > 0) and lists those with owner > 3. For k = 0..3, while out[k] > 0 (at
   most 1000 rounds), it takes from every other range j, while j has listed persons left: a random
   still listed entry (`rand(0, count_j-1)`, at most 1001 tries), `team = f_0cb5_2987(k)`. Then it
   calls f_14a4_1798 again.
   f_0cb5_2987(k): k > 2 → `rand(D:0x2272[2k], 255)`; else loop { x = rand(0,3); if rand(0,1) x =
   rand(0,6); if rand(0,5)==0 x = rand(0,12); t = D:0x226e[k] + x } while t has a human manager.

---------------------------------------------------------------------------------------------
## 5. The market screen `f_2277_04a5`

Layout: left list = own squad (`f_1ecd_06af`, manager 0x304a); right list = market (manager 4),
"Kontostand: <balance>", buttons **LEIHEN** (loan) / **KAUFEN** (buy). Loan mode `L` starts 0 (KAUFEN
highlighted); clicking the button area (`mouse.x > 0x6d`, special screen ids 0x7ac/0x7c9 excluded)
toggles `L = 1 - L` and redraws the market list with display mode `L+1`.
Exit (code 0x66) → `f_0f9d_0002(manager,0)`.

Hover texts:
* market row: `"VON " + (owner==4 ? team name : manager name) + ", " + age + " J."`
* own-squad row (via f_1ecd_29c6 → f_1ecd_2897): `"STÄRKE:" (s10+s11+s12)/3 ", " age " JAHRE"`.

Price column of the market list (f_1ecd_06af column 17, 1ecd:0eeb):
`price = (owner==4) ? slot.+0x28 : f_2277_25de(slot,0) computed as manager 4`; in LEIHEN view
(`mode==2`) `price /= 3`. (`"Nicht mehr als N Spieler einsetzen"` in f_1ecd_1560 is a lineup rule,
not a market filter.)

Right-click (`F:0x2ea2 == 3`) on a player = info screen `f_14a4_0906` (market player only if it is
your own listed player).

### 5.1 Click on an own squad player (slot si)
1. If `si.+0x18 & 0x80`: "Dieser Spieler hört am Saisonende auf." → nothing.
2. **AI offer pending (`+9 & 0x40`)** → sale dialog:
   * `V = f_2277_25de(si, 0)`; `O = (rand(85,150) * V / 10000) * 100`.
   * `f_2277_1b5f(&O, years = si.+0x0b)`: shows `"Angebot: O DM."`, computes
     `Dz = (O/7) * years`, shows `"Ablöse: Dz DM."`, sets `O = O - Dz`; buttons **BEHALTEN**(1) /
     **VERKAUFEN**(2). (Semantics of the two lines UNCERTAIN; the credited amount is `O - (O/7)*years`.)
   * Always: clear `+9 & 0x40`, delete message `+0x30`.
   * VERKAUFEN: `balance += O`; remove slot (`f_1ecd_10ee(si,1,24)`);
     **then (quirk) read `+0x16` and skills from the slot now at index si (the next player!)**:
     `person.team = next.+0x16`, `next.+0x16 = 0`, `f_14a4_23da(that team, (s10+s11+s12)/3 of next)`;
     `person.owner = 5`. (Intended: buyer = `+0x16` of the sold player.)
3. Otherwise **list him on the market**:
   * count market slots whose person is owned by this manager; if `>= 3` →
     "Schon 3 Spieler auf dem Transfermarkt".
   * if `person.owner != manager` (loaned in) → "Dieser Spieler ist nur ausgeliehen".
   * as manager 4: `ms = f_1ecd_37d8(p, 1, 0, 1, 0, 12)`; 0x7f (market full) → message only.
   * `si.+0x0a = 0`; memcpy whole squad slot → market slot ms (so the market slot carries salary in
     +0x28, contract years etc.); remove from squad `f_1ecd_10ee(si,1,24)`. Owner stays the manager.

### 5.2 Click on a market slot i (person p, `ms = market[i]`)
Let `bid` = 0, `OWN = 0`, `H2H = 0`, `NOMARK = 0`.
1. If `ms.+3 & 0x80` and `ms.+3 & (1<<mgr)` → "Angebot wurde bereits abgelehnt" → stop.
2. If `owner(p)` is another human or (owner 4 and own bit not set): show **"Ihr Angebot:"** input
   (numeric) → `bid = input*[224e]/[2252]`.
3. Else if `owner(p) == mgr` (own listed player):
   * `ms.+9 & 0x80` (AI offer) → sale dialog like 5.1 but with `V = f_2277_25de(i,0)` as manager 4
     and `O = (rand(67,110)*V/10000)*100`, years = `ms.+0x0b`; clear 0x80 + message; VERKAUFEN:
     `person.team = ms.+0x16; ms.+0x16 = 0; f_14a4_23da(team, (s10+s11+s12)/3)`;
     `balance(mgr) += O-Dz`; remove market slot (`f_1ecd_10ee(i,1,12)` as mgr 4); `owner = 5`.
     Then stop (bid = 0).
   * no offer → `OWN = 1` (take the player back; no input).
4. Money check: if `bid != 0 and balance >= bid` or `OWN` → continue; else if `bid != 0`
   → "Sie haben nicht genug Geld" → stop; (bid 0 → silently stop).
5. **Seller is another human** (owner ∉ {4, mgr}, not OWN):
   * `V = f_2277_25de(i,0)` as manager 4; if `L`: `V /= 3`.
   * Allowed range: **`V*60/100 <= bid <= V*140/100`**; outside → dialog (addressed to the owning
     human) "DIESER BETRAG LIEGT AUSSERHALB DES ERLAUBTEN !" [NIE/NEIN], `NOMARK=1`, rejected.
   * inside → dialog to the owning human "NEHMEN SIE DAS ANGEBOT (<bid> DM) FÜR <name> AN ?"
     [NA GUT. / NEIN !]. Yes → `H2H = 1`; person skills 57f9..57fb := ms.+0x10..+0x12. No → rejected.
6. **Seller is the AI** (owner 4, not OWN): acceptance `f_2277_2171(i, bid, L)` (§6). 0 → rejected.
7. Rejected: "Ihr Angebot wurde abgelehnt"; if `!H2H && !NOMARK`: `ms.+3 |= (1<<mgr) | 0x80`.
8. Accepted (`H2H`, `OWN`, or AI accepted):
   1. `oldTeam = person.team`.
   2. `ns = f_1ecd_37d8(p, years=1, loan = L ? 0x63 : 0, 0, 0, limit = 24 + OWN)` into own squad.
      0x7f ("Schon 24 Mann im Team") → stop (no mark, no money).
   3. `savedSalary = ns.+0x28`.
   4. If `!OWN && !L`: **contract negotiation** `f_2277_2a8f(ns, 0, …)` (§8). Fail →
      remove ns (`f_1ecd_10ee(ns,1,24)`), `person.team = oldTeam`, "Der Transfer findet nicht statt",
      and mark rejection like step 7 (unless H2H).
   5. If `H2H`: copy ms fields +9, +0x17, +0x0d, +0, +1, +2 to ns.
   6. If `OWN || L`: memcpy ms → ns (52 bytes). If `L`: `ns.+0x0b = 1`, `ns.+0x0c = 0x80 + oldTeam`,
      `ns.+0x28 = savedSalary`, `ns.+3..+5 = 0`.
   7. Remove market slot i (`f_1ecd_10ee(i,1,12)` as manager 4).
   8. If `H2H`: `balance(seller) += bid`; `person.team = team(mgr)`.
   9. If `!L || OWN`: `person.owner = mgr`.
   10. If `H2H`: `balance(mgr) -= bid`.
   11. If `!OWN && !H2H`: `balance(mgr) -= bid`; `f_0000_272d(bid)` (sponsor subsidy, §5.4).
   Note: when a contract was negotiated (step 4) the new slot keeps the negotiated salary/years;
   for AI/H2H buys the slot data come from 37d8 (fresh counters), not from the market slot.

### 5.3 Squad-size and listing limits
* `f_1ecd_37d8(..., limit)` for a human: if the person's team != manager's team and
  `f_112a_00b4(m) > limit-1` → **"Schon 24 Mann im Team"** (0x7f).
  `f_112a_00b4(m)` = all slots of squad m + players with `owner == m` sitting in other squads
  (i.e. own players on the market or loaned out count too). Limit 24 (25 when taking back an own
  listed player).
* For the market (limit 12): if market slot 11 is occupied (`F:0x7995 + m*0x514`, m=4) →
  **"Schon 12 Mann auf dem Transfermarkt"**.
* Max **3** own players listed at once ("Schon 3 Spieler auf dem Transfermarkt", f_2277_04a5).

### 5.4 `f_0000_272d(fee)` sponsor subsidy after buying from the AI
With probability 1/7 (`rand(0,6)==0`): `Z = ((fee/100) * rand(20,65) / 10000) * 10000`;
dialog "NEHMEN SIE EINEN SPONSOR-ZUSCHUSS VON Z DM AN ?" [ABER IMMER! / LIEBER NICHT.]; yes →
`balance += Z`.

### 5.5 `f_1ecd_37d8(p, years, loan, -, -, limit)` — insert person into squad `F:0x304a`
(also used for the market). Limit checks §5.3. Position = first slot s (0..24) that is empty or
whose person index >= p; shift slots s..23 → s+1..24. New slot zeroed, then:
`+0x0f = p`, `+0x13 = rand(80,120)`, `+0x0e = 0`, `f_2277_000a` (`+0x0e = rand(35,65)`,
`+0x14 = rand(1,13) | rand(1,3)<<4 | rand(0,1)<<7`), `+0x0b = years`, `+0x0c = loan`,
`+0x10..+0x12 = person skills`, `+0x28 = f_2277_25de(s, loan==0x63 ? 3 : 1)` (salary).
If manager != 4 and `person.team != team(mgr)`: `person.57ff = person.5800 = 0`,
`person.team = team(mgr)`. Returns s.

---------------------------------------------------------------------------------------------
## 6. AI acceptance of a bid `f_2277_2171(i, bid, L)`
```
V = market[i].+0x28            ; stored asking price
if L: V = V/3
if rand(75,85)*V/100 > bid:      return REJECT
if rand(120,130)*V/100 < bid:    return ACCEPT
return (bid*100/V > rand(80,120)) ? ACCEPT : REJECT
```
(rand calls in this order; the third only if needed.)

---------------------------------------------------------------------------------------------
## 7. Weekly events relevant to transfers — `f_0cb5_13bd` (per human manager)
Called from the game loop for each human manager on days where `(day - D:224c) % 7 == 0` or
`(day-4) % 7 == 0` (1a58:3619..367e; UNCERTAIN wording, see loop spec). **The whole body only runs
while `day <= 321`.** Relevant parts, in execution order:

1. (Only when manager == 0) for each market slot: if `+0x13 > 56`: `+0x13 -= rand(3,9)`;
   with `rand(0,3)==0`: if `+9 > 0x1f` and message `+0x30` exists → `+9 &= 0x1f`, delete message
   (pending AI offer on a listed player expires).
2. For each market slot i:
   * if `(+3 & 0x7f) == (1<<mgr)` and `rand(0,2)==0`: `+3 ^= (1<<mgr)` (this manager may bid again;
     bit 0x80 stays; never cleared when several managers' bits are set).
   * if `owner(p) == mgr` (own listed player): `f_0cb5_2b88(slot)` (injury countdown), and if
     `+0x18 & 0x80 == 0`: as manager 4 `x = f_2277_25de(i,0)/10000`;
     **if `rand(0,180) < x` and `rand(0,2) != 0` and no message**: `+9 |= 0x80`,
     `+0x16 = f_14a4_24bf(i, x, 0)` (German buyer), message created ("<team> bietet für <name>").
3. (stadium events, unrelated)
4. For each own squad slot (day <= 321): injuries, retirement flag
   (`f_14a4_2588(slot) > rand(32,45)` and `+0x18 < 100` → `+0x18 |= 0x80`, message), and
   * with `rand(0,3)==0`: expire pending offer (`+9 > 0x1f` & message → `+9 &= 0x1f`, delete msg).
   * `+0x18`: if `>= 100` and not 0x80: with `rand(0,5)==0` and message present:
     `+0x18 = rand(9,17)`, delete message (extension request withdrawn); else if 1..99 (no 0x80):
     `+0x18--`.
   * **Contract-extension request** (only if `+0x0c == 0`):
     ```
     a  = min(|25 - age|, 8);         A = (-100*a)/8 + 100
     g  = min((w1c+w1e+w20)/36, 5)    (unsigned)
     P1 = ((s10+s11+s12)/3*60 + A*40) / 100
     P  = P1 + trunc(-(P1*g*10)/100)
     if rand(0,P)==0 and rand(0,3)==0 and no message and !(+0x18&0x80) and +0x0b == 1:
         r = rand(0,100); N = r>90 ? 4 : r>70 ? 3 : 2
         +0x18 = 100 + N ; message ("… möchte seinen Vertrag um N Jahre verlängern")
     ```
5. **AI offers for own squad players** (day <= 321), for each slot with person,
   `+0x0c == 0` (not loaned in), `+0x18 & 0x80 == 0`:
   `x = f_2277_25de(s,0)/10000`; **if `rand(0,200) < x` and `rand(0,450) == 0` and no message**:
   `+9 |= 0x40`; `foreign = (x > 90 && rand(0,12)==0)`; `+0x16 = f_14a4_24bf(s, x, foreign)`;
   message. (Resolved by the human in the market screen, §5.1.)

### 7.1 Buyer selection `f_14a4_24bf(slot, x, foreign)`
Range `[0,63]` (or `[64,199]` if foreign). Loop `tries = 0..`: `t = rand(lo,hi)`; skip human teams;
`Q = Σ_{k=0..2}(cur_k + base_k + 50) / 9` (team strengths `+0x18..+0x1d`);
accept if `tries > 500` or (`x < Q+20` and `x+2 > Q`).

### 7.2 Buyer strength update `f_14a4_23da(team, avg)`
`T = Σ_k (cur_k + base_k)` — **quirk: the original indexes with an uninitialised stack word, so
effectively `T = 3*(cur_u + base_u)` for an unknown u** (UNCERTAIN; use the intended sum);
`d = (avg - T/6 + 10)/3`; `k = rand(0,2)`; clamp d to [1,10];
`cur_k += rand(d-1,d+1)`; `base_k += rand(d-1,d+1)` (bytes, no clamp).

---------------------------------------------------------------------------------------------
## 8. Contract negotiation `f_2277_2a8f(slot, p2, …)` and acceptance `f_2277_2270`

Modes: `slot == -1` = interactive squad screen (pick players; from f_1ecd_1560);
`slot >= 0` = **single mode** (market purchase p2=0, season-end expiry p2=1). Returns 1 only in
single mode when the contract was accepted.

### 8.1 Pre-checks (interactive mode only, on picking a player)
* `+0x0c != 0` → "<name> ist leider von <team> nur ausgeliehen." (no talks)
* `+0x18 & 0x80` → "<name> wird sich mit Ablauf des Vertrages zur Ruhe setzen."
* `+0x18 >= 100` → `forcedYears = +0x18 - 100`
* `+0x18` in 1..99 → "<name> ist im Moment nicht verhandlungsbereit."
* else `forcedYears = 0`. (Single mode: forcedYears = 0, no checks.)

### 8.2 Screen
"<name>, <age> JAHRE ALT, <+6> LIGASPIELE"; fields "VERTRAGSDAUER (IN SAISONS)" (prefilled
current years) and "GEHALT PRO MONAT (IN DM)" (prefilled current salary); buttons
**NEUER VERTRAG** (=1) / **ABBRUCH** (=2). Years cannot be edited for loaned players.
**Salary hint** shown (and pre-set as the offer) after entering years Y:
```
S   = f_2277_25de(slot, 5)                       ; base demand
Yp  = (Y-1)*100 + (-100*md)/nmd + 100            ; md = D:225a[lg], nmd = D:2262[lg] (signed div)
k   = ((Yp/6) + 122) * 8 / 10
hint= S*(k+4)/100 ; if day > 321: hint = hint*(152-age)/100
hint= max(hint, current salary)
```
(The hint uses 122 while the real check uses 110 → the hint always exceeds the maximal demand.)

### 8.3 Evaluation after a button press (`res` = button)
* `Y > 4` → "Wer wird sich denn SO lange verpflichten ? (Mal abgesehen von Bundi's)" → res=2.
* If (interactive, or day > 321) and nothing usable entered (`Y==0 && salary==0`, or `Y==0 &&`
  current years 0) and forcedYears == 0 → `res = (day > 321) ? 2 : 3`.
* Defaults: `Y = forcedYears || current years`; `salary = salary || current salary`.
* `res==1 && Y > curYears && salary < curSalary` → "So dumm ist <name> leider nicht..." res=2.
* forcedYears != 0 and (`salary < curSalary` or `Y != forcedYears`) → res=2.
* res==1: if forcedYears == 0: `if !f_2277_2270(slot, Y, salary)` → "<name> ist nicht an Ihrem
  Angebot interessiert." res=2. (forcedYears != 0 → accepted without the check.)
  Accepted: `+0x0b = Y`, `+0x28 = salary`; interactive: "Ihr Angebot wurde angenommen !".
* For res 1 or 2 (not 3): delete message if `+0x18 > 99`; **`+0x18 = rand(10,18)`** (not
  negotiable for 10–18 weeks, even after ABBRUCH).

### 8.4 Player acceptance `f_2277_2270(slot, Y, sal)` (manager = F:0x304a)
```
cy = slot.+0x0b ; cs = slot.+0x28 ; age = person.age ; lg = league(mgr)
if Y == cy and sal > cs:            ACCEPT
if Y == cy and sal < cs:            REJECT
if cy > Y and cs*120/100 < sal:     ACCEPT     (unsigned)
if cy > Y and cs*105/100 >= sal:    REJECT
Yp = (Y-1)*100 + (-100*D:225a[lg])/D:2262[lg] + 100
S  = f_2277_25de(slot, 5)
k  = ((Yp/6) + 110) * 8 / 10                   (Yp/6 signed long div; *8 via shl 3)
S  = S * rand(k-6, k+4) / 100
if day > 321: S = S * rand(132-age, 152-age) / 100
ACCEPT iff S < 100000 and sal > S
```
So: longer contracts cost ~13 %/season more, demands fall during the season, are higher in the
season-end phase for young players; any player whose demand reaches 100 000 DM/month refuses.

### 8.5 Season end (`f_0cb5_0012`), contract parts
* Manager-0 pass, for every person 1..150 located in a squad (`f_3091_17b7`): `age++`; delete
  message, `+9 &= 0x3f`; `+0x0b--` (unless option `D:0x5d4`, then age := rand(18,25), +0x18 = 0).
  **Loan return**: if `person.owner != holder`: owner < 4 → `f_1ecd_37d8(p,1,0,0,0,25)` into the
  owner's squad + memcpy of the slot; owner >= 4 → `owner = 5`. Remove from holder with limit
  `(2 - (slotIndex==4))*12` (original compares the slot index, not the manager — quirk). Then clears
  `+0x18` and `+0x0c` of `SLOT(0, newIndex)` (only correct if the owner is manager 0 — quirk).
* Retirement (each manager pass): persons with contract 0 and `+0x18&0x80` are removed (owner 5);
  persons not in a squad with `rand(32,34) < age` are re-generated as young players.
* "Vertragsverhandlungen": for every human squad slot with `+0x0b == 0`:
  `f_2277_2a8f(s, 1, …)`. Failure → "<name> kehrt Ihrem Verein den Rücken...",
  **compensation `((f_2277_25de(s,0) >> 1) / 1000) * 1000` credited** ("Sie erhalten eine
  Ablösesumme von … DM"), slot removed (`f_1ecd_10ee(s,1,24)`; owner is not reset). Success →
  "… bleibt Ihnen auch die nächste Saison erhalten."
* Youth player (if team rating allows): salary `= f_2277_25de(s,1)*50/100`, contract `rand(2,3)`.
* Initial squads (new game, f_08bc_2184): contract 1 season, salary `f_2277_25de(s,1)`.

---------------------------------------------------------------------------------------------
## 9. Loans — summary
* Mode LEIHEN in the market screen; price shown and AI-acceptance base = asking price / 3;
  human seller range check uses current value / 3.
* Fee = the bid, paid immediately (to the AI: just deducted; to a human seller: transferred).
* Duration: `+0x0b = 1` → until the next season end, then the player returns (owner 4 → becomes
  free; owner human → back into that squad).
* Salary: `f_2277_25de(slot,3)` (= salary formula), paid by the loaning club.
* No contract negotiation, cannot be listed ("nur ausgeliehen"), cannot be renegotiated, gets no
  AI offers (`+0x0c != 0`).

## 10. Releasing players ("Trennungen")
No routine was found in these functions that dismisses a player for a severance payment.
"Trennungen" (D:0x2774) is only the header of the season-end screen listing departures
(retirements, returning loans, expired contracts). Ways to lose a player: sell to an AI offer,
list him on the market, contract expiry (compensation ½ value), retirement. UNCERTAIN whether a
dismissal exists elsewhere (e.g. squad screen f_1ecd_1560 jump tables were not decoded).

## 11. Account balance interactions (manager `+0x2432`, signed long)
| event | change |
|---|---|
| buy/loan from AI | `-bid` (+ optional sponsor subsidy Z) |
| buy/loan from human | buyer `-bid`, seller `+bid` |
| sell to AI (squad offer or listed offer) | `+(O - (O/7)*years)` |
| contract expiry without agreement | `+((value/2)/1000)*1000` |
| salaries | paid elsewhere (monthly finances; `+0x28` is the monthly salary) |
Purchases require `balance >= bid` (no credit check beyond that).

## 12. Other functions asked about
* `f_322b_2a2a(m, flag)` — **manager score/highscore entry**, not market: copies manager and team
  name to `F:0x4cfe/0x4d18`; `score = f_322b_2864(m) + (Σ squad values f_2277_25de(s,0)
  + f_14a4_2bba()*135000 + balance - f_112a_0f8d(0,99)) / 80000 + 40*[+0x227b] + 20*[+0x227c]
  - (500 - 60*([+0x227d]+[+0x227e]+[+0x227f])) + (flag==0 ? 300 : 0)`, min 1, stored in `F:0x4d32`.
  (The "dem Transfermarkt" annotation is a false string match on F:0x4d30.)
* `f_262f_2be8`, `f_262f_29a8`, `f_262f_2551`, `f_262f_281f` — **sponsorship / advertising
  contracts** (shirt + boards), not manager job offers: offers at `F:0x2e + ((m*2+cat)*10+k)*4`
  (long DM) with duration `F:0x2ec2+…` (months); "VERTRAG: N MONAT(E)", "KÜNDBAR", "KEIN
  INTERESSE..." for empty offers, "Sie stehen noch unter Vertrag" if a contract is running.
  Out of scope here.
* `f_2277_0081`, `f_2277_01a9`, `f_2277_0225` belong to screen `f_112a_274c` (sliders/points:
  `20 - Σ mgr[0x2383..0x2386]`, `10 - Σ mgr[0x2388..0x238b]`), not transfers.

## 13. Function list
| function | meaning |
|---|---|
| f_2277_000a | init slot +0x0e / +0x14 random traits |
| f_2277_04a5 | transfer market screen (buy, loan, list, take back, sell on offers) |
| f_2277_1b5f | offer dialog "Angebot/Ablöse" BEHALTEN/VERKAUFEN; returns 1 keep |
| f_2277_1e38 | market refresh (AI players replaced by free persons) |
| f_2277_2171 | AI accepts a bid? |
| f_2277_2270 | player accepts a contract offer? |
| f_2277_25de | player transfer value / salary |
| f_2277_2a8f | contract negotiation screen (interactive / single) |
| f_1ecd_37d8 | insert person into a squad / the market (limits 24 / 12) |
| f_1ecd_10ee | remove squad slot (compaction) |
| f_112a_00b4 | number of players owned by manager (incl. listed/loaned out) |
| f_3091_1109 | squad occupancy count |
| f_3091_17b7 | find the squad slot of a person |
| f_14a4_24bf | choose AI buyer club for a value |
| f_14a4_23da | raise buyer club strength after a sale |
| f_0cb5_13bd | weekly events: offers, extension requests, offer expiry, rejection reset |
| f_0cb5_0012 | season end: loan returns, contract expiry talks, compensation |
| f_14a4_1798 | season-end reshuffle of AI persons / skill drift |
| f_0000_272d | sponsor subsidy after a purchase |
| f_1ecd_2897 | tooltip "STÄRKE: n, a JAHRE" |
| f_1ecd_06af | squad/market list renderer (price column: §5) |

## 14. Open questions / UNCERTAIN
* Meaning of slot counters +0x00..+0x05 and words +0x1c/+0x1e/+0x20 (enter the value formula).
* f_2277_1b5f: what "Angebot" vs "Ablöse" mean to the player; code credits `O - (O/7)*years`.
* f_14a4_23da reads an uninitialised index (original bug); sale via squad-offer reads the *next*
  slot after removal (original bug). Decide whether to replicate.
* Exact weekly cadence of f_0cb5_13bd (days ≡ 0 or 4 mod 7?) belongs to the game-loop spec.
* `D:0x225a` assumed = current matchday of a league.
* Dismissal of players with severance not found.
