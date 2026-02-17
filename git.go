package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// Worktree represents a single git worktree.
type Worktree struct {
	Path       string
	Branch     string
	Commit     string
	IsBare     bool
	ModifiedAt time.Time
}

// ShortPath returns the path with $HOME replaced by ~.
func (w Worktree) ShortPath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return w.Path
	}
	if strings.HasPrefix(w.Path, home) {
		return "~" + w.Path[len(home):]
	}
	return w.Path
}

// RelativeTime returns a human-friendly relative time string.
func (w Worktree) RelativeTime() string {
	d := time.Since(w.ModifiedAt)
	switch {
	case d < time.Minute:
		return "just now"
	case d < time.Hour:
		m := int(d.Minutes())
		if m == 1 {
			return "1 minute ago"
		}
		return fmt.Sprintf("%d minutes ago", m)
	case d < 24*time.Hour:
		h := int(d.Hours())
		if h == 1 {
			return "1 hour ago"
		}
		return fmt.Sprintf("%d hours ago", h)
	default:
		days := int(d.Hours() / 24)
		if days == 1 {
			return "1 day ago"
		}
		return fmt.Sprintf("%d days ago", days)
	}
}

// ListWorktrees runs `git worktree list --porcelain` and parses the output.
func ListWorktrees() ([]Worktree, error) {
	cmd := exec.Command("git", "worktree", "list", "--porcelain")
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to list worktrees: %w", err)
	}
	return parseWorktrees(string(out))
}

func parseWorktrees(output string) ([]Worktree, error) {
	var worktrees []Worktree
	var current *Worktree

	for _, line := range strings.Split(output, "\n") {
		line = strings.TrimRight(line, "\r")

		switch {
		case strings.HasPrefix(line, "worktree "):
			if current != nil {
				worktrees = append(worktrees, *current)
			}
			current = &Worktree{
				Path: strings.TrimPrefix(line, "worktree "),
			}
		case strings.HasPrefix(line, "HEAD "):
			if current != nil {
				current.Commit = strings.TrimPrefix(line, "HEAD ")
			}
		case strings.HasPrefix(line, "branch "):
			if current != nil {
				branch := strings.TrimPrefix(line, "branch ")
				// Strip refs/heads/ prefix
				branch = strings.TrimPrefix(branch, "refs/heads/")
				current.Branch = branch
			}
		case line == "bare":
			if current != nil {
				current.IsBare = true
			}
		case line == "detached":
			if current != nil && current.Branch == "" {
				current.Branch = "(detached)"
			}
		}
	}
	if current != nil {
		worktrees = append(worktrees, *current)
	}

	// Stat each directory for modification time
	for i := range worktrees {
		info, err := os.Stat(worktrees[i].Path)
		if err == nil {
			worktrees[i].ModifiedAt = info.ModTime()
		}
	}

	return worktrees, nil
}

// SortByModified sorts worktrees by modification time (most recent first).
func SortByModified(wss []Worktree) {
	for i := 1; i < len(wss); i++ {
		for j := i; j > 0 && wss[j].ModifiedAt.After(wss[j-1].ModifiedAt); j-- {
			wss[j], wss[j-1] = wss[j-1], wss[j]
		}
	}
}

// StaleInfo describes why a worktree is considered stale.
type StaleInfo struct {
	Worktree   Worktree
	Reason     string
	DirMissing bool
}

// FindStaleWorktrees checks for worktrees whose directories are missing,
// whose branches have been deleted, or whose branches have been merged into main/master.
func FindStaleWorktrees(worktrees []Worktree) []StaleInfo {
	mainBranch := detectMainBranch()
	var stale []StaleInfo

	for _, ws := range worktrees {
		if ws.IsBare {
			continue
		}

		// Check if directory still exists
		if _, err := os.Stat(ws.Path); os.IsNotExist(err) {
			stale = append(stale, StaleInfo{
				Worktree:   ws,
				Reason:     "directory no longer exists",
				DirMissing: true,
			})
			continue
		}

		// Skip if this is the main branch
		if ws.Branch == mainBranch || ws.Branch == "(detached)" {
			continue
		}

		// Check if branch still exists
		if !branchExists(ws.Branch) {
			stale = append(stale, StaleInfo{
				Worktree: ws,
				Reason:   fmt.Sprintf("branch %q no longer exists", ws.Branch),
			})
			continue
		}

		// Check if branch has been merged into main
		if mainBranch != "" && isBranchMerged(ws.Branch, mainBranch) {
			stale = append(stale, StaleInfo{
				Worktree: ws,
				Reason:   fmt.Sprintf("branch %q has been merged into %s", ws.Branch, mainBranch),
			})
		}
	}

	return stale
}

// RemoveWorktree runs `git worktree remove <path>`.
func RemoveWorktree(path string, force bool) error {
	args := []string{"worktree", "remove"}
	if force {
		args = append(args, "--force")
	}
	args = append(args, path)
	cmd := exec.Command("git", args...)
	cmd.Stdout = os.Stderr
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func detectMainBranch() string {
	for _, name := range []string{"main", "master"} {
		if branchExists(name) {
			return name
		}
	}
	return ""
}

func branchExists(branch string) bool {
	cmd := exec.Command("git", "rev-parse", "--verify", fmt.Sprintf("refs/heads/%s", branch))
	return cmd.Run() == nil
}

func isBranchMerged(branch, into string) bool {
	cmd := exec.Command("git", "merge-base", "--is-ancestor", branch, into)
	return cmd.Run() == nil
}

// FindWorktreeByFragment returns a worktree whose path contains the given fragment.
// Returns the worktree and true if exactly one match is found.
func FindWorktreeByFragment(worktrees []Worktree, fragment string) (Worktree, bool) {
	fragment = strings.ToLower(fragment)
	var matches []Worktree
	for _, ws := range worktrees {
		lower := strings.ToLower(ws.Path)
		lowerShort := strings.ToLower(ws.ShortPath())
		lowerBranch := strings.ToLower(ws.Branch)
		if strings.Contains(lower, fragment) ||
			strings.Contains(lowerShort, fragment) ||
			strings.Contains(filepath.Base(lower), fragment) ||
			strings.Contains(lowerBranch, fragment) {
			matches = append(matches, ws)
		}
	}
	if len(matches) == 1 {
		return matches[0], true
	}
	return Worktree{}, false
}
