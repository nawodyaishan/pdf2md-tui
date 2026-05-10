package cli

import (
	"bytes"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/nawodyaishan/pdf2md-tui/pkg/domain"
)

func TestTextOutputDeterministic(t *testing.T) {
	results := []domain.Result{
		{
			InputPath:   "a.pdf",
			OutputPath:  "a.md",
			Status:      domain.StatusOK,
			InputBytes:  1024,
			OutputBytes: 256,
			Duration:    1 * time.Second,
		},
	}

	totals := conversionTotals{
		converted:   1,
		duration:    1 * time.Second,
		inputBytes:  1024,
		outputBytes: 256,
	}

	sys := domain.SysInfo{
		CPUUsage:   30.0,
		MemoryUsed: 100000000,
		MemoryPct:  15.0,
	}

	buf1 := &bytes.Buffer{}
	printTextSummary(buf1, results, totals, sys)

	buf2 := &bytes.Buffer{}
	printTextSummary(buf2, results, totals, sys)

	out1 := buf1.String()
	out2 := buf2.String()

	if out1 != out2 {
		t.Errorf("output should be deterministic\nFirst:\n%s\nSecond:\n%s", out1, out2)
	}
}

func TestTextSummaryWithMultipleFiles(t *testing.T) {
	results := []domain.Result{
		{
			InputPath:   "file1.pdf",
			OutputPath:  "file1.md",
			Status:      domain.StatusOK,
			InputBytes:  2048,
			OutputBytes: 512,
			Duration:    2 * time.Second,
		},
		{
			InputPath:   "file2.pdf",
			OutputPath:  "file2.md",
			Status:      domain.StatusOK,
			InputBytes:  1024,
			OutputBytes: 256,
			Duration:    1 * time.Second,
		},
		{
			InputPath: "file3.pdf",
			Status:    domain.StatusIgnored,
			Err:       domain.ErrRequiresOCR,
		},
		{
			InputPath: "file4.pdf",
			Status:    domain.StatusError,
			Err:       fmt.Errorf("corrupted"),
		},
	}

	totals := conversionTotals{
		converted:   2,
		ignored:     1,
		errCount:    1,
		duration:    3 * time.Second,
		inputBytes:  3072,
		outputBytes: 768,
	}

	sys := domain.SysInfo{
		CPUUsage:   50.0,
		MemoryUsed: 256000000,
		MemoryPct:  50.0,
	}

	buf := &bytes.Buffer{}
	printTextSummary(buf, results, totals, sys)

	output := buf.String()
	if output == "" {
		t.Fatal("summary should produce output")
	}

	if !strings.Contains(output, "Converted") {
		t.Error("output should contain 'Converted'")
	}
	if !strings.Contains(output, "2") {
		t.Error("output should show 2 converted files")
	}
	if !strings.Contains(output, "Converted files") || !strings.Contains(output, "file1.pdf -> file1.md") {
		t.Error("output should list converted files with output paths")
	}
	if !strings.Contains(output, "Skipped files") || !strings.Contains(output, "document requires OCR") {
		t.Error("output should list skipped files with reasons")
	}
}

func TestTextSummaryWithErrors(t *testing.T) {
	results := []domain.Result{
		{
			InputPath: "bad.pdf",
			Status:    domain.StatusError,
			Err:       fmt.Errorf("file corrupted"),
		},
		{
			InputPath: "invalid.pdf",
			Status:    domain.StatusError,
			Err:       fmt.Errorf("invalid PDF structure"),
		},
	}

	totals := conversionTotals{
		converted: 0,
		errCount:  2,
		duration:  2 * time.Second,
	}

	sys := domain.SysInfo{}

	buf := &bytes.Buffer{}
	printTextSummary(buf, results, totals, sys)

	output := buf.String()
	if !strings.Contains(output, "Failed files") {
		t.Error("output should include failed files section")
	}
	if !strings.Contains(output, "bad.pdf") {
		t.Error("output should show failed file names")
	}
}

func TestSummarizeResultsAccuracy(t *testing.T) {
	results := []domain.Result{
		{Status: domain.StatusOK, InputBytes: 1000, OutputBytes: 500, Duration: 1 * time.Second},
		{Status: domain.StatusOK, InputBytes: 2000, OutputBytes: 1000, Duration: 2 * time.Second},
		{Status: domain.StatusIgnored},
		{Status: domain.StatusIgnored},
		{Status: domain.StatusError, Err: fmt.Errorf("test")},
	}

	summary := summarizeResults(results)

	if summary.converted != 2 {
		t.Errorf("expected 2 converted, got %d", summary.converted)
	}
	if summary.ignored != 2 {
		t.Errorf("expected 2 ignored, got %d", summary.ignored)
	}
	if summary.errCount != 1 {
		t.Errorf("expected 1 error, got %d", summary.errCount)
	}
	if summary.inputBytes != 3000 {
		t.Errorf("expected 3000 input bytes, got %d", summary.inputBytes)
	}
	if summary.outputBytes != 1500 {
		t.Errorf("expected 1500 output bytes, got %d", summary.outputBytes)
	}
	if summary.duration != 3*time.Second {
		t.Errorf("expected 3s duration, got %v", summary.duration)
	}
}

func TestMergeResultsCombinesResults(t *testing.T) {
	preflight := []domain.Result{
		{InputPath: "ignored.pdf", Status: domain.StatusIgnored},
		{InputPath: "ignored2.pdf", Status: domain.StatusIgnored},
	}
	live := []domain.Result{
		{InputPath: "converted1.pdf", Status: domain.StatusOK},
		{InputPath: "converted2.pdf", Status: domain.StatusOK},
		{InputPath: "error.pdf", Status: domain.StatusError, Err: fmt.Errorf("test")},
	}

	merged := mergeResults(preflight, live)

	if len(merged) != 5 {
		t.Errorf("expected 5 total results, got %d", len(merged))
	}

	// Verify order: preflight first, then live
	if merged[0].InputPath != "ignored.pdf" {
		t.Error("first result should be from preflight")
	}
	if merged[2].InputPath != "converted1.pdf" {
		t.Error("live results should come after preflight")
	}
}
