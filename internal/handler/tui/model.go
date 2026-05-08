package tui

import (
	"time"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/nawodyaishan/pdf2md-tui/internal/domain"
)

// Messages
type StatusUpdateMsg struct {
	Index  int
	Status string
	Result *domain.Result
}

type SysInfoMsg struct {
	Info domain.SysInfo
}

type BatchCompleteMsg struct {
	Results []domain.Result
}

// Model represents the TUI state
type Model struct {
	// State
	TotalFiles    int
	CurrentFile   int
	WorkerCount   int
	Results       []domain.Result
	SysInfo       domain.SysInfo
	StartTime     time.Time
	FinalDuration time.Duration
	Complete      bool

	// Components
	spinner spinner.Model
	width   int
	height  int

	// Resource Tracking
	PeakCPU    float64
	PeakMemory uint64
	MaxMemPct  float64
	AvgCPU     float64
	sysCount   int

	// Menu
	SelectedMenuIndex int
}

func NewModel(total, workers int) Model {
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(PrimaryColor)

	return Model{
		TotalFiles:  total,
		WorkerCount: workers,
		Results:     make([]domain.Result, 0, total),
		StartTime:   time.Now(),
		spinner:     s,
	}
}

func (m Model) Init() tea.Cmd {
	return m.spinner.Tick
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case tea.KeyMsg:
		if m.Complete {
			switch msg.String() {
			case "up", "k":
				if m.SelectedMenuIndex > 0 {
					m.SelectedMenuIndex--
				}
			case "down", "j":
				if m.SelectedMenuIndex < 1 {
					m.SelectedMenuIndex++
				}
			case "enter", " ":
				return m, tea.Quit
			case "q", "ctrl+c":
				return m, tea.Quit
			}
			return m, nil
		}
		if msg.String() == "q" || msg.String() == "ctrl+c" {
			return m, tea.Quit
		}

	case spinner.TickMsg:
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd

	case StatusUpdateMsg:
		m.CurrentFile++
		if msg.Result != nil {
			m.Results = append(m.Results, *msg.Result)
		}
		return m, nil

	case SysInfoMsg:
		if !m.Complete {
			m.SysInfo = msg.Info

			// Update Peaks
			if msg.Info.CPUUsage > m.PeakCPU {
				m.PeakCPU = msg.Info.CPUUsage
			}
			if msg.Info.MemoryUsed > m.PeakMemory {
				m.PeakMemory = msg.Info.MemoryUsed
			}
			if msg.Info.MemoryPct > m.MaxMemPct {
				m.MaxMemPct = msg.Info.MemoryPct
			}

			// Accumulate for average
			m.AvgCPU += msg.Info.CPUUsage
			m.sysCount++
		}
		return m, nil

	case BatchCompleteMsg:
		m.Complete = true
		m.FinalDuration = time.Since(m.StartTime)
		if len(msg.Results) > 0 {
			m.Results = msg.Results
		}
		return m, nil
	}

	return m, nil
}

func (m Model) View() string {
	if m.width == 0 {
		return "Initializing..."
	}

	header := m.renderHeader()
	dashboard := m.renderDashboard()
	footer := m.renderFooter()

	return lipgloss.JoinVertical(lipgloss.Left,
		header,
		dashboard,
		footer,
	)
}

func (m Model) renderHeader() string {
	title := TitleStyle.Render(" PDF2MD-TUI ")
	status := " Processing..."
	if m.Complete {
		status = lipgloss.NewStyle().Foreground(lipgloss.Color("#00FF00")).Bold(true).Render(" SUCCESS ")
	}

	return HeaderStyle.Width(m.width).Render(
		lipgloss.JoinHorizontal(lipgloss.Center, title, " ", status),
	)
}

func (m Model) renderFooter() string {
	if m.Complete {
		msg := " CONVERSION COMPLETE • PRESS ANY KEY FOR SUMMARY "
		return FooterStyle.Width(m.width).Render(
			SuccessFooterStyle.Render(msg),
		)
	}
	return FooterStyle.Width(m.width).Render(" [q] Quit • [l] Logs • [o] Open Output ")
}
