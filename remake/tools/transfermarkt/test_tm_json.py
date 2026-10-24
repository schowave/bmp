"""Tests of tm_json without the network: the conversions and a run with
--offline from a small cache. They run with the standard library of python3
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

import tm_json as j  # noqa: E402


def squad(*players):
    return [{"name": n, "position": p, "geburtsdatum": "", "marktwert": v} for n, p, v in players]


def row(rank, club, club_id, games="34", goals="60:30", points="70"):
    # The keys are those of the cache files of tm_fetch.table.
    return {"platz": rank, "verein": club, "verein_id": club_id, "spiele": games, "tore": goals, "punkte": points}


class Conversions(unittest.TestCase):
    def test_value(self):
        self.assertEqual(j.value("7,00 Mio. €"), 7e6)
        self.assertEqual(j.value("500 Tsd. €"), 5e5)
        self.assertEqual(j.value("1,20 Mrd. €"), 1.2e9)
        self.assertEqual(j.value(""), -1)

    def test_split(self):
        self.assertEqual(j.split("Rafael van der Vaart"), ("Rafael", "van der Vaart"))
        self.assertEqual(j.split("Ailton"), ("", "Ailton"))

    def test_group(self):
        with contextlib.redirect_stdout(io.StringIO()):
            got = [j.group(p) for p in ("Torwart", "Innenverteidiger", "Defensives Mittelfeld", "Linksaußen", "Mittelstürmer", "Trainer")]
        self.assertEqual(got, ["goalkeepers", "defenders", "midfielders", "forwards", "forwards", None])

    def test_convert(self):
        out = j.convert(squad(("Lars Bender", "Defensives Mittelfeld", "1,00 Mio. €"),
                              ("Sven Bender", "Innenverteidiger", "2,00 Mio. €"),
                              ("Kevin Kuranyi", "Mittelstürmer", "500 Tsd. €"),
                              ("Hans Meier", "Mittelfeld", "3,00 Mio. €")))
        self.assertEqual(out, {"goalkeepers": [], "defenders": ["S.BENDER"],
                               "midfielders": ["MEIER", "L.BENDER"], "forwards": ["KURANYI"]})

    def test_third_tier(self):
        self.assertIn("OBSM", j.third_tier(1993))
        self.assertEqual(j.third_tier(1994), ["RN", "RNO", "RWS", "RS"])
        self.assertEqual(j.third_tier(2007), ["RN", "RS"])
        self.assertEqual(j.third_tier(2008), ["L3"])


class Offline(unittest.TestCase):
    def test_season_from_cache(self):
        with tempfile.TemporaryDirectory() as out:
            cache = os.path.join(out, ".cache", "2010")
            files = {
                "regionalligas.json": ["RLW"],
                "L1.table.json": [row("1", "Borussia Dortmund", "16")],
                "L2.table.json": [row("1", "Hertha BSC", "44")],
                "L3.table.json": [row("1", "Eintracht Braunschweig", "23", goals="81:22", points="85")],
                "RLW.table.json": [row("1", "Preußen Münster", "35"), row("4", "SC Verl", "42")],
                "L1-16.squad.json": squad(("Roman Weidenfeller", "Torwart", "2,00 Mio. €"), ("Mats Hummels", "Innenverteidiger", "9,00 Mio. €")),
                "L2-44.squad.json": squad(("Adrián Ramos", "Mittelstürmer", "3,00 Mio. €")),
                "L3-23.squad.json": squad(("Dominick Kumbela", "Mittelstürmer", "300 Tsd. €")),
                "RLW-35.squad.json": [],
                "CL.teilnehmer.json": [{"verein": "FC Barcelona", "verein_id": "131", "land": "Spanien"},
                                       {"verein": "FC Bayern München", "verein_id": "27", "land": "Deutschland"}],
            }
            for name, data in files.items():
                j.write_json(os.path.join(cache, name), data)
            argv = sys.argv
            sys.argv = ["tm_json.py", "--out", out, "--from", "2010", "--offline"]
            try:
                with contextlib.redirect_stdout(io.StringIO()):
                    j.main()
            finally:
                sys.argv = argv
            with open(os.path.join(out, "2010-11.json"), encoding="utf-8") as f:
                clubs = json.load(f)
            with open(os.path.join(out, "2010-11.europe.json"), encoding="utf-8") as f:
                foreign = json.load(f)
        self.assertEqual([(c["name"], c["league"], c["tier"], c["rank"]) for c in clubs], [
            ("BORUSSIA DORTMUND", "L1", 1, 1), ("HERTHA BSC", "L2", 2, 1),
            ("EINTRACHT BRAUNSCHWEIG", "L3", 3, 1), ("PREUSSEN MÜNSTER", "RLW", 4, 1),
            ("SC VERL", "RLW", 4, 4)])
        self.assertEqual(clubs[0]["players"]["defenders"], ["HUMMELS"])
        self.assertEqual((clubs[2]["games"], clubs[2]["goals"], clubs[2]["points"]), (34, "81:22", 85))
        self.assertEqual(clubs[4]["players"], {g: [] for g in j.GROUP.values()})
        self.assertEqual(foreign, [{"name": "FC BARCELONA", "country": "Spanien", "cup": "CL"}])


if __name__ == "__main__":
    unittest.main()
