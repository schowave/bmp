#!/usr/bin/env python3
"""Holt Abschlusstabellen und Kader von transfermarkt.de für den eigenen
Gebrauch (einmalig, langsam) und schreibt sie als CSV.

Aufruf:
    pip install requests beautifulsoup4
    python3 tm_fetch.py --out ~/tm-daten --from 1990 --to 2008 L1 L2 RLN RLS

Wettbewerbe (Transfermarkt-Kürzel, bitte im Browser prüfen):
    L1 = 1. Bundesliga, L2 = 2. Bundesliga, L3 = 3. Liga (ab 2008),
    Regionalliga 1994-2008 und Oberligen haben eigene Kürzel; die sieht
    man in der Adresse der Wettbewerbsseite (".../wettbewerb/KÜRZEL").

Ausgabe in --out:
    tables.csv  saison,wettbewerb,platz,verein,verein_id,spiele,tore,punkte
    squads.csv  saison,verein,verein_id,name,position,geburtsdatum,marktwert

Das Skript wartet --pause Sekunden zwischen zwei Seiten und bricht bei
HTTP 403/429 ab. Die HTML-Struktur von Transfermarkt ändert sich; die
Selektoren unten sind nicht getestet und müssen eventuell angepasst
werden (Seite im Browser öffnen, "Element untersuchen").
"""

import argparse
import csv
import os
import re
import sys
import time

import requests
from bs4 import BeautifulSoup

BASE = "https://www.transfermarkt.de"
HEADERS = {
    "User-Agent": "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 "
    "(KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36",
    "Accept-Language": "de-DE,de;q=0.9",
}


class Blocked(Exception):
    pass


def get(session, url, pause):
    time.sleep(pause)
    r = session.get(url, headers=HEADERS, timeout=60)
    if r.status_code in (403, 429):
        raise Blocked(f"{url}: HTTP {r.status_code}")
    r.raise_for_status()
    return BeautifulSoup(r.text, "html.parser")


def table(session, comp, year, pause):
    """Abschlusstabelle eines Wettbewerbs in der Saison year/year+1."""
    url = f"{BASE}/x/tabelle/wettbewerb/{comp}/saison_id/{year}"
    soup = get(session, url, pause)
    rows = []
    for tr in soup.select("div.responsive-table table.items tbody tr"):
        tds = tr.find_all("td")
        link = tr.select_one("td.hauptlink a[href*='/verein/']") or tr.select_one("a[href*='/verein/']")
        if not link or len(tds) < 6:
            continue
        m = re.search(r"/verein/(\d+)", link["href"])
        cells = [td.get_text(" ", strip=True) for td in tds]
        rows.append({
            "saison": year,
            "wettbewerb": comp,
            "platz": cells[0],
            "verein": link.get("title") or link.get_text(strip=True),
            "verein_id": m.group(1) if m else "",
            # Die letzten Spalten: Spiele ... Tore, Differenz, Punkte.
            "spiele": cells[-7] if len(cells) >= 7 else "",
            "tore": cells[-3],
            "punkte": cells[-1],
        })
    return rows


def squad(session, club_id, year, pause):
    """Kader eines Vereins in der Saison year/year+1."""
    url = f"{BASE}/x/kader/verein/{club_id}/saison_id/{year}/plus/1"
    soup = get(session, url, pause)
    out = []
    for tr in soup.select("table.items > tbody > tr"):
        name = tr.select_one("td.hauptlink a")
        if not name:
            continue
        inline = tr.select("table.inline-table tr")
        position = inline[-1].get_text(strip=True) if inline else ""
        tds = [td.get_text(" ", strip=True) for td in tr.find_all("td", recursive=False)]
        born = next((t for t in tds if re.match(r"\d{2}\.\d{2}\.\d{4}", t)), "")
        value = tds[-1] if tds and "€" in tds[-1] else ""
        out.append({"name": name.get_text(strip=True), "position": position, "geburtsdatum": born, "marktwert": value})
    return out


def main():
    ap = argparse.ArgumentParser()
    ap.add_argument("--out", required=True)
    ap.add_argument("--from", dest="first", type=int, default=1990)
    ap.add_argument("--to", dest="last", type=int, default=2008)
    ap.add_argument("--pause", type=float, default=4.0)
    ap.add_argument("comps", nargs="+")
    a = ap.parse_args()
    os.makedirs(a.out, exist_ok=True)
    s = requests.Session()
    tf = open(os.path.join(a.out, "tables.csv"), "w", newline="", encoding="utf-8")
    sf = open(os.path.join(a.out, "squads.csv"), "w", newline="", encoding="utf-8")
    tw = csv.DictWriter(tf, ["saison", "wettbewerb", "platz", "verein", "verein_id", "spiele", "tore", "punkte"])
    sw = csv.DictWriter(sf, ["saison", "verein", "verein_id", "name", "position", "geburtsdatum", "marktwert"])
    tw.writeheader()
    sw.writeheader()
    try:
        for year in range(a.first, a.last + 1):
            for comp in a.comps:
                rows = table(s, comp, year, a.pause)
                print(f"{year} {comp}: {len(rows)} Vereine", file=sys.stderr)
                tw.writerows(rows)
                tf.flush()
                for r in rows:
                    if not r["verein_id"]:
                        continue
                    for p in squad(s, r["verein_id"], year, a.pause):
                        sw.writerow({"saison": year, "verein": r["verein"], "verein_id": r["verein_id"], **p})
                    sf.flush()
    except Blocked as e:
        print(f"Abbruch, die Seite sperrt: {e}", file=sys.stderr)
    finally:
        tf.close()
        sf.close()


if __name__ == "__main__":
    main()
