---
name: compare-screen
description: Compares a screen of the Go remake with the same screen of the original DOS game, pixel by pixel. Use when changing, fixing or reviewing a screen in remake/internal/ui, or when a screen test or layout is in doubt.
---

# Compare a screen with the original

Reference: `docs/re/spec_ui.md` §1 (driver commands, click positions,
pitfalls) and the screen's section in §4. Read both first.

1. **Start the original** (once per session), from the repository root:

   ```sh
   (cd tools/re/refgame/wasm && npm install && npx playwright install chromium)
   tools/re/refgame/wasm/serve.sh --driver
   ```

   Run it in the background and wait for "ci ready". Shots while DOSBox boots or
   while the game builds the season crash js-dos; then restart `serve.sh`.

2. **Shoot the original**: drive it to the screen and shoot it, absolute paths:

   ```sh
   tools/re/refgame/wasm/cmd.sh '[["newgame","TEST"],["shot","/abs/orig.png"]]'
   ```

   Click positions of the menus are in spec_ui.md §1.

3. **Shoot the remake** at the same screen, with a demo and an optional input
   script (`internal/ui/script.go`; demo names in `internal/ui/demo.go`):

   ```sh
   mise run remake:shot DEMO /abs/remake.png [SCRIPT]
   ```

   With a script, its `shot FILE` line writes the picture instead of OUT.
   It needs an awake display (`caffeinate -u`); on Linux `xvfb-run`.

4. **Compare**, from `tools/re/pixcmp`:
   - `python3 diff.py orig.png remake.png diff.png`: number of differing pixels
     and a side-by-side picture with the differences in red. Look at it.
   - `python3 find.py orig.png remake.png x0 y0 x1 y1 [r]`: where a region of
     the remake lies in the original, to measure offsets when the game data differ
     (the demo game is another random game: compare layout, not data).
   - `python3 idx.py` dumps palette indices to compare colours.

5. **Fix and repeat** until the layout matches. A difference you keep on purpose
   goes into spec_ui.md §6 (known differences). Then run `mise run check`.
