package cli

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/nawodyaishan/pdf2md-tui/pkg/version"
)

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print the version number",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("pdf2md-tui %s\n", version.Version)
		fmt.Printf("  commit:  %s\n", version.Commit)
		fmt.Printf("  built:   %s\n", version.Date)
		fmt.Printf("  go:      %s\n", version.GoVersion)
	},
}
