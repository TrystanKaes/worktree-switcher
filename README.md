# worktree-switcher

An interactive TUI for managing and switching between git worktrees. Type to fuzzy-filter, press enter to `cd` into your selection.

## Requirements

- Go 1.25.4+
- Git

## Installation

```sh
git clone https://github.com/trystankaes/worktree-switcher.git
cd worktree-switcher
make install
```

This builds the `wt-bin` binary, copies it to `$GOPATH/bin` (or `$HOME/go/bin`), and adds shell integration to your profile (`~/.bashrc`, `~/.bash_profile`, or `~/.zshrc`). Restart your shell or `source` your profile to activate.

<details>
<summary>Manual setup</summary>

If you prefer to configure your shell yourself, skip `make install` and add this to your shell config:

```sh
eval "$(wt-bin init)"
```

You can also specify the shell explicitly: `wt-bin init bash` or `wt-bin init zsh`.

</details>

## Commands

```
wt                        Interactive TUI to select a worktree
wt <fragment>             Switch to worktree matching fragment (path or branch)
wt switch                 Return to the previous worktree
wt create                 Create a worktree for the current branch
wt create <branch>        Create a worktree for the given branch (or new branch)
wt create --detached      Create a detached HEAD worktree
wt create <branch> -d     Create a detached worktree at the tip of <branch>
wt list                   List all worktrees (plain text, scriptable)
wt prune                  Remove stale worktrees (interactive confirmation)
wt prune -f               Remove stale worktrees (no confirmation)
wt sync --from <src>         Copy configured local files from <src> into current worktree
wt sync --to <dst>           Copy configured local files from current worktree into <dst>
wt sync --from <src> --to <dst>  Copy configured local files from <src> into <dst>
wt init [shell]           Output shell integration code (auto-detects from $SHELL)
wt help                   Show this help
```

## Interactive TUI

Run `wt` with no arguments to launch the picker. Worktrees are sorted by most recently modified. The previously selected worktree is pinned to the top with a `(prev)` label.

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

1. Each worktree shows a `[ ]` checkbox; the cursor line shows `[X]` in red
2. Press Enter on a worktree to see a confirmation dialog (Cancel / Delete)
3. Use Left/Right arrows (or `h`/`l`) to choose, Enter to confirm
4. If the worktree has uncommitted changes, a second prompt asks whether to force delete
5. The TUI stays in delete mode after deletion so you can continue removing worktrees
6. Press `d` again or Esc to exit delete mode

## Direct switching

Pass a fragment to jump directly to a matching worktree by path or branch name:

```sh
wt feat        # cd into the worktree whose path or branch contains "feat"
```

The fragment is matched against the full path, short path (`~/...`), base directory name, and branch name. If it matches zero or more than one worktree, an error is returned.

## Quick switch

`wt switch` returns to the last worktree you switched away from. Running it again toggles back, so you can bounce between two worktrees rapidly.

## Creating worktrees

`wt create` creates new worktrees stored at `~/.worktree-switcher/<repo>/<branch>`:

```sh
wt create              # worktree for current branch (auto-numbered: branch/1, branch/2, ...)
wt create feature-x    # worktree for feature-x (creates branch if it doesn't exist)
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

Note: this applies to regular `wt create` flows (CLI and TUI create), not detached `wt create --detached` flows.

To sync configured files between existing worktrees, use:

```sh
wt sync --from <src>
wt sync --to <dst>
wt sync --from <src> --to <dst>
```

`<src>` and `<dst>` can be a branch name, full worktree path, or unique fragment.

If one side is omitted, the current worktree is used for the missing side:

- `wt sync --from feature-x` copies from `feature-x` into current
- `wt sync --to feature-x` copies from current into `feature-x`

`--from`/`--to` values require a unique worktree match.

## Pruning stale worktrees

`wt prune` finds worktrees that are stale and offers to remove them one by one. A worktree is considered stale when:

- Its directory no longer exists on disk
- Its branch has been deleted
- Its branch has been merged into main/master

Use `-f` or `--force` to skip confirmation prompts and remove all stale worktrees at once.

## Shell integration

The `wt` command is a shell function (not the binary directly). This is necessary so it can `cd` into the selected worktree in your current shell session. The binary is called `wt-bin`.

The shell wrapper handles:
- Capturing the path printed by `wt-bin` and running `cd` into it
- The `wt switch` command (pure shell, toggles between current and previous directory)
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
