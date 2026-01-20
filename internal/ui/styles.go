package ui

import "github.com/charmbracelet/lipgloss"

var (
	subtle    = lipgloss.AdaptiveColor{Light: "#D9DCCF", Dark: "#383838"}
	highlight = lipgloss.AdaptiveColor{Light: "#874BFD", Dark: "#7D56F4"}
	special   = lipgloss.AdaptiveColor{Light: "#43BF6D", Dark: "#73F59F"}
	text      = lipgloss.AdaptiveColor{Light: "#1a1a1a", Dark: "#fafafa"}
	muted     = lipgloss.AdaptiveColor{Light: "#666666", Dark: "#888888"}

	HeaderStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(highlight).
			Padding(0, 2).
			Border(lipgloss.RoundedBorder()).
			BorderForeground(subtle).
			MarginBottom(1)

	PanelStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(subtle).
			Padding(1, 2)

	ActivePanelStyle = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(highlight).
				Padding(1, 2)

	TitleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(text).
			MarginBottom(1)

	SelectedStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(special)

	MutedStyle = lipgloss.NewStyle().
			Foreground(muted)

	StatusBarStyle = lipgloss.NewStyle().
			Foreground(muted).
			Padding(0, 1).
			MarginTop(1)

	LabelStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(highlight)

	TagStyle = lipgloss.NewStyle().
			Foreground(special).
			Padding(0, 1).
			Background(lipgloss.Color("#1a1a2e")).
			Bold(true)

	ErrorStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FF0000")).
			Bold(true)

	DialogStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(highlight).
			Padding(2, 3).
			Align(lipgloss.Center)

	ButtonStyle = lipgloss.NewStyle().
			Foreground(text).
			Background(subtle).
			Padding(0, 2).
			Margin(0, 1)

	ActiveButtonStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#FFFFFF")).
				Background(highlight).
				Padding(0, 2).
				Margin(0, 1)

	// List & key hint styling (for a more "lazygit-like" look)
	ListItemStyle = lipgloss.NewStyle().
			Padding(0, 1)

	SelectedListItemStyle = ListItemStyle.Copy().
				Background(highlight).
				Foreground(lipgloss.Color("#000000"))

	KeyStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(highlight)

	KeyHintStyle = lipgloss.NewStyle().
			Foreground(muted)
)

const (
	FolderIcon = "📁"
	NoteIcon   = "📝"
)
