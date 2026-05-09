package service

import (
	"errors"
	"path/filepath"
	"strings"
	"testing"

	"github.com/nawodyaishan/pdf2md-tui/pkg/domain"
)

func TestApplyLLMOptimizations(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{
			name: "collapse spaces but keep vertical whitespace",
			in:   "Hello     World\n\n\nTesting",
			want: "Hello World\n\n\nTesting",
		},
		{
			name: "remove isolated numbers (page numbers)",
			in:   "Hello World\n12\nNext Page",
			want: "Hello World\n\nNext Page",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := applyLLMOptimizations(tt.in); got != tt.want {
				t.Errorf("applyLLMOptimizations() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestRenderMarkdown_TableFencing(t *testing.T) {
	blocks := []domain.PageBlock{
		{Type: domain.BlockTypeText, Text: "Before table"},
		{
			Type: domain.BlockTypeTable,
			Table: domain.TableData{
				Rows: [][]string{
					{"Col1", "Col2", "Col3"},
					{"A", "B", "C"},
				},
			},
		},
		{Type: domain.BlockTypeText, Text: "After table"},
	}

	result := renderMarkdown(blocks, false)

	if !strings.Contains(result, "Before table\n\n| Col1") {
		t.Errorf("table not fenced properly before: %q", result)
	}
	if !strings.Contains(result, "|\n\nAfter table") {
		t.Errorf("table not fenced properly after: %q", result)
	}
}

func TestOutputPath(t *testing.T) {
	cfg := domain.NewConfig()
	cfg.DateFormat = "2006-01-02"
	svc := NewConverterService(cfg, &mockStorage{writes: make(map[string][]byte)}, &mockParser{}, nil)

	testPath := "/path/to/document.pdf"
	outDir := "/output"

	result := svc.OutputPath(testPath, outDir)

	if !strings.Contains(result, "document") {
		t.Errorf("OutputPath should contain document name, got %q", result)
	}
	if !strings.Contains(result, ".md") {
		t.Errorf("OutputPath should have .md extension, got %q", result)
	}
	if !strings.Contains(result, outDir) {
		t.Errorf("OutputPath should contain output dir %q, got %q", outDir, result)
	}
}

// Mocks

type mockStorage struct {
	writes map[string][]byte
}

func (m *mockStorage) WriteMarkdown(path string, data []byte) error {
	m.writes[path] = data
	return nil
}
func (m *mockStorage) CreateImageDir(baseName string) (string, error) { return "", nil }
func (m *mockStorage) ReadImageDir(dir string) ([]string, error)      { return nil, nil }
func (m *mockStorage) FileExists(path string) bool                    { return false }
func (m *mockStorage) MkdirAll(path string) error                     { return nil }
func (m *mockStorage) StatSize(path string) (int64, error) {
	if strings.Contains(path, "doesnotexist") {
		return 0, errors.New("not found")
	}
	return 100, nil
}

type mockParser struct{}

func (m *mockParser) ExtractImages(pdfPath string, imgDir string) ([]domain.ExtractedImage, error) {
	return nil, nil
}
func (m *mockParser) OpenDocument(pdfPath string) (domain.PDFDocument, error) {
	if strings.Contains(pdfPath, "bad") {
		return nil, errors.New("open pdf: bad format")
	}
	return &mockDoc{}, nil
}

type mockDoc struct{}

func (m *mockDoc) NumPages() int { return 1 }
func (m *mockDoc) ExtractPageBlocks(pageNum int) ([]domain.PageBlock, error) {
	return []domain.PageBlock{{Type: domain.BlockTypeText, Text: "Test text"}}, nil
}
func (m *mockDoc) AnalyzePreFlight(samplePages int) (domain.PageAnalysis, error) {
	return domain.PageAnalysis{}, nil
}
func (m *mockDoc) Close() error { return nil }

type mockTokenizer struct{}

func (m *mockTokenizer) Count(text string) int { return len(strings.Fields(text)) }

func TestConverter_Convert_NoPDF(t *testing.T) {
	cfg := domain.NewConfig()
	storage := &mockStorage{writes: make(map[string][]byte)}
	parser := &mockParser{}

	conv := NewConverterService(cfg, storage, parser, &mockTokenizer{})
	tempDir := t.TempDir()

	res := conv.Convert(filepath.Join(tempDir, "doesnotexist.pdf"), tempDir)
	if res.Err == nil {
		t.Error("expected error for non-existent file, got nil")
	}
}

func TestConverter_Convert_NotAPDF(t *testing.T) {
	cfg := domain.NewConfig()
	storage := &mockStorage{writes: make(map[string][]byte)}
	parser := &mockParser{}

	conv := NewConverterService(cfg, storage, parser, &mockTokenizer{})
	tempDir := t.TempDir()

	badPDF := filepath.Join(tempDir, "bad.pdf")
	res := conv.Convert(badPDF, tempDir)
	if res.Err == nil {
		t.Error("expected error for invalid pdf format, got nil")
	}
	if !strings.Contains(res.Err.Error(), "open pdf") {
		t.Errorf("expected open pdf error, got %v", res.Err)
	}
}
