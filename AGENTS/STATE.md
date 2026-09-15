# State

Current snapshot. Overwrite this file each iteration. Aim for ≤50 lines: keep pointers and
the live picture here; push detail into `AGENTS/SPEC.md` (the contract) and `AGENTS/HISTORY.md`
(the log).

## Goal

Build folder-inspect: a Go tool that finds oversized files (graded by kind), archives and
distributives, junk, duplicates and near-duplicates in project document repositories, shows
them in a local web UI with exports, and cleans up via quarantine with pointer stubs.
Contract: `AGENTS/SPEC.md` v0.2.

## Now

- MVP slice 2 shipped 2026-09-15: exact duplicates (size → head hash → full hash, parallel),
  exports CSV/XLSX/HTML, `report` command, timestamped report names, overwrite protection.
- Verified on the owner's two real repositories together (57 GB, 5 862 files): 693 candidate
  files / 582 MB hashed, 156 duplicate groups wasting 294 MB, no read errors; XLSX and HTML
  exports open. Build/vet/gofmt/tests green.
- Not yet tried on a network share (UNC path).

## Next

1. Slice 3a — embedded web UI (`ui` command): serve `report.json` on localhost, open the browser;
   findings by category with sort/filter/search, duplicate groups with a radio for the original
   (pre-selected: suggested), checkboxes → action plan; export buttons.
2. Slice 3b — `plan` / `apply --dry-run` / `apply` / `restore`: quarantine under
   `<root>/.folder-inspect/quarantine/<ts>/`, manifest, pointer stubs `<name>.duplicate.txt`.
3. Near-duplicate names (FR-20), Windows OS-locale detection, `**` in globs.
4. GitHub Actions: build + test on push, release binaries for windows/amd64 and linux/amd64 on tag.

## Open questions

- Near-duplicate name patterns — extend from practice as they come up (FR-20).
- Optional Windows `.lnk` next to the pointer stub — only if colleagues ask (FR-43).
- Console shows paths relative to the root; with several roots this can be ambiguous — the UI
  and exports show absolute paths. Keep or switch the console to absolute when roots > 1?
- Should `scan` without `-out` write into the current folder (now) or into `<root>/.folder-inspect/reports/`?

## Deferred

- No CI yet (GitHub Actions for build/test/release).
- `**` glob support in exclusions and junk patterns.
