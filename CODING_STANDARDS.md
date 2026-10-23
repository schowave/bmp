# Coding standards

Rules a reviewer checks in a diff, beyond what `mise run check` enforces.
The conventions in `AGENTS.md` apply as well.

## Screens decide outside `Draw`

In `remake/internal/ui`, what a screen shows (which lines and rows, wrapping,
truncation, which item is under the pointer, hit rectangles) is computed by
functions or methods that take no `*ebiten.Image`; `Draw` calls them and
paints the result. The tests of package `ui` cannot draw, since Ebitengine
images need the running game loop, so a decision made inside `Draw` stays
untested.

Example: `CupScreen.layout` in `remake/internal/ui/cupscreen.go` computes the
title and the rows; `TestCupScreenRelegationTitle` checks the title of the
relegation without drawing.

## Generated data is checked by its test

Skip `remake/internal/embedded/data` in a diff: `mise run remake:extract`
writes it, and `TestEmbeddedFilesMatchTheOriginal` fails when it differs from
a fresh extract. Review the extractor (`remake/internal/assets`) instead.

## A rule of the original is in its spec

A diff whose code relies on a rule of the original (a comment names
`f_SSSS_OOOO`, `D:` or `F:`) finds that rule in `docs/re/spec_*.md`, or adds
it there in the same diff. A rule found while debugging and kept only in a
code comment is lost to the next port.

Example: the substitution after an injury relies on `D:0x664` (players
allowed on the pitch); `docs/re/spec_match.md` §10.4 says how
`f_1a58_1f75` passes it to `f_1ecd_1560`.
