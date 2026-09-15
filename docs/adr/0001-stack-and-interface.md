# ADR-0001: Stack and interface — Go CLI core with a local web UI

**Status:** Accepted (stack, platforms, CLI core, JSON as the result format). The *form* of the
user interface (embedded local web UI) is the recommended reading of the owner's "нормальный
интерфейс и возможность выгрузки" and awaits explicit confirmation; if rejected, this part is
superseded by a follow-up ADR.
**Date:** 2026-09-15
**Deciders:** owner (wildcar) via `docs/discovery-questionnaire.md`; agent proposal
**Scope:** whole project — language, build, distribution, interface layering

---

## Context

folder-inspect scans *project document repositories* — folders (local disk or mounted network
share) holding documents of implementation projects: office files, PDFs, images, meeting
recordings. Not source code, not git. Tens of thousands of files per repository. Users are the
owner and colleagues who are not necessarily developers; usage is manual, on demand.

Requirements that drive the choice:

- Windows and Linux.
- Single tool a colleague can run without installing a runtime.
- Fast directory walk and content hashing for duplicate detection.
- A "proper" interface to browse findings and pick actions, plus export of results.
- Clean-up actions must be safe: quarantine with restore, replace duplicates with links.

## Decision

1. **Language: Go** (latest stable, standard toolchain). Produces one static executable per
   OS, cross-compiles from one machine, and the standard library covers file walking, hashing,
   JSON and an HTTP server without heavy dependencies.

2. **Layering: CLI core → JSON result → presentations.**
   `folder-inspect scan <root>` walks the tree, runs detectors and writes a **JSON result
   file** (`report.json`). "JSON output" means exactly this: a machine-readable snapshot of all
   findings that every other part consumes — the console summary, the web UI, exports, and the
   action plan. The user never needs to read it by hand; it makes the scan repeatable and
   decouples analysis from presentation.

3. **Interface: embedded local web UI** — `folder-inspect ui report.json` starts a server on
   `localhost`, opens the default browser, and shows findings by category with filters, sizes,
   duplicate groups, and checkboxes to build an action plan. The UI is compiled into the same
   executable (Go `embed`); nothing is installed, no internet access is needed.

4. **Exports:** CSV and XLSX (for colleagues), self-contained HTML (for sending by e-mail),
   plus the native JSON.

5. **Actions:** `plan` → `apply` with a dry-run by default. Quarantine (move to a dated folder
   with a restore manifest) and duplicate replacement with hard links. Never a direct delete.

6. **Distribution:** single executable per OS via GitHub Releases. No installer in the MVP.

## Consequences

- Easier: one binary, no runtime, fast I/O, straightforward cross-compilation, small
  dependency surface. JSON-first makes CI use and later features (scan history, diff) cheap.
- Harder: Go has weak desktop-GUI options, hence the browser-based UI; a colleague must be
  comfortable with "a page opens in the browser". Front-end is plain HTML/JS embedded in the
  binary — no Node build step, to keep the toolchain single-language.
- Hard links require the same volume and may be unsupported on some network shares; the tool
  must detect this and fall back to report-only for that group.
- Dev host currently has no Go toolchain; it must be installed before the first build.

## Alternatives considered

- **Python** — fastest to write, but distribution on Windows without an installed interpreter
  is awkward for non-developer colleagues; slower walk/hash on large trees.
- **Node / TypeScript** — good for a web UI, but requires Node on every machine or a large
  packaged runtime; hashing throughput lower.
- **C# / .NET** — natural Windows GUI (WPF/WinUI), but a cross-platform GUI is harder and
  Linux is a requirement.
- **Rust** — best performance, but higher development cost for marginal gain at tens of
  thousands of files.
- **TUI instead of web UI** — fine for developers, unfriendly for document-oriented users and
  poor for wide tables of paths and sizes.
- **Native desktop GUI in Go (Fyne, Wails)** — heavier dependencies (cgo, WebView2), harder
  builds; revisit only if the browser UI is rejected.
