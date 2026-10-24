#!/usr/bin/env python3
"""Ordnet die Kader der Saisons ohne Marktwert (bis 2003/04) nach Einsaetzen.

Transfermarkt ordnet einen Kader erst ab 2004/05 nach Marktwert, davor nach der
Position. Fuer jeden deutschen Verein einer Saison des Spiels
(internal/seasons/data) sucht das Skript den Kader in OUT/.cache, aus dem seine
Spieler stammen (die meisten gleichen Namen), holt nur fuer diese Vereine die
Leistungsdaten im Wettbewerb des Kaders und schreibt OUT/YYYY-YY.squads.json:
je Verein (Name wie im Spiel) die Spieler nach Position, die mit den meisten
Einsaetzen zuerst, bei gleich vielen die mit mehr Minuten, sonst wie im Kader.
bmpseasons squads setzt sie in die Saisons (mise run remake:seasons-games).

Wiederaufnehmbar: die Leistungsdaten landen als .games.json im Cache.

Aufruf: python3 tm_games.py --out DIR --seasons DIR [--from Y] [--to Y] [--offline]
Braucht requests und beautifulsoup4 (der mise-Task stellt sie mit uv bereit)."""
import argparse, glob, json, os, re, sys, unicodedata
import requests
import tm_fetch as t
import tm_json as j

# Plaetze und Spieler je Position in einer Saison des Spiels (update.go).
NUM_GERMAN = 64
SIZES = {"goalkeepers": 2, "defenders": 6, "midfielders": 7, "forwards": 5}


def fold(name):
    """Name ohne Akzente in Grossbuchstaben, zum Vergleich mit dem Spiel."""
    name = name.split("/")[0].replace("ß", "SS").replace("ẞ", "SS")
    name = unicodedata.normalize("NFKD", name)
    return "".join(c for c in name if not unicodedata.combining(c)).upper().strip()[:25]


def picked(players):
    """Die Namen, die das Spiel von einem Kader nimmt (pickPlayers in update.go)."""
    return {fold(n) for g, size in SIZES.items() for n in players[g][:size]}


def minutes(text):
    """"3.060'" -> 3060."""
    digits = re.sub(r"\D", "", text)
    return int(digits) if digits else 0


def by_games(kader, games):
    """Der Kader nach Einsaetzen und Minuten; sort ist stabil, Gleichstand bleibt wie im Kader."""
    stats = {g["name"]: (j.number(g["einsaetze"]), minutes(g["minuten"])) for g in games}
    return sorted(kader, key=lambda p: tuple(-x for x in stats.get(p["name"], (0, 0))))


def season(session, year, seasons_dir):
    """Die Kader einer Saison je Verein des Spiels; (squads, Fehlerliste)."""
    sid = f"{year}-{str(year + 1)[2:]}"
    with open(os.path.join(seasons_dir, sid + ".json"), encoding="utf-8") as f:
        clubs = json.load(f)[:NUM_GERMAN]
    cache = os.path.join(j.CACHE, str(year))
    squads = {}  # (comp, verein_id) -> (kader, Namen, die das Spiel nimmt)
    for path in glob.glob(os.path.join(cache, "*.squad.json")):
        comp, club_id = os.path.basename(path)[:-len(".squad.json")].rsplit("-", 1)
        with open(path, encoding="utf-8") as f:
            kader = json.load(f)
        squads[(comp, club_id)] = (kader, picked(j.convert(kader)))
    out, failed, unmatched = {}, [], []
    for c in clubs:
        names = {fold(n) for g in SIZES for n in c["players"][g] if n}
        best, score, tie = None, 0, False
        for key, (_, have) in squads.items():
            s = len(names & have)
            if s > score:
                best, score, tie = key, s, False
            elif s == score and s:
                tie = True
        # Ein Verein, den die Saison aus der Vorsaison hat, findet keinen Kader.
        if not names or tie or score < 0.8 * len(names):
            unmatched.append(c["name"])
            continue
        comp, club_id = best
        games = j.cached(os.path.join(cache, f"{comp}-{club_id}.games.json"), t.games, session, club_id, comp, year)
        if games is None:
            failed.append(f"{comp}/{c['name']}")
            continue
        out[c["name"]] = j.convert(by_games(squads[best][0], games))
    print(f"{year}: {len(out)} Vereine nach Einsaetzen, ohne Kader: {unmatched}", flush=True)
    return out, failed


def main():
    ap = argparse.ArgumentParser(description="Kader bis 2003/04 nach Einsaetzen von transfermarkt.de")
    ap.add_argument("--out", required=True, help="Verzeichnis der Saisondateien von tm_json.py (mit .cache)")
    ap.add_argument("--seasons", required=True, help="die Saisons des Spiels (internal/seasons/data)")
    ap.add_argument("--from", dest="first", type=int, default=1995)
    ap.add_argument("--to", dest="last", type=int, default=2003)
    ap.add_argument("--offline", action="store_true", help="nur den Cache lesen")
    a = ap.parse_args()
    out_dir = os.path.abspath(os.path.expanduser(a.out))
    j.CACHE, j.OFFLINE = os.path.join(out_dir, ".cache"), a.offline
    session = requests.Session()
    incomplete = 0
    for year in range(a.first, a.last + 1):
        squads, failed = season(session, year, a.seasons)
        if failed:
            incomplete += 1
            print(f"{year}: unvollstaendig, nicht geschrieben ({len(failed)} Fehler: {failed[:5]})", flush=True)
            continue
        j.write_json(os.path.join(out_dir, f"{year}-{str(year + 1)[2:]}.squads.json"), squads)
    sys.exit(1 if incomplete else 0)


if __name__ == "__main__":
    main()
