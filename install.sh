#!/bin/bash
# Bootstrap a fresh macOS dev environment from this dotfiles repo.
# Safe to re-run — existing files are backed up to *.bak before symlinking.
set -euo pipefail

DOTFILES="$(cd "$(dirname "$0")" && pwd)"

# ── Helpers ───────────────────────────────────────────────

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

section() {
  echo
  echo "━━━ $1 ━━━"
}

command_exists() {
  command -v "$1" &>/dev/null
}

# ── Homebrew ──────────────────────────────────────────────

section "Homebrew"
if command_exists brew; then
  echo "  already installed"
else
  echo "  installing Homebrew..."
  /bin/bash -c "$(curl -fsSL https://raw.githubusercontent.com/Homebrew/install/HEAD/install.sh)"
  # Add to PATH for the rest of this script (zshrc handles it at login)
  eval "$(/opt/homebrew/bin/brew shellenv)"
fi

section "Brew packages"
echo "  running brew bundle..."
brew bundle --file="$DOTFILES/Brewfile"

# ── Rust toolchain (needed for claude-tmux) ───────────────

section "Rust toolchain"
if command_exists cargo; then
  echo "  Rust toolchain already installed"
else
  echo "  initializing Rust toolchain via rustup..."
  rustup-init -y --no-modify-path
  source "$HOME/.cargo/env"
fi

# ── Oh My Zsh + plugins ──────────────────────────────────

section "Oh My Zsh"
if [ -d "$HOME/.oh-my-zsh" ]; then
  echo "  already installed"
else
  echo "  installing Oh My Zsh..."
  RUNZSH=no KEEP_ZSHRC=yes sh -c "$(curl -fsSL https://raw.github.com/ohmyzsh/ohmyzsh/master/tools/install.sh)"
fi

ZSH_CUSTOM="${ZSH_CUSTOM:-$HOME/.oh-my-zsh/custom}"

section "Zsh plugins & theme"
plugins=(
  "themes/powerlevel10k|https://github.com/romkatv/powerlevel10k.git"
  "plugins/zsh-autosuggestions|https://github.com/zsh-users/zsh-autosuggestions.git"
  "plugins/zsh-syntax-highlighting|https://github.com/zsh-users/zsh-syntax-highlighting.git"
)
for entry in "${plugins[@]}"; do
  dest="$ZSH_CUSTOM/${entry%%|*}"
  repo="${entry##*|}"
  name="$(basename "$dest")"
  if [ -d "$dest" ]; then
    echo "  $name: already installed"
  else
    echo "  $name: cloning..."
    git clone --depth=1 "$repo" "$dest"
  fi
done

# ── Symlink configs ──────────────────────────────────────

section "Zsh"
link "$DOTFILES/zsh/zshrc" "$HOME/.zshrc"
link "$DOTFILES/zsh/p10k.zsh" "$HOME/.p10k.zsh"

section "Vim"
link "$DOTFILES/vim/vimrc" "$HOME/.vimrc"

section "Git"
link "$DOTFILES/git/gitconfig" "$HOME/.gitconfig"
link "$DOTFILES/git/gitignore_global" "$HOME/.gitignore_global"

section "Ghostty"
link "$DOTFILES/ghostty/config" "$HOME/.config/ghostty/config"
link "$DOTFILES/ghostty/themes/Green CRT" "$HOME/.config/ghostty/themes/Green CRT"
link "$DOTFILES/ghostty/themes/claude-quiet" "$HOME/.config/ghostty/themes/claude-quiet"

section "iTerm color presets"
mkdir -p "$HOME/.config/iterm"
for theme in "$DOTFILES"/iterm/*.itermcolors; do
  [ -f "$theme" ] && link "$theme" "$HOME/.config/iterm/$(basename "$theme")"
done

section "tmux"
link "$DOTFILES/tmux/tmux.conf" "$HOME/.tmux.conf"

section "Helper scripts (~/bin)"
mkdir -p "$HOME/bin"
for script in "$DOTFILES"/bin/*; do
  link "$script" "$HOME/bin/$(basename "$script")"
done

section "Claude Code"
mkdir -p "$HOME/.claude/rules" "$HOME/.claude/commands"
link "$DOTFILES/claude/statusline-command.sh" "$HOME/.claude/statusline-command.sh"
link "$DOTFILES/claude/settings.json" "$HOME/.claude/settings.json"
link "$DOTFILES/claude/settings.local.json" "$HOME/.claude/settings.local.json"
link "$DOTFILES/claude/CLAUDE.md" "$HOME/.claude/CLAUDE.md"
link "$DOTFILES/claude/rules/deterministic-execution-protocol.md" "$HOME/.claude/rules/deterministic-execution-protocol.md"
for cmd in "$DOTFILES"/claude/commands/*.md; do
  [ -f "$cmd" ] && link "$cmd" "$HOME/.claude/commands/$(basename "$cmd")"
done

# ── Vim plugins ──────────────────────────────────────────

section "Vim plugins"
PLUG_FILE="$HOME/.vim/autoload/plug.vim"
if [ -f "$PLUG_FILE" ]; then
  echo "  vim-plug already installed"
else
  echo "  installing vim-plug..."
  curl -fLo "$PLUG_FILE" --create-dirs \
    https://raw.githubusercontent.com/junegunn/vim-plug/master/plug.vim
fi
echo "  installing vim plugins..."
vim +PlugInstall +qall 2>/dev/null || echo "  (run 'vim +PlugInstall' manually if this failed)"

# ── Node.js via nvm ──────────────────────────────────────

section "Node.js"
export NVM_DIR="$HOME/.nvm"
mkdir -p "$NVM_DIR"
[ -s "/opt/homebrew/opt/nvm/nvm.sh" ] && \. "/opt/homebrew/opt/nvm/nvm.sh"
if command_exists nvm; then
  if nvm ls --no-colors 2>/dev/null | grep -q "node"; then
    echo "  Node.js already installed via nvm"
  else
    echo "  installing latest LTS Node.js..."
    nvm install --lts
  fi
else
  echo "  nvm not available — install Node.js manually after restart"
fi

# ── Claude Code CLI ──────────────────────────────────────

section "Claude Code"
if command_exists claude; then
  echo "  Claude Code CLI already installed"
else
  echo "  installing Claude Code CLI..."
  npm install -g @anthropic-ai/claude-code || echo "  (install Node.js first, then: npm i -g @anthropic-ai/claude-code)"
fi

# ── claude-tmux ──────────────────────────────────────────

section "claude-tmux"
if command_exists claude-tmux; then
  echo "  claude-tmux already installed"
else
  echo "  installing claude-tmux..."
  cargo install claude-tmux || echo "  (run 'cargo install claude-tmux' manually if this failed)"
fi

# ── .env template ────────────────────────────────────────

section "Environment"
if [ -f "$HOME/.env" ]; then
  echo "  ~/.env already exists"
else
  cp "$DOTFILES/.env.template" "$HOME/.env"
  echo "  created ~/.env from template — edit it to add your API keys"
fi

# ── Summary ──────────────────────────────────────────────

section "Done"
cat <<'MSG'
Remaining manual steps:
  1. Restart your terminal (or run: source ~/.zshrc)
  2. Run: gh auth login          (authenticate GitHub CLI)
  3. Edit ~/.env                  (add API keys)
  4. Optional: ./macos/defaults.sh  (set macOS system preferences)
  5. Optional: tmux-tutorial      (learn the keybindings)
MSG
