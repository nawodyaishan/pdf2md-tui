package main

import (
	"fmt"
	"math"
	"os"
	"sort"
	"strings"

	"github.com/ledongthuc/pdf"
)

func main() {
	f, r, err := pdf.Open(os.Args[1])
	if err != nil {
		panic(err)
	}
	defer f.Close()

	targetPage := 8
	if len(os.Args) > 2 {
		fmt.Sscanf(os.Args[2], "%d", &targetPage)
	}

	page := r.Page(targetPage)

	// Use Content() which gives us Text with X,Y coordinates
	content := page.Content()
	fmt.Printf("Page %d: %d text elements, %d rects\n\n", targetPage, len(content.Text), len(content.Rect))

	// Show first 40 text elements with coordinates
	for i, t := range content.Text {
		if i >= 60 {
			break
		}
		if strings.TrimSpace(t.S) == "" {
			continue
		}
		fmt.Printf("  [x=%.1f y=%.1f w=%.1f font=%s size=%.1f] %q\n", t.X, t.Y, t.W, t.Font, t.FontSize, t.S)
	}

	// Now try to group by Y (row detection with tolerance)
	fmt.Println("\n=== Row-grouped output (tolerance=2pt) ===")
	type rowGroup struct {
		y     float64
		texts []pdf.Text
	}

	var groups []rowGroup
	tolerance := 2.0

	for _, t := range content.Text {
		if strings.TrimSpace(t.S) == "" {
			continue
		}
		found := false
		for i := range groups {
			if math.Abs(groups[i].y-t.Y) < tolerance {
				groups[i].texts = append(groups[i].texts, t)
				found = true
				break
			}
		}
		if !found {
			groups = append(groups, rowGroup{y: t.Y, texts: []pdf.Text{t}})
		}
	}

	// Sort rows by Y (top to bottom)
	sort.Slice(groups, func(i, j int) bool {
		return groups[i].y > groups[j].y
	})

	for _, g := range groups {
		// Sort texts in row left to right
		sort.Slice(g.texts, func(i, j int) bool {
			return g.texts[i].X < g.texts[j].X
		})

		var cells []string
		for _, t := range g.texts {
			cells = append(cells, t.S)
		}

		if len(cells) > 2 {
			fmt.Printf("[Y=%.0f cols=%d] %s\n", g.y, len(cells), strings.Join(cells, " | "))
		}
	}
}
