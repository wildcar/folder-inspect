# folder-inspect

Utility for inspecting a **project document repository** — a folder tree (local disk or
mounted network share) that stores documents of implementation projects — and reporting what
should not be there:

- files that are too large for their kind (any file over 100 MB; documents and presentations
  over 15 MB; images over 5 MB — all thresholds configurable);
- archives and distributives (documents must be directly openable, not bundled);
- junk files (`*.bak`, `Thumbs.db`, `.DS_Store`, Office lock files, unfinished downloads…),
  empty folders and zero-size files;
- exact duplicates by content, and copy candidates by name («Копия …», «… (2)», «Copy of …»).

Findings are shown in a local web UI with export to CSV, XLSX and HTML. Clean-up is safe by
design: nothing changes without an explicit, reviewed plan; files go to a restorable quarantine,
duplicates can be replaced with hard links. Not related to git.

Status: **design done, implementation not started.** Stack: Go, single executable for Windows
and Linux (see `docs/adr/0001-stack-and-interface.md`).

## Run locally

```
<not available yet — Go scaffold pending; will be `go run ./cmd/folder-inspect scan <root>`>
```

## Docs

- `AGENTS.md` — entrypoint for AI agents working in this repo (workflow, rules, doc map).
- `AGENTS/SPEC.md` — functional specification (contract).
- `docs/adr/` — architecture decisions.
- `docs/discovery-questionnaire.md` — discovery questionnaire with the owner's answers (Russian).
