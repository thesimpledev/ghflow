package tui

import "github.com/charmbracelet/lipgloss"

var (
	// Colors
	primaryColor   = lipgloss.Color("62")  // Purple
	secondaryColor = lipgloss.Color("241") // Gray
	successColor   = lipgloss.Color("42")  // Green
	failureColor   = lipgloss.Color("196") // Red
	warningColor   = lipgloss.Color("214") // Orange
	pendingColor   = lipgloss.Color("247") // Light gray

	// Status styles
	SuccessStyle = lipgloss.NewStyle().Foreground(successColor)
	FailureStyle = lipgloss.NewStyle().Foreground(failureColor)
	PendingStyle = lipgloss.NewStyle().Foreground(pendingColor)
	RunningStyle = lipgloss.NewStyle().Foreground(warningColor)

	// Layout styles
	TitleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(primaryColor).
			MarginBottom(1)

	SubtitleStyle = lipgloss.NewStyle().
			Foreground(secondaryColor)

	HelpStyle = lipgloss.NewStyle().
			Foreground(secondaryColor).
			MarginTop(1)

	SelectedStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(primaryColor)

	NormalStyle = lipgloss.NewStyle()

	// Item styles
	RepoNameStyle = lipgloss.NewStyle().
			Bold(true)

	BranchStyle = lipgloss.NewStyle().
			Foreground(secondaryColor)

	DimStyle = lipgloss.NewStyle().
			Foreground(secondaryColor)

	// Box styles
	BoxStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(secondaryColor).
			Padding(0, 1)
)

// Status indicators
const (
	StatusIconSuccess = "[ok]"
	StatusIconFailure = "[X]"
	StatusIconRunning = "[~]"
	StatusIconPending = "[?]"
	StatusIconUnknown = "[-]"
)
