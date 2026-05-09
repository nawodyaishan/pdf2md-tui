package cli

import (
	"bytes"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/nawodyaishan/pdf2md-tui/pkg/domain"
)

func TestPrintTextSummaryFormat(t *testing.T) {
	results := []domain.Result{
		{
			InputPath:   "/path/to/file1.pdf",
			OutputPath:  "/path/to/file1.md",
			Status:      domain.StatusOK,
			InputBytes:  2048,
			OutputBytes: 512,
			Duration:    2 * time.Second,
		},
		{
			InputPath:   "/path/to/file2.pdf",
			OutputPath:  "/path/to/file2.md",
			Status:      domain.StatusOK,
			InputBytes:  1024,
			OutputBytes: 256,
			Duration:    1 * time.Second,
		},
		{
			InputPath: "/path/to/file3.pdf",
			Status:    domain.StatusIgnored,
			Err:       domain.ErrRequiresOCR,
		},
	}

	totals := conversionTotals{
		converted:   2,
		ignored:     1,
		errCount:    0,
		duration:    3 * time.Second,
		inputBytes:  3072,
		outputBytes: 768,
	}

	sys := domain.SysInfo{
		CPUUsage:    45.5,
		MemoryUsed:  512000000,
		MemoryPct:   25.5,
	}

	buf := &bytes.Buffer{}
	printTextSummary(buf, results, totals, sys)

	output := buf.String()
	if output == "" {
		t.Fatal("text summary should produce non-empty output")
	}

	// Verify summary contains file counts
	if !strings.Contains(output, "2") {
		t.Error("summary should mention 2 converted files")
	}

	// Verify summary displays timing
	if !strings.Contains(output, "3s") && !strings.Contains(output, "3.") {
		t.Logf("output: %s", output)
		t.Error("summary should show total duration")
	}
}

func TestPrintTextSummaryEmptyResults(t *testing.T) {
	buf := &bytes.Buffer{}
	totals := conversionTotals{}
	sys := domain.SysInfo{}
	printTextSummary(buf, []domain.Result{}, totals, sys)

	output := buf.String()
	if output == "" {
		t.Fatal("should produce output even with empty results")
	}
}

func TestPrintTextSummaryWithErrors(t *testing.T) {
	results := []domain.Result{
		{
			InputPath: "/path/to/bad.pdf",
			Status:    domain.StatusError,
			Err:       fmt.Errorf("corrupted PDF"),
		},
	}

	totals := conversionTotals{
		converted:   0,
		errCount:    1,
		duration:    1 * time.Second,
	}

	sys := domain.SysInfo{}

	buf := &bytes.Buffer{}
	printTextSummary(buf, results, totals, sys)

	output := buf.String()
	if !strings.Contains(output, "1") {
		t.Error("summary should show 1 error")
	}
}

func TestPrintJSONSummarySchema(t *testing.T) {
	results := []domain.Result{
		{
			InputPath:   "/test.pdf",
			OutputPath:  "/test.md",
			Status:      domain.StatusOK,
			InputBytes:  100,
			OutputBytes: 50,
			Duration:    1 * time.Second,
		},
	}

	totals := conversionTotals{
		converted:   1,
		ignored:     0,
		errCount:    0,
		duration:    1 * time.Second,
		inputBytes:  100,
		outputBytes: 50,
	}

	// Test that printJSONSummary executes without panicking
	// (It prints to stdout, which is expected behavior)
	printJSONSummary(results, totals)
}

func TestMergeResultsAggregatesCorrectly(t *testing.T) {
	preflightResults := []domain.Result{
		{InputPath: "ignored.pdf", Status: domain.StatusIgnored},
	}
	liveResults := []domain.Result{
		{InputPath: "converted.pdf", Status: domain.StatusOK},
		{InputPath: "error.pdf", Status: domain.StatusError, Err: fmt.Errorf("test error")},
	}

	merged := mergeResults(preflightResults, liveResults)

	if len(merged) != 3 {
		t.Errorf("expected 3 total results, got %d", len(merged))
	}
}

func TestMergeResultsPreservesOrder(t *testing.T) {
	preflight := []domain.Result{
		{InputPath: "ignored.pdf", Status: domain.StatusIgnored},
	}
	live := []domain.Result{
		{InputPath: "converted.pdf", Status: domain.StatusOK},
	}

	merged := mergeResults(preflight, live)

	if len(merged) != 2 {
		t.Fatalf("expected 2 results, got %d", len(merged))
	}
	if merged[0].InputPath != "ignored.pdf" {
		t.Error("preflight results should come first")
	}
	if merged[1].InputPath != "converted.pdf" {
		t.Error("live results should come after")
	}
}

func TestSummarizeResultsAggregation(t *testing.T) {
	results := []domain.Result{
		{Status: domain.StatusOK, InputBytes: 1024, OutputBytes: 512, Duration: 1 * time.Second},
		{Status: domain.StatusOK, InputBytes: 2048, OutputBytes: 1024, Duration: 2 * time.Second},
		{Status: domain.StatusIgnored},
		{Status: domain.StatusError, Err: fmt.Errorf("test error")},
	}

	summary := summarizeResults(results)

	if summary.converted != 2 {
		t.Errorf("expected 2 converted, got %d", summary.converted)
	}
	if summary.ignored != 1 {
		t.Errorf("expected 1 ignored, got %d", summary.ignored)
	}
	if summary.errCount != 1 {
		t.Errorf("expected 1 error, got %d", summary.errCount)
	}
	if summary.inputBytes != 3072 {
		t.Errorf("expected 3072 input bytes, got %d", summary.inputBytes)
	}
	if summary.outputBytes != 1536 {
		t.Errorf("expected 1536 output bytes, got %d", summary.outputBytes)
	}
	if summary.duration != 3*time.Second {
		t.Errorf("expected 3s duration, got %v", summary.duration)
	}
}
