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
	testscript.Main(m, map[string]func(){
		"pdf2md-tui": func() {
			if err := cli.Execute(); err != nil {
				os.Exit(1)
			}
		},
	})
}

// TestCLI runs end-to-end CLI integration tests via testscript
// Each .txtar file in testdata/scripts/ is executed as a separate subtest
func TestCLI(t *testing.T) {
	wd, _ := os.Getwd()
	projectRoot := filepath.Join(wd, "..", "..", "..")
	testdataDir := filepath.Join(projectRoot, "testdata")

	abs, _ := filepath.Abs(testdataDir)
	if _, err := os.Stat(filepath.Join(abs, "devops_project")); err != nil {
		t.Skipf("skipping CLI E2E tests; large PDF corpus is not present at %s", abs)
	}

	testscript.Run(t, testscript.Params{
		Dir: "testdata/scripts",
		Setup: func(env *testscript.Env) error {
			env.Setenv("TESTDATA_DIR", abs)
			return nil
		},
	})
}
