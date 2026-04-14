# worktree-switcher

A fast, ergonomic CLI for managing git worktrees. Fuzzy-jump to any worktree with a short query, or open an interactive TUI to browse, create, and delete them.

```sh
wt feat        # jump to the best-matching worktree for "feat"
wti            # open the interactive picker
wt create      # create a worktree for the current branch
wts            # bounce back to the previous worktree
```

## Requirements

- macOS or Linux
- Git 2.5+

## Installation

### Homebrew

```sh
brew install TrystanKaes/Tools/worktree-switcher
```

### From source

Requires Go 1.25+.

```sh
git clone https://github.com/TrystanKaes/worktree-switcher.git
cd worktree-switcher
make install
```

## Shell setup

`wt` and `wti` are shell functions, not binaries — they need to run in your shell process to `cd` into the selected worktree. 

Add this to your shell config (`~/.zshrc` or `~/.bashrc`):

```sh
eval "$(worktree-switcher init)"
```

Then reload your shell:

```sh
source ~/.zshrc   # or ~/.bashrc
```

Supports **bash** and **zsh**. Tab completions are included automatically.

## Commands

| Command | Description |
|---------|-------------|
| `wt <query>` | Fuzzy-match a worktree and jump to it |
| `wti` | Open the interactive TUI picker |
| `wt switch` / `wts` | Return to the previous worktree |
| `wt create` | Create a worktree for the current branch |
| `wt create <branch>` | Create a worktree for a branch (creates branch if it doesn't exist) |
| `wt create --detached` | Create a detached HEAD worktree |
| `wt create <branch> -d` | Create a detached worktree at the tip of `<branch>` |
| `wt list` | List all worktrees (path, branch, last modified) |
| `wt prune` | Remove stale worktrees interactively |
| `wt prune -f` | Remove stale worktrees without confirmation |
| `wt sync --from <src>` | Copy configured local files from `<src>` into the current worktree |
| `wt sync --to <dst>` | Copy configured local files from the current worktree into `<dst>` |
| `wt sync --from <src> --to <dst>` | Copy between two specific worktrees |
| `wt help` | Show help |

## Fuzzy switching

`wt <query>` scores worktrees and jumps to the best match instantly:

1. **Exact** — branch name equals the query
2. **Substring** — query appears in the branch name or path
3. **Fuzzy** — query characters appear in order in the branch name or path

Among equal-scored matches, the most recently modified worktree wins. This means short queries reliably land on the right worktree as your working set grows.

```sh
wt main        # exact: jumps to the main branch worktree
wt feat        # substring: matches feature-login, feature-x, etc.
wt fl          # fuzzy: matches feature-login
```

## Interactive TUI

`wti` opens a full-screen picker. Worktrees are sorted by most recently modified, and your previous worktree is pinned to the top.

## Quick switch

`wts` (or `wt switch`) returns to the last worktree you were in. Run it again to toggle back. Useful for rapid back-and-forth between two worktrees.

## Creating worktrees

Worktrees are created under `~/.worktree-switcher/<repo>/` and automatically named after the branch.

```sh
wt create                # current branch: ~/.worktree-switcher/myapp/main
wt create feature-x      # new or existing branch: ~/.worktree-switcher/myapp/feature-x
wt create --detached     # detached HEAD: ~/.worktree-switcher/myapp/detached
wt create feature-x -d   # detached at tip of feature-x
```

Branch names with `/` are sanitized in the path (e.g. `feature/login` → `feature-login`). If a branch is already checked out in another worktree, a suffixed branch is created instead (`feature-x-2`, `feature-x-3`, …).

## Copying files into new worktrees

Some files belong in every worktree but are gitignored — `.env`, `.envrc`, IDE settings, etc. Create a `.worktree-switcher` file in your repository root to copy them automatically on `wt create`:

```
# .worktree-switcher
.env
.envrc
.vscode/settings.json
```

Rules:
- Paths must be relative; `..` is not allowed
- Blank lines and `#` comments are ignored
- Files are copied with permissions preserved; directories are copied recursively
- A missing source file prints a warning but does not fail the create

This does not apply to detached worktrees (`wt create --detached`).

## Syncing files between worktrees

To copy configured files between existing worktrees, use `wt sync`. The `<src>` and `<dst>` arguments accept a branch name, full path, or any unique fragment.

```sh
wt sync --from feature-x           # copy from feature-x into current worktree
wt sync --to feature-x             # copy from current worktree into feature-x
wt sync --from main --to feature-x # copy between two specific worktrees
```

## Pruning stale worktrees

`wt prune` identifies and removes worktrees that are stale. A worktree is considered stale if:

- Its directory no longer exists on disk
- Its branch has been deleted
- Its branch has been merged into `main` or `master`

```sh
wt prune      # review and confirm each removal
wt prune -f   # remove all stale worktrees without prompting
```

## Uninstalling

### Homebrew

```sh
brew uninstall worktree-switcher
brew untap TrystanKaes/Tools   # optional
```

Then remove the shell integration line from your `~/.zshrc` or `~/.bashrc`:

```sh
eval "$(worktree-switcher init)"
```

### From source

```sh
make uninstall
```

Then remove the shell integration line from your profile.

## License

MIT — see [LICENSE](LICENSE).
