package cli_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/nawodyaishan/pdf2md-tui/internal/handler/cli"
	"github.com/rogpeppe/go-internal/testscript"
)

// TestMain registers the pdf2md-tui command for use in testscript scenarios
func TestMain(m *testing.M) {
	//nolint:staticcheck // testscript.RunMain is the standard pattern for this library
	os.Exit(testscript.RunMain(m, map[string]func() int{
		"pdf2md-tui": func() int {
			if err := cli.Execute(); err != nil {
				return 1
			}
			return 0
		},
	}))
}

// TestCLI runs end-to-end CLI integration tests via testscript
// Each .txtar file in testdata/scripts/ is executed as a separate subtest
func TestCLI(t *testing.T) {
	// Resolve project testdata directory
	wd, _ := os.Getwd()
	projectRoot := filepath.Join(wd, "..", "..", "..")
	testdataDir := filepath.Join(projectRoot, "testdata")

	// Verify and resolve absolute path
	abs, _ := filepath.Abs(testdataDir)
	if _, err := os.Stat(abs); err != nil {
		// Fallback: try one more level up
		abs, _ = filepath.Abs(filepath.Join(wd, "..", "..", "..", "..", "testdata"))
	}

	testscript.Run(t, testscript.Params{
		Dir: "testdata/scripts",
		Setup: func(env *testscript.Env) error {
			env.Setenv("TESTDATA_DIR", abs)
			return nil
		},
	})
}
