# State

Current snapshot. Overwrite this file each iteration. Aim for ≤50 lines: keep pointers and
the live picture here; push detail into `AGENTS/SPEC.md` (the contract) and `AGENTS/HISTORY.md`
(the log).

## Goal

Build folder-inspect: a Go tool that finds oversized files (graded by kind), archives and
distributives, junk, duplicate files and folders in project document repositories, shows them
in a local web UI with exports, and cleans up via quarantine with pointer stubs.
Contract: `AGENTS/SPEC.md` v0.2.

## Now

- Slice 3a shipped 2026-09-15: identical-folder groups and overlapping-folder pairs; embedded
  web UI (`ui` command) with category views, duplicate groups with original pick, plan saving,
  exports, RU/EN; reports/exports/plans under `<root>/.folder-inspect/reports/`.
- Verified in the browser on the demo fixture: summary, duplicate groups, "select copies" →
  plan of 2 items saved as `plan-<ts>.json`. Build/vet/gofmt/tests green.
- Not yet run on the owner's real repositories since folder duplicates were added.

## Next

1. Slice 3b — `apply`: read `plan-<ts>.json`, `--dry-run` listing, then move each path to
   `<root>/.folder-inspect/quarantine/<ts>/<relative path>` with a manifest; pointer stub
   `<name>.duplicate.txt` for `quarantine-duplicate` / `quarantine-dir`; `restore <manifest>`.
2. UI: button "apply" is out of scope for now (apply stays a CLI step the user runs after review).
3. Near-duplicate names (FR-20), Windows OS-locale detection, `**` in globs.
4. GitHub Actions: build + test on push, release binaries on tag.

## Open questions

- Should `ui` offer a "rescan" button (re-run the scan from the browser)? Currently CLI only.
- Overlap pairs: is 50 % / 2 files a good default on real data? Check on the owner's repositories.
- Console paths relative to root vs. absolute with several roots (UI and exports are absolute).

## Deferred

- No CI yet (GitHub Actions for build/test/release).
- `**` glob support in exclusions and junk patterns.
