# folder-inspect — functional & technical specification

Source of truth for *what the product does* and *how it is built*. Update before code when
the contract changes.

Status: **contract v0.2** (2026-09-15) — all questionnaire items answered
(`docs/discovery-questionnaire.md`). Items marked ❓ are minor details left to refine during
implementation. Implementation status: ✅ done / ⏳ planned.

## Purpose

Inspect a **project document repository** — a folder tree (local disk or mounted network
share) that stores documents of implementation projects: office files, PDFs, images, meeting
recordings, occasionally code. Report what should not be there: files that are too large for
their kind, archives, junk files, duplicates and near-duplicates. Then help clean the
repository safely: quarantine with restore, replace removed duplicates with a small pointer
file that names the original, export the findings for colleagues.

Explicitly **out of scope**: git repositories and anything git-specific. The product must not
depend on or assume git.

Users: the owner and colleagues (not necessarily developers). Usage: manual, on demand.
Scale: tens of thousands of files per repository.

## Stack

- Go (latest stable), standard toolchain; one static executable per OS (Windows, Linux).
- Standard library for walking, hashing, JSON, HTTP; third-party modules only where they
  clearly pay off: `gopkg.in/yaml.v3` (config), `github.com/xuri/excelize/v2` (XLSX). See ADR-0001.
- No database: the scan result is a JSON file. No network access. No auth.
- Distribution: GitHub Releases.

## Architecture

```
folder-inspect scan <root...>
   │
   ▼
 walker (parallel, no symlink/junction escape, skips system folders)
   │
   ▼
 file index  ──► detectors ──► findings ──► report.json  (native result)
                  │                            │
   ┌──────────────┼─────────────┐              ├──► console summary
   ▼              ▼             ▼              ├──► ui  (localhost web UI, embedded)
 size rules   archives /    junk rules         ├──► export: csv | xlsx | html
 by file kind distributives (patterns)         │
   ▼              ▼                            ▼
 duplicates   empty dirs /              plan.json (dry-run)  ──► apply
 (hash)       zero-size files                                     │
 near-duplicates (names)                             quarantine + restore manifest
                                                     duplicate → quarantine + pointer stub
```

## Functional requirements

### Scanning
- FR-1 ✅ Scan one or more roots given on the command line; a root may be a local path or a
  mounted network path (Windows UNC included — not yet exercised on a real share).
- FR-2 ✅ Do not follow symlinks or junctions; they are listed under "skipped", never traversed.
- FR-3 ✅ Skip and never touch system locations (`$RECYCLE.BIN`, `System Volume Information`,
  the tool's own `.folder-inspect/` folder).
- FR-4 ✅ Exclusions: glob patterns via config (`exclude:`) and repeatable `-exclude` flag.
  A pattern without `/` matches a file/folder name at any depth; with `/` it matches the
  relative path or is a folder prefix. `**` is not supported yet.
- FR-5 ✅ Config file `.folder-inspect.yml` in the scanned root, else in the user home, else
  `-config`; flags override file values; built-in defaults when no file exists. Unknown keys
  are errors. A list in the file replaces the default list.

### Detector: oversized files (graded by kind)
- FR-10 ✅ Size rules are a table *file-kind → threshold*, fully configurable. Defaults:

  | Kind | Extensions (default) | Threshold |
  |---|---|---|
  | any file ("huge": video recordings, distributives) | `*` | 100 MB |
  | text documents | doc docx rtf odt | 15 MB |
  | presentations | ppt pptx odp | 15 MB |
  | spreadsheets | xls xlsx ods | 15 MB |
  | PDF | pdf | 30 MB |
  | images | jpg jpeg png gif bmp tif tiff heic webp | 5 MB |

- FR-11 ✅ A file is reported once: the rule for its extension is tried first; if that does
  not fire, generic (`*`) rules are tried. The finding carries the rule name and threshold.
- FR-12 ✅ Report top-N largest files and top-N heaviest folders regardless of thresholds
  (`top_n`, default 20; `-top` flag).

### Detector: archives and distributives
- FR-13 ✅ Archives by extension: zip rar 7z tar gz tgz bz2 xz z cab arj lzh iso img.
  Rationale: project documents must be directly openable by link; an archive signals a
  distributive, an attempt to keep old versions, or a bundle prepared for sending.
- FR-14 ✅ Office and similar container formats (docx xlsx pptx odt jar apk) are **not**
  archives.
- FR-15 ✅ Installers/distributives (exe msi msix appx deb rpm dmg pkg) are reported as a
  separate "distributive" category next to archives. On by default.

### Detector: junk
- FR-16 ✅ Built-in patterns: `*.bak *.tmp *.temp *.old *.orig *.swp`, Office lock/temp files
  `~$*` and `~*.tmp`, `Thumbs.db`, `desktop.ini`, `.DS_Store`, `._*`, `.Spotlight-V100`,
  `.Trashes`, `*.crdownload *.part` (unfinished downloads). A matching folder is one finding
  with its total size; files inside are not reported again.
- FR-17 ✅ User-defined junk patterns (case-insensitive globs) via config `junk:`.
- FR-18 ✅ Empty folders (only the top-most of a nested empty chain) and zero-size files.

### Detector: duplicates
- FR-19 ✅ Exact duplicates by content: group by size → hash first 64 KB → full SHA-256 of
  the survivors, hashing in parallel. Across all scanned roots. Minimum size 1 KB
  (`duplicates.min_size`); `duplicates.enabled: false` or `-no-dups` skips it. Groups carry
  a stable id (hash prefix), size, count, wasted bytes and members sorted by mtime.
- FR-20 ⏳ Near-duplicates by name ("copy candidates"), reported separately and never
  auto-actionable unless content also matches. Patterns (case-insensitive, RU + EN):
  `Копия <name>`, `<name> - копия`, `<name> - копия (N)`, `Copy of <name>`, `<name> (N)`,
  `<name> - Copy`, `<name>_v2 / _v3 / _final / _старый / _old / _new / _новый`,
  `<name> (Восстановлен)` / `(Recovered)`. ❓ list to be refined with the owner.
- FR-21 ✅ For every duplicate group show the wasted size. The canonical file (the one that
  stays) is **picked by the user in the web UI** (radio per group, pre-selected: `suggested`
  = oldest by modification time, shown with `*`/★ in console and exports). Copies are ticked
  into the plan explicitly; nothing happens to a group without a pick.

### Detector: duplicate folders (owner request 2026-09-15)
- FR-23 ✅ **Identical folders**: same relative file paths and same file contents, derived from
  the file duplicate groups without extra reads (every file of an identical pair has a same-size
  twin and was hashed). Folders holding a file without a twin, or only files below
  `duplicates.min_size`, are never identical. Nested identical sub-folders of identical parents
  are not reported separately. Groups carry size, files, count, wasted bytes and a suggested
  original (oldest mtime); the UI lets the user pick the original and quarantine the copies.
- FR-24 ✅ **Overlapping folders**: pairs (never ancestor/descendant) whose shared content —
  files with identical contents in both — is at least `folder_duplicates.min_overlap` (default
  50 %) of the smaller folder and at least `min_files` (default 2) files. Pairs fully explained
  by a deeper reported pair or by an identical-folder group are dropped as noise. Reported with
  shared files/bytes and the share of each folder; informational only, no actions on pairs.

### Reporting
- FR-30 ✅ `report-<YYYY-MM-DD_HHMMSS>.json` (schema 3) — native result: tool/version,
  start/finish, roots, the effective config, stats (incl. files/bytes hashed), per-category
  summary (duplicates: groups / wasted bytes; overlaps: pairs / shared bytes), all findings
  (path, rel, root, size, mtime, category, rule, threshold, detail, group), duplicate groups,
  identical-folder groups, overlap pairs, top files/folders, errors (scan + hashing), skipped.
- FR-36 ✅ Reports, exports and plans live in **`<first root>/.folder-inspect/reports/`**
  (owner decision 2026-09-15); the scanner never enters `.folder-inspect`. If that folder
  cannot be created (read-only share) the report falls back to the current folder with a
  notice. Default names get a `-2`, `-3` suffix instead of colliding within one second.
- FR-31 ✅ Console summary after a scan: counts and sizes per category, first 10 findings per
  category, duplicate groups with members and the suggested original, top-10 files and
  folders, read errors, files written. `-quiet` suppresses it. `report <json>` re-prints a
  saved report (`-all` lists everything).
- FR-35 ✅ **Nothing is overwritten silently.** The default report name carries the scan
  timestamp; an explicit `-out` or export path that already exists is refused with a hint to
  add `-force`. All output paths are checked before the scan starts.
- FR-32 ✅ Web UI (`folder-inspect ui <report.json | folder>`): binds 127.0.0.1 on a free
  port (or `-port`), opens the default browser (`-no-browser` to skip), serves the embedded
  page (plain HTML/JS/CSS via `embed`, no external resources). Shows summary, every category
  with sort/filter and tick boxes, duplicate file and folder groups with a radio for the
  original and ticks for copies, overlap pairs, top lists, errors; exports CSV/XLSX/HTML;
  RU/EN switch; "show in file manager" for any path inside the roots. The UI changes nothing
  on disk: ticks become an action plan saved via `POST /api/plan`.
- FR-33 ✅ Exports: CSV (UTF-8 BOM, `;` for RU / `,` for EN, one row per finding), XLSX
  (sheets: summary, findings with filters, duplicates, top files, top folders, errors),
  self-contained HTML (inline CSS, no scripts, no external resources). Written by
  `scan -export csv,xlsx,html` next to the JSON, or later by `report -format <fmt> <json>`.
- FR-34 ✅ User-facing text in Russian (default) and English; `-lang ru|en`; auto-detection
  currently reads `LANG`/`LC_ALL`/`LANGUAGE` only (⏳ real OS locale on Windows).

### Actions
- FR-40 ⏳ Read-only by default. Any change to the file system happens only through
  `apply` on a plan the user has reviewed.
- FR-41 ⏳ Plan: ✅ the UI saves `plan-<ts>.json` (schema 1) next to the report — roots, the
  source report, and actions `quarantine` (any finding), `quarantine-duplicate` (copy +
  original), `quarantine-dir` (folder copy + original folder); validated server-side: every
  path inside a root, never a root itself, originals never quarantined. ⏳ `plan` from rules
  ("all junk"), `apply --dry-run`, `apply`.
- FR-42 ⏳ Quarantine: move a file to `<quarantine>/<YYYY-MM-DD_HHMM>/<relative path>` with a
  manifest; `restore` puts files back. Quarantine folder: `<root>/.folder-inspect/quarantine`
  (overridable in config).
- FR-43 ⏳ Removing a duplicate = move it to quarantine **and leave a pointer stub** in its
  place: a small text file `<original name>.duplicate.txt` next to where the file was, saying
  that the duplicate was removed and where the kept original is, as a path **relative to the
  stub's folder**. Contents (RU/EN by locale): tool name, date, relative path to the
  original, quarantine location, SHA-256. Example:

  ```
  Дубликат удалён инструментом folder-inspect, 2026-09-15 14:02.
  Оригинал: ..\..\Проект X\Договоры\Договор.docx
  Копия перемещена в карантин: .folder-inspect\quarantine\2026-09-15_1402\Проект Y\Договор.docx
  SHA-256: 3f2a…
  ```

  `restore` removes the stub when it brings the file back. No hard links or symlinks are
  created (owner decision 2026-09-15). ❓ Optionally also a Windows `.lnk` shortcut to the
  original — later, if colleagues ask for "double-click opens the original".
- FR-44 ⏳ The tool never deletes user files directly. Emptying the quarantine is an explicit
  separate command with confirmation.

### Non-functional
- NFR-1 ⏳ Tens of thousands of files scan in well under a minute on a local SSD; hashing is
  parallel and limited to duplicate candidates.
- NFR-2 ⏳ Unit tests for every detector; integration tests on a generated "dirty repository"
  fixture; the fixture generator is also used for demos.
- NFR-3 ✅ Exit code 1 on scan errors (unreadable paths) — reported, not fatal; 2 on usage errors.

## Project structure

```
cmd/folder-inspect/     entry point; cmd_scan.go, cmd_report.go, cmd_fixture.go (planned: ui, plan, apply, restore)
internal/scan/          walker, file index with per-folder aggregates
internal/detect/        detectors: size.go, ext.go (archives, distributives), junk.go, empty.go, dup.go (the only one with I/O) (planned: names.go)
internal/report/        Report model + JSON I/O, console summary, overwrite policy, shared formatting
internal/export/        csv.go, xlsx.go (excelize), html.go (html/template, self-contained)
internal/ui/            localhost server (server.go: /api/report, /api/export, /api/plan, /api/reveal) + static/ (index.html, app.js, style.css, embedded)
internal/action/        plan.go: Plan / Action model, validation, plan-<ts>.json (planned: apply, quarantine, restore, stubs)
internal/config/        defaults, YAML loading, ByteSize
internal/glob/          case-insensitive glob matching
internal/i18n/          RU / EN message catalogs
internal/fixture/       deterministic dirty-repository generator (tests + `fixture` command)
docs/                   ADRs, questionnaire, example config
```

## Deployment

- Single executable per OS, built with `go build`, published via GitHub Releases. No secrets.

## Current state

- ✅ Discovery fully answered; stack decided (ADR-0001); this contract v0.2. License: MIT.
- ✅ MVP slice 1 (2026-09-15): `scan` → `report.json` + RU/EN console summary with graded
  size rules, archives, distributives, junk, empty folders/files; YAML config; exclusions;
  `fixture` demo generator; unit + end-to-end tests.
- ✅ MVP slice 2 (2026-09-15): exact duplicates with suggested original (FR-19, FR-21 data),
  exports CSV/XLSX/HTML (FR-33), `report` command, timestamped report names and overwrite
  protection (FR-35). Verified on two real repositories (57 GB, ~5 900 files).
- ✅ Slice 3a (2026-09-15): duplicate folders — identical groups and overlapping pairs
  (FR-23, FR-24); embedded web UI with original pick and plan saving (FR-32, FR-41 plan part);
  reports under `<root>/.folder-inspect/reports/` (FR-36).
- ⏳ Slice 3b: `apply --dry-run` / `apply` / `restore`, quarantine, pointer stubs (FR-40, 42–44).
- ⏳ Then: near-duplicate names (FR-20), OS locale detection, `**` in globs.
- ❓ Minor: more near-duplicate name patterns from practice; optional `.lnk` next to the stub.

See `AGENTS/STATE.md` for the live Now / Next snapshot.

## Data sources & dependencies

- Local or mounted file systems only. No external services, no network.
