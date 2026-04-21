package columns

import "time"

// Row is the rich data container for a single worktree entry.
// Basic fields are populated from git.go's Worktree via ToRow;
// rich fields (WorkingTreeStatus, etc.) are filled in by Collect.
//
// This type is the central data carrier for all rendering and layout
// functions in this package, avoiding an import cycle between
// package columns and package main.
type Row struct {
	// Identity — populated at conversion time.
	Path       string
	Branch     string
	ShortPath  string // $HOME replaced with ~
	Commit     string // 40-char SHA from git porcelain
	IsBare     bool
	ModifiedAt time.Time

	// State flags — populated at conversion time or by Collect.
	IsDetached bool // branch == "(detached)"
	IsCurrent  bool // path matches current working directory
	IsMain     bool // branch matches detectMainBranch()
	IsPrevious bool // path matches __WT_LAST_DIR

	// Rich data — nil means not yet collected or unavailable.
	WorkingTreeStatus *WorkingTreeStatus
	WorkingTreeDiff   *LineDiff
	MainCounts        *AheadBehind
	Upstream          *UpstreamStatus
	CommitDetails     *CommitDetails
	GitOp             GitOperation
}
