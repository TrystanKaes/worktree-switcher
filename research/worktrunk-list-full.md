# worktrunk `wt list --full` — table, columns, footer

Research findings only. No recommendations. Every claim cites `path:line` from the cloned worktrunk repo at `/tmp/research/worktrunk/`.

## Scope

How worktrunk (Rust CLI, https://github.com/max-sixty/worktrunk) implements the visibility features of `wt list --full`:

- The table view and its columns (Branch, Status, HEAD±, main↕, main…±, Summary, Remote⇅, CI, plus Path, URL, Commit, Age, Message).
- Terminal-width-aware column hiding.
- The bottom helper line (e.g. `Showing 6 worktrees, 3 with changes, 2 ahead, 4 columns hidden`).
- The underlying data model and how it is computed per worktree.
- The progressive rendering pipeline that makes all of the above feel instant.

Out of scope: picker UI, statusline segment, JSON output, Git operations outside the list path.

---

## Findings

### 1. `--full` flag: parsing and consumption

- Clap arg declared `src/cli/mod.rs:385` as `pub(crate) full: bool` with `#[arg(long)]`; help text at `src/cli/mod.rs:383` is `"Show CI, diff analysis, and LLM summaries"`.
- Handler receives it as `cli_full: bool` in `src/commands/list/mod.rs:151` and forwards to the collector at `src/commands/list/mod.rs:173`.
- Final resolution in `src/commands/list/collect/mod.rs:596-607`:
  ```rust
  let show_full = cli_full || config.list.full();
  let skip_tasks: HashSet<TaskKind> = if show_full {
      HashSet::new()
  } else {
      [TaskKind::BranchDiff, TaskKind::CiStatus, TaskKind::SummaryGenerate]
          .into_iter().collect()
  };
  ```
- Timeouts differ: when `show_full == true`, `(task_timeout, collect_deadline)` are `(None, None)` (`src/commands/list/collect/mod.rs:609-615`).

The flag gates exactly three tasks. Every other column is present in both modes and only hides via the width-driven layout.

---

### 2. Column registry

`src/commands/list/columns.rs` (246 lines, read in full).

- `ColumnKind` enum at `src/commands/list/columns.rs:5-20` lists all fourteen columns: `Gutter, Branch, Status, WorkingDiff, AheadBehind, BranchDiff, Summary, Upstream, CiStatus, Path, Url, Commit, Time, Message`.
- Gutter glyphs defined at `src/commands/list/columns.rs:6`: `@` current, `^` main, `+` regular worktree, space for branch-only rows.
- Headers via `ColumnKind::header()` at `src/commands/list/columns.rs:23-40`. Key non-obvious mappings:
  - `WorkingDiff → "HEAD±"` (`:28`)
  - `AheadBehind → "main↕"` (`:29`)
  - `BranchDiff → "main…±"` (`:30`)
  - `Upstream → "Remote⇅"` (`:32`)
  - `Time → "Age"` (`:34`)
  - `CiStatus → "CI"` (`:35`)
- `COLUMN_SPECS` static registry at `src/commands/list/columns.rs:97-112` binds each kind to a `base_priority` (lower = more important) plus an optional `requires_task`:
  - `Gutter 0`, `Branch 1` (shrinkable), `Status 2`, `WorkingDiff 3`, `AheadBehind 4`, `CiStatus 5`, `BranchDiff 6`, `Path 7`, `Upstream 8`, `Url 9`, `Summary 10`, `Commit 11`, `Time 12`, `Message 13`.
  - `requires_task`: `BranchDiff → TaskKind::BranchDiff`, `Summary → TaskKind::SummaryGenerate`, `CiStatus → TaskKind::CiStatus`, `Url → TaskKind::UrlStatus`. Everything else is `None`.
- Display order is the array order (`src/commands/list/columns.rs:114-119`, `column_display_index`); priority only governs truncation, not layout position.
- `DiffVariant` (`src/commands/list/columns.rs:57-63`): `Signs` (+X -Y), `Arrows` (↑X ↓Y), `UpstreamArrows` (⇡X ⇣Y).
- Unit tests at `src/commands/list/columns.rs:126-244` pin display order, task gates, unique priorities, non-empty headers, and registry completeness.

---

### 3. Width-aware layout and "N columns hidden"

`src/commands/list/layout.rs` (2082 lines).

- Core allocator: `allocate_columns_with_priority()` at `src/commands/list/layout.rs:662-891`.
- Empty-data penalty: `EMPTY_PENALTY = 10` (`src/commands/list/layout.rs:299`). If a column has no data for the current rows, its effective priority becomes `base_priority + 10`, which reliably demotes it below any populated column. Evaluated via `ColumnKind::has_data()` (`src/commands/list/layout.rs:427-444`), which checks a `ColumnDataFlags` struct. Columns always treated as "has data": `Gutter`, `Branch`, `Time`, `Commit`, `Message`.
- Task-gate filter runs first (`src/commands/list/layout.rs:674-680`):
  ```rust
  .filter(|spec| spec.requires_task.is_none_or(|task| !skip_tasks.contains(&task)))
  ```
  Non-`--full` mode drops `BranchDiff/CiStatus/SummaryGenerate` before any width calculations.
- Fixed per-column widths (`src/commands/list/layout.rs:584-593`):
  - `status_fixed = 8` (full position mask)
  - `working_diff_fixed = 9` (`+999 -999`)
  - `ahead_behind_fixed = 7` (`↑99 ↓99`)
  - `branch_diff_fixed = 9`
  - `upstream_fixed = 7`
  - `age_estimate = 4` (`11mo`)
  - `ci_estimate = 1`
- Flex widths (`src/commands/list/layout.rs:696-702`):
  - `MIN_SUMMARY = 10`, `MAX_SUMMARY = 70`
  - `MIN_MESSAGE = 10`, `MAX_MESSAGE = 100`
  - `SUMMARY_THRESHOLD_FOR_LOW_PRIORITY = 50`
- Post-allocation expansion (`src/commands/list/layout.rs:769-837`): Summary grows first; low-priority Commit/Time/Message are dropped if Summary has not reached 50 (`:791-825`); Message expands after Summary saturates.
- Inter-column spacing: `spacing = 2` (`src/commands/list/layout.rs:670`), with no gap after `Gutter` (`:707-716`, `:852-861`).
- Hidden-column count:
  ```rust
  let hidden_column_count = candidate_kinds
      .iter()
      .filter(|kind| !allocated_kinds.contains(kind))
      .count();
  ```
  at `src/commands/list/layout.rs:873-880`. Candidates are columns that passed the task-gate filter; "hidden" means they were eligible but dropped for width.
- Truncation is left-biased for text (`src/commands/list/render.rs:355-356`, `truncate_to_width`). Diff columns use overflow notation: `100-999` rendered full (Signs), `1000 → "1K"`, `≥10000 → "∞"` (`src/commands/list/render.rs:100-179`).
- Progressive-mode caveat: empty penalties are not applied in the pre-data skeleton pass (`src/commands/list/layout.rs:90-101`); it assumes all columns have data and corrects later.

---

### 4. Footer helper line

`src/commands/list/mod.rs` and `src/commands/list/model/stats.rs`.

- Aggregator struct `SummaryMetrics` at `src/commands/list/mod.rs:201-207`:
  ```rust
  worktrees: usize,
  local_branches: usize,
  remote_branches: usize,
  dirty_worktrees: usize,
  ahead_items: usize,
  ```
- Fold: `SummaryMetrics::from_items(items)` at `src/commands/list/mod.rs:210-244`.
  - Worktree vs branch split via `item.worktree_data().is_some()` (`:218-224`).
  - Dirty test: `status_symbols.working_tree.is_dirty()` (`:224-229`).
  - Remote vs local branch classification by `branch.contains('/')` (`:234-238`).
  - Ahead test: `item.counts.is_some_and(|c| c.ahead > 0)` (`:241-242`).
- Message parts built by `summary_parts(include_branches, hidden_columns)` at `src/commands/list/mod.rs:246-284`. Parts appended in order:
  1. `"N worktree[s]"` (always; pluralized) — `:253-264`.
  2. `"N branches"` if `include_branches && local_branches > 0` — `:255-256`.
  3. `"N remote branches"` if `include_branches && remote_branches > 0` — `:258-259`.
  4. `"N with changes"` if `dirty_worktrees > 0` — `:266-267`.
  5. `"N ahead"` if `ahead_items > 0` — `:270-271`.
  6. `"N column[s] hidden"` if `hidden_columns > 0` — `:274-280`.
- Parts joined with `", "` (`src/commands/list/mod.rs:299`).
- Final formatting `format_summary_message()` at `src/commands/list/mod.rs:287-319`:
  - No errors: `"○ Showing {parts}"`.
  - With errors: appends `". N task[s] failed (M timed out)"`.
  - Entire string styled `Style::new().dimmed()` (`:296`). `INFO_SYMBOL` is `○`.
- Caller site: `src/commands/list/collect/mod.rs:1372-1376` passes `layout.hidden_column_count` into the formatter.
- Example renderings from tests:
  - `src/commands/list/mod.rs:460-470`: `"○ Showing 3 worktrees, 2 with changes, 1 ahead, 2 columns hidden"`.
  - `src/commands/list/mod.rs:442-451`: `"5 worktrees, 3 branches, 8 remote branches, 2 with changes, 4 ahead, 2 columns hidden"`.

---

### 5. Per-worktree data model

All fields live on `ListItem` (`src/commands/list/model/item.rs`, 1101 lines) plus sub-structs in `src/commands/list/model/stats.rs` (read in full, 84 lines).

- Identity: `head: String` (`item.rs:198`), `branch: Option<String>` (`:200`), variant `ItemKind::{Worktree(WorktreeData), Branch}`.
- `WorktreeData`: `path: PathBuf` (`item.rs:105`), `is_main/is_current/is_previous: bool` (`:132-137`), `detached` (`:106`), `locked: Option<String>` (`:108`), `prunable: Option<String>` (`:110`), `branch_worktree_mismatch: bool` (`:142`).
- Main-branch relationship:
  - `counts: Option<AheadBehind { ahead, behind }>` — `item.rs:209`, struct `stats.rs:16-20`. Drives `main↕` column.
  - `branch_diff: Option<BranchDiffTotals { diff: LineDiff }>` — `item.rs:211`, struct `stats.rs:23-27`. Drives `main…±`.
  - Integration signals (internal, not rendered directly): `committed_trees_match` (`:216`), `has_file_changes` (`:221`), `would_merge_add` (`:226`), `is_patch_id_match` (`:230`), `is_ancestor` (`:235`), `is_orphan` (`:239`), `has_merge_tree_conflicts` (`:263`).
- Working tree: `working_tree_diff: Option<LineDiff>` (`item.rs:112`) — drives `HEAD±`. Flags via `WorkingTreeStatus { staged, modified, untracked, renamed, deleted }` at `model/status_symbols.rs:290-294`. Plus `has_conflicts`, `has_working_tree_conflicts`, `git_operation` (`item.rs:120-131`).
- Upstream: `UpstreamStatus { remote: Option<String>, ahead: usize, behind: usize }` at `stats.rs:30-38`, exposed via `ActiveUpstream::active()` (`stats.rs:47-56`). Drives `Remote⇅`.
- Commit: `commit: Option<CommitDetails { timestamp: i64, commit_message: String }>` (`item.rs:202`, struct `stats.rs:9-13`). Drives `Commit`, `Age`, `Message`.
- Summary column data: `summary: Option<Option<String>>` (`item.rs:257`). Outer `None` = task not loaded; `Some(None)` = loaded, nothing to say; `Some(Some(text))` = present. LLM-generated, not derived from commit subjects (see §8).
- CI: `pr_status: Option<Option<PrStatus>>` (`item.rs:246`). Same outer/inner load semantics.
- URL column: `url: Option<String>` and `url_active: Option<bool>` (`item.rs:250-253`).
- User marker: `user_marker: Option<Option<String>>` (`item.rs:268`) — per-branch git-config annotation.
- Status symbols assembled into `StatusSymbols` with six independent "gates" (`item.rs:277`); each resolves as its source field arrives.

---

### 6. Status symbols taxonomy

`src/commands/list/model/status_symbols.rs` (789 lines) and `src/commands/list/model/state.rs` (936 lines).

Six positional gates, rendered left-to-right inside the `Status` column. Placeholder `·` used while a gate's data is still loading.

- Gate 1 — Working tree (positions 0-2), `status_symbols.rs:507-522`, cyan (`:511`):
  - `+` staged, `!` modified, `?` untracked, `»` renamed, `✘` deleted, blank = clean.
- Gate 2 — Worktree state (position 3), priority order `✘ > ⤴ > ⤵ > ⚑ > ⊟ > ⊞ > /` (`:551-566`):
  - `✘` merge conflicts (red, `state.rs:432`), `⤴` rebase, `⤵` merge (yellow, `state.rs:433`), `⚑` branch/worktree path mismatch, `⊟` prunable, `⊞` locked, `/` branch-only row (dimmed, `status_symbols.rs:559`).
- Gate 3 — Main state (position 4), priority `^ > ✗ > _ > – > ⊂ > ∅ > ↕ > ↑ > ↓` (`state.rs:120-130`, `item.rs:386-392`):
  - `^` this IS main, `✗` would conflict on merge to main (yellow, `state.rs:189-191`), `_` same commit + clean, `–` same commit + dirty, `⊂` content integrated (subtypes: `Ancestor`, `TreesMatch`, `NoAddedChanges`, `MergeAddsNothing`, `PatchIdMatch`; `state.rs:6`), `∅` orphan, `↕` diverged, `↑` ahead, `↓` behind.
- Gate 4 — Upstream divergence (position 5), `state.rs:8-58`:
  - `|` in sync, `⇡` ahead, `⇣` behind, `⇅` diverged; dimmed (`state.rs:54`). Blank when no remote (`Divergence::None`, `:52`).
- Gate 5 — User marker (position 6), any string (`status_symbols.rs:569-573`).
- Loading placeholder for any still-None gate: dimmed `·` (`status_symbols.rs:453`).

---

### 7. Data collection pipeline

`src/commands/list/collect/mod.rs` (1769 lines) + `collect/{execution,results,tasks,types}.rs`.

Pre-skeleton batch (parallel, ~60ms median), `rayon::scope` at `collect/mod.rs:518-548`:

- `git worktree list --porcelain` (`:520`)
- `git config worktrunk.default-branch` (`:523`)
- `git config --bool core.bare` (`:527`)
- `git rev-parse --show-toplevel` (`:530`)
- `git for-each-ref refs/heads` if `--branches` (`:540`)
- `git for-each-ref refs/remotes` if `--remotes` (`:545`)
- Batched `git log --no-walk --format='%H %ct'` for every HEAD (`:742`).

Post-skeleton phase (`:1010-1022`): `switch_previous` (~5ms), `integration_target` (~10ms), fsmonitor daemons per worktree (~6ms each, parallel).

Per-worktree tasks dispatched via Rayon `par_iter().for_each()` (`:1140-1147`). Items sorted so local tasks run before network ones (`:1132`); network tasks classified in `collect/types.rs:157-162` (`CiStatus`, `UrlStatus`, `SummaryGenerate`).

Commands per task (from `collect/tasks.rs`):

- `CommitDetailsTask` — `git log -1 --format=%...` (`:130-140`).
- `AheadBehindTask` — batched `git rev-list --count base..head` via `batch_ahead_behind()` (`collect/mod.rs:1063`); primes shared cache.
- `CommittedTreesMatchTask` — two `git rev-parse ^{tree}` calls (`:207-232`).
- `HasFileChangesTask` — `git diff --name-only base...head` (`:251-275`).
- `IsAncestorTask` — `git merge-base --is-ancestor` (`:340-364`).
- `BranchDiffTask` — `git diff --numstat base..head` (`:370-397`).
- `WorkingTreeDiffTask` — `git status --porcelain --no-optional-locks`, cached via `status_porcelain_cached()` (`:418-430`).
- `MergeTreeConflictsTask` — `git merge-tree base head` (`:450-473`).
- `WorkingTreeConflictsTask` — `git write-tree` (+optional `git add -A` on temp index) then `git merge-tree` (`:541-557`). Shares cache with working-tree diff.
- `GitOperationTask` — filesystem probes for `.git/rebase-merge`, `.git/rebase-apply`, `MERGE_HEAD` (`:619`).
- `UserMarkerTask` — `git config` (`:635`).
- `UpstreamTask` — `git rev-parse <branch>@{upstream}` lazily via `Branch::upstream()` OnceCell (`:664-684`).
- `CiStatusTask` — GitHub: `gh pr list --head <branch>` with up to 20 PRs (`ci_status/github.rs:85-99`, `MAX_PRS_TO_FETCH` at `ci_status/mod.rs:95`), plus optional `gh api repos/owner/repo/commits/{sha}/check-runs` (`github.rs:192-206`). GitLab: `glab mr list --source-branch` (`gitlab.rs:89-99`) + `glab mr view <iid>` (`:161`) + optional `glab ci list --ref <branch>` (`:204-214`).
- `UrlStatusTask` — TCP `connect_timeout()` to localhost port, 50ms (`tasks.rs:743-747`).
- `SummaryGenerateTask` — `git diff --no-index` feeding an LLM (`:776-783`).

Skips: unborn branches drop COMMIT_TASKS (`collect/execution.rs:40-51`, `:392`); missing LLM command drops `SummaryGenerate` (`execution.rs:393`); prunable worktrees skip everything and get default seeds (`execution.rs:319-321`); missing URL template skips `UrlStatus` (`collect/mod.rs:835-837`).

Error surfaces: `TaskError { item_idx, kind, message, cause }` at `collect/types.rs:202-230`. `ErrorCause::Timeout` detected by walking the IO error chain (`tasks.rs:60-78`). Errored fields stay `None`; `refresh_status_symbols()` still runs and renders `·` for unresolved gates (`collect/results.rs:255-261`, `:369`).

Concurrency knob: Rayon default (≈2× CPU cores); override via `RAYON_NUM_THREADS`.

---

### 8. Summary column source

`src/summary.rs`.

- Inputs: `git diff main...HEAD --stat` + `git diff main...HEAD` (branch diff) plus worktree diff `git -C <path> diff HEAD --stat` and `git -C <path> diff HEAD` (`src/summary.rs:147-170`).
- Hash: `hash_diff()` (`src/summary.rs:122-126`) keys the cache.
- Cache: `.git/wt/cache/summaries/<sanitized-branch>.json` (`src/summary.rs:70-77`).
- Generation: external LLM command invoked with template rendering (`src/summary.rs:180-187`; template `SUMMARY_TEMPLATE:51-66`).
- Windows exclusion: `#[cfg_attr(windows, allow(dead_code))]` on `generate_summary()` at `src/summary.rs:247`.
- Not commit-subject based; entirely generated from diffs.

---

### 9. Progressive rendering

`src/commands/list/progressive_table.rs` (608 lines), `src/commands/list/progressive.rs` (53 lines), `src/commands/list/render.rs` (1464 lines), `src/commands/list/collect/results.rs`.

- Mode selection at `progressive.rs:25-32`: TTY → `Progressive`, else `Buffered`. Comment at `progressive.rs:31-32` flags missing pager env-var detection as a TODO.
- Placeholder glyphs (`render.rs:14-28`): `PLACEHOLDER_BLANK = " "` for the first ~200ms, then `PLACEHOLDER = "·"`. The TODO at `render.rs:14-20` notes the intended original design used `⋯` for timed-out vs `·` for loading but was collapsed because `⋯` was visually too loud.
- Reveal deadline constant: `PLACEHOLDER_REVEAL_DELAY` at `collect/mod.rs:972`, overridable by `WORKTRUNK_PLACEHOLDER_REVEAL_MS`.
- Skeleton emit order: blank placeholders first (`collect/mod.rs:911`), then parallel work items are enqueued.
- Result channel: `chan::unbounded()` at `collect/mod.rs:1070`. Drain loop ticks every 500ms (`collect/results.rs:215`, `STALL_TIMINGS.tick` at `:30`). Stall threshold 5 s (`:29`); stall footer formatted at `collect/results.rs:439-459`.
- Per-result flow: `refresh_status_symbols()` at `collect/results.rs:369` (recomputes affected gates), row cache dedup at `:399-406`, then `update_footer(content)` at `progressive_table.rs:174-191` and row update at `:253-287` via `finalize()`.
- Footer is the final line of the progressive structure (`progressive_table.rs:98-102`).
- Placeholder swap uses interior mutability: `pub placeholder: std::cell::Cell<&'static str>` at `layout.rs:524`. Updated in place without `&mut LayoutConfig`.

---

### 10. ANSI styling inventory

`src/commands/list/render.rs` plus state enums.

- Time column dim: `render.rs:481`.
- Commit column dim: `render.rs:532`.
- Message column dim: `render.rs:551`.
- Diff values: `ADDITION` (green) vs `DELETION` (red) styles at `render.rs:118-179`; `.dimmed()` for `Upstream` dim rendering; overflow tokens (`K`/`C`/`∞`) bolded at `render.rs:108`.
- Upstream "in-sync" `|`: dim at `render.rs:464` (duplicates the `InSync` check on `Divergence::Special`, comment at `:462-463` acknowledges the duplication).
- URL cell: plain if active, dim if inactive (`render.rs:495-501`); hyperlink support at `:563-571` uses `hyperlink_stdout()` and falls back to full URL when unsupported.
- Symbol colors: `+!?»✘` cyan (`status_symbols.rs:511`), operation/state glyphs via `.styled()` methods in `state.rs:62, 189-191, 432-433`.
- Footer line dim: `Style::new().dimmed()` at `mod.rs:296`.

---

## Anomalies

- **Placeholder glyph collapsed** (`render.rs:14-20`). TODO says loading vs timed-out were originally distinct; both now render `·` because `⋯` was too loud. Side-by-side reevaluation is queued.
- **Pager detection stub** (`progressive.rs:31-32`). Mode detection only inspects TTY; `PAGER`/`LESS` env vars noted as missing.
- **Duplicate "in sync" upstream logic** (`render.rs:462-463` vs `upstream.rs`). Code carries two paths for the same decision.
- **`WorkingTreeConflicts` vs `MergeTreeConflicts` duplication** (`collect/execution.rs:381-385`). TODO asks to skip the commit probe when working-tree conflict probe already resolved. Currently both always run in parallel.
- **Stale default-branch warning** (`collect/mod.rs:630`). Persisted value is reused on hot path without validation.
- **Progressive empty-penalty blind spot** (`layout.rs:90-101`). Skeleton layout assumes all columns have data; columns that arrive empty still occupied width in the skeleton pass.
- **Interior mutability for placeholders** (`layout.rs:524`). `Cell<&'static str>` chosen specifically to swap glyph at the reveal deadline without `&mut` on config.
- **JSON field presence** (`item.rs:204-207, 241-243`). Comments flag inconsistency: `counts`, `branch_diff`, and upstream fields are omitted when `None`, yet JSON consumers may expect a stable schema.
- **Asymmetric digit budgets for diff columns** (`layout.rs:627-648`). `WorkingDiff`/`BranchDiff` reserve 3 digits each side (large line deltas); `AheadBehind`/`Upstream` reserve only 2 (commit counts cap at ~99 before switching to K/C/∞ overflow).
- **Recent churn on list subtree is minimal** — latest touch on `summary.rs` is a CRLF fix (commit `e6c50fb`). The rest of the list module has been stable for some time.
- **Ctx overlap**: `CiStatus` has its own cache (`ci_status/cache.rs`, 30-60 s TTL with hash-based jitter at `:39-58`), disjoint from diff/ahead-behind caches; cache key is the branch `full_name` including remote prefix (`cache.rs:332`).

---

## Open questions

- No question raised here becomes a recommendation; these are observations that would need user input to resolve if a port were designed.
  1. Does the target app (`worktree-switcher`, Go, `main.go:127 runList`) need the LLM-summary column at all, or only the cheap columns (Status, HEAD±, main↕, main…±, Remote⇅, Path, Age, Commit, Message)? The three `--full`-gated tasks in worktrunk (`BranchDiff`, `CiStatus`, `SummaryGenerate`) are the only ones requiring external processes or models.
  2. Worktrunk assumes an "integration target" branch (configurable via `worktrunk.default-branch`). The Go app currently uses `detectMainBranch()` which picks the first of `main`/`master` that exists (`git.go:207-214`). Parity for the main↕/main…± columns depends on which rule wins.
  3. Worktrunk's footer counts local vs remote branches separately (`mod.rs:218-244`); `worktree-switcher`'s current `runList` does not render branches at all (`main.go:127-155`). Whether to add branch rows is a scope choice.
  4. The progressive skeleton + 200ms placeholder reveal assumes a TUI capable of repainting; `runList` in the Go app writes once to stdout. Parity requires either a Bubble Tea model or a simpler "collect-then-print" path.
  5. CI status needs `gh`/`glab`. The Go app has no current dependency on either — scope question whether to ship CI support or omit.
  6. `worktrunk.list.full` can be set in config to default on (`collect/mod.rs:596`). Target app currently has no config file for list behavior — config precedence is undecided.

---

## Key files index (for later reference)

- `src/cli/mod.rs:383-385` — `--full` flag
- `src/commands/list/mod.rs:151-319` — handler entry, summary message
- `src/commands/list/columns.rs` — column registry (read in full)
- `src/commands/list/layout.rs:662-891` — layout allocator with priority + empty penalty
- `src/commands/list/render.rs` — cell rendering, colors, placeholders
- `src/commands/list/progressive_table.rs` — skeleton + row/footer updates
- `src/commands/list/model/item.rs` — `ListItem`, `WorktreeData`
- `src/commands/list/model/stats.rs` — `AheadBehind`, `BranchDiffTotals`, `UpstreamStatus`, `CommitDetails`
- `src/commands/list/model/status_symbols.rs` — six gates, symbol taxonomy
- `src/commands/list/model/state.rs` — enums + color styling
- `src/commands/list/collect/mod.rs` — pipeline orchestration
- `src/commands/list/collect/tasks.rs` — per-task git commands
- `src/commands/list/collect/execution.rs` — work-item generation + skips
- `src/commands/list/collect/results.rs` — drain loop, stall footer, dedup
- `src/commands/list/collect/types.rs` — task/error types, `is_network()`
- `src/commands/list/ci_status/{mod,cache,github,gitlab,platform}.rs` — CI integration
- `src/summary.rs` — LLM summary generation + cache
