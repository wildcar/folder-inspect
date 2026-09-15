# folder-inspect

Utility for inspecting a **project document repository** — a folder tree (local disk or
mounted network share) that stores documents of implementation projects — and reporting what
should not be there:

- files that are too large for their kind (any file over 100 MB; documents, presentations and
  spreadsheets over 15 MB; PDF over 30 MB; images over 5 MB — all thresholds configurable);
- archives and installers/distributives (documents must be directly openable, not bundled);
- junk files (`*.bak`, `Thumbs.db`, `.DS_Store`, Office lock files, unfinished downloads…),
  empty folders and zero-size files;
- exact duplicate files by content, grouped, with the oldest copy suggested as the original;
- **duplicate folders**: identical folders (same files, same contents) and folders that share
  a large part of their content;
- copy candidates by name («Копия …», «… (2)», «Copy of …») — *planned*.

Every scan writes `report-<date>_<time>.json` into `<folder>/.folder-inspect/reports/` (never
overwriting a previous one) and prints a summary in the console. `folder-inspect ui <folder>`
opens the newest report in the browser: findings by category, duplicate groups where you pick
the original and tick the copies, exports to **CSV, XLSX and HTML**, Russian/English. Ticks
become an action plan (`plan-<date>.json`); the UI itself changes nothing on disk.

Planned next: `apply` — moves planned files to a restorable quarantine and leaves a small
pointer file naming the kept original where a duplicate was. Not related to git.

Status: **slice 3a** — scanning, duplicates (files and folders), exports and the web UI work;
apply/quarantine are next. Stack: Go, single executable for Windows and Linux
(see `docs/adr/0001-stack-and-interface.md`). License: MIT.

## Run locally

Requires Go 1.27+.

```
go build -o dist/folder-inspect.exe ./cmd/folder-inspect

# try it on a generated demo repository (~170 MB of fake files)
dist/folder-inspect.exe fixture C:\Temp\demo-repo
dist/folder-inspect.exe scan C:\Temp\demo-repo
dist/folder-inspect.exe ui C:\Temp\demo-repo

# real use: one or more folders; report goes to <first folder>\.folder-inspect\reports\
dist/folder-inspect.exe scan -export xlsx,html "D:\Проекты" "\\server\share\Проекты"
dist/folder-inspect.exe ui "D:\Проекты"
dist/folder-inspect.exe scan -lang en -exclude "Старое" -exclude "*.log" "D:\Проекты"

# later: re-print or export a saved report without scanning again
dist/folder-inspect.exe report "D:\Проекты\.folder-inspect\reports\report-2026-09-15_142744.json"
dist/folder-inspect.exe report -format xlsx "D:\Проекты\.folder-inspect\reports\report-2026-09-15_142744.json"
```

Useful flags: `scan -out <file>` (explicit report path; existing files are refused unless
`-force`), `-no-dups` (skip reading file contents), `-quiet`, `-top N`, `-config <file>`;
`ui -port N`, `ui -no-browser`.

Configuration: put `.folder-inspect.yml` into the scanned folder or your home folder — see
`docs/folder-inspect.example.yml`. Exit code is 1 when some paths could not be read, 2 on
usage errors.

## Docs

- `AGENTS.md` — entrypoint for AI agents working in this repo (workflow, rules, doc map).
- `AGENTS/SPEC.md` — functional specification (contract).
- `docs/adr/` — architecture decisions.
- `docs/folder-inspect.example.yml` — annotated example configuration.
- `docs/discovery-questionnaire.md` — discovery questionnaire with the owner's answers (Russian).
