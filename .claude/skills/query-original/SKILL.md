---
name: query-original
description: Looks up how the original BMMAIN.EXE does something in its decompilation (strings, functions, callers, data constants). Use when porting or checking a rule or screen of the remake, when a spec in docs/re is unclear, or when code names a function f_SSSS_OOOO.
---

# Query the original

The specs in `docs/re/spec_*.md` come first; go to the decompilation when they
are silent or in doubt, and correct the spec with what you find.

1. **Make sure the work directory exists**: `tools/re/work/` (gitignored).
   Without it run `mise run re:build`; the decompilation needs `GHIDRA` set,
   the disassembly does not.

2. **Query**, from the repository root:

   | Question | Command |
   | --- | --- |
   | Where is a text used? | `python3 tools/re/q.py str PATTERN` |
   | What does a function do? | `python3 tools/re/q.py fn f_SSSS_OOOO` (`--full` keeps declarations) |
   | Which data segment is meant? | `python3 tools/re/q.py lst f_SSSS_OOOO` |
   | Who calls it? | `python3 tools/re/q.py calls f_SSSS_OOOO` |
   | A table of constants | `python3 tools/re/q.py seg D 24c4 '<5h'` (or `str`, `ptrs:N`) |

   Notation and pitfalls (segments, umlaut encoding, 32-bit helpers) are in
   `tools/re/CONTEXT.md`. Ghidra often gets 32-bit arithmetic wrong: check
   formulas against the listing.

3. **Name the function** in a comment of the code you port (`f_SSSS_OOOO`), and
   keep the order of `rand()` calls (`game.Rand`, ADR 0003).
