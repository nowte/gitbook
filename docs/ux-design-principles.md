# gitBook — UX Design Principles & Intent Language System

**Version:** v1.03.00+  
**Last Updated:** 2026-05-16  
**Author:** System UX Architect

---

## Core Philosophy

> The user declares their **intent**. The system handles the rest.

gitBook hides all Git internals from the user. The user never needs to know:
- what "staging" or "the index" means
- what a "commit hash" is
- what "upstream", "remote", "HEAD", or "detached HEAD" means
- what "rebasing" does internally
- what "non-fast-forward" means

Instead, they think in terms of:
- **"Save my work"** → `/commit`
- **"Send it to GitHub"** → `/push`
- **"Get the latest version"** → `/pull`
- **"Start a new feature"** → `/start`
- **"Go back to before"** → `/undo`

---

## Intent-to-Command Mapping

| User intent | gitBook command | What it does underneath |
|---|---|---|
| Save my work | `/commit` | git add + git commit |
| Upload to GitHub | `/push` | git push |
| Download latest | `/pull` | git pull |
| Start a feature | `/start <name>` | git checkout -b feature/<name> |
| Finish a feature | `/finish` | git checkout main + git merge |
| See what changed | `/status` | git status + branch + ahead/behind |
| Preview before saving | `/review` | git diff --staged + git diff |
| Undo last save | `/reset 1` | git reset --soft HEAD~1 |
| Full undo (one command) | `/auto-push` | git add . + git commit + git push |
| Set aside unfinished | `/stash` | git stash |
| Go back to a checkpoint | `/undo` | git reset --hard <snapshot hash> |

---

## Language Rules

### Forbidden words (never shown to user)

| Git term | Replace with |
|---|---|
| commit | save / snapshot |
| stage / staging area | mark as ready / mark for saving |
| upstream | GitHub |
| remote | GitHub / online |
| branch | workspace |
| HEAD | last save / current position |
| detached HEAD | not in any workspace |
| rebase | move workspace onto |
| merge | combine / bring together |
| push | send / upload |
| pull | download / get latest |
| fetch | check for updates |
| stash | set aside |
| hash / SHA | save ID |
| checkout | switch to |
| index | — (never shown) |
| origin | GitHub (or "your connected service") |

### Error message pattern

NEVER:
```
fatal: upstream not set for branch 'main'
```

ALWAYS:
```
This project is not connected to GitHub yet.
Connect it with: /github <your-github-link>
```

---

## UX Flow Rules

### Rule 1: One screen = one purpose
Never ask multiple unrelated questions at once.  
Wizard collects fields one at a time with `Step N of M` progress.

### Rule 2: Setup is a one-time flow
If GitHub is not connected, the flow is interrupted once:
- User is informed clearly
- Setup wizard starts
- On completion, user is returned to their original action

### Rule 3: Skip is always natural
Optional fields say "(optional — press Enter to skip)".  
Empty input = skip. Never force unnecessary input.

### Rule 4: Errors are human
Every error has two parts:
1. What happened (in plain language)
2. What to do next (a specific `/command`)

```
Could not upload — GitHub has changes you don't have.
Download them first: /pull
```

### Rule 5: Confirmations protect the user
Destructive actions (delete workspace, hard reset) always:
- Explain exactly what will be deleted
- Say it cannot be undone
- Require typing 'confirm' explicitly

---

## One-Command Shortcuts (auto-pipelines)

For users who just want things to work:

| Goal | Command | Steps inside |
|---|---|---|
| Upload everything | `/auto-push` | stage + commit + push |
| Save locally | `/auto-save` | stage + commit |
| Get latest | `/auto-sync` | fetch + pull + status |
| Brand new project | `/auto-start` | init + setup + github + first push |
| Publish a release | `/auto-release` | stage + commit + tag + push + tag-push |
| Fresh project | `/auto-fresh` | init + gitignore + stage + commit |

Each step shows:
- A human-readable description ("Marking all changed files…")
- Elapsed time in ms
- On failure: `continue` or `abort` choice (no technical output)

---

## Translation System

All user-facing strings live in `internal/lang/`.  
Default language: **English** (`lang_en.go`).  
Turkish translation: `lang_tr.go`.

### Adding a new language

1. Copy `lang_example.go` to `lang_xx.go`
2. Implement `GetTranslations() map[string]string`
3. Register in `manager.go` `SetLanguage()` switch
4. User switches with `/language xx`

### Key naming convention

```
cmd_*         — command descriptions shown in help/list
msg_*         — runtime messages (success, info, warnings)
err_hint_*    — error recovery hints
help_*        — help section headers
auto_*        — pipeline step labels
wizard_*      — wizard UI text
smart_*       — AI/smart system messages
tutorial_*    — tutorial content
placeholder_* — input placeholder text
usage_*       — usage examples
info_*        — info/config display labels
status_*      — status display labels
git_*         — git operation result messages
```

---

## Checkpoint System (Snapshots)

Before every destructive operation, gitBook automatically saves a checkpoint:
- `reset --hard` 
- `rebase`
- `merge`

Checkpoints are stored in `.gitbook/snapshots/` as lightweight JSON.  
Max 10 checkpoints — oldest deleted automatically.

User-facing:
- Before operation: `[i] Checkpoint saved — run /undo to go back if needed.`
- `/undo` → restores most recent checkpoint
- `/snapshots` → lists all checkpoints with timestamps

---

## Visual Design Tokens

| Element | Color | Lipgloss constant |
|---|---|---|
| Success | Green 82 | `colorGreen` |
| Error | Red 196 | `colorRed` |
| Warning | Yellow 220 | `colorYellow` |
| Info | Blue 33 | `colorBlue / accentBlue` |
| Dimmed / hint | Gray 241 | `subtleGrey` |
| Selected item | Orange 208 | `tipOrange` |
| Background | Dark 234 | `bgDark` |

---

## Roadmap Integration

See `docs/roadmap.md` for upcoming UX improvements:
- v1.04.00: Live preview of changes in `/review`  
- v1.04.00: Interactive file selection for `/stage`  
- v1.04.00: Paginated `/log` with commit details

