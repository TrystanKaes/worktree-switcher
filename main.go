package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func main() {
	args := os.Args[1:]

	if len(args) > 0 {
		switch args[0] {
		case "list":
			if err := runList(); err != nil {
				fmt.Fprintln(os.Stderr, err)
				os.Exit(1)
			}
			return

		case "prune":
			force := false
			for _, a := range args[1:] {
				if a == "-f" || a == "--force" {
					force = true
				}
			}
			if err := RunPrune(force); err != nil {
				fmt.Fprintln(os.Stderr, err)
				os.Exit(1)
			}
			return

		case "create":
			if err := runCreate(args[1:]); err != nil {
				fmt.Fprintln(os.Stderr, err)
				os.Exit(1)
			}
			return

		case "init":
			if err := runInit(args[1:]); err != nil {
				fmt.Fprintln(os.Stderr, err)
				os.Exit(1)
			}
			return

		case "sync":
			if err := runSync(args[1:]); err != nil {
				fmt.Fprintln(os.Stderr, err)
				os.Exit(1)
			}
			return

		case "--help", "-h", "help":
			printUsage()
			return

		default:
			// Treat as a path/branch fragment for direct switching
			if err := runDirect(strings.Join(args, " ")); err != nil {
				fmt.Fprintln(os.Stderr, err)
				os.Exit(1)
			}
			return
		}
	}

	// No args: interactive TUI
	if err := runInteractive(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func runInteractive() error {
	worktrees, err := ListWorktrees()
	if err != nil {
		return err
	}
	if len(worktrees) == 0 {
		return fmt.Errorf("no worktrees found (are you in a git repository?)")
	}

	SortByModified(worktrees)

	// Pin previous worktree to top if __WT_LAST_DIR is set
	previousIdx := -1
	if lastDir := os.Getenv("__WT_LAST_DIR"); lastDir != "" {
		for i, ws := range worktrees {
			if ws.Path == lastDir {
				if i > 0 {
					// Move to front
					prev := worktrees[i]
					copy(worktrees[1:i+1], worktrees[0:i])
					worktrees[0] = prev
				}
				previousIdx = 0
				break
			}
		}
	}

	selected, err := RunTUI(worktrees, previousIdx)
	if err != nil {
		return err
	}
	if selected == "" {
		os.Exit(1)
	}
	fmt.Println(selected)
	return nil
}

func runList() error {
	worktrees, err := ListWorktrees()
	if err != nil {
		return err
	}

	SortByModified(worktrees)

	// Compute column widths
	maxPath := 0
	maxBranch := 0
	for _, ws := range worktrees {
		sp := ws.ShortPath()
		if len(sp) > maxPath {
			maxPath = len(sp)
		}
		if len(ws.Branch) > maxBranch {
			maxBranch = len(ws.Branch)
		}
	}

	for _, ws := range worktrees {
		path := ws.ShortPath()
		pathPad := path + strings.Repeat(" ", maxPath-len(path))
		branchPad := ws.Branch + strings.Repeat(" ", maxBranch-len(ws.Branch))
		fmt.Printf("%s  %s  %s\n", pathPad, branchPad, ws.RelativeTime())
	}
	return nil
}

func runDirect(fragment string) error {
	worktrees, err := ListWorktrees()
	if err != nil {
		return err
	}

	ws, ok := FindWorktreeByFragment(worktrees, fragment)
	if !ok {
		return fmt.Errorf("no unique worktree match for %q (found 0 or multiple matches)", fragment)
	}
	fmt.Println(ws.Path)
	return nil
}

func runInit(args []string) error {
	shell := ""
	if len(args) > 0 {
		shell = args[0]
	} else {
		shellEnv := os.Getenv("SHELL")
		if shellEnv == "" {
			return fmt.Errorf("no shell specified and $SHELL is not set\nUsage: wt-bin init [bash|zsh]")
		}
		shell = filepath.Base(shellEnv)
	}

	switch shell {
	case "bash":
		fmt.Print(bashInitCode)
	case "zsh":
		fmt.Print(zshInitCode)
	default:
		return fmt.Errorf("unsupported shell: %s (supported: bash, zsh)", shell)
	}
	return nil
}

const bashInitCode = `wt() {
  if [[ "$1" == "switch" ]]; then
    if [[ -n "$__WT_LAST_DIR" ]]; then
      local prev="$__WT_LAST_DIR"
      __WT_LAST_DIR="$PWD"
      cd "$prev" || return 1
    else
      echo "wt: no previous worktree" >&2
      return 1
    fi
    return 0
  fi

  if [[ "$1" == "list" || "$1" == "prune" || "$1" == "help" || "$1" == "--help" || "$1" == "-h" || "$1" == "init" || "$1" == "sync" ]]; then
    wt-bin "$@"
    return $?
  fi

  local dir
  dir="$(wt-bin "$@")"
  local rc=$?

  if [[ $rc -eq 0 && -n "$dir" ]]; then
    __WT_LAST_DIR="$PWD"
    cd "$dir" || return 1
  fi
  return $rc
}

_wt_completions() {
  local cur="${COMP_WORDS[COMP_CWORD]}"
  if [[ $COMP_CWORD -eq 1 ]]; then
    COMPREPLY=($(compgen -W "switch create list prune sync help" -- "$cur"))
  elif [[ "${COMP_WORDS[1]}" == "create" && $COMP_CWORD -eq 2 ]]; then
    COMPREPLY=($(compgen -W "--detached -d" -- "$cur"))
  elif [[ "${COMP_WORDS[1]}" == "prune" && $COMP_CWORD -eq 2 ]]; then
    COMPREPLY=($(compgen -W "-f --force" -- "$cur"))
  elif [[ "${COMP_WORDS[1]}" == "sync" && $COMP_CWORD -eq 2 ]]; then
    COMPREPLY=($(compgen -W "--from --to" -- "$cur"))
  fi
}

complete -F _wt_completions wt
`

const zshInitCode = `wt() {
  if [[ "$1" == "switch" ]]; then
    if [[ -n "$__WT_LAST_DIR" ]]; then
      local prev="$__WT_LAST_DIR"
      __WT_LAST_DIR="$PWD"
      cd "$prev" || return 1
    else
      echo "wt: no previous worktree" >&2
      return 1
    fi
    return 0
  fi

  if [[ "$1" == "list" || "$1" == "prune" || "$1" == "help" || "$1" == "--help" || "$1" == "-h" || "$1" == "init" || "$1" == "sync" ]]; then
    wt-bin "$@"
    return $?
  fi

  local dir
  dir="$(wt-bin "$@")"
  local rc=$?

  if [[ $rc -eq 0 && -n "$dir" ]]; then
    __WT_LAST_DIR="$PWD"
    cd "$dir" || return 1
  fi
  return $rc
}

_wt() {
  local -a subcommands
  subcommands=(
    'switch:Return to previous worktree'
    'create:Create a new worktree'
    'list:List all worktrees'
    'prune:Remove stale worktrees'
    'sync:Copy configured local files between worktrees'
    'help:Show help'
  )

  if (( CURRENT == 2 )); then
    _describe 'command' subcommands
  elif (( CURRENT == 3 )) && [[ "${words[2]}" == "create" ]]; then
    local -a create_flags
    create_flags=(
      '--detached:Create a detached HEAD worktree'
      '-d:Create a detached HEAD worktree'
    )
    _describe 'flag' create_flags
  elif (( CURRENT == 3 )) && [[ "${words[2]}" == "prune" ]]; then
    local -a prune_flags
    prune_flags=(
      '-f:Force removal without confirmation'
      '--force:Force removal without confirmation'
    )
    _describe 'flag' prune_flags
  elif (( CURRENT == 3 )) && [[ "${words[2]}" == "sync" ]]; then
    local -a sync_flags
    sync_flags=(
      '--from:Source worktree (branch, path, or fragment)'
      '--to:Destination worktree (branch, path, or fragment)'
    )
    _describe 'flag' sync_flags
  fi
}

compdef _wt wt
`

func runSync(args []string) error {
	worktrees, err := ListWorktrees()
	if err != nil {
		return err
	}

	fromSpec := ""
	toSpec := ""
	for i := 0; i < len(args); i++ {
		a := args[i]
		switch {
		case strings.HasPrefix(a, "--from="):
			fromSpec = strings.TrimSpace(strings.TrimPrefix(a, "--from="))
		case strings.HasPrefix(a, "--to="):
			toSpec = strings.TrimSpace(strings.TrimPrefix(a, "--to="))
		case a == "--from":
			if i+1 >= len(args) {
				return fmt.Errorf("missing value for --from")
			}
			fromSpec = strings.TrimSpace(args[i+1])
			i++
		case a == "--to":
			if i+1 >= len(args) {
				return fmt.Errorf("missing value for --to")
			}
			toSpec = strings.TrimSpace(args[i+1])
			i++
		default:
			return fmt.Errorf("unknown argument %q\nusage: wt sync [--from <branch|path|fragment>] [--to <branch|path|fragment>]", a)
		}
	}

	if fromSpec == "" && toSpec == "" {
		return fmt.Errorf("must specify at least one of --from or --to\nusage: wt sync [--from <branch|path|fragment>] [--to <branch|path|fragment>]")
	}

	current, err := currentWorktree(worktrees)
	if err != nil {
		return err
	}

	source := current
	dest := current

	if fromSpec != "" {
		source, err = resolveWorktree(worktrees, fromSpec)
		if err != nil {
			return err
		}
	}
	if toSpec != "" {
		dest, err = resolveWorktree(worktrees, toSpec)
		if err != nil {
			return err
		}
	}

	copyFilesFromSource(source.Path, dest.Path, fmt.Sprintf("%q", source.BranchOrPathLabel()))
	fmt.Fprintf(os.Stderr, "wt: copy: sync complete from %s to %s\n", source.ShortPath(), dest.ShortPath())
	return nil
}

func currentWorktree(worktrees []Worktree) (Worktree, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return Worktree{}, err
	}
	cwd = filepath.Clean(cwd)
	bestLen := -1
	var best Worktree
	for _, wt := range worktrees {
		root := filepath.Clean(wt.Path)
		if cwd == root || strings.HasPrefix(cwd, root+string(os.PathSeparator)) {
			if len(root) > bestLen {
				best = wt
				bestLen = len(root)
			}
		}
	}
	if bestLen == -1 {
		return Worktree{}, fmt.Errorf("current directory is not inside a known worktree")
	}
	return best, nil
}

func resolveWorktree(worktrees []Worktree, spec string) (Worktree, error) {
	spec = strings.TrimSpace(spec)
	if spec == "" {
		return Worktree{}, fmt.Errorf("empty worktree spec")
	}

	for _, wt := range worktrees {
		if wt.Branch == spec {
			return wt, nil
		}
	}

	absSpec, _ := filepath.Abs(spec)
	absSpec = filepath.Clean(absSpec)
	for _, wt := range worktrees {
		if filepath.Clean(wt.Path) == absSpec {
			return wt, nil
		}
	}

	match, ok := FindWorktreeByFragment(worktrees, spec)
	if !ok {
		return Worktree{}, fmt.Errorf("no unique worktree match for %q (found 0 or multiple matches)", spec)
	}
	return match, nil
}

func (w Worktree) BranchOrPathLabel() string {
	if w.Branch != "" {
		return w.Branch
	}
	return w.Path
}

func runCreate(args []string) error {
	detached := false
	var branch string
	for _, a := range args {
		if a == "--detached" || a == "-d" {
			detached = true
		} else if branch == "" {
			branch = a
		}
	}

	repo, err := RepoName()
	if err != nil {
		return err
	}

	if branch == "" && !detached {
		// No branch arg: create worktree for current branch
		cur, err := CurrentBranch()
		if err != nil {
			return err
		}
		if cur == "HEAD" {
			return fmt.Errorf("cannot determine current branch (detached HEAD); use --detached or specify a branch")
		}
		path, actualBranch, err := CreateWorktreeForBranch(repo, cur, true)
		if err != nil {
			return err
		}
		if actualBranch != cur {
			fmt.Fprintf(os.Stderr, "branch %q already checked out, created %q instead\n", cur, actualBranch)
		}
		fmt.Println(path)
		return nil
	}

	if detached && branch == "" {
		// Detached HEAD from current commit
		path, err := WorktreePath(repo, "detached", true)
		if err != nil {
			return err
		}
		if err := AddWorktree(path, "", false, true); err != nil {
			return err
		}
		fmt.Println(path)
		return nil
	}

	if detached {
		// Detached at tip of specified branch
		path, err := WorktreePath(repo, branch, false)
		if err != nil {
			return err
		}
		if err := AddWorktree(path, branch, false, true); err != nil {
			return err
		}
		fmt.Println(path)
		return nil
	}

	// Branch specified, not detached
	path, actualBranch, err := CreateWorktreeForBranch(repo, branch, false)
	if err != nil {
		return err
	}
	if actualBranch != branch {
		fmt.Fprintf(os.Stderr, "branch %q already checked out, created %q instead\n", branch, actualBranch)
	}
	fmt.Println(path)
	return nil
}

func printUsage() {
	fmt.Fprintln(os.Stderr, `worktree-switcher — Interactive git worktree switcher (BuiltByBurns)

Usage:
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
  wt sync --from src        Copy from src into current worktree
  wt sync --to dst          Copy from current worktree into dst
  wt sync --from src --to dst  Copy from src worktree into dst worktree
  wt init [shell]           Output shell integration code (auto-detects from $SHELL)
  wt help                   Show this help

Copy-on-create config:
  Create .worktree-switcher in your main worktree root to copy local files
  (e.g. .env, .envrc, .vscode/settings.json) into newly created worktrees.
  One relative path per line; '#' comments and blank lines are ignored.

Shell setup:
  Add to your shell config:  eval "$(wt-bin init)"

Navigation (TUI):
  ↑/↓         Move cursor
  Enter       Select worktree
  Type         Filter worktrees
  Backspace   Clear filter
  d            Toggle delete mode (when not filtering)
  Esc/Ctrl+C  Quit without selecting`)
}
