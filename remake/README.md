# Bundesliga Manager Professional — Go remake

A re-implementation of the 1990s DOS football management game *Bundesliga
Manager Professional* in Go with [Ebitengine](https://ebitengine.org).

The rules were reverse-engineered from the original `BMMAIN.EXE` (see the
specifications in [`../docs/re`](../docs/re)). The pictures, fonts and the
club database are not redrawn: they are extracted from the original files
in [`../bmp`](../bmp) into [`internal/embedded/data`](internal/embedded/data)
and built into every binary. Screens are drawn on a 320×240 canvas like the
original.

## Contents

- [Requirements](#requirements)
- [Quick start](#quick-start)
- [Command-line options](#command-line-options)
- [Playing](#playing)
- [Project layout](#project-layout)
- [Development](#development)
- [Deviations from the original](#deviations-from-the-original)
- [Legal](#legal)

## Requirements

- **Go** in the version of [`go.mod`](go.mod)
- **[mise](https://mise.jdx.dev)** for the tasks below (optional; each task
  is a plain `go` command, see [`../mise.toml`](../mise.toml))
- **Linux only:** the GL and X11 headers Ebitengine builds against, e.g.
  `sudo apt-get install libgl1-mesa-dev xorg-dev`

The original game in `../bmp` is only needed to extract the files anew, for
the tests against `BMMAIN.EXE` and for the reverse-engineering tools.

## Quick start

From the repository root:

```sh
mise run remake:build   # = go build -o bin/bmp ./cmd/bmp
remake/bin/bmp
```

Or run it straight from the sources with `mise run remake:run` (`go run
./cmd/bmp`).

### The original files

`internal/embedded/data` holds what the game needs from `../bmp` (which
files: `Extract` in `internal/assets/extract.go`). It is checked in and
embedded by every build, so neither building nor running needs the game
directory.

`mise run remake:extract` writes the directory anew from `../bmp`; a test
fails when the two differ. `-data DIR` or `$BMP_DATA` makes the program read
the files from a game directory instead.

## Command-line options

| Flag | Default | Purpose |
|---|---|---|
| `-data DIR` | `$BMP_DATA`, else the built-in files | Directory of the original game |
| `-saves DIR` | user config directory | Save games and the high-score tables (`HIGH.xy` in the format of the original, so the tables of `../bmp` can be copied there) |
| `-scale N` | `3` | Window size as a multiple of 320×240 |
| `-demo NAME` | | Open a development demo screen |
| `-demos` | | List the demo screens and quit |
| `-shot FILE` | | Save the canvas as a PNG and quit (with `-shot-frame N`, default 10) |
| `-script FILE` | | Play a development input script and quit (see `internal/ui/script.go`) |
| `-control DIR` | | Play the scripts written to `DIR/cmd` while running, for playtests (driver: `tools/playtest.sh`) |
| `-tps N` | `60` | Ticks a second with `-control`; 600 plays ten times as fast |
| `-load FILE` | | Start with a save game |

## Playing

The controls follow the original: the left mouse button selects; the right
mouse button confirms in some places (setup, lineup reserves, sponsors) but
does not leave screens, which close with the HAUPT MENU or WEITER button.

1. **Setup:** click the name field below a face and type the name. Click
   left or right of the middle of the logo strip to scroll it (hold the
   button to keep it moving), and click the empty field above the face to
   give that manager the club in the middle. Up to four managers. A right
   click opens the options (level, game length, historic start), a second
   right click starts the game. NEU in the disk menu adds managers later.
2. **Cup draws** ask whether to watch them ("ABER KLAR" shows the draw).
3. **Main menu:**
   - The icons on the left open the desk (statistics, eternal table,
     history, stadium, bank, sponsors), the leagues (tables, results,
     strengths) and the team (lineup with the pitch to place the players,
     contracts, transfer market, training, training camp, top scorers).
   - On the right: cups and the disk menu (load, save, high scores,
     options, NEU).
   - A click on the calendar sheet ends the turn, a click in the middle
     opens the messages, and a right click on the manager tearing his hair quits
     the game.
4. **Match days** run live with the goal scenes:

   | Input | Effect |
   |---|---|
   | Left button | Plays the rest of the half at once |
   | Space | Opens the options |
   | `r` or `w` after a scene | Replays it |
   | Right button | Names the author of the last scene |

## Project layout

| Path | Content |
|---|---|
| `cmd/bmp` | The game |
| `cmd/bmpextract` | Extracts the original files into `internal/embedded/data` |
| `cmd/scenepreview` | Plays a single goal scene of `TORE/` in a window, for development |
| `internal/assets` | EXEPACK unpacker, picture format, palettes, the bitmap fonts inside `BMMAIN.EXE`, `MANA.DAT` |
| `internal/game` | The rules: calendar, fixtures, match engine, cups, finances, stadium, players, transfers, season end. Deterministic (the random generator of the MS C runtime), saved as JSON |
| `internal/ui` | The screens on the 320×240 canvas |
| `internal/embedded` | The original files built into the binary |
| `testdata/original` | Reference screenshots of the original (see its README) |
| `tools/re` | Reverse-engineering tools and the decompilation of `BMMAIN.EXE` (see [`tools/re/README.md`](tools/re/README.md)) |
| `tools/shots.sh` | Renders every demo screen to a PNG |
| `tools/playtest.sh` | Plays the game through `-control` for playtests (see [`../docs/playtest.md`](../docs/playtest.md)) |

## Development

### Tests

```sh
mise run remake:test    # go vet, gofmt -l and go test -timeout 300s
```

The tests compare constant tables with the original executable, check the
start finances against the original, play several seasons with random
managers and drive the screens headless with the built-in files. Tests of
`BMMAIN.EXE` and of the extraction are skipped when `../bmp` is missing.
The timeout turns a screen that never closes into a failure instead of a
hang.

`mise run hooks` enables the pre-commit hook, which builds, vets and
checks the formatting of the staged remake. CI ([`../.github/workflows/ci.yml`](../.github/workflows/ci.yml))
runs `mise run check` (golangci-lint, then this task and the tests of the save
server in `../wasm`) under xvfb, and uploads all demo screens as the
`remake-shots` artifact and the package docs (`go doc -all`) as
`remake-godoc`.

### Screenshots

```sh
mise run remake:shot DEMO OUT.png [SCRIPT]   # one screen
tools/shots.sh [OUTDIR]                      # every screen, default shots/
```

Ebitengine needs a display: on macOS keep it awake (`caffeinate -u`), on
Linux use `xvfb-run`.

### Working on screens and rules

- **Screens:** [`../docs/re/spec_ui.md`](../docs/re/spec_ui.md) describes how
  to drive the original in the browser and compare pixels, the original's
  drawing functions, layouts and mouse rules.
- **Rules:** `../docs/re/spec_*.md`.
- **Decompilation:** `mise run re:build` writes it to `tools/re/work/`;
  `python3 tools/re/q.py` queries strings, functions, callers and data
  constants.

Notes for coding agents are in [`CLAUDE.md`](CLAUDE.md) and, for the whole
repository, [`../AGENTS.md`](../AGENTS.md).

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

## Legal

The original game files are copyrighted by Software 2000.
`internal/embedded/data` and every binary contain them, so share them only
where you would share the `bmp/` directory itself.
