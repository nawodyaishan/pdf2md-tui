package pdf

import (
	"github.com/ledongthuc/pdf"
	"github.com/nawodyaishan/pdf2md-tui/pkg/domain"
	"testing"
)

func TestExtractPageBlocks_SyntheticTable(t *testing.T) {
	// Setup a synthetic table row
	texts := []pdf.Text{
		{S: "Col1", X: 50, Y: 100, W: 30, FontSize: 10},
		{S: "Col2", X: 150, Y: 100, W: 30, FontSize: 10},
	}
	page := NewMockPage(texts)

	blocks := extractPageBlocks(page)

	// Since we only have 1 row (less than tableMinCols=2), it should be text
	if len(blocks) != 1 || blocks[0].Type != domain.BlockTypeText {
		t.Errorf("Expected 1 text block, got %d", len(blocks))
	}
}
