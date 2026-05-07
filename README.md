# pdf2md-tui

> High-speed PDF → Markdown ingestion engine for multimodal RAG pipelines. Extracts structured text + isolated images so downstream chunkers, LlamaIndex, and VLM agents get context that actually works.

[![CI](https://github.com/nawodyaishan/pdf2md-tui/actions/workflows/ci.yml/badge.svg)](https://github.com/nawodyaishan/pdf2md-tui/actions/workflows/ci.yml)
[![Go Version](https://img.shields.io/github/go-mod/go-version/nawodyaishan/pdf2md-tui)](go.mod)
[![Latest Release](https://img.shields.io/github/v/release/nawodyaishan/pdf2md-tui)](https://github.com/nawodyaishan/pdf2md-tui/releases/latest)
[![License: MIT](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)

---

<div align="center">
  <img src="public/banner.jpeg" alt="pdf2md-tui banner" width="600" />
</div>

## Why?

**PDFs break AI pipelines.** Binary encoding, embedded fonts, and layout metadata inflate token counts — but the real damage is structural. Legacy parsers flatten documents into raw text, destroying tables, orphaning image references, and producing vector embeddings that are functionally useless for retrieval.

`pdf2md-tui` solves this by extracting **structured Markdown** with preserved table layouts alongside an isolated `./images/` directory — the exact format that modern RAG frameworks (LlamaIndex, LangChain) and Vision-Language Models expect.

| Document | Raw PDF (est. tokens) | Clean Markdown (est. tokens) | Savings |
|---|---|---|---|
| 50-page technical spec | ~3.1M | ~450K | **~85%** |
| 200-page legal contract | ~12M | ~1.8M | **~85%** |
| Research paper (12 pages) | ~720K | ~108K | **~85%** |

*Token estimates based on ~4 chars/token (GPT-4 tokenizer). Actual savings depend on PDF structure.*

**Built for the "Look Twice" methodology** — extract text + isolate images at ingestion time (Phase 1), then let your downstream VLM pipeline handle deep visual reasoning at retrieval time (Phase 2).

**Use cases:**

- Preprocessing document archives for RAG pipelines (LlamaIndex `SimpleDirectoryReader` compatible)
- Building multimodal knowledge bases with text + image vector stores
- Feeding autonomous AI agents via MCP with structured, parseable document context
- Reducing token costs when processing large document sets via API

---

## Features

- **Chunking-safe tables** — positional text analysis detects column alignment and emits GFM pipe tables as indivisible atomic units, preventing downstream chunkers from destroying table integrity
- **Two-path extraction** — positional extraction first (preserves structure); falls back to plain-text for edge cases
- **Worker pool** — concurrent conversion using `runtime.NumCPU()` workers by default; configurable via `--workers`
- **Live TUI** — animated spinner, progress bar, and a context-quality summary table on completion
- **Interactive Menu** — run `pdf2md-tui` without arguments to launch a guided configuration wizard
- **Graceful OCR detection** — scanned/image-only PDFs are detected and skipped cleanly, reported in the summary (no empty output files)
- **`--strip-noise`** — aggressively removes page numbers, repeated headers/footers, and excess whitespace for maximum token density
- **Smart Overwrites** — interactively prompts before overwriting existing files, bypassed via `--force`
- **Date-stamped outputs** — `report_2026-05-06.md` so you always know which version was processed
- **Single static binary** — no runtime dependencies; `CGO_ENABLED=0`, pure Go

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

### Interactive Mode

Simply run the tool without arguments to launch the guided interactive wizard. It will prompt you for directories, flags, and handle file overwrites smoothly:

```bash
pdf2md-tui
```

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
| `--force` | `-f` | `false` | Overwrite existing output files without prompting |
| `--strip-noise` | | `false` | Aggressively remove page numbers, headers/footers, and excess whitespace |
| `--extract-images` | | `false` | Extract embedded images and inject markdown references |
| `--verbose` | `-v` | `false` | Print per-file errors to stderr |

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

The roadmap is organized around maximizing **context quality** for downstream AI pipelines:

- **Near-term (v0.x)** — Image extraction pipeline (`pdfcpu`), graceful OCR detection, chunking-safe table output, `--quiet` JSON mode for CI/MCP
- **Mid-term (v1.x)** — "Look Twice" VLM pipeline (cloud vision providers), MCP server prototype, `.docx`/`.txt` ingestion
- **Long-term (v2.x)** — Pluggable post-processors, full MCP server + REST API

See [ROADMAP.md](ROADMAP.md) for the full vision, strategic goals, and how to contribute.

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
