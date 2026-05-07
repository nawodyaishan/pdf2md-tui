package cli

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sync"
	"time"

	"github.com/nawodyaishan/pdf2md-tui/internal/domain"
	"github.com/nawodyaishan/pdf2md-tui/internal/handler/tui"
	"github.com/nawodyaishan/pdf2md-tui/internal/repository/discovery"
	"github.com/nawodyaishan/pdf2md-tui/internal/repository/pdf"
	"github.com/nawodyaishan/pdf2md-tui/internal/repository/storage"
	"github.com/nawodyaishan/pdf2md-tui/internal/service"
	"github.com/spf13/cobra"
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
		if !quiet {
			ui.PrintBanner()
			ui.StartDiscovery()
		}

		pdfFiles, err := discovery.FindPDFs(targetDir, recursive)
		if !quiet {
			ui.StopDiscovery(len(pdfFiles))
		}

		if err != nil {
			return fmt.Errorf("error during discovery: %w", err)
		}

		if len(pdfFiles) == 0 {
			if !quiet {
				ui.PrintNoPDFsFound(targetDir)
			}
			return nil
		}

		outDirPath := filepath.Join(targetDir, outputDir)
		if err := os.MkdirAll(outDirPath, 0755); err != nil {
			return fmt.Errorf("failed to create output directory: %w", err)
		}

		cfg := domain.NewConfig()
		cfg.DateFormat = dateFormat
		cfg.StripNoise = stripNoise
		cfg.ExtractImages = extractImages

		store := storage.NewStorage()
		parser := pdf.NewParser()
		conv := service.NewConverterService(cfg, store, parser)

		// Pre-flight: detect output files that already exist.
		if !forceOverwrite && !quiet {
			var existing []string
			for _, f := range pdfFiles {
				if store.FileExists(conv.OutputPath(f, outDirPath)) {
					existing = append(existing, conv.OutputPath(f, outDirPath))
				}
			}
			if len(existing) > 0 {
				if tui.IsInteractive() {
					ok, promptErr := ui.ConfirmOverwrite(existing)
					if promptErr != nil {
						return promptErr
					}
					if !ok {
						fmt.Println("Conversion cancelled.")
						return nil
					}
				} else {
					ui.WarnOverwrite(existing)
				}
			}
		}

		// Results collection for JSON summary
		var allResults []domain.Result

		// Pre-flight: Detect scanned/image-only PDFs (OCR heuristic)
		var convertible []string
		var ignoredCount int
		for _, f := range pdfFiles {
			doc, err := parser.OpenDocument(f)
			if err == nil {
				_, err = doc.AnalyzePreFlight(3)
				_ = doc.Close()
			}

			if errors.Is(err, domain.ErrRequiresOCR) {
				ignoredCount++
				if verbose && !quiet {
					fmt.Fprintf(os.Stderr, "\nSkipping %s: Requires OCR", f)
				}
				allResults = append(allResults, domain.Result{
					InputPath: f,
					Status:    domain.StatusIgnored,
				})
			} else {
				convertible = append(convertible, f)
			}
		}

		if len(convertible) == 0 {
			if !quiet {
				ui.PrintSummary(0, 0, 0, 0, ignoredCount)
			} else {
				printJSONSummary(allResults, 0, 0, 0, ignoredCount)
			}
			return nil
		}

		numWorkers := workers
		if numWorkers <= 0 {
			numWorkers = runtime.NumCPU()
		}

		if !quiet {
			ui.StartConversion(len(convertible))
		}

		jobs := make(chan string, len(convertible))
		workerResults := make(chan domain.Result, len(convertible))

		var wg sync.WaitGroup
		for w := 0; w < numWorkers; w++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				for pdf := range jobs {
					res := conv.Convert(pdf, outDirPath)
					workerResults <- res
					if !quiet {
						ui.Increment()
					}
				}
			}()
		}

		for _, f := range convertible {
			jobs <- f
		}
		close(jobs)

		go func() {
			wg.Wait()
			close(workerResults)
		}()

		var totalIn, totalOut int64
		var totalDur time.Duration
		var errCount int

		for res := range workerResults {
			allResults = append(allResults, res)
			if res.Err != nil {
				errCount++
				if verbose && !quiet {
					fmt.Fprintf(os.Stderr, "\nError processing %s: %v", res.InputPath, res.Err)
				}
			} else {
				totalIn += res.InputBytes
				totalOut += res.OutputBytes
				totalDur += res.Duration
			}
		}

		if !quiet {
			ui.StopConversion()
			ui.PrintSummary(totalIn, totalOut, totalDur, errCount, ignoredCount)
		} else {
			printJSONSummary(allResults, totalIn, totalOut, totalDur, ignoredCount)
		}

		if errCount > 0 {
			return errors.New("conversion completed with errors")
		}
		return nil
	},
}

func printJSONSummary(results []domain.Result, totalIn, totalOut int64, duration time.Duration, ignored int) {
	summary := domain.Summary{
		Duration:  duration.String(),
		Converted: 0,
		Skipped:   ignored,
		Errors:    0,
	}

	for _, res := range results {
		status := "ok"
		if res.Status == domain.StatusIgnored {
			status = "ignored"
		} else if res.Err != nil {
			status = "error"
			summary.Errors++
		} else {
			summary.Converted++
		}

		errMsg := ""
		if res.Err != nil {
			errMsg = res.Err.Error()
		}

		summary.Files = append(summary.Files, domain.FileSummary{
			Input:       res.InputPath,
			Output:      res.OutputPath,
			Status:      status,
			Error:       errMsg,
			InputBytes:  res.InputBytes,
			OutputBytes: res.OutputBytes,
		})
	}

	data, _ := json.MarshalIndent(summary, "", "  ")
	fmt.Println(string(data))
}
