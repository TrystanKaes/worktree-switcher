package columns

import (
	"fmt"
	"strings"
)

// Metrics aggregates summary counts for the footer line.
// Mirrors worktrunk SummaryMetrics (mod.rs:201-207).
type Metrics struct {
	Worktrees      int
	DirtyWorktrees int // rows with working-tree changes
	AheadItems     int // rows with MainCounts.Ahead > 0
	HiddenColumns  int
}

// Aggregate computes Metrics from a set of rows.
// hidden is the HiddenCount from Layout.
func Aggregate(rows []Row, hidden int) Metrics {
	m := Metrics{HiddenColumns: hidden}
	for _, r := range rows {
		if r.IsBare {
			continue
		}
		m.Worktrees++
		if r.WorkingTreeStatus != nil && r.WorkingTreeStatus.IsDirty() {
			m.DirtyWorktrees++
		}
		if r.MainCounts != nil && r.MainCounts.Ahead > 0 {
			m.AheadItems++
		}
	}
	return m
}

// Format renders the footer summary string.
// Example: "○ Showing 6 worktrees, 3 with changes, 2 ahead, 4 columns hidden"
// Mirrors worktrunk format_summary_message (mod.rs:287-319).
func Format(m Metrics) string {
	var parts []string

	// 1. N worktree[s] (always)
	parts = append(parts, plural(m.Worktrees, "worktree", "worktrees"))

	// 2. N with changes
	if m.DirtyWorktrees > 0 {
		parts = append(parts, fmt.Sprintf("%d with changes", m.DirtyWorktrees))
	}

	// 3. N ahead
	if m.AheadItems > 0 {
		parts = append(parts, plural(m.AheadItems, "ahead", "ahead"))
	}

	// 4. N column[s] hidden
	if m.HiddenColumns > 0 {
		parts = append(parts, plural(m.HiddenColumns, "column hidden", "columns hidden"))
	}

	return "○ Showing " + strings.Join(parts, ", ")
}

func plural(n int, singular, plur string) string {
	if n == 1 {
		return fmt.Sprintf("%d %s", n, singular)
	}
	return fmt.Sprintf("%d %s", n, plur)
}
