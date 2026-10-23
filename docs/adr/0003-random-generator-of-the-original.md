---
status: accepted
---

# Use the random generator of the original

Many rules of the game are random draws (match events, player development, AI
offers). The specs give the order of the `rand()` calls, so that the remake can
be checked against the original.

## Considered Options

- `game.Rand`, the linear congruential generator of the Microsoft C 6 runtime the
  original was built with, called in the original's order: sequences can be
  compared with the original; every random decision of the game goes through it.
- `math/rand`: rejected, results could never be compared with the original.

## Consequences

- The game is deterministic for a seed, which the save games and tests rely on.
- Reordering random calls changes results; a change to the order needs the spec.
