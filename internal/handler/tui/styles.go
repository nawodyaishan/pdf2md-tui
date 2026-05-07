package tui

import "github.com/charmbracelet/lipgloss"

var (
	// Colors
	PrimaryColor   = lipgloss.Color("#00FFFF") // Cyan
	SecondaryColor = lipgloss.Color("#FF00FF") // Magenta
	AccentColor    = lipgloss.Color("#5F5FFF") // Bright Blue
	GrayColor      = lipgloss.Color("#3C3C3C")
	LightGrayColor = lipgloss.Color("#808080")
	BgColor        = lipgloss.Color("#1A1A1A")

	// Styles
	HeaderStyle = lipgloss.NewStyle().
			Foreground(PrimaryColor).
			Bold(true).
			Padding(0, 1).
			MarginBottom(1)

	FooterStyle = lipgloss.NewStyle().
			Foreground(LightGrayColor).
			Italic(true).
			Padding(0, 1)

	TitleStyle = lipgloss.NewStyle().
			Background(PrimaryColor).
			Foreground(BgColor).
			Padding(0, 1).
			Bold(true)

	MetricLabelStyle = lipgloss.NewStyle().
				Foreground(LightGrayColor).
				Width(15)

	MetricValueStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#FFFFFF")).
				Bold(true)

	BorderStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(GrayColor).
			Padding(1)

	CoreActiveStyle = lipgloss.NewStyle().Foreground(PrimaryColor)
	CoreIdleStyle   = lipgloss.NewStyle().Foreground(GrayColor)

	SuccessFooterStyle = lipgloss.NewStyle().
				Background(PrimaryColor).
				Foreground(lipgloss.Color("#000000")).
				Bold(true).
				Padding(0, 1)

	DimmedStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#444444"))
)
