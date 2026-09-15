# Agent Instructions

Primary entrypoint for any agent (Claude, Codex, DeepSeek, etc.) working in this repository.

Keep this file under ~200 lines — it is loaded into context every session. Details belong
in `AGENTS/` docs; repeatable procedures belong in skills under `.claude/skills/`.

## Project

folder-inspect — Go tool that inspects a *project document repository* (a folder tree of implementation-project documents, local or on a mounted share) for oversized files graded by kind, archives, junk, duplicates and near-duplicates; shows findings in a local web UI with CSV/XLSX/HTML export; cleans up via restorable quarantine, leaving a pointer stub where a duplicate was removed.

Not related to git in any way. Stack decision: `docs/adr/0001-stack-and-interface.md`. Contract: `AGENTS/SPEC.md`.

## Environment

- OS / shell: see `AGENTS/ENV.md`
- Commit identity: `wildcar <wildcar@mail.ru>`
- Details, credentials, command cheat-sheet: `AGENTS/ENV.md`

## Document Map

| File | Role |
|------|------|
| `AGENTS.md` | This entrypoint. Workflow, rules, map. |
| `CLAUDE.md` | Compatibility pointer to `AGENTS.md`. |
| `AGENTS/SPEC.md` | Functional specification — source of truth for product behavior. |
| `AGENTS/STATE.md` | Current snapshot: goal, now, next, open questions, deferred. Overwritten each iteration. |
| `AGENTS/HISTORY.md` | Append-only iteration log, newest first. Read only the top few entries. |
| `AGENTS/MEMORY.md` | Durable cross-session memory: working agreements + project facts. The ONLY agent memory store — see Memory. |
| `AGENTS/ENV.md` | Host, tools, credentials, command cheat-sheet. Read on demand. |
| `README.md` | Public-facing readme: what the project is, how to run it locally. |
| `docs/` | Domain / reference docs. Read on demand when a task touches that area. |
| `docs/adr/` | Architecture Decision Records — one file per significant decision (see `docs/adr/TEMPLATE.md`). |
| `docs/discovery-questionnaire.md` | Discovery questionnaire (in Russian, addressed to the owner): task clarification + stack/interface choice, with defaults. |

## Startup Checklist

1. Read `AGENTS.md` (this file).
2. Read `AGENTS/SPEC.md`.
3. Read `AGENTS/STATE.md`.
4. Read top 3–5 entries in `AGENTS/HISTORY.md`.
5. Read `AGENTS/MEMORY.md` (working agreements + durable facts).
6. Check `git status --short` before editing; do not overwrite unrelated user changes.

Open `AGENTS/ENV.md` only when you need environment details. Open the relevant file under `docs/` when the task touches that domain.

## Change Workflow

For every iteration that changes code or behavior:

1. If the functional contract changes — update `AGENTS/SPEC.md` first.
2. Make the changes.
3. Overwrite `AGENTS/STATE.md` to reflect the new current state.
4. Prepend a new entry to `AGENTS/HISTORY.md` using the format below.
5. Commit and push once the Definition of Done (below) is met.

### `AGENTS/HISTORY.md` entry format (≤5 lines, newest first)

```
## YYYY-MM-DD · <short iteration title>
- What: <one line — what changed>
- Why: <one line — reason / task>
- Files: <key paths, comma-separated>
- Next: <one line — what was planned right after>
```

Keep each entry tight. Long explanations belong in commit messages or `SPEC.md`. An optional `Fixes-on-the-fly:` line is fine when an iteration also corrected small things discovered mid-way.

When `HISTORY.md` grows past ~200 lines, move the oldest entries to `AGENTS/HISTORY-ARCHIVE.md` (create it on demand) and keep the recent ones here.

When you ship a deferred item from `STATE.md`, write a normal HISTORY entry and remove the item from `STATE.md`.

Parallel work: only the top-level session writes `STATE.md` and `HISTORY.md`. Subagents and secondary sessions report back instead of editing these files.

## Definition of Done

An iteration is done only when:

1. Build succeeds.
2. Lint / typecheck is clean.
3. Tests pass.
4. SPEC / STATE / HISTORY (and MEMORY, if a durable fact emerged) are updated.

Commands for 1–3 live in Stack & Commands. A step that does not apply yet (e.g. the project has no test suite) goes to `AGENTS/STATE.md` → Deferred instead of being silently skipped.

## Memory

`AGENTS/MEMORY.md` is the **single** store of durable agent memory in this project.
Do not use external or per-tool memory stores (memory directories outside the repo, a
tool's built-in memory, etc.): memory must travel with the repository when cloned.

- Read `AGENTS/MEMORY.md` at the start of every session (see Startup Checklist).
- When you learn a durable fact or a working agreement, append a short bullet there and
  commit it together with the related change.
- Split of concerns: durable facts/agreements -> `MEMORY.md`; current snapshot ->
  `STATE.md`; iteration log -> `HISTORY.md`.

Recording rules — keep these a habit:

- One bullet = one fact; keep it short. Long explanations belong in commit messages or `SPEC.md`.
- For working agreements, add a brief **why** so the rule doesn't look arbitrary.
- Convert relative dates to absolute ("today" → the concrete date).
- Do NOT record what is already in the code, git history, or SPEC/STATE/HISTORY.
- Consolidate from time to time: merge duplicates, drop stale or wrong entries.

## Language Rules

- Source code, technical docs, code comments: English.
- Conversation with the user: Russian.
- End-user UI text: Russian, with ability to extend to other languages.
- Existing docs already written in another language are an established contract — keep editing them in that language; don't silently translate.

## Project Rules

Hard constraints and invariants this project must not violate. Keep each rule one line.

- The product must not depend on, detect, or assume git — the scanned "repositories" are document folders.
- Read-only by default: no file-system change happens outside an explicit `apply` of a reviewed plan.
- The tool never deletes user documents directly; removal means quarantine with a restore manifest. `apply` re-verifies a duplicate against its original right before moving it. The one owner-approved exception (2026-09-15): junk, empty files and empty folders (`quarantine.delete_categories`) are deleted outright — no copy, no stub — after re-checking that empty items are still empty; the manifest records them and `restore` recreates the empty ones.
- Stub texts (`stub.*` keys) are user-facing documents left in the owner's repositories: keep them calm, factual, and free of tool jargon; the owner's own texts from the config win.
- Permanent deletion exists only in `quarantine purge` behind `-yes`, only inside a batch folder, never from the UI.
- Never traverse symlinks/junctions out of a scan root; never modify system folders or the quarantine folder during a scan.
- The product never creates hard links or symlinks; a removed duplicate leaves a human-readable pointer stub instead.
- Nothing the tool writes may silently overwrite an existing file: default names carry a timestamp, explicit paths go through `report.CheckOverwrite` and need `-force`.
- Scan results live in `report.json`; every presentation (console, UI, exports) is derived from it, not from a second scan.
- Web UI assets are plain HTML/JS embedded with `embed`; no Node/npm build step in the toolchain.
- Third-party Go modules only where the standard library clearly falls short (currently: YAML config, XLSX export).

## Stack & Commands

Stack one-liner plus the commands an agent needs on day one. Keep the full cheat-sheet in `AGENTS/ENV.md`; here keep only the essentials.

Stack: Go 1.27 (module `github.com/wildcar/folder-inspect`), standard toolchain, single static binary per OS (Windows, Linux). Third-party modules: `gopkg.in/yaml.v3` (config), `github.com/xuri/excelize/v2` (XLSX export).

```bash
# install      — go mod download
# dev / run    — go run ./cmd/folder-inspect scan <root>          (also: ui, report, plan, apply, restore, quarantine list|show|purge, fixture, version)
# build        — go build -o dist/folder-inspect.exe ./cmd/folder-inspect
# test         — go test ./...
# lint         — go vet ./... && gofmt -l .   (gofmt -l must print nothing)
# demo         — go run ./cmd/folder-inspect fixture /tmp/demo && go run ./cmd/folder-inspect scan /tmp/demo
# package      — scripts/build.sh <version>   (release archives → dist/release/; CI does this on every push)
# release      — git tag -a vX.Y.Z -m vX.Y.Z && git push origin vX.Y.Z   (release.yml publishes the GitHub Release)
```

## Architecture

Pipeline: `pipeline.Run` = `scan.Walk` → `detect.Run` + `detect.Duplicates` (the only detector with I/O) + `detect.DuplicateDirs` → `report.Build` → `<root>/.folder-inspect/reports/report-<ts>.json` → console / `export` (csv, xlsx, html) / `ui` (browser) or `plan` (rules) → `plan-<ts>.json` → `action.Apply` → `<root>/.folder-inspect/quarantine/<ts>/` + `manifest.json` + `<name>.removed.txt` stubs (junk and empty items: deleted outright, manifest only) → `action.Restore`.

```
cmd/folder-inspect/   CLI entry point; one file per command (cmd_scan.go, cmd_report.go, cmd_ui.go, cmd_plan.go, cmd_apply.go, cmd_restore.go, cmd_quarantine.go, cmd_fixture.go)
internal/pipeline/    Run(roots, cfg, version): the whole scan, shared by scan and the UI's rescan
internal/scan/        walker + file index with per-folder aggregates; never follows links; skips system dirs and .folder-inspect
internal/detect/      detectors over the index: size.go (graded rules), ext.go (archives, distributives), junk.go, empty.go, dup.go (size → head hash → full hash, parallel), dirdup.go (identical folders + overlap pairs, derived from dup groups), names.go (copy/version markers → similar-name groups)
internal/report/      Report model (schema v3), console summary, output policy (DefaultDir, DefaultName, UniquePath, CheckOverwrite), shared formatting (HumanSize, Qualifier)
internal/export/      csv.go, xlsx.go (excelize), html.go (html/template, self-contained page)
internal/ui/          localhost server + embedded static page (plain JS): /api/report, /api/export, /api/plan, /api/reveal, /api/rescan, /api/apply, /api/quarantine, /api/restore
internal/action/      plan.go (model + validation), rules.go (plan from rules: categories, keep policies, filters), apply.go (quarantine batches, manifest, re-verification), stub.go, restore.go, quarantine.go (list, describe, purge), options.go
internal/config/      defaults + YAML (.folder-inspect.yml), ByteSize with binary units
internal/glob/        case-insensitive glob matching shared by scan and detect
internal/i18n/        RU (default) / EN message catalogs; a test enforces key parity
internal/fixture/     deterministic "dirty repository" generator for tests and demos (`fixture` command)
docs/folder-inspect.example.yml   annotated example config
planned: internal/ui/ (embedded web UI), internal/action/ (plan, apply, quarantine, restore, pointer stubs)
```

## Code Style

- `gofmt` formatting, `go vet` clean; standard Go naming; errors wrapped with `%w` and context.
- Sizes are `int64` bytes internally (`config.ByteSize` in config/JSON); **binary units** everywhere: 1 MB = 1 048 576 bytes, as Windows Explorer shows. Human formatting only at presentation time (`report.HumanSize`).
- Detectors are pure functions over the file index; no I/O besides hashing in the duplicate detector.
- All name/pattern matching is case-insensitive (`internal/glob`) — users are on case-insensitive file systems.
- Findings carry machine values (`Rule`, `Threshold`, `Detail`); translation happens in the presentation layer via `i18n` keys, never inside detectors.
- Every user-visible string goes through `i18n.T`; add the key to both RU and EN maps (the i18n test fails otherwise). The web UI gets the whole catalog from `/api/report` and uses the same keys (`ui.*` for UI-only labels).
- The UI server binds 127.0.0.1 only and validates every path it acts on (plan, reveal) against the report roots.
- Match the conventions of surrounding code: comment density, naming, idiom.
