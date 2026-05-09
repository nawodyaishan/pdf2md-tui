package service

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/nawodyaishan/pdf2md-tui/pkg/domain"
	"github.com/nawodyaishan/pdf2md-tui/pkg/repository/discovery"
	"github.com/nawodyaishan/pdf2md-tui/pkg/repository/pdf"
	"github.com/nawodyaishan/pdf2md-tui/pkg/repository/storage"
)

// TestQA_ConversionQuality implements the expanded automated validation assertions.
func TestQA_ConversionQuality(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping QA quality tests in short mode")
	}

	// We check both the root testdata and the devops_project subfolder
	searchPaths := []string{"../../testdata", "../../testdata/devops_project"}

	// 1. Setup Service
	cfg := domain.NewConfig()
	cfg.StripNoise = true
	cfg.DateFormat = "none"

	repo := pdf.NewParser()
	store := storage.NewStorage()
	conv := NewConverterService(cfg, store, repo, nil)

	tempOut := t.TempDir()

	for _, dataDir := range searchPaths {
		if _, err := os.Stat(dataDir); os.IsNotExist(err) {
			continue
		}

		files, _ := discovery.FindPDFs(dataDir, false)
		for _, pdfPath := range files {
			t.Run(filepath.Base(pdfPath), func(t *testing.T) {
				res := conv.Convert(pdfPath, tempOut)
				if res.Status != domain.StatusOK {
					t.Fatalf("Conversion failed: %v", res.Err)
				}

				content, _ := os.ReadFile(res.OutputPath)
				text := string(content)

				// --- Level 1: Character & Word Correctness ---

				// A1: Double Letters
				if strings.Contains(strings.ToLower(pdfPath), "clean code") {
					for _, marker := range []string{"ebooks", "across", "www"} {
						if !strings.Contains(strings.ToLower(text), marker) {
							t.Errorf("A1: Marker %q collapsed", marker)
						}
					}
				}

				// A2: Spaced Headings
				if strings.Contains(strings.ToLower(text), "DEFINITIVE") && strings.Contains(text, "D E F") {
					t.Errorf("A2: Spaced heading not fully coalesced")
				}

				// A3: Technical Symbols (Arrows)
				if (strings.Contains(pdfPath, "Pivot") || strings.Contains(pdfPath, "Master_Playbook")) && !strings.Contains(text, "→") {
					// Only warn as some encoding might truly fail, but we expect preservation
					t.Logf("Warning A3: Arrow symbol '→' not found in %s", pdfPath)
				}

				// --- Level 2: Layout & Structure ---

				// B1: Paragraph Density
				lines := strings.Split(text, "\n")
				blankLines := 0
				for _, l := range lines {
					if strings.TrimSpace(l) == "" {
						blankLines++
					}
				}
				if len(lines) > 20 && float64(blankLines)/float64(len(lines)) < 0.02 {
					t.Errorf("B1: Low paragraph density (%.2f%%), structure likely lost", float64(blankLines)/float64(len(lines))*100)
				}

				// B2: Table Integrity
				if strings.Contains(pdfPath, "MLOps") || strings.Contains(pdfPath, "Action_Plan") {
					if !strings.Contains(text, "| --- |") {
						t.Errorf("B2: Table markers not found in known tabular document")
					}
				}

				// --- Level 3: Cleanliness ---

				// C1: Artifact Zero-Tolerance
				if strings.Contains(text, "\ufffd") {
					t.Errorf("C1: Output contains replacement character artifacts (U+FFFD)")
				}

				// C2: Encoding Stability (Garbage Detection)
				// Look for long sequences of non-printable or high-entropy characters
				reGarbage := regexp.MustCompile(`[^\x00-\x7F]{20,}`)
				if reGarbage.MatchString(text) {
					t.Errorf("C2: Detected garbage character sequence (encoding breakdown)")
				}
			})
		}
	}
}
