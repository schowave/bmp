# Remake

- **Screens**: before changing one, read `../docs/re/spec_ui.md`: how to drive the original in the browser and compare pixels, the original's drawing functions, layouts and mouse rules.
- **Decompilation** of BMMAIN.EXE: `tools/re/work/` (`mise run re:build` if missing); query it with `python3 tools/re/q.py` (strings, functions, callers, data constants).
- **Rules** of the game: `../docs/re/spec_*.md`.
- **Screenshots** of the remake: `mise run remake:shot DEMO OUT [SCRIPT]`, every screen with `tools/shots.sh`. Both need an awake display (`caffeinate -u`).
- **Playtesting**, or a bug the user reports from play: `tools/playtest.sh` plays the game fast and reports what it shows, so reproduce the bug there; read `../docs/playtest.md` before reporting a bug from a playtest.
- **Restarting** the game for the user: `mise run remake:restart` (in the background); it ends only the game it started before.
- **Parallel agents** on screens: start each with `isolation: "worktree"`; package `internal/ui` is one namespace, so a shared tree clashes names and breaks the build mid-edit.
- **Checks**: `mise run remake:test` (with `-short`), `mise run remake:test-long` for the tests that play seasons; `mise run hooks` enables the pre-commit build and vet.
