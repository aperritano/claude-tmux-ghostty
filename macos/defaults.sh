#!/bin/bash
# macOS system preferences
# Run once on a fresh Mac, then log out / restart for all changes to take effect.
set -euo pipefail

echo "Applying macOS defaults..."

# ── Keyboard ──────────────────────────────────────────────
# Fast key repeat and short delay
defaults write NSGlobalDomain KeyRepeat -int 2
defaults write NSGlobalDomain InitialKeyRepeat -int 15

# Disable press-and-hold for special characters (enables key repeat everywhere)
defaults write NSGlobalDomain ApplePressAndHoldEnabled -bool false

# Disable auto-correct and smart features that interfere with coding
defaults write NSGlobalDomain NSAutomaticSpellingCorrectionEnabled -bool false
defaults write NSGlobalDomain NSAutomaticCapitalizationEnabled -bool false
defaults write NSGlobalDomain NSAutomaticDashSubstitutionEnabled -bool false
defaults write NSGlobalDomain NSAutomaticQuoteSubstitutionEnabled -bool false
defaults write NSGlobalDomain NSAutomaticPeriodSubstitutionEnabled -bool false

# ── Dock ──────────────────────────────────────────────────
# Auto-hide the Dock
defaults write com.apple.dock autohide -bool true

# Remove the auto-hide delay
defaults write com.apple.dock autohide-delay -float 0

# Speed up the animation when hiding/showing the Dock
defaults write com.apple.dock autohide-time-modifier -float 0.3

# Minimize windows into their application icon
defaults write com.apple.dock minimize-to-application -bool true

# Don't show recent applications in Dock
defaults write com.apple.dock show-recents -bool false

# Set icon size
defaults write com.apple.dock tilesize -int 48

# ── Finder ────────────────────────────────────────────────
# Show all file extensions
defaults write NSGlobalDomain AppleShowAllExtensions -bool true

# Show hidden files
defaults write com.apple.finder AppleShowAllFiles -bool true

# Show path bar at bottom
defaults write com.apple.finder ShowPathbar -bool true

# Show status bar
defaults write com.apple.finder ShowStatusBar -bool true

# Default to list view
defaults write com.apple.finder FXPreferredViewStyle -string "Nlsv"

# Keep folders on top when sorting by name
defaults write com.apple.finder _FXSortFoldersFirst -bool true

# Disable the warning when changing a file extension
defaults write com.apple.finder FXEnableExtensionChangeWarning -bool false

# Avoid creating .DS_Store files on network or USB volumes
defaults write com.apple.desktopservices DSDontWriteNetworkStores -bool true
defaults write com.apple.desktopservices DSDontWriteUSBStores -bool true

# ── Screenshots ───────────────────────────────────────────
# Save screenshots to ~/Screenshots
mkdir -p "$HOME/Screenshots"
defaults write com.apple.screencapture location -string "$HOME/Screenshots"

# Save screenshots as PNG
defaults write com.apple.screencapture type -string "png"

# Disable shadow in screenshots
defaults write com.apple.screencapture disable-shadow -bool true

# ── Trackpad ──────────────────────────────────────────────
# Enable tap to click
defaults write com.apple.driver.AppleBluetoothMultitouch.trackpad Clicking -bool true
defaults -currentHost write NSGlobalDomain com.apple.mouse.tapBehavior -int 1

# ── Misc ──────────────────────────────────────────────────
# Expand save and print panels by default
defaults write NSGlobalDomain NSNavPanelExpandedStateForSaveMode -bool true
defaults write NSGlobalDomain NSNavPanelExpandedStateForSaveMode2 -bool true
defaults write NSGlobalDomain PMPrintingExpandedStateForPrint -bool true
defaults write NSGlobalDomain PMPrintingExpandedStateForPrint2 -bool true

# Disable the "Are you sure you want to open this application?" dialog
defaults write com.apple.LaunchServices LSQuarantine -bool false

# Restart affected apps
echo "Restarting Dock and Finder..."
killall Dock
killall Finder

echo "Done. Some changes require a logout/restart to take effect."
