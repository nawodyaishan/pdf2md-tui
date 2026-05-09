package tui

import (
	"fmt"
	"runtime"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/nawodyaishan/pdf2md-tui/pkg/domain"
	"github.com/nawodyaishan/pdf2md-tui/pkg/version"
	"github.com/pterm/pterm"
)

const (
	author  = "nawodyaishan"
	repoURL = "github.com/nawodyaishan/pdf2md-tui"
)

// Progress represents the TUI state for the conversion process.
type Progress struct {
	spinner      *pterm.SpinnerPrinter
	area         *pterm.AreaPrinter
	totalCount   int
	currentCount int
	workerCount  int
	lastSysInfo  domain.SysInfo
}

// New creates a new Progress instance.
func New() *Progress {
	return &Progress{}
}

func (p *Progress) SetBatchInfo(total, workers int) {
	p.totalCount = total
	p.workerCount = workers
}

// PrintBanner renders the branded startup banner.
func (p *Progress) PrintBanner() {
	brand := lipgloss.JoinHorizontal(lipgloss.Left,
		TitleStyle.Render(" PDF2MD "),
		" ",
		StatusMutedStyle.Render("TUI"),
	)
	subtitle := lipgloss.NewStyle().
		Foreground(PrimaryColor).
		Italic(true).
		Render("LLM-optimized PDF → Markdown converter")
	meta := SubtleTextStyle.Render(fmt.Sprintf("v%s • by @%s", version.Version, author))

	banner := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(GrayColor).
		Padding(1, 2).
		Render(lipgloss.JoinVertical(lipgloss.Left, brand, subtitle, meta))

	pterm.Println(banner)
	pterm.Println()
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
			// handled by PrintNoPDFsFound in main loop if quiet isn't on
		} else {
			p.spinner.Success(fmt.Sprintf("Discovered %d PDF files.", count))
		}
		p.spinner = nil
	}
}

// PrintNoPDFsFound prints a graceful message when no PDFs are discovered.
func (p *Progress) PrintNoPDFsFound(dir string) {
	pterm.Warning.Printf("No PDF files found in '%s'.\n", dir)
	pterm.Info.Println("Use 'pdf2md-tui --help' for usage instructions.")
}

// StartConversion starts the progress bar for conversion.
func (p *Progress) StartConversion(total, workers int) {
	p.totalCount = total
	p.workerCount = workers
	p.currentCount = 0

	// Initialize area for live updates
	area, _ := pterm.DefaultArea.Start()
	p.area = area
	p.renderLiveDashboard()
}

// Increment increments the progress bar and refreshes the live area.
func (p *Progress) Increment() {
	p.currentCount++
	p.renderLiveDashboard()
}

// UpdateLiveStats updates the system info and refreshes the live area.
func (p *Progress) UpdateLiveStats(sys domain.SysInfo) {
	p.lastSysInfo = sys
	p.renderLiveDashboard()
}

// StopConversion stops the live area updates.
func (p *Progress) StopConversion() {
	if p.area != nil {
		_ = p.area.Stop()
		p.area = nil
	}
}

func (p *Progress) renderLiveDashboard() {
	if p.area == nil {
		return
	}

	// 1. Conversion Progress
	barWidth := 40
	progressPct := float64(p.currentCount) / float64(p.totalCount)
	filled := int(progressPct * float64(barWidth))
	barStr := fmt.Sprintf("%s [%s%s] %d/%d (%d%%)",
		pterm.Bold.Sprint("Converting PDFs:"),
		pterm.LightCyan(strings.Repeat("█", filled)),
		pterm.NewStyle(pterm.FgGray).Sprint(strings.Repeat("░", barWidth-filled)),
		p.currentCount, p.totalCount, int(progressPct*100),
	)

	// 2. Resource Info (Compact)
	cpuWidth := 15
	cpuFilled := int((p.lastSysInfo.CPUUsage / 100.0) * float64(cpuWidth))
	if cpuFilled > cpuWidth {
		cpuFilled = cpuWidth
	}
	cpuBar := fmt.Sprintf("[%s%s]",
		pterm.LightMagenta(strings.Repeat("■", cpuFilled)),
		pterm.NewStyle(pterm.FgGray).Sprint(strings.Repeat(" ", cpuWidth-cpuFilled)),
	)

	resStr := fmt.Sprintf(
		" %s %s %.1f%%  %s %s %.1f%%",
		pterm.LightCyan("CPU:"), cpuBar, p.lastSysInfo.CPUUsage,
		pterm.LightMagenta("MEM:"), pterm.Gray(FormatBytes(int64(p.lastSysInfo.MemoryUsed))), p.lastSysInfo.MemoryPct,
	)

	// 3. Hardware Map (Single Line)
	coreMap := ""
	totalCores := runtime.NumCPU()
	for i := 0; i < totalCores; i++ {
		if i < p.workerCount {
			coreMap += pterm.NewStyle(pterm.FgCyan).Sprint("▣")
		} else {
			coreMap += pterm.NewStyle(pterm.FgDarkGray).Sprint("□")
		}
	}
	hwStr := fmt.Sprintf(" %s %s", pterm.LightBlue("CORES:"), coreMap)

	// Combine
	p.area.Update(fmt.Sprintf("%s\n%s\n%s", barStr, resStr, hwStr))
}

// PrintSummary prints a branded summary of the conversion results.
func (p *Progress) PrintSummary(results []domain.Result, inputBytes, outputBytes int64, duration time.Duration, errCount, ignoredCount int, sys domain.SysInfo) {
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
	errStyle := pterm.NewStyle(pterm.FgLightGreen)
	errLabel := "✓ None"

	if errCount > 0 || ignoredCount > 0 {
		errStyle = pterm.NewStyle(pterm.FgLightYellow)
		var labels []string
		if errCount > 0 {
			labels = append(labels, fmt.Sprintf("%d failed", errCount))
			errStyle = pterm.NewStyle(pterm.FgLightRed, pterm.Bold)
		}
		if ignoredCount > 0 {
			labels = append(labels, fmt.Sprintf("%d skipped (OCR req)", ignoredCount))
		}
		errLabel = "✗ " + strings.Join(labels, ", ")
	}

	// Processing Info Panel (Left)
	processingInfo := fmt.Sprintf(
		"%s %d\n%s %s\n%s %s",
		pterm.Cyan("Files Processed:"), len(results),
		pterm.Cyan("Errors / Ignored:"), errStyle.Sprint(errLabel),
		pterm.Cyan("Total Duration:"), duration.Round(time.Millisecond).String(),
	)

	// Efficiency Stats Panel (Middle)
	efficiencyStats := fmt.Sprintf(
		"%s %s %s\n%s %s %s\n%s %s",
		pterm.LightMagenta("PDF Source:"), FormatBytes(inputBytes), pterm.Gray(fmt.Sprintf("(~%d tkn)", inputTokens)),
		pterm.LightMagenta("MD Output: "), FormatBytes(outputBytes), pterm.Gray(fmt.Sprintf("(~%d tkn)", outputTokens)),
		pterm.LightMagenta("Efficiency:"), pterm.NewStyle(pterm.FgLightGreen, pterm.Bold).Sprintf("▼ %.1f%%", savings),
	)

	// Core Utilization Panel (Right)
	totalCores := runtime.NumCPU()
	coreMap := ""
	for i := 0; i < totalCores; i++ {
		if i > 0 && i%4 == 0 {
			coreMap += "\n"
		}
		if i < p.workerCount {
			coreMap += pterm.NewStyle(pterm.FgCyan).Sprint("▣ ")
		} else {
			coreMap += pterm.NewStyle(pterm.FgDarkGray).Sprint("□ ")
		}
	}

	coreInfo := fmt.Sprintf(
		"%s %d/%d\n%s\n%s",
		pterm.LightBlue("Cores Utilized:"), p.workerCount, totalCores,
		coreMap,
		pterm.Gray(fmt.Sprintf("Parallelism: %.1fx", float64(p.workerCount))),
	)

	// Resource Usage Panel
	cpuColor := pterm.FgLightGreen
	if sys.CPUUsage > 80 {
		cpuColor = pterm.FgLightRed
	} else if sys.CPUUsage > 50 {
		cpuColor = pterm.FgLightYellow
	}

	memColor := pterm.FgLightGreen
	if sys.MemoryPct > 80 {
		memColor = pterm.FgLightRed
	}

	resourceInfo := fmt.Sprintf(
		"%s %s\n%s %s\n%s %s",
		pterm.LightBlue("Resource Peaks:"), "",
		pterm.Cyan("Avg CPU Load:"), pterm.NewStyle(cpuColor).Sprintf("%.1f%%", sys.CPUUsage),
		pterm.Cyan("Peak Memory: "), pterm.NewStyle(memColor).Sprintf("%s (%.1f%%)", FormatBytes(int64(sys.MemoryUsed)), sys.MemoryPct),
	)

	// Create panels
	panels := pterm.Panels{
		{
			{Data: processingInfo},
			{Data: efficiencyStats},
			{Data: coreInfo},
			{Data: resourceInfo},
		},
	}

	// Render panels
	_ = pterm.DefaultPanel.WithPanels(panels).WithPadding(2).Render()

	// If there are failures, list them clearly
	if errCount > 0 {
		pterm.Println()
		pterm.DefaultSection.
			WithLevel(2).
			WithStyle(pterm.NewStyle(pterm.FgLightRed, pterm.Bold)).
			Println("❌ Failure Details")

		for _, res := range results {
			if res.Err != nil {
				pterm.Printf("%s  %s\n", pterm.NewStyle(pterm.FgLightRed).Sprint("•"), pterm.NewStyle(pterm.FgDefault, pterm.Bold).Sprint(res.InputPath))
				pterm.Printf("   %s %s\n", pterm.NewStyle(pterm.FgGray).Sprint("Reason:"), pterm.NewStyle(pterm.FgLightRed, pterm.Italic).Sprint(res.Err.Error()))
			}
		}
	}

	// Footer
	pterm.Println()
	pterm.DefaultCenter.WithCenterEachLineSeparately().Println(
		pterm.Gray(fmt.Sprintf("─── %s ───", repoURL)),
	)
	pterm.Println()
}

// ShowPostConversionMenu displays interactive options after conversion.
func (p *Progress) ShowPostConversionMenu(hasErrors bool) (string, error) {
	options := []string{"📁 Open Output Directory", "🚪 Exit"}
	if hasErrors {
		options = []string{"📁 Open Output Directory", "📄 View Detailed Log", "🚪 Exit"}
	}

	selected, err := pterm.DefaultInteractiveSelect.
		WithMaxHeight(5).
		WithOptions(options).
		Show("What would you like to do next?")
	if err != nil {
		return "", err
	}

	switch selected {
	case "📁 Open Output Directory":
		return "open_dir", nil
	case "📄 View Detailed Log":
		return "view_log", nil
	default:
		return "exit", nil
	}
}

func FormatBytes(b int64) string {
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

// RunDashboard starts the Bubble Tea program and returns the program instance and a channel to send updates.
func (p *Progress) RunDashboard(total, workers int) (*tea.Program, chan tea.Msg) {
	model := NewModel(total, workers)
	prog := tea.NewProgram(model, tea.WithAltScreen())
	msgChan := make(chan tea.Msg)

	// Background loop to feed the program
	go func() {
		for msg := range msgChan {
			prog.Send(msg)
		}
	}()

	return prog, msgChan
}
