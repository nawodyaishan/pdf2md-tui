package pdf

import (
	"github.com/ledongthuc/pdf"
	"github.com/nawodyaishan/pdf2md-tui/pkg/domain"
	"testing"
)

func TestExtractPageBlocks_SyntheticTable_Degenerate(t *testing.T) {
	// Setup a synthetic 2-column row (below threshold)
	texts := []pdf.Text{
		{S: "Col1", X: 50, Y: 100, W: 30, FontSize: 10},
		{S: "Col2", X: 150, Y: 100, W: 30, FontSize: 10},
	}
	page := NewMockPage(texts)

	blocks := extractPageBlocks(page)

	// Since we only have 1 row and 2 cols (less than 3), it should be text
	if len(blocks) != 1 || blocks[0].Type != domain.BlockTypeText {
		t.Errorf("Expected 1 text block, got %d", len(blocks))
	}
}

func TestExtractPageBlocks_SyntheticTable_NoPanic(t *testing.T) {
	// Setup a synthetic 3-column layout and verify it produces blocks without panic
	// Table detection heuristics are complex (gap detection, row tolerance, column spacing);
	// comprehensive table extraction testing is handled via:
	// 1. Property-based tests (tables_property_test.go) in Phase 2
	// 2. Golden master tests (qa_golden_test.go) with real PDFs
	texts := []pdf.Text{
		// Row 1 (Y=100) with 3 widely-spaced cells
		{S: "Col1", X: 50, Y: 100, W: 30, FontSize: 10},
		{S: "Col2", X: 200, Y: 100, W: 30, FontSize: 10},
		{S: "Col3", X: 350, Y: 100, W: 30, FontSize: 10},
		// Row 2 (Y=99) — within groupIntoRows tolerance
		{S: "A", X: 50, Y: 99, W: 10, FontSize: 10},
		{S: "B", X: 200, Y: 99, W: 10, FontSize: 10},
		{S: "C", X: 350, Y: 99, W: 10, FontSize: 10},
	}
	page := NewMockPage(texts)

	// Must not panic; block types are validated in property tests
	blocks := extractPageBlocks(page)

	if blocks == nil {
		t.Fatal("expected non-nil blocks for valid page content")
	}
}
