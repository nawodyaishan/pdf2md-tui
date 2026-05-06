package converter

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/pdfcpu/pdfcpu/pkg/api"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
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

// Convert processes a single PDF file.
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

	baseName := strings.TrimSuffix(filepath.Base(pdfPath), filepath.Ext(pdfPath))
	dateSuffix := ""
	if c.DateFormat != "" && c.DateFormat != "none" {
		dateSuffix = "_" + time.Now().Format(c.DateFormat)
	}
	outFileName := fmt.Sprintf("%s%s.md", baseName, dateSuffix)
	res.OutputPath = filepath.Join(outDir, outFileName)

	f, err := os.Open(pdfPath)
	if err != nil {
		res.Err = fmt.Errorf("open pdf: %w", err)
		return res
	}
	defer f.Close()

	ctx, err := api.ReadContext(f, model.NewDefaultConfiguration())
	if err != nil {
		res.Err = fmt.Errorf("read context: %w", err)
		return res
	}

	if err := api.OptimizeContext(ctx); err != nil {
		// Ignore optimization errors, try to proceed
	}

	var buf bytes.Buffer
	buf.WriteString(fmt.Sprintf("# %s\n\n", filepath.Base(pdfPath)))

	// pdfcpu doesn't have high-level text extraction. 
	// We extract raw content streams and perform a naive parse of text operands (Tj, TJ).
	for i := 1; i <= ctx.PageCount; i++ {
		page, err := api.ExtractPage(ctx, i)
		if err != nil {
			continue // Skip problematic pages
		}
		
		content, _ := io.ReadAll(page)
		text := extractNaiveText(content)
		
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

// extractNaiveText attempts to extract text strings from PDF content streams.
// Looks for (Text) Tj and [(Text) 120 (More)] TJ
func extractNaiveText(content []byte) string {
	str := string(content)
	var result []string
	
	// Very naive regex to catch strings inside parentheses before Tj/TJ
	// e.g. (Hello World) Tj
	reTj := regexp.MustCompile(`\((.*?)\)\s*Tj`)
	matchesTj := reTj.FindAllStringSubmatch(str, -1)
	for _, m := range matchesTj {
		if len(m) > 1 {
			result = append(result, m[1])
		}
	}

	// For TJ e.g. [(H) 120 (ello)] TJ
	reTJ := regexp.MustCompile(`\[(.*?)\]\s*TJ`)
	matchesTJ := reTJ.FindAllStringSubmatch(str, -1)
	reInner := regexp.MustCompile(`\((.*?)\)`)
	for _, m := range matchesTJ {
		if len(m) > 1 {
			innerMatches := reInner.FindAllStringSubmatch(m[1], -1)
			var innerText string
			for _, im := range innerMatches {
				if len(im) > 1 {
					innerText += im[1]
				}
			}
			if innerText != "" {
				result = append(result, innerText)
			}
		}
	}

	return strings.Join(result, " ")
}

func applyLLMOptimizations(text string) string {
	// Remove isolated numbers (often page numbers) on their own line
	rePageNums := regexp.MustCompile(`(?m)^\s*\d+\s*$`)
	text = rePageNums.ReplaceAllString(text, "")
	
	// Collapse multiple spaces and newlines into a single space
	reSpaces := regexp.MustCompile(`\s+`)
	text = reSpaces.ReplaceAllString(text, " ")
	
	return strings.TrimSpace(text)
}
