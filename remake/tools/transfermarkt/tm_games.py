#!/usr/bin/env python3
"""Orders the squads of the seasons without market values (to 2003/04) by games.

Transfermarkt orders a squad by market value only from 2004/05, before by
position. For every German club of a season of the game (internal/seasons/data)
the script finds the squad in OUT/.cache its players come from (the most names
in common), fetches the performance data in the squad's competition for these
clubs only and writes OUT/YYYY-YY.squads.json: per club (named as in the game)
the players by position, the most games first, on equal games the more
minutes, else as in the squad. bmpseasons squads puts them into the seasons
(mise run remake:seasons-games).

Resumable: the performance data lands in the cache as .games.json.

Usage: python3 tm_games.py --out DIR --seasons DIR [--from Y] [--to Y] [--offline]
Needs requests and beautifulsoup4 (the mise task provides them with uv)."""
import argparse, glob, json, os, re, sys, unicodedata
import requests
import tm_fetch as t
import tm_json as j

# Places and players per position in a season of the game (update.go).
NUM_GERMAN = 64
SIZES = {"goalkeepers": 2, "defenders": 6, "midfielders": 7, "forwards": 5}


def fold(name):
    """A name without accents in capitals, to compare it with the game."""
    name = name.split("/")[0].replace("ß", "SS").replace("ẞ", "SS")
    name = unicodedata.normalize("NFKD", name)
    return "".join(c for c in name if not unicodedata.combining(c)).upper().strip()[:25]


def picked(players):
    """The names the game takes from a squad (pickPlayers in update.go)."""
    return {fold(n) for g, size in SIZES.items() for n in players[g][:size]}


def minutes(text):
    """"3.060'" -> 3060."""
    digits = re.sub(r"\D", "", text)
    return int(digits) if digits else 0


def by_games(squad, games):
    """The squad by games and minutes; sort is stable, a tie stays as in the squad."""
    stats = {g["name"]: (j.number(g["einsaetze"]), minutes(g["minuten"])) for g in games}
    return sorted(squad, key=lambda p: tuple(-x for x in stats.get(p["name"], (0, 0))))


def season(session, year, seasons_dir):
    """The squads of a season per club of the game; (squads, failures)."""
    sid = f"{year}-{str(year + 1)[2:]}"
    with open(os.path.join(seasons_dir, sid + ".json"), encoding="utf-8") as f:
        clubs = json.load(f)[:NUM_GERMAN]
    cache = os.path.join(j.CACHE, str(year))
    squads = {}  # (comp, club id) -> (squad, the names the game takes)
    for path in glob.glob(os.path.join(cache, "*.squad.json")):
        comp, club_id = os.path.basename(path)[:-len(".squad.json")].rsplit("-", 1)
        with open(path, encoding="utf-8") as f:
            squad = json.load(f)
        squads[(comp, club_id)] = (squad, picked(j.convert(squad)))
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
        # A club the season has from the season before finds no squad.
        if not names or tie or score < 0.8 * len(names):
            unmatched.append(c["name"])
            continue
        comp, club_id = best
        games = j.cached(os.path.join(cache, f"{comp}-{club_id}.games.json"), t.games, session, club_id, comp, year)
        if games is None:
            failed.append(f"{comp}/{c['name']}")
            continue
        out[c["name"]] = j.convert(by_games(squads[best][0], games))
    print(f"{year}: {len(out)} clubs by games, without a squad: {unmatched}", flush=True)
    return out, failed


def main():
    ap = argparse.ArgumentParser(description="squads to 2003/04 by games from transfermarkt.de")
    ap.add_argument("--out", required=True, help="directory of the season files of tm_json.py (with .cache)")
    ap.add_argument("--seasons", required=True, help="the seasons of the game (internal/seasons/data)")
    ap.add_argument("--from", dest="first", type=int, default=1995)
    ap.add_argument("--to", dest="last", type=int, default=2003)
    ap.add_argument("--offline", action="store_true", help="read the cache only")
    a = ap.parse_args()
    out_dir = os.path.abspath(os.path.expanduser(a.out))
    j.CACHE, j.OFFLINE = os.path.join(out_dir, ".cache"), a.offline
    session = requests.Session()
    incomplete = 0
    for year in range(a.first, a.last + 1):
        squads, failed = season(session, year, a.seasons)
        if failed:
            incomplete += 1
            print(f"{year}: incomplete, not written ({len(failed)} failures: {failed[:5]})", flush=True)
            continue
        j.write_json(os.path.join(out_dir, f"{year}-{str(year + 1)[2:]}.squads.json"), squads)
    sys.exit(1 if incomplete else 0)


if __name__ == "__main__":
    main()
