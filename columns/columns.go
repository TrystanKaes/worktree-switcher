package columns

// ColumnKind identifies a display column by role.
type ColumnKind int

const (
	ColGutter      ColumnKind = iota // @/^/+ glyph
	ColBranch                        // branch name (shrinkable)
	ColStatus                        // six-gate status symbols
	ColWorkingDiff                   // HEAD± working-tree line diff
	ColAheadBehind                   // main↕ commits ahead/behind main
	ColUpstream                      // Remote⇅ upstream divergence
	ColPath                          // worktree path
	ColCommit                        // 7-char short SHA
	ColAge                           // relative age
	ColMessage                       // commit subject
)

// ColumnSpec describes a column entry in the static registry.
type ColumnSpec struct {
	Kind         ColumnKind
	BasePriority uint8 // lower = more important; governs drop order (worktrunk §2)
	Shrinkable   bool  // may be reduced to header width before being dropped
	Header       string
}

// Specs is the static column registry in display order.
// Priority values match worktrunk §2, adjusted for dropped columns
// (BranchDiff, CiStatus, Summary, Url are not ported).
var Specs = []ColumnSpec{
	{ColGutter, 0, false, ""},
	{ColBranch, 1, true, "Branch"},
	{ColStatus, 2, false, "Status"},
	{ColWorkingDiff, 3, false, "HEAD±"},
	{ColAheadBehind, 4, false, "main↕"},
	{ColUpstream, 5, false, "Remote⇅"},
	{ColPath, 6, false, "Path"},
	{ColCommit, 7, false, "Commit"},
	{ColAge, 8, false, "Age"},
	{ColMessage, 9, false, "Message"},
}

// specFor returns the ColumnSpec for the given kind.
// Panics if the kind is not in Specs (programming error).
func specFor(kind ColumnKind) ColumnSpec {
	for _, s := range Specs {
		if s.Kind == kind {
			return s
		}
	}
	panic("columns: unknown ColumnKind")
}
