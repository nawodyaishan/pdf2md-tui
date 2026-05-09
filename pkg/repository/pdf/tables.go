package pdf

import (
	"fmt"
	"math"
	"sort"
	"strings"

	ledongpdf "github.com/ledongthuc/pdf"
	"github.com/nawodyaishan/pdf2md-tui/pkg/domain"
	"golang.org/x/text/unicode/norm"
)

const (
	tableMinColumnGap     = 50.0 // minimum gap between columns in points (~0.7 inches)
	tableYRowTolerance    = 3.0  // Y-coordinate proximity to group words into the same row
	tableMinCols          = 2    // minimum number of columns to be considered a table
	tableColumnFreqThresh = 0.3  // percentage of rows a column must appear in to be valid
	tableBucketSize       = 5.0  // bucket size for grouping X-coordinates

	charYToleranceLineMerge = 1.0 // Y-tolerance to group characters into the same line
	charXNoiseFloor         = 0.5 // X-tolerance to consider coordinates stacked or noisy
	charFallbackWidthRatio  = 0.4 // ratio of fontSize to use when width is missing
	charDefaultWidthRatio   = 0.5 // ratio to use for median when width is missing
	charSpaceRatio          = 0.4 // ratio of median width for kerning/stacking gap (G3)
	wordSpaceRatio          = 1.0 // ratio of median width for standard word space
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

// indexedText wraps ledongpdf.Text with its original stream index.
type indexedText struct {
	ledongpdf.Text
	index int
}

// coalesceChars merges individual characters into words based on X proximity and Y alignment.
// It uses a "Cluster-First" approach: grouping by Y-coordinate lines, sorting by X,
// and then applying adaptive thresholding based on median character widths.
func coalesceChars(texts []ledongpdf.Text) []word {
	if len(texts) == 0 {
		return nil
	}

	lines := groupTextsIntoLines(texts)

	var words []word
	for _, line := range lines {
		words = append(words, processLineIntoWords(line)...)
	}

	// 4. Post-processing: Ligatures, PUA, and formatting normalization
	for i := range words {
		words[i].text = sanitizeText(words[i].text)
	}

	return words
}

func groupTextsIntoLines(texts []ledongpdf.Text) [][]indexedText {
	indexed := make([]indexedText, len(texts))
	for i, t := range texts {
		indexed[i] = indexedText{Text: t, index: i}
	}

	// 2. Cluster-First: Group into lines using Y-tolerance
	sort.SliceStable(indexed, func(i, j int) bool {
		return indexed[i].Y > indexed[j].Y
	})

	var lines [][]indexedText
	if len(indexed) > 0 {
		var currentLine []indexedText
		lastY := indexed[0].Y
		for _, t := range indexed {
			if math.Abs(t.Y-lastY) > charYToleranceLineMerge {
				sortLineByX(currentLine)
				lines = append(lines, currentLine)
				currentLine = []indexedText{t}
				lastY = t.Y
			} else {
				currentLine = append(currentLine, t)
			}
		}
		sortLineByX(currentLine)
		lines = append(lines, currentLine)
	}
	return lines
}

func processLineIntoWords(line []indexedText) []word {
	if len(line) == 0 {
		return nil
	}

	// Calculate line-level statistics for adaptive thresholds.
	var widths []float64
	var gaps []float64
	var lastRight float64
	processedFirst := false
	for _, t := range line {
		if t.S == "" || t.FontSize <= 0 {
			continue
		}
		w := t.W
		if w <= 0 {
			w = t.FontSize * charDefaultWidthRatio
		}
		widths = append(widths, w)
		if processedFirst {
			gap := t.X - lastRight
			gaps = append(gaps, gap)
		}
		lastRight = t.X + w
		processedFirst = true
	}
	medianW := calculateMedian(widths)
	meanGap := calculateMean(gaps)
	stdDevGap := calculateStdDev(gaps)

	relStdDev := 1.0
	if meanGap > 0 {
		relStdDev = stdDevGap / meanGap
	}

	// G3: Spaced headings have highly consistent gaps (low relStdDev).
	// We've seen headings with relStdDev ~0.8 still failing to coalesce.
	isSpacedHeading := len(gaps) >= 2 && relStdDev < 1.0 && meanGap > (medianW*charSpaceRatio)

	charSpaceThreshold := medianW * charSpaceRatio
	wordSpaceThreshold := medianW * wordSpaceRatio

	if isSpacedHeading {
		// Increase thresholds to prevent splitting intentional letter-spacing
		// Use the max gap on the line to ensure everything is coalesced.
		maxGap := 0.0
		for _, g := range gaps {
			if g > maxGap {
				maxGap = g
			}
		}
		charSpaceThreshold = maxGap + 2.0
		wordSpaceThreshold = maxGap * 2.0
	}

	var words []word
	var currentWord word
	var lastX, lastW float64
	started := false

	var lastIndex int
	for _, t := range line {
		s := t.S
		if s == "" || t.FontSize <= 0 {
			continue
		}

		w := t.W
		if w <= 0 {
			w = t.FontSize * charFallbackWidthRatio
		}

		if !started {
			currentWord = word{x: t.X, y: t.Y, text: s}
			lastX = t.X
			lastW = w
			lastIndex = t.index
			started = true
			continue
		}

		// Deduplication (Geometric Fix)
		// We avoid aggressive deduplication because it often collapses legitimate
		// double letters in low-quality or uniquely encoded PDFs.
		// Modern PDF libraries like ledongthuc/pdf often handle basic deduplication.

		gap := t.X - (lastX + lastW)
		isSequential := t.index == lastIndex+1

		effectiveWordSpace := wordSpaceThreshold
		if isSequential {
			if t.W <= 0 {
				effectiveWordSpace = 1000.0 // Virtually infinite
			} else {
				effectiveWordSpace = math.Max(wordSpaceThreshold*1.5, medianW*1.5)
			}
		}

		if gap < charSpaceThreshold {
			currentWord.text += s
		} else if gap < effectiveWordSpace {
			currentWord.text += " " + s
		} else {
			if strings.TrimSpace(currentWord.text) != "" {
				words = append(words, currentWord)
			}
			currentWord = word{x: t.X, y: t.Y, text: s}
		}
		lastX = t.X
		lastW = w
		lastIndex = t.index
	}
	if started && strings.TrimSpace(currentWord.text) != "" {
		words = append(words, currentWord)
	}
	return words
}

func calculateMean(values []float64) float64 {
	if len(values) == 0 {
		return 0
	}
	var sum float64
	for _, v := range values {
		sum += v
	}
	return sum / float64(len(values))
}

func calculateStdDev(values []float64) float64 {
	if len(values) == 0 {
		return 0
	}
	mean := calculateMean(values)
	var sqSum float64
	for _, v := range values {
		sqSum += (v - mean) * (v - mean)

	}
	return math.Sqrt(sqSum / float64(len(values)))
}

func sortLineByX(line []indexedText) {
	sort.SliceStable(line, func(i, j int) bool {
		// Use noise floor for X comparison
		if math.Abs(line[i].X-line[j].X) > charXNoiseFloor {
			return line[i].X < line[j].X
		}
		// If X is stacked, maintain stream order
		return line[i].index < line[j].index
	})
}

func calculateMedian(values []float64) float64 {
	n := len(values)
	if n == 0 {
		return 0
	}
	sorted := make([]float64, n)
	copy(sorted, values)
	sort.Float64s(sorted)
	if n%2 == 1 {
		return sorted[n/2]
	}
	return (sorted[n/2-1] + sorted[n/2]) / 2
}

// sanitizeText cleans up common PDF extraction artifacts like ligatures and non-printable characters.
func sanitizeText(s string) string {
	// 1. Apply NFKC normalization to resolve common ligatures (fi, fl) and PUA characters (G4)
	s = norm.NFKC.String(s)

	// 2. Explicitly handle problematic PDF-specific artifacts that NFKC might miss
	r := strings.NewReplacer(
		"\ufffd", "", // Generic replacement char
		"\u0000", "", // Null
		"\uFEFF", "", // BOM
	)
	s = r.Replace(s)

	// 3. Remove non-printable characters but keep standard whitespace
	return strings.Map(func(r rune) rune {
		if r == '\n' || r == '\t' || r == '\r' || r >= 32 {
			return r
		}
		return -1
	}, s)
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
// Returns the detected column positions if minColumns are found, nil otherwise.
// Requires a minimum gap of tableMinColumnGap points between columns to distinguish real table
// columns from tightly-packed body text.
func detectColumnPositions(rows []tableRow, minColumns int) []float64 {
	// Count how often each X position appears across rows (rounded to nearest tableBucketSize)
	xFreq := make(map[int]int)
	for _, row := range rows {
		seen := make(map[int]bool) // avoid double-counting within one row
		for _, cell := range row.cells {
			bucket := int(math.Round(cell.x/tableBucketSize)) * int(tableBucketSize)
			if !seen[bucket] {
				xFreq[bucket]++
				seen[bucket] = true
			}
		}
	}

	// Column positions are X values that appear in at least tableColumnFreqThresh of the rows
	threshold := int(float64(len(rows)) * tableColumnFreqThresh)
	if threshold < 2 {
		threshold = 2
	}

	var candidates []float64
	for x, count := range xFreq {
		if count >= threshold {
			candidates = append(candidates, float64(x))
		}
	}

	sort.Float64s(candidates)

	// Merge columns that are too close together
	var columns []float64
	for _, c := range candidates {
		if len(columns) == 0 || (c-columns[len(columns)-1]) >= tableMinColumnGap {
			columns = append(columns, c)
		}
	}

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

// extractPageBlocks uses positional text data to detect table structures and text blocks.
// Returns a slice of PageBlocks representing the structured content of the page.
func extractPageBlocks(page ledongpdf.Page) []domain.PageBlock {
	content := page.Content()
	if len(content.Text) == 0 {
		return nil
	}

	words := coalesceChars(content.Text)
	if len(words) == 0 {
		return nil
	}

	// Group words into rows by Y coordinate
	rows := groupIntoRows(words, tableYRowTolerance)

	// Detect column positions across rows to find table regions and build blocks
	return buildPageBlocks(rows)
}

// buildPageBlocks converts rows to PageBlocks, detecting tables.
func buildPageBlocks(rows []tableRow) []domain.PageBlock {
	var blocks []domain.PageBlock

	i := 0
	for i < len(rows) {
		// Table-Aware Heuristic: only apply strict table sorting if 3+ distinct
		// X-clusters persist across 3+ contiguous Y-levels (G2 consistency).
		tableEnd := findTableEnd(rows, i, tableMinCols)

		if tableEnd >= i+tableMinCols {
			// We found a table region from rows[i] to rows[tableEnd-1]
			tableRows := rows[i:tableEnd]
			columns := detectColumnPositions(tableRows, tableMinCols)

			if columns != nil {
				// Build table block
				var tableData domain.TableData
				for _, row := range tableRows {
					cells := assignCellsToColumns(row.cells, columns)
					tableData.Rows = append(tableData.Rows, cells)
				}
				blocks = append(blocks, domain.PageBlock{
					Type:  domain.BlockTypeTable,
					Table: tableData,
				})
				i = tableEnd
				continue
			}
		}

		// Not a table — render as plain text block
		var lineWords []string
		for _, cell := range rows[i].cells {
			lineWords = append(lineWords, cell.text)
		}
		line := strings.Join(lineWords, " ")
		if strings.TrimSpace(line) != "" {
			blocks = append(blocks, domain.PageBlock{
				Type: domain.BlockTypeText,
				Text: line,
			})
		}
		i++
	}

	return blocks
}

// ValidateTableStructure enforces row-column consistency.
func ValidateTableStructure(table domain.TableData) error {
	if len(table.Rows) == 0 {
		return nil
	}
	expectedCols := len(table.Rows[0])
	for i, row := range table.Rows {
		if len(row) != expectedCols {
			return fmt.Errorf("row %d has %d columns, expected %d", i, len(row), expectedCols)
		}
	}
	return nil
}
func extractColumnClusters(cells []tableCell, minGap float64) []float64 {
	var clusters []float64
	for _, cell := range cells {
		found := false
		for _, c := range clusters {
			if math.Abs(c-cell.x) < minGap {
				found = true
				break
			}
		}
		if !found {
			clusters = append(clusters, cell.x)
		}
	}
	sort.Float64s(clusters)
	return clusters
}

// findTableEnd looks for the end of a contiguous table region starting at startIdx.
// A table row is defined as having cells in minColumns+ distinct columns.
func findTableEnd(rows []tableRow, startIdx, minColumns int) int {
	end := startIdx

	for end < len(rows) {
		clusters := extractColumnClusters(rows[end].cells, tableMinColumnGap)
		if len(clusters) >= minColumns {
			end++
		} else {
			break
		}
	}

	return end
}
