package converter

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCleanExtractedText(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{
			name: "word-per-line rejoining",
			in:   "Hello\n \nWorld\n \nTest\n",
			want: "Hello World Test",
		},
		{
			name: "paragraph breaks on empty lines",
			in:   "First\n \nparagraph\n\nSecond\n \nparagraph\n",
			want: "First paragraph\n\nSecond paragraph",
		},
		{
			name: "empty input",
			in:   "",
			want: "",
		},
		{
			name: "whitespace only",
			in:   "  \n \n  \n",
			want: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := cleanExtractedText(tt.in); got != tt.want {
				t.Errorf("cleanExtractedText() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestApplyLLMOptimizations(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{
			name: "collapse spaces",
			in:   "Hello     World\n\n\nTesting",
			want: "Hello World Testing",
		},
		{
			name: "remove isolated numbers (page numbers)",
			in:   "Hello World\n12\nNext Page",
			want: "Hello World Next Page",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := applyLLMOptimizations(tt.in); got != tt.want {
				t.Errorf("applyLLMOptimizations() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestConverter_Convert_NoPDF(t *testing.T) {
	conv := New("2006-01-02", true)
	tempDir := t.TempDir()

	res := conv.Convert(filepath.Join(tempDir, "doesnotexist.pdf"), tempDir)
	if res.Err == nil {
		t.Error("expected error for non-existent file, got nil")
	}
}

func TestConverter_Convert_NotAPDF(t *testing.T) {
	conv := New("2006-01-02", false)
	tempDir := t.TempDir()

	badPDF := filepath.Join(tempDir, "bad.pdf")
	os.WriteFile(badPDF, []byte("this is not a real pdf"), 0644)

	res := conv.Convert(badPDF, tempDir)
	if res.Err == nil {
		t.Error("expected error for invalid pdf format, got nil")
	}
	if !strings.Contains(res.Err.Error(), "open pdf") {
		t.Errorf("expected open pdf error, got %v", res.Err)
	}
}
