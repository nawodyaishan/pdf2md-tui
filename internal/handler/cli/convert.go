package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sync"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/nawodyaishan/pdf2md-tui/internal/handler/tui"
	"github.com/nawodyaishan/pdf2md-tui/pkg/domain"
	"github.com/nawodyaishan/pdf2md-tui/pkg/repository/discovery"
	"github.com/nawodyaishan/pdf2md-tui/pkg/repository/pdf"
	"github.com/nawodyaishan/pdf2md-tui/pkg/repository/storage"
	"github.com/nawodyaishan/pdf2md-tui/pkg/repository/sysinfo"
	"github.com/nawodyaishan/pdf2md-tui/pkg/service"
	"github.com/pterm/pterm"
	"github.com/spf13/cobra"
)

type conversionTotals struct {
	inputBytes  int64
	outputBytes int64
	duration    time.Duration
	errCount    int
	ignored     int
	converted   int
}

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
		if quiet {
			pterm.SetDefaultOutput(os.Stderr)
			pterm.DefaultLogger.Writer = os.Stderr
		}
		logF, logErr := os.OpenFile(logFile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
		if logErr == nil {
			defer func() { _ = logF.Close() }()
			pterm.DefaultLogger.Writer = logF
			if verbose || quiet {
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
			} else {
				printJSONSummary(nil, conversionTotals{duration: 0})
			}
			return nil
		}

		cfg := domain.NewConfig()
		cfg.DateFormat = dateFormat
		cfg.StripNoise = stripNoise
		cfg.ExtractImages = extractImages
		cfg.EntropyThreshold = entropyThresh

		store := storage.NewStorage()

		parser := pdf.NewParser()
		tokenCount, _ := pdf.NewTokenizer()
		conv := service.NewConverterService(cfg, store, parser, tokenCount)
		outDirPath := filepath.Join(targetDir, outputDir)

		existing := existingOutputPaths(pdfFiles, conv, store, outDirPath)
		needsPrompt, overwriteErr := resolveOverwritePolicy(existing, forceOverwrite, !quiet && tui.IsInteractive())
		if overwriteErr != nil {
			return overwriteErr
		}
		if needsPrompt {
			ok, promptErr := ui.ConfirmOverwrite(existing)
			if promptErr != nil {
				return promptErr
			}
			if !ok {
				fmt.Println("Conversion cancelled.")
				return nil
			}
		}
		if err := os.MkdirAll(outDirPath, 0755); err != nil {
			return fmt.Errorf("failed to create output directory: %w", err)
		}

		var preflightResults []domain.Result

		// Pre-flight: Detect scanned/image-only PDFs (OCR heuristic)
		var convertible []string
		for _, f := range pdfFiles {
			doc, err := parser.OpenDocument(f)
			if err == nil {
				_, err = doc.AnalyzePreFlight(3)
				_ = doc.Close()
			}

			if domain.IsSkippablePreflightError(err) {
				pterm.DefaultLogger.Warn("Skipping file during pre-flight", pterm.DefaultLogger.Args("file", f, "reason", err.Error()))
				var inputBytes int64
				if info, statErr := os.Stat(f); statErr == nil {
					inputBytes = info.Size()
				}
				preflightResults = append(preflightResults, domain.Result{
					InputPath:  f,
					InputBytes: inputBytes,
					Status:     domain.StatusIgnored,
					Err:        err,
				})
			} else {
				convertible = append(convertible, f)
			}
		}

		if len(convertible) == 0 {
			totals := summarizeResults(preflightResults)
			if !quiet {
				ui.PrintSummary(preflightResults, totals.inputBytes, totals.outputBytes, totals.duration, totals.errCount, totals.ignored, domain.SysInfo{})
			} else {
				printJSONSummary(preflightResults, totals)
			}
			return nil
		}

		numWorkers := workers
		if numWorkers <= 0 {
			numWorkers = runtime.NumCPU()
		}

		useDashboard := !quiet && tui.IsInteractive()
		var prog *tea.Program
		var finalModel tea.Model
		var msgChan chan tea.Msg
		var tuiModel tui.Model

		if useDashboard {
			ui.SetBatchInfo(len(pdfFiles), numWorkers)
			clearTerminal(os.Stdout)
			tuiModel = tui.NewModel(len(pdfFiles), numWorkers)
			tuiModel.Results = append(tuiModel.Results, preflightResults...)
			tuiModel.CurrentFile = len(preflightResults)
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
				close(msgChan)
			}
		}()

		var liveResults []domain.Result

		if useDashboard {
			// Now run the TUI in main thread
			var err error
			finalModel, err = prog.Run()
			if err != nil {
				pterm.Error.Printf("UI error: %v\n", err)
			}

			// Restore logger
			if logF != nil {
				pterm.DefaultLogger.Writer = logF
			}
			if verbose && logF != nil {
				pterm.DefaultLogger.Writer = io.MultiWriter(logF, os.Stderr)
			}

			if m, ok := finalModel.(tui.Model); ok {
				liveResults = append(liveResults, m.Results...)
				switch m.CompletionAction {
				case tui.CompletionActionOpenDir:
					_ = openDir(outDirPath)
				case tui.CompletionActionViewLog:
					_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Detailed log: %s\n", logFile)
				}
			}
		} else {
			// Non-interactive mode waits for results and prints a plain-text summary.
			for res := range workerResults {
				liveResults = append(liveResults, res)
			}
		}

		allResults := liveResults
		if !useDashboard {
			allResults = mergeResults(preflightResults, liveResults)
		}
		totals := summarizeResults(allResults)

		if !useDashboard {
			sysCancel()
		}

		if quiet {
			printJSONSummary(allResults, totals)
		} else {
			if !useDashboard {
				if totals.errCount > 0 {
					_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Detailed log: %s\n", logFile)
				}
				printTextSummary(cmd.OutOrStdout(), allResults, totals, maxSys)
			}
		}

		if totals.errCount > 0 {
			return fmt.Errorf("conversion completed with %d errors; see %s for details", totals.errCount, logFile)
		}
		return nil
	},
}

func mergeResults(preflightResults, liveResults []domain.Result) []domain.Result {
	results := make([]domain.Result, 0, len(preflightResults)+len(liveResults))
	results = append(results, preflightResults...)
	results = append(results, liveResults...)
	return results
}

func summarizeResults(results []domain.Result) conversionTotals {
	var totals conversionTotals

	for _, res := range results {
		switch {
		case res.Status == domain.StatusIgnored:
			totals.ignored++
		case res.Err != nil || res.Status == domain.StatusError:
			totals.errCount++
		default:
			totals.converted++
			totals.inputBytes += res.InputBytes
			totals.outputBytes += res.OutputBytes
			totals.duration += res.Duration
		}
	}

	return totals
}

func existingOutputPaths(pdfFiles []string, conv *service.ConverterService, store domain.PDFStorage, outDirPath string) []string {
	var existing []string
	for _, f := range pdfFiles {
		outPath := conv.OutputPath(f, outDirPath)
		if store.FileExists(outPath) {
			existing = append(existing, outPath)
		}
	}
	return existing
}

func resolveOverwritePolicy(existing []string, force, interactive bool) (bool, error) {
	if force || len(existing) == 0 {
		return false, nil
	}
	if interactive {
		return true, nil
	}
	return false, fmt.Errorf("refusing to overwrite %d existing output file(s) without --force", len(existing))
}

func printJSONSummary(results []domain.Result, totals conversionTotals) {
	summary := domain.Summary{
		TotalFiles: len(results),
		Successful: totals.converted,
		Failed:     totals.errCount,
		ElapsedMs:  totals.duration.Milliseconds(),
	}

	for _, res := range results {
		status := "ok"
		if res.Status == domain.StatusIgnored {
			status = "ignored"
		} else if res.Err != nil || res.Status == domain.StatusError {
			status = "error"
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
	fmt.Fprintln(os.Stdout, string(data))
}

func printTextSummary(w io.Writer, results []domain.Result, totals conversionTotals, sys domain.SysInfo) {
	_, _ = fmt.Fprintln(w, "Conversion summary")
	_, _ = fmt.Fprintf(w, "Converted: %d\n", totals.converted)
	_, _ = fmt.Fprintf(w, "Skipped: %d\n", totals.ignored)
	_, _ = fmt.Fprintf(w, "Errors: %d\n", totals.errCount)
	_, _ = fmt.Fprintf(w, "Duration: %s\n", totals.duration.Round(time.Millisecond))
	_, _ = fmt.Fprintf(w, "Input bytes: %d\n", totals.inputBytes)
	_, _ = fmt.Fprintf(w, "Output bytes: %d\n", totals.outputBytes)
	_, _ = fmt.Fprintf(w, "Peak CPU: %.1f%%\n", sys.CPUUsage)
	_, _ = fmt.Fprintf(w, "Peak memory: %d bytes (%.1f%%)\n", sys.MemoryUsed, sys.MemoryPct)

	if totals.converted > 0 {
		_, _ = fmt.Fprintln(w, "Converted files:")
		for _, res := range results {
			if res.Status == domain.StatusOK && res.Err == nil {
				_, _ = fmt.Fprintf(w, "- %s -> %s\n", res.InputPath, res.OutputPath)
			}
		}
	}

	if totals.ignored > 0 {
		_, _ = fmt.Fprintln(w, "Skipped files:")
		for _, res := range results {
			if res.Status == domain.StatusIgnored {
				_, _ = fmt.Fprintf(w, "- %s: %s\n", res.InputPath, resultReason(res, "skipped by pre-flight check"))
			}
		}
	}

	if totals.errCount > 0 {
		_, _ = fmt.Fprintln(w, "Failed files:")
		for _, res := range results {
			if res.Status != domain.StatusIgnored && (res.Err != nil || res.Status == domain.StatusError) {
				_, _ = fmt.Fprintf(w, "- %s: %s\n", res.InputPath, resultReason(res, "conversion failed"))
			}
		}
	}
}

func resultReason(res domain.Result, fallback string) string {
	if res.Err != nil {
		return res.Err.Error()
	}
	return fallback
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

func clearTerminal(w io.Writer) {
	_, _ = fmt.Fprint(w, "\033[2J\033[H")
}
