package tui

import "github.com/charmbracelet/lipgloss"

// Colours used across the TUI. Kept as package-level vars so a future
// theme system can swap them.
var (
	colorPrimary   = lipgloss.Color("#B48EAD") // orchid purple
	colorSecondary = lipgloss.Color("#81A1C1") // steel blue
	colorMuted     = lipgloss.Color("#4C566A") // grey
	colorText      = lipgloss.Color("#ECEFF4") // near-white
	colorBg        = lipgloss.Color("#2E3440") // dark bg
	colorGreen     = lipgloss.Color("#A3BE8C")
	colorYellow    = lipgloss.Color("#EBCB8B")
	colorRed       = lipgloss.Color("#BF616A")
	colorOrange    = lipgloss.Color("#D08770")
	colorCyan      = lipgloss.Color("#88C0D0")
)

// Layout styles.
var (
	sidebarStyle = lipgloss.NewStyle().
			Padding(1, 1).
			BorderStyle(lipgloss.RoundedBorder()).
			BorderForeground(colorMuted)

	sidebarActiveStyle = lipgloss.NewStyle().
				Padding(1, 1).
				BorderStyle(lipgloss.RoundedBorder()).
				BorderForeground(colorPrimary)

	detailStyle = lipgloss.NewStyle().
			Padding(1, 2).
			BorderStyle(lipgloss.RoundedBorder()).
			BorderForeground(colorMuted)

	statusBarStyle = lipgloss.NewStyle().
			Foreground(colorText).
			Background(lipgloss.Color("#3B4252")).
			Padding(0, 1)

	projectHeaderStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(colorSecondary)

	selectedItemStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(colorPrimary)

	normalItemStyle = lipgloss.NewStyle().
			Foreground(colorText)

	dimStyle = lipgloss.NewStyle().
			Foreground(colorMuted)

	detailLabelStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(colorSecondary).
				Width(12)

	detailValueStyle = lipgloss.NewStyle().
				Foreground(colorText)

	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(colorPrimary).
			Padding(0, 1)

	filterPromptStyle = lipgloss.NewStyle().
				Foreground(colorYellow)

	dialogStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(colorPrimary).
			Padding(1, 2).
			Width(50)

	dialogTitleStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(colorPrimary).
				MarginBottom(1)

	dialogSelectedStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(colorPrimary)

	dialogNormalStyle = lipgloss.NewStyle().
				Foreground(colorText)

	// paneViewStyle wraps the captured terminal content with minimal
	// padding to maximise the visible area.
	paneViewStyle = lipgloss.NewStyle().
			BorderStyle(lipgloss.RoundedBorder()).
			BorderForeground(colorMuted)

	// paneHeaderStyle renders the compact agent header above the pane.
	paneHeaderStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(colorPrimary).
			PaddingLeft(1)
)

// statusStyle returns the lipgloss style for a given agent status string.
func statusStyle(status string) lipgloss.Style {
	switch status {
	case "idle":
		return lipgloss.NewStyle().Foreground(colorGreen)
	case "thinking":
		return lipgloss.NewStyle().Foreground(colorCyan)
	case "tool_use":
		return lipgloss.NewStyle().Foreground(colorYellow)
	case "error":
		return lipgloss.NewStyle().Foreground(colorRed)
	case "done":
		return lipgloss.NewStyle().Foreground(colorGreen).Bold(true)
	default:
		return lipgloss.NewStyle().Foreground(colorMuted)
	}
}

// statusIcon returns a unicode indicator for the agent status.
func statusIcon(status string) string {
	switch status {
	case "idle":
		return "~"
	case "thinking":
		return "*"
	case "tool_use":
		return ">"
	case "error":
		return "!"
	case "done":
		return "+"
	default:
		return "?"
	}
}
