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
- The tool never deletes user files directly; removal means quarantine with a restore manifest.
- Never traverse symlinks/junctions out of a scan root; never modify system folders or the quarantine folder during a scan.
- The product never creates hard links or symlinks; a removed duplicate leaves a human-readable pointer stub instead.
- Scan results live in `report.json`; every presentation (console, UI, exports) is derived from it, not from a second scan.
- Web UI assets are plain HTML/JS embedded with `embed`; no Node/npm build step in the toolchain.
- Third-party Go modules only where the standard library clearly falls short (currently: YAML config, XLSX export).

## Stack & Commands

Stack one-liner plus the commands an agent needs on day one. Keep the full cheat-sheet in `AGENTS/ENV.md`; here keep only the essentials.

Stack: Go (latest stable), standard toolchain, single static binary per OS (Windows, Linux). Go is **not installed on the dev host yet** — see `AGENTS/ENV.md`.

```bash
# install      — go mod download
# dev / run    — go run ./cmd/folder-inspect scan <root>
# build        — go build -o dist/ ./cmd/folder-inspect
# test         — go test ./...
# lint         — go vet ./... && gofmt -l .   (gofmt -l must print nothing)
```

## Architecture

Intended layout (no code yet — see `AGENTS/SPEC.md` → Project structure for the full map):

```
cmd/folder-inspect/   CLI entry point and commands (scan, report, ui, plan, apply, restore)
internal/scan/        walker + file index
internal/detect/      detectors: size rules, archives, junk, duplicates, names, empty
internal/report/      JSON model, console summary, CSV/XLSX/HTML exports
internal/ui/          embedded web UI + localhost handlers
internal/action/      plan, apply, quarantine, restore, pointer stubs
internal/config/      YAML config, defaults, flag merge
internal/i18n/        RU / EN messages
testdata/             dirty-repository fixture generator
```

## Code Style

- `gofmt` formatting, `go vet` clean; standard Go naming; errors wrapped with `%w` and context.
- Sizes are `int64` bytes internally; human-readable formatting only at presentation time (1 MB = 1 000 000 bytes? — decide in the first size-rule commit and record here).
- Detectors are pure functions over the file index; no I/O besides hashing in the duplicate detector.
- Match the conventions of surrounding code: comment density, naming, idiom.
