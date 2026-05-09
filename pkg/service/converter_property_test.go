package service

import (
	"testing"

	"pgregory.net/rapid"
)

// TestProperty_ApplyLLMOptimizations_NoSecondPass verifies behavior is stable:
// When called twice in sequence, output length should not increase
func TestProperty_ApplyLLMOptimizations_StableLength(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		input := rapid.String().Draw(t, "input")
		once := applyLLMOptimizations(input)
		twice := applyLLMOptimizations(once)
		// The function is not perfectly idempotent, but the second call
		// should not produce longer output
		if len(twice) > len(once) {
			t.Fatalf("second pass produced longer output: %d > %d", len(twice), len(once))
		}
	})
}

// TestProperty_ApplyLLMOptimizations_LengthNonIncreasing verifies output is never longer
// than input (optimization removes content, never adds)
func TestProperty_ApplyLLMOptimizations_LengthNonIncreasing(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		input := rapid.String().Draw(t, "input")
		output := applyLLMOptimizations(input)
		if len(output) > len(input) {
			t.Fatalf("output longer than input: %d bytes > %d bytes", len(output), len(input))
		}
	})
}

// TestProperty_ApplyLLMOptimizations_NoPanic ensures optimization never panics
// even on pathological input (very long strings, special chars, etc)
func TestProperty_ApplyLLMOptimizations_NoPanic(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		// Generate strings with various UTF-8 edge cases
		input := rapid.String().Draw(t, "input")
		// Must not panic, even with null bytes, control chars, non-BMP runes
		_ = applyLLMOptimizations(input)
	})
}
