# pdf2md-tui

> Fast local PDF ingestion for RAG/VLM pipelines.
> Batch-convert PDFs into structured Markdown plus extracted images, with a TUI for humans and JSON output for automation.

No cloud API, no GPU, no Python environment — just a single Go binary.

[![CI](https://github.com/nawodyaishan/pdf2md-tui/actions/workflows/ci.yml/badge.svg)](https://github.com/nawodyaishan/pdf2md-tui/actions/workflows/ci.yml)
[![Go Version](https://img.shields.io/github/go-mod/go-version/nawodyaishan/pdf2md-tui)](go.mod)
[![Latest Release](https://img.shields.io/github/v/release/nawodyaishan/pdf2md-tui)](https://github.com/nawodyaishan/pdf2md-tui/releases/latest)
[![License: MIT](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)

---

<div align="center">
  <img src="assets/banner.jpeg" alt="pdf2md-tui banner" width="600" />
</div>

## Why?

**PDFs break AI pipelines.** Binary encoding, embedded fonts, and layout metadata inflate token counts — but the real damage is structural. Legacy parsers flatten documents into raw text, destroying tables, orphaning image references, and producing vector embeddings that are functionally useless for retrieval.

`pdf2md-tui` turns folders of digital PDFs into a filesystem-friendly output contract: **structured Markdown** plus a local `./images/` directory. That gives downstream chunkers, vector stores, LlamaIndex/LangChain workflows, and VLM agents cleaner context to work with.

| Document | Naive VLM / image processing (est. token cost) | Clean Markdown (est. token cost) | Context Quality |
|---|---|---|---|
| 50-page technical spec | ~42,500 tokens | ~15,000 tokens | **Cleaner, structured** |
| 200-page legal contract | ~170,000 tokens | ~60,000 tokens | **Cleaner, structured** |
| Research paper (12 pages) | ~10,200 tokens | ~3,500 tokens | **Cleaner, structured** |

*Estimates illustrate the common cost gap between image-heavy ingestion and text-first ingestion. Actual savings depend on document density, extraction quality, and provider pricing.*

**Built for the "Look Twice" methodology** — extract text + isolate images at ingestion time (Phase 1), then let your downstream VLM pipeline handle deep visual reasoning at retrieval time (Phase 2).

`pdf2md-tui` is intentionally narrow: it does not try to be a full document intelligence platform. It focuses on fast local batch conversion of digital PDFs into Markdown plus image assets, with predictable files that downstream tools can index.

**Use it when:**

- You have folders of digital PDFs to prepare for RAG.
- You want local Markdown and image files without a cloud parser.
- You need a CLI/TUI that can run in scripts and terminals.
- You prefer a single static binary over a Python/ML stack.

**Use a heavier parser when:**

- You need OCR-heavy scanned document handling.
- You need advanced formula, chart, or table understanding.
- You need bounding boxes, semantic element JSON, or enterprise connectors.
- You need maximum extraction accuracy over local simplicity.

---

## Features

- **Table-aware Markdown** — detects column-aligned rows and emits GFM pipe tables; more robust chunk-preserving table blocks are planned
- **Two-path extraction** — positional extraction first (preserves structure); falls back to plain-text for edge cases
- **Worker pool** — concurrent conversion using `runtime.NumCPU()` workers by default; configurable via `--workers`
- **Live TUI** — Bubble Tea dashboard for batch conversion with live stats, recent activity, and a completion menu
- **Interactive Menu** — run `pdf2md-tui` without arguments in a terminal to launch a guided configuration wizard
- **Graceful OCR detection** — scanned/image-only PDFs are detected and skipped cleanly, reported in the summary (no empty output files)
- **`--strip-noise`** — aggressively removes page numbers, repeated headers/footers, and excess whitespace for maximum token density
- **Smart Overwrites** — prompts before overwrite in a terminal; non-interactive and `--quiet` runs fail unless `--force` is set
- **Automation-friendly modes** — `--quiet` emits a JSON summary; non-TTY non-quiet runs fall back to a plain-text summary without alt-screen UI
- **Date-stamped outputs** — `report_2026-05-06.md` so you always know which version was processed
- **Single static binary** — no runtime dependencies; `CGO_ENABLED=0`, pure Go

---

## 📦 Installation


### Homebrew (macOS and Linux)

```bash
brew tap nawodyaishan/tap
brew install pdf2md-tui
```

If you previously installed the Cask-based package, migrate once:

```bash
brew uninstall --cask pdf2md-tui
brew install pdf2md-tui
```

### Go install

```bash
go install github.com/nawodyaishan/pdf2md-tui/cmd/pdf2md-tui@latest
```

### Binary download

Download a pre-built binary for your platform from the [latest release](https://github.com/nawodyaishan/pdf2md-tui/releases/latest).

Platforms: Linux, macOS, Windows × amd64 / arm64. Packages: `.tar.gz`, `.zip`, `.deb`, `.rpm`.

---

## Development

### Prerequisites

Before cloning and developing, install the required tools:

**Required:**
- **Go 1.26.3+** — [Install](https://go.dev/dl)
- **Lefthook** — Git hooks runner for code quality checks
  ```bash
  # macOS / Linux
  brew install lefthook
  
  # Or via Go
  go install github.com/evilmartians/lefthook/v2@v2.1.6
  ```

**Recommended:**
- **golangci-lint** — Unified linter (optional if you only run tests)
  ```bash
  # macOS / Linux
  brew install golangci-lint
  
  # Or via Go
  go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
  ```

**Why Lefthook?** It prevents commits that fail linting, formatting, or tests, catching issues before they reach CI.

> **Note:** If lefthook isn't installed, commits will be blocked with a helpful error message. This is intentional and protects code quality.

### Setup

After cloning, run the setup script (recommended):

```bash
bash scripts/setup-dev.sh
```

Or manually:

```bash
# Install git hooks (runs on pre-commit and pre-push)
make hooks-install

# Or directly
lefthook install
```

This will activate automated checks:
- **pre-commit**: `gofmt`, `go vet`, `golangci-lint` on staged files
- **pre-push**: full test suite + build validation

> **Note:** Commits will be **blocked** if lefthook is not installed. This protects code quality by ensuring all commits pass linting before reaching CI.

### Commands

| Command | Purpose |
|---------|---------|
| `make test` | Run full test suite with race detector and coverage |
| `make lint` | Run golangci-lint (requires lefthook/golangci-lint) |
| `make vet` | Run go vet analysis |
| `make fmt` | Format all Go files |
| `make build` | Build binary to `bin/pdf2md-tui` |
| `make cover` | Open HTML coverage report |

### Test Coverage

Coverage baseline measured on clean checkout (`ubuntu-latest`):
- **Current**: 47.9% (verified 2026-05-10)
- **CI gate**: 45%+ required
- **Target**: 70%+ (phased roadmap in [docs/COVERAGE.md](docs/COVERAGE.md))

Measure coverage locally:
```bash
go test -race -coverprofile=coverage.out -covermode=atomic ./...
go tool cover -func=coverage.out | tail -1
make cover  # Opens HTML report in browser
```

See [docs/COVERAGE.md](docs/COVERAGE.md) for the complete coverage roadmap and policy.

---

## Usage

### Interactive Mode

Run the tool without arguments in a terminal to launch the guided interactive wizard. It prompts for the target directory and common flags, and currently preselects `Extract images` in the multiselect:

```bash
pdf2md-tui
```

The zero-argument menu is only used when both stdin and stdout are interactive terminals. Otherwise the command defaults to converting the current directory.

### CLI Mode

```bash
# Convert all PDFs in ./docs, write Markdown to ./docs/md/
pdf2md-tui convert ./docs

# Recurse into subdirectories and strip layout noise for LLM ingestion
pdf2md-tui convert ./archive --recursive --strip-noise

# Use 8 workers, custom output directory, no date suffix, and force overwrite existing files
pdf2md-tui convert ./papers --workers 8 --output out --date-format none --force

# Extract embedded images and inject markdown links into the output
pdf2md-tui convert ./docs --extract-images

# CI-friendly mode: emit only JSON to stdout
pdf2md-tui convert ./docs --quiet

# Print version and build info
pdf2md-tui version
```

### Go Library Usage

Since the refactoring to Clean Architecture, you can embed the conversion engine directly into your own Go applications.

> [!NOTE]
> The core logic resides in the `pkg/` directory, making it importable as a standard Go module.

```go
import (
	"github.com/nawodyaishan/pdf2md-tui/pkg/domain"
	"github.com/nawodyaishan/pdf2md-tui/pkg/repository/pdf"
	"github.com/nawodyaishan/pdf2md-tui/pkg/repository/storage"
	"github.com/nawodyaishan/pdf2md-tui/pkg/service"
)

func main() {
	// 1. Initialize configuration
	cfg := domain.NewConfig()
	cfg.ExtractImages = true
	cfg.StripNoise = true

	// 2. Initialize dependencies (Clean Architecture)
	store := storage.NewStorage()
	parser := pdf.NewParser()

	// 3. Initialize the service
	conv := service.NewConverterService(cfg, store, parser)

	// 4. Run conversion
	// Convert(pdfPath, outDir) returns a domain.Result
	res := conv.Convert("input.pdf", "output_dir")

	if res.Err != nil {
		fmt.Printf("Conversion failed: %v\n", res.Err)
		return
	}

	fmt.Printf("Successfully converted %s to %s (Saved %d bytes)\n", 
		res.InputPath, res.OutputPath, res.InputBytes - res.OutputBytes)
}
```


### Flags

| Flag | Short | Default | Description |
|------|-------|---------|-------------|
| `--output` | `-o` | `md/` | Output subdirectory (relative to the target directory) |
| `--recursive` | `-r` | `false` | Scan subdirectories |
| `--workers` | `-w` | `NumCPU` | Number of concurrent conversion workers |
| `--date-format` | | `2006-01-02` | Date suffix format (Go reference time); `none` disables the suffix |
| `--force` | `-f` | `false` | Overwrite existing output files without prompting |
| `--strip-noise` | | `false` | Aggressively remove page numbers, headers/footers, and excess whitespace |
| `--extract-images` | | `false` | Extract embedded images and inject markdown references |
| `--quiet` | `-q` | `false` | Suppress the TUI and emit a JSON summary to stdout |
| `--verbose` | `-v` | `false` | Print per-file errors to stderr |
| `--log-file` | | `pdf2md.log` | Path for detailed conversion logs |

### Runtime Modes

- Interactive terminal: shows the pterm banner/discovery phase, then launches the Bubble Tea dashboard during conversion. When the batch completes, the dashboard offers `Open Output Directory`, `View Detailed Log` when failures occurred, and `Exit`.
- Non-interactive without `--quiet`: skips the Bubble Tea alt-screen and prints a concise text summary after conversion finishes.
- `--quiet`: prints only the JSON summary to stdout. If any conversions fail, the command still exits non-zero.

### Overwrite and Log Behavior

- Existing output files require confirmation only in an interactive terminal.
- Non-interactive runs and `--quiet` refuse to overwrite existing outputs unless `--force` is set.
- When failures occur, inspect the configured `--log-file` path for details. The completion menu exposes `View Detailed Log` in the dashboard, and non-interactive summaries print the log path directly.

### Output structure

```
./docs/
├── report.pdf
├── spec.pdf
└── md/
    ├── images/
    │   ├── report/
    │   │   ├── report_1_5.png
    │   │   └── report_2_11.jpg
    │   └── spec/
    │       └── spec_1_8.png
    ├── report_2026-05-06.md
    └── spec_2026-05-06.md
```

---

## How it works

1. **Discovery** — scans the target directory (optionally recursive) for `.pdf` files.
2. **Worker pool** — distributes files across `N` goroutines via a buffered channel.
3. **Extraction** — for each page, attempts positional character extraction to preserve table structure; falls back to plain-text if the page is image-only or yields no content.
4. **Table detection** — identifies column-aligned rows (≥3 columns, ≥50 pt gaps, appearing in ≥40% of rows) and renders them as GFM pipe tables.
5. **Output** — writes one `.md` file per PDF to the output directory.

See [docs/ARCHITECTURE.md](docs/ARCHITECTURE.md) for a detailed breakdown of the extraction pipeline and concurrency model.

---

## Recent Quality Improvements

The core extraction engine has moved beyond simple text scraping toward more reliable Markdown reconstruction for RAG workflows.

### Advanced Extraction Engine (v1.2.7+)

- **Adaptive spacing heuristics** — uses line-level gap statistics to coalesce intentionally tracked headings, such as `D E F I N I T I V E`, while preserving natural word separation
- **Double-letter preservation** — avoids aggressive character deduplication so legitimate double letters, such as `across`, `ebooks`, and `www`, are preserved
- **Block-aware optimization** — scopes whitespace cleanup to individual blocks so paragraph boundaries and table structures are less likely to be damaged
- **Standardized Unicode** — applies `norm.NFKC` normalization for ligatures and private-use characters

### Automated QA Validation

The project tracks extraction regressions through the [QA Test Plan](docs/specs/qa_test_plan.md) and automated tests covering:

- **Content fidelity** — legitimate repeated characters should not collapse
- **Structural integrity** — optimized output should retain useful paragraph density
- **Semantic coherence** — styled headings and technical symbols should survive cleanup
- **Cleanliness** — replacement characters (`U+FFFD`) and obvious encoding garbage are treated as defects

---

## Roadmap Status

- [x] **🛡️ Graceful OCR Detection** — Detect and skip scanned PDFs without failing.
- [x] **🖼️ Image Extraction Pipeline** — Extract raw images for "Look Twice" VLM workflows.
- [x] **⚡ Zero-Arg Usability** — Run `pdf2md-tui` in any folder with no arguments.
- [x] **🏗️ Clean Architecture** — Decoupled domain/service/repository structure for scaling.
- [x] **🧱 Table-Aware Markdown** — Basic GFM pipe table support (v1.0). Robust indivisible blocks planned.
- [x] **🔇 CI-Friendly Quiet Mode** — Non-interactive JSON output for automation.
- [ ] **🔌 MCP Server Wrapper** — Native tool support for Model Context Protocol agents.
- [ ] **☁️ VLM Cloud Integration** — High-accuracy Markdown generation via GPT-4o/Claude.

See [ROADMAP.md](ROADMAP.md) for the full strategic vision.

---

## Building from source

```bash
git clone https://github.com/nawodyaishan/pdf2md-tui.git
cd pdf2md-tui
make build          # → bin/pdf2md-tui

make test           # run tests with race detector
make lint           # golangci-lint
make check          # fmt + vet + lint + test
make help           # list all targets
```

### Git hooks

This repository uses [Lefthook](https://lefthook.dev/) as a Go-friendly alternative to Husky.

The hook split follows the usual Go workflow:

- `pre-commit`: format staged Go files with `gofmt -w` and run `go vet ./...`
- `pre-push`: run `golangci-lint run ./...`, `go test -race ./...`, and `go build ./cmd/pdf2md-tui`

Install Lefthook and wire the hooks into `.git/hooks`:

```bash
brew install lefthook
make hooks-install
```

Official Go-based install is also supported:

```bash
go install github.com/evilmartians/lefthook/v2@v2.1.6
make hooks-install
```

You can also run the hook suites manually:

```bash
make hooks-run-pre-commit
make hooks-run-pre-push
```

---

## Roadmap

The roadmap is organized around maximizing **context quality** for downstream AI pipelines:

- **Near-term (v0.x)** — Image extraction pipeline (`pdfcpu`), graceful OCR detection, stronger table-aware Markdown output, `--quiet` JSON mode for CI/MCP
- **Mid-term (v1.x)** — "Look Twice" VLM pipeline (cloud vision providers), MCP server prototype, `.docx`/`.txt` ingestion
- **Long-term (v2.x)** — Pluggable post-processors, full MCP server + REST API

See [ROADMAP.md](ROADMAP.md) for the full vision, strategic goals, and how to contribute.

---

## Contributing

Bug reports and feature requests: open an issue using the provided templates.

For code contributions, see **[CONTRIBUTING.md](CONTRIBUTING.md)** for:
- Testing guidelines (tracked, ignored, and generated fixtures)
- Git hooks setup (lefthook pre-commit/pre-push)
- Code review checklist
- Conventional commits format

Quick checklist:

1. Check [ROADMAP.md](ROADMAP.md) and open issues for `help wanted` items.
2. Fork, branch, implement, add tests, and open a PR.
3. Install git hooks: `make hooks-install`
4. PRs must pass `make ci-local` (local CI simulation)
5. Coverage should not decrease (check `make cover`)

---

## License

MIT — see [LICENSE](LICENSE).
