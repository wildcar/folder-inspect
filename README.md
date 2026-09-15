# folder-inspect

Utility for inspecting a **project document repository** — a folder tree (local disk or
mounted network share) that stores documents of implementation projects — and cleaning it up
safely. It reports what should not be there:

- files that are too large for their kind (any file over 100 MB; documents, presentations and
  spreadsheets over 15 MB; PDF over 30 MB; images over 5 MB — all thresholds configurable);
- archives and installers/distributives (documents must be directly openable, not bundled);
- junk files (`*.bak`, `Thumbs.db`, `.DS_Store`, Office lock files, unfinished downloads…),
  empty folders and zero-size files;
- exact duplicate files by content, grouped, with the oldest copy suggested as the original;
- duplicate folders: identical folders and folders that share a large part of their content;
- copy candidates by name: «Копия …», «… (2)», «… - копия», «…_v2», «…_final», «(Восстановлен)»
  grouped with the base file (informational — contents may differ).

**Nothing is ever deleted.** You tick what to remove in the web UI; `apply` moves those files
into a dated quarantine folder inside the repository and leaves a short `<name>.removed.txt`
note where each file was: what was removed, when, why (archive, distributive, video, oversized,
or a duplicate with the relative path to the kept original) and how to bring it back.
`restore` puts everything back and removes the notes; `quarantine list|show` inspects the
batches, and only `quarantine purge -yes` deletes quarantined copies for good. Not related to git.

Status: **all MVP features work** — scan, duplicates (files, folders, names), exports, web UI
with rescan/apply/restore, CLI apply/restore/quarantine. Stack: Go, single executable for
Windows and Linux (see `docs/adr/0001-stack-and-interface.md`). License: MIT.

## Run locally

Requires Go 1.27+.

```
go build -o dist/folder-inspect.exe ./cmd/folder-inspect

# try it on a generated demo repository (~170 MB of fake files)
dist/folder-inspect.exe fixture C:\Temp\demo-repo
dist/folder-inspect.exe scan C:\Temp\demo-repo
dist/folder-inspect.exe ui C:\Temp\demo-repo
```

Typical loop on a real repository:

```
dist/folder-inspect.exe scan "D:\Проекты"          # report → D:\Проекты\.folder-inspect\reports\report-<date>.json
dist/folder-inspect.exe ui "D:\Проекты"            # browse, pick originals, tick copies/junk, "Check plan", "Apply plan"
dist/folder-inspect.exe restore "D:\Проекты"       # undo the latest quarantine batch (or pass its manifest.json)
```

The same steps without the browser:

```
dist/folder-inspect.exe scan -export xlsx,html "D:\Проекты" "\\server\share\Проекты"
dist/folder-inspect.exe report -format xlsx "D:\Проекты\.folder-inspect\reports\report-2026-09-15_142744.json"
dist/folder-inspect.exe apply -dry-run "D:\Проекты\.folder-inspect\reports\plan-2026-09-15_150639.json"
dist/folder-inspect.exe apply "D:\Проекты\.folder-inspect\reports\plan-2026-09-15_150639.json"
dist/folder-inspect.exe restore -dry-run "D:\Проекты\.folder-inspect\quarantine\2026-09-15_153710\manifest.json"
dist/folder-inspect.exe quarantine list "D:\Проекты"
dist/folder-inspect.exe quarantine show "D:\Проекты\.folder-inspect\quarantine\2026-09-15_153710"
dist/folder-inspect.exe quarantine purge -yes "D:\Проекты\.folder-inspect\quarantine\2026-09-15_153710"   # irreversible
```

Useful flags: `scan -out <file>` (explicit report path; existing files are refused unless
`-force`), `-no-dups` (skip reading file contents), `-quiet`, `-top N`, `-config <file>`,
`-lang ru|en` (console, UI and stub language); `ui -port N`, `ui -no-browser`; `apply -no-stubs`.

Configuration: put `.folder-inspect.yml` into the scanned folder or your home folder — see
`docs/folder-inspect.example.yml` (thresholds, junk patterns, exclusions, duplicate and folder
thresholds, quarantine folder, which categories get a note and your own note texts). Exit code
is 1 when some paths could not be read or some actions failed, 2 on usage errors.

## Docs

- `AGENTS.md` — entrypoint for AI agents working in this repo (workflow, rules, doc map).
- `AGENTS/SPEC.md` — functional specification (contract).
- `docs/adr/` — architecture decisions.
- `docs/folder-inspect.example.yml` — annotated example configuration.
- `docs/discovery-questionnaire.md` — discovery questionnaire with the owner's answers (Russian).
