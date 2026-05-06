# pdf2md-tui

A high-performance Go CLI utility for batch-converting PDFs to clean, **LLM-friendly Markdown** — optimized for token efficiency when feeding documents into AI/LLM pipelines.

## Why LLM-Optimized Markdown?

PDFs are among the worst formats for LLM consumption. Binary encoding, embedded fonts, layout metadata, headers/footers, and page-break artifacts all waste tokens and degrade context quality. A 50-page PDF can burn 3-5x more tokens than the equivalent clean Markdown.

**pdf2md-tui** solves this by extracting **only the semantic text content** from PDFs and outputting minimal, clean Markdown that:

- **Eliminates layout noise** — no page numbers, headers/footers, or formatting artifacts
- **Maximizes token density** — pure text content means every token carries meaning
- **Preserves structure** — headings, paragraphs, and lists remain intact
- **Batch-ready** — convert entire documentation directories in seconds for RAG pipelines, fine-tuning datasets, or context window stuffing
- **Date-stamped outputs** — track which version of a document was processed

## Installation

### Homebrew (macOS & Linux)
```bash
brew tap nawodyaishan/tap
brew install pdf2md-tui
```

### Go Install
```bash
go install github.com/nawodyaishan/pdf2md-tui/cmd/pdf2md-tui@latest
```

## Usage

```bash
pdf2md-tui convert ./docs --recursive --strip-noise
```

### Flags

| Flag | Short | Default | Description |
|------|-------|---------|-------------|
| `--output` | `-o` | `md/` | Output subdirectory name |
| `--recursive` | `-r` | `false` | Scan subdirectories |
| `--workers` | `-w` | `NumCPU`| Concurrent conversion workers |
| `--date-format` | | `2006-01-02` | Date suffix format |
| `--verbose` | `-v` | `false` | Verbose logging |
| `--strip-noise` | | `false` | Aggressively remove boilerplate for LLM optimization |

## License
MIT
