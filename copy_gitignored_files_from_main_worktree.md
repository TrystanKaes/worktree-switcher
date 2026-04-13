# Copy git-ignored files from main worktree on create

## Context

When users create a new worktree with `wt create`, they frequently need files that are intentionally gitignored — things like `.env`, `.envrc`, local config overrides, or IDE settings. Today those files only live in the main worktree, so every new worktree starts broken until the user manually copies them over.

This plan adds an opt-in per-repo config file that lists relative paths to copy from the main worktree (the one checked out on `main`/`master`) into any newly created worktree. Missing config, missing source worktree, and missing files all degrade gracefully to a warning — worktree creation itself is never blocked.

## Design decisions (from user)

- **Config source**: per-repo config file committed at the repo root
- **Source worktree**: the worktree whose branch matches `detectMainBranch()` (main/master)
- **Patterns**: literal relative paths only (no globs)
- **Error handling**: warn to stderr and continue; never fail the create

## Config file format

File: `.worktree-switcher` at the repo root (the main worktree root). One literal relative path per line. `#` starts a comment. Blank lines are ignored.

```
# Files copied from the main worktree to each new worktree
.env
.envrc
.vscode/settings.json
```

- Paths must be relative; anything containing `..` or starting with `/` is rejected with a warning.
- Directories are copied recursively; files are copied with their mode preserved.
- Parent directories are created as needed in the destination.

We intentionally pick a plain line-based format so we don't pull in a TOML/YAML dependency (go.mod currently only has bubbletea/lipgloss).

## Files to modify

### New file: `copy.go`
Houses all the copy-on-create logic so `git.go` stays focused on git plumbing.

- `const copyConfigFilename = ".worktree-switcher"`
- `func loadCopyConfig(mainWorktreePath string) ([]string, error)` — reads `<mainWorktreePath>/.worktree-switcher`, returns the cleaned list of relative paths. Returns `(nil, nil)` if the file doesn't exist (not an error).
- `func findMainWorktreePath(worktrees []Worktree, mainBranch string) string` — scans `worktrees` for a non-bare entry whose `Branch == mainBranch`; returns `""` if not found.
- `func copyFilesFromMain(destPath string)` — orchestrator called after a successful `AddWorktree`. Steps:
  1. `mainBranch := detectMainBranch()`; bail quietly if empty.
  2. `worktrees, _ := ListWorktrees()`; find main worktree path.
  3. If main path is missing or equals `destPath` (creating main itself), bail quietly.
  4. `paths, err := loadCopyConfig(mainPath)`; if err or empty, bail (warn on err).
  5. For each path: validate (relative, no `..`), `os.Stat` the source, copy file or recursive dir to `filepath.Join(destPath, p)`. Warn on each failure, keep going.
  6. Print a single `stderr` summary line: `copied N files from main worktree` (only when N > 0).
- `func copyPath(src, dst string) error` — handles file or directory. Use `os.MkdirAll` for parent dirs, preserve file mode via `info.Mode()`. For directories, walk with `filepath.WalkDir`. Symlinks: copy as symlinks via `os.Readlink` / `os.Symlink`.

All warnings go through a single helper `warnCopy(format string, args ...any)` that writes to `os.Stderr` prefixed with `wt: copy: ` so they're easy to spot and grep.

### `git.go` — `CreateWorktreeForBranch` (lines 299–337)
Add a single call to `copyFilesFromMain(path)` at the two success points:
- After line 325 (`return path, newBranch, nil`) — actually inserted just before the `return` on line 326.
- After line 334 (`if err := AddWorktree(...)`) — just before `return path, branch, nil` on line 336.

Both CLI (`runCreate` in `main.go:280`) and TUI (`ui.go:124-160`) funnel through this single function, so one insertion covers both flows.

We deliberately do **not** hook into the detached-HEAD branches in `runCreate` (`main.go:316-340`) — detached worktrees are throwaway and don't benefit from this feature. If that turns out to be wrong, it's a one-line addition per branch later.

### Existing helpers to reuse (do not reimplement)
- `detectMainBranch()` — `git.go:207`
- `ListWorktrees()` — `git.go:61`
- `Worktree.Branch` / `Worktree.Path` / `Worktree.IsBare` — `git.go:13`

## Behavioral guarantees

- If `.worktree-switcher` doesn't exist → silent no-op. Existing users see no change.
- If the main worktree itself can't be located → single warning, create still succeeds.
- If a listed file is missing in the source → warning per file, create still succeeds.
- Creating the main worktree (e.g. the first `wt create` on `main`) → no-op (source == dest).
- Permission errors during copy → warning per file, create still succeeds.
- Final stdout remains exactly one line: the new worktree path (required by the shell wrapper for `cd`). All copy output goes to **stderr**.

## Verification

End-to-end manual test in this repo:

1. Build: `make build` (or `go build -o wt`).
2. In a scratch repo with `main` checked out, create `.worktree-switcher`:
   ```
   .env
   .vscode/settings.json
   ```
   and populate those files with test content.
3. `./wt create test-branch` → verify the new worktree path printed on stdout contains both files with identical contents and modes.
4. `./wt create test-branch-2` when `.env` is missing from main → expect a `wt: copy:` warning on stderr, worktree still created, path still printed on stdout.
5. `./wt create main` (or whatever the main branch name is) → expect no copy activity (source == dest guard).
6. Delete `.worktree-switcher` and run `./wt create bare-test` → expect zero copy output (silent no-op, backward compatible).
7. TUI path: run `./wt`, pick `+ Create new worktree`, type a branch name → expect the same copy behavior as the CLI.
8. `go vet ./...` and `go build ./...` should remain clean. No new dependencies in `go.mod`.
