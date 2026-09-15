# History

Newest first. Each entry ≤5 lines using the format defined in `AGENTS.md`.

---

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
