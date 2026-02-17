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

This builds the `ws-bin` binary, copies it to `$GOPATH/bin` (or `$HOME/go/bin`), and adds shell integration to your profile (`~/.bashrc`, `~/.bash_profile`, or `~/.zshrc`). Restart your shell or `source` your profile to activate.

<details>
<summary>Manual setup</summary>

If you prefer to configure your shell yourself, skip `make install` and add this to your shell config:

```sh
eval "$(ws-bin init)"
```

You can also specify the shell explicitly: `ws-bin init bash` or `ws-bin init zsh`.

</details>

## Usage

```
ws              Interactive TUI to select a worktree
ws <fragment>   Switch to worktree matching fragment (path or branch)
ws back         Return to the previous worktree
ws list         List all worktrees (plain text, scriptable)
ws prune        Remove stale worktrees (interactive confirmation)
ws prune -f     Remove stale worktrees (no confirmation)
ws init [shell] Output shell integration code (auto-detects from $SHELL)
ws help         Show this help
```

### Interactive TUI

Run `ws` with no arguments to launch the picker. Worktrees are sorted by most recently modified.

| Key          | Action                 |
| ------------ | ---------------------- |
| Up / Down    | Move cursor            |
| Enter        | Select worktree        |
| Type         | Fuzzy-filter the list  |
| Backspace    | Clear filter           |
| Esc / Ctrl+C | Quit without selecting |

### Direct switching

Pass a fragment to jump directly to a matching worktree by path or branch name:

```sh
ws feat        # cd into the worktree whose path or branch contains "feat"
```

If the fragment matches zero or more than one worktree, an error is returned.

### Quick back

`ws back` returns to the last worktree you switched away from. Running it again toggles back, so you can bounce between two worktrees. The previous worktree is also pinned to the top of the interactive TUI with a `(prev)` label.

### Pruning stale worktrees

`ws prune` finds worktrees that are stale and offers to remove them. A worktree is considered stale when:

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
