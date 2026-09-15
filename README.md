# folder-inspect

Utility for inspecting a **project document repository** — a folder tree (local disk or
mounted network share) that stores documents of implementation projects — and reporting what
should not be there:

- files that are too large for their kind (any file over 100 MB; documents, presentations and
  spreadsheets over 15 MB; PDF over 30 MB; images over 5 MB — all thresholds configurable);
- archives and installers/distributives (documents must be directly openable, not bundled);
- junk files (`*.bak`, `Thumbs.db`, `.DS_Store`, Office lock files, unfinished downloads…),
  empty folders and zero-size files;
- exact duplicates by content, grouped, with the oldest copy suggested as the original;
- copy candidates by name («Копия …», «… (2)», «Copy of …») — *planned*.

Every scan writes `report-<date>_<time>.json` (never overwriting a previous one) and prints a
summary in the console (Russian by default, English with `-lang en`). The same report exports
to **CSV, XLSX and a self-contained HTML page** for colleagues. Planned: a local web UI and
safe clean-up — nothing changes without an explicit, reviewed plan; files go to a restorable
quarantine, and a removed duplicate leaves a small pointer file naming the kept original.
Not related to git.

Status: **MVP slice 2** — scanning, duplicates and exports work; web UI and clean-up are next.
Stack: Go, single executable for Windows and Linux (see `docs/adr/0001-stack-and-interface.md`).
License: MIT.

## Run locally

Requires Go 1.27+.

```
go build -o dist/folder-inspect.exe ./cmd/folder-inspect

# try it on a generated demo repository (~170 MB of fake files)
dist/folder-inspect.exe fixture C:\Temp\demo-repo
dist/folder-inspect.exe scan -export html C:\Temp\demo-repo

# real use: one or more folders, report + Excel + HTML next to it
dist/folder-inspect.exe scan -export xlsx,html "D:\Проекты" "\\server\share\Проекты"
dist/folder-inspect.exe scan -lang en -exclude "Старое" -exclude "*.log" "D:\Проекты"

# later: re-print or export a saved report without scanning again
dist/folder-inspect.exe report report-2026-09-15_142744.json
dist/folder-inspect.exe report -format xlsx report-2026-09-15_142744.json
```

Useful flags: `-out <file>` (explicit report path; existing files are refused unless `-force`),
`-no-dups` (skip reading file contents), `-quiet`, `-top N`, `-config <file>`.

Configuration: put `.folder-inspect.yml` into the scanned folder or your home folder — see
`docs/folder-inspect.example.yml`. Exit code is 1 when some paths could not be read, 2 on
usage errors.

## Docs

- `AGENTS.md` — entrypoint for AI agents working in this repo (workflow, rules, doc map).
- `AGENTS/SPEC.md` — functional specification (contract).
- `docs/adr/` — architecture decisions.
- `docs/folder-inspect.example.yml` — annotated example configuration.
- `docs/discovery-questionnaire.md` — discovery questionnaire with the owner's answers (Russian).
