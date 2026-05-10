package domain

import "errors"

var (
	// ErrRequiresOCR is returned when the pre-flight analysis detects
	// that a document is scanned or image-only, requiring OCR.
	ErrRequiresOCR = errors.New("document requires OCR")

	// ErrCorruptedEncoding is returned when the PDF contains extractable
	// text, but the extracted characters indicate broken text encoding.
	ErrCorruptedEncoding = errors.New("document has corrupted PDF text encoding")
)

// IsSkippablePreflightError reports whether a pre-flight error should skip
// the file without failing the whole batch.
func IsSkippablePreflightError(err error) bool {
	return errors.Is(err, ErrRequiresOCR) || errors.Is(err, ErrCorruptedEncoding)
}
