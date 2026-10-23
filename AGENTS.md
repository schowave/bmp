# Bundesliga Manager Professional

The 1990s DOS football management game, playable in the browser from two
container images (VNC and WASM), and a re-implementation of the game in Go
(`remake/`). The original files in `bmp/` belong to Software 2000.

Run the remake with `mise run remake:run`, the images with `mise run vnc:run`
or `mise run wasm:run-docker`; `mise tasks` lists every task.

## Verify your work

Run `mise run check` after every change. It runs golangci-lint
(`.golangci.yml`) over both Go modules, then vet, the gofmt check and the
tests of `remake/` (`remake:test`) and of the save server in `wasm/`
(`wasm:test`). Fix every failure it reports.

The slower stages, which CI runs on every pull request and push to `main`:

- Changes to `vnc/`, `web/` or `bmp/`: run `mise run vnc:bench` (builds the
  VNC image with podman, then checks picture, clicks and idle pausing).
- Changes to `wasm/`, `web/` or `bmp/`: run `mise run wasm:smoke` (builds the
  WASM image, then checks page, bundle and the save API).
- Changes to a screen in `remake/internal/ui`: render it with
  `mise run remake:shot` and compare it with the original as
  `docs/re/spec_ui.md` §1 describes (skill `compare-screen`).

When a check fails, fix the code. Changing a test's expectation, skipping a
test, or loosening `.golangci.yml` needs a reason in the commit message (not
enforced).

Enforced by: the CI job `remake` (`mise run check`), the jobs `vnc` and `wasm`
(bench and smoke test), and the pre-commit hook of `mise run hooks` (build, vet
and gofmt of the staged remake).

## Working with the platform

Use `gh` for pull requests and CI results, for example `gh pr create`,
`gh run list`, `gh run view --log-failed`.

## Definition of done

- The checks above are green.
- New task, flag or command → `README.md` or `remake/README.md` (not enforced).
- A rule of the original found or corrected → the matching
  `docs/re/spec_*.md` (not enforced).
- The remake deliberately differs from the original → *Deviations from the
  original* in `remake/README.md` (not enforced).
- New game term → `docs/glossary.md` (not enforced).
- Decision with real alternatives → new ADR in `docs/adr/` (not enforced).

## Conventions

- The remake reproduces the original, bugs included, unless
  `remake/README.md` lists the deviation. Behaviour comes from the specs and
  the decompilation, not from what seems sensible.
- Code that ports a function of the original names it in a comment
  (`f_SSSS_OOOO`, the address in the unpacked `BMMAIN.EXE`).
- `remake/internal/game` holds the rules and draws nothing (no Ebitengine);
  `remake/internal/ui` holds the screens. Random decisions go through
  `game.Rand`, the generator of the original's C runtime.
- Identifiers and comments are in English; text the player sees is German,
  as in the original.
- Commit messages are one sentence in the third person present: "Fixes …",
  "Adds …".
- Use the terms of `docs/glossary.md` as they are.

## Hard limits

- Ask before releasing (`mise run release`, `gh workflow run`), pushing to
  `main`, or pushing images.
- Ask before changing CI or release workflows, adding a dependency, or
  editing `bmp/`. `remake/internal/embedded/data` is generated: change it only
  with `mise run remake:extract`.
- Stop and ask when the spec and the decompilation disagree, or when matching
  the original would mean copying one of its bugs that `remake/README.md` does
  not mention.
- Keep secrets out of code, logs and prompts; the only ones are the Docker Hub
  credentials in the GitHub secrets of the release workflow.

## Where to find context

- Remake: `remake/CLAUDE.md` (screens, decompilation, playtesting),
  `remake/README.md` (layout, flags, deviations).
- Rules and screens of the original: `docs/re/spec_*.md`.
- Glossary: `docs/glossary.md`.
- Review rules beyond the checks: `CODING_STANDARDS.md`.
- Architecture decisions: `docs/adr/`.
- Images: `README.md`, `wasm/README.md`, `vnc/bench/README.md`.

The agent setup (this file, the `CLAUDE.md` files, `.claude/`) is owned by
David Schowalter (@schowave), who reviews every change to it.
