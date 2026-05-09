package storage

import (
	"os"

	"github.com/nawodyaishan/pdf2md-tui/pkg/domain"
)

type diskStorage struct{}

// NewStorage creates a new local disk storage instance.
func NewStorage() domain.PDFStorage {
	return &diskStorage{}
}

func (s *diskStorage) WriteMarkdown(path string, data []byte) error {
	return os.WriteFile(path, data, 0644)
}

func (s *diskStorage) CreateImageDir(baseName string) (string, error) {
	// The extraction itself writes the image dir in images.go,
	// but this could be useful if the parser needs to know where to extract.
	return "", nil
}

func (s *diskStorage) ReadImageDir(dir string) ([]string, error) {
	return nil, nil // Handled inside pdfcpu directly for now.
}

func (s *diskStorage) FileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func (s *diskStorage) MkdirAll(path string) error {
	return os.MkdirAll(path, 0755)
}

func (s *diskStorage) StatSize(path string) (int64, error) {
	info, err := os.Stat(path)
	if err != nil {
		return 0, err
	}
	return info.Size(), nil
}
