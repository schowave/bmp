#!/usr/bin/env python3
"""Holt die Tabellen und Kader der 1. bis 3. Liga einer oder mehrerer Saisons
von transfermarkt.de und schreibt pro Saison OUT/YYYY-YY.json: je Verein Liga,
Stufe, Tabellenplatz, Spiele, Tore, Punkte und die Spieler nach Position, die
wertvollsten zuerst; ab 2008 auch die Regionalliga (Stufe 4). Dazu
OUT/YYYY-YY.europe.json mit den auslaendischen Vereinen der Europapokale. remake/cmd/bmpseasons macht daraus die Saisons des Spiels
(mise run remake:seasons-update).

Wiederaufnehmbar: Tabellen und Kader landen in OUT/.cache, vorhandene Saisons
werden uebersprungen; --refresh holt die gewaehlten Saisons neu, --offline
liest nur den Cache.

Aufruf: python3 tm_json.py --out DIR [--from Y] [--to Y] [--refresh] [--offline]
Braucht requests und beautifulsoup4 (der mise-Task stellt sie mit uv bereit)."""
import argparse, datetime, json, os, re, shutil, sys, time
import random
import requests
import tm_fetch as t

PARTICLES = {"von", "van", "de", "der", "den", "da", "di", "dos", "do", "du", "zu", "ter", "ten", "le", "la", "el", "al", "af", "dal", "del", "della", "vom", "zum", "zur", "mac", "mc"}
GROUP = {"Torwart": "goalkeepers", "Abwehr": "defenders", "Mittelfeld": "midfielders", "Sturm": "forwards"}


def group(position):
    """Transfermarkt nennt die genaue Position (Innenverteidiger, Linksaussen, ...)."""
    position = position.lower()
    if "torwart" in position:
        return "goalkeepers"
    if "verteidiger" in position or "libero" in position or position == "abwehr":
        return "defenders"
    if "mittelfeld" in position:
        return "midfielders"
    if "stürmer" in position or "spitze" in position or "außen" in position or position == "sturm":
        return "forwards"
    print("unbekannte Position:", position, flush=True)
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
    """'7,00 Mio. €' -> 7000000; -1, wenn Transfermarkt keinen Wert kennt."""
    m = re.match(r"([\d.,]+)\s*(Mrd|Mio|Tsd)?", text)
    if not m:
        return -1
    n = float(m.group(1).replace(".", "").replace(",", "."))
    return n * {"Mrd": 1e9, "Mio": 1e6, "Tsd": 1e3, None: 1}[m.group(2)]


def convert(players):
    """Nachnamen in Grossbuchstaben; bei gleichem Nachnamen im Verein Initial davor."""
    # Teuerste zuerst; ohne Marktwert (vor ca. 2004) bleibt es bei der Reihenfolge von Transfermarkt.
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
    """'14' -> 14; 0, wenn es keine Zahl ist."""
    m = re.match(r"-?\d+", str(text).strip())
    return int(m.group(0)) if m else 0


def running_season():
    """Erstes Jahr der laufenden Saison; eine Saison beginnt im Juli."""
    today = datetime.date.today()
    return today.year if today.month >= 7 else today.year - 1


def third_tier(year):
    """Dritte Ebene: Oberligen (bis 1993), Regionalligen (1994-2007), 3. Liga (ab 2008).
    Codes von Transfermarkt; die Kuerzel gelten nur fuer die genannten Saisons."""
    if year == 1990:
        return ["NO93", "OLNA", "OL6B", "SW93", "OL7", "OBW3", "BA93", "BO91", "OLNS"]
    if year <= 1993:
        return ["NO93", "OLNA", "OL6B", "SW93", "OL7", "OBW3", "BA93", "OBSM", "OBSN", "OBSS"]
    if year <= 1999:
        return ["RN", "RNO", "RWS", "RS"]
    if year <= 2007:
        return ["RN", "RS"]
    return ["L3"]


OUT = CACHE = None  # main setzt sie aus --out
OFFLINE = False
PAUSE = 2.0
MAX_PASSES = 5


def write_json(path, data):
    """Atomar schreiben, damit ein Abbruch keine halbe Datei hinterlaesst."""
    os.makedirs(os.path.dirname(path), exist_ok=True)
    tmp = path + ".tmp"
    with open(tmp, "w", encoding="utf-8") as f:
        json.dump(data, f, ensure_ascii=False, indent="\t")
    os.replace(tmp, path)


def cached(path, fn, *args):
    """Liefert den Cache-Eintrag oder holt ihn; bei Fehler None (wird nicht gecacht)."""
    if os.path.exists(path):
        with open(path, encoding="utf-8") as f:
            return json.load(f)
    data = fetch(fn, *args)
    if data is not None:
        write_json(path, data)
    return data


def fetch(fn, session, *args):
    """Mehrere Versuche mit wachsender Wartezeit. Bei Sperre (403/429) lange warten,
    bei anderen Fehlern kurz. None, wenn alle Versuche scheitern oder --offline gilt."""
    if OFFLINE:
        print(f"nicht im Cache: {fn.__name__} {args}", flush=True)
        return None
    for attempt in range(6):
        try:
            return fn(session, *args, PAUSE)
        except t.Blocked as e:
            wait = min(60 * 2 ** attempt, 1800)
            print(f"gesperrt ({e}), warte {wait} s", flush=True)
        except (requests.RequestException, ValueError, AttributeError, TypeError) as e:
            wait = min(10 * 2 ** attempt, 300)
            print(f"Fehler {args[:2]} (Versuch {attempt + 1}): {e!r}, warte {wait} s", flush=True)
        time.sleep(wait + random.uniform(0, 5))
    return None


def season(session, year):
    """Eine Saison; (clubs, Fehlerliste). Cache pro Tabelle und Kader. Die
    Regionalliga ab 2008 (Stufe 4) ist optional: sie fuellt die Plaetze ausserhalb
    der Ligen, ohne sie nimmt das Spiel die Vereine der Vorsaison."""
    clubs, failed = [], []
    comps = [("L1", 1), ("L2", 2)] + [(c, 3) for c in third_tier(year)]
    if year >= 2008:
        codes = cached(os.path.join(CACHE, str(year), "regionalligas.json"), t.regionalligas, session, year)
        if codes:
            comps += [(c, 4) for c in codes]
        else:
            print(f"{year}: Regionalliga nicht gefunden, weiter ohne", flush=True)
    for comp, tier in comps:
        rows = cached(os.path.join(CACHE, str(year), f"{comp}.table.json"), t.table, session, comp, year)
        if not rows:  # None (Fehler) oder leer: nicht cachen, spaeter nochmal
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
            # Aus der Regionalliga kommen nur die Ersten ins Spiel: Kader nur fuer sie.
            if tier < 4 or entry["rank"] <= 3:
                kader = cached(os.path.join(CACHE, str(year), f"{comp}-{r['verein_id']}.squad.json"), t.squad, session, r["verein_id"], year)
                if kader is None:
                    if tier < 4:
                        failed.append(f"{comp}/{r['verein']}")
                    continue
                entry["players"] = convert(kader)
            ok += 1
            clubs.append(entry)
        print(f"{year} {comp}: {ok}/{len(rows)} Vereine", flush=True)
    return clubs, failed


EUROPE = ["CL", "EL", "UCOL", "CLQ", "ELQ", "ECLQ"]


def europe(session, year):
    """Die auslaendischen Vereine der Europapokale einer Saison, die Hauptrunden
    vor den Qualifikationen; [] wenn nichts zu holen ist (dann nimmt das Spiel die
    der Vorsaison)."""
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
    ap = argparse.ArgumentParser(description="Tabellen und Kader von transfermarkt.de je Saison")
    ap.add_argument("--out", required=True, help="Verzeichnis der Saisondateien (mit .cache)")
    ap.add_argument("--from", dest="first", type=int, default=running_season())
    ap.add_argument("--to", dest="last", type=int)
    ap.add_argument("--refresh", action="store_true", help="die gewaehlten Saisons neu holen")
    ap.add_argument("--offline", action="store_true", help="nur den Cache lesen")
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
                print(f"{year}: unvollstaendig, nicht geschrieben ({len(failed)} Fehler: {failed[:5]})", flush=True)
                continue
            write_json(path, clubs)
            foreign = europe(session, year)
            if foreign:
                write_json(os.path.join(OUT, f"{year}-{str(year + 1)[2:]}.europe.json"), foreign)
            print(f"{year}: geschrieben ({len(clubs)} Vereine, {len(foreign)} auslaendische)", flush=True)
        if not open_seasons:
            print("fertig", flush=True)
            return
        if a.offline:
            sys.exit(1)
        print(f"Durchgang {n}: {open_seasons} Saisons offen, neuer Versuch", flush=True)
        time.sleep(300)
    sys.exit(1)


if __name__ == "__main__":
    main()
