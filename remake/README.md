# Bundesliga Manager Professional — Go remake

A re-implementation of the 1990s DOS football management game *Bundesliga
Manager Professional* in Go with [Ebitengine](https://ebitengine.org).

The rules were reverse-engineered from the original `BMMAIN.EXE` (see the
specifications in [`../docs/re`](../docs/re)). The pictures, fonts and the
club database are not redrawn: they are extracted from the original files
in [`../bmp`](../bmp) into open formats in
[`internal/embedded/data`](internal/embedded/data) and built into every binary. Screens are drawn on a 320×240 canvas like the
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

`mise run remake:restart` builds and starts it and ends the game it started
before, for trying out a change. Or run it straight from the sources with `mise run remake:run` (`go run
./cmd/bmp`).

### Windows and Mac programs

`mise run remake:dist` builds into `bin/dist`:

- `bmp.exe`, the game for 64-bit Windows, which cross-compiles on any system.
- On macOS only: `Bundesliga Manager Professional.app`, one binary for Apple
  Silicon and Intel, and the same app zipped for passing on.

The game files are built in, so each is a single file; saves go to the user's
config directory (`%AppData%\bmp`, `~/Library/Application Support/bmp`). The
programs have no signature from a developer account: Windows SmartScreen warns
on the first start, and on another Mac the app opens only with right click →
*Open*. They carry the original files of Software 2000.

The icon of both programs and of the game's window is `cmd/bmp/icon.png`: the
title banner (PIC/4.VGA) in two lines on the marble of PIC/22.CP.
`mise run remake:icon` (`go run ./cmd/bmpicon`) draws it anew from the
embedded files. For the `.exe`, the task runs go-winres (`go run
github.com/tc-hib/go-winres@v0.3.3`), which links the icon and the version of
`VERSION` into it; the app gets an `.icns` made with `iconutil`.

### The original files

`internal/embedded/data` holds what the game needs from `../bmp`, converted
into open formats (`Extract` in `internal/assets/extract.go`, decision in
[ADR 0005](../docs/adr/0005-extract-into-open-formats.md)). It is checked in
and embedded by every build, so neither building nor running needs the game
directory.

| File | From the original | Content |
|---|---|---|
| `clubs.json` | `MANA.DAT` | 200 clubs: name, three strength values; the 64 German ones also logo (picture number) and 20 player names by position (2 goalkeepers, 6 defenders, 7 midfielders, 5 forwards). Names in UTF-8, limited to code page 437 and 22 (clubs) or 25 (players) characters |
| `pic/N.png`, `pic/full/N.png` | `PIC/N.VGA`, `PIC/N.CP` | Pictures as indexed PNGs: the pixels are palette indices, the colour table is palette 0 for viewing. The club logos are `pic/80.png` to `pic/143.png` |
| `palettes.json` | `PIC/1.PAL` | The three palettes, 256 colours `#rrggbb` each |
| `scenes/N.T.json` etc. | `TORE/N.T` etc. | Goal scenes, one frame per line: panorama scroll and 27 objects `[x, y, sprite]` |
| `scenes/counts.json` | `TORE/ANZAHL` | Numbers of normal, penalty and joke scenes |
| `fonts.json` | `BMMAIN.EXE` | The bitmap fonts |

`mise run remake:extract` writes the directory anew from `../bmp`; a test
fails when the two differ, another that the conversion loses nothing.
`-data DIR` or `$BMP_DATA` makes the program read an extract from another
directory, for example one with edited clubs or logos (`go run
./cmd/bmpextract -out DIR` writes one to start from). An edited picture must
stay an indexed PNG.

## Command-line options

| Flag | Default | Purpose |
|---|---|---|
| `-data DIR` | `$BMP_DATA`, else the built-in files | Directory of an extract of the original (see [The original files](#the-original-files)) |
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

### Hidden keys (cheats)

Like the original, the options screen reacts to a few keys that are not
shown anywhere (`f_262f_01b8`, `Cheat` in `internal/game/cheats.go`). Open
the screen with OPTIONEN in the disk menu or with Space on a match day and
type the letter; the frame of the screen flashes briefly when the key was
taken. The keys act on the manager who is on turn.

| Key | Effect |
|---|---|
| `0` to `5` | Sets the level to that digit, as shown in the header of the screen (`LEVEL n`); `0` is the easiest |
| `s` | Every player of the squad gets tendency 68 and fatigue 50 |
| `a` | Like `s`, and all three skills of every player become 99 and the account 99,999,999 DM |
| `g` | Adds 200,000 DM to the account |
| `d`, `p` | Keys of the original without an effect in the remake, which has no state for them (`d` clears the table of loaded saves, `p` sets a flag) |

The level selects the high-score table (`HIGH.<length><level>`), so a game
whose level was changed here is entered in the table of the new level.

The remake adds keys of its own, which the original does not have (why they
rate by the engine of the original: [ADR 0006](../docs/adr/0006-rate-lineups-by-the-engine-of-the-original.md)):

| Where | Key | Effect |
|---|---|---|
| Tactics board of the lineup screen in a turn (not the substitution screen of a match) | `o` | Sets the lineup with the most expected points against the next opponent |
| Training screen | `o` | Sets the training plan with the strongest eleven over the rest of the season |
| Stadium screen | `o` | Sets the ticket price with the most gate money at the next home match |
| Main menu of a turn | `o` | All three: the training, the lineup, the ticket price |

#### The lineup key

It tries the automatic lineups of the three formations, improves each by
swapping players (also with the bench, never an injured or suspended one) or
moving them to a free cell of their line, and takes the best. The rating is
the expected points per match (2 for a win, 1 for a draw) that the engine of
the original gives the lineup, so it weighs the goals scored against the goals
let in as a result does: the strengths of the lineup as the original
calculates them (`f_0f9d_0002`), the chances of a half (`f_0f9d_08e9`) and the
odds of a goal (`f_0f9d_0c3b`), in `internal/game/lineupmodel.go`; a test
checks it against a run of the dice of the engine. Cards, injuries,
substitutions and the 9 goals a side stops at are left out. Nobody is fielded
out of place: the goalkeeper stays in goal and a field player in his own line
or next to it. A tired player (Erschöpfung 130 or more) plays only if he is
worth 0,05 points more than who would play for him: a match adds 7 to 10 to
the Erschöpfung of a player in the team, and above 135 his training does
little and an injury in training gets likelier. The rating in the note does
not count this rule, so it can be lower than before when a tired player rests.

- With a match today the rating is against the next opponent in his venue:
  the strengths of the clubs change before the turn and not again before the
  match, so he plays as the strengths screen shows him. The lineup values
  of the manager have dice of their own (a wrong side or position, no
  goalkeeper), so the rating is the mean of eight fixed series of them. The
  note names the opponent: `GEGEN` for a home match, `BEI` for an away
  match. An opponent with a human
  manager counts as no opponent, because his lineup is not known yet.
- Without a match, or against a human, the rating is the mean over every other
  club of the league, home and away, and the dice count with their mean. One club
  would not do there: the odds of a goal of the original have a step (with the
  attack 0 to 3 points behind the defence a chance is a goal for sure, one
  point ahead it is two in three), which the search would use for nothing
  against the others.

Each key leaves a note of what it did instead of a message, as long as the
screen is open: in the footer of the lineup screen (while no hint is shown
there; beside the tactics board only the parts that fit), under the intensity
of the training screen, below the table of the stadium screen, and in the
middle of the main menu until the turn ends. The lineup note gives the points
with a decimal comma (`OPTIMAL 1,13`): the rating of the new lineup and the
ratings of the automatic lineups of the three formations (`STAND.`: 4-4-2,
3-5-2, 3-4-3), against the same opponent or clubs. A change counts only if it raises the rating by 0,03 points (the model
is not more exact), so the lineup is left alone where nothing is gained. The
lineup stays automatic if the automatic one of the formation is already the
best, otherwise it becomes manual.

#### The training key

It sets the 20 training points, the intensity and the 10 position points. It
plays the rest of the season with the training of the original (`develop`,
the injuries in training and the Erschöpfung of the matches) on copies of the
game for each plan it tries: every split of the training points at intensity
6 and 10 over a part of the season, then every intensity for the best splits
and single moves of the position points over all of it
(`internal/game/trainingopt.go`). The strength is that of the best goalkeeper
and the ten best field players, (Kondition + Technik + Form) / 3 less the
Erschöpfung as the lineup counts it; the note names it for the new plan and
the old one (`OPTIMAL 41,0  BISHER 37,5`). Training camps and the youth work
are left alone. The plan looks to the end of the season, so pressing it again
now and then is enough: after transfers or long injuries.

#### The ticket price key

It tries every price of the screen (5..25 DM) and keeps the one with the most
expected gate money, attendance × price, at the next home match: today's if
it is at home, else the next home match of the league. The attendance is that
of the original (`f_0f9d_11e0`, `internal/game/ticketopt.go`), the mean of 32
fixed series of its dice. The note names the price, its gain against the
price before and the gate money (`EINTRITT 12 +12.157  EINNAHME 132.157`)
and the opponent; without a home match left the price
stays. The price changes nothing else in the original.

#### The key of the main menu

It runs the training key, the lineup key and the ticket price key, and the
note names the ratings with their gain against the ones before
(`AUFSTELLUNG 1,13 +0,41`, `TRAINING 41,0 +3,5`, `EINTRITT 12 +12.157`) and the
opponent of the lineup. A submenu or the panel of the messages covers it; while
the panel with new messages is open, it takes the keys, so close it first.

## Project layout

| Path | Content |
|---|---|
| `cmd/bmp` | The game |
| `cmd/bmpextract` | Extracts the original files into open formats in `internal/embedded/data` |
| `cmd/bmpicon` | Draws the icon `cmd/bmp/icon.png` from the embedded title banner |
| `cmd/scenepreview` | Plays a single goal scene in a window, for development |
| `internal/assets` | Readers of the original files (EXEPACK unpacker, picture format, palettes, the bitmap fonts inside `BMMAIN.EXE`, `MANA.DAT`, goal scenes) and of the extract |
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
mise run remake:test       # go vet, gofmt -l and go test -short -timeout 300s
mise run remake:test-long  # all tests, also those that play whole seasons (CI)
```

Why the season tests are not in `check`: [ADR 0007](../docs/adr/0007-season-tests-outside-check.md).

The tests compare constant tables with the original executable, check the
start finances against the original, play several seasons with random
managers (only `remake:test-long`) and drive the screens headless with the
built-in files. Tests of
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
- When a 1- or 3-year game is over, ENDE under the high scores returns to the
  title screen, where a new game can start; the original quits the program
  (`f_076b_114c` in `f_1a58_3176`).
- The second and third league are called after the real tiers of the
  season: Regionalliga until 1973/74, then Zweite Liga; Amateurliga until
  1977/78, Am.-Oberliga until 1993/94, Regionalliga until 2007/08, then
  3. Liga. The original always says Zweite Liga and Am.-Oberliga.
- Bugs of the match engine that the remake leaves out:
  - In cup matches the original checks the minutes of the chances against
    the person table instead of the chances (`f_0310_12fe`) and draws again
    on a hit; the remake takes every minute at once, like in the leagues.
  - In extra time the original gives cards and injuries in the
    Europapokal to no human team and in the DFB-Pokal only to the home team
    (`f_17c2_1226`); the remake gives them to every human team of a tied
    match.
  - When manager 0 has fewer than 8 players, the original has no cards and
    injuries for any human that day, and a team with fewer than 4 players
    on the pitch ends the events of all matches of the minute
    (`f_0310_2ee5`); the remake decides per match.
  - When the away manager of a match of two humans comes first by index and
    forfeits, the original returns before the home manager
    (`f_1a58_20b2`); the remake still books his gate money.
- A new game starts with only the newspaper on: the half-time scores, results
  and tables of all leagues, the goal scenes, the cup screens (DfB-Pokal,
  Europapokale) and the results of the replays (Nachholspiele) off, and the
  speed of the match days at the maximum (70); the original starts with them
  on and speed 40. With the cup screens off the
  draws are not shown, and the results of a cup round only in the last rounds
  of the season (after day 315). All stay changeable in the options screen.
- During a match the lineup screen leaves out the formation buttons below the
  tactics board, which do nothing there; the original draws them. A click
  anywhere in a row of the list substitutes, also on the name; the original
  takes it only on the number and the position. The player taken out leaves
  the board at once (not checked against the original).
- WINTERPAUSE is shown once for each run of days of the winter break between
  two turns. The original also reads a pixel of the screen before it shows
  the flag (`f_3930_043a`), so it may show it less often; this is not
  checked against the running original (`docs/re/spec_ui.md`).
- The timed messages of manager 3 expire like those of the others; the
  original counts down only those of the managers 1 to 3 (index 0..2), so his
  never go.
- The key `o` on the training screen sets an optimal training plan, on the
  stadium screen an optimal ticket price, on the main menu all of them with the
  lineup (see *Hidden keys*).
- The key `o` on the tactics board sets an optimal lineup (see *Hidden keys*).
- When a player is sold to an offer for a squad player, the buyer and the
  strength boost come from the sold player; the original reads the next
  slot after removing him (`f_2277_04a5`).
- A manager added with NEU has no history before he joined; the original
  fills it with empty seasons that lower his rating. His start players come
  from persons 1..150; the original can draw 151, behind the person table.
- The high scores at the game end take a manager once, are sorted stably
  and highlight his row; the original can enter him twice
  (`f_1a58_3176`) and highlights row 18 in the menu.

## Legal

The original game files are copyrighted by Software 2000.
`internal/embedded/data` and every binary contain them; converted into PNG
and JSON they are still the same pictures and data. Share them only where
you would share the `bmp/` directory itself.
