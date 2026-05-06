# pdf2md-tui — Production-Grade Implementation Plan

A high-performance Go CLI utility for batch-converting PDFs to clean, **LLM-friendly Markdown** — optimized for token efficiency when feeding documents into AI/LLM pipelines.

---

## Why LLM-Optimized Markdown?

> [!NOTE]
> PDFs are among the worst formats for LLM consumption. Binary encoding, embedded fonts, layout metadata, headers/footers, and page-break artifacts all waste tokens and degrade context quality. A 50-page PDF can burn 3-5x more tokens than the equivalent clean Markdown.

**pdf2md-tui** solves this by extracting **only the semantic text content** from PDFs and outputting minimal, clean Markdown that:

- **Eliminates layout noise** — no page numbers, headers/footers, or formatting artifacts
- **Maximizes token density** — pure text content means every token carries meaning
- **Preserves structure** — headings, paragraphs, and lists remain intact
- **Batch-ready** — convert entire documentation directories in seconds for RAG pipelines, fine-tuning datasets, or context window stuffing
- **Date-stamped outputs** — track which version of a document was processed (`report_2026-05-06.md`)

**Target use cases:**
- Preparing corporate documents for RAG (Retrieval-Augmented Generation) ingestion
- Building LLM training/fine-tuning datasets from PDF archives
- Converting research papers for AI-assisted literature review
- Preprocessing legal/financial documents for AI analysis
- Reducing token costs when processing large document sets via API calls

---

## User Review Required

> [!IMPORTANT]
> **GitHub Username / Org**: The Homebrew tap requires your GitHub username (e.g., `nawodyaishan`). Please confirm.

> [!NOTE]
> **PDF Library — DECIDED**: Using [`pdfcpu/pdfcpu`](https://github.com/pdfcpu/pdfcpu) (8.5K stars, Apache-2.0, comprehensive PDF processing). Chosen for its mature API, active maintenance, pure Go implementation (no CGO), and rich feature set including text extraction, page manipulation, and metadata access.

> [!IMPORTANT]
> **Cobra CLI Framework**: The original spec uses raw `os.Args`. This plan upgrades to `github.com/spf13/cobra` for proper subcommand support (`convert`, `version`), flags (`--recursive`, `--output`, `--workers`), and auto-generated help/completions. Confirm this is acceptable.

## Open Questions

1. **Recursive scanning** — Should the tool support scanning subdirectories (`--recursive` flag), or stay flat as in the original spec?
2. **Output format control** — Should we add options for header/footer stripping, page separators, or just plain text extraction?
3. **LLM-specific formatting** — Should we add a `--strip-noise` flag that aggressively removes boilerplate (page numbers, repeated headers/footers) to further reduce token waste?
4. **License** — What license should be used? (MIT recommended for wide distribution)
5. **Minimum Go version** — Go 1.21+ as specified, or bump to 1.22+ for better stdlib features?

---

## Proposed Changes

### Phase 1: Project Foundation

---

#### [NEW] Project Structure

```
pdf2md-tui/
├── cmd/
│   └── pdf2md-tui/
│       └── main.go              # Entry point, minimal — delegates to root command
├── internal/
│   ├── cli/
│   │   ├── root.go              # Cobra root command + global flags
│   │   ├── convert.go           # Convert subcommand (default action)
│   │   └── version.go           # Version subcommand with build info
│   ├── converter/
│   │   ├── converter.go         # Core PDF→MD conversion logic (uses pdfcpu)
│   │   └── converter_test.go    # Unit tests for converter
│   ├── discovery/
│   │   ├── discovery.go         # PDF file discovery & manifest building
│   │   └── discovery_test.go    # Unit tests for discovery
│   └── tui/
│       └── progress.go          # pterm progress bar wrapper
├── pkg/
│   └── version/
│       └── version.go           # Version variables for ldflags injection
├── testdata/
│   ├── sample.pdf               # Test PDF for integration tests
│   └── empty.pdf                # Edge case: empty PDF
├── docs/
│   └── implementation_plan.md   # This document
├── .github/
│   └── workflows/
│       ├── ci.yml               # Lint, test, build on PRs
│       └── release.yml          # GoReleaser + Homebrew tap on tags
├── .goreleaser.yml              # GoReleaser v2 config
├── Makefile                     # Dev commands (build, test, lint, snapshot)
├── go.mod
├── go.sum
├── README.md
├── LICENSE
└── .gitignore
```

---

#### [NEW] [go.mod](file:///Users/nawodyaishan/Documents/GitHub/pdf2md-tui/go.mod)

Initialize Go module with dependencies:

```
module github.com/nawodyaishan/pdf2md-tui

go 1.22

require (
    github.com/pdfcpu/pdfcpu v0.9.x
    github.com/pterm/pterm v0.12.x
    github.com/spf13/cobra v1.8.x
)
```

---

#### [NEW] [pkg/version/version.go](file:///Users/nawodyaishan/Documents/GitHub/pdf2md-tui/pkg/version/version.go)

Version injection via `ldflags` — GoReleaser auto-populates these at build time:

```go
package version

var (
    Version   = "dev"
    Commit    = "none"
    Date      = "unknown"
    GoVersion = "unknown"
)
```

GoReleaser injects via:
```yaml
ldflags:
  - -s -w
  - -X github.com/nawodyaishan/pdf2md-tui/pkg/version.Version={{.Version}}
  - -X github.com/nawodyaishan/pdf2md-tui/pkg/version.Commit={{.ShortCommit}}
  - -X github.com/nawodyaishan/pdf2md-tui/pkg/version.Date={{.Date}}
  - -X github.com/nawodyaishan/pdf2md-tui/pkg/version.GoVersion={{.Env.GOVERSION}}
```

---

#### [NEW] [cmd/pdf2md-tui/main.go](file:///Users/nawodyaishan/Documents/GitHub/pdf2md-tui/cmd/pdf2md-tui/main.go)

Minimal entry point:

```go
package main

import (
    "os"
    "github.com/nawodyaishan/pdf2md-tui/internal/cli"
)

func main() {
    if err := cli.Execute(); err != nil {
        os.Exit(1)
    }
}
```

---

### Phase 2: Core Logic

---

#### [NEW] [internal/cli/root.go](file:///Users/nawodyaishan/Documents/GitHub/pdf2md-tui/internal/cli/root.go)

Cobra root command with global flags:

| Flag | Short | Type | Default | Description |
|------|-------|------|---------|-------------|
| `--output` | `-o` | string | `md/` | Output subdirectory name |
| `--recursive` | `-r` | bool | `false` | Scan subdirectories |
| `--workers` | `-w` | int | `runtime.NumCPU()` | Concurrent conversion workers |
| `--date-format` | | string | `2006-01-02` | Date suffix format |
| `--verbose` | `-v` | bool | `false` | Verbose logging |
| `--strip-noise` | | bool | `false` | Aggressively remove boilerplate for LLM optimization |

Default behavior: If no subcommand is given, `convert` runs as the default.

---

#### [NEW] [internal/cli/convert.go](file:///Users/nawodyaishan/Documents/GitHub/pdf2md-tui/internal/cli/convert.go)

The `convert` subcommand orchestrates the pipeline:

1. **Validate** directory argument
2. **Discover** PDFs via `discovery.FindPDFs()`
3. **Scaffold** output directory idempotently
4. **Initialize** pterm progress bar
5. **Fan-out** conversion to a worker pool (bounded by `--workers`)
6. **Collect** results and print summary (including token savings estimate)

Key improvement over original: **Worker pool pattern** using goroutines + channels for concurrent PDF processing. Large directories with hundreds of PDFs benefit significantly.

```go
// Worker pool pattern
jobs := make(chan string, len(pdfFiles))
results := make(chan Result, len(pdfFiles))

for w := 0; w < workers; w++ {
    go worker(jobs, results, outDir, currentDate)
}

for _, f := range pdfFiles {
    jobs <- f
}
close(jobs)
```

**Token savings summary** — after conversion, the tool prints:
```
📊 Token Efficiency Report
   PDF source size:   12.4 MB (est. ~3.1M tokens as raw PDF)
   Markdown output:    1.8 MB (est. ~450K tokens as clean MD)
   Token savings:     ~85% reduction
```

*(Token estimates based on ~4 chars/token heuristic for English text)*

---

#### [NEW] [internal/cli/version.go](file:///Users/nawodyaishan/Documents/GitHub/pdf2md-tui/internal/cli/version.go)

Version command displaying build metadata:

```
pdf2md-tui v1.2.0
  commit:  abc1234
  built:   2026-05-06T11:00:00Z
  go:      go1.22.3
```

---

#### [NEW] [internal/converter/converter.go](file:///Users/nawodyaishan/Documents/GitHub/pdf2md-tui/internal/converter/converter.go)

Core conversion logic, isolated from CLI concerns:

```go
type Converter struct {
    DateFormat string
    StripNoise bool  // LLM optimization: remove boilerplate
}

type Result struct {
    InputPath    string
    OutputPath   string
    InputBytes   int64   // Original PDF size
    OutputBytes  int64   // Clean MD size
    Duration     time.Duration
    Err          error
}

func (c *Converter) Convert(pdfPath, outDir string) Result { ... }
```

Key improvements:
- **Structured results** — not just success/fail, but metrics including size reduction
- **Explicit error wrapping** — `fmt.Errorf("extracting %s: %w", path, err)` for debugging
- **Memory-efficient** — uses `bytes.Buffer` with pooling for high-throughput scenarios
- **LLM-optimized output** — when `StripNoise` is enabled:
  - Collapses excessive whitespace and blank lines
  - Removes repeated headers/footers (detected via page-boundary heuristics)
  - Strips page numbers and form-feed characters
  - Normalizes Unicode to ASCII where possible (reduces token fragmentation)
- **Markdown header injection** — optionally prepends `# Filename` and source metadata as a YAML frontmatter block

---

#### [NEW] [internal/discovery/discovery.go](file:///Users/nawodyaishan/Documents/GitHub/pdf2md-tui/internal/discovery/discovery.go)

PDF discovery with optional recursion:

```go
func FindPDFs(dir string, recursive bool) ([]string, error) {
    if recursive {
        return filepath.WalkDir(...)
    }
    return os.ReadDir(...) // flat scan
}
```

---

#### [NEW] [internal/tui/progress.go](file:///Users/nawodyaishan/Documents/GitHub/pdf2md-tui/internal/tui/progress.go)

Pterm wrapper with:
- Animated spinner during discovery phase
- Progress bar during conversion phase
- Color-coded summary table at completion (including token savings)
- Graceful fallback for non-TTY environments (CI pipelines)

---

### Phase 3: Build, Test & Quality

---

#### [NEW] [Makefile](file:///Users/nawodyaishan/Documents/GitHub/pdf2md-tui/Makefile)

```makefile
.PHONY: build test lint snapshot release clean

APP_NAME    := pdf2md-tui
VERSION     := $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
GOVERSION   := $(shell go version | awk '{print $$3}')
LDFLAGS     := -s -w \
  -X github.com/nawodyaishan/pdf2md-tui/pkg/version.Version=$(VERSION) \
  -X github.com/nawodyaishan/pdf2md-tui/pkg/version.Commit=$(shell git rev-parse --short HEAD) \
  -X github.com/nawodyaishan/pdf2md-tui/pkg/version.Date=$(shell date -u +%Y-%m-%dT%H:%M:%SZ) \
  -X github.com/nawodyaishan/pdf2md-tui/pkg/version.GoVersion=$(GOVERSION)

build:
	go build -ldflags="$(LDFLAGS)" -o bin/$(APP_NAME) ./cmd/pdf2md-tui

test:
	go test -race -coverprofile=coverage.out ./...

lint:
	golangci-lint run ./...

snapshot:
	goreleaser release --snapshot --clean

clean:
	rm -rf bin/ dist/ coverage.out
```

---

#### [NEW] Unit Tests

- `internal/converter/converter_test.go` — Table-driven tests with sample PDFs
- `internal/discovery/discovery_test.go` — Flat vs recursive, empty dirs, permission errors
- Edge cases: encrypted PDFs, zero-page PDFs, non-UTF8 content

---

### Phase 4: Distribution — GoReleaser + Homebrew

---

#### [NEW] [.goreleaser.yml](file:///Users/nawodyaishan/Documents/GitHub/pdf2md-tui/.goreleaser.yml)

```yaml
version: 2

project_name: pdf2md-tui

before:
  hooks:
    - go mod tidy
    - go generate ./...

builds:
  - id: pdf2md-tui
    main: ./cmd/pdf2md-tui
    binary: pdf2md-tui
    env:
      - CGO_ENABLED=0
    goos:
      - linux
      - darwin
      - windows
    goarch:
      - amd64
      - arm64
    ldflags:
      - -s -w
      - -X github.com/nawodyaishan/pdf2md-tui/pkg/version.Version={{.Version}}
      - -X github.com/nawodyaishan/pdf2md-tui/pkg/version.Commit={{.ShortCommit}}
      - -X github.com/nawodyaishan/pdf2md-tui/pkg/version.Date={{.Date}}

archives:
  - id: default
    format: tar.gz
    format_overrides:
      - goos: windows
        format: zip
    name_template: "{{ .ProjectName }}_{{ .Version }}_{{ .Os }}_{{ .Arch }}"

checksum:
  name_template: 'checksums.txt'

changelog:
  sort: asc
  filters:
    exclude:
      - '^docs:'
      - '^test:'
      - '^ci:'
      - '^chore:'

# Homebrew Tap Formula
brews:
  - repository:
      owner: nawodyaishan
      name: homebrew-tap
      token: "{{ .Env.HOMEBREW_TAP_TOKEN }}"
    directory: Formula
    homepage: "https://github.com/nawodyaishan/pdf2md-tui"
    description: "High-performance TUI tool for batch PDF to LLM-friendly Markdown conversion"
    license: "MIT"
    test: |
      system "#{bin}/pdf2md-tui", "version"
    install: |
      bin.install "pdf2md-tui"

# nFPM for deb/rpm packages (bonus)
nfpms:
  - id: packages
    package_name: pdf2md-tui
    vendor: nawodyaishan
    homepage: https://github.com/nawodyaishan/pdf2md-tui
    description: "Batch PDF to LLM-optimized Markdown converter with TUI progress"
    license: MIT
    formats:
      - deb
      - rpm
```

---

#### [NEW] [homebrew-tap repository](https://github.com/nawodyaishan/homebrew-tap)

You need to create a separate GitHub repo: `nawodyaishan/homebrew-tap`. GoReleaser auto-pushes the formula there on release.

**User installation:**
```bash
brew tap nawodyaishan/tap
brew install pdf2md-tui
```

**Required secrets in `pdf2md-tui` repo:**
- `HOMEBREW_TAP_TOKEN` — PAT with `repo` scope for the tap repo

---

### Phase 5: Future Work — npm Binary Wrapper

> [!NOTE]
> **npm distribution is deferred to a future release.** The esbuild-style platform-specific package pattern (JS shim + `optionalDependencies` per OS/arch) is well-researched and documented here for when we're ready to add it. For now, users install via **Homebrew**, **`go install`**, or **GitHub Releases direct download**.

When implemented, the npm distribution will follow the pattern used by esbuild, turbo, and notion-cli:
- Main wrapper package with JS shim (`@nawodyaishan/pdf2md-tui`)
- Platform-specific optional dependency packages (darwin-arm64, darwin-x64, linux-x64, linux-arm64, win32-x64)
- Each platform `package.json` declares `os` and `cpu` fields for automatic platform matching
- Fallback download from GitHub Releases if optional dep fails

---

### Phase 5b: CI/CD Pipeline

---

#### [NEW] [.github/workflows/ci.yml](file:///Users/nawodyaishan/Documents/GitHub/pdf2md-tui/.github/workflows/ci.yml)

Triggered on every PR and push to `main`:

```yaml
name: CI
on:
  push:
    branches: [main]
  pull_request:

jobs:
  lint:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with: { go-version: '1.22' }
      - uses: golangci/golangci-lint-action@v6

  test:
    runs-on: ${{ matrix.os }}
    strategy:
      matrix:
        os: [ubuntu-latest, macos-latest, windows-latest]
        go: ['1.22']
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with: { go-version: ${{ matrix.go }} }
      - run: go test -race -coverprofile=coverage.out ./...
      - uses: codecov/codecov-action@v4
        if: matrix.os == 'ubuntu-latest'

  build:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with: { go-version: '1.22' }
      - run: go build -o /dev/null ./cmd/pdf2md-tui
```

---

#### [NEW] [.github/workflows/release.yml](file:///Users/nawodyaishan/Documents/GitHub/pdf2md-tui/.github/workflows/release.yml)

Triggered on version tags (`v*`):

```yaml
name: Release
on:
  push:
    tags: ['v*']

permissions:
  contents: write
  id-token: write

jobs:
  goreleaser:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
        with: { fetch-depth: 0 }
      - uses: actions/setup-go@v5
        with: { go-version: '1.22' }
      - uses: goreleaser/goreleaser-action@v6
        with:
          version: '~> v2'
          args: release --clean
        env:
          GITHUB_TOKEN: ${{ secrets.GITHUB_TOKEN }}
          HOMEBREW_TAP_TOKEN: ${{ secrets.HOMEBREW_TAP_TOKEN }}
```

---

### Phase 6: Documentation & Polish

---

#### [NEW] [README.md](file:///Users/nawodyaishan/Documents/GitHub/pdf2md-tui/README.md)

Professional README with:
- **LLM optimization value proposition** front and center
- Token savings benchmarks (before/after table)
- Animated GIF/SVG demo (captured via `vhs` or `asciinema`)
- Installation methods: Homebrew, `go install`, binary download
- Usage examples and flag reference
- Architecture diagram
- Contributing guide

#### [NEW] [LICENSE](file:///Users/nawodyaishan/Documents/GitHub/pdf2md-tui/LICENSE)

MIT License

---

## Key Improvements Over Original Spec

| Area | Original | Improved |
|------|----------|----------|
| **Purpose** | Generic PDF→MD | LLM-optimized, token-efficient output |
| **CLI Framework** | Raw `os.Args` | Cobra with subcommands, flags, completions |
| **Concurrency** | Sequential loop | Worker pool with configurable parallelism |
| **Error Handling** | Print & continue | Structured `Result` type with error wrapping |
| **Version Info** | None | `ldflags` injection via GoReleaser |
| **Distribution** | Manual `go build` | Homebrew tap + GitHub Releases + deb/rpm (npm future) |
| **CI/CD** | None | GitHub Actions (lint, test, cross-platform, release) |
| **Testing** | None | Table-driven unit tests + integration tests |
| **Project Structure** | Single `main.go` | Clean architecture (`cmd/`, `internal/`, `pkg/`) |
| **TUI** | Basic progress bar | Spinner + progress bar + summary table + non-TTY fallback |
| **Output Control** | Hardcoded `md/` | Configurable `--output` flag |
| **LLM Features** | None | `--strip-noise`, token savings report, clean output |

---

## Verification Plan

### Automated Tests
```bash
# Unit tests with race detection
go test -race -coverprofile=coverage.out ./...

# Lint
golangci-lint run ./...

# Local snapshot release (no publish)
goreleaser release --snapshot --clean

# Verify built binary
./dist/pdf2md-tui_linux_amd64_v1/pdf2md-tui version
./dist/pdf2md-tui_linux_amd64_v1/pdf2md-tui ./testdata/
```

### Manual Verification
1. Run the built binary against a directory containing 10+ PDFs of varying sizes
2. Verify progress bar renders correctly in both TTY and piped output
3. Verify token savings report accuracy against manual `tiktoken` counts
4. Test Homebrew installation: `brew tap nawodyaishan/tap && brew install pdf2md-tui`
5. Test `go install github.com/nawodyaishan/pdf2md-tui/cmd/pdf2md-tui@latest`
6. Cross-platform smoke test on macOS (arm64), Linux (amd64), Windows (amd64)

### Browser Tests
- Verify GitHub Release page shows all expected artifacts
- Verify Homebrew tap repo has correct formula
