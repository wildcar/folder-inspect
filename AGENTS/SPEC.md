# folder-inspect — functional & technical specification

Source of truth for *what the product does* and *how it is built*. Keep it in sync with
reality (update before code when the contract changes).

Status: **draft**. Everything below marked ⏳ is a proposal derived from the initial task
statement and the defaults in `docs/discovery-questionnaire.md`. It becomes a contract only
after the questionnaire is answered.

## Purpose

Inspect a folder tree on a local machine (project directories, git working copies, document
storages) and report what bloats it: very large files, archives, junk files (build output,
caches, temp/OS files), duplicates. Then help the owner clean the storage or repository
safely. Primary user: the repository owner and a small technical team.

## Stack

⏳ Not chosen. Decided by questionnaire section B; recorded as `docs/adr/0001-*.md`.
Questionnaire default: Go, CLI core, single binary, Windows + Linux.

## Architecture

⏳ To be designed after the stack decision. Intended shape (stack-independent):

```
scan roots ──► walker (parallel, symlink-safe) ──► file index
                                                       │
            ┌──────────────┬──────────────┬────────────┼──────────────┐
            ▼              ▼              ▼            ▼              ▼
       large-files     archives      junk rules    duplicates     empty dirs
       detector        detector      (gitignore    (size → partial (and zero-
                                      syntax)       → full hash)    size files)
            └──────────────┴──────────────┴────────────┴──────────────┘
                                          ▼
                              findings ──► report (console, JSON)
                                          ▼
                            action plan (dry-run) ──► apply (quarantine / delete)
```

## Functional requirements

All ⏳ (planned), pending questionnaire answers. Numbering follows questionnaire sections.

### Scanning
- FR-1 ⏳ Scan one or more root folders; recognise git working copies inside the tree.
- FR-2 ⏳ Do not follow symlinks / junctions outside the scan root.
- FR-3 ⏳ Never modify anything under `.git/` or system folders.
- FR-4 ⏳ Exclusions via config file in the root (`.folder-inspect.yml`) and CLI flags.

### Detectors
- FR-10 ⏳ Large files: configurable threshold (default 50 MB; 5 MB inside git repos); top-N largest files and folders.
- FR-11 ⏳ Archives: detect by extension and magic bytes; office formats and jar/apk are not treated as archives.
- FR-12 ⏳ Junk: built-in rule set (dependencies, build output, IDE caches, temp, OS noise) + user rules in gitignore syntax; files matched by the repo's own `.gitignore` are reported as candidates.
- FR-13 ⏳ Exact duplicates by content: size → partial hash → full hash; across all roots; min size 1 KB.
- FR-14 ⏳ Similar names (versions/copies) as a soft, informational report.
- FR-15 ⏳ Empty folders and zero-size files.

### Reporting
- FR-20 ⏳ Console report (tables) and JSON report.
- FR-21 ⏳ Non-zero exit code when findings exceed configured thresholds (CI use).
- FR-22 ⏳ HTML report; scan history and diff between scans — second wave.

### Actions
- FR-30 ⏳ Read-only by default; actions only with an explicit flag.
- FR-31 ⏳ Dry-run produces an action plan that can be saved and applied later.
- FR-32 ⏳ Quarantine (move to a folder with the ability to restore); OS trash / delete; replace duplicates with links — second wave.
- FR-33 ⏳ Suggest `.gitignore` entries — second wave.

### UX
- FR-40 ⏳ User-facing text in Russian and English, selectable by flag, auto-detected from OS.

## Project structure

⏳ To be defined after the stack decision.

## Deployment

- ⏳ Default: single executable published via GitHub Releases. No secrets.

## Current state

- ✅ Repository initialised; docs scaffold; discovery questionnaire written.
- ⏳ Waiting for questionnaire answers → ADR-0001 (stack) → MVP implementation.

See `AGENTS/STATE.md` for the live Now / Next snapshot.

## Data sources & dependencies

- Local file system only. No external services.
