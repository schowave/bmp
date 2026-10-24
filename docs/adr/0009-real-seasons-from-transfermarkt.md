---
status: accepted
---

# Real seasons from Transfermarkt, kept as club databases

ADR 0008 built the real seasons from raw data of Wikipedia, Wikidata and
Transfermarkt kept in `remake/data` (CSV, logos) by a builder. The owner of
the repository does not want to keep the raw data, only the seasons the game
reads, and wants to get new seasons from transfermarkt.de with the script
they wrote for it.

## Considered Options

- Keep the 37 seasons in `remake/internal/seasons/data` (club databases as
  JSON, logos as PNG) as the data, with the credits of their logos and of the
  Wikipedia articles of their first build in `remake/internal/seasons/credits`.
  New or updated seasons come from `tools/transfermarkt` (tables and squads
  of the 1. to 3. Liga) and `cmd/bmpseasons update`, which takes what the
  tables lack from the season before.
- Keep the raw data and the builder of ADR 0008: rejected by the owner; it
  doubles the data in the repository.

## Consequences

- The Wikipedia/Wikidata fetcher, the builder and `remake/data` are gone; a
  season can no longer be rebuilt from its sources, only updated.
- `tools/transfermarkt` fetches transfermarkt.de, which blocks automated
  access and whose terms forbid reusing its database; the owner decided to
  keep the script and its results in this public repository.
- A new season takes its foreign clubs, the logos of its clubs and the clubs
  of the places outside the leagues (from 2008/09 on) from the season before;
  a club the seasons do not know gets a plain shield.
- The rules of ADR 0008 for the places stay: overflowing leagues pass clubs
  down, short ones take the best of the tier below, strengths come from goals
  and points per game.
