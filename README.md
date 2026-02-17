# worktree-switcher

An interactive TUI for switching between git worktrees. Type to fuzzy-filter, press enter to `cd` into your selection.

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

## Usage

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
wt init [shell]           Output shell integration code (auto-detects from $SHELL)
wt help                   Show this help
```

### Interactive TUI

Run `wt` with no arguments to launch the picker. Worktrees are sorted by most recently modified.

| Key          | Action                          |
| ------------ | ------------------------------- |
| Up / Down    | Move cursor                     |
| Enter        | Select worktree                 |
| Type         | Fuzzy-filter the list           |
| Backspace    | Clear filter                    |
| d            | Toggle delete mode (when not filtering) |
| Esc / Ctrl+C | Quit without selecting          |

### Creating worktrees

`wt create` creates new worktrees stored at `~/.worktree-switcher/<repo>/<branch>`:

```sh
wt create              # worktree for current branch (numbered: branch/1, branch/2, ...)
wt create feature-x    # worktree for feature-x (creates branch if it doesn't exist)
wt create --detached   # detached HEAD worktree (numbered: detached/1, detached/2, ...)
wt create feature-x -d # detached worktree at the tip of feature-x
```

After creation, the shell wrapper automatically `cd`s into the new worktree.

### Deleting worktrees from the TUI

Press `d` in the interactive TUI (when not filtering) to enter delete mode. The selected line turns red. Press Enter to see a confirmation dialog, then confirm or cancel. The TUI stays open after deletion so you can continue working.

### Direct switching

Pass a fragment to jump directly to a matching worktree by path or branch name:

```sh
wt feat        # cd into the worktree whose path or branch contains "feat"
```

If the fragment matches zero or more than one worktree, an error is returned.

### Quick switch

`wt switch` returns to the last worktree you switched away from. Running it again toggles back, so you can bounce between two worktrees. The previous worktree is also pinned to the top of the interactive TUI with a `(prev)` label.

### Pruning stale worktrees

`wt prune` finds worktrees that are stale and offers to remove them. A worktree is considered stale when:

- Its directory no longer exists
- Its branch has been deleted
- Its branch has been merged into main/master

Use `-f` or `--force` to skip confirmation prompts.

## Cross-platform builds

```sh
make build-all
```

Produces binaries in `dist/` for:

- `darwin/arm64`
- `darwin/amd64`
- `linux/amd64`
