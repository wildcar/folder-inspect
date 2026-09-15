# State

Current snapshot. Overwrite this file each iteration. Aim for ≤50 lines: keep pointers and
the live picture here; push detail into `AGENTS/SPEC.md` (the contract) and `AGENTS/HISTORY.md`
(the log).

## Goal

Build folder-inspect: a Go tool that finds oversized files (graded by kind), archives and
distributives, junk, duplicate files and folders, copy candidates by name in project document
repositories, shows them in a local web UI with exports, and cleans up via quarantine with
explanatory stubs — by ticks in the UI or by rules from the CLI. Contract: `AGENTS/SPEC.md`
v0.2 — every FR-1…FR-46 including FR-41a is implemented.

## Now

- v0.1.0 released 2026-09-15 by the owner via the tag workflow; CI green on main.
- `plan` from rules added 2026-09-15 (FR-41a): `-select` categories, `-duplicates` /
  `-dir-duplicates oldest|newest|shallowest`, `-include/-exclude/-min-size`, `-dry-run`,
  `-quiet` (path only, for scripts), `-out/-force`. Similar names and overlaps are refused
  by design. Verified on the fixture end to end (plan → apply → quarantine list → restore).
- Build/vet/gofmt/tests green (11 packages, 6 new rule tests).

## Next

1. Owner tries `plan -select junk -dry-run` on a real repository; then tag v0.2.0.
2. Windows OS-locale detection for the default language.
3. `**` globs in exclusions, junk patterns and plan filters.
4. Owner review of the similar-name markers on real data; stub text for copies.

## Open questions

- Markers list: add `_итоговый`, `(старая версия)`, `- финал`? Collect from the owner's real names.
- Should `plan` also accept `-rule <size rule name>` (e.g. only `image` oversize)? Wait for a need.
- Purge from the UI stays out by design; revisit only if colleagues ask.

## Deferred

- linux/arm64 or macOS builds — add to `scripts/build.sh` only if someone asks.
- Per-folder summary note as an option.
