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

- MVP slice 1 shipped 2026-09-15: `folder-inspect scan <root...>` writes `report.json` and prints
  a RU/EN summary; detectors: size rules, archives, distributives, junk, empty; YAML config;
  `fixture` command generates a demo repository. Build/vet/gofmt/tests green (Go 1.27.1).
- Verified manually on the generated fixture (~170 MB) in both languages, with and without `-exclude`.
- Not yet tried on a real project repository or a network share.

## Next

1. Slice 2a — exact duplicates (`internal/detect/dup.go`): size → 64 KB head hash → full SHA-256,
   parallel hashing of candidates only, groups with wasted size in the report and console.
2. Slice 2b — exports: CSV, XLSX (excelize), self-contained HTML; `folder-inspect report <report.json> -format …`.
3. Slice 3 — embedded web UI (`ui` command) with canonical-file pick per duplicate group;
   `plan` / `apply --dry-run` / `apply` / `restore`; quarantine + pointer stubs.
4. Then: near-duplicate names, Windows OS-locale detection, `**` in globs, GitHub Releases build.
5. Ask the owner to run `scan` on a real repository and share the console output (not the paths if sensitive).

## Open questions

- Near-duplicate name patterns — extend from practice as they come up (FR-20).
- Optional Windows `.lnk` next to the pointer stub — only if colleagues ask (FR-43).
- Should the console show paths relative to the root (current) or absolute when several roots are scanned?

## Deferred

- No CI yet (GitHub Actions for build/test/release) — add once slice 2 lands.
- `**` glob support in exclusions and junk patterns.
