package tui

import (
	"fmt"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/nawodyaishan/pdf2md-tui/internal/domain"
)

func (m Model) renderDashboard() string {
	if m.Complete {
		return m.renderCompletionView()
	}

	stats := m.renderStats()
	cores := m.renderCoreGrid()
	table := m.renderFileTable()

	contentWidth := availableWidth(m.width, 4, 16, 120)
	panelHeight := 10

	var topRow string
	if m.width < 90 {
		statsPanel := BorderStyle.Width(contentWidth).Height(panelHeight).Render(stats)
		coresPanel := BorderStyle.Width(contentWidth).Height(panelHeight).Render(cores)
		topRow = lipgloss.JoinVertical(lipgloss.Left, statsPanel, coresPanel)
	} else {
		panelWidth := maxInt(20, (contentWidth-2)/2)
		topRow = lipgloss.JoinHorizontal(lipgloss.Top,
			BorderStyle.Width(panelWidth).Height(panelHeight).Render(stats),
			BorderStyle.Width(panelWidth).Height(panelHeight).Render(cores),
		)
	}

	tableHeight := maxInt(6, m.height-lipgloss.Height(topRow)-6)

	return lipgloss.JoinVertical(lipgloss.Left,
		topRow,
		BorderStyle.Width(contentWidth).Height(tableHeight).Render(table),
	)
}

func (m Model) renderStats() string {
	cpuUsage := m.SysInfo.CPUUsage
	memPct := m.SysInfo.MemoryPct
	barWidth := 16
	if m.width < 90 {
		barWidth = 12
	}

	cpuBar := renderProgressBar(int(cpuUsage), 100, barWidth, SecondaryColor)
	memBar := renderProgressBar(int(memPct), 100, barWidth, AccentColor)
	batchBar := renderProgressBar(m.CurrentFile, m.TotalFiles, barWidth, PrimaryColor)

	return lipgloss.JoinVertical(lipgloss.Left,
		CardTitleStyle.Render("Pipeline Metrics"),
		SubtleTextStyle.Render("throughput, resource load, and session uptime"),
		"",
		renderMetricLine("Batch", batchBar, fmt.Sprintf("%d/%d", m.CurrentFile, m.TotalFiles)),
		renderMetricLine("CPU", cpuBar, fmt.Sprintf("%.1f%%", cpuUsage)),
		renderMetricLine("Memory", memBar, fmt.Sprintf("%.1f%%", memPct)),
		"",
		renderInlinePair("Uptime", timeSince(m.StartTime), "Workers", fmt.Sprintf("%d", m.WorkerCount)),
	)
}

func (m Model) renderCoreGrid() string {
	totalCores := runtime.NumCPU()
	var grid strings.Builder
	grid.WriteString(CardTitleStyle.Render("Worker Topology") + "\n")
	grid.WriteString(SubtleTextStyle.Render(fmt.Sprintf("%d active workers across %d logical cores", m.WorkerCount, totalCores)) + "\n\n")

	cols := 8
	if m.width < 100 {
		cols = 6
	}

	for i := 0; i < totalCores; i++ {
		style := DimmedStyle
		char := "·"
		if i < m.WorkerCount {
			style = CoreActiveStyle
			char = "●"
		}
		grid.WriteString(style.Render(char + " "))
		if (i+1)%cols == 0 {
			grid.WriteString("\n")
		}
	}

	grid.WriteString("\n")
	grid.WriteString(renderInlinePair("Parallelism", fmt.Sprintf("%.1fx", float64(m.WorkerCount)), "Host", runtime.GOOS))

	return grid.String()
}

func (m Model) renderFileTable() string {
	if len(m.Results) == 0 {
		return lipgloss.JoinVertical(lipgloss.Left,
			CardTitleStyle.Render("Recent Activity"),
			SubtleTextStyle.Render("waiting for workers to emit the first result"),
		)
	}

	var s strings.Builder
	s.WriteString(CardTitleStyle.Render("Recent Activity") + "\n")
	s.WriteString(SubtleTextStyle.Render("latest completed files in this batch") + "\n\n")

	count := 0
	for i := len(m.Results) - 1; i >= 0 && count < 10; i-- {
		res := m.Results[i]
		name := truncatePath(filepath.Base(res.InputPath), 28)
		if name == "" {
			name = truncatePath(res.InputPath, 28)
		}
		fmt.Fprintf(&s, "%s  %s  %s\n",
			renderResultStatus(res),
			ValueStrongStyle.Render(name),
			SubtleTextStyle.Render(formatDuration(res.Duration)),
		)
		count++
	}

	return s.String()
}

func renderProgressBar(current, total, width int, color lipgloss.Color) string {
	if total == 0 {
		return "[]"
	}

	// Success Bloom: turn green if complete
	if current >= total && total > 0 {
		color = SuccessColor
	}

	ratio := float64(current) / float64(total)
	filled := int(ratio * float64(width))
	if filled > width {
		filled = width
	}

	return fmt.Sprintf("[%s%s]",
		lipgloss.NewStyle().Foreground(color).Render(strings.Repeat("█", filled)),
		lipgloss.NewStyle().Foreground(GrayColor).Render(strings.Repeat("░", width-filled)),
	)
}

func truncatePath(p string, max int) string {
	if len(p) <= max {
		return p
	}
	return "..." + p[len(p)-max+3:]
}

func formatDuration(d time.Duration) string {
	return fmt.Sprintf("%.2fs", d.Seconds())
}

func (m Model) renderCompletionView() string {
	cardWidth := availableWidth(m.width, 8, 24, 110)
	summary := m.renderSummaryPanel(cardWidth)
	menu := lipgloss.JoinVertical(lipgloss.Left,
		CardTitleStyle.Render("Next Action"),
		SubtleTextStyle.Render("choose what happens after this batch"),
		"",
		m.renderCompletionMenu(cardWidth),
	)

	content := lipgloss.JoinVertical(lipgloss.Center,
		summary,
		lipgloss.NewStyle().Height(2).Render(""), // Spacer
		menu,
	)

	horizontalPadding := 1
	if cardWidth >= 60 {
		horizontalPadding = 2
	}
	if cardWidth >= 84 {
		horizontalPadding = 3
	}

	card := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(GrayColor).
		Padding(1, horizontalPadding).
		Render(content)

	topPadding := clampInt((m.height-lipgloss.Height(card))/4, 0, 4)
	placed := lipgloss.PlaceHorizontal(m.width, lipgloss.Center, card)
	if topPadding == 0 {
		return placed
	}
	return strings.Repeat("\n", topPadding) + placed
}

func (m Model) renderSummaryPanel(width int) string {
	var results = m.Results
	var totalIn, totalOut int64
	var totalInTkn, totalOutTkn int
	var errCount, ignoredCount int

	for _, r := range results {
		switch {
		case r.Err != nil || r.Status == domain.StatusError:
			errCount++
		case r.Status == domain.StatusIgnored:
			ignoredCount++
		default:
			totalIn += r.InputBytes
			totalOut += r.OutputBytes
			totalInTkn += r.InputTokens
			totalOutTkn += r.OutputTokens
		}
	}

	savings := 0.0
	if totalIn > 0 {
		savings = float64(totalIn-totalOut) / float64(totalIn) * 100
	}

	title := lipgloss.NewStyle().
		Width(width).
		Align(lipgloss.Center).
		Foreground(TextColor).
		Bold(true).
		Render("Session Summary")

	avgCPU := 0.0
	if m.sysCount > 0 {
		avgCPU = m.AvgCPU / float64(m.sysCount)
	}

	duration := time.Since(m.StartTime)
	if m.Complete {
		duration = m.FinalDuration
	}

	leftCol := fmt.Sprintf(
		"%s\n%s\n%s",
		renderSummaryField("Files Processed", fmt.Sprintf("%d", len(results))),
		renderSummaryField("Errors / Ignored", formatErrorCount(errCount, ignoredCount)),
		renderSummaryField("Total Duration", fmt.Sprintf("%.2fs", duration.Seconds())),
	)

	midCol := fmt.Sprintf(
		"%s\n%s\n%s",
		renderSummaryField("PDF Source", fmt.Sprintf("%s (%d tkn)", formatBytes(totalIn), totalInTkn)),
		renderSummaryField("MD Output", fmt.Sprintf("%s (%d tkn)", formatBytes(totalOut), totalOutTkn)),
		renderSummaryField("Efficiency", fmt.Sprintf("▼ %.1f%%", savings)),
	)

	rightCol := fmt.Sprintf(
		"%s\n%s\n%s",
		renderSummaryField("Cores Utilized", fmt.Sprintf("%d/%d", m.WorkerCount, runtime.NumCPU())),
		renderSummaryField("Avg CPU Load", fmt.Sprintf("%.1f%%", avgCPU)),
		renderSummaryField("Peak Memory", fmt.Sprintf("%s (%.1f%%)", formatBytes(int64(m.PeakMemory)), m.MaxMemPct)),
	)

	stats := m.renderSummaryStats(width, leftCol, midCol, rightCol)
	subtitle := lipgloss.NewStyle().
		Width(width).
		MaxWidth(width).
		Align(lipgloss.Center).
		Foreground(LightGrayColor).
		Render("conversion outcomes, output efficiency, and resource peaks")

	return lipgloss.JoinVertical(lipgloss.Center,
		title,
		subtitle,
		"",
		stats,
	)
}

func renderSummaryField(label, value string) string {
	return lipgloss.JoinVertical(lipgloss.Left,
		SectionTitleStyle.Render(label),
		ValueStrongStyle.Render(value),
	)
}

func renderMetricLine(label, bar, value string) string {
	return lipgloss.JoinHorizontal(lipgloss.Center,
		MetricLabelStyle.Render(label),
		bar,
		" ",
		ValueStrongStyle.Render(value),
	)
}

func renderInlinePair(leftLabel, leftValue, rightLabel, rightValue string) string {
	return lipgloss.JoinHorizontal(lipgloss.Left,
		SectionTitleStyle.Render(leftLabel),
		" ",
		ValueStrongStyle.Render(leftValue),
		"   ",
		SectionTitleStyle.Render(rightLabel),
		" ",
		ValueStrongStyle.Render(rightValue),
	)
}

func renderResultStatus(res domain.Result) string {
	switch {
	case res.Err != nil || res.Status == domain.StatusError:
		return statusPill("FAIL", ErrorColor)
	case res.Status == domain.StatusIgnored:
		return statusPill("SKIP", WarningColor)
	default:
		return statusPill("DONE", SuccessColor)
	}
}

func statusPill(label string, color lipgloss.Color) string {
	return lipgloss.NewStyle().
		Foreground(color).
		Bold(true).
		Render(label)
}

func (m Model) renderSummaryStats(width int, leftCol, midCol, rightCol string) string {
	sections := []string{leftCol, midCol, rightCol}
	gap := 2

	switch {
	case width >= 96:
		colWidth := maxInt(24, (width-(gap*2))/3)
		return lipgloss.JoinHorizontal(lipgloss.Top,
			renderSummarySection(sections[0], colWidth),
			renderSummarySection(sections[1], colWidth),
			renderSummarySection(sections[2], colWidth),
		)
	case width >= 64:
		colWidth := maxInt(24, (width-gap)/2)
		topRow := lipgloss.JoinHorizontal(lipgloss.Top,
			renderSummarySection(sections[0], colWidth),
			renderSummarySection(sections[1], colWidth),
		)
		return lipgloss.JoinVertical(lipgloss.Left,
			topRow,
			renderSummarySection(sections[2], width),
		)
	default:
		rendered := make([]string, 0, len(sections))
		for _, section := range sections {
			rendered = append(rendered, renderSummarySection(section, width))
		}
		return lipgloss.JoinVertical(lipgloss.Left, rendered...)
	}
}

func renderSummarySection(content string, width int) string {
	innerWidth := maxInt(1, width-2)
	return lipgloss.NewStyle().
		Width(innerWidth).
		MaxWidth(innerWidth).
		Padding(1, 1).
		Render(content)
}

func formatErrorCount(errs, ignored int) string {
	if errs == 0 && ignored == 0 {
		return lipgloss.NewStyle().Foreground(SuccessColor).Render("None")
	}
	if errs > 0 && ignored > 0 {
		return lipgloss.NewStyle().Foreground(ErrorColor).Render(fmt.Sprintf("%d errors, %d skipped", errs, ignored))
	}
	if errs > 0 {
		return lipgloss.NewStyle().Foreground(ErrorColor).Render(fmt.Sprintf("%d errors", errs))
	}
	return lipgloss.NewStyle().Foreground(WarningColor).Render(fmt.Sprintf("%d skipped", ignored))
}

func (m Model) renderCompletionMenu(width int) string {
	options := m.completionMenuItems()
	var s strings.Builder

	menuWidth := availableWidth(width, 0, 18, 34)
	contentWidth := maxInt(1, menuWidth-2)
	for i, opt := range options {
		label := completionMenuLabel(opt, menuWidth)
		style := lipgloss.NewStyle().Width(contentWidth).MaxWidth(contentWidth).PaddingLeft(2)
		if i == m.SelectedMenuIndex {
			s.WriteString(style.Foreground(SecondaryColor).Bold(true).Render("▶ "+label) + "\n")
		} else {
			s.WriteString(style.Foreground(GrayColor).Render("  "+label) + "\n")
		}
	}

	return s.String()
}

func completionMenuLabel(item completionMenuItem, width int) string {
	if width >= 26 {
		return item.Label
	}

	switch item.Action {
	case CompletionActionOpenDir:
		return "Open Output Folder"
	case CompletionActionViewLog:
		return "View Detailed Log"
	default:
		return item.Label
	}
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

func timeSince(t time.Time) string {
	return fmt.Sprintf("%ds", int(time.Since(t).Seconds()))
}

func availableWidth(viewportWidth, margin, minWidth, maxWidth int) int {
	width := viewportWidth - margin
	if width <= 0 {
		width = viewportWidth
	}
	if width <= 0 {
		return minWidth
	}
	if width < minWidth {
		return width
	}
	return clampInt(width, minWidth, maxWidth)
}

func clampInt(value, minValue, maxValue int) int {
	if value < minValue {
		return minValue
	}
	if value > maxValue {
		return maxValue
	}
	return value
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}
