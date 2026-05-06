package converter

import (
	"fmt"
	"math"
	"sort"
	"strings"

	"github.com/ledongthuc/pdf"
)

// word represents a coalesced word with its bounding position.
type word struct {
	x    float64
	y    float64
	text string
}

// tableRow is a row of cells detected in a table.
type tableRow struct {
	y     float64
	cells []tableCell
}

// tableCell is a single cell with its X position and text.
type tableCell struct {
	x    float64
	text string
}

// extractWithTables uses positional text data to detect and preserve table structure.
// Returns the page content as markdown, with tables formatted as pipe tables.
func extractWithTables(page pdf.Page) string {
	content := page.Content()
	if len(content.Text) == 0 {
		return ""
	}

	// Step 1: Coalesce individual characters into words
	words := coalesceChars(content.Text)
	if len(words) == 0 {
		return ""
	}

	// Step 2: Group words into rows by Y coordinate
	rows := groupIntoRows(words, 3.0)

	// Step 3: Detect column positions across rows to find table regions
	// A "table" is a contiguous sequence of rows where each row has cells
	// landing on similar X positions (3+ columns consistently)
	return renderRowsAsMarkdown(rows)
}

// coalesceChars merges individual characters into words based on X proximity and Y alignment.
func coalesceChars(texts []pdf.Text) []word {
	if len(texts) == 0 {
		return nil
	}

	// Sort by Y descending (top to bottom), then X ascending (left to right)
	sorted := make([]pdf.Text, len(texts))
	copy(sorted, texts)
	sort.Slice(sorted, func(i, j int) bool {
		if math.Abs(sorted[i].Y-sorted[j].Y) > 1.0 {
			return sorted[i].Y > sorted[j].Y
		}
		return sorted[i].X < sorted[j].X
	})

	var words []word
	var current word
	var lastX float64
	var lastY float64
	started := false

	for _, t := range sorted {
		s := t.S
		if s == "" {
			continue
		}

		if !started {
			current = word{x: t.X, y: t.Y, text: s}
			lastX = t.X
			lastY = t.Y
			started = true
			continue
		}

		sameLine := math.Abs(t.Y-lastY) < 2.0
		// Estimate character width from font size (rough: 0.6 * fontSize)
		charWidth := t.FontSize * 0.6
		if charWidth < 3.0 {
			charWidth = 5.0
		}
		closeEnough := (t.X - lastX) < charWidth*1.8

		if sameLine && closeEnough {
			// Same word — append character
			current.text += s
		} else {
			// Flush current word
			if strings.TrimSpace(current.text) != "" {
				words = append(words, current)
			}
			current = word{x: t.X, y: t.Y, text: s}
		}
		lastX = t.X
		lastY = t.Y
	}
	// Flush last word
	if strings.TrimSpace(current.text) != "" {
		words = append(words, current)
	}

	return words
}

// groupIntoRows groups words into rows based on Y-coordinate proximity.
func groupIntoRows(words []word, tolerance float64) []tableRow {
	var rows []tableRow

	for _, w := range words {
		found := false
		for i := range rows {
			if math.Abs(rows[i].y-w.y) < tolerance {
				rows[i].cells = append(rows[i].cells, tableCell{x: w.x, text: w.text})
				found = true
				break
			}
		}
		if !found {
			rows = append(rows, tableRow{
				y:     w.y,
				cells: []tableCell{{x: w.x, text: w.text}},
			})
		}
	}

	// Sort rows top to bottom (Y descending in PDF coordinates)
	sort.Slice(rows, func(i, j int) bool {
		return rows[i].y > rows[j].y
	})

	// Sort cells within each row left to right
	for i := range rows {
		sort.Slice(rows[i].cells, func(a, b int) bool {
			return rows[i].cells[a].x < rows[i].cells[b].x
		})
	}

	return rows
}

// detectColumnPositions analyzes a range of rows to find consistent column X positions.
// Returns the detected column positions if 3+ columns are found, nil otherwise.
func detectColumnPositions(rows []tableRow, minColumns int) []float64 {
	// Count how often each X position appears across rows (rounded to nearest 5pt)
	xFreq := make(map[int]int)
	for _, row := range rows {
		seen := make(map[int]bool) // avoid double-counting within one row
		for _, cell := range row.cells {
			bucket := int(math.Round(cell.x/5.0)) * 5
			if !seen[bucket] {
				xFreq[bucket]++
				seen[bucket] = true
			}
		}
	}

	// Column positions are X values that appear in at least 40% of the rows
	threshold := int(float64(len(rows)) * 0.4)
	if threshold < 2 {
		threshold = 2
	}

	var columns []float64
	for x, count := range xFreq {
		if count >= threshold {
			columns = append(columns, float64(x))
		}
	}

	sort.Float64s(columns)

	if len(columns) < minColumns {
		return nil
	}
	return columns
}

// assignCellsToColumns maps each cell in a row to the nearest detected column.
func assignCellsToColumns(cells []tableCell, columns []float64) []string {
	result := make([]string, len(columns))

	for _, cell := range cells {
		bestCol := 0
		bestDist := math.Abs(cell.x - columns[0])
		for j, col := range columns {
			dist := math.Abs(cell.x - col)
			if dist < bestDist {
				bestDist = dist
				bestCol = j
			}
		}
		if result[bestCol] != "" {
			result[bestCol] += " " + cell.text
		} else {
			result[bestCol] = cell.text
		}
	}

	return result
}

// renderRowsAsMarkdown converts rows to markdown, detecting and formatting tables.
func renderRowsAsMarkdown(rows []tableRow) string {
	var buf strings.Builder

	// Use a sliding window to detect table regions
	i := 0
	for i < len(rows) {
		// Look ahead to see if the next N rows form a table
		tableEnd := findTableEnd(rows, i, 3)

		if tableEnd > i+1 {
			// We found a table region from rows[i] to rows[tableEnd-1]
			tableRows := rows[i:tableEnd]
			columns := detectColumnPositions(tableRows, 3)

			if columns != nil {
				// Render as markdown table
				for ri, row := range tableRows {
					cells := assignCellsToColumns(row.cells, columns)
					buf.WriteString("| ")
					buf.WriteString(strings.Join(cells, " | "))
					buf.WriteString(" |\n")

					// Add separator after first row (header)
					if ri == 0 {
						buf.WriteString("|")
						for range columns {
							buf.WriteString(" --- |")
						}
						buf.WriteString("\n")
					}
				}
				buf.WriteString("\n")
				i = tableEnd
				continue
			}
		}

		// Not a table — render as plain text paragraph
		var lineWords []string
		for _, cell := range rows[i].cells {
			lineWords = append(lineWords, cell.text)
		}
		line := strings.Join(lineWords, " ")
		if strings.TrimSpace(line) != "" {
			buf.WriteString(line)
			buf.WriteString("\n")
		}
		i++
	}

	return buf.String()
}

// findTableEnd looks for the end of a contiguous table region starting at startIdx.
// A table row is defined as having cells in 3+ distinct X-position buckets.
func findTableEnd(rows []tableRow, startIdx, minColumns int) int {
	end := startIdx

	for end < len(rows) {
		row := rows[end]
		// Count distinct X-position buckets in this row
		buckets := make(map[int]bool)
		for _, cell := range row.cells {
			bucket := int(math.Round(cell.x/5.0)) * 5
			buckets[bucket] = true
		}
		if len(buckets) >= minColumns {
			end++
		} else {
			break
		}
	}

	return end
}

// formatMarkdownTable is a helper to format a slice of rows as markdown pipe table.
func formatMarkdownTable(header []string, dataRows [][]string) string {
	var buf strings.Builder

	// Header
	buf.WriteString("| ")
	buf.WriteString(strings.Join(header, " | "))
	buf.WriteString(" |\n|")
	for range header {
		buf.WriteString(" --- |")
	}
	buf.WriteString("\n")

	// Data rows
	for _, row := range dataRows {
		// Pad row to match header length
		for len(row) < len(header) {
			row = append(row, "")
		}
		buf.WriteString("| ")
		buf.WriteString(strings.Join(row, " | "))
		buf.WriteString(" |\n")
	}

	return buf.String()
}

// padToLength pads a string slice to the specified length with empty strings.
func padToLength(s []string, length int) []string {
	for len(s) < length {
		s = append(s, "")
	}
	return s
}

// maxCols returns the maximum number of columns across all rows.
func maxCols(rows [][]string) int {
	max := 0
	for _, row := range rows {
		if len(row) > max {
			max = len(row)
		}
	}
	return max
}

// isTableRow heuristically determines if a row looks like it belongs to a table.
// Uses the number of distinct X-position buckets as the primary signal.
func isTableRow(row tableRow, minColumns int) bool {
	buckets := make(map[int]bool)
	for _, cell := range row.cells {
		bucket := int(math.Round(cell.x / 5.0)) * 5
		buckets[bucket] = true
	}
	return len(buckets) >= minColumns
}

// formatTableSeparator generates the markdown separator line for a table.
func formatTableSeparator(numCols int) string {
	parts := make([]string, numCols)
	for i := range parts {
		parts[i] = "---"
	}
	return fmt.Sprintf("| %s |", strings.Join(parts, " | "))
}
