# folder-inspect — functional & technical specification

Source of truth for *what the product does* and *how it is built*. Update before code when
the contract changes.

Status: **contract v0.1** (2026-09-15) — derived from the owner's answers in
`docs/discovery-questionnaire.md`. Items marked ❓ are still open with a proposed default;
everything else is agreed. Implementation status: ✅ done / ⏳ planned.

## Purpose

Inspect a **project document repository** — a folder tree (local disk or mounted network
share) that stores documents of implementation projects: office files, PDFs, images, meeting
recordings, occasionally code. Report what should not be there: files that are too large for
their kind, archives, junk files, duplicates and near-duplicates. Then help clean the
repository safely: quarantine with restore, replace duplicates with links, export the findings
for colleagues.

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
                                                     hard-link replacement
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
  | spreadsheets ❓ | xls xlsx ods | 15 MB |
  | PDF ❓ | pdf | 30 MB |
  | images | jpg jpeg png gif bmp tif tiff heic webp | 5 MB |

- FR-11 ⏳ A file is reported once, under the most specific matching rule, with the rule name.
- FR-12 ⏳ Report top-N largest files and top-N heaviest folders regardless of thresholds.

### Detector: archives and distributives
- FR-13 ⏳ Archives by extension: zip rar 7z tar gz tgz bz2 xz z cab arj lzh iso img.
  Rationale: project documents must be directly openable by link; an archive signals a
  distributive, an attempt to keep old versions, or a bundle prepared for sending.
- FR-14 ⏳ Office and similar container formats (docx xlsx pptx odt jar apk) are **not**
  archives.
- FR-15 ❓ Installers/distributives (exe msi msix appx deb rpm dmg pkg) — proposed as a
  separate "distributive" category reported alongside archives. Default: on.

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
- FR-21 ⏳ For every duplicate group show the wasted size and a suggested canonical file
  (❓ default: the one with the oldest modification time; alternatives: shortest path,
  explicit pick in the UI).

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
  manifest; `restore` puts files back. Quarantine folder defaults to `<root>/.folder-inspect/quarantine`
  ❓ (alternative: a folder outside the root, set in config).
- FR-43 ⏳ Replace duplicates with **hard links** to the canonical file (same volume required).
  If the volume or file system does not support hard links, the group is reported as
  not-linkable and left untouched. ❓ Owner to confirm hard links are acceptable: editing one
  linked copy changes all copies.
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
internal/action/        plan, apply, quarantine, restore, hardlink
internal/config/        YAML config, defaults, flag merge
internal/i18n/          RU / EN message catalogs
testdata/               fixture generator for a dirty repository
docs/                   ADRs, questionnaire, user docs
```

## Deployment

- Single executable per OS, built with `go build`, published via GitHub Releases. No secrets.

## Current state

- ✅ Discovery done; stack decided (ADR-0001); this contract v0.1.
- ⏳ Go toolchain not yet installed on the dev host; project scaffold pending.
- ❓ Open: web UI form confirmation, xls/pdf thresholds, installers category, hard-link
  acceptance, canonical-file rule, quarantine location, near-duplicate name patterns.

See `AGENTS/STATE.md` for the live Now / Next snapshot.

## Data sources & dependencies

- Local or mounted file systems only. No external services, no network.
