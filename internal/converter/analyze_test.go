package converter

import (
	"errors"
	"testing"
)

func TestAnalyzePDF_NonExistentFile(t *testing.T) {
	_, err := AnalyzePDF("does_not_exist.pdf", 3)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if errors.Is(err, ErrRequiresOCR) {
		t.Fatalf("expected os error, got %v", err)
	}
}

// Additional tests for TextRich and Scanned will require synthetic testdata PDFs.
// For now, we test the error paths.
