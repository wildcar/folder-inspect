# History

Newest first. Each entry ≤5 lines using the format defined in `AGENTS.md`.

---

## 2026-09-15 · CI and releases
- What: `.github/workflows/ci.yml` (gofmt, vet linux+windows, tests on ubuntu+windows, cross-build artifact), `.github/workflows/release.yml` (tag `v*` → checks → archives → GitHub Release via `gh` on the runner, pre-release for suffixed tags), `scripts/build.sh` (windows/linux amd64 archives with README/LICENSE/example config + SHA256SUMS, version via ldflags; zip fallbacks for a Windows dev host).
- Why: owner confirmed the Explorer fix and asked for GitHub Actions and releases; colleagues need a downloadable executable.
- Files: .github/workflows/ci.yml, .github/workflows/release.yml, scripts/build.sh, README.md, AGENTS/SPEC.md, AGENTS/ENV.md, AGENTS/MEMORY.md, AGENTS/STATE.md
- Next: first tag v0.1.0 by the owner; rule-based `plan` command.

## 2026-09-15 · Similar names per folder; Explorer reveal fix
- What: `SimilarNames` groups within one folder only (key includes the parent path); `reveal` split into reveal_windows.go (hand-built command line `explorer.exe /select,"path"`) and reveal_other.go.
- Why: owner feedback — repository-wide name groups were noisy; "show in file manager" opened Documents because Go quoted the whole `/select,<path>` argument.
- Files: internal/detect/names.go, internal/ui/reveal_windows.go, internal/ui/reveal_other.go, internal/ui/server.go, tests, AGENTS/SPEC.md
- Next: CI + release binaries.

## 2026-09-15 · Slice 4: similar names, quarantine command, quarantine view
- What: `detect.SplitName`/`SimilarNames` (copy/version markers RU+EN → groups by base name + ext, base file first, dup cross-reference; findings, summary, console, CSV/XLSX/HTML, UI view with ticks → stub naming the base file); `action.ListBatches/Describe/Purge` + `quarantine list|show|purge -yes`; UI `/api/quarantine`, `/api/restore`, Quarantine view with Restore; report schema 4; config `similar_names`.
- Why: owner confirmed slices 1–3b and asked for similar names and a quarantine command.
- Files: internal/detect/names.go, internal/action/quarantine.go, cmd/folder-inspect/cmd_quarantine.go, internal/ui/server.go, internal/ui/static/app.js, internal/report/*, internal/export/*, internal/pipeline/pipeline.go, internal/config/config.go, internal/i18n/i18n.go, AGENTS/SPEC.md, AGENTS.md, README.md
- Next: owner review of markers on real data; CI + release binaries.

## 2026-09-15 · Slice 3b: apply, quarantine, restore, stubs, rescan
- What: `internal/action` apply.go (dated quarantine batches per root, manifest, duplicate re-verification by hash, deeper-paths-first, dry-run), restore.go, stub.go (`<name>.removed.txt` with category reasons, RU/EN via `i18n.TL`, custom `quarantine.stub_texts`), options.go; `pipeline.Run` shared by scan and UI; UI: Rescan, Check plan, Apply plan (confirm → result screen → auto rescan), plan pruning after rescan; CLI `apply`, `restore`; config `quarantine`, `videos`.
- Why: owner asked for slice 3b plus rescan from the UI and stubs for every quarantined file, modelled on their SVN notes ("Distributives were removed…").
- Files: internal/action/*, internal/pipeline/pipeline.go, internal/ui/server.go, internal/ui/static/*, cmd/folder-inspect/cmd_apply.go, cmd/folder-inspect/cmd_restore.go, internal/config/config.go, internal/i18n/i18n.go, AGENTS/SPEC.md, AGENTS.md, README.md, docs/folder-inspect.example.yml
- Next: owner tries the loop on a real repository; then near-duplicate names and a quarantine listing command.

## 2026-09-15 · Slice 3a: duplicate folders, web UI, reports under the root
- What: `detect.DuplicateDirs` (identical folders by signature of file hashes; overlapping pairs with ratio/min-files thresholds and nested-pair suppression), report schema 3, console/CSV/XLSX/HTML sections; `internal/ui` localhost server with embedded page (categories, groups with original pick, plan saving, exports, RU/EN, reveal), `internal/action` plan model; `ui` command; default report path `<root>/.folder-inspect/reports/` with fallback and `-2` suffixing.
- Why: owner asked for slice 3 (web UI), reports under the root, and duplicate folders (full and partial).
- Files: internal/detect/dirdup.go, internal/ui/*, internal/action/plan.go, cmd/folder-inspect/cmd_ui.go, internal/report/*, internal/export/*, internal/i18n/i18n.go, internal/config/config.go, internal/fixture/fixture.go, AGENTS/SPEC.md, AGENTS.md, README.md
- Next: slice 3b — apply / quarantine / restore / pointer stubs.

## 2026-09-15 · MVP slice 2: duplicates, exports, no silent overwrite
- What: `detect.Duplicates` (size → 64 KB head hash → full SHA-256, parallel; groups with suggested original), report schema 2 with duplicate groups and hashing stats, `internal/export` (CSV with BOM and locale separator, XLSX via excelize, self-contained HTML), `report` command, `scan -export`, timestamped default report name + `-force` overwrite guard, `-no-dups`, `duplicates:` config section.
- Why: owner asked for slice 2 and flagged that a second run silently overwrote `report.json`.
- Files: internal/detect/dup.go, internal/export/*, internal/report/*, cmd/folder-inspect/cmd_report.go, cmd/folder-inspect/cmd_scan.go, internal/config/config.go, internal/i18n/i18n.go, AGENTS/SPEC.md, AGENTS.md, README.md, docs/folder-inspect.example.yml
- Fixes-on-the-fly: scan duration now includes hashing; BOM written as bytes (a literal BOM inside a Go string does not compile).
- Next: slice 3 — embedded web UI, then plan/apply/restore with quarantine and pointer stubs.

## 2026-09-15 · Go scaffold + MVP slice 1: scan → report.json + console
- What: Go module, walker (no link following, system dirs skipped, folder aggregates), detectors (graded size rules, archives, distributives, junk, empty), YAML config with ByteSize, RU/EN i18n, report.json schema 1, console summary, `fixture` demo generator, unit + end-to-end tests; README/AGENTS updated with real commands and layout.
- Why: Go installed by the owner (1.27.1, per-user); first usable slice of the contract.
- Files: go.mod, cmd/folder-inspect/*, internal/{scan,detect,report,config,glob,i18n,fixture}/*, docs/folder-inspect.example.yml, AGENTS.md, AGENTS/SPEC.md, AGENTS/ENV.md, README.md, .gitignore
- Next: slice 2 — exact duplicates, then CSV/XLSX/HTML exports.

## 2026-09-15 · Discovery closed → SPEC v0.2, MIT license
- What: final answers recorded — browser UI accepted, xls 15 MB / pdf 30 MB, distributives category on, pointer stub instead of hard links, canonical duplicate picked in UI, quarantine in root; LICENSE (MIT) added; per-user Go install documented.
- Why: owner answered the remaining seven questions; hard links rejected in favour of a visible trace of removed duplicates.
- Files: AGENTS/SPEC.md, docs/adr/0001-stack-and-interface.md, docs/discovery-questionnaire.md, LICENSE, AGENTS/STATE.md, AGENTS/ENV.md, AGENTS/MEMORY.md, AGENTS.md, README.md
- Next: Go installed by owner → scaffold → MVP slice 1.

## 2026-09-15 · Discovery answers → ADR-0001 and SPEC v0.1
- What: recorded owner's answers in the questionnaire; ADR-0001 (Go, CLI core, JSON result, local web UI, exports, quarantine + hard links); SPEC rewritten as contract v0.1 with graded size rules; project rules and stack commands in AGENTS.md.
- Why: questionnaire answered — storages are project document repositories, not git; stack chosen (Go).
- Files: docs/adr/0001-stack-and-interface.md, AGENTS/SPEC.md, docs/discovery-questionnaire.md, AGENTS.md, AGENTS/STATE.md, AGENTS/ENV.md, AGENTS/MEMORY.md, README.md
- Next: install Go on the dev host, then scaffold the project and MVP slice 1 (scan → report.json).

## 2026-09-15 · Repository init and discovery questionnaire
- What: `git init`, remote `origin` → github.com/wildcar/folder-inspect; filled project scaffold (README, SPEC draft, STATE, MEMORY, ENV); wrote discovery questionnaire.
- Why: kick-off — task statement is short, stack/interface must be chosen by questionnaire before any code.
- Files: docs/discovery-questionnaire.md, AGENTS.md, AGENTS/SPEC.md, AGENTS/STATE.md, AGENTS/ENV.md, AGENTS/MEMORY.md, README.md
- Next: collect answers → ADR-0001 (stack) → project scaffold → MVP.
