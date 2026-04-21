package columns

import "strings"

// RenderLegend returns a compact multi-line legend for the gutter and Status column.
// The Status cell is: [3 work flags][op][main][remote].
//
// Pass a zero-value Styles{} for plain (uncolored) output — suitable for help
// text or piping. Pass TUI/list styles for coloured output.
func RenderLegend(styles Styles) string {
	label := func(s string) string { return styles.Header.Render(s) }
	sym := func(s string) string { return styles.Branch.Render(s) }
	desc := func(s string) string { return styles.Dim.Render(s) }
	sp := "  "

	entry := func(s, d string) string { return sym(s) + " " + desc(d) }

	lines := []string{
		label("Status legend"),
		// Gutter column
		"  " + label("Gutter") + sp + entry("@", "current") + sp + entry("^", "main") + sp + entry("+", "worktree"),
		// Status column: work flags + in-progress ops + detached
		"  " + label("Flags ") + sp + entry("✓", "clean") + sp + entry("●", "staged") + sp + entry("!", "modified") + sp + entry("?", "untracked") + sp + entry("»", "renamed") + sp + entry("✘", "deleted") + sp + entry("⤵", "merge") + sp + entry("⤴", "rebase") + sp + entry("⊞", "cherry-pick") + sp + entry("✘", "conflicts") + sp + entry("⚑", "detached"),
		// Divergence symbols — shared meaning across main↕ and Remote⇅ columns
		"  " + label("Sync  ") + sp + entry("↑/⇡", "ahead") + sp + entry("↓/⇣", "behind") + sp + entry("↕/⇅", "both") + sp + entry("✓", "synced") + sp + entry("·", "loading"),
	}
	return strings.Join(lines, "\n")
}
