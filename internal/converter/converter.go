package converter

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/ledongthuc/pdf"
)

// Converter handles the PDF to MD conversion settings.
type Converter struct {
	DateFormat string
	StripNoise bool
}

// Result holds metrics and results of a conversion.
type Result struct {
	InputPath   string
	OutputPath  string
	InputBytes  int64
	OutputBytes int64
	Duration    time.Duration
	Err         error
}

// New creates a new Converter.
func New(dateFormat string, stripNoise bool) *Converter {
	if dateFormat == "" {
		dateFormat = "2006-01-02"
	}
	return &Converter{
		DateFormat: dateFormat,
		StripNoise: stripNoise,
	}
}

// Convert processes a single PDF file into LLM-friendly Markdown.
func (c *Converter) Convert(pdfPath, outDir string) Result {
	start := time.Now()
	res := Result{
		InputPath: pdfPath,
	}

	info, err := os.Stat(pdfPath)
	if err != nil {
		res.Err = fmt.Errorf("stat: %w", err)
		return res
	}
	res.InputBytes = info.Size()

	baseName, outFile := outputFilename(pdfPath, c.DateFormat, start)
	res.OutputPath = filepath.Join(outDir, outFile)

	// Open PDF using ledongthuc/pdf which handles font decoding and glyph mapping
	f, reader, err := pdf.Open(pdfPath)
	if err != nil {
		res.Err = fmt.Errorf("open pdf: %w", err)
		return res
	}
	defer func() { _ = f.Close() }()

	var buf bytes.Buffer
	buf.WriteString(fmt.Sprintf("# %s\n\n", baseName))

	totalPages := reader.NumPage()
	for i := 1; i <= totalPages; i++ {
		page := reader.Page(i)
		if page.V.IsNull() {
			continue
		}

		// Try positional extraction first (preserves tables)
		text := extractWithTables(page)

		// Fallback to plain text if positional extraction yields nothing
		if strings.TrimSpace(text) == "" {
			plainText, err := page.GetPlainText(nil)
			if err != nil {
				continue
			}
			text = cleanExtractedText(plainText)
		}

		if c.StripNoise {
			text = applyLLMOptimizations(text)
		}

		if strings.TrimSpace(text) != "" {
			buf.WriteString(text)
			buf.WriteString("\n\n")
		}
	}

	outData := buf.Bytes()
	if err := os.WriteFile(res.OutputPath, outData, 0644); err != nil {
		res.Err = fmt.Errorf("write md: %w", err)
		return res
	}

	res.OutputBytes = int64(len(outData))
	res.Duration = time.Since(start)
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
func (c *Converter) OutputPath(pdfPath, outDir string) string {
	_, filename := outputFilename(pdfPath, c.DateFormat, time.Now())
	return filepath.Join(outDir, filename)
}

// cleanExtractedText joins word fragments that ledongthuc/pdf separates with newlines.
// The library outputs each word on its own line with single-space lines as separators:
//
//	"Hello\n \nWorld\n"   → "Hello World"
//
// Truly empty lines (no space) signal paragraph breaks.
func cleanExtractedText(text string) string {
	lines := strings.Split(text, "\n")
	var paragraphs []string
	var currentWords []string

	for _, line := range lines {
		// A line that is exactly a single space is a word separator from the library
		if line == " " {
			continue // skip — the preceding/following non-empty lines are the words
		}

		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			// Truly empty line = paragraph break
			if len(currentWords) > 0 {
				paragraphs = append(paragraphs, strings.Join(currentWords, " "))
				currentWords = nil
			}
			continue
		}
		currentWords = append(currentWords, trimmed)
	}
	// Flush remaining words
	if len(currentWords) > 0 {
		paragraphs = append(paragraphs, strings.Join(currentWords, " "))
	}

	return strings.TrimSpace(strings.Join(paragraphs, "\n\n"))
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
