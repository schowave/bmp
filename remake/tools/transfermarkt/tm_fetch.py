#!/usr/bin/env python3
"""Fetches pages of transfermarkt.de and reads them: final tables, squads,
performance data, the groups of the Regionalliga and the participants of the
European cups. tm_json.py and tm_games.py call these functions.

Competitions (codes of Transfermarkt, check them in the browser):
    L1 = 1. Bundesliga, L2 = 2. Bundesliga, L3 = 3. Liga (from 2008),
    the Regionalliga 1994-2008 and the Oberligen have codes of their own;
    they show in the address of the competition's page (".../wettbewerb/CODE").

Every function waits pause seconds before the page and reports HTTP 403/429
as Blocked. The HTML of Transfermarkt changes; when a function reads nothing
any more, adapt its selectors (open the page in the browser, "Inspect").
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
    """The final table of a competition in the season year/year+1."""
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
            # The last columns: games ... goals, difference, points.
            "spiele": cells[-7] if len(cells) >= 7 else "",
            "tore": cells[-3],
            "punkte": cells[-1],
        })
    return rows


def regionalligas(session, year, pause):
    """The codes of the Regionalliga groups of a season, from the overview of the
    German competitions (country 40): the links .../regionalliga-.../wettbewerb/CODE."""
    url = f"{BASE}/wettbewerbe/national/wettbewerbe/40/saison_id/{year}"
    soup = get(session, url, pause)
    codes = []
    for a in soup.select("a[href*='/wettbewerb/']"):
        code = regionalliga_code(a.get("href", ""))
        if code and code not in codes:
            codes.append(code)
    return codes


def regionalliga_code(href):
    """The code of a link to a Regionalliga group, else None; the names of
    former groups carry their last season ("regionalliga-nord-bis-11-12-")."""
    m = re.search(r"/regionalliga[a-z0-9-]*/startseite/wettbewerb/([A-Z0-9]+)", href)
    return m.group(1) if m else None


def participants(session, comp, year, pause):
    """The participants of a European cup (CL, EL, UCOL, CLQ, ELQ, ECLQ) in a
    season: club and country, in the order of the page."""
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
    """The squad of a club in the season year/year+1."""
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
    """The games and minutes of the players of a club in the competition comp in
    the season year/year+1, from its performance data; also for seasons without
    market values."""
    url = f"{BASE}/x/leistungsdaten/verein/{club_id}/reldata/{comp}%26{year}/plus/1"
    soup = get(session, url, pause)
    out = []
    for tr in soup.select("table.items > tbody > tr"):
        name = tr.select_one("td.hauptlink a")
        tds = [td.get_text(" ", strip=True) for td in tr.find_all("td", recursive=False)]
        if not name or len(tds) < 10:
            continue
        # From the end: games, goals, assists, yellow, yellow-red, red, substituted on, off, points per game, minutes.
        # A player without games has a text instead of the number ("Wurde ... nicht eingesetzt").
        out.append({"name": name.get_text(strip=True), "einsaetze": tds[-10], "minuten": tds[-1]})
    return out

