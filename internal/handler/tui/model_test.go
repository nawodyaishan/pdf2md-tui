package tui

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/nawodyaishan/pdf2md-tui/pkg/domain"
)

func TestCompleteStateQSetsExitAction(t *testing.T) {
	model := NewModel(1, 1)
	model.Complete = true
	model.SelectedMenuIndex = 0

	updated, cmd := model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
	got := updated.(Model)

	if got.CompletionAction != CompletionActionExit {
		t.Fatalf("expected exit action, got %q", got.CompletionAction)
	}
	if got.SelectedMenuIndex != 0 {
		t.Fatalf("expected menu index to remain 0, got %d", got.SelectedMenuIndex)
	}
	if cmd == nil {
		t.Fatal("expected quit command")
	}
}

func TestCompleteStateEnterSelectsHighlightedAction(t *testing.T) {
	model := NewModel(1, 1)
	model.Complete = true
	model.Results = []domain.Result{{Status: domain.StatusError, Err: domain.ErrRequiresOCR}}
	model.SelectedMenuIndex = 1

	updated, cmd := model.Update(tea.KeyMsg{Type: tea.KeyEnter})
	got := updated.(Model)

	if got.CompletionAction != CompletionActionViewLog {
		t.Fatalf("expected %q, got %q", CompletionActionViewLog, got.CompletionAction)
	}
	if cmd == nil {
		t.Fatal("expected quit command")
	}
}

func TestCompletionMenuBounds(t *testing.T) {
	noFailure := NewModel(1, 1)
	noFailure.Complete = true

	updated, _ := noFailure.Update(tea.KeyMsg{Type: tea.KeyDown})
	updated, _ = updated.(Model).Update(tea.KeyMsg{Type: tea.KeyDown})
	got := updated.(Model)

	if got.SelectedMenuIndex != 1 {
		t.Fatalf("expected selection to clamp at 1, got %d", got.SelectedMenuIndex)
	}

	withFailure := NewModel(1, 1)
	withFailure.Complete = true
	withFailure.Results = []domain.Result{{Status: domain.StatusError, Err: domain.ErrRequiresOCR}}

	updated, _ = withFailure.Update(tea.KeyMsg{Type: tea.KeyDown})
	updated, _ = updated.(Model).Update(tea.KeyMsg{Type: tea.KeyDown})
	updated, _ = updated.(Model).Update(tea.KeyMsg{Type: tea.KeyDown})
	got = updated.(Model)

	if got.SelectedMenuIndex != 2 {
		t.Fatalf("expected selection to clamp at 2, got %d", got.SelectedMenuIndex)
	}
}

func TestActiveConversionIgnoresQ(t *testing.T) {
	model := NewModel(2, 1)

	updated, cmd := model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
	got := updated.(Model)

	if got.Complete {
		t.Fatal("active conversion should remain incomplete")
	}
	if got.CompletionAction != CompletionActionNone {
		t.Fatalf("expected no completion action, got %q", got.CompletionAction)
	}
	if cmd != nil {
		t.Fatal("expected no quit command during active conversion")
	}
}

func TestRenderSummaryPanelStacksWithinNarrowWidth(t *testing.T) {
	model := NewModel(3, 2)
	model.Complete = true
	model.Results = []domain.Result{
		{Status: domain.StatusOK, InputBytes: 1024, OutputBytes: 512, InputTokens: 10, OutputTokens: 5},
		{Status: domain.StatusOK, InputBytes: 2048, OutputBytes: 1024, InputTokens: 20, OutputTokens: 10},
		{Status: domain.StatusIgnored},
	}
	model.PeakMemory = 4096

	rendered := model.renderSummaryPanel(54)

	if lipgloss.Width(rendered) > 54 {
		t.Fatalf("expected summary width <= 54, got %d", lipgloss.Width(rendered))
	}
	if !strings.Contains(rendered, "Session Summary") || !strings.Contains(rendered, "Files Processed") {
		t.Fatal("expected responsive summary to retain the redesigned summary structure")
	}
}

func TestRenderCompletionMenuUsesFullLabelOnNarrowWidth(t *testing.T) {
	model := NewModel(1, 1)
	model.Complete = true
	model.Results = []domain.Result{{Status: domain.StatusError, Err: domain.ErrRequiresOCR}}

	rendered := model.renderCompletionMenu(24)

	if !strings.Contains(rendered, "Open Output") {
		t.Fatal("expected menu to keep the open output action visible")
	}
	if !strings.Contains(rendered, "View Detailed Log") {
		t.Fatal("expected menu to show failure action")
	}
	if lipgloss.Width(rendered) > 24 {
		t.Fatalf("expected menu width <= 24, got %d", lipgloss.Width(rendered))
	}
}
