package converter

import "errors"

// Status represents the final outcome of a conversion attempt.
type Status int

const (
	// StatusOK indicates the document was converted successfully.
	StatusOK Status = iota
	// StatusIgnored indicates the document was skipped gracefully (e.g., requires OCR).
	StatusIgnored
	// StatusError indicates the conversion failed unexpectedly.
	StatusError
)

// ErrRequiresOCR is returned when the pre-flight analysis detects
// that a document is scanned or image-only, requiring OCR.
var ErrRequiresOCR = errors.New("converter: document requires OCR")
