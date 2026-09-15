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

## Project facts

- GitHub remote: https://github.com/wildcar/folder-inspect (was empty at init, 2026-09-15).
- Commit identity is set per-repo (`git config user.name/email`), because the host's global git identity is a different (work) account.
- No `gh` CLI on the dev host; GitHub operations go through plain `git` or the web UI.
- Domain: a "project repository" here is a folder of documents for an implementation project; files must be directly openable by link, which is why archives count as findings.
