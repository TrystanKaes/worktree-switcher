# Plan: Homebrew Formula via Custom Tap + GoReleaser

**Plan file:** plans/plan-homebrew-formula.md  
**Checkpointed at:** 2026-04-13T00:00:00Z  
**Started:** 2026-04-13T00:00:00Z  
**Current:** Step 8 (manual)

## Resume from

**All automated steps complete.** Steps 5 and 8 are manual — user must perform them.

### Remaining manual steps

**Step 5:** Create `TrystanKaes/homebrew-Tools` tap repo on GitHub with placeholder formula at `Formula/worktree-switcher.rb` (see plan for skeleton). Can use:
```sh
gh repo create TrystanKaes/homebrew-Tools --public
```
Then add the placeholder formula.

**Step 8:** Commit all changes, tag v0.1.0, push tag:
```sh
git add .
git commit -m "Add GoReleaser, GitHub Actions release workflow, Homebrew tap setup"
git tag v0.1.0
git push && git push --tags
```
GitHub Actions runs GoReleaser automatically on tag push.

**Required secret:** Add `HOMEBREW_TAP_TOKEN` (PAT with `repo` scope on `homebrew-Tools`) to the `worktree-switcher` repo secrets in GitHub settings.

### Files touched since last checkpoint

- `.github/workflows/release.yml` — Created. Triggers on `v*` tags. Uses goreleaser-action v6. Passes GITHUB_TOKEN + HOMEBREW_TAP_TOKEN.
- `README.md` — Added Homebrew install section before git clone method.
- `.gitignore` — Added `plans/`.

## Checklist

- [x] **Step 1:** Add version flag to the binary *(2026-04-13)*
  - Files: `main.go`
- [x] **Step 2:** Wire version into Makefile ldflags *(2026-04-13)*
  - Files: `Makefile`
- [x] **Step 3:** Create .goreleaser.yaml *(2026-04-13)*
  - Files: `.goreleaser.yaml`
- [x] **Step 4:** Create GitHub Actions release workflow *(2026-04-13)*
  - Files: `.github/workflows/release.yml`
- [x] **Step 5:** Create the tap repo *(2026-04-13)*
  - `Formula/worktree-switcher.rb` placeholder committed to `TrystanKaes/homebrew-Tools` via gh API
- [x] **Step 6:** Update README *(2026-04-13)*
  - Files: `README.md`
- [x] **Step 7:** Update .gitignore *(2026-04-13)*
  - Files: `.gitignore`
- [ ] **Step 8:** Tag and release *(manual step — user action required)*

## Last gate

**Timestamp:** 2026-04-13  
YAML structure check on `.github/workflows/release.yml` → **PASS**  
`plans/` gitignored → **PASS**

## Counters

- Files read since last checkpoint: 4
- Steps completed since last checkpoint: 4

## Conflicts

None.
