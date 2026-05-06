package cli

import (
	"fmt"

	"github.com/pterm/pterm"
	"github.com/spf13/cobra"
	"github.com/nawodyaishan/pdf2md-tui/pkg/version"
)

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print the version number",
	Run: func(cmd *cobra.Command, args []string) {
		// Mini banner for version command
		pterm.DefaultCenter.WithCenterEachLineSeparately().Println(
			pterm.LightCyan("pdf2md-tui"),
		)

		data := [][]string{
			{"", ""},
			{"Version", pterm.LightMagenta(version.Version)},
			{"Commit", version.Commit},
			{"Built", version.Date},
			{"Go", version.GoVersion},
		}

		pterm.DefaultTable.
			WithHasHeader().
			WithData(data).
			Render()

		fmt.Println()
		pterm.DefaultCenter.WithCenterEachLineSeparately().Println(
			pterm.Gray(14, "by @nawodyaishan • github.com/nawodyaishan/pdf2md-tui"),
		)
	},
}
