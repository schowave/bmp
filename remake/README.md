# Bundesliga Manager Professional — Go re-implementation

A re-implementation of the game in Go with [Ebitengine](https://ebitengine.org).
The rules were reverse-engineered from the original `BMMAIN.EXE`
(specifications in [`../docs/re`](../docs/re)); the pictures, fonts and the
club database are read from the original files at runtime. None of them are
part of this repository.

## Run

```sh
cd remake
go run ./cmd/bmp -data ../bmp
```

`-data` is the directory of the original game (the one with `BMMAIN.EXE`,
`MANA.DAT` and `PIC/`). Without it the program looks at `$BMP_DATA`, `./bmp`
and `../bmp`. Save games and high scores go to the user config directory
(`-saves` to change it), `-scale` sets the window size.

## Playing

The controls follow the original: the left mouse button selects, the right
mouse button confirms or goes back.

1. **Setup:** click the name field below a face and type a name, scroll the
   logo strip and click the empty field above the face to give that manager
   the club in the middle of the strip. Up to four managers. Right click
   opens the options (level, game length, historic start), right click again
   starts the game.
2. **Main menu:** the icons on the left open the desk (statistics, eternal
   table, history, stadium, bank, sponsors), the leagues (tables, results,
   strengths) and the team (lineup, contracts, transfer market, training,
   training camp, top scorers). On the right: cups, save/load/high scores,
   and the manager tearing his hair ends the turn.
3. **Match days** run live; *TEMPO* changes the speed, the right mouse
   button jumps to the end, *AUSWECHSELN* substitutes players.

## Layout

| Package | Content |
|---|---|
| `internal/assets` | EXEPACK unpacker, picture format, palettes, the bitmap fonts inside `BMMAIN.EXE`, `MANA.DAT` |
| `internal/game` | The rules: calendar, fixtures, match engine, cups, finances, stadium, players, transfers, season end. Deterministic (the random generator of the MS C runtime), saved as JSON |
| `internal/ui` | The screens on a 320×240 canvas like the original |
| `tools/re` | The reverse-engineering tools |

## Tests

```sh
go test ./...
```

The tests compare constant tables with the original executable, check the
start finances against the original, play several seasons with random
managers and drive the screens headless. Tests that need the original files
are skipped when `../bmp` is missing.

## Deviations from the original

The engine follows the original including several of its bugs (for example
the shared popularity value after a season, the penalty score in two-legged
ties). Deliberate differences:

- No newspaper and no animated goal scenes; the match report is a list.
- The statistics screen has no series and records tables.
- Prices of the training camps use the squad of the manager who asks.
- Original bugs that read uninitialised or stale memory are replaced by the
  intended behaviour. This includes the season-end return of listed
  players, which in the original can remove the wrong squad slot.
- Free players without a club get market skills from a fixed club strength
  (the original reads a byte behind the club table), tuned to the market of
  the reference game.
- The rejection marks of market bids are a field of their own; in the
  original they share a byte with the league goals of the player.
