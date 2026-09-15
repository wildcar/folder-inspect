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

- CI and releases added 2026-09-15: `ci.yml` (gofmt, vet linux+windows, tests ubuntu+windows,
  cross-build artifact) and `release.yml` (tag `v*` → GitHub Release with
  `folder-inspect_<ver>_{windows,linux}_amd64` archives + SHA256SUMS). `scripts/build.sh`
  verified locally (both archives, `version` prints the ldflags value). No tag pushed yet —
  cutting v0.1.0 is the owner's call.
- Owner confirmed 2026-09-15: slices 1–4 work on real repositories; the Explorer reveal fix
  works; similar names are grouped per folder.
- Build/vet/gofmt/tests green (11 packages).

## Next

1. First release: owner pushes `v0.1.0` (`git tag -a v0.1.0 -m "v0.1.0" && git push origin v0.1.0`), then check the Release page.
2. `plan` from rules without the UI ("all junk", "all archives") for scripted clean-ups.
3. Owner review of the similar-name markers on real data (false positives?) and the stub text for copies.
4. Smaller: Windows OS-locale detection, `**` globs, per-folder summary note as an option.

## Open questions

- Markers list: add `_итоговый`, `(старая версия)`, `- финал`? Collect from the owner's real names.
- Purge from the UI stays out by design; revisit only if colleagues ask.

## Deferred

- `**` glob support in exclusions and junk patterns.
- linux/arm64 or macOS builds — add to `scripts/build.sh` only if someone asks.
