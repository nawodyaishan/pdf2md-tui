package tui

import "github.com/charmbracelet/lipgloss"

var (
	// Theme colors
	PrimaryColor   = lipgloss.Color("#5EEAD4")
	SecondaryColor = lipgloss.Color("#93C5FD")
	AccentColor    = lipgloss.Color("#F6C177")
	SuccessColor   = lipgloss.Color("#86EFAC")
	WarningColor   = lipgloss.Color("#FCD34D")
	ErrorColor     = lipgloss.Color("#FDA4AF")
	TextColor      = lipgloss.Color("#E5EEF9")
	GrayColor      = lipgloss.Color("#243244")
	LightGrayColor = lipgloss.Color("#8FA3BF")
	BgColor        = lipgloss.Color("#020617")
	SurfaceColor   = lipgloss.Color("#0F172A")
	SurfaceAlt     = lipgloss.Color("#111827")

	HeaderStyle = lipgloss.NewStyle().
			Foreground(TextColor).
			Padding(0, 1, 1, 1).
			MarginBottom(1)

	FooterStyle = lipgloss.NewStyle().
			Foreground(LightGrayColor).
			Padding(1, 1, 0, 1)

	TitleStyle = lipgloss.NewStyle().
			Foreground(PrimaryColor).
			Padding(0, 1, 0, 0).
			Bold(true)

	MetricLabelStyle = lipgloss.NewStyle().
				Foreground(LightGrayColor).
				Width(15)

	MetricValueStyle = lipgloss.NewStyle().
				Foreground(TextColor).
				Bold(true)

	BorderStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(GrayColor).
			Padding(1, 2)

	CoreActiveStyle = lipgloss.NewStyle().Foreground(PrimaryColor)
	CoreIdleStyle   = lipgloss.NewStyle().Foreground(GrayColor)

	StatusActiveStyle = lipgloss.NewStyle().
				Foreground(SecondaryColor).
				Bold(true)

	StatusSuccessStyle = lipgloss.NewStyle().
				Foreground(SuccessColor).
				Bold(true)

	StatusMutedStyle = lipgloss.NewStyle().
				Foreground(LightGrayColor)

	CardTitleStyle = lipgloss.NewStyle().
			Foreground(TextColor).
			Bold(true)

	SectionTitleStyle = lipgloss.NewStyle().
				Foreground(LightGrayColor).
				Bold(true)

	SubtleTextStyle = lipgloss.NewStyle().
			Foreground(LightGrayColor)

	ValueStrongStyle = lipgloss.NewStyle().
				Foreground(TextColor).
				Bold(true)

	SuccessFooterStyle = lipgloss.NewStyle().
				Foreground(TextColor).
				Bold(true)

	KeyHintStyle = lipgloss.NewStyle().
			Foreground(SecondaryColor).
			Bold(true)

	KeyHintMutedStyle = lipgloss.NewStyle().
				Foreground(TextColor)

	DimmedStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#475569"))
)
