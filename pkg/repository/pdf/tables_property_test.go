package pdf

import (
	"math"
	"testing"

	ledongpdf "github.com/ledongthuc/pdf"
	"pgregory.net/rapid"
)

// TestProperty_CoalesceChars_WordCountBound verifies the fundamental invariant:
// coalesceChars never produces more words than input characters
func TestProperty_CoalesceChars_WordCountBound(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		n := rapid.IntRange(0, 200).Draw(t, "n")
		texts := make([]ledongpdf.Text, n)
		for i := range texts {
			texts[i] = ledongpdf.Text{
				S:        rapid.StringMatching(`[A-Za-z0-9]`).Draw(t, "char"),
				X:        rapid.Float64Range(0, 612).Draw(t, "x"),
				Y:        rapid.Float64Range(0, 792).Draw(t, "y"),
				FontSize: rapid.Float64Range(6, 72).Draw(t, "fs"),
				W:        rapid.Float64Range(1, 30).Draw(t, "w"),
			}
		}
		words := coalesceChars(texts)
		if len(words) > len(texts) {
			t.Fatalf("coalesceChars produced %d words from %d chars — impossible", len(words), len(texts))
		}
	})
}

// TestProperty_GroupIntoRows_SortOrder verifies rows are sorted correctly:
// - Y-coordinates descending (top to bottom in PDF coords)
// - Cells within each row sorted X-ascending (left to right)
func TestProperty_GroupIntoRows_SortOrder(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		n := rapid.IntRange(1, 50).Draw(t, "n")
		words := make([]word, n)
		for i := range words {
			words[i] = word{
				text: rapid.StringMatching(`[A-Za-z]`).Draw(t, "w"),
				x:    rapid.Float64Range(0, 500).Draw(t, "x"),
				y:    rapid.Float64Range(0, 700).Draw(t, "y"),
			}
		}
		rows := groupIntoRows(words, tableYRowTolerance)

		// Verify rows are sorted Y-descending
		for i := 1; i < len(rows); i++ {
			if rows[i].y > rows[i-1].y {
				t.Fatalf("rows not sorted Y-descending at index %d: %.2f > %.2f", i, rows[i].y, rows[i-1].y)
			}
		}

		// Verify cells within each row are sorted X-ascending
		for rowIdx, r := range rows {
			for cellIdx := 1; cellIdx < len(r.cells); cellIdx++ {
				if r.cells[cellIdx].x < r.cells[cellIdx-1].x {
					t.Fatalf("cells not sorted X-ascending in row %d at cell %d: %.2f < %.2f",
						rowIdx, cellIdx, r.cells[cellIdx].x, r.cells[cellIdx-1].x)
				}
			}
		}
	})
}

// TestProperty_GroupIntoRows_NoPanic ensures the grouping algorithm never panics
func TestProperty_GroupIntoRows_NoPanic(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		n := rapid.IntRange(0, 100).Draw(t, "n")
		words := make([]word, n)
		for i := range words {
			words[i] = word{
				text: rapid.String().Draw(t, "text"),
				x:    rapid.Float64Range(math.Inf(-1), math.Inf(1)).Draw(t, "x"),
				y:    rapid.Float64Range(math.Inf(-1), math.Inf(1)).Draw(t, "y"),
			}
		}
		// Must not panic even with infinite or NaN coordinates
		_ = groupIntoRows(words, tableYRowTolerance)
	})
}

// TestProperty_ExtractPageBlocks_NoPanic ensures the extraction never panics
func TestProperty_ExtractPageBlocks_NoPanic(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		n := rapid.IntRange(0, 100).Draw(t, "n")
		texts := make([]ledongpdf.Text, n)
		for i := range texts {
			texts[i] = ledongpdf.Text{
				S:        rapid.String().Draw(t, "s"),
				X:        rapid.Float64Range(0, 612).Draw(t, "x"),
				Y:        rapid.Float64Range(0, 792).Draw(t, "y"),
				FontSize: rapid.Float64Range(0, 100).Draw(t, "fs"),
				W:        rapid.Float64Range(0, 50).Draw(t, "w"),
			}
		}
		page := NewMockPage(texts)
		// Must not panic on any valid page
		_ = extractPageBlocks(page)
	})
}

// TestProperty_CoalesceChars_OutputNotEmpty verifies non-empty input produces output
func TestProperty_CoalesceChars_OutputNotEmpty(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		texts := []ledongpdf.Text{
			{S: "Hello", X: 10, Y: 100, FontSize: 12, W: 7},
			{S: "World", X: 100, Y: 100, FontSize: 12, W: 7},
		}
		words := coalesceChars(texts)
		if len(words) == 0 {
			t.Fatalf("coalesceChars returned empty for valid input")
		}
	})
}
