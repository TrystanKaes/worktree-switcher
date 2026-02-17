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

		case "init":
			if err := runInit(args[1:]); err != nil {
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

	// Pin previous worktree to top if __WS_LAST_DIR is set
	previousIdx := -1
	if lastDir := os.Getenv("__WS_LAST_DIR"); lastDir != "" {
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
			return fmt.Errorf("no shell specified and $SHELL is not set\nUsage: ws-bin init [bash|zsh]")
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

const bashInitCode = `ws() {
  if [[ "$1" == "back" ]]; then
    if [[ -n "$__WS_LAST_DIR" ]]; then
      local prev="$__WS_LAST_DIR"
      __WS_LAST_DIR="$PWD"
      cd "$prev" || return 1
    else
      echo "ws: no previous worktree" >&2
      return 1
    fi
    return 0
  fi

  if [[ "$1" == "list" || "$1" == "prune" || "$1" == "help" || "$1" == "--help" || "$1" == "-h" || "$1" == "init" ]]; then
    ws-bin "$@"
    return $?
  fi

  local dir
  dir="$(ws-bin "$@")"
  local rc=$?

  if [[ $rc -eq 0 && -n "$dir" ]]; then
    __WS_LAST_DIR="$PWD"
    cd "$dir" || return 1
  fi
  return $rc
}

_ws_completions() {
  local cur="${COMP_WORDS[COMP_CWORD]}"
  if [[ $COMP_CWORD -eq 1 ]]; then
    COMPREPLY=($(compgen -W "back list prune help" -- "$cur"))
  elif [[ "${COMP_WORDS[1]}" == "prune" && $COMP_CWORD -eq 2 ]]; then
    COMPREPLY=($(compgen -W "-f --force" -- "$cur"))
  fi
}

complete -F _ws_completions ws
`

const zshInitCode = `ws() {
  if [[ "$1" == "back" ]]; then
    if [[ -n "$__WS_LAST_DIR" ]]; then
      local prev="$__WS_LAST_DIR"
      __WS_LAST_DIR="$PWD"
      cd "$prev" || return 1
    else
      echo "ws: no previous worktree" >&2
      return 1
    fi
    return 0
  fi

  if [[ "$1" == "list" || "$1" == "prune" || "$1" == "help" || "$1" == "--help" || "$1" == "-h" || "$1" == "init" ]]; then
    ws-bin "$@"
    return $?
  fi

  local dir
  dir="$(ws-bin "$@")"
  local rc=$?

  if [[ $rc -eq 0 && -n "$dir" ]]; then
    __WS_LAST_DIR="$PWD"
    cd "$dir" || return 1
  fi
  return $rc
}

_ws() {
  local -a subcommands
  subcommands=(
    'back:Return to previous worktree'
    'list:List all worktrees'
    'prune:Remove stale worktrees'
    'help:Show help'
  )

  if (( CURRENT == 2 )); then
    _describe 'command' subcommands
  elif (( CURRENT == 3 )) && [[ "${words[2]}" == "prune" ]]; then
    local -a prune_flags
    prune_flags=(
      '-f:Force removal without confirmation'
      '--force:Force removal without confirmation'
    )
    _describe 'flag' prune_flags
  fi
}

compdef _ws ws
`

func printUsage() {
	fmt.Fprintln(os.Stderr, `worktree-switcher — Interactive git worktree switcher

Usage:
  ws              Interactive TUI to select a worktree
  ws <fragment>   Switch to worktree matching fragment (path or branch)
  ws back         Return to the previous worktree
  ws list         List all worktrees (plain text, scriptable)
  ws prune        Remove stale worktrees (interactive confirmation)
  ws prune -f     Remove stale worktrees (no confirmation)
  ws init [shell] Output shell integration code (auto-detects from $SHELL)
  ws help         Show this help

Shell setup:
  Add to your shell config:  eval "$(ws-bin init)"

Navigation (TUI):
  ↑/↓         Move cursor
  Enter       Select worktree
  Type         Filter worktrees
  Backspace   Clear filter
  Esc/Ctrl+C  Quit without selecting`)
}
