package pdf

import (
	"testing"
)

func TestNewTokenizer(t *testing.T) {
	tokenizer, err := NewTokenizer()
	if err != nil {
		t.Fatalf("NewTokenizer failed: %v", err)
	}
	if tokenizer == nil {
		t.Fatal("NewTokenizer returned nil tokenizer")
	}
}

func TestTokenizerCount(t *testing.T) {
	tokenizer, err := NewTokenizer()
	if err != nil {
		t.Fatalf("Failed to create tokenizer: %v", err)
	}

	tests := []struct {
		name     string
		text     string
		minCount int
	}{
		{"Empty string", "", 0},
		{"Single word", "hello", 1},
		{"Multiple words", "hello world test", 3},
		{"Long text", "The quick brown fox jumps over the lazy dog. This is a longer piece of text.", 15},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			count := tokenizer.Count(tt.text)
			if count < tt.minCount {
				t.Errorf("expected at least %d tokens, got %d", tt.minCount, count)
			}
			if count < 0 {
				t.Errorf("token count should not be negative: %d", count)
			}
		})
	}
}

func TestTokenizerCountIdempotent(t *testing.T) {
	tokenizer, err := NewTokenizer()
	if err != nil {
		t.Fatalf("Failed to create tokenizer: %v", err)
	}

	text := "This is a test string for tokenization"
	count1 := tokenizer.Count(text)
	count2 := tokenizer.Count(text)

	if count1 != count2 {
		t.Errorf("expected consistent token count, got %d and %d", count1, count2)
	}
}
