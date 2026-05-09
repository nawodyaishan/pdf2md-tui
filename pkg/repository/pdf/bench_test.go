package pdf

import (
	"fmt"
	"testing"

	ledongpdf "github.com/ledongthuc/pdf"
)

// BenchmarkCoalesceChars measures character coalescing performance across different input sizes
// Run with: go test -bench=BenchmarkCoalesceChars -benchmem ./pkg/repository/pdf/
func BenchmarkCoalesceChars(b *testing.B) {
	for _, n := range []int{100, 500, 1000} {
		texts := make([]ledongpdf.Text, n)
		for i := range texts {
			texts[i] = ledongpdf.Text{
				S:        fmt.Sprintf("word%d", i),
				X:        float64(i * 8),
				Y:        100,
				FontSize: 12,
				W:        7,
			}
		}

		b.Run(fmt.Sprintf("N=%d", n), func(b *testing.B) {
			b.ReportAllocs()
			for i := 0; i < b.N; i++ {
				_ = coalesceChars(texts)
			}
		})
	}
}

// BenchmarkExtractPageBlocks_SyntheticTable measures extraction performance on table-like content
// Simulates a 3-column × 20-row table structure
func BenchmarkExtractPageBlocks_SyntheticTable(b *testing.B) {
	texts := make([]ledongpdf.Text, 60) // 3 cols × 20 rows
	for i := range texts {
		col := i % 3
		row := i / 3
		texts[i] = ledongpdf.Text{
			S:        fmt.Sprintf("cell%d", i),
			X:        float64(col*150 + 50),
			Y:        float64(100 - row*12),
			FontSize: 10,
			W:        28,
		}
	}
	page := NewMockPage(texts)

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_ = extractPageBlocks(page)
	}
}

// BenchmarkGroupIntoRows measures row grouping performance on word lists
func BenchmarkGroupIntoRows(b *testing.B) {
	for _, n := range []int{50, 200, 500} {
		words := make([]word, n)
		for i := range words {
			words[i] = word{
				text: fmt.Sprintf("w%d", i),
				x:    float64(i * 10),
				y:    float64((i / 10) * 15), // ~10 words per row
			}
		}

		b.Run(fmt.Sprintf("N=%d", n), func(b *testing.B) {
			b.ReportAllocs()
			for i := 0; i < b.N; i++ {
				_ = groupIntoRows(words, tableYRowTolerance)
			}
		})
	}
}
