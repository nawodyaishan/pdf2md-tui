package pdf

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"github.com/nawodyaishan/pdf2md-tui/pkg/domain"
	"github.com/pdfcpu/pdfcpu/pkg/api"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
)

// ExtractImages uses pdfcpu to extract all images from a PDF into an isolated directory.
// It returns a slice of ExtractedImage, grouped and sorted by page number.
func ExtractImages(pdfPath string, outDir string) ([]domain.ExtractedImage, error) {
	baseName := strings.TrimSuffix(filepath.Base(pdfPath), filepath.Ext(pdfPath))
	imgDir := filepath.Join(outDir, "images", baseName)

	if err := os.MkdirAll(imgDir, 0755); err != nil {
		return nil, fmt.Errorf("create image dir: %w", err)
	}

	conf := model.NewDefaultConfiguration()
	// Extract images
	err := api.ExtractImagesFile(pdfPath, imgDir, nil, conf)
	if err != nil {
		return nil, fmt.Errorf("extract images: %w", err)
	}

	// Read extracted files
	entries, err := os.ReadDir(imgDir)
	if err != nil {
		return nil, fmt.Errorf("read image dir: %w", err)
	}

	var images []domain.ExtractedImage

	for _, e := range entries {
		if e.IsDir() {
			continue
		}

		name := e.Name()
		// pdfcpu generated files are formatted as: <basename>_<pageNumber>_<objId>.<ext>
		// We split by '_' to extract the page number.
		parts := strings.Split(name, "_")
		if len(parts) >= 2 {
			// The page number is typically the second to last part, or the part immediately following basename
			// Since basename itself could have '_', we find the part that represents pageNumber.
			// pdfcpu's exact format: basename_pageNum_objId.ext

			// Try to parse page number from parts[len(parts)-2]
			pageNumStr := parts[len(parts)-2]
			pageNum, err := strconv.Atoi(pageNumStr)
			if err == nil {
				// Construct relative path for Markdown injection
				relPath := filepath.Join("images", baseName, name)
				images = append(images, domain.ExtractedImage{
					PageNumber: pageNum,
					Path:       relPath,
				})
			}
		}
	}

	// Sort images by page number and then by name
	sort.Slice(images, func(i, j int) bool {
		if images[i].PageNumber == images[j].PageNumber {
			return images[i].Path < images[j].Path
		}
		return images[i].PageNumber < images[j].PageNumber
	})

	return images, nil
}
