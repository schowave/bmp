#!/usr/bin/env python3
"""Holt Seiten von transfermarkt.de und liest sie aus: Abschlusstabellen,
Kader, Leistungsdaten, die Staffeln der Regionalliga und die Teilnehmer der
Europapokale. tm_json.py und tm_games.py rufen diese Funktionen auf.

Wettbewerbe (Transfermarkt-Kürzel, bitte im Browser prüfen):
    L1 = 1. Bundesliga, L2 = 2. Bundesliga, L3 = 3. Liga (ab 2008),
    Regionalliga 1994-2008 und Oberligen haben eigene Kürzel; die sieht
    man in der Adresse der Wettbewerbsseite (".../wettbewerb/KÜRZEL").

Jede Funktion wartet pause Sekunden vor der Seite und meldet HTTP 403/429
als Blocked. Die HTML-Struktur von Transfermarkt ändert sich; liest eine
Funktion nichts mehr, die Selektoren anpassen (Seite im Browser öffnen,
"Element untersuchen").
"""

import re
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


def regionalligas(session, year, pause):
    """Kuerzel der Regionalliga-Staffeln einer Saison, aus der Uebersicht der
    deutschen Wettbewerbe (Land 40): die Links .../regionalliga-.../wettbewerb/KUERZEL."""
    url = f"{BASE}/wettbewerbe/national/wettbewerbe/40/saison_id/{year}"
    soup = get(session, url, pause)
    codes = []
    for a in soup.select("a[href*='/wettbewerb/']"):
        m = re.search(r"/(regionalliga[a-z-]*)/startseite/wettbewerb/([A-Z0-9]+)", a.get("href", ""))
        if m and m.group(2) not in codes:
            codes.append(m.group(2))
    return codes


def participants(session, comp, year, pause):
    """Teilnehmer eines Europapokals (CL, EL, UCOL, CLQ, ELQ, ECLQ) einer Saison:
    Verein und Land, in der Reihenfolge der Seite."""
    url = f"{BASE}/x/teilnehmer/pokalwettbewerb/{comp}/saison_id/{year}"
    soup = get(session, url, pause)
    out, seen = [], set()
    for tr in soup.select("table.items tbody tr"):
        link = tr.select_one("td.hauptlink a[href*='/verein/']") or tr.select_one("a[href*='/verein/']")
        if not link:
            continue
        m = re.search(r"/verein/(\d+)", link["href"])
        if not m or m.group(1) in seen:
            continue
        seen.add(m.group(1))
        flag = tr.select_one("img.flaggenrahmen")
        out.append({"verein": link.get("title") or link.get_text(strip=True), "verein_id": m.group(1),
                    "land": flag.get("title", "") if flag else ""})
    return out


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


def games(session, club_id, comp, year, pause):
    """Einsaetze und Minuten der Spieler eines Vereins im Wettbewerb comp in der
    Saison year/year+1, aus seinen Leistungsdaten; auch fuer Saisons ohne Marktwert."""
    url = f"{BASE}/x/leistungsdaten/verein/{club_id}/reldata/{comp}%26{year}/plus/1"
    soup = get(session, url, pause)
    out = []
    for tr in soup.select("table.items > tbody > tr"):
        name = tr.select_one("td.hauptlink a")
        tds = [td.get_text(" ", strip=True) for td in tr.find_all("td", recursive=False)]
        if not name or len(tds) < 10:
            continue
        # Von hinten: Einsaetze, Tore, Vorlagen, Gelb, Gelb-Rot, Rot, Ein-, Auswechslungen, PPS, Minuten.
        # Ohne Einsatz steht statt der Zahl ein Text ("Wurde ... nicht eingesetzt").
        out.append({"name": name.get_text(strip=True), "einsaetze": tds[-10], "minuten": tds[-1]})
    return out

