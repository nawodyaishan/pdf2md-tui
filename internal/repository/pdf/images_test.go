package converter

import (
	"os"
	"path/filepath"
	"testing"
)

func TestExtractImages_NonExistentFile(t *testing.T) {
	outDir := t.TempDir()
	
	_, err := ExtractImages("does_not_exist.pdf", outDir)
	if err == nil {
		t.Fatal("expected error for non-existent file, got nil")
	}
}

// Full testing will rely on actual test PDFs (e.g. Newsletter.pdf) in integration tests.
func TestExtractImages_WithImages(t *testing.T) {
	// Let's assume we have testdata/Newsletter.pdf relative to module root.
	// We construct path to it.
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	
	// Assuming tests run in internal/converter
	pdfPath := filepath.Join(cwd, "..", "..", "testdata", "Newsletter.pdf")
	if _, err := os.Stat(pdfPath); os.IsNotExist(err) {
		t.Skip("Newsletter.pdf not found, skipping image extraction test")
	}

	outDir := t.TempDir()
	
	images, err := ExtractImages(pdfPath, outDir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	
	if len(images) == 0 {
		t.Error("expected to extract images, got 0")
	}
	
	// Ensure paths are well-formed
	for _, img := range images {
		if img.PageNumber <= 0 {
			t.Errorf("expected positive page number, got %d", img.PageNumber)
		}
		if img.Path == "" {
			t.Errorf("expected non-empty image path")
		}
	}
}
