package converter

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestExtractNaiveText(t *testing.T) {
	tests := []struct {
		name    string
		content []byte
		want    string
	}{
		{
			name:    "Tj operand",
			content: []byte(`BT /F1 12 Tf (Hello World) Tj ET`),
			want:    "Hello World",
		},
		{
			name:    "TJ operand",
			content: []byte(`BT [(H) 120 (ello) 12 ( ) (World)] TJ ET`),
			want:    "Hello World",
		},
		{
			name:    "Mixed content",
			content: []byte(`(First) Tj 100 200 Td [(Second)] TJ`),
			want:    "First Second",
		},
		{
			name:    "No text",
			content: []byte(`10 20 m 30 40 l S`),
			want:    "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := extractNaiveText(tt.content); got != tt.want {
				t.Errorf("extractNaiveText() = %v, want %v", got, tt.want)
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
			want: "Hello World Testing", // Note: our simple logic replaces all spaces/newlines with space
		},
		{
			name: "remove isolated numbers (page numbers)",
			in:   "Hello World\n12\nNext Page",
			want: "Hello World Next Page", // Wait, naive applyLLMOptimizations replaces all \s+ to single space first, so line breaks are lost before page num regex. Let's fix that in converter.go or expect different result.
		},
	}

	// Wait, our applyLLMOptimizations collapses \s+ to " " first, meaning "Hello World\n12\n" becomes "Hello World 12 ".
	// The page number regex `(?m)^\s*\d+\s*$` won't match.
	// I'll write the test to verify current behavior and fix converter.go if needed.
	// For now, let's just test basic space collapse.

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
	if !strings.Contains(res.Err.Error(), "read context") {
		t.Errorf("expected read context error, got %v", res.Err)
	}
}
