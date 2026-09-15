# Environment

Host facts, tools, credentials, and command cheat-sheet for this project.
Update whenever a new tool, credential, or host-specific quirk is learned.

## Host

If the project runs in more than one place (e.g. local dev + a server), split per environment.

- **Dev**: Windows 10 Pro (10.0.19045), PowerShell 7 primary + Git Bash; repo at `C:\Users\sergey_e\D\repo\folder-inspect`.
- **Prod**: none — a locally run utility; distribution target decided by questionnaire (default: single executable via GitHub Releases).

## Tools

- git 2.53 (Windows). No `gh` CLI installed.
- **Go 1.27.1** (windows/amd64), installed 2026-09-15 per-user into `%LOCALAPPDATA%\Programs\go`
  (`go env GOPATH` = `C:\Users\sergey_e\go`). The dev user has **no admin rights**, so the
  MSI / winget / choco routes are out; to upgrade, re-run the per-user install (PowerShell 7):

  ```powershell
  $ver = (Invoke-RestMethod 'https://go.dev/dl/?mode=json')[0].version
  $zip = "$env:TEMP\$ver.windows-amd64.zip"
  Invoke-WebRequest "https://go.dev/dl/$ver.windows-amd64.zip" -OutFile $zip
  New-Item -ItemType Directory -Force "$env:LOCALAPPDATA\Programs" | Out-Null
  if (Test-Path "$env:LOCALAPPDATA\Programs\go") { Remove-Item -Recurse -Force "$env:LOCALAPPDATA\Programs\go" }
  Expand-Archive $zip -DestinationPath "$env:LOCALAPPDATA\Programs" -Force
  $userPath = [Environment]::GetEnvironmentVariable('Path', 'User')
  [Environment]::SetEnvironmentVariable('Path', "$env:LOCALAPPDATA\Programs\go\bin;$env:USERPROFILE\go\bin;$userPath", 'User')
  & "$env:LOCALAPPDATA\Programs\go\bin\go.exe" version
  ```

  Restart the terminal (and the Claude desktop app) afterwards so the new PATH is picked up.
- No `golangci-lint`; lint = `go vet` + `gofmt -l`.
- Also present but not used by the project: Python 3.14, Node.js, PostgreSQL 18 client, Pandoc.

## Credentials & secrets

- Where they live and how to read them — **pointers only, never store values here**.
- Note which files are gitignored (e.g. local env files) and must not be committed.

## Environments

For projects with several targets (dev / test / prod), one row per environment.

| Env | Host | Identifier | Role / account | Where used |
|-----|------|------------|----------------|------------|
| dev  | <...> | <...> | <...> | <...> |
| test | <...> | <...> | <...> | <...> |
| prod | <...> | <...> | <...> | <...> |

## Commands cheat-sheet

Split by environment when shells differ (e.g. PowerShell on dev, bash on prod).

### Dev

```
go build ./...                      # build
go test ./...                       # tests
go vet ./... && gofmt -l .          # lint / format check (gofmt -l must print nothing)
GOOS=linux GOARCH=amd64 go build -o dist/folder-inspect ./cmd/folder-inspect     # cross-compile (bash)
$env:GOOS="linux"; go build -o dist/folder-inspect ./cmd/folder-inspect          # cross-compile (pwsh)
```

### Prod

```
<deploy, migrate, logs, restart, ...>
```

## Host-specific quirks

A running log of gotchas — the things that cost an hour the first time. Split by environment.

### Dev

- Global git identity on this host is a work account; the repo overrides it locally with `wildcar <wildcar@mail.ru>` (`git config user.name/email`, not `--global`).
- Long bash heredocs with Cyrillic content failed to parse in the agent's Bash tool; use a file-write tool instead.
- Creating symlinks needs Developer Mode or admin on Windows; `scan.TestWalkDoesNotFollowSymlinks` skips itself when it cannot create one.
- `go test` on the fixture package writes ~200 KB (scale 1024); the `fixture` command at scale 1 writes ~170 MB — point it at a temp folder.

### Prod

- —
