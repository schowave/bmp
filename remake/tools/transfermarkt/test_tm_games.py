"""Tests of tm_games without the network: the conversions and a season from a
small cache. They run with the standard library of python3
(mise run remake:test-tm); requests and beautifulsoup4 are stubbed when
missing."""
import contextlib, io, json, os, sys, tempfile, types, unittest

sys.path.insert(0, os.path.dirname(os.path.abspath(__file__)))
for name, attrs in {"requests": {"Session": object, "RequestException": Exception},
                    "bs4": {"BeautifulSoup": None}}.items():
    try:
        __import__(name)
    except ImportError:
        sys.modules[name] = types.SimpleNamespace(**attrs)

import tm_games as g  # noqa: E402
import tm_json as j  # noqa: E402


def squad(*players):
    return [{"name": n, "position": p, "geburtsdatum": "", "marktwert": ""} for n, p in players]


def games(*rows):
    return [{"name": n, "einsaetze": e, "minuten": m} for n, e, m in rows]


def players(**groups):
    out = {k: [] for k in g.SIZES}
    out.update(groups)
    return out


def write(path, data):
    os.makedirs(os.path.dirname(path), exist_ok=True)
    with open(path, "w", encoding="utf-8") as f:
        json.dump(data, f, ensure_ascii=False)


class Conversions(unittest.TestCase):
    def test_fold(self):
        self.assertEqual(g.fold("Élber"), "ELBER")
        self.assertEqual(g.fold("MÜLLER"), "MULLER")
        self.assertEqual(g.fold("Groß"), "GROSS")
        self.assertEqual(g.fold("ANDONOW/ANDONOV"), "ANDONOW")

    def test_minutes(self):
        self.assertEqual(g.minutes("3.060'"), 3060)
        self.assertEqual(g.minutes("-"), 0)

    def test_by_games(self):
        listed = squad(("Sven Scheuer", "Torwart"), ("Oliver Kahn", "Torwart"), ("Bernd Dreher", "Torwart"),
                       ("Markus Münch", "Linker Verteidiger"), ("Ali Daei", "Mittelstürmer"))
        got = g.by_games(listed, games(("Oliver Kahn", "34", "3.060'"),
                                       ("Sven Scheuer", "Wurde in dieser Spielzeit nicht eingesetzt", "-"),
                                       ("Markus Münch", "Stand in dieser Spielzeit nicht im Kader", "-"),
                                       ("Ali Daei", "34", "2.500'")))
        # More minutes first; without games and without performance data as in the squad.
        self.assertEqual([p["name"] for p in got],
                         ["Oliver Kahn", "Ali Daei", "Sven Scheuer", "Bernd Dreher", "Markus Münch"])


class Offline(unittest.TestCase):
    def test_season_from_cache(self):
        """A club of the game gets its squad by games; the squad of a club the
        game lacks and a club without a squad are left out."""
        with tempfile.TemporaryDirectory() as out:
            cache = os.path.join(out, ".cache", "1997")
            write(os.path.join(cache, "L1-27.squad.json"),
                  squad(("Oliver Kahn", "Torwart"), ("Sven Scheuer", "Torwart"), ("Bernd Dreher", "Torwart"),
                        ("Christian Saba", "Innenverteidiger"), ("Michael Tarnat", "Linker Verteidiger"),
                        ("Giovane Élber", "Mittelstürmer")))
            write(os.path.join(cache, "L1-27.games.json"),
                  games(("Oliver Kahn", "34", "3.060'"), ("Bernd Dreher", "1", "90'"),
                        ("Michael Tarnat", "33", "2.579'"), ("Giovane Élber", "28", "2.095'")))
            write(os.path.join(cache, "RS-28.squad.json"),
                  squad(("Uwe Gospodarek", "Torwart"), ("Paulo Sergio", "Rechtsaußen")))
            seasons = os.path.join(out, "seasons")
            write(os.path.join(seasons, "1997-98.json"), [
                {"name": "FC BAYERN MÜNCHEN", "players": players(goalkeepers=["KAHN", "SCHEUER"],
                                                                 defenders=["SABA", "TARNAT"], forwards=["ELBER"])},
                {"name": "FC GÜTERSLOH", "players": players(goalkeepers=["NIEMAND"])},
            ])
            j.CACHE, j.OFFLINE = os.path.join(out, ".cache"), True
            with contextlib.redirect_stdout(io.StringIO()):
                got, failed = g.season(None, 1997, seasons)
        self.assertEqual(failed, [])
        self.assertEqual(got, {"FC BAYERN MÜNCHEN": {
            "goalkeepers": ["KAHN", "DREHER", "SCHEUER"], "defenders": ["TARNAT", "SABA"],
            "midfielders": [], "forwards": ["ÉLBER"]}})


if __name__ == "__main__":
    unittest.main()
