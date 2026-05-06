package tui

import (
	"fmt"
	"time"

	"github.com/pterm/pterm"
	"github.com/pterm/pterm/putils"
	"github.com/nawodyaishan/pdf2md-tui/pkg/version"
)

const (
	author    = "nawodyaishan"
	repoURL   = "github.com/nawodyaishan/pdf2md-tui"
)

// Progress represents the TUI state for the conversion process.
type Progress struct {
	spinner    *pterm.SpinnerPrinter
	bar        *pterm.ProgressbarPrinter
	totalCount int
}

// New creates a new Progress instance.
func New() *Progress {
	return &Progress{}
}

// PrintBanner renders the branded startup banner.
func (p *Progress) PrintBanner() {
	banner, _ := pterm.DefaultBigText.WithLetters(
		putils.LettersFromStringWithStyle("pdf", pterm.NewStyle(pterm.FgCyan)),
		putils.LettersFromStringWithStyle("2md", pterm.NewStyle(pterm.FgLightMagenta)),
	).Srender()
	pterm.Print(banner)

	pterm.DefaultCenter.WithCenterEachLineSeparately().Println(
		pterm.LightCyan("⚡ LLM-Optimized PDF → Markdown Converter"),
	)
	pterm.DefaultCenter.WithCenterEachLineSeparately().Println(
		pterm.Gray(14, fmt.Sprintf("v%s • by @%s", version.Version, author)),
	)
	pterm.Println() // breathing room
}

// StartDiscovery starts the spinner for the discovery phase.
func (p *Progress) StartDiscovery() {
	spinner, _ := pterm.DefaultSpinner.
		WithRemoveWhenDone(false).
		Start("Scanning for PDF files...")
	p.spinner = spinner
}

// StopDiscovery stops the spinner.
func (p *Progress) StopDiscovery(count int) {
	if p.spinner != nil {
		if count == 0 {
			p.spinner.Warning("No PDF files found.")
		} else {
			p.spinner.Success(fmt.Sprintf("Discovered %d PDF files.", count))
		}
		p.spinner = nil
	}
}

// StartConversion starts the progress bar for conversion.
func (p *Progress) StartConversion(total int) {
	p.totalCount = total
	bar, _ := pterm.DefaultProgressbar.
		WithTotal(total).
		WithTitle("Converting PDFs").
		WithBarCharacter("█").
		WithLastCharacter("█").
		WithElapsedTimeRoundingFactor(time.Millisecond).
		Start()
	p.bar = bar
}

// Increment increments the progress bar.
func (p *Progress) Increment() {
	if p.bar != nil {
		p.bar.Increment()
	}
}

// StopConversion stops the progress bar.
func (p *Progress) StopConversion() {
	if p.bar != nil {
		_, _ = p.bar.Stop()
	}
}

// PrintSummary prints a branded summary of the conversion results.
func (p *Progress) PrintSummary(inputBytes, outputBytes int64, duration time.Duration, errCount int) {
	pterm.Println() // space before summary

	// Section header with icon
	pterm.DefaultSection.
		WithStyle(pterm.NewStyle(pterm.FgLightCyan, pterm.Bold)).
		Println("📊 Conversion Summary")

	savings := float64(0)
	if inputBytes > 0 {
		savings = float64(inputBytes-outputBytes) / float64(inputBytes) * 100
	}

	// Token estimates (~4 chars per token)
	inputTokens := inputBytes / 4
	outputTokens := outputBytes / 4

	// Status row color
	errStyle := pterm.FgGreen
	errLabel := "✓ None"
	if errCount > 0 {
		errStyle = pterm.FgRed
		errLabel = fmt.Sprintf("✗ %d failed", errCount)
	}

	data := [][]string{
		{"Metric", "Value"},
		{"Files Processed", fmt.Sprintf("%d", p.totalCount)},
		{"Errors", pterm.NewStyle(errStyle).Sprint(errLabel)},
		{"Duration", duration.Round(time.Millisecond).String()},
		{"", ""},
		{"PDF Source Size", formatBytes(inputBytes) + pterm.Gray(14, fmt.Sprintf("  (~%d tokens)", inputTokens))},
		{"Markdown Output", formatBytes(outputBytes) + pterm.Gray(14, fmt.Sprintf("  (~%d tokens)", outputTokens))},
		{"Token Savings", pterm.NewStyle(pterm.FgLightGreen, pterm.Bold).Sprintf("▼ %.1f%% reduction", savings)},
	}

	_ = pterm.DefaultTable.
		WithHasHeader().
		WithHeaderStyle(pterm.NewStyle(pterm.FgCyan, pterm.Bold)).
		WithData(data).
		Render()

	// Footer
	pterm.Println()
	pterm.DefaultCenter.WithCenterEachLineSeparately().Println(
		pterm.Gray(14, fmt.Sprintf("─── %s ───", repoURL)),
	)
	pterm.Println()
}

func formatBytes(b int64) string {
	const unit = 1024
	if b < unit {
		return fmt.Sprintf("%d B", b)
	}
	div, exp := int64(unit), 0
	for n := b / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(b)/float64(div), "KMGTPE"[exp])
}
