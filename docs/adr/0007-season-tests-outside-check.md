---
status: accepted
---

# Run the tests that play whole seasons outside `check`

`mise run check` runs after every change. Two tests that play whole seasons
(the lineup and training keys) and the stress test of random managers took it
from about 3 to 15 seconds.

## Considered Options

- `remake:test` runs `go test -short`, the tests that play seasons skip
  themselves under `-short`, and `remake:test-long` runs all of them, also as
  a step of the CI job `remake`. Chosen.
- Keep them in `check`: rejected, every change waits 12 seconds more, and the
  tests grow with every strategy they compare.
- Shorter seasons in the tests: rejected, a gain of a strategy shows only over
  whole seasons, and fewer of them make the tests flaky.

## Consequences

- A change to the rules in `remake/internal/game` runs `remake:test-long`
  before it is pushed (`AGENTS.md`); CI runs it for every pull request.
- A new test that plays seasons skips itself under `testing.Short()`.
