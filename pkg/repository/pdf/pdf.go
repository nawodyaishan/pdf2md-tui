package pdf

import (
	"os"

	ledongpdf "github.com/ledongthuc/pdf"
	"github.com/nawodyaishan/pdf2md-tui/pkg/domain"
)

type parser struct{}

// NewParser creates a new PDF parser instance.
func NewParser() domain.PDFParser {
	return &parser{}
}

func (p *parser) ExtractImages(pdfPath string, imgDir string) ([]domain.ExtractedImage, error) {
	return ExtractImages(pdfPath, imgDir)
}

func (p *parser) OpenDocument(pdfPath string) (domain.PDFDocument, error) {
	f, reader, err := ledongpdf.Open(pdfPath)
	if err != nil {
		return nil, err
	}
	return &document{
		file:   f,
		reader: reader,
	}, nil
}

type document struct {
	file   *os.File
	reader *ledongpdf.Reader
}

func (d *document) NumPages() int {
	return d.reader.NumPage()
}

// pageWrapper wraps ledongpdf.Page to satisfy the Page interface.
type pageWrapper struct {
	page ledongpdf.Page
}

func (pw *pageWrapper) Content() ledongpdf.Content {
	return pw.page.Content()
}

func (d *document) ExtractPageBlocks(pageNum int) ([]domain.PageBlock, error) {
	page := d.reader.Page(pageNum)
	if page.V.IsNull() {
		return nil, nil
	}

	blocks := extractPageBlocks(&pageWrapper{page: page})
	return blocks, nil
}

func (d *document) AnalyzePreFlight(samplePages int) (domain.PageAnalysis, error) {
	return AnalyzePreFlight(d.reader, samplePages)
}

func (d *document) Close() error {
	if d.file != nil {
		return d.file.Close()
	}
	return nil
}
