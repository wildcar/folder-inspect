# Agent Instructions

Primary entrypoint for any agent (Claude, Codex, DeepSeek, etc.) working in this repository.

Keep this file under ~200 lines — it is loaded into context every session. Details belong
in `AGENTS/` docs; repeatable procedures belong in skills under `.claude/skills/`.

## Project

folder-inspect — utility that inspects a local folder tree (including git working copies) for very large files, archives, junk files and duplicates, and helps clean them up.

Stack, interface and platform are **not chosen yet** — they are decided by the discovery questionnaire in `docs/discovery-questionnaire.md`. Do not start implementation before the questionnaire is answered (or its defaults explicitly accepted) and the choice is recorded as an ADR in `docs/adr/`.

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

Hard constraints and invariants this project must not violate. Fill in after the first iteration; keep each rule one line.

<!-- Examples of the kind of rule that belongs here:
- Do not delete files — move retired code to `deprecated/` (or `git mv`) so git history survives.
- Library / framework lock-ins ("X is the only allowed library for Y").
- Runtime / boundary invariants the agent must respect.
-->

- —

## Stack & Commands

Stack one-liner plus the commands an agent needs on day one. Keep the full cheat-sheet in `AGENTS/ENV.md`; here keep only the essentials.

```bash
# install      — <how to install dependencies>
# dev / run    — <how to start the app locally>
# build        — <how to produce a production build>
# test         — <how to run the test suite>
# lint         — <how to lint / typecheck>
```

## Architecture

Map of the codebase so an agent knows where things live. Fill in after the structure stabilizes.

```
<top-level layout — key directories and their responsibility>
```

## Code Style

- <language mode / formatter / linter conventions>
- <naming, units, formatting rules that aren't obvious from the code>
- Match the conventions of surrounding code: comment density, naming, idiom.
