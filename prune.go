package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

// RunPrune detects stale worktrees and removes them.
// If force is true, removes without prompting.
func RunPrune(force bool) error {
	worktrees, err := ListWorktrees()
	if err != nil {
		return err
	}

	stale := FindStaleWorktrees(worktrees)
	if len(stale) == 0 {
		fmt.Fprintln(os.Stderr, "No stale worktrees found.")
		return nil
	}

	fmt.Fprintf(os.Stderr, "Found %d stale worktree(s):\n\n", len(stale))

	reader := bufio.NewReader(os.Stdin)
	removed := 0

	for _, s := range stale {
		fmt.Fprintf(os.Stderr, "  %s\n", s.Worktree.ShortPath())
		fmt.Fprintf(os.Stderr, "    branch: %s\n", s.Worktree.Branch)
		fmt.Fprintf(os.Stderr, "    reason: %s\n", s.Reason)

		shouldRemove := force
		if !force {
			fmt.Fprintf(os.Stderr, "  Remove? [y/N] ")
			answer, _ := reader.ReadString('\n')
			answer = strings.TrimSpace(strings.ToLower(answer))
			shouldRemove = answer == "y" || answer == "yes"
		}

		if shouldRemove {
			// Use --force for missing directories since git worktree remove
			// needs it when the directory doesn't exist
			err := RemoveWorktree(s.Worktree.Path, s.DirMissing)
			if err != nil {
				fmt.Fprintf(os.Stderr, "  ✗ failed to remove: %v\n\n", err)
			} else {
				fmt.Fprintf(os.Stderr, "  ✓ removed\n\n")
				removed++
			}
		} else {
			fmt.Fprintf(os.Stderr, "  skipped\n\n")
		}
	}

	fmt.Fprintf(os.Stderr, "Removed %d of %d stale worktree(s).\n", removed, len(stale))
	return nil
}
