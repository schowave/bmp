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
