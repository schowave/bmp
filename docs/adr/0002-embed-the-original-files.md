---
status: accepted
---

# Embed the original files in the remake

The remake draws with the pictures, fonts and club database of the original
instead of redrawn ones. It needs them at run time, and `bmp/` is not next to a
binary someone downloads.

## Considered Options

- Extract what the remake needs into `remake/internal/embedded/data` with
  `mise run remake:extract`, commit it and build it into every binary: one file to
  run, tests without `bmp/`; the binary carries copyrighted files.
- Read `bmp/` at run time only: rejected, every run and test would need the
  game directory. It stays possible with `-data DIR` or `$BMP_DATA` (since
  ADR 0005 with an extract instead of the game directory).
- Redraw the assets: rejected, the remake aims to look like the original pixel
  for pixel.

## Consequences

- `remake/internal/embedded/data` is generated; a test fails when it differs from
  a fresh extraction.
- Binaries may only be shared where `bmp/` itself may be (see `remake/README.md`).
