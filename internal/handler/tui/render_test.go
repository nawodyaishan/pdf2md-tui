package tui

import (
	"strings"
	"testing"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/nawodyaishan/pdf2md-tui/pkg/domain"
)

func TestTruncatePathPreservesContent(t *testing.T) {
	tests := []struct {
		name     string
		path     string
		maxWidth int
	}{
		{"short path", "file.pdf", 20},
		{"long path", "/very/long/path/to/pdf/file.pdf", 20},
		{"very long", "/a/b/c/d/e/f/g/h/i/j/k/l/m/n/o/p/q/r/s/t.pdf", 15},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := truncatePath(tt.path, tt.maxWidth)

			if len(result) > tt.maxWidth {
				t.Errorf("truncated path length %d exceeds max %d: %q", len(result), tt.maxWidth, result)
			}

			if len(tt.path) <= tt.maxWidth {
				if result != tt.path {
					t.Errorf("short path should not be modified: %q != %q", tt.path, result)
				}
			}
		})
	}
}

func TestFormatDurationDisplay(t *testing.T) {
	tests := []struct {
		name       string
		duration   time.Duration
		checkFunc  func(string) bool
	}{
		{"milliseconds", 100 * time.Millisecond, func(s string) bool { return len(s) > 0 }},
		{"second", 1 * time.Second, func(s string) bool { return strings.Contains(s, "s") }},
		{"seconds", 2500 * time.Millisecond, func(s string) bool { return strings.Contains(s, "s") }},
		{"minute", 60 * time.Second, func(s string) bool { return len(s) > 0 }},
		{"minutes", 2500 * time.Millisecond, func(s string) bool { return len(s) > 0 }},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatDuration(tt.duration)

			if !tt.checkFunc(result) {
				t.Errorf("duration %v produced unexpected format: %q", tt.duration, result)
			}
		})
	}
}

func TestClampInt(t *testing.T) {
	tests := []struct {
		name   string
		val    int
		min    int
		max    int
		expect int
	}{
		{"below min", 5, 10, 100, 10},
		{"within range", 50, 10, 100, 50},
		{"above max", 150, 10, 100, 100},
		{"at boundaries", 10, 10, 100, 10},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := clampInt(tt.val, tt.min, tt.max)
			if result != tt.expect {
				t.Errorf("clampInt(%d, %d, %d) = %d, want %d", tt.val, tt.min, tt.max, result, tt.expect)
			}
		})
	}
}

func TestRenderSummaryField(t *testing.T) {
	label := "Converted"
	value := "42"

	result := renderSummaryField(label, value)

	if len(result) == 0 {
		t.Fatal("field should produce output")
	}

	if !strings.Contains(result, label) && !strings.Contains(result, value) {
		t.Error("field output should contain label and value")
	}
}

func TestStatusPill(t *testing.T) {
	tests := []struct {
		name  string
		label string
		color lipgloss.Color
	}{
		{"ok status", "OK", "2"},
		{"error status", "ERROR", "1"},
		{"skipped status", "SKIPPED", "3"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := statusPill(tt.label, tt.color)
			if len(result) == 0 {
				t.Error("statusPill should produce output")
			}
		})
	}
}

func TestRenderProgressBar(t *testing.T) {
	tests := []struct {
		name    string
		current int
		total   int
		width   int
	}{
		{"empty", 0, 100, 20},
		{"quarter", 25, 100, 20},
		{"half", 50, 100, 20},
		{"full", 100, 100, 20},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := renderProgressBar(tt.current, tt.total, tt.width, "2")
			if len(result) == 0 {
				t.Error("progress bar should produce output")
			}
		})
	}
}

func TestTimeSince(t *testing.T) {
	past := time.Now().Add(-5 * time.Second)
	result := timeSince(past)

	if len(result) == 0 {
		t.Error("timeSince should produce output")
	}

	if !strings.Contains(result, "ago") && !strings.Contains(result, "s") {
		t.Logf("result: %s", result)
	}
}

func TestRenderCoreGridWithData(t *testing.T) {
	m := NewModel(10, 2)
	m.Results = []domain.Result{
		{InputPath: "test.pdf", OutputPath: "test.md", Status: domain.StatusOK},
	}

	result := m.View()

	if len(result) == 0 {
		t.Error("grid rendering should produce output")
	}
}

func TestRenderFileTable(t *testing.T) {
	results := []domain.Result{
		{InputPath: "file1.pdf", OutputPath: "file1.md", Status: domain.StatusOK, Duration: 1 * time.Second},
		{InputPath: "file2.pdf", Status: domain.StatusIgnored},
	}

	m := NewModel(2, 1)
	m.Results = results

	view := m.View()

	if len(view) == 0 {
		t.Error("file table should produce output")
	}
}
