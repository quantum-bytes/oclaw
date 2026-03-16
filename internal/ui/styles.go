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
	chatTextColor  = lipgloss.Color("#9CA3AF") // Light gray for chat text

	// Shimmer gradient colors for thinking animation
	shimmerColors = []lipgloss.Color{
		"#6366F1", // Indigo
		"#8B5CF6", // Violet
		"#A78BFA", // Light violet
		"#C4B5FD", // Lavender
		"#A78BFA", // Light violet
		"#8B5CF6", // Violet
		"#7C3AED", // Purple
		"#6D28D9", // Deep purple
		"#5B21B6", // Darker purple
		"#6D28D9", // Deep purple
	}

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

	userTextStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#D1D5DB")) // Brighter for user text

	assistantLabelStyle = lipgloss.NewStyle().
				Foreground(assistantColor).
				Bold(true)

	assistantTextStyle = lipgloss.NewStyle().
				Foreground(chatTextColor)

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
