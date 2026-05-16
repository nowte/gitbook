# gitBook

> A terminal-based Git assistant that turns complex commands into simple `/slashes`.

```
/stage        instead of    git add .
/commit fix   instead of    git commit -m "fix"
/push         instead of    git push origin main
```

Built with [Bubble Tea](https://github.com/charmbracelet/bubbletea). Written in Go.

---

## Features

- **Smart error hints** — when a git command fails, gitBook suggests the exact `/command` to fix it
- **Guided tutorial** — `/tutorial` walks you through Git basics across 7 paginated screens
- **Smart commit** — enforces Conventional Commits format when a `work` profile is active
- **Smart push** — shows a diff summary and asks for confirmation before pushing to protected branches
- **Auto `.gitignore`** — `/gitignore` scans your project and generates the right rules
- **Profiles** — switch between `work`, `personal`, `oss`, and `explore` modes with `/profile`
- **Protected branches** — `main`, `master`, and `release` require confirmation for destructive ops
- **Concurrent-safe** — write operations are serialised; config files use atomic rename
- **Snapshot & undo** — automatic snapshot before every destructive op (`/reset-hard`, `/rebase`, `/finish`); restore with `/undo`
- **Instance guard** — warns when another gitBook process is already running on the same repo
- **Panic recovery** — internal errors are caught gracefully; the TUI stays alive
- **Bilingual** — full Turkish and English support (`/language tr` / `/language en`)
- **Audit log** — every git command is logged to `$TMPDIR/gitbook/logs/`

---

## Quick Start

```
# 1. Build
go build -o gitbook ./cmd/gitbook

# 2. Run inside any directory
./gitbook

# 3. Initialise a new repo
/init

# 4. Set your identity
/setup

# 5. Start working
/stage
/commit feat: initial commit
/push
```

---

## Command Reference

| gitBook command | Equivalent git command |
|---|---|
| `/init` | `git init` |
| `/status` | `git status` |
| `/stage` | `git add .` |
| `/stage <file>` | `git add <file>` |
| `/commit <msg>` | `git commit -m "<msg>"` |
| `/push` | `git push` |
| `/pull` | `git pull` |
| `/branch` | `git branch` |
| `/start <name>` | `git checkout -b feature/<name>` |
| `/finish` | `git merge feature/<name>` (from base branch) |
| `/cleanup` | `git branch -d feature/<name>` |
| `/log` | `git log --oneline` |
| `/diff` | `git diff` |
| `/diff <branch>` | `git diff <branch>` |
| `/stash` | `git stash` |
| `/stash-list` | `git stash list` |
| `/stash-pop` | `git stash pop` |
| `/reset <n>` | `git reset --soft HEAD~<n>` |
| `/reset-hard <n>` | `git reset --hard HEAD~<n>` |
| `/revert <hash>` | `git revert <hash>` |
| `/rebase <branch>` | `git rebase <branch>` |
| `/cherry-pick <hash>` | `git cherry-pick <hash>` |
| `/clone <url>` | `git clone <url>` |
| `/remote` | `git remote -v` |
| `/tag <name>` | `git tag <name>` |
| `/blame <file>` | `git blame <file>` |
| `/github <url>` | `git remote add origin <url> && git push -u origin main` |
| `/sync` | `git fetch` + ahead/behind count |

### Smart commands (v1.02.00+)

| Command | Description |
|---|---|
| `/analyze` | Diff analysis — what changed, which language, warnings |
| `/suggest` | AI-style commit message suggestion (Conventional Commits) |
| `/gitignore` | Scan project and generate `.gitignore` |
| `/profile` | Manage project profiles (`work` / `personal` / `explore` / `oss`) |
| `/tutorial` | 7-page interactive Git tutorial |
| `/next` | Next tutorial page |
| `/prev` | Previous tutorial page |

### Safety commands (v1.03.00+)

| Command | Description |
|---|---|
| `/undo` | Restore repo to the most recent snapshot |
| `/snapshots` | List all saved snapshots |

---

## Auto Pipeline Commands

`/auto-*` commands run multiple git steps in sequence as a single pipeline. Each step shows a spinner with elapsed time. If a step fails, gitBook asks **continue or abort?** before proceeding.

### Why use `/auto-*` instead of individual commands?

| | Individual commands | `/auto-*` commands |
|---|---|---|
| Steps per command | 1 | 2–6 |
| Progress indicator | None | Spinner + ms per step |
| On failure | Stops, shows error | Asks: continue or abort? |
| Use case | Precise, step-by-step control | Fast, single-shot workflows |

**Example — pushing changes the manual way vs. auto:**

```
# Manual (3 separate commands, no pipeline)
/stage
/commit feat: add login page
/push

# Auto (one command, pipeline with spinner)
/auto-push "feat: add login page"
```

### Auto command reference

| Command | Pipeline steps | Manual equivalent |
|---|---|---|
| `/auto-push [msg] [remote]` | stage → commit → push | `/stage` + `/commit` + `/push` |
| `/auto-save [msg]` | stage → commit | `/stage` + `/commit` |
| `/auto-sync` | fetch → pull → status | `/fetch` + `/pull` + `/status` |
| `/auto-start [name] [email] [url]` | init → identity → github → stage → first commit → push | `/init` + `/setup` + `/github` + `/stage` + `/commit` + `/push` |
| `/auto-release [msg] [tag] [remote]` | stage → commit → tag → push → push tags | `/stage` + `/commit` + `/tag` + `/push` + `/tag-push` |
| `/auto-fresh [msg]` | init → gitignore → stage → commit | `/init` + `/gitignore` + `/stage` + `/commit` |

---

## Project Structure

```
gitbook/
├── cmd/gitbook/         # Entry point
├── internal/
│   ├── config/          # .gitbook/config.json management (mutex-safe, instance lock)
│   ├── git/             # Git operations (concurrent-safe, retry logic, snapshots)
│   ├── lang/            # i18n — Turkish & English
│   ├── smart/           # Commit suggester, diff analyser, gitignore generator, profiles
│   └── ui/              # Bubble Tea TUI — home, chat, handlers, wizard, undo
├── docs/
│   ├── roadmap.md
│   └── system-audit-report.md
└── versions/            # Changelog per release
```

---

## Configuration

gitBook stores its config in `.gitbook/config.json` inside your repository:

```json
{
  "version": "0.1.0",
  "initiated_at": "2026-05-13T10:00:00Z",
  "rules": {
    "protected_branches": ["main", "master", "release"],
    "require_commit_msg": true
  }
}
```

Profiles are stored in `.gitbook/profiles.json`. Initialise defaults with `/profile init`.

---

## Requirements

- Go 1.22+
- Git 2.x installed and on `$PATH`
- A terminal with UTF-8 support

---

## Building from Source

```bash
git clone https://github.com/nowte/gitbook
cd gitbook
go mod download
go build -o gitbook ./cmd/gitbook
./gitbook
```

To cross-compile for Windows:

```bash
GOOS=windows GOARCH=amd64 go build -o gitbook.exe ./cmd/gitbook
```

---

## Contributing

1. Fork the repository
2. Create a feature branch: `git checkout -b feature/your-idea`
3. Follow the existing handler pattern in `internal/ui/handlers.go`
4. Add translations to both `lang_tr.go` and `lang_en.go`
5. Open a pull request

See [docs/roadmap.md](docs/roadmap.md) for planned features.

---

## License

MIT — see [LICENSE](LICENSE)

---

*Built by [nowte](https://github.com/nowte)*
