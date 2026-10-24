#!/usr/bin/env python3
"""Fetches the tables and squads of the 1st to 3rd league of one or more
seasons from transfermarkt.de and writes per season OUT/YYYY-YY.json: per club
league, tier, table place, games, goals, points and the players by position,
the most valuable first; from 2008 also the Regionalliga (tier 4). Besides
OUT/YYYY-YY.europe.json with the foreign clubs of the European cups.
remake/cmd/bmpseasons makes the seasons of the game from them
(mise run remake:seasons-update).

Resumable: tables and squads land in OUT/.cache, seasons already written are
skipped; --refresh fetches the chosen seasons anew, --offline reads the cache
only.

Usage: python3 tm_json.py --out DIR [--from Y] [--to Y] [--refresh] [--offline]
Needs requests and beautifulsoup4 (the mise task provides them with uv)."""
import argparse, datetime, json, os, re, shutil, sys, time
import random
import requests
import tm_fetch as t

PARTICLES = {"von", "van", "de", "der", "den", "da", "di", "dos", "do", "du", "zu", "ter", "ten", "le", "la", "el", "al", "af", "dal", "del", "della", "vom", "zum", "zur", "mac", "mc"}
GROUP = {"Torwart": "goalkeepers", "Abwehr": "defenders", "Mittelfeld": "midfielders", "Sturm": "forwards"}


def group(position):
    """The group of a position; Transfermarkt names the exact one (Innenverteidiger, Linksaußen, ...)."""
    position = position.lower()
    if "torwart" in position:
        return "goalkeepers"
    if "verteidiger" in position or "libero" in position or position == "abwehr":
        return "defenders"
    if "mittelfeld" in position:
        return "midfielders"
    if "stürmer" in position or "spitze" in position or "außen" in position or position == "sturm":
        return "forwards"
    print("unknown position:", position, flush=True)
    return None


def split(name):
    parts = name.replace("\xa0", " ").split()
    if len(parts) == 1:
        return "", parts[0]
    i = len(parts) - 1
    while i > 1 and parts[i - 1].lower() in PARTICLES:
        i -= 1
    return " ".join(parts[:i]), " ".join(parts[i:])


def value(text):
    """'7,00 Mio. €' -> 7000000; -1 when Transfermarkt knows no value."""
    m = re.match(r"([\d.,]+)\s*(Mrd|Mio|Tsd)?", text)
    if not m:
        return -1
    n = float(m.group(1).replace(".", "").replace(",", "."))
    return n * {"Mrd": 1e9, "Mio": 1e6, "Tsd": 1e3, None: 1}[m.group(2)]


def convert(players):
    """Surnames in capitals; the initial before a surname two players of the club share."""
    # The most valuable first; without market values (before 2004) the order of Transfermarkt stays,
    # tm_games.py orders these seasons by games.
    players = sorted(players, key=lambda p: -value(p["marktwert"]))
    pairs = [(split(p["name"]), group(p["position"])) for p in players]
    counts = {}
    for (_, last), _g in pairs:
        counts[last.upper()] = counts.get(last.upper(), 0) + 1
    out = {g: [] for g in GROUP.values()}
    for (first, last), g in pairs:
        if g is None:
            continue
        n = last.upper()
        if first and counts[n] > 1:
            n = first[0].upper() + "." + n
        out[g].append(n)
    return out


def number(text):
    """'14' -> 14; 0 when it is no number."""
    m = re.match(r"-?\d+", str(text).strip())
    return int(m.group(0)) if m else 0


def running_season():
    """The first year of the running season; a season starts in July."""
    today = datetime.date.today()
    return today.year if today.month >= 7 else today.year - 1


def third_tier(year):
    """The third tier: Oberligen (to 1993), Regionalligen (1994-2007), 3. Liga (from 2008).
    Codes of Transfermarkt; they hold only for the seasons named."""
    if year == 1990:
        return ["NO93", "OLNA", "OL6B", "SW93", "OL7", "OBW3", "BA93", "BO91", "OLNS"]
    if year <= 1993:
        return ["NO93", "OLNA", "OL6B", "SW93", "OL7", "OBW3", "BA93", "OBSM", "OBSN", "OBSS"]
    if year <= 1999:
        return ["RN", "RNO", "RWS", "RS"]
    if year <= 2007:
        return ["RN", "RS"]
    return ["L3"]


OUT = CACHE = None  # main sets them from --out
OFFLINE = False
PAUSE = 2.0
MAX_PASSES = 5


def write_json(path, data):
    """Writes atomically, so that an abort leaves no half file."""
    os.makedirs(os.path.dirname(path), exist_ok=True)
    tmp = path + ".tmp"
    with open(tmp, "w", encoding="utf-8") as f:
        json.dump(data, f, ensure_ascii=False, indent="\t")
    os.replace(tmp, path)


def cached(path, fn, *args):
    """Returns the cache entry or fetches it; None on a failure (not cached)."""
    if os.path.exists(path):
        with open(path, encoding="utf-8") as f:
            return json.load(f)
    data = fetch(fn, *args)
    if data is not None:
        write_json(path, data)
    return data


def fetch(fn, session, *args):
    """Several attempts with growing waits: long when blocked (403/429), short on
    other failures. None when all attempts fail or with --offline."""
    if OFFLINE:
        print(f"not in the cache: {fn.__name__} {args}", flush=True)
        return None
    for attempt in range(6):
        try:
            return fn(session, *args, PAUSE)
        except t.Blocked as e:
            wait = min(60 * 2 ** attempt, 1800)
            print(f"blocked ({e}), waiting {wait} s", flush=True)
        except (requests.RequestException, ValueError, AttributeError, TypeError) as e:
            wait = min(10 * 2 ** attempt, 300)
            print(f"failure {args[:2]} (attempt {attempt + 1}): {e!r}, waiting {wait} s", flush=True)
        time.sleep(wait + random.uniform(0, 5))
    return None


def season(session, year):
    """A season; (clubs, failures). One cache file per table and squad. The
    Regionalliga from 2008 (tier 4) is optional: it fills the places outside
    the leagues; without it the game takes the clubs of the season before."""
    clubs, failed = [], []
    comps = [("L1", 1), ("L2", 2)] + [(c, 3) for c in third_tier(year)]
    if year >= 2008:
        codes = cached(os.path.join(CACHE, str(year), "regionalligas.json"), t.regionalligas, session, year)
        if codes:
            comps += [(c, 4) for c in codes]
        else:
            print(f"{year}: Regionalliga not found, going on without", flush=True)
    for comp, tier in comps:
        rows = cached(os.path.join(CACHE, str(year), f"{comp}.table.json"), t.table, session, comp, year)
        if not rows:  # None (failure) or empty: do not cache, try again later
            if tier < 4:
                failed.append(comp)
            if rows is not None:
                os.remove(os.path.join(CACHE, str(year), f"{comp}.table.json"))
            continue
        ok = 0
        for r in rows:
            if not r["verein_id"]:
                continue
            entry = {"name": r["verein"].upper(), "league": comp, "tier": tier, "rank": number(r["platz"]),
                     "games": number(r["spiele"]), "goals": r["tore"], "points": number(r["punkte"]),
                     "players": {g: [] for g in GROUP.values()}}
            # Only the first of the Regionalliga enter the game: squads only for them.
            if tier < 4 or entry["rank"] <= 3:
                squad = cached(os.path.join(CACHE, str(year), f"{comp}-{r['verein_id']}.squad.json"), t.squad, session, r["verein_id"], year)
                if squad is None:
                    if tier < 4:
                        failed.append(f"{comp}/{r['verein']}")
                    continue
                entry["players"] = convert(squad)
            ok += 1
            clubs.append(entry)
        print(f"{year} {comp}: {ok}/{len(rows)} clubs", flush=True)
    return clubs, failed


EUROPE = ["CL", "EL", "UCOL", "CLQ", "ELQ", "ECLQ"]


def europe(session, year):
    """The foreign clubs of the European cups of a season, the main rounds
    before the qualifications; [] when there is nothing to fetch (then the game
    takes those of the season before)."""
    out, seen = [], set()
    for comp in EUROPE:
        rows = cached(os.path.join(CACHE, str(year), f"{comp}.teilnehmer.json"), t.participants, session, comp, year)
        for r in rows or []:
            if r["verein_id"] in seen or r["land"] in ("Deutschland", "Germany"):
                continue
            seen.add(r["verein_id"])
            out.append({"name": r["verein"].upper(), "country": r["land"], "cup": comp})
    return out


def main():
    global OUT, CACHE, OFFLINE
    ap = argparse.ArgumentParser(description="tables and squads per season from transfermarkt.de")
    ap.add_argument("--out", required=True, help="directory of the season files (with .cache)")
    ap.add_argument("--from", dest="first", type=int, default=running_season())
    ap.add_argument("--to", dest="last", type=int)
    ap.add_argument("--refresh", action="store_true", help="fetch the chosen seasons anew")
    ap.add_argument("--offline", action="store_true", help="read the cache only")
    a = ap.parse_args()
    last = a.first if a.last is None else a.last
    OUT, OFFLINE = os.path.abspath(os.path.expanduser(a.out)), a.offline
    CACHE = os.path.join(OUT, ".cache")
    os.makedirs(OUT, exist_ok=True)
    years = range(a.first, last + 1)
    if a.refresh and not a.offline:
        for year in years:
            shutil.rmtree(os.path.join(CACHE, str(year)), ignore_errors=True)
            for suffix in (".json", ".europe.json"):
                path = os.path.join(OUT, f"{year}-{str(year + 1)[2:]}{suffix}")
                if os.path.exists(path):
                    os.remove(path)
    session = requests.Session()
    for n in range(1, MAX_PASSES + 1):
        open_seasons = 0
        for year in years:
            path = os.path.join(OUT, f"{year}-{str(year + 1)[2:]}.json")
            if os.path.exists(path) and not a.offline:
                continue
            clubs, failed = season(session, year)
            if failed:
                open_seasons += 1
                print(f"{year}: incomplete, not written ({len(failed)} failures: {failed[:5]})", flush=True)
                continue
            write_json(path, clubs)
            foreign = europe(session, year)
            if foreign:
                write_json(os.path.join(OUT, f"{year}-{str(year + 1)[2:]}.europe.json"), foreign)
            print(f"{year}: written ({len(clubs)} clubs, {len(foreign)} foreign)", flush=True)
        if not open_seasons:
            print("done", flush=True)
            return
        if a.offline:
            sys.exit(1)
        print(f"pass {n}: {open_seasons} seasons open, trying again", flush=True)
        time.sleep(300)
    sys.exit(1)


if __name__ == "__main__":
    main()
