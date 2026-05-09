package service

import (
	"flag"
	"os"
	"path/filepath"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/nawodyaishan/pdf2md-tui/pkg/domain"
	"github.com/nawodyaishan/pdf2md-tui/pkg/repository/discovery"
	"github.com/nawodyaishan/pdf2md-tui/pkg/repository/pdf"
	"github.com/nawodyaishan/pdf2md-tui/pkg/repository/storage"
)

var update = flag.Bool("update", false, "update golden master files")

// TestQA_GoldenMaster performs snapshot testing for conversion fidelity.
func TestQA_GoldenMaster(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping snapshot tests")
	}

	testdataDir := "../../testdata/devops_project"
	goldenDir := filepath.Join(testdataDir, "golden")
	if err := os.MkdirAll(goldenDir, 0755); err != nil {
		t.Fatalf("Failed to create golden dir: %v", err)
	}

	cfg := domain.NewConfig()
	cfg.StripNoise = true
	repo := pdf.NewParser()
	store := storage.NewStorage()
	conv := NewConverterService(cfg, store, repo, nil)

	files, err := discovery.FindPDFs(testdataDir, false)
	if err != nil {
		t.Fatalf("Failed to discover test PDFs: %v", err)
	}

	for _, pdfPath := range files {
		t.Run(filepath.Base(pdfPath), func(t *testing.T) {
			tempOut := t.TempDir()
			res := conv.Convert(pdfPath, tempOut)
			if res.Status != domain.StatusOK {
				t.Fatalf("Conversion failed: %v", res.Err)
			}

			actual, _ := os.ReadFile(res.OutputPath)
			goldenFile := filepath.Join(goldenDir, filepath.Base(pdfPath)+".golden.md")

			if *update {
				if err := os.WriteFile(goldenFile, actual, 0644); err != nil {
					t.Fatalf("Failed to write golden file: %v", err)
				}
				return
			}

			expected, err := os.ReadFile(goldenFile)
			if err != nil {
				t.Fatalf("Golden file missing. Run with -update: %v", err)
			}

			if diff := cmp.Diff(string(expected), string(actual)); diff != "" {
				t.Errorf("Snapshot mismatch (-expected +actual):\n%s", diff)
			}
		})
	}
}
