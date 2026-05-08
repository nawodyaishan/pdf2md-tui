package pdf

import (
	"github.com/nawodyaishan/pdf2md-tui/internal/domain"
	"github.com/pkoukk/tiktoken-go"
)

type TiktokenTokenizer struct {
	encoding *tiktoken.Tiktoken
}

// NewTokenizer creates a new Tokenizer using the o200k_base encoding (GPT-4o standard).
// This is the most modern and dense tokenization standard.
func NewTokenizer() (domain.Tokenizer, error) {
	enc, err := tiktoken.GetEncoding("o200k_base")
	if err != nil {
		// Fallback to cl100k_base if o200k_base is not found
		enc, err = tiktoken.GetEncoding("cl100k_base")
		if err != nil {
			return nil, err
		}
	}
	return &TiktokenTokenizer{encoding: enc}, nil
}

func (t *TiktokenTokenizer) Count(text string) int {
	if text == "" {
		return 0
	}
	tokens := t.encoding.Encode(text, nil, nil)
	return len(tokens)
}
