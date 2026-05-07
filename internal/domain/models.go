package domain

import "time"

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

// Result holds metrics and results of a conversion.
type Result struct {
	InputPath   string
	OutputPath  string
	InputBytes  int64
	OutputBytes int64
	Duration    time.Duration
	Status      Status
	Err         error
}

// ExtractedImage tracks an image saved to disk.
type ExtractedImage struct {
	PageNumber int
	Path       string // Relative path to the image
}

// PageAnalysis represents the result of pre-flight analysis for a single page.
type PageAnalysis struct {
	CharCount  int
	XObjectCnt int
}

// PDFStorage defines the interface for interacting with the file system.
type PDFStorage interface {
	WriteMarkdown(path string, data []byte) error
	CreateImageDir(baseName string) (string, error)
	ReadImageDir(dir string) ([]string, error)
	FileExists(path string) bool
	MkdirAll(path string) error
	StatSize(path string) (int64, error)
}

// PDFParser defines the interface for reading and parsing PDFs.
// This isolates the specific PDF library dependencies.
type PDFParser interface {
	ExtractImages(pdfPath string, imgDir string) ([]ExtractedImage, error)
	OpenDocument(pdfPath string) (PDFDocument, error)
}

// PDFDocument represents an open PDF file ready for extraction.
type PDFDocument interface {
	NumPages() int
	ExtractPageText(pageNum int) (string, error)
	AnalyzePreFlight(samplePages int) (PageAnalysis, error)
	Close() error
}
