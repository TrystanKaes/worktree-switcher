package columns

import "time"

// LineDiff counts added and deleted lines in working tree or branch diff.
type LineDiff struct {
	Added   int
	Deleted int
}

// AheadBehind counts commits ahead/behind a reference branch.
type AheadBehind struct {
	Ahead  int
	Behind int
}

// UpstreamStatus describes the upstream remote tracking branch.
// Remote == "" means no upstream is configured.
type UpstreamStatus struct {
	Remote string
	Ahead  int
	Behind int
}

// CommitDetails holds metadata for the HEAD commit.
type CommitDetails struct {
	Timestamp time.Time
	ShortSHA  string
	Message   string
}

// WorkingTreeStatus flags the types of working-tree changes present.
type WorkingTreeStatus struct {
	Staged    bool
	Modified  bool
	Untracked bool
	Renamed   bool
	Deleted   bool
	Conflicts bool
}

// IsDirty reports whether the working tree has any changes.
func (s WorkingTreeStatus) IsDirty() bool {
	return s.Staged || s.Modified || s.Untracked || s.Renamed || s.Deleted || s.Conflicts
}

// GitOperation records any in-progress git operation.
type GitOperation int

const (
	GitOpNone       GitOperation = iota
	GitOpMerge                   // .git/MERGE_HEAD present
	GitOpRebase                  // .git/rebase-merge or .git/rebase-apply present
	GitOpCherryPick              // .git/CHERRY_PICK_HEAD present
)
