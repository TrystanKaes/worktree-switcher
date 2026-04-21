package columns

import "github.com/mattn/go-runewidth"

// emptyPenalty is added to a column's effective priority when no row has data
// for that column. Mirrors worktrunk layout.rs:299, EMPTY_PENALTY = 10.
const emptyPenalty = 10

// Fixed widths for columns that never change size (worktrunk §3).
const (
	gutterWidth      = 1 // single glyph
	statusWidth      = 8 // 7-gate mask + trailing space
	workingDiffWidth = 9 // "+999 -999"
	aheadBehindWidth = 7 // "↑99 ↓99"
	upstreamWidth    = 7 // "⇡99 ⇣99"
	commitWidth      = 7 // short SHA
	ageWidth         = 4 // "11mo"

	// Flex column caps
	maxBranchWidth  = 40
	maxPathWidth    = 60
	maxMessageWidth = 80

	// Spacing between columns (0 between Gutter and its successor).
	colSpacing = 2
)

// Layout is the result of Allocate: which columns fit and how wide each is.
type Layout struct {
	Allocations []ColAlloc
	HiddenCount int
	TotalWidth  int
}

// ColAlloc records the allocated width of a single column.
type ColAlloc struct {
	Kind   ColumnKind
	Width  int
	Shrunk bool // true if a shrinkable column was reduced to header width
}

// alwaysHasData returns true for columns that are treated as always populated
// regardless of whether the rich data fields are nil. Mirrors worktrunk §3
// (Gutter, Branch, Time, Commit, Message).
func alwaysHasData(kind ColumnKind) bool {
	switch kind {
	case ColGutter, ColBranch, ColPath, ColCommit, ColAge, ColMessage:
		return true
	}
	return false
}

// hasData checks whether any row carries non-zero data for the given column.
func hasData(rows []Row, kind ColumnKind) bool {
	for _, r := range rows {
		switch kind {
		case ColStatus:
			if r.WorkingTreeStatus != nil || r.Upstream != nil {
				return true
			}
		case ColWorkingDiff:
			if r.WorkingTreeDiff != nil &&
				(r.WorkingTreeDiff.Added > 0 || r.WorkingTreeDiff.Deleted > 0) {
				return true
			}
		case ColAheadBehind:
			if r.MainCounts != nil {
				return true
			}
		case ColUpstream:
			if r.Upstream != nil && r.Upstream.Remote != "" {
				return true
			}
		}
	}
	return false
}

// contentWidth computes the initial (pre-shrink) display width for a column
// given the full row set.
func contentWidth(rows []Row, spec ColumnSpec) int {
	switch spec.Kind {
	case ColGutter:
		return gutterWidth
	case ColStatus:
		return statusWidth
	case ColWorkingDiff:
		return workingDiffWidth
	case ColAheadBehind:
		return aheadBehindWidth
	case ColUpstream:
		return upstreamWidth
	case ColCommit:
		return commitWidth
	case ColAge:
		return ageWidth
	case ColBranch:
		headerW := runewidth.StringWidth(spec.Header)
		maxW := headerW
		for _, r := range rows {
			w := runewidth.StringWidth(r.Branch)
			if w > maxW {
				maxW = w
			}
		}
		if maxW > maxBranchWidth {
			maxW = maxBranchWidth
		}
		return maxW
	case ColPath:
		headerW := runewidth.StringWidth(spec.Header)
		maxW := headerW
		for _, r := range rows {
			w := runewidth.StringWidth(r.ShortPath)
			if w > maxW {
				maxW = w
			}
		}
		if maxW > maxPathWidth {
			maxW = maxPathWidth
		}
		return maxW
	case ColMessage:
		headerW := runewidth.StringWidth(spec.Header)
		maxW := headerW
		for _, r := range rows {
			if r.CommitDetails != nil {
				w := runewidth.StringWidth(r.CommitDetails.Message)
				if w > maxW {
					maxW = w
				}
			}
		}
		if maxW > maxMessageWidth {
			maxW = maxMessageWidth
		}
		return maxW
	}
	return 0
}

// Allocate computes which columns fit in termWidth and how wide each is.
//
// Algorithm (mirrors worktrunk layout.rs:662-891):
//  1. Compute effective priority: base + emptyPenalty if column has no data.
//  2. Compute content widths for each column.
//  3. Include all columns; then repeatedly drop the lowest-importance column
//     (highest effective priority) until the total fits within termWidth.
//     Shrinkable columns are reduced to header width before being dropped.
func Allocate(rows []Row, termWidth int) Layout {
	type entry struct {
		spec         ColumnSpec
		width        int // current allocated width (may shrink)
		effectivePri int
		shrunk       bool
		dropped      bool
	}

	entries := make([]entry, len(Specs))
	for i, spec := range Specs {
		ep := int(spec.BasePriority)
		if !alwaysHasData(spec.Kind) && !hasData(rows, spec.Kind) {
			ep += emptyPenalty
		}
		entries[i] = entry{
			spec:         spec,
			width:        contentWidth(rows, spec),
			effectivePri: ep,
		}
	}

	// totalWidth computes the current total display width of non-dropped entries.
	totalWidth := func() int {
		total := 0
		prevKind := ColumnKind(-1)
		for _, e := range entries {
			if e.dropped {
				continue
			}
			if prevKind >= 0 && prevKind != ColGutter {
				total += colSpacing
			}
			total += e.width
			prevKind = e.spec.Kind
		}
		return total
	}

	// Drop or shrink the least-important included column until everything fits.
	for totalWidth() > termWidth {
		// Find the included entry with the highest effective priority to drop.
		worst := -1
		worstPri := -1
		for i, e := range entries {
			if e.dropped {
				continue
			}
			if e.effectivePri > worstPri {
				worstPri = e.effectivePri
				worst = i
			}
		}
		if worst == -1 {
			break // nothing left to drop
		}

		// Shrinkable columns get one pass at header-width before being dropped.
		if entries[worst].spec.Shrinkable && !entries[worst].shrunk {
			hw := runewidth.StringWidth(entries[worst].spec.Header)
			if entries[worst].width > hw {
				entries[worst].width = hw
				entries[worst].shrunk = true
				continue
			}
		}
		entries[worst].dropped = true
	}

	// Build result.
	var allocs []ColAlloc
	hiddenCount := 0
	total := 0
	prevKind := ColumnKind(-1)
	for _, e := range entries {
		if e.dropped {
			hiddenCount++
			continue
		}
		if prevKind >= 0 && prevKind != ColGutter {
			total += colSpacing
		}
		allocs = append(allocs, ColAlloc{
			Kind:   e.spec.Kind,
			Width:  e.width,
			Shrunk: e.shrunk,
		})
		total += e.width
		prevKind = e.spec.Kind
	}

	return Layout{
		Allocations: allocs,
		HiddenCount: hiddenCount,
		TotalWidth:  total,
	}
}
