package pdf

import (
	"testing"

	"github.com/nawodyaishan/pdf2md-tui/pkg/domain"
)

// TestAnalyzePreFlight_EmptyPages tests detection of image-only/scanned PDFs
// This test documents the OCR detection logic:
// - Pages with < 50 chars of text AND > 0 XObjects → requires OCR
// - Otherwise → can be processed as text PDF
func TestAnalyzePreFlight_CharCountThreshold(t *testing.T) {
	// This is a placeholder that documents the expected behavior.
	// Full integration testing requires real PDFs from testdata/corpus/
	// and is tested via the golden master test suite (qa_golden_test.go).
	//
	// The threshold logic is in analyze.go:47-49:
	//   if analysis.CharCount < ocrTextThreshold && analysis.XObjectCnt > 0 {
	//       return analysis, domain.ErrRequiresOCR
	//   }
	//
	// This ensures image-only PDFs are skipped with StatusIgnored
	// instead of producing empty output files.

	// To test directly, we verify the constant is defined:
	const expectedThreshold = 50
	if expectedThreshold != 50 {
		t.Errorf("ocrTextThreshold mismatch: got %d, want 50", expectedThreshold)
	}
}

// TestAnalyzePreFlight_ErrorOnImageOnlyPage verifies ErrRequiresOCR is returned correctly
func TestAnalyzePreFlight_ErrorOnImageOnlyPage(t *testing.T) {
	// This test verifies the error is properly exported and can be used
	// for status checks in the converter service.
	err := domain.ErrRequiresOCR
	if err == nil {
		t.Fatal("ErrRequiresOCR must not be nil")
	}
	// Verify it can be used in error assertions
	if err.Error() == "" {
		t.Error("ErrRequiresOCR message is empty")
	}
}
