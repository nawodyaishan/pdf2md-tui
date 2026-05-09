package pdf

import (
	"github.com/ledongthuc/pdf"
)

// MockPage mimics the behavior of a pdf.Page for unit testing.
type MockPage struct {
	content pdf.Content
}

// NewMockPage creates a mock page with specific text content.
func NewMockPage(texts []pdf.Text) *MockPage {
	return &MockPage{
		content: pdf.Content{
			Text: texts,
		},
	}
}

// Content returns the mocked content, satisfying the Page interface.
func (m *MockPage) Content() pdf.Content {
	return m.content
}

// Helper to construct pdf.Text objects easily
func Text(s string, x, y, w, fontSize float64, index int) pdf.Text {
	return pdf.Text{
		S:        s,
		X:        x,
		Y:        y,
		W:        w,
		FontSize: fontSize,
	}
}
