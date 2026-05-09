package pdf

import (
	"encoding/binary"
	"math"
	"testing"

	ledongpdf "github.com/ledongthuc/pdf"
)

// FuzzCoalesceChars tests the character coalescing algorithm with arbitrary byte input.
// The fuzzer mutates the seed corpus to find edge cases and panics.
// Run with: go test -fuzz=FuzzCoalesceChars -fuzztime=30s ./pkg/repository/pdf/
func FuzzCoalesceChars(f *testing.F) {
	// Seed corpus: basic inputs the fuzzer will mutate
	f.Add([]byte{
		0x41, 0x42, 0x43, // "ABC"
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x24, 0x40, // X=10.0
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x59, 0x40, // Y=100.0
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x1C, 0x40, // FontSize=7.0
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x1C, 0x40, // W=7.0
	})
	f.Add([]byte{}) // Empty input

	f.Fuzz(func(t *testing.T, data []byte) {
		texts := parseTextsFromBytes(data)
		result := coalesceChars(texts)

		// Invariant: output word count <= input char count
		if len(result) > len(texts) {
			t.Fatalf("word count %d exceeds char count %d", len(result), len(texts))
		}

		// Invariant: output is deterministic (calling twice yields same result)
		result2 := coalesceChars(texts)
		if len(result) != len(result2) {
			t.Fatalf("coalesceChars not deterministic: %d words first, %d words second", len(result), len(result2))
		}
	})
}

// parseTextsFromBytes deserializes fuzzer input into PDF text objects.
// Format: stride of 32 bytes per text (char + 3x float64)
func parseTextsFromBytes(data []byte) []ledongpdf.Text {
	const stride = 32 // 1 byte + 3x uint64 (float64)
	var texts []ledongpdf.Text

	for i := 0; i+stride <= len(data); i += stride {
		chunk := data[i : i+stride]
		s := string(rune(chunk[0]))
		if s == "\x00" {
			s = " " // Avoid null chars in strings
		}

		// Extract 3 float64 values at fixed offsets
		x := math.Float64frombits(binary.LittleEndian.Uint64(chunk[8:16]))
		y := math.Float64frombits(binary.LittleEndian.Uint64(chunk[16:24]))
		fs := math.Float64frombits(binary.LittleEndian.Uint64(chunk[24:32]))

		// Clamp to reasonable PDF ranges
		if math.IsNaN(x) || math.IsInf(x, 0) {
			x = 100
		}
		if math.IsNaN(y) || math.IsInf(y, 0) {
			y = 100
		}
		if math.IsNaN(fs) || math.IsInf(fs, 0) || fs <= 0 {
			fs = 12
		}

		texts = append(texts, ledongpdf.Text{
			S:        s,
			X:        x,
			Y:        y,
			FontSize: fs,
			W:        7, // Default char width
		})
	}

	return texts
}
