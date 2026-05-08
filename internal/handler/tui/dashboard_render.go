package tui

import (
	"fmt"
	"runtime"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
)

func (m Model) renderDashboard() string {
	if m.Complete {
		return m.renderCompletionView()
	}

	// 1. Stats Panel (Left)
	stats := m.renderStats()
	// 2. Core Grid Panel (Right)
	cores := m.renderCoreGrid()

	// 3. File Table (Bottom)
	table := m.renderFileTable()

	topRow := lipgloss.JoinHorizontal(lipgloss.Top,
		BorderStyle.Width(m.width/2-2).Height(10).Render(stats),
		BorderStyle.Width(m.width/2-2).Height(10).Render(cores),
	)

	return lipgloss.JoinVertical(lipgloss.Left,
		topRow,
		BorderStyle.Width(m.width-4).Height(m.height-22).Render(table),
	)
}

func (m Model) renderStats() string {
	cpuUsage := m.SysInfo.CPUUsage
	memPct := m.SysInfo.MemoryPct

	cpuBar := renderProgressBar(int(cpuUsage), 100, 20, PrimaryColor)
	memBar := renderProgressBar(int(memPct), 100, 20, SecondaryColor)

	// Batch Progress
	batchBar := renderProgressBar(m.CurrentFile, m.TotalFiles, 20, AccentColor)

	return fmt.Sprintf(
		"%s %s %s\n%s %s %s\n%s %s %s\n\n%s %s",
		MetricLabelStyle.Render("CPU Load"), cpuBar, MetricValueStyle.Render(fmt.Sprintf("%.1f%%", cpuUsage)),
		MetricLabelStyle.Render("Memory"), memBar, MetricValueStyle.Render(fmt.Sprintf("%.1f%%", memPct)),
		MetricLabelStyle.Render("Batch"), batchBar, MetricValueStyle.Render(fmt.Sprintf("%d/%d", m.CurrentFile, m.TotalFiles)),
		MetricLabelStyle.Render("Uptime"), MetricValueStyle.Render(timeSince(m.StartTime)),
	)
}

func (m Model) renderCoreGrid() string {
	totalCores := runtime.NumCPU()
	var grid strings.Builder
	grid.WriteString(lipgloss.NewStyle().Foreground(SecondaryColor).Bold(true).Render("ENGINE CORE MAP") + "\n\n")

	cols := 8
	if m.width < 100 {
		cols = 6
	}

	for i := 0; i < totalCores; i++ {
		style := DimmedStyle
		char := "□"
		if i < m.WorkerCount {
			style = CoreActiveStyle
			char = "▣"
		}
		grid.WriteString(style.Render(" " + char + " "))
		if (i+1)%cols == 0 {
			grid.WriteString("\n")
		}
	}

	return grid.String()
}

func (m Model) renderFileTable() string {
	if len(m.Results) == 0 {
		return "Waiting for workers..."
	}

	var s strings.Builder
	s.WriteString(lipgloss.NewStyle().Bold(true).Render("Recent Activity") + "\n\n")

	count := 0
	for i := len(m.Results) - 1; i >= 0 && count < 10; i-- {
		res := m.Results[i]
		status := CoreActiveStyle.Render("✔")
		if res.Err != nil {
			status = lipgloss.NewStyle().Foreground(SecondaryColor).Render("✘")
		}
		fmt.Fprintf(&s, "%s %-40s %s\n", status, truncatePath(res.InputPath, 40), formatDuration(res.Duration))
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
		color = lipgloss.Color("#00FF00")
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
	summary := m.renderSummaryPanel()
	menu := m.renderCompletionMenu()

	content := lipgloss.JoinVertical(lipgloss.Center,
		summary,
		lipgloss.NewStyle().Height(2).Render(""), // Spacer
		menu,
	)

	return lipgloss.Place(m.width, m.height-5, lipgloss.Center, lipgloss.Center,
		BorderStyle.BorderForeground(lipgloss.Color("#00FF00")).
			Padding(1, 4).
			Render(content),
	)
}

func (m Model) renderSummaryPanel() string {
	var results = m.Results
	var totalIn, totalOut int64
	var totalInTkn, totalOutTkn int
	var errCount int

	for _, r := range results {
		if r.Err != nil {
			errCount++
		} else {
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

	title := lipgloss.NewStyle().Foreground(lipgloss.Color("#00FF00")).Bold(true).Render("📊 CONVERSION SUMMARY")

	avgCPU := 0.0
	if m.sysCount > 0 {
		avgCPU = m.AvgCPU / float64(m.sysCount)
	}

	duration := time.Since(m.StartTime)
	if m.Complete {
		duration = m.FinalDuration
	}

	leftCol := fmt.Sprintf(
		"Files Processed: %d\nErrors / Ignored: %s\nTotal Duration:  %.2fs",
		len(results),
		formatErrorCount(errCount),
		duration.Seconds(),
	)

	midCol := fmt.Sprintf(
		"PDF Source: %s (%d tkn)\nMD Output:  %s (%d tkn)\nEfficiency: ▼ %.1f%%",
		formatBytes(totalIn), totalInTkn,
		formatBytes(totalOut), totalOutTkn,
		savings,
	)

	rightCol := fmt.Sprintf(
		"Cores Utilized: %d/%d\nAvg CPU Load:   %.1f%%\nPeak Memory:    %s (%.1f%%)",
		m.WorkerCount, runtime.NumCPU(),
		avgCPU,
		formatBytes(int64(m.PeakMemory)), m.MaxMemPct,
	)

	stats := lipgloss.JoinHorizontal(lipgloss.Top,
		lipgloss.NewStyle().Width(35).Render(leftCol),
		lipgloss.NewStyle().Width(35).Render(midCol),
		lipgloss.NewStyle().Width(35).Render(rightCol),
	)

	return lipgloss.JoinVertical(lipgloss.Center, title, "\n", stats)
}

func formatErrorCount(errs int) string {
	if errs == 0 {
		return lipgloss.NewStyle().Foreground(lipgloss.Color("#00FF00")).Render("✓ None")
	}
	return lipgloss.NewStyle().Foreground(lipgloss.Color("#FF0000")).Render(fmt.Sprintf("✘ %d", errs))
}

func (m Model) renderCompletionMenu() string {
	options := []string{"📁 Open Output Directory", "🚪 Exit"}
	var s strings.Builder

	menuWidth := 30
	for i, opt := range options {
		style := lipgloss.NewStyle().Width(menuWidth).PaddingLeft(2)
		if i == m.SelectedMenuIndex {
			s.WriteString(style.Foreground(PrimaryColor).Bold(true).Render("▶ "+opt) + "\n")
		} else {
			s.WriteString(style.Foreground(GrayColor).Render("  "+opt) + "\n")
		}
	}

	return s.String()
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
