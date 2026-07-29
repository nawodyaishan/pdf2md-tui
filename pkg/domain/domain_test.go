package domain

import (
	"errors"
	"testing"
	"time"
)

func TestNewConfig(t *testing.T) {
	cfg := NewConfig()
	if cfg == nil {
		t.Fatal("NewConfig returned nil")
	}
	if cfg.DateFormat != "2006-01-02" {
		t.Errorf("expected DateFormat to be '2006-01-02', got %q", cfg.DateFormat)
	}
}

func TestExtractionProfileConstants(t *testing.T) {
	tests := []struct {
		name     string
		profile  ExtractionProfile
		expected string
	}{
		{"Default profile", ProfileDefault, "default"},
		{"Academic profile", ProfileAcademic, "academic"},
		{"Legacy profile", ProfileLegacy, "legacy"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if string(tt.profile) != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, string(tt.profile))
			}
		})
	}
}

func TestStatusConstants(t *testing.T) {
	if StatusOK != 0 {
		t.Errorf("expected StatusOK to be 0, got %d", StatusOK)
	}
	if StatusIgnored != 1 {
		t.Errorf("expected StatusIgnored to be 1, got %d", StatusIgnored)
	}
	if StatusError != 2 {
		t.Errorf("expected StatusError to be 2, got %d", StatusError)
	}
}

func TestIsSkippablePreflightError(t *testing.T) {
	if !IsSkippablePreflightError(ErrRequiresOCR) {
		t.Fatal("expected OCR pre-flight error to be skippable")
	}
	if !IsSkippablePreflightError(ErrCorruptedEncoding) {
		t.Fatal("expected corrupted encoding pre-flight error to be skippable")
	}
	if IsSkippablePreflightError(errors.New("boom")) {
		t.Fatal("unexpected generic error classified as skippable")
	}
}

func TestBlockTypeConstants(t *testing.T) {
	if BlockTypeText != "text" {
		t.Errorf("expected BlockTypeText to be 'text', got %q", BlockTypeText)
	}
	if BlockTypeTable != "table" {
		t.Errorf("expected BlockTypeTable to be 'table', got %q", BlockTypeTable)
	}
}

func TestResult(t *testing.T) {
	result := Result{
		InputPath:   "/test/input.pdf",
		OutputPath:  "/test/output.md",
		InputBytes:  1024,
		OutputBytes: 512,
		Duration:    time.Second,
		Status:      StatusOK,
	}
	if result.InputPath != "/test/input.pdf" {
		t.Errorf("InputPath mismatch")
	}
	if result.Status != StatusOK {
		t.Errorf("Status mismatch")
	}
}

func TestTableData(t *testing.T) {
	table := TableData{
		Rows: [][]string{
			{"A", "B"},
			{"1", "2"},
		},
	}
	if len(table.Rows) != 2 {
		t.Errorf("expected 2 rows, got %d", len(table.Rows))
	}
}

func TestExtractedImage(t *testing.T) {
	img := ExtractedImage{
		PageNumber: 1,
		Path:       "image_1.png",
	}
	if img.PageNumber != 1 {
		t.Errorf("PageNumber mismatch")
	}
	if img.Path != "image_1.png" {
		t.Errorf("Path mismatch")
	}
}

func TestPageAnalysis(t *testing.T) {
	analysis := PageAnalysis{
		CharCount:  100,
		XObjectCnt: 5,
	}
	if analysis.CharCount != 100 {
		t.Errorf("CharCount mismatch")
	}
	if analysis.XObjectCnt != 5 {
		t.Errorf("XObjectCnt mismatch")
	}
}

func TestSummary(t *testing.T) {
	summary := Summary{
		Successful: 5,
		Skipped:   2,
		Failed:    1,
		ElapsedMs:  5000,
		Files: []FileSummary{
			{
				Input:       "test.pdf",
				Output:      "test.md",
				Status:      "ok",
				InputBytes:  1024,
				OutputBytes: 512,
			},
		},
	}
	if summary.Successful != 5 {
		t.Errorf("Successful count mismatch")
	}
	if len(summary.Files) != 1 {
		t.Errorf("Files count mismatch")
	}
}
