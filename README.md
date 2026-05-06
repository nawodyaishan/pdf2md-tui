# pdf2md-tui

> Batch-convert PDFs to LLM-optimized Markdown — Go CLI with a live TUI, worker pool concurrency, and table detection.

[![CI](https://github.com/nawodyaishan/pdf2md-tui/actions/workflows/ci.yml/badge.svg)](https://github.com/nawodyaishan/pdf2md-tui/actions/workflows/ci.yml)
[![Go Version](https://img.shields.io/github/go-mod/go-version/nawodyaishan/pdf2md-tui)](go.mod)
[![Latest Release](https://img.shields.io/github/v/release/nawodyaishan/pdf2md-tui)](https://github.com/nawodyaishan/pdf2md-tui/releases/latest)
[![License: MIT](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)

---

<div align="center">
  <img src="public/banner.jpeg" alt="pdf2md-tui banner" width="600" />
</div>

## Why?

**PDFs are among the worst formats for LLM consumption.** Binary encoding, embedded fonts, layout metadata, headers, footers, and page-break artifacts all inflate token counts and degrade context quality. A 50-page PDF can burn 3–5× more tokens than the equivalent clean Markdown.

`pdf2md-tui` extracts only the **semantic text content** and writes minimal, clean Markdown — every token carries meaning.

| Document | Raw PDF (est. tokens) | Clean Markdown (est. tokens) | Savings |
|---|---|---|---|
| 50-page technical spec | ~3.1M | ~450K | **~85%** |
| 200-page legal contract | ~12M | ~1.8M | **~85%** |
| Research paper (12 pages) | ~720K | ~108K | **~85%** |

*Token estimates based on ~4 chars/token (GPT-4 tokenizer). Actual savings depend on PDF structure.*

**Use cases:**

- Preprocessing document archives for RAG (Retrieval-Augmented Generation) pipelines
- Building LLM fine-tuning datasets from PDF collections
- Reducing token costs when processing large document sets via API
- Converting research papers for AI-assisted literature review

---

## Features

- **Live TUI** — animated spinner, progress bar, and a token-savings summary table on completion
- **Worker pool** — concurrent conversion using `runtime.NumCPU()` workers by default; configurable via `--workers`
- **Table detection** — positional text analysis coalesces characters into words, groups rows, and detects column alignment to emit GFM pipe tables
- **Two-path extraction** — positional extraction first; falls back to plain-text for image-heavy pages
- **`--strip-noise`** — aggressively removes page numbers, repeated headers/footers, and excess whitespace for maximum token density
- **Date-stamped outputs** — `report_2026-05-06.md` so you always know which version was processed
- **Single static binary** — no runtime dependencies; CGO disabled

---

## Installation

### Homebrew (macOS and Linux)

```bash
brew tap nawodyaishan/tap
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

## Usage

```bash
# Convert all PDFs in ./docs, write Markdown to ./docs/md/
pdf2md-tui convert ./docs

# Recurse into subdirectories and strip layout noise for LLM ingestion
pdf2md-tui convert ./archive --recursive --strip-noise

# Use 8 workers, custom output directory, no date suffix
pdf2md-tui convert ./papers --workers 8 --output out --date-format none

# Print version and build info
pdf2md-tui version
```

### Flags

| Flag | Short | Default | Description |
|------|-------|---------|-------------|
| `--output` | `-o` | `md/` | Output subdirectory (relative to the target directory) |
| `--recursive` | `-r` | `false` | Scan subdirectories |
| `--workers` | `-w` | `NumCPU` | Number of concurrent conversion workers |
| `--date-format` | | `2006-01-02` | Date suffix format (Go reference time); `none` disables the suffix |
| `--strip-noise` | | `false` | Aggressively remove page numbers, headers/footers, and excess whitespace |
| `--verbose` | `-v` | `false` | Print per-file errors to stderr |
| `--force` | `-f` | `false` | Overwrite existing output files without prompting |

### Output structure

```
./docs/
├── report.pdf
├── spec.pdf
└── md/
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

---

## Roadmap

Near-term planned features include a `--quiet` flag for CI use, `.txt`/`.docx` ingestion, and JSON output format. Mid-term work includes an OCR fallback for scanned PDFs. See [ROADMAP.md](ROADMAP.md) for the full picture and how to contribute.

---

## Contributing

Bug reports and feature requests: open an issue using the provided templates.

For code contributions:

1. Check [ROADMAP.md](ROADMAP.md) and open issues for `help wanted` items.
2. Fork, branch, implement, add tests, and open a PR.
3. PRs must pass `make check` (fmt + vet + lint + test with race detector).

---

## License

MIT — see [LICENSE](LICENSE).
