package service

import (
	"bytes"
	"fmt"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/nawodyaishan/pdf2md-tui/pkg/domain"
)

// ConverterService handles the PDF to MD conversion settings and orchestration.
type ConverterService struct {
	config    *domain.Config
	storage   domain.PDFStorage
	parser    domain.PDFParser
	tokenizer domain.Tokenizer
}

// NewConverterService creates a new ConverterService with injected dependencies.
func NewConverterService(config *domain.Config, storage domain.PDFStorage, parser domain.PDFParser, tokenizer domain.Tokenizer) *ConverterService {
	return &ConverterService{
		config:    config,
		storage:   storage,
		parser:    parser,
		tokenizer: tokenizer,
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
	defer func() { _ = doc.Close() }()

	// Analyze pre-flight
	_, err = doc.AnalyzePreFlight(3)
	if err == domain.ErrRequiresOCR {
		res.Status = domain.StatusIgnored
		res.Err = domain.ErrRequiresOCR
		return res
	}

	var buf bytes.Buffer
	fmt.Fprintf(&buf, "# %s\n\n", baseName)

	var imagesByPage map[int][]domain.ExtractedImage
	if c.config.ExtractImages {
		// Ensure output dir is created
		_ = c.storage.MkdirAll(outDir)
		if imgs, err := c.parser.ExtractImages(pdfPath, outDir); err == nil {
			imagesByPage = make(map[int][]domain.ExtractedImage)
			for _, img := range imgs {
				imagesByPage[img.PageNumber] = append(imagesByPage[img.PageNumber], img)
			}
		}
	}

	totalPages := doc.NumPages()
	for i := 1; i <= totalPages; i++ {
		blocks, err := doc.ExtractPageBlocks(i)
		if err != nil || len(blocks) == 0 {
			continue
		}

		if c.config.ExtractImages && len(imagesByPage[i]) > 0 {
			for _, img := range imagesByPage[i] {
				fmt.Fprintf(&buf, "![image](%s)\n\n", img.Path)
			}
		}

		text := renderMarkdown(blocks)

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
	if c.tokenizer != nil {
		res.OutputTokens = c.tokenizer.Count(string(outData))
		// For InputTokens, we estimate from InputBytes if we don't have raw text
		// In a high-quality impl, we could count tokens as we extract text per page
		res.InputTokens = int(float64(res.InputBytes) / 1024 * 264)
	}
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

	// Collapse horizontal whitespace (spaces, tabs) but preserve newlines
	reHorizontalSpaces := regexp.MustCompile(`[^\S\r\n]+`)
	text = reHorizontalSpaces.ReplaceAllString(text, " ")

	// Normalize multiple newlines to exactly \n\n for structural preservation (G2)
	reVerticalSpaces := regexp.MustCompile(`\n{3,}`)
	text = reVerticalSpaces.ReplaceAllString(text, "\n\n")

	return strings.TrimSpace(text)
}

// renderMarkdown converts PageBlocks to markdown, detecting and formatting tables.
// Implements Table Fencing (G2) by ensuring tables are surrounded by \n\n.
func renderMarkdown(blocks []domain.PageBlock) string {
	var buf strings.Builder

	for _, block := range blocks {
		switch block.Type {
		case domain.BlockTypeText:
			buf.WriteString(block.Text)
			buf.WriteString("\n")
		case domain.BlockTypeTable:
			buf.WriteString("\n\n") // Fencing: blank line before table

			for ri, row := range block.Table.Rows {
				buf.WriteString("| ")
				buf.WriteString(strings.Join(row, " | "))
				buf.WriteString(" |\n")

				// Add separator after first row (header)
				if ri == 0 {
					buf.WriteString("|")
					for range row {
						buf.WriteString(" --- |")
					}
					buf.WriteString("\n")
				}
			}
			buf.WriteString("\n") // Blank line after table (fencing)
		}
	}

	// bindHeadersToParagraphs: ensure that Markdown headers are never separated from
	// their immediately following paragraph by more than one newline.
	// Regex: header line followed by excessive whitespace before next content
	re := regexp.MustCompile(`(?m)(^#{1,6}\s+.+\n)\n+`)
	return re.ReplaceAllString(buf.String(), "$1")
}
