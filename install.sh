#!/usr/bin/env bash
# install.sh — install worktree-switcher and configure shell integration for worktree-switcher
set -euo pipefail

BINARY="${1:-./worktree-switcher}"
INSTALL_DIR="${INSTALL_DIR:-$HOME/.local/bin}"

# ── 1. Install binary ────────────────────────────────────────────────────────

if [[ ! -f "$BINARY" ]]; then
  echo "error: binary not found at '$BINARY'" >&2
  echo "Run 'make build' first, or pass the binary path as the first argument." >&2
  exit 1
fi

mkdir -p "$INSTALL_DIR"
cp "$BINARY" "$INSTALL_DIR/worktree-switcher"
chmod +x "$INSTALL_DIR/worktree-switcher"
echo "Installed worktree-switcher to $INSTALL_DIR/worktree-switcher"

# ── 2. Detect shell profile ──────────────────────────────────────────────────

shell_name="$(basename "$SHELL")"

case "$shell_name" in
  bash)
    if [[ -f "$HOME/.bash_profile" ]]; then
      profile="$HOME/.bash_profile"
    elif [[ -f "$HOME/.bashrc" ]]; then
      profile="$HOME/.bashrc"
    else
      profile="$HOME/.bashrc"
      touch "$profile"
    fi
    ;;
  zsh)
    profile="$HOME/.zshrc"
    if [[ ! -f "$profile" ]]; then
      touch "$profile"
    fi
    ;;
  *)
    echo ""
    echo "Unsupported shell: $shell_name (supported: bash, zsh)"
    echo "Add this to your shell config manually:"
    echo ""
    echo '  export PATH="'"$INSTALL_DIR"':$PATH"'
    echo '  eval "$(worktree-switcher init)"'
    exit 0
    ;;
esac

# ── 3. Ensure INSTALL_DIR is on PATH ────────────────────────────────────────

path_export="export PATH=\"$INSTALL_DIR:\$PATH\""
path_added=false

# Check both the current runtime PATH and the profile file
if ! echo ":$PATH:" | grep -qF ":$INSTALL_DIR:"; then
  if ! grep -qF "$INSTALL_DIR" "$profile" 2>/dev/null; then
    printf '\n# worktree-switcher: add worktree-switcher to PATH\n%s\n' "$path_export" >> "$profile"
    echo "Added PATH entry to $profile"
    path_added=true
  fi
fi

# ── 4. Add eval line (idempotent) ────────────────────────────────────────────

init_marker='worktree-switcher init'

if grep -qF "$init_marker" "$profile" 2>/dev/null; then
  echo "Shell integration already present in $profile"
else
  printf '\n# worktree-switcher\neval "$(worktree-switcher init)"\n' >> "$profile"
  echo "Added shell integration to $profile"
fi

# ── 5. Summary ───────────────────────────────────────────────────────────────

echo ""
echo "Done. To activate, restart your shell or run:"
echo "  source $profile"

if [[ "$path_added" == true ]]; then
  echo ""
  echo "Note: $INSTALL_DIR was added to your PATH in $profile."
  echo "This takes effect in new shell sessions."
fi
