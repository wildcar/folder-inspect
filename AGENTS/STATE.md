# State

Current snapshot. Overwrite this file each iteration. Aim for ≤50 lines: keep pointers and
the live picture here; push detail into `AGENTS/SPEC.md` (the contract) and `AGENTS/HISTORY.md`
(the log).

## Goal

Build folder-inspect: a Go tool that finds oversized files (graded by kind), archives and
distributives, junk, duplicate files and folders, copy candidates by name in project document
repositories, shows them in a local web UI with exports, and cleans up via quarantine with
explanatory stubs. Contract: `AGENTS/SPEC.md` v0.2 — every FR-1…FR-46 is implemented; the
rule-based `plan` command is the only listed item still open.

## Now

- Slice 4 shipped 2026-09-15: similar names (RU/EN copy and version markers, groups with the
  base file marked, exact-duplicate cross-reference), `quarantine list|show|purge` (purge needs
  `-yes`, batch-folder-only, manifest kept), Quarantine view with Restore in the UI, report
  schema 4. Owner confirmed slices 1–3b work on real repositories.
- Verified on the demo fixture: console shows 2 similar-name groups; manual plan with an
  archive and a "(1)" copy → apply → stub for the copy names the base file → list/show →
  purge preview → purge -yes → second purge refused.
- Build/vet/gofmt/tests green (11 packages).

## Next

1. Owner review of the similar-name markers on real data (false positives?) and the stub text for copies.
2. GitHub Actions: build + test on push; release binaries (windows/amd64, linux/amd64) on tag; version via ldflags.
3. `plan` from rules without the UI ("all junk", "all archives") for scripted clean-ups.
4. Smaller: Windows OS-locale detection, `**` globs, per-folder summary note as an option.

## Open questions

- Markers list: add `_итоговый`, `(старая версия)`, `- финал`? Collect from the owner's real names.
- Explorer reveal fix (hand-built command line) needs the owner's confirmation on their machine.
- Purge from the UI stays out by design; revisit only if colleagues ask.

## Deferred

- No CI yet (GitHub Actions for build/test/release).
- `**` glob support in exclusions and junk patterns.
