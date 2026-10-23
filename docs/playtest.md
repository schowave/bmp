# Playtesting the remake

Play the remake through its control mode with `remake/tools/playtest.sh`
(commands in its header). Every command answers with what the game shows:
the scene on top, its text, the date, the place, the money and the latest
message. Read that answer; take a screenshot (`do 'shot FILE'`) only when
the picture itself is in question.

- Scripts: `remake/internal/ui/script.go`; named click targets
  (`click menu.calendar`, `click box.1`, …): `App.target` in
  `remake/internal/ui/control.go`. Other places are canvas pixels
  (320×240).
- `do 'dump FILE'` writes the running game as a save file (JSON), to read
  squads, tables and money exactly; `start -load FILE` continues from one.
- `turn N` and `advance` click through match days with the defaults:
  every period at once, draws not watched, substitutions skipped.

## Simulating seasons

To measure what a strategy is worth (a lineup, training, transfers), play
seasons headless instead of clicking: `playSeason` in
`remake/internal/game/season_sim_test.go` plays a season to the last
matchday with a function before every turn and returns place, points and goal
difference. Compare strategies over several seeds in a test of your own;
`TestOptimalLineupWinsPoints` is an example. `NewManager.Team` is the club in
MANA.DAT order; the club's id and league in the game come from
`Manager.Team` and `Manager.League` after `NewGame`.

## Before reporting a bug

The remake copies the original, bugs included. These looked like bugs in
playtests and are the original:

| Seen | Original rule |
|---|---|
| The weekday does not match the real calendar | `spec_season.md` §2.1 |
| Newspaper: "IM SPIEL A IN B" for an away game | original text `%x10GEGEN#IN#%` (`tools/re/q.py str SAHEN`) |
| A sale credits less than the offer ("Ablöse" deducted) | `spec_transfers.md` §5.1 |
| Clicking an own listed player on the market takes him back | `spec_transfers.md` §5.2 |
| A new player refuses the salary prefilled in the panel | `spec_transfers.md` §8.2–8.4 |
| Monthly expenses far above the salaries (big stadium) | `spec_finance.md` §3.2 |
| The strength in the results drops after a lineup change | `spec_match.md` §3 |
| A refused player stays not negotiable for weeks | `spec_transfers.md` §8.3 |
| The stadium order field ignores typing (click the upper or lower edge) | `f_0000_0602`, `remake/internal/ui/desk_stadium.go` |
| A 1- or 3-year game ends with the high scores, without season report | `spec_season.md` §2.6 steps 7–9 |
| Typing letters in the options screen changes skills, money or the level | cheat keys of `f_262f_01b8`, `remake/internal/game/cheats.go` |
| A goal of a computer club names no scorer | computer clubs have no squads |
| A boundary board without logo still earns 45,000 DM ("KÜNDBAR") | `spec_finance.md` §12 |
| The yellow bar of the file list also covers empty rows | XOR bar of `f_3930_052d` |

Add a row when a suspected bug turns out to be the original.
