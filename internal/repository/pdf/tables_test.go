package pdf

import (
	"strings"
	"testing"

	ledongpdf "github.com/ledongthuc/pdf"
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
		{S: "A", X: 10, Y: 100, FontSize: 12},
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
	// Simulate "Hello" as 5 characters at 12pt font, ~7pt apart
	texts := []ledongpdf.Text{
		{S: "H", X: 10, Y: 100, FontSize: 12},
		{S: "e", X: 17, Y: 100, FontSize: 12},
		{S: "l", X: 24, Y: 100, FontSize: 12},
		{S: "l", X: 31, Y: 100, FontSize: 12},
		{S: "o", X: 38, Y: 100, FontSize: 12},
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
	// Two words "Hi" and "Go" separated by a large gap
	texts := []ledongpdf.Text{
		{S: "H", X: 10, Y: 100, FontSize: 12},
		{S: "i", X: 17, Y: 100, FontSize: 12},
		{S: "G", X: 200, Y: 100, FontSize: 12},
		{S: "o", X: 207, Y: 100, FontSize: 12},
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
		{S: "T", X: 10, Y: 100, FontSize: 12},
		{S: "o", X: 17, Y: 100, FontSize: 12},
		{S: "p", X: 24, Y: 100, FontSize: 12},
		{S: "B", X: 10, Y: 80, FontSize: 12},
		{S: "o", X: 17, Y: 80, FontSize: 12},
		{S: "t", X: 24, Y: 80, FontSize: 12},
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
		{S: "", X: 10, Y: 100, FontSize: 12},
		{S: "A", X: 20, Y: 100, FontSize: 12},
		{S: "", X: 30, Y: 100, FontSize: 12},
	}
	result := coalesceChars(texts)
	if len(result) != 1 {
		t.Fatalf("expected 1 word, got %d", len(result))
	}
	if result[0].text != "A" {
		t.Errorf("expected 'A', got %q", result[0].text)
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

// --- renderRowsAsMarkdown tests ---

func TestRenderRowsAsMarkdown_TableRenderedAsPipe(t *testing.T) {
	rows := []tableRow{
		{y: 400, cells: []tableCell{{x: 72, text: "Name"}, {x: 250, text: "Age"}, {x: 450, text: "City"}}},
		{y: 380, cells: []tableCell{{x: 72, text: "Alice"}, {x: 250, text: "30"}, {x: 450, text: "NYC"}}},
		{y: 360, cells: []tableCell{{x: 72, text: "Bob"}, {x: 250, text: "25"}, {x: 450, text: "LA"}}},
	}

	result := renderRowsAsMarkdown(rows)

	if !strings.Contains(result, "| Name") {
		t.Error("expected pipe table header with 'Name'")
	}
	if !strings.Contains(result, "| --- |") {
		t.Error("expected pipe table separator")
	}
	if !strings.Contains(result, "| Alice") {
		t.Error("expected pipe table data row with 'Alice'")
	}
}

func TestRenderRowsAsMarkdown_PlainTextNotPiped(t *testing.T) {
	// Single-column rows should NOT be rendered as table
	rows := []tableRow{
		{y: 400, cells: []tableCell{{x: 72, text: "Hello world"}}},
		{y: 380, cells: []tableCell{{x: 72, text: "Another paragraph"}}},
	}

	result := renderRowsAsMarkdown(rows)

	if strings.Contains(result, "|") {
		t.Errorf("plain text should not contain pipe characters, got %q", result)
	}
	if !strings.Contains(result, "Hello world") {
		t.Error("expected plain text to be preserved")
	}
}

func TestRenderRowsAsMarkdown_MixedTableAndText(t *testing.T) {
	rows := []tableRow{
		// Plain text
		{y: 500, cells: []tableCell{{x: 72, text: "Intro paragraph"}}},
		// Table
		{y: 400, cells: []tableCell{{x: 72, text: "Col1"}, {x: 250, text: "Col2"}, {x: 450, text: "Col3"}}},
		{y: 380, cells: []tableCell{{x: 72, text: "A"}, {x: 250, text: "B"}, {x: 450, text: "C"}}},
		// Plain text after table
		{y: 300, cells: []tableCell{{x: 72, text: "Conclusion"}}},
	}

	result := renderRowsAsMarkdown(rows)

	if !strings.Contains(result, "Intro paragraph") {
		t.Error("intro text missing")
	}
	if !strings.Contains(result, "| Col1") {
		t.Error("table not rendered with pipes")
	}
	if !strings.Contains(result, "Conclusion") {
		t.Error("conclusion text missing")
	}
}
