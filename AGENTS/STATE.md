# State

Current snapshot. Overwrite this file each iteration. Aim for ≤50 lines: keep pointers and
the live picture here; push detail into `AGENTS/SPEC.md` (the contract) and `AGENTS/HISTORY.md`
(the log).

## Goal

Build folder-inspect: a Go tool that finds oversized files (graded by kind), archives, junk,
duplicates and near-duplicates in project document repositories, shows them in a local web UI
with exports, and cleans up via quarantine and hard links. Contract: `AGENTS/SPEC.md` v0.1.

## Now

- Discovery answered 2026-09-15; ADR-0001 (Go, CLI core + JSON result + local web UI) recorded.
- SPEC v0.1 written as the contract; open ❓ items listed there and in the questionnaire.
- **Blocked on toolchain:** Go is not installed on the dev host — install before scaffolding.
- No code yet.

## Next

1. Install Go on the dev host (owner or agent with permission); record version in `AGENTS/ENV.md`.
2. Scaffold: `go.mod`, `cmd/folder-inspect`, `internal/{scan,detect,report,config}`, fixture generator in `testdata/`.
3. MVP slice 1: `scan` → `report.json` + console summary with size rules, archives, junk, empty dirs.
4. MVP slice 2: exact duplicates; exports CSV/XLSX/HTML.
5. MVP slice 3: embedded web UI; `plan` / `apply` / `restore` with quarantine.
6. Then: hard-link replacement, near-duplicate names, RU/EN.

## Open questions

- Web UI in the browser accepted as the "proper interface"? (ADR-0001, A11)
- Thresholds for xls/xlsx (15 MB?) and pdf (30 MB?). (A6)
- Report installers (exe/msi/deb/rpm/dmg/pkg) as a "distributive" category? (A7)
- Hard links acceptable given "edit one copy = edit all"? (A10)
- Canonical file in a duplicate group: oldest mtime / shortest path / manual pick? (A9)
- Quarantine location: inside root vs. configured folder outside. (A10)
- More near-duplicate name patterns from practice. (A9)
- License: MIT? (B9)

## Deferred

- Definition of Done steps 1–3 (build, lint, tests) — no code and no Go toolchain yet.
- LICENSE file — pending B9.
