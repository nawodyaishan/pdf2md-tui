package pdf

import (
	"strings"

	ledongpdf "github.com/ledongthuc/pdf"
	"github.com/nawodyaishan/pdf2md-tui/pkg/domain"
)

// ocrTextThreshold is the minimum number of trimmed text characters
// required across the sampled pages to not be considered a scanned PDF.
const ocrTextThreshold = 50

// encodingCorruptionThreshold is the maximum ratio of non-ASCII characters
// before the document is considered to have a corrupted text encoding.
const encodingCorruptionThreshold = 0.20

// AnalyzePreFlight performs a lightweight pre-flight check on the first samplePages of a PDF.
// It returns ErrRequiresOCR if the document appears to be scanned, image-only, or has corrupted encoding.
func AnalyzePreFlight(reader *ledongpdf.Reader, samplePages int) (domain.PageAnalysis, error) {
	var analysis domain.PageAnalysis
	var totalChars, nonAsciiChars int

	totalPages := reader.NumPage()
	if totalPages == 0 {
		return analysis, nil
	}

	limit := samplePages
	if totalPages < limit {
		limit = totalPages
	}

	for i := 1; i <= limit; i++ {
		page := reader.Page(i)
		if page.V.IsNull() {
			continue
		}

		// 1. Get Text
		if text, err := page.GetPlainText(nil); err == nil {
			trimmed := strings.TrimSpace(text)
			analysis.CharCount += len(trimmed)
			for _, ch := range trimmed {
				totalChars++
				if ch > 127 {
					nonAsciiChars++
				}
			}
		}

		// 2. Count XObjects (Heuristic for images/forms)
		xobj := page.V.Key("Resources").Key("XObject")
		if xobj.Kind() == ledongpdf.Dict {
			analysis.XObjectCnt += xobj.Len()
		}
	}

	if analysis.CharCount < ocrTextThreshold && analysis.XObjectCnt > 0 {
		return analysis, domain.ErrRequiresOCR
	}

	if totalChars > 100 && float64(nonAsciiChars)/float64(totalChars) > encodingCorruptionThreshold {
		return analysis, domain.ErrRequiresOCR
	}

	return analysis, nil
}
