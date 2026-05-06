# Architecture

`pdf2md-tui` is a Go CLI that converts PDF files to LLM-optimized Markdown. This document describes the internal structure, data flow, and key design decisions for contributors.

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
│   │   ├── tables.go      # Positional extraction: coalesceChars → groupIntoRows → renderRowsAsMarkdown
│   │   └── tables_test.go
│   ├── discovery/         # File system scanning
│   │   ├── discovery.go   # FindPDFs(dir, recursive) → []string
│   │   └── discovery_test.go
│   └── tui/               # Terminal UI wrappers
│       └── progress.go    # pterm: banner, spinner, progress bar, summary table
├── pkg/
│   └── version/           # Build-time version variables (injected via ldflags)
│       └── version.go
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
│  7. Launch worker pool               │
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
│  4. os.WriteFile → .md file          │
│  5. Return Result{metrics, error}    │
└──────────────────────────────────────┘
        │
        ▼
┌──────────────────────────────────────┐
│  internal/tui/progress.go            │
│  PrintSummary(in, out, dur, errors)  │
│  Shows token savings estimate        │
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

Token savings are estimated as `(inputBytes × 3) / outputBytes` — a rough proxy for the PDF overhead ratio — displayed as a percentage reduction.

When stdout is not a TTY, pterm auto-disables animations. The `--quiet` flag (planned) will suppress all TUI output for CI use.

---

## Key Dependencies

| Dependency | Purpose | Notes |
|-----------|---------|-------|
| `github.com/ledongthuc/pdf` | PDF parsing, glyph extraction | Pure Go, no CGO; chosen for positional text API (`Content()`) |
| `github.com/spf13/cobra` | CLI framework | Subcommands, flag binding, auto-help |
| `github.com/pterm/pterm` | TUI components | Spinner, progress bar, tables, styled output |

---

## Testing Strategy

- **Unit tests** in `converter_test.go`, `tables_test.go`, `discovery_test.go` use table-driven cases and `t.TempDir()` for filesystem isolation.
- Tests run with `-race` in CI to catch data races in the worker pool.
- No mocks: `converter` and `discovery` operate on real temp files. The TUI is not unit-tested (rendering is visual).
- Integration tests against `testdata/` PDFs are excluded from the default `go test ./...` run via build tags (planned).

---

## Adding a New Input Format

1. Add a `Discover<Format>` function in `internal/discovery/` or extend `FindPDFs` to accept a `[]string` of extensions.
2. Implement a `Convert<Format>` method on `Converter` (or a new struct) returning `Result`.
3. Wire a format-detection branch in `cli/convert.go` based on file extension.
4. Add unit tests with fixture files in `testdata/`.
