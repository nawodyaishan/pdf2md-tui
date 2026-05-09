package pdf

import (
	"testing"

	ledongpdf "github.com/ledongthuc/pdf"
	"github.com/nawodyaishan/pdf2md-tui/pkg/domain"
)

// --- coalesceChars tests ---

func TestCoalesceChars_Empty(t *testing.T) {
	result := coalesceChars(nil)
	if len(result) != 0 {
		t.Errorf("expected empty, got %d words", len(result))
	}
}

func TestCoalesceChars_SingleCharacter(t *testing.T) {
	texts := []ledongpdf.Text{
		{S: "A", X: 10, Y: 100, FontSize: 12, W: 7},
	}
	result := coalesceChars(texts)
	if len(result) != 1 {
		t.Fatalf("expected 1 word, got %d", len(result))
	}
	if result[0].text != "A" {
		t.Errorf("expected 'A', got %q", result[0].text)
	}
}

func TestCoalesceChars_JoinsAdjacentChars(t *testing.T) {
	// Simulate "Hello" as 5 characters, each ~7pt wide, placed with no gap
	texts := []ledongpdf.Text{
		{S: "H", X: 10, Y: 100, FontSize: 12, W: 7},
		{S: "e", X: 17, Y: 100, FontSize: 12, W: 7},
		{S: "l", X: 24, Y: 100, FontSize: 12, W: 7},
		{S: "l", X: 31, Y: 100, FontSize: 12, W: 7},
		{S: "o", X: 38, Y: 100, FontSize: 12, W: 7},
	}
	result := coalesceChars(texts)
	if len(result) != 1 {
		t.Fatalf("expected 1 word, got %d: %+v", len(result), result)
	}
	if result[0].text != "Hello" {
		t.Errorf("expected 'Hello', got %q", result[0].text)
	}
}

func TestCoalesceChars_SplitsDistantChars(t *testing.T) {
	// Two words "Hi" and "Go" separated by a large gap (176pt)
	texts := []ledongpdf.Text{
		{S: "H", X: 10, Y: 100, FontSize: 12, W: 7},
		{S: "i", X: 17, Y: 100, FontSize: 12, W: 7},
		{S: "G", X: 200, Y: 100, FontSize: 12, W: 7},
		{S: "o", X: 207, Y: 100, FontSize: 12, W: 7},
	}
	result := coalesceChars(texts)
	if len(result) != 2 {
		t.Fatalf("expected 2 words, got %d: %+v", len(result), result)
	}
	if result[0].text != "Hi" || result[1].text != "Go" {
		t.Errorf("expected ['Hi', 'Go'], got [%q, %q]", result[0].text, result[1].text)
	}
}

func TestCoalesceChars_SplitsOnDifferentLines(t *testing.T) {
	// "Top" on Y=100, "Bot" on Y=80
	texts := []ledongpdf.Text{
		{S: "T", X: 10, Y: 100, FontSize: 12, W: 7},
		{S: "o", X: 17, Y: 100, FontSize: 12, W: 7},
		{S: "p", X: 24, Y: 100, FontSize: 12, W: 7},
		{S: "B", X: 10, Y: 80, FontSize: 12, W: 7},
		{S: "o", X: 17, Y: 80, FontSize: 12, W: 7},
		{S: "t", X: 24, Y: 80, FontSize: 12, W: 7},
	}
	result := coalesceChars(texts)
	if len(result) != 2 {
		t.Fatalf("expected 2 words, got %d", len(result))
	}
	if result[0].text != "Top" || result[1].text != "Bot" {
		t.Errorf("expected ['Top', 'Bot'], got [%q, %q]", result[0].text, result[1].text)
	}
}

func TestCoalesceChars_SkipsEmptyStrings(t *testing.T) {
	texts := []ledongpdf.Text{
		{S: "", X: 10, Y: 100, FontSize: 12, W: 7},
		{S: "A", X: 20, Y: 100, FontSize: 12, W: 7},
		{S: "", X: 30, Y: 100, FontSize: 12, W: 7},
	}
	result := coalesceChars(texts)
	if len(result) != 1 {
		t.Fatalf("expected 1 word, got %d", len(result))
	}
	if result[0].text != "A" {
		t.Errorf("expected 'A', got %q", result[0].text)
	}
}

// --- New two-threshold tests ---

func TestCoalesceChars_DetectsWordBoundaries(t *testing.T) {
	// "Hi" and "Go" separated by a gap larger than wordSpace (16pt > 8pt)
	// H(10,W=7) → rightEdge=17; i(17) → gap=0 → same word "Hi", rightEdge=24
	// G(40) → gap=40-24=16 > wordSpace=8 → separate word
	texts := []ledongpdf.Text{
		{S: "H", X: 10, Y: 100, FontSize: 12, W: 7},
		{S: "i", X: 17, Y: 100, FontSize: 12, W: 7},
		{S: "G", X: 40, Y: 100, FontSize: 12, W: 7},
		{S: "o", X: 47, Y: 100, FontSize: 12, W: 7},
	}
	result := coalesceChars(texts)
	if len(result) != 2 {
		t.Fatalf("expected 2 words, got %d: %+v", len(result), result)
	}
	if result[0].text != "Hi" {
		t.Errorf("expected word 0 'Hi', got %q", result[0].text)
	}
	if result[1].text != "Go" {
		t.Errorf("expected word 1 'Go', got %q", result[1].text)
	}
}

func TestCoalesceChars_InsertsSpaceBetweenWords(t *testing.T) {
	// "The" then "cert" with a moderate gap: rightEdge of 'e'=31, c at X=38
	// gap = 38-31 = 7, charSpace=2 < 7 < wordSpace=8 → space insertion
	texts := []ledongpdf.Text{
		{S: "T", X: 10, Y: 100, FontSize: 12, W: 7},
		{S: "h", X: 17, Y: 100, FontSize: 12, W: 7},
		{S: "e", X: 24, Y: 100, FontSize: 12, W: 7},
		{S: "c", X: 38, Y: 100, FontSize: 12, W: 7},
		{S: "e", X: 45, Y: 100, FontSize: 12, W: 7},
		{S: "r", X: 52, Y: 100, FontSize: 12, W: 7},
		{S: "t", X: 59, Y: 100, FontSize: 12, W: 7},
	}
	result := coalesceChars(texts)
	if len(result) != 1 {
		t.Fatalf("expected 1 word, got %d: %+v", len(result), result)
	}
	if result[0].text != "The cert" {
		t.Errorf("expected 'The cert', got %q", result[0].text)
	}
}

func TestCoalesceChars_PreservesWordOrder(t *testing.T) {
	// Three words "First Second Third" on same line, each separated by gap > wordSpace
	// "First" at X=10..45 (W=7 on 6 chars, rightEdge=45+7=52)
	// "Second": S at X=68, gap = 68-52 = 16 > wordSpace=8 → separate word
	// "Third": T at X=128, gap > wordSpace → separate word
	texts := []ledongpdf.Text{
		{S: "F", X: 10, Y: 100, FontSize: 12, W: 7},
		{S: "i", X: 17, Y: 100, FontSize: 12, W: 7},
		{S: "r", X: 24, Y: 100, FontSize: 12, W: 7},
		{S: "s", X: 31, Y: 100, FontSize: 12, W: 7},
		{S: "t", X: 38, Y: 100, FontSize: 12, W: 7},
		{S: "S", X: 68, Y: 100, FontSize: 12, W: 7},
		{S: "e", X: 75, Y: 100, FontSize: 12, W: 7},
		{S: "c", X: 82, Y: 100, FontSize: 12, W: 7},
		{S: "o", X: 89, Y: 100, FontSize: 12, W: 7},
		{S: "n", X: 96, Y: 100, FontSize: 12, W: 7},
		{S: "d", X: 103, Y: 100, FontSize: 12, W: 7},
		{S: "T", X: 128, Y: 100, FontSize: 12, W: 7},
		{S: "h", X: 135, Y: 100, FontSize: 12, W: 7},
		{S: "i", X: 142, Y: 100, FontSize: 12, W: 7},
		{S: "r", X: 149, Y: 100, FontSize: 12, W: 7},
		{S: "d", X: 156, Y: 100, FontSize: 12, W: 7},
	}
	result := coalesceChars(texts)
	if len(result) != 3 {
		t.Fatalf("expected 3 words, got %d: %+v", len(result), result)
	}
	if result[0].text != "First" {
		t.Errorf("expected word 0 'First', got %q", result[0].text)
	}
	if result[1].text != "Second" {
		t.Errorf("expected word 1 'Second', got %q", result[1].text)
	}
	if result[2].text != "Third" {
		t.Errorf("expected word 2 'Third', got %q", result[2].text)
	}
}

func TestCoalesceChars_SkipsZeroFontSize(t *testing.T) {
	// Characters with FontSize=0 are skipped entirely (no update to lastX/lastW).
	// Gap from A's right edge (17) to C's X (22) = 5, which is < wordSpaceThreshold (5.6),
	// so a space is inserted between them.
	texts := []ledongpdf.Text{
		{S: "A", X: 10, Y: 100, FontSize: 12, W: 7},
		{S: "B", X: 17, Y: 100, FontSize: 0, W: 0}, // should be skipped
		{S: "C", X: 22, Y: 100, FontSize: 12, W: 7},
	}
	result := coalesceChars(texts)
	if len(result) != 1 {
		t.Fatalf("expected 1 word, got %d: %+v", len(result), result)
	}
	// "A C" — the zero-fontSize character is skipped but gap still produces a space
	if result[0].text != "A C" {
		t.Errorf("expected 'A C', got %q", result[0].text)
	}
}

func TestCoalesceChars_LargeGapSeparatesWords(t *testing.T) {
	// Words far apart on same line (> wordSpace) become separate word objects
	texts := []ledongpdf.Text{
		{S: "A", X: 10, Y: 100, FontSize: 12, W: 10},
		{S: "B", X: 200, Y: 100, FontSize: 12, W: 10},
		{S: "C", X: 210, Y: 100, FontSize: 12, W: 10},
	}
	result := coalesceChars(texts)
	if len(result) != 2 {
		t.Fatalf("expected 2 words, got %d: %+v", len(result), result)
	}
	if result[0].text != "A" {
		t.Errorf("expected word 0 'A', got %q", result[0].text)
	}
	if result[1].text != "BC" {
		t.Errorf("expected word 1 'BC', got %q", result[1].text)
	}
}

func TestCoalesceChars_PunctuationNearWords(t *testing.T) {
	// Punctuation attaches to preceding word
	// "word" + "." (tight gap) then "Next" (large gap)
	texts := []ledongpdf.Text{
		{S: "w", X: 10, Y: 100, FontSize: 12, W: 7},
		{S: "o", X: 17, Y: 100, FontSize: 12, W: 7},
		{S: "r", X: 24, Y: 100, FontSize: 12, W: 7},
		{S: "d", X: 31, Y: 100, FontSize: 12, W: 7},
		{S: ".", X: 38, Y: 100, FontSize: 12, W: 4}, // narrow period
		{S: "N", X: 70, Y: 100, FontSize: 12, W: 7},
		{S: "e", X: 77, Y: 100, FontSize: 12, W: 7},
		{S: "x", X: 84, Y: 100, FontSize: 12, W: 7},
		{S: "t", X: 91, Y: 100, FontSize: 12, W: 7},
	}
	result := coalesceChars(texts)
	if len(result) != 2 {
		t.Fatalf("expected 2 words, got %d: %+v", len(result), result)
	}
	if result[0].text != "word." {
		t.Errorf("expected word 0 'word.', got %q", result[0].text)
	}
	if result[1].text != "Next" {
		t.Errorf("expected word 1 'Next', got %q", result[1].text)
	}
}

func TestCoalesceChars_HyphenatedTerms(t *testing.T) {
	// "Asia-Pacific" is a single hyphenated word (all gaps < charSpace)
	// followed by "specifically" with a moderate gap (charSpace < gap < wordSpace)
	// right edge after 'c' at X=85,W=7 → 92; s of "specifically" at X=97 → gap=5
	// charSpace=2 < 5 < wordSpace=8 → space insertion
	texts := []ledongpdf.Text{
		{S: "A", X: 10, Y: 100, FontSize: 12, W: 7},
		{S: "s", X: 17, Y: 100, FontSize: 12, W: 7}, {S: "i", X: 24, Y: 100, FontSize: 12, W: 7},
		{S: "a", X: 31, Y: 100, FontSize: 12, W: 7},
		{S: "-", X: 38, Y: 100, FontSize: 12, W: 5},
		{S: "P", X: 43, Y: 100, FontSize: 12, W: 7}, {S: "a", X: 50, Y: 100, FontSize: 12, W: 7},
		{S: "c", X: 57, Y: 100, FontSize: 12, W: 7}, {S: "i", X: 64, Y: 100, FontSize: 12, W: 7},
		{S: "f", X: 71, Y: 100, FontSize: 12, W: 7}, {S: "i", X: 78, Y: 100, FontSize: 12, W: 7},
		{S: "c", X: 85, Y: 100, FontSize: 12, W: 7},
		{S: "s", X: 97, Y: 100, FontSize: 12, W: 7},
		{S: "p", X: 104, Y: 100, FontSize: 12, W: 7}, {S: "e", X: 111, Y: 100, FontSize: 12, W: 7},
		{S: "c", X: 118, Y: 100, FontSize: 12, W: 7}, {S: "i", X: 125, Y: 100, FontSize: 12, W: 7},
		{S: "f", X: 132, Y: 100, FontSize: 12, W: 7}, {S: "i", X: 139, Y: 100, FontSize: 12, W: 7},
		{S: "c", X: 146, Y: 100, FontSize: 12, W: 7},
		{S: "a", X: 153, Y: 100, FontSize: 12, W: 7}, {S: "l", X: 160, Y: 100, FontSize: 12, W: 7},
		{S: "l", X: 167, Y: 100, FontSize: 12, W: 7}, {S: "y", X: 174, Y: 100, FontSize: 12, W: 7},
	}
	result := coalesceChars(texts)
	if len(result) != 1 {
		t.Fatalf("expected 1 word, got %d: %+v", len(result), result)
	}
	if result[0].text != "Asia-Pacific specifically" {
		t.Errorf("expected 'Asia-Pacific specifically', got %q", result[0].text)
	}
}

func TestCoalesceChars_MixedFontSizes(t *testing.T) {
	// "Big" (fontSize=24) then "small" (fontSize=12) on same line.
	// Right edge of "Big": g(38,W=14) → 52.
	// s at X=65: gap=65-52=13. Thresholds use current char's fontSize (12):
	//   charSpace=2, wordSpace=8. Since 13 > 8, this is a separate word.
	texts := []ledongpdf.Text{
		{S: "B", X: 10, Y: 100, FontSize: 24, W: 14},
		{S: "i", X: 24, Y: 100, FontSize: 24, W: 14},
		{S: "g", X: 38, Y: 100, FontSize: 24, W: 14},
		{S: "s", X: 65, Y: 100, FontSize: 12, W: 7},
		{S: "m", X: 72, Y: 100, FontSize: 12, W: 7},
		{S: "a", X: 79, Y: 100, FontSize: 12, W: 7},
		{S: "l", X: 86, Y: 100, FontSize: 12, W: 7},
		{S: "l", X: 93, Y: 100, FontSize: 12, W: 7},
	}
	result := coalesceChars(texts)
	if len(result) != 2 {
		t.Fatalf("expected 2 words (%dpt + %dpt separated by 13pt gap), got %d: %+v", 24, 12, len(result), result)
	}
	if result[0].text != "Big" {
		t.Errorf("expected word 0 'Big', got %q", result[0].text)
	}
	if result[1].text != "small" {
		t.Errorf("expected word 1 'small', got %q", result[1].text)
	}
}

// --- groupIntoRows tests ---

func TestGroupIntoRows_Empty(t *testing.T) {
	rows := groupIntoRows(nil, 3.0)
	if len(rows) != 0 {
		t.Errorf("expected empty, got %d rows", len(rows))
	}
}

func TestGroupIntoRows_GroupsByYTolerance(t *testing.T) {
	words := []word{
		{x: 10, y: 100, text: "A"},
		{x: 200, y: 101, text: "B"}, // within 3pt tolerance of Y=100
		{x: 10, y: 80, text: "C"},
		{x: 200, y: 79, text: "D"}, // within 3pt tolerance of Y=80
	}
	rows := groupIntoRows(words, 3.0)
	if len(rows) != 2 {
		t.Fatalf("expected 2 rows, got %d", len(rows))
	}
	// First row (top) should have A and B
	if len(rows[0].cells) != 2 {
		t.Errorf("row 0: expected 2 cells, got %d", len(rows[0].cells))
	}
	// Second row should have C and D
	if len(rows[1].cells) != 2 {
		t.Errorf("row 1: expected 2 cells, got %d", len(rows[1].cells))
	}
}

func TestGroupIntoRows_SortedTopToBottom(t *testing.T) {
	words := []word{
		{x: 10, y: 50, text: "Bottom"},
		{x: 10, y: 200, text: "Top"},
		{x: 10, y: 100, text: "Middle"},
	}
	rows := groupIntoRows(words, 3.0)
	if len(rows) != 3 {
		t.Fatalf("expected 3 rows, got %d", len(rows))
	}
	if rows[0].cells[0].text != "Top" {
		t.Errorf("first row should be 'Top', got %q", rows[0].cells[0].text)
	}
	if rows[2].cells[0].text != "Bottom" {
		t.Errorf("last row should be 'Bottom', got %q", rows[2].cells[0].text)
	}
}

func TestGroupIntoRows_CellsSortedLeftToRight(t *testing.T) {
	words := []word{
		{x: 300, y: 100, text: "Third"},
		{x: 10, y: 100, text: "First"},
		{x: 150, y: 100, text: "Second"},
	}
	rows := groupIntoRows(words, 3.0)
	if len(rows) != 1 {
		t.Fatalf("expected 1 row, got %d", len(rows))
	}
	if rows[0].cells[0].text != "First" {
		t.Errorf("expected first cell 'First', got %q", rows[0].cells[0].text)
	}
	if rows[0].cells[2].text != "Third" {
		t.Errorf("expected last cell 'Third', got %q", rows[0].cells[2].text)
	}
}

// --- detectColumnPositions tests ---

func TestDetectColumnPositions_SimpleTable(t *testing.T) {
	// 3 columns at x≈72, x≈250, x≈450 across 4 rows
	rows := []tableRow{
		{y: 400, cells: []tableCell{{x: 72, text: "Name"}, {x: 250, text: "Age"}, {x: 450, text: "City"}}},
		{y: 380, cells: []tableCell{{x: 72, text: "Alice"}, {x: 250, text: "30"}, {x: 450, text: "NYC"}}},
		{y: 360, cells: []tableCell{{x: 72, text: "Bob"}, {x: 250, text: "25"}, {x: 450, text: "LA"}}},
		{y: 340, cells: []tableCell{{x: 72, text: "Eve"}, {x: 250, text: "28"}, {x: 450, text: "SF"}}},
	}

	cols := detectColumnPositions(rows, 3)
	if cols == nil {
		t.Fatal("expected columns to be detected, got nil")
	}
	if len(cols) != 3 {
		t.Errorf("expected 3 columns, got %d: %v", len(cols), cols)
	}
}

func TestDetectColumnPositions_NotEnoughColumns(t *testing.T) {
	// Only 2 distinct X positions — should not be a table
	rows := []tableRow{
		{y: 400, cells: []tableCell{{x: 72, text: "Hello"}, {x: 250, text: "World"}}},
		{y: 380, cells: []tableCell{{x: 72, text: "Foo"}, {x: 250, text: "Bar"}}},
	}

	cols := detectColumnPositions(rows, 3)
	if cols != nil {
		t.Errorf("expected nil (not enough columns), got %v", cols)
	}
}

func TestDetectColumnPositions_CloseColumnsAreMerged(t *testing.T) {
	// Characters close together (< 50pt gap) should merge into one column
	rows := []tableRow{
		{y: 400, cells: []tableCell{{x: 72, text: "A"}, {x: 80, text: "B"}, {x: 90, text: "C"}, {x: 100, text: "D"}}},
		{y: 380, cells: []tableCell{{x: 72, text: "E"}, {x: 80, text: "F"}, {x: 90, text: "G"}, {x: 100, text: "H"}}},
	}

	cols := detectColumnPositions(rows, 3)
	if cols != nil {
		t.Errorf("expected nil (columns too close together), got %v", cols)
	}
}

func TestDetectColumnPositions_MergesNearbyKeepsDistant(t *testing.T) {
	// Two groups: close chars at x≈72-80, then far away at x≈300-310, then x≈500
	rows := []tableRow{
		{y: 400, cells: []tableCell{{x: 72, text: "A"}, {x: 80, text: "B"}, {x: 300, text: "C"}, {x: 310, text: "D"}, {x: 500, text: "E"}}},
		{y: 380, cells: []tableCell{{x: 72, text: "F"}, {x: 80, text: "G"}, {x: 300, text: "H"}, {x: 310, text: "I"}, {x: 500, text: "J"}}},
		{y: 360, cells: []tableCell{{x: 72, text: "K"}, {x: 80, text: "L"}, {x: 300, text: "M"}, {x: 310, text: "N"}, {x: 500, text: "O"}}},
	}

	cols := detectColumnPositions(rows, 3)
	if cols == nil {
		t.Fatal("expected columns to be detected")
	}
	if len(cols) != 3 {
		t.Errorf("expected 3 merged columns, got %d: %v", len(cols), cols)
	}
}

// --- assignCellsToColumns tests ---

func TestAssignCellsToColumns_BasicAssignment(t *testing.T) {
	columns := []float64{70, 250, 450}
	cells := []tableCell{
		{x: 72, text: "Alice"},
		{x: 252, text: "30"},
		{x: 448, text: "NYC"},
	}

	result := assignCellsToColumns(cells, columns)
	if len(result) != 3 {
		t.Fatalf("expected 3 entries, got %d", len(result))
	}
	if result[0] != "Alice" || result[1] != "30" || result[2] != "NYC" {
		t.Errorf("unexpected assignment: %v", result)
	}
}

func TestAssignCellsToColumns_MultipleCellsSameColumn(t *testing.T) {
	columns := []float64{70, 250, 450}
	cells := []tableCell{
		{x: 72, text: "Hello"},
		{x: 78, text: "World"}, // close to 70, should merge into column 0
		{x: 252, text: "Data"},
		{x: 448, text: "End"},
	}

	result := assignCellsToColumns(cells, columns)
	if result[0] != "Hello World" {
		t.Errorf("expected 'Hello World' in column 0, got %q", result[0])
	}
	if result[1] != "Data" {
		t.Errorf("expected 'Data' in column 1, got %q", result[1])
	}
}

func TestAssignCellsToColumns_EmptyRow(t *testing.T) {
	columns := []float64{70, 250, 450}
	cells := []tableCell{}

	result := assignCellsToColumns(cells, columns)
	for i, r := range result {
		if r != "" {
			t.Errorf("expected empty string at column %d, got %q", i, r)
		}
	}
}

// --- findTableEnd tests ---

func TestFindTableEnd_DetectsTableRegion(t *testing.T) {
	rows := []tableRow{
		// Table rows: 3+ columns with 50pt+ gaps
		{y: 400, cells: []tableCell{{x: 72, text: "A"}, {x: 250, text: "B"}, {x: 450, text: "C"}}},
		{y: 380, cells: []tableCell{{x: 72, text: "D"}, {x: 250, text: "E"}, {x: 450, text: "F"}}},
		// Non-table row: single column
		{y: 360, cells: []tableCell{{x: 72, text: "Just a paragraph"}}},
	}

	end := findTableEnd(rows, 0, 3)
	if end != 2 {
		t.Errorf("expected table to end at index 2, got %d", end)
	}
}

func TestFindTableEnd_NoTableFromStart(t *testing.T) {
	rows := []tableRow{
		{y: 400, cells: []tableCell{{x: 72, text: "Just text"}}},
	}

	end := findTableEnd(rows, 0, 3)
	if end != 0 {
		t.Errorf("expected 0 (no table), got %d", end)
	}
}

func TestFindTableEnd_BodyTextNotTable(t *testing.T) {
	// Body text has many characters but all at tightly-packed X positions (< 50pt gaps)
	rows := []tableRow{
		{y: 400, cells: []tableCell{{x: 10, text: "T"}, {x: 17, text: "h"}, {x: 24, text: "e"}, {x: 40, text: "q"}, {x: 47, text: "u"}}},
	}

	end := findTableEnd(rows, 0, 3)
	if end != 0 {
		t.Errorf("expected 0 (body text is not a table), got %d", end)
	}
}

// --- buildPageBlocks tests ---

func TestBuildPageBlocks_TableRendered(t *testing.T) {
	rows := []tableRow{
		{y: 400, cells: []tableCell{{x: 72, text: "Name"}, {x: 250, text: "Age"}, {x: 450, text: "City"}}},
		{y: 380, cells: []tableCell{{x: 72, text: "Alice"}, {x: 250, text: "30"}, {x: 450, text: "NYC"}}},
		{y: 360, cells: []tableCell{{x: 72, text: "Bob"}, {x: 250, text: "25"}, {x: 450, text: "LA"}}},
	}

	blocks := buildPageBlocks(rows)

	if len(blocks) != 1 || blocks[0].Type != domain.BlockTypeTable {
		t.Fatalf("expected 1 table block, got %d blocks", len(blocks))
	}
	if len(blocks[0].Table.Rows) != 3 {
		t.Errorf("expected 3 table rows, got %d", len(blocks[0].Table.Rows))
	}
	if blocks[0].Table.Rows[0][0] != "Name" {
		t.Errorf("expected table header 'Name', got %q", blocks[0].Table.Rows[0][0])
	}
}

func TestBuildPageBlocks_PlainTextNotTable(t *testing.T) {
	// Single-column rows should NOT be rendered as table
	rows := []tableRow{
		{y: 400, cells: []tableCell{{x: 72, text: "Hello world"}}},
		{y: 380, cells: []tableCell{{x: 72, text: "Another paragraph"}}},
	}

	blocks := buildPageBlocks(rows)

	if len(blocks) != 2 {
		t.Fatalf("expected 2 text blocks, got %d", len(blocks))
	}
	if blocks[0].Type != domain.BlockTypeText || blocks[0].Text != "Hello world" {
		t.Errorf("expected text block 'Hello world', got %v", blocks[0])
	}
}

func TestBuildPageBlocks_MixedTableAndText(t *testing.T) {
	rows := []tableRow{
		// Plain text
		{y: 500, cells: []tableCell{{x: 72, text: "Intro paragraph"}}},
		// Table
		{y: 400, cells: []tableCell{{x: 72, text: "Col1"}, {x: 250, text: "Col2"}, {x: 450, text: "Col3"}}},
		{y: 380, cells: []tableCell{{x: 72, text: "A"}, {x: 250, text: "B"}, {x: 450, text: "C"}}},
		{y: 360, cells: []tableCell{{x: 72, text: "D"}, {x: 250, text: "E"}, {x: 450, text: "F"}}},
		// Plain text after table
		{y: 300, cells: []tableCell{{x: 72, text: "Conclusion"}}},
	}

	blocks := buildPageBlocks(rows)

	if len(blocks) != 3 {
		t.Fatalf("expected 3 blocks, got %d", len(blocks))
	}
	if blocks[0].Type != domain.BlockTypeText || blocks[0].Text != "Intro paragraph" {
		t.Error("intro text missing or wrong type")
	}
	if blocks[1].Type != domain.BlockTypeTable {
		t.Error("table block missing or wrong type")
	}
	if blocks[2].Type != domain.BlockTypeText || blocks[2].Text != "Conclusion" {
		t.Error("conclusion text missing or wrong type")
	}
}

// --- New tests for Phase 2 ---

func TestExtractColumnClusters_GroupsCloseCoordinates(t *testing.T) {
	cells := []tableCell{
		{x: 10, text: "A"},
		{x: 12, text: "B"},
		{x: 65, text: "C"},
		{x: 70, text: "D"},
		{x: 120, text: "E"},
	}
	// With minGap 50, {10, 12} -> cluster 10, {65, 70} -> cluster 65, {120} -> cluster 120
	clusters := extractColumnClusters(cells, 50.0)
	if len(clusters) != 3 {
		t.Fatalf("expected 3 clusters, got %d", len(clusters))
	}
	if clusters[0] != 10 || clusters[1] != 65 || clusters[2] != 120 {
		t.Errorf("unexpected clusters: %v", clusters)
	}
}

func TestCoalesceChars_StackedCoordinates(t *testing.T) {
	// Fake bold where the exact same character is printed with a tiny offset
	// We no longer deduplicate aggressively to avoid collapsing legitimate double letters.
	texts := []ledongpdf.Text{
		{S: "A", X: 10.0, Y: 100, FontSize: 12, W: 7},
		{S: "A", X: 10.1, Y: 100, FontSize: 12, W: 7},
		{S: "B", X: 17.0, Y: 100, FontSize: 12, W: 7},
	}
	result := coalesceChars(texts)
	if len(result) != 1 {
		t.Fatalf("expected 1 word, got %d", len(result))
	}
	// Since deduplication is disabled, we expect "AAB"
	if result[0].text != "AAB" {
		t.Errorf("expected 'AAB', got %q", result[0].text)
	}
}

func TestCoalesceChars_DoubleLetters(t *testing.T) {
	// Legitimate double letters (e.g., 'ww' in www)
	// These are adjacent but NOT stacked. Overlap should be near zero.
	// w1: (10, 100, W=10) -> rightEdge=20
	// w2: (20, 100, W=10) -> rightEdge=30
	texts := []ledongpdf.Text{
		{S: "w", X: 10.0, Y: 100, FontSize: 12, W: 10},
		{S: "w", X: 20.0, Y: 100, FontSize: 12, W: 10},
		{S: "w", X: 30.0, Y: 100, FontSize: 12, W: 10},
	}
	result := coalesceChars(texts)
	if len(result) != 1 {
		t.Fatalf("expected 1 word, got %d", len(result))
	}
	if result[0].text != "www" {
		t.Errorf("expected 'www', got %q", result[0].text)
	}
}

func TestCoalesceChars_SpacedHeading(t *testing.T) {
	// Spaced heading: "T E S T" with consistent wide gaps
	// medianW=7, gap=5. charSpaceThreshold is usually 0.2*medianW=1.4
	// wordSpaceThreshold is 0.8*medianW=5.6
	// In a normal line, 5pt gap would be a space.
	// In a spaced heading line (low stddev), it should coalesce without spaces.
	texts := []ledongpdf.Text{
		{S: "T", X: 10, Y: 100, FontSize: 12, W: 7},
		{S: "E", X: 22, Y: 100, FontSize: 12, W: 7}, // gap=5
		{S: "S", X: 34, Y: 100, FontSize: 12, W: 7}, // gap=5
		{S: "T", X: 46, Y: 100, FontSize: 12, W: 7}, // gap=5
	}
	result := coalesceChars(texts)
	if len(result) != 1 {
		t.Fatalf("expected 1 word, got %d", len(result))
	}
	if result[0].text != "TEST" {
		t.Errorf("expected 'TEST', got %q", result[0].text)
	}
}

func TestSanitizeText_Ligatures(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{"fi ligature", "ﬁ", "fi"},
		{"fl ligature", "ﬂ", "fl"},
		{"combined", "Brieﬁng", "Briefing"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := sanitizeText(tt.in)
			if got != tt.want {
				t.Errorf("sanitizeText(%q) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}
