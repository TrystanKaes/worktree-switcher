#!/usr/bin/env bash
# install.sh — configure shell integration for worktree-switcher
set -euo pipefail

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
    echo "Unsupported shell: $shell_name (supported: bash, zsh)"
    echo "Add this to your shell config manually:"
    echo '  eval "$(wt-bin init)"'
    exit 0
    ;;
esac

init_marker='wt-bin init'

if grep -qF "$init_marker" "$profile" 2>/dev/null; then
  echo "Shell already configured in $profile"
else
  printf '\n# worktree-switcher\neval "$(wt-bin init)"\n' >> "$profile"
  echo "Added shell integration to $profile"
fi
