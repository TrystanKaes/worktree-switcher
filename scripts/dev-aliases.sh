#!/usr/bin/env bash
# Prints dev shell integration to stdout by rewriting the binary's own init output.
# Usage: source <(make dev-aliases)
#    or: source <(bash scripts/dev-aliases.sh)

# Clear any stale aliases with these names (from older versions of this script
# that wrote aliases directly into shell profiles). Without this, alias expansion
# inside function bodies causes "syntax error near `)'" at source time.
echo 'unalias wt-dev wti-dev wts-dev 2>/dev/null || true'

worktree-switcher-dev init | perl -pe '
  s/worktree-switcher/worktree-switcher-dev/g;
  s/_wt_completions/_wt_dev_completions/g;
  s/\bwti\b/wti-dev/g;
  s/\bwts\b/wts-dev/g;
  s/\b_wt\b/_wt_dev/g;
  s/\bwt\b/wt-dev/g;
  s/^([\w-]+)\(\)/function $1/;
'
