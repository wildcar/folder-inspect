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
- Standard library for walking, hashing, JSON, HTTP; a small number of vetted third-party
  modules where they clearly pay off (YAML config, XLSX export). See ADR-0001.
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
- FR-1 ⏳ Scan one or more roots given on the command line; a root may be a local path or a
  mounted network path (Windows UNC included).
- FR-2 ⏳ Do not follow symlinks or junctions; report them but never traverse or modify them.
- FR-3 ⏳ Skip and never touch system locations (`$RECYCLE.BIN`, `System Volume Information`,
  the tool's own quarantine folder).
- FR-4 ⏳ Exclusions: paths and glob patterns via config file and flags.
- FR-5 ⏳ Config file `.folder-inspect.yml` in the scanned root and/or user home; flags
  override file values; sane built-in defaults when no config exists.

### Detector: oversized files (graded by kind)
- FR-10 ⏳ Size rules are a table *file-kind → threshold*, fully configurable. Defaults:

  | Kind | Extensions (default) | Threshold |
  |---|---|---|
  | any file ("huge": video recordings, distributives) | `*` | 100 MB |
  | text documents | doc docx rtf odt | 15 MB |
  | presentations | ppt pptx odp | 15 MB |
  | spreadsheets | xls xlsx ods | 15 MB |
  | PDF | pdf | 30 MB |
  | images | jpg jpeg png gif bmp tif tiff heic webp | 5 MB |

- FR-11 ⏳ A file is reported once, under the most specific matching rule, with the rule name.
- FR-12 ⏳ Report top-N largest files and top-N heaviest folders regardless of thresholds.

### Detector: archives and distributives
- FR-13 ⏳ Archives by extension: zip rar 7z tar gz tgz bz2 xz z cab arj lzh iso img.
  Rationale: project documents must be directly openable by link; an archive signals a
  distributive, an attempt to keep old versions, or a bundle prepared for sending.
- FR-14 ⏳ Office and similar container formats (docx xlsx pptx odt jar apk) are **not**
  archives.
- FR-15 ⏳ Installers/distributives (exe msi msix appx deb rpm dmg pkg) are reported as a
  separate "distributive" category next to archives. On by default.

### Detector: junk
- FR-16 ⏳ Built-in patterns: `*.bak *.tmp *.temp *.old *.orig *.swp`, Office lock/temp files
  `~$*` and `~*.tmp`, `Thumbs.db`, `desktop.ini`, `.DS_Store`, `._*`, `.Spotlight-V100`,
  `.Trashes`, `*.crdownload *.part` (unfinished downloads).
- FR-17 ⏳ User-defined junk patterns in gitignore-like glob syntax via config.
- FR-18 ⏳ Empty folders and zero-size files.

### Detector: duplicates
- FR-19 ⏳ Exact duplicates by content: group by size → hash first 64 KB → full SHA-256.
  Across all scanned roots. Minimum size 1 KB (configurable).
- FR-20 ⏳ Near-duplicates by name ("copy candidates"), reported separately and never
  auto-actionable unless content also matches. Patterns (case-insensitive, RU + EN):
  `Копия <name>`, `<name> - копия`, `<name> - копия (N)`, `Copy of <name>`, `<name> (N)`,
  `<name> - Copy`, `<name>_v2 / _v3 / _final / _старый / _old / _new / _новый`,
  `<name> (Восстановлен)` / `(Recovered)`. ❓ list to be refined with the owner.
- FR-21 ⏳ For every duplicate group show the wasted size. The canonical file (the one that
  stays) is **picked by the user in the web UI**; the UI pre-selects the oldest by
  modification time as a suggestion. Nothing happens to a group without a pick.

### Reporting
- FR-30 ⏳ `report.json` — native result: metadata (roots, time, config, tool version) and all
  findings with path, size, mtime, category, rule, group id.
- FR-31 ⏳ Console summary after a scan: counts and sizes per category, top findings.
- FR-32 ⏳ Web UI (`folder-inspect ui`): localhost page in the default browser, findings by
  category, sort/filter/search, duplicate groups, tick boxes to assemble an action plan,
  export buttons. Embedded into the executable; no external resources.
- FR-33 ⏳ Exports: CSV, XLSX, self-contained HTML.
- FR-34 ⏳ User-facing text in Russian and English; selectable by flag, auto-detected from OS.

### Actions
- FR-40 ⏳ Read-only by default. Any change to the file system happens only through
  `apply` on a plan the user has reviewed.
- FR-41 ⏳ `plan` produces `plan.json` (from UI selection or from rules such as
  "all junk", "all archives"); `apply --dry-run` prints what would happen; `apply` executes.
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
- NFR-3 ⏳ Non-zero exit code on scan errors (unreadable paths) — reported, not fatal.

## Project structure

⏳ To be laid out with the Go scaffold. Intended:

```
cmd/folder-inspect/     entry point, CLI commands
internal/scan/          walker, file index
internal/detect/        one package per detector (size, archive, junk, dup, names, empty)
internal/report/        JSON model, console summary, exports (csv, xlsx, html)
internal/ui/            embedded web UI (static HTML/JS) + local HTTP handlers
internal/action/        plan, apply, quarantine, restore, pointer stubs
internal/config/        YAML config, defaults, flag merge
internal/i18n/          RU / EN message catalogs
testdata/               fixture generator for a dirty repository
docs/                   ADRs, questionnaire, user docs
```

## Deployment

- Single executable per OS, built with `go build`, published via GitHub Releases. No secrets.

## Current state

- ✅ Discovery fully answered; stack decided (ADR-0001); this contract v0.2. License: MIT.
- ⏳ Go toolchain not yet installed on the dev host; project scaffold pending.
- ❓ Minor: more near-duplicate name patterns from practice; optional `.lnk` next to the stub.

See `AGENTS/STATE.md` for the live Now / Next snapshot.

## Data sources & dependencies

- Local or mounted file systems only. No external services, no network.
