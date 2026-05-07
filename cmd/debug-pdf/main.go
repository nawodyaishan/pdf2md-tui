package main

import (
	"fmt"
	"log"
	"os"

	"github.com/ledongthuc/pdf"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: debug-pdf <file.pdf>")
		os.Exit(1)
	}

	pdfPath := os.Args[1]
	f, r, err := pdf.Open(pdfPath)
	if err != nil {
		log.Fatalf("failed to open pdf: %v", err)
	}
	defer func() { _ = f.Close() }()

	numPages := r.NumPage()
	fmt.Printf("PDF has %d pages\n", numPages)

	// Analyze first page
	page := r.Page(1)
	content := page.Content()

	fmt.Printf("Page 1 has %d text objects\n", len(content.Text))
	fmt.Println("Index | Text | X | Y | W | FontSize | Font")
	fmt.Println("--------------------------------------------")
	for i, t := range content.Text {
		fmt.Printf("%5d | %4q | %7.2f | %7.2f | %7.2f | %7.2f | %s\n", i, t.S, t.X, t.Y, t.W, t.FontSize, t.Font)
		if i > 500 { // Limit output
			fmt.Println("... truncated ...")
			break
		}
	}
}
