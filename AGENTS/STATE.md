# State

Current snapshot. Overwrite this file each iteration. Aim for ≤50 lines: keep pointers and
the live picture here; push detail into `AGENTS/SPEC.md` (the contract) and `AGENTS/HISTORY.md`
(the log).

## Goal

Build folder-inspect: a Go tool that finds oversized files (graded by kind), archives and
distributives, junk, duplicate files and folders, copy candidates by name in project document
repositories, shows them in a local web UI with exports, and cleans up via quarantine with
explanatory stubs — by ticks in the UI or by rules from the CLI. Contract: `AGENTS/SPEC.md`
v0.2 — every FR-1…FR-46 including FR-41a and FR-42a is implemented.

## Now

- v0.2.0 released 2026-09-15 (plan from rules, CI, releases). After the tag, on main:
  - delete categories (FR-42a): junk, empty files and folders are deleted outright by `apply`
    (re-checked as empty first), recorded in the manifest, empty ones recreated by `restore`,
    junk counted as gone; status `deleted` for junk-only batches; `delete_categories: []`
    restores the old behaviour;
  - UI: 📂 for each folder of an overlapping pair; folders open in Explorer / file manager
    directly instead of being selected in their parent.
- Verified on the fixture: plan (junk, empty, archive) → apply prints 1 moved / 7 deleted /
  1 stub → list → restore recreates 3 empties, brings the archive back, reports 4 junk as gone.
  UI overlap view checked in the browser (two icons per row).
- Build/vet/gofmt/tests green (11 packages).

## Next

1. Owner tries apply on a real repository (junk deletion, empties restored) → tag v0.2.1.
2. Windows OS-locale detection for the default language.
3. `**` globs in exclusions, junk patterns and plan filters.
4. Owner review of the similar-name markers on real data; stub text for copies.

## Open questions

- Should `.bak` stay in junk now that junk is deleted outright (a .bak may hold content)? Owner
  listed bak as junk at discovery; ask if a real repository shows valuable .bak files.
- Markers list: add `_итоговый`, `(старая версия)`, `- финал`? Collect from the owner's real names.
- Purge from the UI stays out by design; revisit only if colleagues ask.

## Deferred

- linux/arm64 or macOS builds — add to `scripts/build.sh` only if someone asks.
- Per-folder summary note as an option; `-rule <size rule>` filter for `plan`.
