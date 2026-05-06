package discovery

import (
	"os"
	"path/filepath"
	"strings"
)

// FindPDFs scans the given directory for .pdf files.
// If recursive is true, it searches all subdirectories.
func FindPDFs(dir string, recursive bool) ([]string, error) {
	var pdfs []string

	if recursive {
		err := filepath.WalkDir(dir, func(path string, d os.DirEntry, err error) error {
			if err != nil {
				return err // Permission denied or other errors
			}
			if !d.IsDir() && strings.ToLower(filepath.Ext(path)) == ".pdf" {
				pdfs = append(pdfs, path)
			}
			return nil
		})
		if err != nil {
			return nil, err
		}
	} else {
		entries, err := os.ReadDir(dir)
		if err != nil {
			return nil, err
		}
		for _, entry := range entries {
			if !entry.IsDir() {
				if strings.ToLower(filepath.Ext(entry.Name())) == ".pdf" {
					pdfs = append(pdfs, filepath.Join(dir, entry.Name()))
				}
			}
		}
	}

	return pdfs, nil
}
