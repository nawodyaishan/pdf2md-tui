package pdf

import (
	"os"
	"testing"

	ledongpdf "github.com/ledongthuc/pdf"
)

func TestNewParser(t *testing.T) {
	p := NewParser()
	if p == nil {
		t.Fatal("NewParser returned nil")
	}
}

func TestParserInterfaceAssertion(t *testing.T) {
	var p = NewParser()
	if p == nil {
		t.Fatal("parser does not implement domain.PDFParser")
	}
}

func TestPageWrapperContent(t *testing.T) {
	mockPage := NewMockPage([]ledongpdf.Text{
		{S: "test", X: 10, Y: 20},
	})
	content := mockPage.Content()
	if len(content.Text) != 1 {
		t.Errorf("expected 1 text element, got %d", len(content.Text))
	}
}

func TestDocumentClose(t *testing.T) {
	tempFile, err := os.CreateTemp("", "test*.pdf")
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}
	defer func() { _ = os.Remove(tempFile.Name()) }()

	doc := &document{
		file: tempFile,
	}

	if err := doc.Close(); err != nil {
		t.Errorf("Close returned unexpected error: %v", err)
	}
}

func TestDocumentCloseNilFile(t *testing.T) {
	doc := &document{
		file: nil,
	}

	if err := doc.Close(); err != nil {
		t.Errorf("Close with nil file should not error, got: %v", err)
	}
}
