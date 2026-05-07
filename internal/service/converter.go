package service

import (
	"bytes"
	"fmt"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/nawodyaishan/pdf2md-tui/internal/domain"
)

// ConverterService handles the PDF to MD conversion settings and orchestration.
type ConverterService struct {
	config  *domain.Config
	storage domain.PDFStorage
	parser  domain.PDFParser
}

// NewConverterService creates a new ConverterService with injected dependencies.
func NewConverterService(config *domain.Config, storage domain.PDFStorage, parser domain.PDFParser) *ConverterService {
	return &ConverterService{
		config:  config,
		storage: storage,
		parser:  parser,
	}
}

// Convert processes a single PDF file into LLM-friendly Markdown.
func (c *ConverterService) Convert(pdfPath, outDir string) domain.Result {
	start := time.Now()
	res := domain.Result{
		InputPath: pdfPath,
	}

	size, err := c.storage.StatSize(pdfPath)
	if err != nil {
		res.Status = domain.StatusError
		res.Err = fmt.Errorf("stat: %w", err)
		return res
	}
	res.InputBytes = size

	baseName, outFile := outputFilename(pdfPath, c.config.DateFormat, start)
	res.OutputPath = filepath.Join(outDir, outFile)

	// Pre-flight check (if not extracting images, or as a general OCR heuristic)
	// We'll open the document once, and use it.
	doc, err := c.parser.OpenDocument(pdfPath)
	if err != nil {
		res.Status = domain.StatusError
		res.Err = fmt.Errorf("open pdf: %w", err)
		return res
	}
	defer doc.Close()

	// Analyze pre-flight
	_, err = doc.AnalyzePreFlight(3)
	if err == domain.ErrRequiresOCR {
		res.Status = domain.StatusIgnored
		res.Err = domain.ErrRequiresOCR
		return res
	}

	var buf bytes.Buffer
	buf.WriteString(fmt.Sprintf("# %s\n\n", baseName))

	var imagesByPage map[int][]domain.ExtractedImage
	if c.config.ExtractImages {
		// Ensure output dir is created
		c.storage.MkdirAll(outDir)
		if imgs, err := c.parser.ExtractImages(pdfPath, outDir); err == nil {
			imagesByPage = make(map[int][]domain.ExtractedImage)
			for _, img := range imgs {
				imagesByPage[img.PageNumber] = append(imagesByPage[img.PageNumber], img)
			}
		}
	}

	totalPages := doc.NumPages()
	for i := 1; i <= totalPages; i++ {
		text, err := doc.ExtractPageText(i)
		if err != nil || strings.TrimSpace(text) == "" {
			continue
		}

		if c.config.ExtractImages && len(imagesByPage[i]) > 0 {
			for _, img := range imagesByPage[i] {
				buf.WriteString(fmt.Sprintf("![image](%s)\n\n", img.Path))
			}
		}

		if c.config.StripNoise {
			text = applyLLMOptimizations(text)
		}

		if strings.TrimSpace(text) != "" {
			buf.WriteString(text)
			buf.WriteString("\n\n")
		}
	}

	outData := buf.Bytes()
	if err := c.storage.MkdirAll(outDir); err != nil {
		res.Status = domain.StatusError
		res.Err = fmt.Errorf("mkdir outDir: %w", err)
		return res
	}

	if err := c.storage.WriteMarkdown(res.OutputPath, outData); err != nil {
		res.Status = domain.StatusError
		res.Err = fmt.Errorf("write md: %w", err)
		return res
	}

	res.OutputBytes = int64(len(outData))
	res.Duration = time.Since(start)
	res.Status = domain.StatusOK
	return res
}

// outputFilename returns the document base name and the full output filename for a PDF.
func outputFilename(pdfPath, dateFormat string, now time.Time) (baseName, filename string) {
	baseName = strings.TrimSuffix(filepath.Base(pdfPath), filepath.Ext(pdfPath))
	dateSuffix := ""
	if dateFormat != "" && dateFormat != "none" {
		dateSuffix = "_" + now.Format(dateFormat)
	}
	return baseName, fmt.Sprintf("%s%s.md", baseName, dateSuffix)
}

// OutputPath returns the expected output path for a PDF without performing conversion.
// Used by the CLI for pre-flight overwrite checks.
func (c *ConverterService) OutputPath(pdfPath, outDir string) string {
	_, filename := outputFilename(pdfPath, c.config.DateFormat, time.Now())
	return filepath.Join(outDir, filename)
}

// applyLLMOptimizations aggressively strips noise for LLM consumption.
func applyLLMOptimizations(text string) string {
	// Remove isolated numbers (often page numbers) on their own line
	rePageNums := regexp.MustCompile(`(?m)^\s*\d+\s*$`)
	text = rePageNums.ReplaceAllString(text, "")

	// Collapse multiple spaces and newlines into a single space
	reSpaces := regexp.MustCompile(`\s+`)
	text = reSpaces.ReplaceAllString(text, " ")

	return strings.TrimSpace(text)
}
