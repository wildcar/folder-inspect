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

- Discovery closed 2026-09-15: every question answered; ADR-0001 accepted; SPEC v0.2; MIT license added.
- **Blocked on toolchain:** Go is not installed on the dev host. Per-user install command (no admin)
  is in `AGENTS/ENV.md`; the owner was given it in chat.
- No code yet.

## Next

1. Owner installs Go 1.27.x per-user; agent verifies `go version` and records it in `AGENTS/ENV.md`.
2. Scaffold: `go.mod`, `cmd/folder-inspect`, `internal/{scan,detect,report,config}`, fixture generator in `testdata/`.
3. MVP slice 1: `scan` → `report.json` + console summary: size rules, archives + distributives, junk, empty dirs.
4. MVP slice 2: exact duplicates; exports CSV/XLSX/HTML.
5. MVP slice 3: embedded web UI with canonical-file pick; `plan` / `apply` / `restore`, quarantine, pointer stubs.
6. Then: near-duplicate names, RU/EN.

## Open questions

- Near-duplicate name patterns — extend from practice as they come up (FR-20).
- Optional Windows `.lnk` next to the pointer stub — only if colleagues ask (FR-43).
- Size unit for thresholds and display: 1 MB = 1 000 000 or 1 048 576 bytes — decide in the first size-rule commit.

## Deferred

- Definition of Done steps 1–3 (build, lint, tests) — no code and no Go toolchain yet.
