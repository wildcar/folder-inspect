# State

Current snapshot. Overwrite this file each iteration. Aim for ≤50 lines: keep pointers and
the live picture here; push detail into `AGENTS/SPEC.md` (the contract) and `AGENTS/HISTORY.md`
(the log).

## Goal

Build folder-inspect: a Go tool that finds oversized files (graded by kind), archives and
distributives, junk, duplicate files and folders in project document repositories, shows them
in a local web UI with exports, and cleans up via quarantine with explanatory stubs.
Contract: `AGENTS/SPEC.md` v0.2 — all MVP requirements FR-1…FR-45 are implemented except FR-20
(near-duplicate names) and the rule-based `plan` command.

## Now

- Slice 3b shipped 2026-09-15: `apply` (dry-run, dated quarantine batches with manifest,
  duplicate re-verification), `restore`, per-file stubs `<name>.removed.txt` with reasons by
  category and configurable texts, Rescan and Apply/Check buttons in the UI, `pipeline` package.
- Verified end to end on the demo fixture: UI → plan (archive, distributive, 2 duplicate
  copies) → "Check plan" → CLI `apply` → 4 moved, 4 stubs, manifest → UI Rescan reflects it →
  CLI `restore` → everything back, stubs removed, second restore is a no-op.
- Build/vet/gofmt/tests green (11 packages). Not yet applied on the owner's real repositories.

## Next

1. Ask the owner to try the full loop on a real repository (scan → ui → apply → restore) and
   review the stub wording; adjust `stub.*` texts or add `quarantine.stub_texts` examples.
2. Near-duplicate names (FR-20): «Копия …», «… (2)», «Copy of …», `_v2/_final` as a soft category.
3. `quarantine` command: list batches, show what is inside, empty a batch after confirmation (FR-44).
4. GitHub Actions: build + test on push, release binaries (windows/amd64, linux/amd64) on tag.
5. Smaller: Windows OS-locale detection, `**` globs, `plan` from rules ("all junk").

## Open questions

- Stub file name: `<name>.removed.txt` (current) vs. the owner's per-folder notes — keep per file
  as requested; a per-folder summary note could be added later.
- Should `apply` from the UI require typing the number of items instead of a confirm dialog?
- Overlap thresholds (50 % / 2 files) on real data — 27 pairs looked plausible; revisit after use.

## Deferred

- No CI yet (GitHub Actions for build/test/release).
- `**` glob support in exclusions and junk patterns.
