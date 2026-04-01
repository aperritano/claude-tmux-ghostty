#!/bin/bash
# Symlink dotfiles into place
set -euo pipefail

DOTFILES="$(cd "$(dirname "$0")" && pwd)"

link() {
  local src="$1" dst="$2"
  if [ -L "$dst" ]; then
    rm "$dst"
  elif [ -e "$dst" ]; then
    echo "  backup: $dst -> ${dst}.bak"
    mv "$dst" "${dst}.bak"
  fi
  mkdir -p "$(dirname "$dst")"
  ln -s "$src" "$dst"
  echo "  linked: $dst -> $src"
}

echo "Installing dotfiles from $DOTFILES"
echo

echo "Ghostty:"
link "$DOTFILES/ghostty/config" "$HOME/.config/ghostty/config"
link "$DOTFILES/ghostty/themes/Green CRT" "$HOME/.config/ghostty/themes/Green CRT"
link "$DOTFILES/ghostty/themes/crt-green.bak" "$HOME/.config/ghostty/themes/crt-green.bak"
echo

echo "tmux:"
link "$DOTFILES/tmux/tmux.conf" "$HOME/.tmux.conf"
echo

echo "Helper scripts (~/bin):"
mkdir -p "$HOME/bin"
for script in "$DOTFILES"/bin/tmux-*; do
  link "$script" "$HOME/bin/$(basename "$script")"
done
echo

echo "Done. Reload tmux config with: tmux source ~/.tmux.conf"
