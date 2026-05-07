package main

import (
	"os"

	"github.com/nawodyaishan/pdf2md-tui/internal/handler/cli"
)

func main() {
	if err := cli.Execute(); err != nil {
		os.Exit(1)
	}
}
