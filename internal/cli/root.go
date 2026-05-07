package cli

import (
	"fmt"
	"os"

	"github.com/nawodyaishan/pdf2md-tui/internal/tui"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

var (
	outputDir      string
	recursive      bool
	workers        int
	dateFormat     string
	verbose        bool
	stripNoise     bool
	forceOverwrite bool
)

var rootCmd = &cobra.Command{
	Use:   "pdf2md-tui [directory]",
	Short: "A high-performance CLI utility to batch-convert PDFs to Markdown.",
	Long: `pdf2md-tui scans a directory for PDF files and converts them into LLM-friendly Markdown format.
It provides a progress bar and handles concurrent processing for speed.

Run without arguments in a terminal to launch the interactive menu.`,
	Args: cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		// Show interactive menu when: no directory arg, no flags explicitly set, and running in a TTY.
		if len(args) == 0 && !anyFlagChanged(cmd) && tui.IsInteractive() {
			return runInteractiveMenu(cmd)
		}
		return convertCmd.RunE(cmd, args)
	},
}

func init() {
	rootCmd.PersistentFlags().StringVarP(&outputDir, "output", "o", "md", "Output subdirectory name")
	rootCmd.PersistentFlags().BoolVarP(&recursive, "recursive", "r", false, "Scan subdirectories for PDFs")
	rootCmd.PersistentFlags().IntVarP(&workers, "workers", "w", 0, "Concurrent conversion workers (default: NumCPU)")
	rootCmd.PersistentFlags().StringVar(&dateFormat, "date-format", "2006-01-02", "Date suffix format (e.g., 2006-01-02)")
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "Enable verbose logging")
	rootCmd.PersistentFlags().BoolVar(&stripNoise, "strip-noise", false, "Aggressively remove boilerplate for LLM optimization")
	rootCmd.PersistentFlags().BoolVarP(&forceOverwrite, "force", "f", false, "Overwrite existing output files without prompting")

	rootCmd.AddCommand(convertCmd)
	rootCmd.AddCommand(versionCmd)
}

// Execute runs the root command.
func Execute() error {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		return err
	}
	return nil
}

// anyFlagChanged reports whether the user explicitly set any flag on the command.
func anyFlagChanged(cmd *cobra.Command) bool {
	changed := false
	cmd.Flags().Visit(func(_ *pflag.Flag) {
		changed = true
	})
	return changed
}

// runInteractiveMenu shows the TUI main menu and delegates based on the user's choice.
func runInteractiveMenu(cmd *cobra.Command) error {
	action, cfg, err := tui.ShowMainMenu()
	if err != nil {
		return err
	}

	switch action {
	case "convert":
		recursive = cfg.Recursive
		stripNoise = cfg.StripNoise
		return convertCmd.RunE(cmd, []string{cfg.Directory})
	case "version":
		return versionCmd.RunE(cmd, nil)
	default: // "exit"
		return nil
	}
}
