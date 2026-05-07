# Architecture

`pdf2md-tui` is a Go CLI that converts PDF files to LLM-optimized Markdown. This document describes the internal structure, data flow, key design decisions, scalability principles, code quality standards, and test strategy for contributors.

---

## Package Layout

```
pdf2md-tui/
├── cmd/pdf2md-tui/        # Binary entry point
│   └── main.go            # Calls cli.Execute(); nothing else
├── internal/
│   ├── cli/               # Cobra command definitions + flag wiring
│   │   ├── root.go        # Root command; declares all global flags
│   │   ├── convert.go     # "convert" subcommand — orchestrates the pipeline
│   │   └── version.go     # "version" subcommand — prints build metadata
│   ├── converter/         # PDF extraction and Markdown generation
│   │   ├── converter.go   # Converter struct, Convert(), cleanExtractedText(), applyLLMOptimizations()
│   │   ├── converter_test.go
│   │   ├── analyze.go     # [PLANNED] AnalyzePDF() pre-flight heuristic for OCR detection
│   │   ├── images.go      # [PLANNED] pdfcpu-based image extraction pipeline
│   │   ├── tables.go      # Positional extraction: coalesceChars → groupIntoRows → renderRowsAsMarkdown
│   │   └── tables_test.go
│   ├── discovery/         # File system scanning
│   │   ├── discovery.go   # FindPDFs(dir, recursive) → []string
│   │   └── discovery_test.go
│   └── tui/               # Terminal UI wrappers
│       ├── progress.go    # pterm: banner, spinner, progress bar, summary table
│       └── menu.go        # Interactive TUI menu
├── pkg/
│   └── version/           # Build-time version variables (injected via ldflags)
│       └── version.go
├── testdata/              # Fixture PDFs for integration tests
└── docs/                  # Internal documentation
```

`internal/` packages are not importable by external Go modules. `pkg/version` is exported because GoReleaser tooling may reference it.

---

## Data Flow

```
pdf2md-tui convert <dir> [flags]
        │
        ▼
┌──────────────────────────────────────┐
│  internal/cli/convert.go             │
│  1. Validate <dir> argument          │
│  2. ui.PrintBanner()                 │
│  3. ui.StartDiscovery()              │
│  4. discovery.FindPDFs(dir, -r)      │  ──▶  []string (absolute PDF paths)
│  5. ui.StopDiscovery(count)          │
│  6. os.MkdirAll(outDir)              │
│  7. [PLANNED] Pre-flight analysis    │  ──▶  Partition: convertible vs. OCR-required
│  8. Launch worker pool               │
└──────────────────────────────────────┘
        │
        ▼  (per worker)
┌──────────────────────────────────────┐
│  internal/converter/converter.go     │
│  Converter.Convert(pdfPath, outDir)  │
│  1. os.Stat → InputBytes             │
│  2. pdf.Open (ledongthuc/pdf)        │
│  3. For each page:                   │
│     a. extractWithTables(page)       │  ──▶  Markdown with pipe tables
│     b. fallback: GetPlainText()      │       + cleanExtractedText()
│     c. applyLLMOptimizations()       │       (if --strip-noise)
│  4. [PLANNED] extractImages(page)    │  ──▶  ./images/{doc}/ + MD references
│  5. os.WriteFile → .md file          │
│  6. Return Result{metrics, error}    │
└──────────────────────────────────────┘
        │
        ▼
┌──────────────────────────────────────┐
│  internal/tui/progress.go            │
│  PrintSummary(in, out, dur, errors)  │
│  Shows token savings estimate        │
│  [PLANNED] Shows ignored/skipped     │
└──────────────────────────────────────┘
```

---

## Worker Pool

The concurrency model in `cli/convert.go` is a classic fan-out/fan-in pool:

```go
jobs    := make(chan string, len(pdfFiles))   // buffered; all jobs queued before workers start
results := make(chan converter.Result, len(pdfFiles))

var wg sync.WaitGroup
for w := 0; w < numWorkers; w++ {
    wg.Add(1)
    go func() {
        defer wg.Done()
        for pdf := range jobs {
            results <- conv.Convert(pdf, outDir)
            ui.Increment()               // thread-safe pterm call
        }
    }()
}

for _, f := range pdfFiles { jobs <- f }
close(jobs)                              // signals workers to exit range loop

go func() { wg.Wait(); close(results) }()

for res := range results { /* aggregate */ }
```

- `jobs` is fully pre-loaded before workers start, so no send-side blocking after launch.
- `results` capacity matches the job count, eliminating receiver-side blocking within workers.
- `ui.Increment()` is called inside the worker goroutine; pterm's progress bar is internally goroutine-safe.

---

## PDF Extraction: Two-Path Strategy

Every page goes through two attempts, in order:

### Path 1 — Positional extraction (`tables.go`)

Uses `page.Content()` from `ledongthuc/pdf`, which returns raw `[]pdf.Text` — individual character glyphs with X/Y coordinates and font size.

```
pdf.Text{S: "H", X: 72.0, Y: 720.0, FontSize: 12}
pdf.Text{S: "i", X: 78.5, Y: 720.0, FontSize: 12}
...
```

The pipeline in `tables.go`:

1. **`coalesceChars`** — Sort characters by Y descending then X ascending. Merge adjacent characters into words when `ΔX < charWidth × 1.8` and `ΔY < 2pt`.
2. **`groupIntoRows`** — Cluster words into rows where `ΔY < 3pt` (tolerance for baseline variation).
3. **`findTableEnd` + `detectColumnPositions`** — Sliding window over row sequences; a run of rows each having ≥3 columns spaced ≥50pt apart is classified as a table. Column positions must appear in ≥40% of rows.
4. **`renderRowsAsMarkdown`** — Table regions become GFM pipe tables; non-table rows emit as plain text lines.

### Path 2 — Plain-text fallback (`converter.go`)

If positional extraction yields only whitespace (image-only page, encoding issues), `page.GetPlainText(nil)` is called. The library returns words separated by `" \n"` (single-space lines). `cleanExtractedText` reassembles paragraphs by treating truly-empty lines as paragraph breaks and single-space lines as intra-word separators.

### `--strip-noise` post-pass

`applyLLMOptimizations` runs on the output of either path when the flag is set:
- Removes standalone digit lines (page numbers): `(?m)^\s*\d+\s*$`
- Collapses all whitespace runs to a single space — deliberately lossy, trades Markdown structure for token density.

---

## Version Injection

`pkg/version` variables default to `"dev"`, `"none"`, `"unknown"`. They are overwritten at link time:

```makefile
LDFLAGS := \
  -X github.com/nawodyaishan/pdf2md-tui/pkg/version.Version=$(VERSION) \
  -X github.com/nawodyaishan/pdf2md-tui/pkg/version.Commit=$(shell git rev-parse --short HEAD) \
  -X github.com/nawodyaishan/pdf2md-tui/pkg/version.Date=$(shell date -u +%Y-%m-%dT%H:%M:%SZ) \
  -X github.com/nawodyaishan/pdf2md-tui/pkg/version.GoVersion=$(GOVERSION)
```

GoReleaser performs the same injection using `{{.Version}}`, `{{.ShortCommit}}`, and `{{.Date}}` template variables.

---

## TUI Design

`internal/tui` wraps `github.com/pterm/pterm` and exposes four lifecycle calls:

| Method | Phase | Output |
|--------|-------|--------|
| `PrintBanner()` | Startup | Branded header with version |
| `StartDiscovery()` / `StopDiscovery(n)` | Scanning | Animated spinner → "Found N PDFs" |
| `StartConversion(n)` / `Increment()` / `StopConversion()` | Converting | Progress bar |
| `PrintSummary(in, out, dur, errs)` | Complete | Table with sizes, durations, token savings |

Token savings are estimated as `(inputBytes - outputBytes) / inputBytes × 100` — displayed as a percentage reduction.

When stdout is not a TTY, pterm auto-disables animations. The `--quiet` flag (planned) will suppress all TUI output for CI use, emitting only a JSON summary to stdout.

---

## Key Dependencies

| Dependency | Purpose | Notes |
|-----------|---------|-------|
| `github.com/ledongthuc/pdf` | PDF parsing, glyph extraction | Pure Go, no CGO; chosen for positional text API (`Content()`) |
| `github.com/spf13/cobra` | CLI framework | Subcommands, flag binding, auto-help |
| `github.com/pterm/pterm` | TUI components | Spinner, progress bar, tables, styled output |
| `github.com/pdfcpu/pdfcpu` | **[PLANNED]** Image extraction | Pure Go, Apache 2.0, 8.5k★; `ExtractImages` API for XObject parsing |

---

## Scalable Architecture

This section defines the architectural principles and patterns that ensure the codebase scales cleanly as we add new capabilities (image extraction, OCR detection, MCP server, quiet mode).

### Core Principle: Interface-Driven Package Boundaries

Every package boundary must be expressible as a Go interface. This enables testability (mock injection), composability (swap implementations), and MCP readiness (wrap the same interface in a server binary).

```
┌─────────────────────────────────────────────────────┐
│  cmd/pdf2md-tui/main.go                             │
│  cmd/pdf2md-mcp/main.go  [PLANNED]                  │
│         │                                           │
│         ▼                                           │
│  internal/cli/   OR   mcp-server wrapper            │
│         │                                           │
│         ▼                                           │
│  internal/converter (core interface)                │
│         │                                           │
│    ┌────┼────────────┐                              │
│    ▼    ▼            ▼                              │
│  analyze.go  tables.go  images.go                   │
│         │                                           │
│  internal/discovery (file scanning interface)       │
│         │                                           │
│  internal/tui (display adapter — NEVER imported     │
│                by converter or discovery)            │
└─────────────────────────────────────────────────────┘
```

### Package Isolation Rules

| Rule | Constraint |
|------|------------|
| **converter → NO tui** | `internal/converter` must **never** import `internal/tui` or `pterm`. The core engine must be embeddable by any consumer (CLI, MCP server, library). |
| **converter → NO cli** | `internal/converter` must not import `internal/cli`, `cobra`, or `pflag`. Options are passed via struct, not read from globals. |
| **discovery → NO tui** | `internal/discovery` is a pure file-scanning utility. No display concerns. |
| **tui → NO converter** | `internal/tui` formats data it receives; it does not call converter logic directly. The orchestrator (`cli/convert.go`) bridges them. |
| **cli → orchestrator only** | `internal/cli` is the glue layer that wires converter, discovery, and tui together. All composition happens here. |

### Dependency Injection via Options Struct

All converter configuration flows through a typed options struct — never through global variables or flag references:

```go
// internal/converter/converter.go
type Options struct {
    DateFormat    string
    StripNoise    bool
    ExtractImages bool   // [PLANNED]
}

type Result struct {
    InputPath    string
    OutputPath   string
    InputBytes   int64
    OutputBytes  int64
    Duration     time.Duration
    Status       Status   // [PLANNED] StatusOK | StatusIgnored | StatusError
    Err          error
}
```

### Status Enum Pattern (Planned)

To support graceful degradation, `Result` will carry a typed status:

```go
type Status int

const (
    StatusOK      Status = iota // Converted successfully
    StatusIgnored               // Skipped (e.g., requires OCR)
    StatusError                 // Failed with error
)
```

The CLI aggregates these statuses for the summary table without the converter knowing about TUI rendering.

### Error Sentinel Pattern

Domain-specific errors use typed sentinels for clean pattern matching in the worker pool:

```go
// internal/converter/analyze.go
var ErrRequiresOCR = errors.New("converter: document requires OCR")

func AnalyzePDF(path string) (PageAnalysis, error) {
    // ...
    if textYield < threshold && imageCount > 0 {
        return analysis, ErrRequiresOCR
    }
    return analysis, nil
}
```

The CLI catches these via `errors.Is(err, converter.ErrRequiresOCR)` — no string matching.

### MCP Readiness

The package architecture is designed so that a future `cmd/pdf2md-mcp/` binary can import `internal/converter` directly:

```go
// cmd/pdf2md-mcp/main.go (future)
func handleConvertTool(req mcp.Request) mcp.Response {
    conv := converter.New(converter.Options{...})
    result := conv.Convert(req.Params["path"], req.Params["outDir"])
    return mcp.Response{Data: result}
}
```

No TUI, no cobra — just the core converter behind a protocol adapter.

---

## Code Quality Standards

These standards are mandatory for all contributions. They are enforced by CI and reviewers.

### Go Conventions

| Area | Rule |
|------|------|
| **Naming** | Follow [Effective Go](https://go.dev/doc/effective_go) naming: exported names are `PascalCase`, unexported are `camelCase`. Acronyms are uppercase (`PDF`, `URL`, `OCR`). |
| **File naming** | Lowercase, underscores for multi-word: `converter.go`, `tables_test.go`, `analyze.go`. |
| **Package naming** | Short, lowercase, singular nouns: `converter`, `discovery`, `tui`. Never `utils` or `helpers`. |
| **Line length** | Soft limit of 120 characters. Hard limit of 140. Use intermediate variables to break long expressions. |
| **Comments** | All exported types, functions, and constants must have a godoc comment starting with the name. Explain *why*, not *what*. |

### Error Handling

| Pattern | When |
|---------|------|
| **Return errors** | Always return `error` as the last return value. Never panic in library code. |
| **Wrap with context** | Use `fmt.Errorf("operation: %w", err)` to preserve the error chain. |
| **Typed sentinels** | Define `var ErrXxx = errors.New(...)` for domain-specific errors the caller needs to match with `errors.Is()`. |
| **Panic recovery** | Only in goroutine entry points (worker pool, page extraction). The `safeExtractPage()` pattern is the canonical example. |
| **No `_` for errors** | Never discard errors with `_` unless inside a `defer` where the function already has a return error. Document the justification. |

### Concurrency Rules

| Rule | Rationale |
|------|-----------|
| **No shared mutable state** | Workers communicate via channels. The `Result` struct is the unit of communication. |
| **`sync.WaitGroup` for fan-out** | One `wg.Add(1)` per goroutine, `defer wg.Done()` at the top of the goroutine body. |
| **Buffered channels sized to job count** | Prevents goroutine leaks from blocked sends. |
| **No goroutines in `converter`** | Concurrency is the orchestrator's (`cli/convert.go`) responsibility. The converter package is synchronous and single-threaded per call. |
| **`-race` in all CI test runs** | Non-negotiable. Every `go test` invocation includes `-race`. |

### Function Design

| Principle | Rule |
|-----------|------|
| **Single responsibility** | Functions do one thing. If a function name contains "and", split it. |
| **Max 50 lines** | If a function exceeds 50 lines, extract helper functions. `renderRowsAsMarkdown` is at the limit. |
| **Max 3 parameters** | Beyond 3, use an options struct. `Convert(path, outDir string)` is fine. Future additions go in `Options`. |
| **Early returns** | Guard clauses at the top. Avoid deep nesting. `if err != nil { return ... }` is idiomatic. |
| **No magic numbers** | Use named constants. `const minGap = 50.0` in `tables.go` is the pattern. |

### Import Organization

Standard library imports are separated from third-party imports by a blank line. Internal imports come after third-party:

```go
import (
    "fmt"
    "os"
    "strings"

    "github.com/ledongthuc/pdf"

    "github.com/nawodyaishan/pdf2md-tui/internal/tui"
)
```

### Zero-Cgo Enforcement

All builds must pass with `CGO_ENABLED=0`. This is enforced in CI:

```yaml
env:
  CGO_ENABLED: 0
```

Any PR introducing a Cgo dependency will be rejected. This is a hard architectural constraint, not a preference.

---

## Testing Strategy

### Test Philosophy

Tests are first-class code. They follow the same quality standards as production code:
- **Readable** — a test name is a specification.
- **Isolated** — tests share no state. Each creates its own temp directories via `t.TempDir()`.
- **Fast** — unit tests complete in < 1 second total. Integration tests are gated behind build tags.
- **Deterministic** — no randomness, no external network calls, no time-sensitive assertions.

### Test Matrix

| Level | Package | What | How | CI Gate |
|-------|---------|------|-----|---------|
| **Unit** | `converter` | `cleanExtractedText`, `applyLLMOptimizations`, `outputFilename` | Table-driven, no I/O | ✅ Always |
| **Unit** | `converter` | `coalesceChars`, `groupIntoRows`, `detectColumnPositions`, `assignCellsToColumns`, `findTableEnd` | Synthetic `pdf.Text` slices | ✅ Always |
| **Unit** | `converter` | `renderRowsAsMarkdown` (mixed table/text) | Synthetic `tableRow` slices | ✅ Always |
| **Unit** | `converter` | Panic recovery (`safeExtractPage`) | Controlled panics | ✅ Always |
| **Unit** | `converter` | `Convert()` error paths (missing file, invalid PDF) | `t.TempDir()` with dummy files | ✅ Always |
| **Unit** | `discovery` | `FindPDFs` flat, recursive, empty, permission errors | `t.TempDir()` with fixture structure | ✅ Always |
| **Unit** | `converter` | **[PLANNED]** `AnalyzePDF()` heuristic | Synthetic page data | ✅ Always |
| **Unit** | `converter` | **[PLANNED]** `extractImages()` with mock XObjects | In-memory test PDFs | ✅ Always |
| **Integration** | `converter` | Full `Convert()` on real `testdata/` PDFs | Build tag `//go:build integration` | ⚙️ CI only |
| **Integration** | `cli` | End-to-end binary invocation | `os/exec` running built binary | ⚙️ CI only |
| **Smoke** | `cli` | `make run` against `testdata/` | Makefile target | 🖐 Manual |

### Test Naming Convention

Test names must describe the behavior being verified, not the implementation:

```go
// ✅ GOOD: Describes behavior
func TestCleanExtractedText_ParagraphBreaksOnEmptyLines(t *testing.T)
func TestCoalesceChars_SplitsDistantChars(t *testing.T)
func TestConverter_Convert_NoPDF(t *testing.T)

// ❌ BAD: Describes implementation
func TestFunction1(t *testing.T)
func TestThatRegexWorks(t *testing.T)
```

### Table-Driven Test Pattern

All unit tests with multiple cases must use the table-driven pattern:

```go
func TestSomething(t *testing.T) {
    tests := []struct {
        name string
        in   SomeInput
        want SomeOutput
    }{
        {name: "happy path", in: ..., want: ...},
        {name: "empty input", in: ..., want: ...},
        {name: "edge case", in: ..., want: ...},
    }
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            got := FunctionUnderTest(tt.in)
            if got != tt.want {
                t.Errorf("FunctionUnderTest() = %v, want %v", got, tt.want)
            }
        })
    }
}
```

### Test Coverage Targets

| Package | Current | Target | Notes |
|---------|---------|--------|-------|
| `converter` (core) | ~70% | **≥ 85%** | Every exported function and every error path |
| `converter` (tables) | ~80% | **≥ 85%** | All heuristic branches including edge cases |
| `discovery` | ~90% | **≥ 90%** | Including permission errors |
| `tui` | 0% | **N/A** | Visual rendering — not unit-testable. Validated via smoke tests. |
| `cli` | 0% | **≥ 50%** | Integration tests via binary execution |
| **Overall** | ~60% | **≥ 80%** | Enforced in CI via coverage threshold |

### Coverage Enforcement in CI

```yaml
# .github/workflows/ci.yml
- name: Test with coverage
  run: |
    go test -race -coverprofile=coverage.out -covermode=atomic ./...
    COVERAGE=$(go tool cover -func=coverage.out | grep total | awk '{print $3}' | tr -d '%')
    echo "Coverage: ${COVERAGE}%"
    if (( $(echo "$COVERAGE < 80" | bc -l) )); then
      echo "::error::Coverage ${COVERAGE}% is below the 80% threshold"
      exit 1
    fi
```

### Fixture Management (`testdata/`)

- **Real PDFs** in `testdata/` are committed to the repo for integration tests.
- Fixtures must not exceed **1 MB each** to keep the repo lean.
- Each fixture should test a specific extraction scenario (tables, mixed content, image-only, malformed).
- Planned fixtures:

| Fixture | Purpose | Tests |
|---------|---------|-------|
| `testdata/text-only.pdf` | Clean text extraction | `cleanExtractedText`, `Convert` |
| `testdata/table-3col.pdf` | 3-column table detection | `detectColumnPositions`, `renderRowsAsMarkdown` |
| `testdata/scanned-image.pdf` | Image-only pages (OCR required) | `AnalyzePDF`, graceful skip |
| `testdata/mixed-content.pdf` | Tables + paragraphs + images | Full pipeline integration |
| `testdata/malformed.pdf` | Corrupt/truncated PDF | Panic recovery, error handling |

### No Mocks Policy

The project avoids mock frameworks. Instead:
- **`converter`** and **`discovery`** operate on real temp files (`t.TempDir()`).
- **`tables.go`** functions accept concrete types (`[]pdf.Text`, `[]word`, `[]tableRow`), making them naturally testable with synthetic data.
- **TUI** is not unit-tested — it is validated visually via `make run`.
- If a future interface boundary requires test doubles, prefer hand-written fakes over mock libraries.

### Race Detection

All CI test runs include `-race`:

```bash
go test -race ./...
```

This is non-negotiable. The worker pool uses goroutines + channels, and even benign races can cause intermittent failures or data corruption. The race detector catches:
- Unsynchronized access to shared variables.
- Channel misuse patterns.
- Goroutine leaks from unconsumed results.

---

## Adding a New Feature

### New Extraction Capability (e.g., Image Extraction)

1. Create a new file in `internal/converter/` (e.g., `images.go`).
2. Define functions that accept concrete types and return `(result, error)`.
3. Add configuration to the `Options` struct — never add global variables.
4. Wire the capability into `Convert()` conditionally based on `Options`.
5. Write unit tests in `images_test.go` using synthetic data.
6. Update `cli/convert.go` to pass the new flag to `Options`.
7. Update this document and `CLAUDE.md`.

### New Input Format (e.g., `.docx`)

1. Add a `Discover<Format>` function in `internal/discovery/` or extend `FindPDFs` to accept a `[]string` of extensions.
2. Implement a `Convert<Format>` method on `Converter` (or a new struct) returning `Result`.
3. Wire a format-detection branch in `cli/convert.go` based on file extension.
4. Add unit tests with fixture files in `testdata/`.

### New CLI Flag

1. Define the flag variable in `cli/root.go` (persistent) or `cli/convert.go` (command-specific).
2. Map the flag to the appropriate `Options` field at the call site.
3. Never read flag variables directly in `converter` or `discovery`.

### New TUI Status (e.g., `StatusIgnored`)

1. Add the status to the `Status` enum in `converter`.
2. Return it from the appropriate converter function.
3. In `cli/convert.go`, match on the status in the results aggregation loop.
4. In `tui/progress.go`, add a display row for the new status.
