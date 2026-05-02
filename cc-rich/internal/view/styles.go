// cc-rich/internal/view/styles.go
// Lipgloss color tokens and styles used across all view models.
// Anchored to terminal-256 indexes that match the rest of the dotfiles
// palette (statusline, pane-border, etc.).
package view

import "github.com/charmbracelet/lipgloss"

var (
	ColorAccent   = lipgloss.Color("13") // magenta — active border
	ColorMuted    = lipgloss.Color("8")  // grey
	ColorWarn     = lipgloss.Color("11") // amber
	ColorOK       = lipgloss.Color("10") // green
	ColorBranch   = lipgloss.Color("3")  // yellow
	ColorWorktree = lipgloss.Color("14") // cyan

	StyleHeader = lipgloss.NewStyle().Foreground(ColorAccent).Bold(true)
	StyleMuted  = lipgloss.NewStyle().Foreground(ColorMuted)
	StyleUser   = lipgloss.NewStyle().Foreground(ColorBranch).Bold(true)
	StyleAsst   = lipgloss.NewStyle().Foreground(ColorWorktree).Bold(true)
	StyleButton = lipgloss.NewStyle().Background(ColorAccent).Foreground(lipgloss.Color("0")).Padding(0, 1)
)
