---
status: accepted
---

# Extract the original files into open formats

ADR 0002 embeds the files of the original in the remake. They were copied as
they are: `MANA.DAT`, the RLE pictures `PIC/*.VGA` and `*.CP`, `PIC/1.PAL` and
the goal scenes of `TORE/`. Nobody can read, diff or replace these without the
parsers of `internal/assets`, and the club database is to be replaced by
current clubs and players.

## Considered Options

- Convert on extraction: `clubs.json` (names in UTF-8, players by position),
  the pictures as indexed PNGs whose pixels are the palette indices,
  `palettes.json`, one JSON file per goal scene, the fonts as before. The
  remake reads only this format, also with `-data DIR`. Lossless (a test
  compares every file with the original), standard library only.
- Keep the original formats and write a converter for editing: rejected, two
  formats to keep in step, and diffs of the embedded data stay unreadable.
- Read both formats at run time: rejected, every reader exists twice; the
  original is read only by `bmpextract` (`assets.Original`).

## Consequences

- `-data DIR` and `$BMP_DATA` take an extract, not the game directory; the
  error message says how to extract one.
- A replaced picture must stay an indexed PNG; its colour table only serves
  to view it (palette 0), the game draws with the palette of the screen.
- The 320-byte table of `MANA.DAT` that the original's loader skips, and two
  stray bytes after the builder's name in `TORE/3.T` and `3.V`, are not
  extracted.
- The files are still those of Software 2000; ADR 0002's rule on sharing
  binaries stands.
