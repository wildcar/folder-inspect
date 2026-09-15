# Memory

Durable agent memory for this repository: working agreements and facts that are NOT
derivable from the code, git history, or SPEC/STATE/HISTORY.

This is the ONLY agent memory store in the project. Do not use external or per-tool memory
stores — memory must travel with the repo (see AGENTS.md -> Memory). Read at the start of
every session; when you learn something durable, append a short bullet here and commit it
together with the related change.

MEMORY.md = durable facts/agreements; current state -> STATE.md; iteration log -> HISTORY.md.

## Working agreements (feedback)

- Stack, platform and interface are chosen via the discovery questionnaire, not assumed — the owner asked for it explicitly (2026-09-15); why: the initial task statement was intentionally short and needs clarification first.
- Owner-facing discovery docs (questionnaire) are written in Russian even though technical docs are English — the owner has to answer them.
- Do not couple the product to git in any way (2026-09-15) — the owner's storages are document repositories of implementation projects, not code; "repository" in this project means a document folder.
- The owner answers questionnaires by the numbering used in the chat summary, not the document's A/B/C ids — map answers back to the document when recording them.
- The owner wants a "proper interface" for colleagues, not a developer-only CLI (2026-09-15); why: users are project people working with documents.
- No hard links or symlinks anywhere in the product (2026-09-15); why: the owner wants a removed duplicate to leave a visible, human-readable trace pointing to the original by relative path.
- Decisions that affect users' files (which duplicate stays) are made by a person in the UI, not by a heuristic; the tool only suggests. The rule-based `plan` command keeps this: duplicate groups are touched only with an explicit `-duplicates <policy>` flag (no default), and similar names / overlapping folders cannot be planned by rule at all (2026-09-15).
- The owner has no opinion on licensing — MIT was accepted on the agent's suggestion (2026-09-15); do not re-ask.
- Similar-name groups are per folder, not repository-wide (owner decision 2026-09-15); why: copies land next to their source, cross-folder groups were noise.
- On Windows, never pass `/select,<path>` to explorer.exe as a Go argument — Go quotes it whole and Explorer opens Documents; build the command line via `SysProcAttr.CmdLine` (2026-09-15).
- Outputs must never be overwritten silently (owner feedback 2026-09-15 after `report.json` was replaced by a second run); why: colleagues will run scans repeatedly and compare — a lost report is lost evidence. Rule now in AGENTS.md → Project Rules.

## Project facts

- GitHub remote: https://github.com/wildcar/folder-inspect (was empty at init, 2026-09-15).
- Commit identity is set per-repo (`git config user.name/email`), because the host's global git identity is a different (work) account.
- No `gh` CLI on the dev host; GitHub operations go through plain `git` or the web UI.
- Domain: a "project repository" here is a folder of documents for an implementation project; files must be directly openable by link, which is why archives count as findings.
- Releases are cut by pushing an annotated tag `vX.Y.Z` (2026-09-15); the version string is the tag without `v`, embedded via ldflags. The repo is public, so CI status can be read from the GitHub REST API without a token.
- Real repositories seen 2026-09-15: two roots, ~33 GB / 4 579 files and ~24 GB / 1 283 files, no read errors; oversized findings are mostly video (mp4, mkv, avi, insv, lrv) and photos. Sizes of this order are the performance target.
