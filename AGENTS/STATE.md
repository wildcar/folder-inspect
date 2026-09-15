# State

Current snapshot. Overwrite this file each iteration. Aim for ≤50 lines: keep pointers and
the live picture here; push detail into `AGENTS/SPEC.md` (the contract) and `AGENTS/HISTORY.md`
(the log).

## Goal

Build folder-inspect: a tool that finds large files, archives, junk and duplicates in a
folder tree / git working copies and helps clean them up. See `AGENTS/SPEC.md`.

## Now

- Discovery phase. Repository initialised (2026-09-15), remote `origin` = github.com/wildcar/folder-inspect.
- `docs/discovery-questionnaire.md` written and handed to the owner; waiting for answers.
- No code, no stack chosen.

## Next

1. Collect questionnaire answers (or explicit acceptance of the defaults).
2. Record the stack / interface decision as `docs/adr/0001-stack-and-interface.md`.
3. Turn SPEC ⏳ items into the agreed contract; fill Stack & Commands in `AGENTS.md`.
4. Scaffold the project in the chosen stack; add a "dirty folder" fixture generator for tests.
5. MVP: large files, archives, junk rules, exact duplicates, empty dirs; console + JSON report; read-only.

## Open questions

- All items in `docs/discovery-questionnaire.md` (sections A, B, C). Key blockers: A1 (what is scanned), A10 (report-only vs. cleanup), B1 (interface), B2 (language).
- Should the questionnaire answers be kept as a separate doc or folded into SPEC only?

## Deferred

- Definition of Done steps 1–3 (build, lint, tests) do not apply yet — no code exists.
- License file: pending B9 (default MIT).
