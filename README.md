# worktree-switcher

A CLI for managing and switching between git worktrees. Fuzzy-match a worktree by name to jump straight to it, or launch the interactive TUI picker with `wti`.

## Requirements

- Go 1.25.4+
- Git

## Installation

### Homebrew (recommended)

```sh
brew install TrystanKaes/Tools/worktree-switcher
```

Then add shell integration to your shell config (`~/.zshrc`, `~/.bashrc`, or `~/.bash_profile`):

```sh
eval "$(worktree-switcher init)"
```

Restart your shell or `source` your config file.

### From source

```sh
git clone https://github.com/TrystanKaes/worktree-switcher.git
cd worktree-switcher
make install
```

`make install` does three things:

1. **Builds the binary** — runs `go build` to produce the `worktree-switcher` binary.
2. **Installs the binary** — copies it to `~/.local/bin` (created if it doesn't exist). If that directory isn't on your `$PATH`, it is added to your shell profile automatically.
3. **Adds shell integration** — appends `eval "$(worktree-switcher init)"` to your shell profile, auto-detected from `$SHELL` (supports bash and zsh). Safe to re-run — won't duplicate entries.

Restart your shell or `source` your profile to activate.

To install to a different location:

```sh
make install INSTALL_DIR=/usr/local/bin
```

## Uninstall

```sh
make uninstall
```

Removes the binary from `~/.local/bin`. Then manually remove the following lines from your shell profile:

```sh
export PATH="$HOME/.local/bin:$PATH"   # if the installer added this
eval "$(worktree-switcher init)"
```

<details>
<summary>Manual setup</summary>

If you prefer to set things up yourself:

1. Build the binary and place it somewhere on your `$PATH`:

```sh
make build
cp worktree-switcher ~/.local/bin/    # or anywhere on $PATH
```

2. Add this to your shell config (`~/.zshrc`, `~/.bashrc`, etc.):

```sh
eval "$(worktree-switcher init)"
```

You can also specify the shell explicitly: `worktree-switcher init bash` or `worktree-switcher init zsh`.

</details>

## Commands

```
wt                        Show help
wt <query>                Fuzzy-match a worktree and switch to it (best by recency)
wti                       Interactive TUI to select a worktree
wt switch                 Return to the previous worktree
wt create                 Create a worktree for the current branch (auto-numbered)
wt create <branch>        Create a worktree for the given branch (or new branch)
wt create --detached      Create a detached HEAD worktree (auto-numbered)
wt create <branch> -d     Create a detached worktree at the tip of <branch>
wt list                   List all worktrees (plain text, scriptable)
wt prune                  Remove stale worktrees (interactive confirmation)
wt prune -f               Remove stale worktrees (no confirmation)
wt sync --from <src>      Copy configured local files from <src> into current worktree
wt sync --to <dst>        Copy configured local files from current worktree into <dst>
wt sync --from <src> --to <dst>  Copy configured local files from <src> into <dst>
wt init [shell]           Output shell integration code (auto-detects from $SHELL)
wt help                   Show this help
worktree-switcher --version  Print version
```

## Interactive TUI

Run `wti` to launch the picker. Worktrees are sorted by most recently modified. The previously selected worktree is pinned to the top with a `(prev)` label.

### Keybindings

| Key          | Action                                 |
| ------------ | -------------------------------------- |
| Up / Down    | Move cursor                            |
| Enter        | Select worktree (or confirm action)    |
| Type         | Fuzzy-filter the list                  |
| Backspace    | Clear last filter character            |
| d            | Toggle delete mode (when not filtering)|
| Esc / Ctrl+C | Quit (or exit current mode)            |

### Creating worktrees from the TUI

A `+ Create new worktree` option appears at the bottom of the list. Select it and press Enter to open a branch name input. Type a branch name and press Enter to create the worktree and `cd` into it. Press Esc to cancel.

If the branch already exists and is checked out in another worktree, a new branch is automatically created (e.g. `main-2`, `main-3`).

### Deleting worktrees from the TUI

Press `d` (when not filtering) to enter delete mode:

1. The header changes to `WORKTREE SWITCHER (delete mode)` in red
2. Navigate to the worktree you want to remove; the cursor line is highlighted in red
3. Press Enter to open a confirmation dialog (Cancel / Delete)
4. Use Left/Right arrows (or `h`/`l`) to choose, Enter to confirm
5. If the worktree has uncommitted changes, a second prompt asks whether to force delete
6. The TUI stays in delete mode after deletion so you can continue removing worktrees
7. Press `d` again or Esc to exit delete mode

## Fuzzy switching

Pass a query to jump directly to the best-matching worktree:

```sh
wt feat        # cd into the best worktree matching "feat"
wt main        # exact branch match takes priority
wt fx          # fuzzy match — characters appear in order in path or branch
```

Matching uses a tiered scoring system:

1. **Exact** (score 3) — branch name equals the query
2. **Substring** (score 2) — query is a substring of the path, short path, or branch
3. **Fuzzy** (score 1) — all query characters appear in order in the path, short path, or branch

Among equal scores, the most recently modified worktree wins. This means you can type a short query and reliably land on the worktree you used most recently.

## Quick switch

`wt switch` returns to the last worktree you switched away from. Running it again toggles back, so you can bounce between two worktrees rapidly.

## Creating worktrees

`wt create` creates new worktrees stored under `~/.worktree-switcher/<repo>/`:

```sh
wt create              # worktree for current branch (auto-numbered: branch/1, branch/2, ...)
wt create feature-x    # worktree for feature-x at ~/.worktree-switcher/repo/feature-x
wt create --detached   # detached HEAD worktree (auto-numbered: detached/1, detached/2, ...)
wt create feature-x -d # detached worktree at the tip of feature-x
```

After creation, the shell wrapper automatically `cd`s into the new worktree.

If the requested branch is already checked out in another worktree, a new branch is created from it (e.g. `feature-x-2`) and a message is printed to stderr.

Branch names containing `/` are sanitized to `-` in the filesystem path (e.g. `feature/login` becomes `~/.worktree-switcher/myrepo/feature-login`).

### Copying local gitignored files into new worktrees

You can opt in to copying local-only files (such as `.env`, `.envrc`, or IDE settings) from your main worktree into each newly created worktree.

Create a `.worktree-switcher` file in your repository root (the main worktree path), with one relative path per line:

```txt
# Files copied from the main worktree to each new worktree
.env
.envrc
.vscode/settings.json
```

Rules:

- Paths must be relative (no absolute paths, no `..`)
- Blank lines and lines starting with `#` are ignored
- Files are copied with mode preserved
- Directories are copied recursively
- Symlinks are copied as symlinks

Behavior:

- Missing `.worktree-switcher` file: silent no-op (fully backward compatible)
- Missing configured source files: warning to stderr, create still succeeds
- Copy errors (permissions, etc.): warning to stderr, create still succeeds
- Output contract is preserved: the created worktree path is still printed to stdout

Note: file copying applies to regular `wt create` flows (CLI and TUI create), not detached worktrees created with `--detached`.

To sync configured files between existing worktrees, use:

```sh
wt sync --from <src>
wt sync --to <dst>
wt sync --from <src> --to <dst>
```

`<src>` and `<dst>` can be a branch name, full worktree path, or unique fragment.

If one side is omitted, the current worktree is used:

- `wt sync --from feature-x` copies from `feature-x` into current
- `wt sync --to feature-x` copies from current into `feature-x`

## Pruning stale worktrees

`wt prune` finds worktrees that are stale and offers to remove them one by one. A worktree is considered stale when:

- Its directory no longer exists on disk
- Its branch has been deleted
- Its branch has been merged into main/master

Use `-f` or `--force` to skip confirmation prompts and remove all stale worktrees at once.

## Shell integration

The `wt` and `wti` commands are shell functions (not the binary directly). This is necessary so they can `cd` into the selected worktree in your current shell session. The binary is called `worktree-switcher`.

The shell wrapper handles:
- `wt <query>` — captures the path printed by `worktree-switcher` and runs `cd` into it
- `wti` — launches the interactive TUI via `worktree-switcher interactive` and `cd`s into the selection
- `wt switch` — pure shell, toggles between current and previous directory using `$__WT_LAST_DIR`
- Passing through commands that don't need `cd` (`list`, `prune`, `sync`, `help`, `init`)
- Tab completions for all subcommands and their flags

## Cross-platform builds

```sh
make build-all
```

Produces binaries in `dist/` for:

- `darwin/arm64`
- `darwin/amd64`
- `linux/amd64`
