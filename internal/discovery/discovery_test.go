package discovery

import (
	"os"
	"path/filepath"
	"testing"
)

func TestFindPDFs(t *testing.T) {
	// Setup a temporary directory structure for testing
	tempDir, err := os.MkdirTemp("", "pdf-discovery-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create test files
	files := []string{
		"file1.pdf",
		"file2.PDF",  // test case insensitivity
		"not_pdf.txt",
		"sub/file3.pdf",
		"sub/not_pdf.txt",
	}

	for _, f := range files {
		path := filepath.Join(tempDir, f)
		err := os.MkdirAll(filepath.Dir(path), 0755)
		if err != nil {
			t.Fatalf("Failed to create sub dir: %v", err)
		}
		if err := os.WriteFile(path, []byte("dummy content"), 0644); err != nil {
			t.Fatalf("Failed to write file: %v", err)
		}
	}

	tests := []struct {
		name      string
		dir       string
		recursive bool
		wantCount int
		wantErr   bool
	}{
		{
			name:      "Flat discovery",
			dir:       tempDir,
			recursive: false,
			wantCount: 2, // file1.pdf, file2.PDF
			wantErr:   false,
		},
		{
			name:      "Recursive discovery",
			dir:       tempDir,
			recursive: true,
			wantCount: 3, // file1.pdf, file2.PDF, sub/file3.pdf
			wantErr:   false,
		},
		{
			name:      "Non-existent directory",
			dir:       filepath.Join(tempDir, "does-not-exist"),
			recursive: false,
			wantCount: 0,
			wantErr:   true,
		},
		{
			name:      "Empty directory",
			dir:       filepath.Join(tempDir, "empty"),
			recursive: true,
			wantCount: 0,
			wantErr:   false,
		},
	}

	// Create empty dir for test
	os.MkdirAll(filepath.Join(tempDir, "empty"), 0755)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := FindPDFs(tt.dir, tt.recursive)
			if (err != nil) != tt.wantErr {
				t.Errorf("FindPDFs() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if len(got) != tt.wantCount {
				t.Errorf("FindPDFs() count = %v, want %v", len(got), tt.wantCount)
			}
		})
	}
}

func TestFindPDFs_PermissionError(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("Skipping permission test as root")
	}

	tempDir, err := os.MkdirTemp("", "pdf-discovery-perm-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	subDir := filepath.Join(tempDir, "restricted")
	if err := os.MkdirAll(subDir, 0000); err != nil { // No permissions
		t.Fatalf("Failed to create restricted dir: %v", err)
	}
	defer os.Chmod(subDir, 0755) // Ensure we can clean it up

	_, err = FindPDFs(tempDir, true)
	if err == nil {
		t.Error("Expected permission error, got nil")
	}
}
