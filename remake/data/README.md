# Raw data of the real seasons

`mise run remake:seasons-fetch` (`go run ./cmd/bmpseasons fetch`) writes
this directory from the German Wikipedia and Wikidata;
`mise run remake:seasons` builds `internal/seasons/data` from all of it,
offline.
Correct a mistake in the CSV files and build again; a later fetch
overwrites the files of the seasons it fetches.

| Path | Content |
|---|---|
| `clubs.csv` | By hand: the clubs of the original with their logo (picture 80–143) and, when set, a name that replaces the one of the tables |
| `seasons/YYYY-YY/tables.csv` | Final tables (the table so far for the running season): tier (1 Bundesliga, 2 2. Bundesliga, 3 the third tier of the year, 4 the Regionalliga from 2008 on), group, place, club (its article), name shown, won, drawn, lost, goals |
| `seasons/YYYY-YY/squads.csv` | Players per club: name, surname, position (TW, ABW, MIT, ANG), league games (-1 unknown), source |
| `seasons/YYYY-YY/logos.csv` | Logo file and colours per club: the logo of the squad article of the season, else the current one of the club article |
| `seasons/YYYY-YY/europe.csv` | Foreign clubs of the European cups (0 Champions League/Landesmeister, 1 Pokalsieger, 2 UEFA-Pokal/Europa League, 3 Conference League) and how often the article links them |
| `seasons/YYYY-YY/articles.txt` | The Wikipedia articles read for the season |
| `seasons/YYYY-YY/missing.txt` | What the fetch could not find: articles, tables, squads with fewer than 11 players |
| `transfermarkt/YYYY-YY.csv` | Squads of Transfermarkt (`bmpseasons import-tm -tm DIR`), apart from the openly licensed data: club (its article), the name Transfermarkt gives it, players in its order with their position; the build prefers them to `squads.csv` |
| `logos/` | The logos as PNG, 76 pixels wide, and `index.csv` with file, licence, author and description page |
| `.cache/` | Every response of the fetch (not checked in): a second fetch reads it instead of the network, except for the running season; `-refresh` fetches everything anew |

## Sources and licences

- **Tables and squads:** the German Wikipedia (season articles of the
  leagues, "Kader der …" articles), licensed
  [CC BY-SA 4.0](https://creativecommons.org/licenses/by-sa/4.0/); the
  articles of each season are listed in its `articles.txt`, their authors
  in the history of each article.
- **Squads where Wikipedia has none, clubs of the European cups:**
  [Wikidata](https://www.wikidata.org), CC0.
- **Logos:** Wikimedia Commons and the German Wikipedia, which hosts only
  free files; most club logos are in the public domain as simple text
  logos. Licence, author and source of every file are in
  `logos/index.csv`. A club without a logo gets a plain shield in its
  colours. Club names and logos are trademarks of the clubs.
- **Squads of Transfermarkt** (`transfermarkt/`): from season files the
  owner of the repository collected from transfermarkt.de themselves and
  imported with `bmpseasons import-tm`; `bmpseasons fetch` does not access
  transfermarkt.de, which blocks automated access. Transfermarkt's terms
  do not allow reusing its database; the data is not openly licensed.
