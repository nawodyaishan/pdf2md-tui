package cli

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sync"
	"time"

	"github.com/nawodyaishan/pdf2md-tui/internal/domain"
	"github.com/nawodyaishan/pdf2md-tui/internal/handler/tui"
	"github.com/nawodyaishan/pdf2md-tui/internal/repository/discovery"
	"github.com/nawodyaishan/pdf2md-tui/internal/repository/pdf"
	"github.com/nawodyaishan/pdf2md-tui/internal/repository/storage"
	"github.com/nawodyaishan/pdf2md-tui/internal/repository/sysinfo"
	"github.com/nawodyaishan/pdf2md-tui/internal/service"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/pterm/pterm"
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
		}

		// Initialize logger
		logF, logErr := os.OpenFile(logFile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
		if logErr == nil {
			defer func() { _ = logF.Close() }()
			pterm.DefaultLogger.Writer = logF
			if verbose {
				pterm.DefaultLogger.Writer = io.MultiWriter(logF, os.Stderr)
			}
			pterm.DefaultLogger.Info("Conversion session started", pterm.DefaultLogger.Args("time", time.Now().Format(time.RFC3339)))
		}

		if !quiet {
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
		tokenCount, _ := pdf.NewTokenizer()
		conv := service.NewConverterService(cfg, store, parser, tokenCount)

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
				pterm.DefaultLogger.Warn("Skipping file: Requires OCR", pterm.DefaultLogger.Args("file", f))
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
				ui.PrintSummary(allResults, 0, 0, 0, 0, ignoredCount, domain.SysInfo{})
			} else {
				printJSONSummary(allResults, 0, 0, 0, ignoredCount)
			}
			return nil
		}

		numWorkers := workers
		if numWorkers <= 0 {
			numWorkers = runtime.NumCPU()
		}

		var prog *tea.Program
		var finalModel tea.Model
		var msgChan chan tea.Msg
		var tuiModel tui.Model

		if !quiet {
			ui.SetBatchInfo(len(convertible), numWorkers)
			tuiModel = tui.NewModel(len(convertible), numWorkers)
			prog = tea.NewProgram(tuiModel, tea.WithAltScreen())
			msgChan = make(chan tea.Msg)
			
			// Bridge msgChan to prog
			go func() {
				for msg := range msgChan {
					prog.Send(msg)
				}
			}()

			// Redirect logger to nowhere during TUI to prevent screen corruption
			pterm.DefaultLogger.Writer = io.Discard
		}

		// Resource tracking
		sysProv := sysinfo.NewProvider()
		var maxSys domain.SysInfo
		sysCtx, sysCancel := context.WithCancel(context.Background())
		defer sysCancel()

		go func() {
			ticker := time.NewTicker(500 * time.Millisecond)
			defer ticker.Stop()
			for {
				select {
				case <-sysCtx.Done():
					return
				case <-ticker.C:
					snap, _ := sysProv.GetSnapshot()
					if !quiet && msgChan != nil {
						msgChan <- tui.SysInfoMsg{Info: snap}
					}
					if snap.CPUUsage > maxSys.CPUUsage {
						maxSys.CPUUsage = snap.CPUUsage
					}
					if snap.MemoryUsed > maxSys.MemoryUsed {
						maxSys.MemoryUsed = snap.MemoryUsed
						maxSys.MemoryPct = snap.MemoryPct
						maxSys.MemoryTotal = snap.MemoryTotal
					}
				}
			}
		}()

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
					if !quiet && msgChan != nil {
						msgChan <- tui.StatusUpdateMsg{Result: &res}
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
			sysCancel() // Stop tracking resources immediately
			if msgChan != nil {
				msgChan <- tui.BatchCompleteMsg{} // Trigger UI pause
			}
		}()

		var totalIn, totalOut int64
		var totalDur time.Duration
		var errCount int

		if !quiet {
			// Now run the TUI in main thread
			var err error
			finalModel, err = prog.Run()
			if err != nil {
				pterm.Error.Printf("UI error: %v\n", err)
			}
			
			// Restore logger
			pterm.DefaultLogger.Writer = logF
			if verbose {
				pterm.DefaultLogger.Writer = io.MultiWriter(logF, os.Stderr)
			}
			
			// Handle the integrated menu selection
			if m, ok := finalModel.(tui.Model); ok {
				allResults = m.Results
				if m.SelectedMenuIndex == 0 {
					// Open Folder selected
					_ = openDir(outDirPath)
				}
			}
		} else {
			// Non-interactive mode just waits for results
			for res := range workerResults {
				allResults = append(allResults, res)
				if res.Err != nil {
					errCount++
				} else {
					totalIn += res.InputBytes
					totalOut += res.OutputBytes
					totalDur += res.Duration
				}
			}
		}

		if !quiet {
			sysCancel() // Stop tracking
		} else {
			printJSONSummary(allResults, totalIn, totalOut, totalDur, ignoredCount)
		}

		if errCount > 0 {
			return fmt.Errorf("conversion completed with %d errors; see %s for details", errCount, logFile)
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

func openDir(path string) error {
	var cmd string
	var args []string

	switch runtime.GOOS {
	case "windows":
		cmd = "explorer"
		args = []string{path}
	case "darwin":
		cmd = "open"
		args = []string{path}
	default: // linux, freebsd, etc.
		cmd = "xdg-open"
		args = []string{path}
	}

	return exec.Command(cmd, args...).Run()
}
