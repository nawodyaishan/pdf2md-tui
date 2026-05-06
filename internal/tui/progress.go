package tui

import (
	"fmt"
	"time"

	"github.com/pterm/pterm"
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

// StartDiscovery starts the spinner for the discovery phase.
func (p *Progress) StartDiscovery() {
	spinner, _ := pterm.DefaultSpinner.Start("Scanning for PDF files...")
	p.spinner = spinner
}

// StopDiscovery stops the spinner.
func (p *Progress) StopDiscovery(count int) {
	if p.spinner != nil {
		if count == 0 {
			p.spinner.Warning("No PDF files found.")
		} else {
			p.spinner.Success(fmt.Sprintf("Found %d PDF files.", count))
		}
		p.spinner = nil
	}
}

// StartConversion starts the progress bar for conversion.
func (p *Progress) StartConversion(total int) {
	p.totalCount = total
	bar, _ := pterm.DefaultProgressbar.WithTotal(total).WithTitle("Converting PDFs").Start()
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
		p.bar.Stop()
	}
}

// PrintSummary prints a colorful summary of the conversion.
func (p *Progress) PrintSummary(inputBytes, outputBytes int64, duration time.Duration, errCount int) {
	pterm.DefaultSection.Println("Conversion Summary")

	savings := float64(0)
	if inputBytes > 0 {
		savings = float64(inputBytes-outputBytes) / float64(inputBytes) * 100
	}

	// Calculate rough token estimates (4 chars per token)
	inputTokens := inputBytes / 4
	outputTokens := outputBytes / 4

	data := [][]string{
		{"Metric", "Value"},
		{"Total Processed", fmt.Sprintf("%d", p.totalCount)},
		{"Errors", fmt.Sprintf("%d", errCount)},
		{"Duration", duration.String()},
		{"PDF Source Size", formatBytes(inputBytes) + fmt.Sprintf(" (~%d tokens)", inputTokens)},
		{"Markdown Size", formatBytes(outputBytes) + fmt.Sprintf(" (~%d tokens)", outputTokens)},
		{"Token Savings", fmt.Sprintf("~%.1f%% reduction", savings)},
	}

	pterm.DefaultTable.WithHasHeader().WithData(data).Render()
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
