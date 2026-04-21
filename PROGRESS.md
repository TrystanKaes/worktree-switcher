# Plan: Mirror worktrunk list UI — rich columns + footer

**Plan file:** plans/plan-worktrunk-list-ui.md  
**Started:** 2026-04-21T00:00:00Z  
**Checkpointed at:** 2026-04-21T03:00:00Z  
**Current:** COMPLETE

## Status

All 10 steps complete. Post-review bug fixes applied.

## Checklist

- [x] **Step 1:** Data types + Row struct *(2026-04-21)*
- [x] **Step 2:** Parallel collector *(2026-04-21)*
- [x] **Step 3:** Column registry + layout *(2026-04-21)*
- [x] **Step 4:** Cell renderers *(2026-04-21)*
- [x] **Step 5:** Footer *(2026-04-21)*
- [x] **Step 6:** `wt list` integration *(2026-04-21)*
- [x] **Step 7:** `wti` TUI integration *(2026-04-21)*
- [x] **Step 8:** Responsive width *(2026-04-21)* (done as part of 6+7)
- [x] **Step 9:** README update *(2026-04-21)*
- [x] **Step 10:** Final polish + bug fixes *(2026-04-21)*

## Post-review fixes applied

1. **Typed-nil panic in `cellDiff`**: Replaced single generic `cellDiff(interface{...})` with two typed functions `cellLineDiff(*LineDiff, int)` and `cellAheadBehindDiff(*AheadBehind, int)`. The original design passed typed nil pointers through an interface, causing `d == nil` to evaluate false → nil dereference panic.

2. **`fetchGitOperation` broken for linked worktrees**: Linked worktrees have `.git` as a regular file (gitlink), not a directory. Changed implementation to call `git -C <path> rev-parse --git-dir` to resolve the real git directory, then probe markers (MERGE_HEAD, rebase-merge, etc.) relative to that path.

3. **Unchecked type assertion in RunTUI**: `result.(model)` → `final, ok := result.(model)` with graceful nil return on failure.

## Files created

- `columns/stats.go`
- `columns/row.go`
- `columns/collect.go`
- `columns/columns.go`
- `columns/layout.go`
- `columns/render.go`
- `columns/footer.go`

## Files modified

- `main.go` — new imports, rewritten runList, updated runInteractive, toRows(), listStyles()
- `ui.go` — model with rows/width/layout/footer, tuiStyles(), WindowSizeMsg, new render block, updated RunTUI
- `README.md` — updated wt list description, added List columns section
- `go.mod` — mattn/go-runewidth and charmbracelet/x/term promoted to direct

## Last gate

**Timestamp:** 2026-04-21
- `go build .` → **PASS**
- `go vet ./...` → **PASS**
- `gofmt -l .` → **PASS**
- `make build` → **PASS**
- `wt list | cat` → **PASS** (legacy 3-col format preserved)

## Architecture decision

Plan said extend `Worktree` struct with pointer fields from `columns` package — circular import. Used `columns.Row` parallel slice instead. TUI model keeps `[]Worktree` for operations + `[]columns.Row` for rendering.

Plan said use `golang.org/x/term` — used `charmbracelet/x/term` (already in go.sum) instead. Same API, no new dep.
