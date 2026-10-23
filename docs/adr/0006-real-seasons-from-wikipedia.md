---
status: accepted
---

# Real seasons from Wikipedia and Wikidata

The player chooses a season from 1990/91 to 2026/27 before the setup and
plays with the clubs, players and logos of that season. The game has fixed
places: 18, 20 and 20 clubs in three leagues, 6 outside them, 136 foreign
clubs, 20 players per club.

## Considered Options

- The German Wikipedia for tables (all tiers and groups, also the Oberliga
  and Regionalliga) and squads ("Kader der …" articles), Wikidata for the
  squads Wikipedia lacks and to recognise the clubs of the European cups,
  logos from Wikipedia/Commons. Fetched once into `remake/data` (CSV and
  PNG, checked in, correctable by hand) and built offline into
  `internal/seasons`. CC BY-SA and CC0, standard library only.
- Wikidata alone: rejected, it lacks the participants of several seasons of
  the 2. Bundesliga and of every Oberliga and Regionalliga season.
- transfermarkt.de: rejected, it blocks automated access and its terms
  forbid extracting its database. The CC0 dataset built from it covers only
  the Bundesliga from 2012 on.
- OpenLigaDB: rejected, ODbL (share-alike) and no squads.

## Consequences

- Leagues with more or fewer clubs than places (1991/92: 20 in the
  Bundesliga; 1991–93: 24 in the 2. Bundesliga; from 1994: 18) pass their
  last clubs down or take the best of the tier below; the third league is
  the Oberliga (until 1993/94), the Regionalliga (until 2007/08) or the 3.
  Liga, its groups merged by place. Second teams of professional clubs are
  left out of the amateur tiers. Strengths come from the goals and points
  per game, spread over the ranges of the original.
- Squads of small clubs are incomplete; their empty slots get random names
  like in the original.
- `remake/data/README.md` lists sources and licences; `logos/index.csv`
  credits every logo.
- A new season means `mise run remake:seasons-fetch -- -from Y -to Y` and
  `mise run remake:seasons`.
