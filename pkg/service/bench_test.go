package service

import (
	"testing"

	"github.com/nawodyaishan/pdf2md-tui/pkg/domain"
)

// BenchmarkRenderMarkdown_MixedContent measures Markdown rendering with diverse block types
// Simulates realistic conversion output with text and table blocks
func BenchmarkRenderMarkdown_MixedContent(b *testing.B) {
	blocks := []domain.PageBlock{
		{Type: domain.BlockTypeText, Text: "Introduction paragraph with substantive content for LLM ingestion."},
		{Type: domain.BlockTypeTable, Table: domain.TableData{
			Rows: [][]string{
				{"Header1", "Header2", "Header3"},
				{"A", "B", "C"},
				{"1", "2", "3"},
				{"X", "Y", "Z"},
			},
		}},
		{Type: domain.BlockTypeText, Text: "Conclusion paragraph following the table structure."},
		{Type: domain.BlockTypeText, Text: "Additional context and summary information."},
	}

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_ = renderMarkdown(blocks, false)
	}
}

// BenchmarkApplyLLMOptimizations measures text optimization performance
func BenchmarkApplyLLMOptimizations(b *testing.B) {
	text := `Some paragraph with multiple    spaces and
isolated    line

42

Another section with content.
Page 99

Final text here.`

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_ = applyLLMOptimizations(text)
	}
}
