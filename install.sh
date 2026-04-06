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

echo "Zsh:"
link "$DOTFILES/zsh/zshrc" "$HOME/.zshrc"
echo

echo "Vim:"
link "$DOTFILES/vim/vimrc" "$HOME/.vimrc"
echo

echo "Ghostty:"
link "$DOTFILES/ghostty/config" "$HOME/.config/ghostty/config"
link "$DOTFILES/ghostty/themes/Green CRT" "$HOME/.config/ghostty/themes/Green CRT"
echo

echo "tmux:"
link "$DOTFILES/tmux/tmux.conf" "$HOME/.tmux.conf"
echo

echo "Helper scripts (~/bin):"
mkdir -p "$HOME/bin"
for script in "$DOTFILES"/bin/*; do
  link "$script" "$HOME/bin/$(basename "$script")"
done
echo

echo "Claude Code:"
mkdir -p "$HOME/.claude/rules"
link "$DOTFILES/claude/statusline-command.sh" "$HOME/.claude/statusline-command.sh"
link "$DOTFILES/claude/settings.json" "$HOME/.claude/settings.json"
link "$DOTFILES/claude/settings.local.json" "$HOME/.claude/settings.local.json"
link "$DOTFILES/claude/CLAUDE.md" "$HOME/.claude/CLAUDE.md"
link "$DOTFILES/claude/rules/deterministic-execution-protocol.md" "$HOME/.claude/rules/deterministic-execution-protocol.md"
echo

echo "Done. Reload tmux config with: tmux source ~/.tmux.conf"
