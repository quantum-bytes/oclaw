package ui

import "github.com/charmbracelet/lipgloss"

var (
	// Colors
	accentColor    = lipgloss.Color("#7C3AED") // Purple
	dimColor       = lipgloss.Color("#6B7280") // Gray
	successColor   = lipgloss.Color("#10B981") // Green
	errorColor     = lipgloss.Color("#EF4444") // Red
	warningColor   = lipgloss.Color("#F59E0B") // Amber
	userColor      = lipgloss.Color("#3B82F6") // Blue
	assistantColor = lipgloss.Color("#8B5CF6") // Violet
	thinkingColor  = lipgloss.Color("#6B7280") // Gray

	// Header
	headerStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#FFFFFF")).
			Background(accentColor).
			Padding(0, 1)

	headerAgentStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#E5E7EB")).
				Background(accentColor).
				Padding(0, 1)

	// Status bar
	statusBarStyle = lipgloss.NewStyle().
			Foreground(dimColor).
			Padding(0, 1)

	statusConnectedStyle = lipgloss.NewStyle().
				Foreground(successColor).
				Bold(true)

	statusDisconnectedStyle = lipgloss.NewStyle().
				Foreground(errorColor).
				Bold(true)

	statusReconnectingStyle = lipgloss.NewStyle().
				Foreground(warningColor)

	// Chat messages
	userLabelStyle = lipgloss.NewStyle().
			Foreground(userColor).
			Bold(true)

	assistantLabelStyle = lipgloss.NewStyle().
				Foreground(assistantColor).
				Bold(true)

	thinkingLabelStyle = lipgloss.NewStyle().
				Foreground(thinkingColor).
				Italic(true)

	thinkingTextStyle = lipgloss.NewStyle().
				Foreground(thinkingColor).
				Italic(true)

	// Input
	inputPromptStyle = lipgloss.NewStyle().
				Foreground(accentColor).
				Bold(true)

	// Overlays
	overlayTitleStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(accentColor).
				Padding(0, 1)

	overlayItemStyle = lipgloss.NewStyle().
				Padding(0, 2)

	overlaySelectedStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#FFFFFF")).
				Background(accentColor).
				Padding(0, 2)

	// Borders
	borderStyle = lipgloss.NewStyle().
			Border(lipgloss.NormalBorder()).
			BorderForeground(dimColor)

	// Help
	helpKeyStyle = lipgloss.NewStyle().
			Foreground(accentColor).
			Bold(true)

	helpDescStyle = lipgloss.NewStyle().
			Foreground(dimColor)

	// Spinner/streaming indicator
	spinnerStyle = lipgloss.NewStyle().
			Foreground(assistantColor)
)
