package converter

import (
	"strings"

	"github.com/ledongthuc/pdf"
)

// PageAnalysis holds the results of the pre-flight check.
type PageAnalysis struct {
	TextCharCount int
	XObjectCount  int
	PagesSampled  int
}

// ocrTextThreshold is the minimum number of trimmed text characters
// required across the sampled pages to not be considered a scanned PDF.
const ocrTextThreshold = 50

// AnalyzePDF performs a lightweight pre-flight check on the first samplePages of a PDF.
// It returns ErrRequiresOCR if the document appears to be scanned or image-only.
func AnalyzePDF(pdfPath string, samplePages int) (PageAnalysis, error) {
	var analysis PageAnalysis

	f, reader, err := pdf.Open(pdfPath)
	if err != nil {
		return analysis, err
	}
	defer f.Close() //nolint:errcheck

	totalPages := reader.NumPage()
	if totalPages == 0 {
		return analysis, nil
	}

	limit := samplePages
	if totalPages < limit {
		limit = totalPages
	}
	analysis.PagesSampled = limit

	for i := 1; i <= limit; i++ {
		page := reader.Page(i)
		if page.V.IsNull() {
			continue
		}

		// 1. Get Text
		if text, err := page.GetPlainText(nil); err == nil {
			analysis.TextCharCount += len(strings.TrimSpace(text))
		}

		// 2. Count XObjects (Heuristic for images/forms)
		xobj := page.V.Key("Resources").Key("XObject")
		if xobj.Kind() == pdf.Dict {
			analysis.XObjectCount += xobj.Len()
		}
	}

	if analysis.TextCharCount < ocrTextThreshold && analysis.XObjectCount > 0 {
		return analysis, ErrRequiresOCR
	}

	return analysis, nil
}
