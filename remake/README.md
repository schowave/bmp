# Bundesliga Manager Professional — Go re-implementation

A re-implementation of the game in Go with [Ebitengine](https://ebitengine.org).
The rules were reverse-engineered from the original `BMMAIN.EXE`
(specifications in [`../docs/re`](../docs/re)); the pictures, fonts and the
club database come from the original files in `../bmp`, either built into
the binary or read at runtime.

## Run

```sh
mise run remake:build   # bin/bmp with the original files built in
bin/bmp
```

The build extracts what the game needs from `../bmp` (pictures, `MANA.DAT`
and the fonts of `BMMAIN.EXE`, about 1.4 MB) into
`internal/embedded/data`, which git ignores, and embeds it with the build
tag `embeddata`. The binary then runs without the game directory.

Without the tag, for example with `go run ./cmd/bmp`, the program reads the
files at runtime: `-data` names the directory of the original game, otherwise
it looks at `$BMP_DATA`, `./bmp` and `../bmp`. `-data` also overrides the
built-in files. Save games and high scores go to the user config directory
(`-saves` to change it), `-scale` sets the window size.

The original files are copyrighted by Software 2000. A binary built with
`embeddata` contains them, so share it only where you would share the
`bmp/` directory itself.

## Playing

The controls follow the original: the left mouse button selects; the right
mouse button confirms in some places (setup, lineup reserves, sponsors) but
does not leave screens, which close with the HAUPT MENU or WEITER button.

1. **Setup:** click the name field below a face and type the name, click
   left or right of the middle of the logo strip to scroll it (hold the
   button to keep it moving) and click the empty field above the face to
   give that manager the club in the middle. Up to four managers. Right
   click opens the options (level, game length, historic start), right
   click again starts the game. NEU in the disk menu adds managers later.
2. **Cup draws** ask whether to watch them ("ABER KLAR" shows the draw).
3. **Main menu:** the icons on the left open the desk (statistics, eternal
   table, history, stadium, bank, sponsors), the leagues (tables, results,
   strengths) and the team (lineup with the pitch to place the players,
   contracts, transfer market, training, training camp, top scorers). On
   the right: cups and the disk menu (load, save, high scores, options,
   NEU). A click on the calendar sheet ends the turn; the manager tearing
   his hair quits the game. A click in the middle opens the messages.
4. **Match days** run live with the goal scenes; the left button plays the
   rest of the half at once, the space bar opens the options, 'r' or 'w'
   after a scene replays it, the right button names the author of the last
   scene.

## Layout

| Package | Content |
|---|---|
| `internal/assets` | EXEPACK unpacker, picture format, palettes, the bitmap fonts inside `BMMAIN.EXE`, `MANA.DAT` |
| `internal/game` | The rules: calendar, fixtures, match engine, cups, finances, stadium, players, transfers, season end. Deterministic (the random generator of the MS C runtime), saved as JSON |
| `internal/ui` | The screens on a 320×240 canvas like the original |
| `internal/embedded`, `cmd/bmpextract` | The original files built into the binary (tag `embeddata`) |
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

- The best series of the statistics screen start again every season.
- Prices of the training camps use the squad of the manager who asks.
- Original bugs that read uninitialised or stale memory are replaced by the
  intended behaviour. This includes the season-end return of listed
  players, which in the original can remove the wrong squad slot.
- Free players without a club get market skills from a fixed club strength
  (the original reads a byte behind the club table), tuned to the market of
  the reference game.
- The rejection marks of market bids are a field of their own; in the
  original they share a byte with the league goals of the player.
