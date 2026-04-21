package columns

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"
)

// Collect fills in rich data fields on the provided rows in place.
// mainBranch is the name of the main branch (e.g. "main" or "master");
// pass "" when unknown.
//
// Each row must have Path, Branch, IsDetached, and IsBare set before calling.
// Rows with IsBare == true are skipped. Errors on individual tasks leave the
// corresponding field nil; rendering falls back to the dimmed "·" placeholder.
//
// Tasks run concurrently with a pool bounded to 2×runtime.NumCPU goroutines.
func Collect(rows []Row, mainBranch string) {
	concurrency := max(runtime.NumCPU()*2, 2)
	sem := make(chan struct{}, concurrency)
	var wg sync.WaitGroup
	for i := range rows {
		if rows[i].IsBare {
			continue
		}
		wg.Add(1)
		sem <- struct{}{}
		go func(row *Row) {
			defer wg.Done()
			defer func() { <-sem }()
			collectRow(row, mainBranch)
		}(&rows[i])
	}
	wg.Wait()
}

func collectRow(row *Row, mainBranch string) {
	// Tasks run sequentially within a worktree; errors leave fields nil.
	row.CommitDetails = fetchCommitDetails(row.Path)
	row.WorkingTreeStatus, row.WorkingTreeDiff = fetchStatusAndDiff(row.Path)
	if mainBranch != "" && !row.IsDetached && row.Branch != mainBranch {
		row.MainCounts = fetchAheadBehind(row.Path, mainBranch)
	}
	row.Upstream = fetchUpstream(row.Path)
	row.GitOp = fetchGitOperation(row.Path)
}

// runGit runs a git command with a timeout, returning its stdout.
func runGit(timeout time.Duration, args ...string) ([]byte, error) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	cmd := exec.CommandContext(ctx, "git", args...)
	return cmd.Output()
}

// fetchCommitDetails retrieves short SHA, timestamp, and subject for HEAD.
func fetchCommitDetails(path string) *CommitDetails {
	// null-separated fields for safe multi-field parsing
	out, err := runGit(time.Second, "-C", path, "log", "-1", "--format=%h%x00%ct%x00%s", "HEAD")
	if err != nil {
		return nil
	}
	parts := strings.SplitN(strings.TrimRight(string(out), "\n"), "\x00", 3)
	if len(parts) < 3 {
		return nil
	}
	ts, err := strconv.ParseInt(parts[1], 10, 64)
	if err != nil {
		return nil
	}
	return &CommitDetails{
		ShortSHA:  parts[0],
		Timestamp: time.Unix(ts, 0),
		Message:   parts[2],
	}
}

// fetchStatusAndDiff returns working-tree status flags and line-diff counts.
// Two git commands are issued: status --porcelain for flags, diff HEAD --shortstat
// for line counts.
func fetchStatusAndDiff(path string) (*WorkingTreeStatus, *LineDiff) {
	statusOut, err := runGit(time.Second, "-C", path, "status", "--porcelain=v1")
	if err != nil {
		return nil, nil
	}
	st := parseStatus(string(statusOut))

	diffOut, err := runGit(time.Second, "-C", path, "diff", "HEAD", "--shortstat")
	if err != nil {
		return st, &LineDiff{} // status succeeded; diff failed → zero counts
	}
	diff := parseShortstat(string(diffOut))
	return st, diff
}

func parseStatus(output string) *WorkingTreeStatus {
	var st WorkingTreeStatus
	for _, line := range strings.Split(output, "\n") {
		if len(line) < 2 {
			continue
		}
		x, y := line[0], line[1]
		// Conflict markers
		if x == 'U' || y == 'U' || (x == 'A' && y == 'A') || (x == 'D' && y == 'D') {
			st.Conflicts = true
		}
		// Staged: explicit allowlist of index statuses that mean staged changes.
		// Using a positive check avoids false positives from 'U' (unmerged/conflict)
		// or any unexpected characters in the porcelain output.
		switch x {
		case 'M', 'A', 'D', 'R', 'C', 'T':
			st.Staged = true
		}
		// Working tree
		switch y {
		case 'M':
			st.Modified = true
		case '?':
			st.Untracked = true
		case 'R':
			st.Renamed = true
		case 'D':
			st.Deleted = true
		}
	}
	return &st
}

func parseShortstat(output string) *LineDiff {
	// e.g. " 3 files changed, 14 insertions(+), 2 deletions(-)"
	var d LineDiff
	for _, part := range strings.Split(output, ",") {
		part = strings.TrimSpace(part)
		f := strings.Fields(part)
		if len(f) < 2 {
			continue
		}
		n, _ := strconv.Atoi(f[0])
		switch {
		case strings.HasPrefix(f[1], "insertion"):
			d.Added = n
		case strings.HasPrefix(f[1], "deletion"):
			d.Deleted = n
		}
	}
	return &d
}

// fetchAheadBehind counts commits ahead/behind mainBranch using rev-list.
func fetchAheadBehind(path, mainBranch string) *AheadBehind {
	// "--left-right --count A...B" prints "<behind> <ahead>" relative to B
	out, err := runGit(time.Second, "-C", path, "rev-list", "--left-right", "--count",
		mainBranch+"...HEAD")
	if err != nil {
		return nil
	}
	parts := strings.Fields(strings.TrimSpace(string(out)))
	if len(parts) != 2 {
		return nil
	}
	behind, _ := strconv.Atoi(parts[0])
	ahead, _ := strconv.Atoi(parts[1])
	return &AheadBehind{Ahead: ahead, Behind: behind}
}

// fetchUpstream returns the upstream tracking status, or nil if no upstream.
func fetchUpstream(path string) *UpstreamStatus {
	out, err := runGit(time.Second, "-C", path, "rev-parse",
		"--abbrev-ref", "--symbolic-full-name", "@{upstream}")
	if err != nil {
		return nil // exit 128 = no upstream configured
	}
	remote := strings.TrimSpace(string(out))

	countOut, err := runGit(time.Second, "-C", path, "rev-list",
		"--left-right", "--count", "@{upstream}...HEAD")
	if err != nil {
		return &UpstreamStatus{Remote: remote}
	}
	parts := strings.Fields(strings.TrimSpace(string(countOut)))
	if len(parts) != 2 {
		return &UpstreamStatus{Remote: remote}
	}
	behind, _ := strconv.Atoi(parts[0])
	ahead, _ := strconv.Atoi(parts[1])
	return &UpstreamStatus{Remote: remote, Ahead: ahead, Behind: behind}
}

// fetchGitOperation probes filesystem markers for in-progress git operations.
//
// For linked worktrees, .git is a file (gitlink), not a directory. The actual
// operation markers live in the real git dir returned by `git rev-parse --git-dir`.
// Using that path ensures correct detection for all worktrees.
func fetchGitOperation(path string) GitOperation {
	// Resolve the real git directory (handles both main and linked worktrees).
	out, err := runGit(time.Second, "-C", path, "rev-parse", "--git-dir")
	if err != nil {
		return GitOpNone
	}
	gitDir := strings.TrimSpace(string(out))
	if !filepath.IsAbs(gitDir) {
		gitDir = filepath.Join(path, gitDir)
	}

	checks := []struct {
		marker string
		op     GitOperation
	}{
		{"MERGE_HEAD", GitOpMerge},
		{"rebase-merge", GitOpRebase},
		{"rebase-apply", GitOpRebase},
		{"CHERRY_PICK_HEAD", GitOpCherryPick},
	}
	for _, c := range checks {
		if _, err := os.Stat(filepath.Join(gitDir, c.marker)); err == nil {
			return c.op
		}
	}
	return GitOpNone
}
