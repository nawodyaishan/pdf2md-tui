package domain

import "errors"

var (
	// ErrRequiresOCR is returned when the pre-flight analysis detects
	// that a document is scanned or image-only, requiring OCR.
	ErrRequiresOCR = errors.New("document requires OCR")
)
