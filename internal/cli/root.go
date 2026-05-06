package cli

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var (
	outputDir   string
	recursive   bool
	workers     int
	dateFormat  string
	verbose     bool
	stripNoise  bool
)

var rootCmd = &cobra.Command{
	Use:   "pdf2md-tui [directory]",
	Short: "A high-performance CLI utility to batch-convert PDFs to Markdown.",
	Long: `pdf2md-tui scans a directory for PDF files and converts them into LLM-friendly Markdown format.
It provides a progress bar and handles concurrent processing for speed.

If no subcommand is provided, it defaults to the 'convert' command.`,
	Args: cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		// Default to convert
		return convertCmd.RunE(cmd, args)
	},
}

func init() {
	// Global flags
	rootCmd.PersistentFlags().StringVarP(&outputDir, "output", "o", "md", "Output subdirectory name")
	rootCmd.PersistentFlags().BoolVarP(&recursive, "recursive", "r", false, "Scan subdirectories for PDFs")
	rootCmd.PersistentFlags().IntVarP(&workers, "workers", "w", 0, "Concurrent conversion workers (default: NumCPU)")
	rootCmd.PersistentFlags().StringVar(&dateFormat, "date-format", "2006-01-02", "Date suffix format (e.g., 2006-01-02)")
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "Enable verbose logging")
	rootCmd.PersistentFlags().BoolVar(&stripNoise, "strip-noise", false, "Aggressively remove boilerplate for LLM optimization")

	rootCmd.AddCommand(convertCmd)
	rootCmd.AddCommand(versionCmd)
}

// Execute adds all child commands to the root command and sets flags appropriately.
func Execute() error {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		return err
	}
	return nil
}
