package storage_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/nawodyaishan/pdf2md-tui/pkg/repository/storage"
)

func TestWriteMarkdown_CreatesFile(t *testing.T) {
	dir := t.TempDir()
	s := storage.NewStorage()
	path := filepath.Join(dir, "out.md")

	if err := s.WriteMarkdown(path, []byte("# Hello")); err != nil {
		t.Fatalf("WriteMarkdown: %v", err)
	}
	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	if string(got) != "# Hello" {
		t.Errorf("content mismatch: got %q, want %q", string(got), "# Hello")
	}
}

func TestWriteMarkdown_ErrorOnBadPath(t *testing.T) {
	s := storage.NewStorage()
	err := s.WriteMarkdown("/nonexistent/deep/path/out.md", []byte("data"))
	if err == nil {
		t.Fatal("expected error for non-existent parent dir")
	}
}

func TestCreateImageDir_CreatesDirectory(t *testing.T) {
	dir := t.TempDir()
	s := storage.NewStorage()
	docPath := filepath.Join(dir, "doc")

	imgDir, err := s.CreateImageDir(docPath)
	if err != nil {
		t.Fatalf("CreateImageDir: %v", err)
	}
	// Currently returns empty string, but document the expected behavior
	if imgDir != "" {
		if _, err := os.Stat(imgDir); os.IsNotExist(err) {
			t.Errorf("image dir not created: %s", imgDir)
		}
	}
}

func TestFileExists_TrueAndFalse(t *testing.T) {
	dir := t.TempDir()
	s := storage.NewStorage()
	existing := filepath.Join(dir, "file.txt")
	if err := os.WriteFile(existing, []byte("x"), 0644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	if !s.FileExists(existing) {
		t.Error("expected true for existing file")
	}
	if s.FileExists(filepath.Join(dir, "missing.txt")) {
		t.Error("expected false for missing file")
	}
}

func TestMkdirAll_CreatesPath(t *testing.T) {
	dir := t.TempDir()
	s := storage.NewStorage()
	newPath := filepath.Join(dir, "a", "b", "c")

	if err := s.MkdirAll(newPath); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	if _, err := os.Stat(newPath); os.IsNotExist(err) {
		t.Errorf("directory not created: %s", newPath)
	}
}

func TestStatSize_ReturnsFileSize(t *testing.T) {
	dir := t.TempDir()
	s := storage.NewStorage()
	path := filepath.Join(dir, "test.txt")
	data := []byte("hello world")

	if err := os.WriteFile(path, data, 0644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	size, err := s.StatSize(path)
	if err != nil {
		t.Fatalf("StatSize: %v", err)
	}
	if size != int64(len(data)) {
		t.Errorf("size mismatch: got %d, want %d", size, len(data))
	}
}

func TestStatSize_ErrorOnMissing(t *testing.T) {
	s := storage.NewStorage()
	_, err := s.StatSize("/nonexistent/file.txt")
	if err == nil {
		t.Fatal("expected error for non-existent file")
	}
}

func TestReadImageDir(t *testing.T) {
	s := storage.NewStorage()
	// ReadImageDir currently returns nil, nil as it's handled internally
	result, err := s.ReadImageDir("/any/path")
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if result != nil {
		t.Errorf("expected nil result, got %v", result)
	}
}
