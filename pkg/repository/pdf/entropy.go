package pdf

import (
	"math"
)

// CalculateShannonEntropy measures the information density of text blocks.
// High entropy in text blocks often indicates corrupted/garbage encoding.
func CalculateShannonEntropy(data string) float64 {
	if data == "" {
		return 0
	}
	counts := make(map[rune]float64)
	for _, r := range data {
		counts[r]++
	}
	var entropy float64
	l := float64(len(data))
	for _, c := range counts {
		p := c / l
		entropy -= p * math.Log2(p)
	}
	return entropy
}
