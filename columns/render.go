package columns

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/mattn/go-runewidth"
)

// Styles holds lipgloss styles for the renderer.
// Constructed by the caller so the colour palette is consistent with the TUI.
type Styles struct {
	Dim      lipgloss.Style
	Addition lipgloss.Style
	Deletion lipgloss.Style
	Selected lipgloss.Style
	Delete   lipgloss.Style
	Branch   lipgloss.Style
	Danger   lipgloss.Style
	Header   lipgloss.Style
	Prev     lipgloss.Style
}

// RenderHeader returns the header row string for the given layout.
func RenderHeader(l Layout, styles Styles) string {
	var parts []string
	for _, alloc := range l.Allocations {
		spec := specFor(alloc.Kind)
		var cell string
		if spec.Kind == ColGutter {
			cell = padRight("", alloc.Width)
		} else {
			cell = padRight(spec.Header, alloc.Width)
		}
		parts = append(parts, styles.Header.Render(cell))
	}
	return joinWithSpacing(l.Allocations, parts)
}

// RenderRow returns the rendered row string.
//
//   - highlighted: cursor is on this row — use named styles on key cells.
//   - inDeleteMode: delete mode active — use Delete style on highlighted rows.
//   - isPrev: this is the pinned previous worktree row.
func RenderRow(row Row, l Layout, styles Styles, highlighted, inDeleteMode, isPrev bool) string {
	var parts []string
	for _, alloc := range l.Allocations {
		raw := rawCell(row, alloc)
		padded := padRight(raw, alloc.Width)

		var cell string
		if highlighted {
			cell = styledCellHighlighted(row, alloc, padded, styles, inDeleteMode)
		} else {
			cell = styles.Dim.Render(padded)
		}
		parts = append(parts, cell)
	}

	result := joinWithSpacing(l.Allocations, parts)

	if isPrev {
		result += "  " + styles.Prev.Render("(prev)")
	}
	return result
}

// rawCell returns the unstyled, unpadded content for a cell.
func rawCell(row Row, alloc ColAlloc) string {
	switch alloc.Kind {
	case ColGutter:
		return cellGutter(row)
	case ColBranch:
		return truncLeft(row.Branch, alloc.Width)
	case ColStatus:
		return cellStatus(row, alloc.Width)
	case ColWorkingDiff:
		return cellLineDiff(row.WorkingTreeDiff, alloc.Width)
	case ColAheadBehind:
		return cellAheadBehindDiff(row.MainCounts, alloc.Width)
	case ColUpstream:
		return cellUpstream(row, alloc.Width)
	case ColPath:
		return truncLeft(row.ShortPath, alloc.Width)
	case ColCommit:
		return cellCommit(row, alloc.Width)
	case ColAge:
		return cellAge(row, alloc.Width)
	case ColMessage:
		return cellMessage(row, alloc.Width)
	}
	return ""
}

// styledCellHighlighted applies per-column styles to a highlighted (cursor) row.
func styledCellHighlighted(_ Row, alloc ColAlloc, padded string, styles Styles, inDeleteMode bool) string {
	switch alloc.Kind {
	case ColBranch, ColPath:
		if inDeleteMode {
			return styles.Delete.Render(padded)
		}
		return styles.Selected.Render(padded)
	case ColGutter:
		if inDeleteMode {
			return styles.Delete.Render(padded)
		}
		return styles.Selected.Render(padded)
	case ColAge, ColCommit:
		return styles.Dim.Render(padded)
	case ColStatus:
		return padded // status has inline ANSI from cellStatus; don't re-wrap
	default:
		return padded
	}
}

// cellGutter returns the single-character gutter symbol.
// Worktrunk §2: @ current, ^ main, + regular worktree.
func cellGutter(row Row) string {
	switch {
	case row.IsCurrent:
		return "@"
	case row.IsMain:
		return "^"
	default:
		return "+"
	}
}

// cellStatus builds the Status cell: 3 work-flag chars + op char + main char + remote char.
// Placeholder "·" used when source data is nil.
func cellStatus(row Row, width int) string {
	// Work flags (up to 3 chars, left-to-right): staged, modified, untracked, renamed, deleted
	var g1 string
	if row.WorkingTreeStatus == nil {
		g1 = "···" // loading placeholder
	} else {
		s := row.WorkingTreeStatus
		var flags []string
		if s.Staged {
			flags = append(flags, "●")
		}
		if s.Modified {
			flags = append(flags, "!")
		}
		if s.Untracked {
			flags = append(flags, "?")
		}
		if s.Renamed {
			flags = append(flags, "»")
		}
		if s.Deleted {
			flags = append(flags, "✘")
		}
		// Conflicts omitted here — the op char already shows ✘ for conflict state.
		switch len(flags) {
		case 0:
			g1 = "✓  "
		case 1:
			g1 = flags[0] + "  "
		case 2:
			g1 = flags[0] + flags[1] + " "
		default:
			g1 = flags[0] + flags[1] + flags[2]
		}
	}

	// Op char (1 char): in-progress git operation, or conflict
	var g2 string
	switch row.GitOp {
	case GitOpMerge:
		g2 = "⤵"
	case GitOpRebase:
		g2 = "⤴"
	case GitOpCherryPick:
		g2 = "⊞"
	default:
		g2 = " "
	}
	if row.WorkingTreeStatus != nil && row.WorkingTreeStatus.Conflicts {
		g2 = "✘"
	}

	// Main char (1 char): relationship to the main branch.
	// IsMain shows space — gutter already carries the ^ symbol.
	var g3 string
	switch {
	case row.IsMain:
		g3 = " "
	case row.IsDetached:
		g3 = "⚑"
	case row.MainCounts == nil:
		g3 = "·" // not yet collected
	case row.MainCounts.Ahead > 0 && row.MainCounts.Behind > 0:
		g3 = "↕"
	case row.MainCounts.Ahead > 0:
		g3 = "↑"
	case row.MainCounts.Behind > 0:
		g3 = "↓"
	default:
		g3 = "✓" // in sync with main
	}

	// Remote char (1 char): divergence from upstream remote
	var g4 string
	switch {
	case row.Upstream == nil:
		g4 = "·" // not yet collected (or no upstream if explicitly empty Remote)
	case row.Upstream.Remote == "":
		g4 = " " // no upstream
	case row.Upstream.Ahead > 0 && row.Upstream.Behind > 0:
		g4 = "⇅"
	case row.Upstream.Ahead > 0:
		g4 = "⇡"
	case row.Upstream.Behind > 0:
		g4 = "⇣"
	default:
		g4 = "✓" // in sync with remote
	}

	raw := g1 + g2 + g3 + g4
	return padRight(raw, width)
}

// cellLineDiff formats a LineDiff as "+A -D".
// Typed parameter avoids the Go typed-nil-in-interface bug.
func cellLineDiff(d *LineDiff, width int) string {
	if d == nil {
		return padRight("·", width)
	}
	return padRight(fmt.Sprintf("+%s -%s", fmtCount(d.Added), fmtCount(d.Deleted)), width)
}

// cellAheadBehindDiff formats an AheadBehind as "↑A ↓B".
// Typed parameter avoids the Go typed-nil-in-interface bug.
func cellAheadBehindDiff(d *AheadBehind, width int) string {
	if d == nil {
		return padRight("·", width)
	}
	return padRight(fmt.Sprintf("↑%s ↓%s", fmtCount(d.Ahead), fmtCount(d.Behind)), width)
}

// fmtCount formats an integer for diff columns, using K/∞ overflow notation.
// Mirrors worktrunk render.rs:100-179.
func fmtCount(n int) string {
	switch {
	case n < 0:
		return "0"
	case n < 1000:
		return fmt.Sprintf("%d", n)
	case n < 10000:
		return fmt.Sprintf("%dK", n/1000)
	default:
		return "∞"
	}
}

// cellUpstream formats upstream divergence as "⇡A ⇣B" or blank.
func cellUpstream(row Row, width int) string {
	if row.Upstream == nil {
		return padRight("·", width)
	}
	if row.Upstream.Remote == "" {
		return padRight("", width)
	}
	u := row.Upstream
	if u.Ahead == 0 && u.Behind == 0 {
		return padRight("✓", width)
	}
	return padRight(fmt.Sprintf("⇡%s ⇣%s", fmtCount(u.Ahead), fmtCount(u.Behind)), width)
}

// cellCommit returns the 7-char short SHA, or "·" placeholder.
func cellCommit(row Row, width int) string {
	if row.CommitDetails != nil && row.CommitDetails.ShortSHA != "" {
		return padRight(row.CommitDetails.ShortSHA, width)
	}
	// Fall back to first 7 chars of the 40-char Commit field
	if len(row.Commit) >= 7 {
		return row.Commit[:7]
	}
	return padRight("·", width)
}

// cellAge formats the age of the row. Uses CommitDetails.Timestamp when
// available, falling back to ModifiedAt.
// Compact format: "Nm", "Nh", "Nd", "Nw", "Nmo", "Ny". Right-aligned in
// ageWidth (4) chars. Mirrors worktrunk §3.
func cellAge(row Row, width int) string {
	var t time.Time
	if row.CommitDetails != nil && !row.CommitDetails.Timestamp.IsZero() {
		t = row.CommitDetails.Timestamp
	} else if !row.ModifiedAt.IsZero() {
		t = row.ModifiedAt
	} else {
		return padRight("·", width)
	}
	return padLeft(compactAge(time.Since(t)), width)
}

func compactAge(d time.Duration) string {
	switch {
	case d < time.Minute:
		return "now"
	case d < time.Hour:
		return fmt.Sprintf("%dm", int(d.Minutes()))
	case d < 24*time.Hour:
		return fmt.Sprintf("%dh", int(d.Hours()))
	case d < 7*24*time.Hour:
		return fmt.Sprintf("%dd", int(d.Hours()/24))
	case d < 4*7*24*time.Hour:
		return fmt.Sprintf("%dw", int(d.Hours()/(24*7)))
	case d < 12*30*24*time.Hour:
		return fmt.Sprintf("%dmo", int(d.Hours()/(24*30)))
	default:
		return fmt.Sprintf("%dy", int(d.Hours()/(24*365)))
	}
}

// cellMessage returns the commit subject, right-truncated.
func cellMessage(row Row, width int) string {
	if row.CommitDetails == nil || row.CommitDetails.Message == "" {
		return padRight("", width)
	}
	return truncRight(row.CommitDetails.Message, width)
}

// joinWithSpacing concatenates cell strings with inter-column spacing.
// No spacing is added before the column that immediately follows ColGutter.
func joinWithSpacing(allocs []ColAlloc, parts []string) string {
	if len(allocs) == 0 {
		return ""
	}
	var b strings.Builder
	for i, part := range parts {
		if i > 0 && allocs[i-1].Kind != ColGutter {
			b.WriteString(strings.Repeat(" ", colSpacing))
		}
		b.WriteString(part)
	}
	return b.String()
}

// padRight pads s to exactly width display cells (right-padded with spaces).
// If s is wider than width, it is truncated (left-biased).
func padRight(s string, width int) string {
	w := runewidth.StringWidth(s)
	if w >= width {
		return truncLeft(s, width)
	}
	return s + strings.Repeat(" ", width-w)
}

// padLeft pads s to exactly width display cells (left-padded with spaces).
func padLeft(s string, width int) string {
	w := runewidth.StringWidth(s)
	if w >= width {
		return truncLeft(s, width)
	}
	return strings.Repeat(" ", width-w) + s
}

// truncLeft truncates s to at most width display cells, adding "…" prefix
// when truncation occurs (left-biased: beginning of string is dropped).
func truncLeft(s string, width int) string {
	w := runewidth.StringWidth(s)
	if w <= width {
		return s
	}
	if width <= 1 {
		return "…"
	}
	// Drop runes from the front until we fit.
	runes := []rune(s)
	for len(runes) > 0 {
		candidate := "…" + string(runes)
		if runewidth.StringWidth(candidate) <= width {
			return candidate
		}
		runes = runes[1:]
	}
	return "…"
}

// truncRight truncates s to at most width display cells, adding "…" suffix.
func truncRight(s string, width int) string {
	w := runewidth.StringWidth(s)
	if w <= width {
		return s + strings.Repeat(" ", width-w)
	}
	if width <= 1 {
		return "…"
	}
	runes := []rune(s)
	for len(runes) > 0 {
		candidate := string(runes) + "…"
		if runewidth.StringWidth(candidate) <= width {
			return candidate + strings.Repeat(" ", width-runewidth.StringWidth(candidate))
		}
		runes = runes[:len(runes)-1]
	}
	return "…"
}
