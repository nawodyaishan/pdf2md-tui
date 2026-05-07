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
	InputPath    string
	OutputPath   string
	InputBytes   int64
	OutputBytes  int64
	InputTokens  int
	OutputTokens int
	Duration     time.Duration
	Status       Status
	Err          error
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

// Summary is a JSON-serializable report of the conversion session.
type Summary struct {
	Converted int           `json:"converted"`
	Skipped   int           `json:"skipped"`
	Errors    int           `json:"errors"`
	Duration  string        `json:"duration"`
	Files     []FileSummary `json:"files"`
}

// FileSummary represents the outcome for a single file.
type FileSummary struct {
	Input       string `json:"input"`
	Output      string `json:"output,omitempty"`
	Status      string `json:"status"` // "ok", "ignored", "error"
	Error       string `json:"error,omitempty"`
	InputBytes  int64  `json:"input_bytes"`
	OutputBytes int64  `json:"output_bytes,omitempty"`
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

// BlockType defines the structural type of a PageBlock.
type BlockType string

const (
	BlockTypeText  BlockType = "text"
	BlockTypeTable BlockType = "table"
)

// PageBlock represents a structural element extracted from the PDF.
type PageBlock struct {
	Type  BlockType
	Text  string    // Used if Type == BlockTypeText
	Table TableData // Used if Type == BlockTypeTable
}

// TableData represents a structured grid of cells for a table block.
type TableData struct {
	Rows [][]string
}

// Tokenizer defines the interface for counting tokens in a string.
type Tokenizer interface {
	Count(text string) int
}

// PDFDocument represents an open PDF file ready for extraction.
type PDFDocument interface {
	NumPages() int
	ExtractPageBlocks(pageNum int) ([]PageBlock, error)
	AnalyzePreFlight(samplePages int) (PageAnalysis, error)
	Close() error
}
