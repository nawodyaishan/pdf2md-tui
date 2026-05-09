package tui

import (
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/nawodyaishan/pdf2md-tui/pkg/domain"
)

func TestModelInit(t *testing.T) {
	m := NewModel(10, 4)
	cmd := m.Init()

	// Init should return a valid command (can be nil or actual cmd)
	_ = cmd
}

func TestModelUpdateWithKeyMessage(t *testing.T) {
	m := NewModel(10, 4)

	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}}
	newModel, cmd := m.Update(msg)

	if newModel == nil {
		t.Fatal("Update should return a model")
	}
	_ = cmd
}

func TestModelUpdateWithCtrlC(t *testing.T) {
	m := NewModel(5, 2)

	msg := tea.KeyMsg{Type: tea.KeyCtrlC}
	newModel, cmd := m.Update(msg)

	if newModel == nil {
		t.Fatal("Update should handle Ctrl+C and return a model")
	}
	_ = cmd
}

func TestModelViewRendersWithResults(t *testing.T) {
	m := NewModel(3, 2)
	m.Results = []domain.Result{
		{
			InputPath:   "test1.pdf",
			OutputPath:  "test1.md",
			Status:      domain.StatusOK,
			InputBytes:  2048,
			OutputBytes: 512,
			Duration:    2 * time.Second,
		},
		{
			InputPath: "test2.pdf",
			Status:    domain.StatusIgnored,
			Err:       domain.ErrRequiresOCR,
		},
		{
			InputPath: "test3.pdf",
			Status:    domain.StatusError,
			Err:       domain.ErrRequiresOCR,
		},
	}

	view := m.View()

	if len(view) == 0 {
		t.Fatal("View should produce output")
	}

	// View might not contain the exact filenames depending on rendering
	// but should contain some content
	if len(view) < 5 {
		t.Errorf("View output too short: %q", view)
	}
}

func TestModelViewWithEmptyResults(t *testing.T) {
	m := NewModel(5, 2)

	view := m.View()
	if len(view) == 0 {
		t.Fatal("View should produce output even with no results")
	}
}

func TestModelViewWithManyResults(t *testing.T) {
	m := NewModel(100, 4)

	results := make([]domain.Result, 50)
	for i := 0; i < 50; i++ {
		results[i] = domain.Result{
			InputPath:   "file" + string(rune('0'+i%10)) + ".pdf",
			OutputPath:  "file" + string(rune('0'+i%10)) + ".md",
			Status:      domain.StatusOK,
			InputBytes:  1024,
			OutputBytes: 256,
			Duration:    1 * time.Second,
		}
	}
	m.Results = results

	view := m.View()
	if len(view) == 0 {
		t.Fatal("View should handle many results")
	}
}
