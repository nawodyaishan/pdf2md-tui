# Implementation Plan: Near-term v0.x — "Reliability & Reach"

> [!NOTE]
> **Pre-requisites completed:** `ROADMAP.md` rewritten with North Star vision. `docs/ARCHITECTURE.md` updated with scalable architecture, code quality standards, and test strategy. `README.md` updated with new vision, Look Twice methodology, and roadmap summary. Features 1 (OCR Detection) and 2 (Image Extraction) have been implemented.

This plan covers the 3 features in the Near-term roadmap tier, ordered by dependency:

```mermaid
graph LR
    C[3. Chunking Tables]
    D[4. Zero-Arg]
    E[5. Quiet Mode]
```

---

## Feature 3: 🧱 Chunking-Safe Table Output

**Goals:** G2 | **Label:** `area/extraction`

### What Changes

Ensure that the Markdown output is structured so that downstream chunkers (LlamaIndex `SentenceSplitter`, Docling `HybridChunker`) cannot destroy table integrity or orphan headers from their paragraphs.

### Implementation

#### [MODIFY] [internal/repository/pdf/tables.go](./internal/repository/pdf/tables.go)

**Table fencing:** Ensure tables are surrounded by exactly two blank lines (`\n\n`) before and after. This makes them indivisible atomic units for newline-based chunkers:

```go
// In renderRowsAsMarkdown(), when emitting a table:
buf.WriteString("\n")  // blank line before table
// ... render pipe table rows ...
buf.WriteString("\n")  // blank line after table (existing)
```

#### [MODIFY] [internal/repository/pdf/pdf.go](./internal/repository/pdf/pdf.go)

**Header-paragraph binding:** In the page text assembly loop, ensure that Markdown headers (`# `, `## `, etc.) are never separated from their immediately following paragraph by more than one newline. This prevents chunkers from splitting a header into one chunk and its body into another.

```go
// Post-processing pass: collapse any sequence of >2 newlines between a header
// line and its following paragraph to exactly 1 newline.
func bindHeadersToParagraphs(text string) string {
    // Regex: header line followed by excessive whitespace before next content
    re := regexp.MustCompile(`(^#{1,6}\s+.+\n)\n{2,}`)
    return re.ReplaceAllString(text, "$1\n")
}
```

### Test Requirements

#### [MODIFY] `internal/repository/pdf/tables_test.go`

| Test | Description |
|------|-------------|
| `TestRenderRowsAsMarkdown_TableFencedWithBlankLines` | Table output has exactly `\n\n` before and `\n\n` after |
| `TestRenderRowsAsMarkdown_ConsecutiveTables` | Two adjacent tables separated by `\n\n` only |

#### [NEW or MODIFY] `internal/repository/pdf/pdf_test.go`

| Test | Description |
|------|-------------|
| `TestBindHeadersToParagraphs_CollapseExcessiveGap` | `"# Title\n\n\n\nParagraph"` → `"# Title\nParagraph"` |
| `TestBindHeadersToParagraphs_PreserveSingleGap` | `"# Title\nParagraph"` remains unchanged |
| `TestBindHeadersToParagraphs_NoHeaders` | Plain text passes through unchanged |

### Acceptance Criteria

- [ ] GFM pipe tables in output are always fenced by blank lines
- [ ] Headers are never orphaned from their immediately following paragraph
- [ ] All existing `tables_test.go` tests pass without modification
- [ ] Output is validated against a known `testdata/` PDF with tables

---

## Feature 4: ⚡ Zero-Arg Execution

**Goals:** All | **Label:** `area/cli`

### What Changes

Allow `pdf2md-tui` to be invoked without any arguments. When run in a TTY, it defaults to the current working directory and launches the interactive wizard. When no PDFs are found, it prints a graceful message.

### Implementation

#### [MODIFY] [internal/handler/cli/root.go](./internal/handler/cli/root.go)

The current root command already handles zero-arg with TTY detection:

```go
if len(args) == 0 && !anyFlagChanged(cmd) && tui.IsInteractive() {
    return runInteractiveMenu(cmd)
}
return convertCmd.RunE(cmd, args)
```

**Changes needed:**
1. When running non-interactively with zero args (piped input, CI), default to CWD conversion instead of showing the menu.
2. After running `discovery.FindPDFs(".", recursive)`, if zero PDFs found → print a graceful message and exit `0`.

```go
// Non-interactive zero-arg: convert CWD
if len(args) == 0 && !anyFlagChanged(cmd) && !tui.IsInteractive() {
    return convertCmd.RunE(cmd, []string{"."})
}
```

#### [MODIFY] [internal/handler/cli/convert.go](./internal/handler/cli/convert.go)

Enhance the "zero PDFs found" path with a user-friendly message:

```go
if len(pdfFiles) == 0 {
    ui.PrintNoPDFsFound(targetDir)  // "No PDFs found in ./current-dir. Run pdf2md-tui --help for usage."
    return nil
}
```

#### [MODIFY] [internal/handler/tui/progress.go](./internal/handler/tui/progress.go)

Add `PrintNoPDFsFound(dir string)` method that prints a styled, helpful message.

### Test Requirements

| Test | Description |
|------|-------------|
| `TestConvertCmd_DefaultsToCWD` | With no args, `targetDir` resolves to `"."` |
| `TestConvertCmd_NoPDFsGracefulExit` | Empty directory → exit 0, no error |

### Acceptance Criteria

- [ ] Running `pdf2md-tui` in a directory with PDFs starts conversion
- [ ] Running `pdf2md-tui` in an empty directory prints a helpful message and exits 0
- [ ] Interactive mode (TTY) still shows the wizard menu
- [ ] Non-interactive mode (piped/CI) converts CWD directly

---

## Feature 5: 🔇 CI-Friendly Quiet Mode

**Goals:** G4 | **Label:** `area/cli` | **Depends on:** Status enum (Implemented)

### What Changes

Add a `--quiet` flag that suppresses all TUI output (banner, spinner, progress bar), emitting only a structured JSON summary to stdout. This enables clean piping to `jq` and is the foundation for the future MCP server.

### Implementation

#### [MODIFY] [internal/handler/cli/root.go](./internal/handler/cli/root.go)

Add the flag:
```go
var quiet bool

rootCmd.PersistentFlags().BoolVarP(&quiet, "quiet", "q", false, "Suppress TUI output; emit JSON summary to stdout")
```

#### [NEW] `internal/domain/summary.go`

JSON-serializable summary struct:

```go
type Summary struct {
    Converted int            `json:"converted"`
    Skipped   int            `json:"skipped"`
    Errors    int            `json:"errors"`
    Duration  string         `json:"duration"`
    Files     []FileSummary  `json:"files"`
}

type FileSummary struct {
    Input      string `json:"input"`
    Output     string `json:"output,omitempty"`
    Status     string `json:"status"`  // "ok", "ignored", "error"
    Error      string `json:"error,omitempty"`
    InputBytes int64  `json:"input_bytes"`
    OutputBytes int64 `json:"output_bytes,omitempty"`
}
```

#### [MODIFY] [internal/handler/cli/convert.go](./internal/handler/cli/convert.go)

- If `quiet` is true:
  - Skip all `ui.*` calls (banner, spinner, progress bar, summary).
  - After all results are collected, marshal `Summary` to JSON and write to `os.Stdout`.
- If `quiet` is false: existing behavior unchanged.

#### Exit Code Semantics

| Code | Meaning |
|------|---------|
| `0` | All files processed (converted + skipped). No failures. |
| `1` | At least one file failed with an error. |
| `2` | Invalid arguments or configuration. |

### Test Requirements

#### [NEW] `internal/domain/summary_test.go`

| Test | Description |
|------|-------------|
| `TestSummary_JSONMarshal` | Verify correct JSON output format |
| `TestSummary_EmptyFiles` | Zero files → valid JSON with zeros |
| `TestSummary_MixedStatuses` | OK + ignored + error → correct counts |

### Acceptance Criteria

- [ ] `pdf2md-tui convert ./testdata --quiet` outputs valid JSON to stdout, nothing else
- [ ] JSON is parseable by `jq` — `pdf2md-tui convert ./testdata -q | jq .converted`
- [ ] Exit code is `0` when all files succeed or are gracefully skipped
- [ ] Exit code is `1` when any file fails
- [ ] `--quiet` with `--verbose` is rejected or `--quiet` takes precedence (document the behavior)

---

## Verification Plan

### Automated Tests

```bash
# Run full suite with race detection
make test

# Run only repository tests
go test -race -v ./internal/repository/pdf/

# Run with coverage enforcement
go test -race -coverprofile=coverage.out ./...
go tool cover -func=coverage.out | grep total

# Build with CGO disabled (must pass)
CGO_ENABLED=0 go build ./cmd/pdf2md-tui/
```

### Smoke Tests

```bash
# Feature 3: Chunking-safe tables
# Verify: output .md files have tables fenced with blank lines

# Feature 4: Zero-arg
cd testdata && ../bin/pdf2md-tui
# Verify: wizard launches or conversion starts

# Feature 5: Quiet mode
./bin/pdf2md-tui convert ./testdata -q | jq .
# Verify: valid JSON, no TUI output
```

### Manual Verification

- Inspect generated Markdown files for correct table formatting.
- Test on PDFs from real-world sources (research papers, financial reports).
