package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sync"
	"time"

	"github.com/spf13/cobra"
	"github.com/nawodyaishan/pdf2md-tui/internal/converter"
	"github.com/nawodyaishan/pdf2md-tui/internal/discovery"
	"github.com/nawodyaishan/pdf2md-tui/internal/tui"
)

var convertCmd = &cobra.Command{
	Use:   "convert [directory]",
	Short: "Convert PDFs in the given directory to Markdown",
	Args:  cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		targetDir := "."
		if len(args) > 0 {
			targetDir = args[0]
		}

		info, err := os.Stat(targetDir)
		if err != nil || !info.IsDir() {
			return fmt.Errorf("invalid directory: %s", targetDir)
		}

		ui := tui.New()
		ui.PrintBanner()
		ui.StartDiscovery()

		pdfFiles, err := discovery.FindPDFs(targetDir, recursive)
		ui.StopDiscovery(len(pdfFiles))

		if err != nil {
			return fmt.Errorf("error during discovery: %w", err)
		}

		if len(pdfFiles) == 0 {
			return nil
		}

		outDirPath := filepath.Join(targetDir, outputDir)
		if err := os.MkdirAll(outDirPath, 0755); err != nil {
			return fmt.Errorf("failed to create output directory: %w", err)
		}

		numWorkers := workers
		if numWorkers <= 0 {
			numWorkers = runtime.NumCPU()
		}

		conv := converter.New(dateFormat, stripNoise)
		ui.StartConversion(len(pdfFiles))

		jobs := make(chan string, len(pdfFiles))
		results := make(chan converter.Result, len(pdfFiles))

		var wg sync.WaitGroup
		for w := 0; w < numWorkers; w++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				for pdf := range jobs {
					res := conv.Convert(pdf, outDirPath)
					results <- res
					ui.Increment()
				}
			}()
		}

		for _, f := range pdfFiles {
			jobs <- f
		}
		close(jobs)

		go func() {
			wg.Wait()
			close(results)
		}()

		var totalIn, totalOut int64
		var totalDur time.Duration // Using simple sum for summary
		var errCount int

		for res := range results {
			if res.Err != nil {
				errCount++
				if verbose {
					fmt.Fprintf(os.Stderr, "\nError processing %s: %v", res.InputPath, res.Err)
				}
			} else {
				totalIn += res.InputBytes
				totalOut += res.OutputBytes
				totalDur += res.Duration
			}
		}

		ui.StopConversion()
		ui.PrintSummary(totalIn, totalOut, totalDur, errCount)

		return nil
	},
}
