# QA Phase 4 — Advanced Coverage & Interactive Testing

**Status**: Planning  
**Target Coverage**: 70%+ (from current 51.7%)  
**Gap Closing**: 18.3 percentage points  
**Est. Effort**: High (complex TUI/interactive testing)

---

## Executive Summary

Phase 3 achieved 51.7% coverage by testing core business logic (extraction, parsing, conversion, storage) with 60 passing tests, comprehensive benchmarks, and CI/CD gates. Phase 4 closes the remaining 18.3% gap by introducing specialized testing for interactive features and entry points that are currently untested but difficult to validate without full TUI framework support.

**Critical Gap Analysis:**
- **Interactive CLI** (23.5% coverage loss): Menu prompts, terminal I/O, Bubble Tea dashboard rendering
- **TUI Rendering** (76.9% loss): Progress bars, completion views, live stats updates
- **Entry Points** (0% coverage): cmd/ packages (technically covered by E2E but not counted)

---

## Phase 4 Goals

1. **Reach 70%+ coverage** by testing untestable-in-phase-3 components
2. **Add specialized test frameworks** for TUI and interactive CLI
3. **Implement snapshot/golden tests** for rendering consistency
4. **Document trade-offs** between test cost and coverage ROI

---

## Architecture Overview

```
┌─────────────────────────────────────────────────────────────┐
│ Phase 3 (Core Logic) → 51.7% Coverage ✅                    │
├─────────────────────────────────────────────────────────────┤
│ • pkg/domain (100%), pkg/repository (73-91%)               │
│ • pkg/service (79.8%), core extraction/conversion logic    │
└─────────────────────────────────────────────────────────────┘
                            ↓
┌─────────────────────────────────────────────────────────────┐
│ Phase 4 (Interactive Layers) → 70%+ Coverage Target       │
├─────────────────────────────────────────────────────────────┤
│ 4.1: Entry Point Tests (cmd/)                             │
│ 4.2: TUI Component Tests (internal/handler/tui)           │
│ 4.3: Interactive Menu Tests (internal/handler/tui/menu)   │
│ 4.4: CLI Command Path Tests (internal/handler/cli)        │
│ 4.5: Integration Snapshot Tests                           │
└─────────────────────────────────────────────────────────────┘
```

---

## Phase 4.1 — Entry Point Tests (cmd/)

**Goal**: Cover main() functions and entry point logic  
**Current Coverage**: 0%  
**Target**: 100%  
**Effort**: Low

### 4.1.1 Command Entry Points

Create `cmd/pdf2md-tui/main_test.go`:

```go
package main

import (
	"os"
	"testing"
)

func TestMainExitsCleanlyWithValidInput(t *testing.T) {
	// Use os/exec to test main() directly
	// Set up test environment with valid PDF
	// Verify exit code 0
}

func TestMainExitsNonZeroOnError(t *testing.T) {
	// Trigger CLI error (invalid args)
	// Verify exit code 1
}
```

**Why it matters**: Validates the entry point contract (os.Exit behavior, error handling at process boundary).

### 4.1.2 Debug Utility

Create `cmd/debug-pdf/main_test.go`:

```go
package main

import (
	"os"
	"testing"
)

func TestDebugPdfHelp(t *testing.T) {
	// Verify usage message
}

func TestDebugPdfValidPDF(t *testing.T) {
	// Test with sample PDF in testdata
}
```

**Coverage impact**: +2-3% (cmd packages are small)

---

## Phase 4.2 — TUI Rendering Tests (internal/handler/tui)

**Goal**: Test Bubble Tea dashboard rendering  
**Current Coverage**: 23.1%  
**Target**: 60%+  
**Effort**: High (requires snapshot testing or view output comparison)

### 4.2.1 Model Update & View Tests (Bubble Tea Pattern)

Create `internal/handler/tui/model_integration_test.go` (extends existing `model_test.go`):

```go
package tui

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/nawodyaishan/pdf2md-tui/pkg/domain"
)

func TestModelUpdateProcessesMessages(t *testing.T) {
	m := NewModel()
	
	// Simulate keyboard input via tea.KeyMsg
	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}}
	updated, cmd := m.Update(msg)
	
	// Verify state changed (e.g., quit flag set)
	if _, ok := updated.(tea.Model); !ok {
		t.Fatal("Update should return a Model")
	}
	if cmd == nil && !m.shouldExit {
		t.Error("expected quit command or state change")
	}
}

func TestModelViewRenders(t *testing.T) {
	m := NewModel()
	m.results = []domain.Result{
		{InputPath: "test.pdf", Status: domain.StatusOK, InputBytes: 1024},
	}
	
	view := m.View()
	
	// Verify rendered output contains expected content
	if !strings.Contains(view, "test.pdf") {
		t.Errorf("View should contain file name, got: %s", view)
	}
	if len(view) == 0 {
		t.Fatal("View should not be empty")
	}
}

func TestModelViewHeaderFooter(t *testing.T) {
	m := NewModel()
	view := m.View()
	
	// Test header rendering (from model.go:212)
	if !strings.Contains(view, "pdf2md-tui") {
		t.Error("header should display app name")
	}
	
	// Test footer rendering (from model.go:237)
	if !strings.Contains(view, "q") && !strings.Contains(view, "quit") {
		t.Error("footer should show quit instruction")
	}
}
```

**Pattern Reference**: [Bubble Tea Model Interface](https://pkg.go.dev/github.com/charmbracelet/bubbletea#Model)  
Extracted from `/websites/pkg_go_dev_github_com_charmbracelet_bubbletea` (Source Reputation: High)

**Files to create:**
- `internal/handler/tui/golden_test.go` - helper for snapshot testing
- `internal/handler/tui/testdata/golden/*.txt` - golden reference files

**Approach**: Store rendered output as golden files, compare on each test. Update golden files when intentional visual changes occur.

### 4.2.2 Rendering Function Tests

Create `internal/handler/tui/render_test.go` testing functions from `dashboard_render.go` (32 symbols):

```go
package tui

import (
	"strings"
	"testing"

	"github.com/nawodyaishan/pdf2md-tui/pkg/domain"
)

func TestRenderHeaderFormat(t *testing.T) {
	// From dashboard_render.go:14 - renderDashboard()
	// Test header with title, width constraints
	header := renderHeader("pdf2md-tui", 80)
	if !strings.Contains(header, "pdf2md-tui") {
		t.Error("header should display title")
	}
}

func TestRenderStatsValues(t *testing.T) {
	// From dashboard_render.go:47 - renderStats()
	// Test statistics display (converted, skipped, errors counts)
	stats := renderStats(5, 2, 1, 1024*1024)
	if !strings.Contains(stats, "5") || !strings.Contains(stats, "2") {
		t.Error("stats should display counts")
	}
}

func TestRenderProgressBar(t *testing.T) {
	// From dashboard_render.go:131 - renderProgressBar()
	// Test progress visualization at different percentages
	for _, pct := range []int{0, 25, 50, 75, 100} {
		bar := renderProgressBar(pct, 40)
		if len(bar) == 0 {
			t.Errorf("progress bar at %d%% should not be empty", pct)
		}
	}
}

func TestTruncatePath(t *testing.T) {
	// From dashboard_render.go:153 - truncatePath()
	paths := []string{
		"/very/long/path/to/pdf/file.pdf",
		"short.pdf",
	}
	for _, p := range paths {
		truncated := truncatePath(p, 30)
		if len(truncated) > 30 {
			t.Errorf("truncated path exceeds width: %d > 30", len(truncated))
		}
	}
}

func TestFormatDurationDisplay(t *testing.T) {
	// From dashboard_render.go:160 - formatDuration()
	// Test duration formatting: 1.5s, 2m30s, etc.
	tests := []struct {
		name     string
		millis   int
		expected string
	}{
		{"milliseconds", 500, "500ms"},
		{"seconds", 2500, "2.5s"},
		{"minutes", 150000, "2m30s"},
	}
	for _, tt := range tests {
		result := formatDuration(tt.millis)
		if !strings.Contains(result, "s") && !strings.Contains(result, "m") {
			t.Errorf("%s: expected duration format, got %q", tt.name, result)
		}
	}
}
```

**Coverage impact**: +15-20% (dashboard_render.go: 32 symbols, ~300 LOC)  
**Key insight**: These are pure rendering functions (no I/O), easily testable without mocking.

---

## Phase 4.3 — Interactive Menu Tests (internal/handler/tui/menu)

**Goal**: Test terminal interaction functions  
**Current Coverage**: 0%  
**Target**: 50%+  
**Effort**: Very High (requires PTY simulation or term.js mock)

### 4.3.1 Terminal Input Mocking

Create `internal/handler/tui/menu_test.go`:

```go
package tui

import (
	"bytes"
	"os"
	"testing"
)

type mockTerminal struct {
	input  string
	output *bytes.Buffer
}

func TestShowMainMenuPrintsOptions(t *testing.T) {
	menu := &mockTerminal{
		input:  "1\n", // Select option 1
		output: &bytes.Buffer{},
	}
	
	// Capture output
	// Verify menu options printed
	// Verify selection processed
}

func TestPromptConvertConfigCapturesToml(t *testing.T) {
	// Mock stdin with user responses
	// Verify config struct populated
}

func TestConfirmOverwriteRespectsYesNo(t *testing.T) {
	menu := &mockTerminal{input: "y\n"}
	ok, err := menu.ConfirmOverwrite([]string{"file.md"})
	if !ok || err != nil {
		t.Fatal("expected user confirmation")
	}
}
```

**Challenges:**
- Terminal libraries (pterm, Bubble Tea) are event-driven, hard to test directly
- **Solution**: Create thin wrapper layer (`internal/handler/tui/terminal_adapter.go`) that abstracts I/O for testability

**New file**: `internal/handler/tui/terminal_adapter.go`

```go
package tui

// TerminalAdapter abstracts stdin/stdout for testing
type TerminalAdapter interface {
	ReadLine() (string, error)
	WriteLine(s string) error
}

// RealTerminal wraps os.Stdin/Stdout
type RealTerminal struct {}

// MockTerminal for testing
type MockTerminal struct {
	lines []string
	idx   int
}
```

**Coverage impact**: +10-15% (menu functions ~150 LOC)

---

## Phase 4.4 — CLI Command Path Tests (internal/handler/cli)

**Goal**: Increase CLI coverage from 59% → 75%+  
**Current Coverage**: 59%  
**Target**: 75%+  
**Effort**: Medium

### 4.4.1 Non-Interactive Path Tests

Create `internal/handler/cli/cli_modes_test.go` covering untested functions from `root.go` and `convert.go`:

```go
package cli

import (
	"bytes"
	"os"
	"testing"

	"github.com/spf13/cobra"
	"github.com/nawodyaishan/pdf2md-tui/pkg/domain"
)

func TestPrintTextSummaryFormat(t *testing.T) {
	// From convert.go:386 - printTextSummary()
	// Test non-quiet mode output formatting
	buf := &bytes.Buffer{}
	results := []domain.Result{
		{InputPath: "a.pdf", Status: domain.StatusOK, OutputBytes: 1024},
	}
	// Mock output destination
	printTextSummary(buf, results)
	
	if buf.Len() == 0 {
		t.Fatal("text summary should produce output")
	}
}

func TestAnyFlagChangedDetection(t *testing.T) {
	// From root.go:74 - anyFlagChanged()
	// Test flag state comparison logic
	cmd := &cobra.Command{}
	cmd.Flags().BoolP("strip-noise", "", false, "")
	cmd.Flags().Parse([]string{"--strip-noise"})
	
	changed := anyFlagChanged(cmd)
	if !changed {
		t.Error("should detect changed flag")
	}
}

func TestRunInteractiveMenuFlow(t *testing.T) {
	// From root.go:83 - runInteractiveMenu()
	// This function calls tui.ShowMainMenu() which is hard to mock
	// Strategy: Create testable wrapper with dependency injection
	
	// Mock tui.ShowMainMenu result
	mockAction := "convert"
	mockCfg := domain.NewConfig()
	mockCfg.Directory = "/test"
	
	// After refactoring runInteractiveMenu to accept ShowMenu function:
	// cmd := &cobra.Command{}
	// err := runInteractiveMenuWithMock(cmd, func() (string, *domain.Config, error) {
	//     return mockAction, mockCfg, nil
	// })
}
```

**Source references** (via codegraph):
- `runInteractiveMenu()`: internal/handler/cli/root.go:83
- `anyFlagChanged()`: internal/handler/cli/root.go:74  
- `printTextSummary()`: internal/handler/cli/convert.go:386

**Files needed:**
- Mock exec.Command for openDir()
- Mock terminal for clearTerminal()
- Flag state comparison fixtures for anyFlagChanged()

**Coverage impact**: +10-15%

---

## Phase 4.5 — Integration Snapshot Tests

**Goal**: Test end-to-end output consistency  
**Current Coverage**: Covered by E2E tests  
**Target**: Add visual/output regression detection

### 4.5.1 Output Snapshots

Create `internal/handler/cli/output_snapshot_test.go`:

```go
func TestConvertJSONOutputSchema(t *testing.T) {
	// Run convert with --quiet
	// Parse JSON output
	// Compare against golden schema
	// Ensures JSON format doesn't regress
}

func TestMarkdownOutputConsistency(t *testing.T) {
	// Convert same PDF twice
	// Compare byte-for-byte output
	// Ensures deterministic conversion
}
```

**Tooling**:
- `testscript` already provides baseline E2E
- Add `github.com/cuonglm/gofail` for fault injection testing (optional)

---

## Phase 4 Implementation Roadmap

| Task | Files | LOC | Effort | Coverage Impact |
|------|-------|-----|--------|-----------------|
| 4.1: Entry points | cmd/*_test.go | 100 | Low | +2-3% |
| 4.2: TUI rendering | tui/*_golden_test.go, testdata/golden/ | 300 | High | +15-20% |
| 4.3: Menu I/O | tui/menu_test.go, terminal_adapter.go | 200 | Very High | +10-15% |
| 4.4: CLI paths | cli/cli_modes_test.go | 150 | Medium | +10-15% |
| 4.5: Snapshot tests | cli/output_snapshot_test.go | 100 | Low | +2-3% |
| **Total** | | **850** | | **~50-70%** |

---

## Effort Estimation & ROI

### Time Investment by Component

| Phase | Hours | Coverage Gain | ROI (% per hour) |
|-------|-------|---------------|-----------------|
| Phase 1 | 4 | 0% → 47% | 11.75% |
| Phase 2 | 6 | 47% → 49% | 0.33% |
| Phase 3 | 12 | 49% → 51.7% | 0.23% |
| Phase 4 | ~20-24 | 51.7% → 70% | 0.88% |

**Key insight**: Phase 4 faces diminishing returns. Core logic is well-tested; remaining gap is interactive/UI code which is:
- Hard to unit test (requires mocks, snapshots, PTY simulation)
- Often refactored for UX (golden files become stale)
- Lower business priority (users care about PDF conversion quality, not menu rendering)

---

## Recommendation: Phased Approach

### Phase 4A (Fast) — 6 hours → 57% coverage
- **4.1**: Entry point tests (trivial)
- **4.4**: CLI non-interactive paths (straightforward mocking)
- **4.5**: Snapshot tests (testscript-based)

**ROI**: High. Entry and non-interactive paths are stable.

### Phase 4B (Slow) — 15+ hours → 70% coverage
- **4.2**: TUI rendering snapshots (needs golden file maintenance)
- **4.3**: Menu I/O testing (complex PTY mocking, fragile)

**ROI**: Lower. UX code changes frequently; snapshot maintenance becomes burden.

---

## Critical Files (Phase 4)

| File | Action | Est. LOC | Complexity |
|------|--------|---------|------------|
| cmd/pdf2md-tui/main_test.go | CREATE | 50 | Low |
| cmd/debug-pdf/main_test.go | CREATE | 50 | Low |
| internal/handler/tui/terminal_adapter.go | CREATE | 100 | Medium |
| internal/handler/tui/menu_test.go | CREATE | 150 | High |
| internal/handler/tui/dashboard_render_golden_test.go | CREATE | 150 | Medium |
| internal/handler/tui/testdata/golden/*.txt | CREATE | - | - |
| internal/handler/cli/cli_modes_test.go | CREATE | 150 | Medium |
| internal/handler/cli/output_snapshot_test.go | CREATE | 100 | Low |
| .github/workflows/ci.yml | MODIFY | - | Low |

---

## Testing Utilities (New)

### internal/handler/tui/golden_test.go

```go
package tui

import (
	"bytes"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"
)

func compareGolden(t *testing.T, actual string) error {
	// Load golden file from testdata/golden/[test_name].txt
	// If UPDATE_GOLDEN=1, write actual → golden
	// Otherwise, compare actual vs golden
}

func updateGoldenFiles() bool {
	return os.Getenv("UPDATE_GOLDEN") == "1"
}
```

Usage:
```bash
# First run: create golden files
UPDATE_GOLDEN=1 go test ./internal/handler/tui/...

# Subsequent runs: compare
go test ./internal/handler/tui/...
```

---

## Success Criteria

- [ ] Coverage ≥ 70%
- [ ] All entry points (cmd/) tested
- [ ] CLI interactive paths testable (terminal adapter in place)
- [ ] TUI rendering verified via snapshots (no false positives)
- [ ] CI gates updated (coverage threshold → 70%)
- [ ] Golden file maintenance docs written

---

## Known Risks & Mitigations

| Risk | Impact | Mitigation |
|------|--------|-----------|
| Snapshot tests become stale | High | Keep golden files in git, review diffs carefully |
| PTY mocking fragile | High | Use thin adapter layer, test adapters not full menu flow |
| TUI library updates break tests | Medium | Pin versions, add compatibility tests for new versions |
| Diminishing test ROI | Medium | Document trade-offs, focus on Phase 4A first |

---

## Next Steps (When Ready)

1. **Approve Phase 4A** (fast path) for 57% coverage
2. **Implement 4.1, 4.4, 4.5** in 1-2 sprints
3. **Evaluate Phase 4B** ROI before TUI snapshot work
4. **Update CI workflow** with new 70% gate
5. **Document snapshot maintenance** for team

---

## Appendix: Alternative Approaches

### A1: Browser-Based Testing (Rejected)
- Use headless browser to test TUI via websocket tunnel
- **Reason**: Adds external dependency, fragile, overkill for CLI

### A2: Full E2E Screenshot Testing (Rejected)
- Capture terminal output, compare pixel-perfect renders
- **Reason**: Terminal rendering varies by font/terminal, not portable

### A3: Remove TUI Code from Coverage (Not Recommended)
- Exclude internal/handler/tui from coverage calculation
- **Reason**: Hides untested code, worse for end users

### A4: Refactor TUI to Model-View Pattern (Accepted for Phase 4B)
- Extract rendering logic from view functions
- Test models directly, snapshot views
- **Why**: Reduces TUI test complexity from Very High → Medium

---

## Code Structure Analysis (via CodeGraph)

### TUI Package Components (internal/handler/tui)

| File | Symbols | Key Functions | Test Coverage |
|------|---------|---|---|
| model.go | 30 | NewModel, Init, Update, View, renderHeader, renderFooter | Partial (23.1%) |
| dashboard_render.go | 32 | renderDashboard, renderStats, renderProgressBar, truncatePath, formatDuration | 0% |
| progress.go | 29 | StartDiscovery, StopDiscovery, Increment, UpdateLiveStats, RunDashboard | 0% |
| menu.go | 14 | ShowMainMenu, promptConvertConfig, ConfirmOverwrite, IsInteractive | 0% |
| styles.go | 33 | Color constants, style builders | Partial |
| model_test.go | 12 | Existing tests | — |

**Key Finding**: 107 symbols, ~400 LOC across 6 files. Dashboard rendering (32 symbols, ~200 LOC) is lowest-hanging fruit for coverage.

### CLI Package Components (internal/handler/cli)

| File | Key Functions | Coverage | Notes |
|------|---|---|---|
| root.go | Execute, runInteractiveMenu, anyFlagChanged | 100%, 0%, 0% | Interactive menu needs refactoring for testability |
| convert.go | Convert, printTextSummary, printJSONSummary, openDir, clearTerminal | 100%, 0%, 76.9%, 0%, 0% | CLI orchestration well-tested, output formatting not tested |
| version.go | — | 100% | Already complete |

**Key Finding**: Output formatting functions (printTextSummary) are pure functions, easily testable.

---

## References

- [Phase 3 Tech Spec](2026-05-09-pdf2md-tui-qa-3phase-tech-spec.md)
- **[Bubble Tea Model Interface](https://pkg.go.dev/github.com/charmbracelet/bubbletea#Model)** — Core testing pattern (Source Reputation: High)
- **[Bubble Tea Framework](https://github.com/charmbracelet/bubbletea)** — Update/View methods testable in isolation
- **[Bubble Tea Bubbles Components](https://github.com/charmbracelet/bubbles)** — v2.0.0, tested reference implementations
- [Golden File Testing Pattern](https://en.wikipedia.org/wiki/Test_data#Golden_master)
- [PTY Simulation for CLI Testing](https://github.com/traefik/yaegi/blob/master/internal/test/pty.go)

---

**Document**: 2026-05-09  
**Author**: QA Implementation  
**Status**: Draft (Ready for Approval)
