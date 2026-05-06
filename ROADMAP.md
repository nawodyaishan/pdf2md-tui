# Roadmap

This document tracks the planned evolution of `pdf2md-tui`. Items are grouped by horizon; priorities shift based on community feedback. The canonical source of truth for active work is [GitHub Issues](https://github.com/nawodyaishan/pdf2md-tui/issues) — roadmap items are labelled `roadmap`.

---

## Near-term (v0.x)

### `--quiet` / CI-friendly output mode
**Status:** Planned | **Label:** [`area/cli`](https://github.com/nawodyaishan/pdf2md-tui/labels/area%2Fcli)

When stdout is not a TTY, the TUI (spinner, progress bar, summary table) currently auto-degrades but still emits some output. A dedicated `--quiet` flag will suppress all stdout except errors, making the tool composable in CI pipelines and shell scripts.

```bash
pdf2md-tui convert ./docs --quiet && echo "done"
```

### `.txt` and `.docx` ingestion
**Status:** Planned | **Label:** [`area/ingestion`](https://github.com/nawodyaishan/pdf2md-tui/labels/area%2Fingestion)

Extend the `discovery` package and `Converter` to accept plain-text and Word documents. `.txt` files require only encoding normalization; `.docx` can be unpacked as ZIP and the `word/document.xml` parsed for semantic structure. This lets teams run a single command over heterogeneous document archives.

### Shell completions
**Status:** Planned | **Label:** [`area/cli`](https://github.com/nawodyaishan/pdf2md-tui/labels/area%2Fcli)

Generate Bash, Zsh, Fish, and PowerShell completions via Cobra's built-in `completion` subcommand and document the install steps in the README.

### `--output-format` flag
**Status:** Planned | **Label:** [`area/output`](https://github.com/nawodyaishan/pdf2md-tui/labels/area%2Foutput)

Support `--output-format=json` to emit structured JSON (`{ "file": "...", "content": "...", "bytes": N, "tokens_est": N }`) for downstream pipeline integration (e.g., piping into `jq` or a RAG ingestion script).

---

## Mid-term (v1.x)

### OCR fallback for scanned PDFs
**Status:** Research | **Label:** [`area/ocr`](https://github.com/nawodyaishan/pdf2md-tui/labels/area%2Focr)

The current extraction pipeline relies on embedded text in PDFs. Scanned documents (image-only pages) yield no text. The planned approach:

1. Detect image-only pages by checking if `page.Content().Text` is empty after positional extraction.
2. Rasterize the page using a pure-Go PDF renderer (candidate: `github.com/dontpanic92/docconv` or shell out to `pdftoppm`).
3. Pass the rasterized image to [Tesseract](https://github.com/otiai10/gosseract) via its Go bindings.
4. Gate the OCR path behind a `--ocr` flag to keep the default binary dependency-free.

This is a mid-term item because Tesseract adds a native dependency and complicates cross-platform distribution.

### Markdown post-processing pipeline
**Status:** Planned | **Label:** [`area/output`](https://github.com/nawodyaishan/pdf2md-tui/labels/area%2Foutput)

A pluggable post-processor chain applied after extraction: heading detection (lines in ALL CAPS → `## Heading`), list normalization (bullet normalization, numbered list repair), and YAML frontmatter injection (`title`, `source`, `date`, `page_count`, `token_est`).

### Watch mode
**Status:** Planned | **Label:** [`area/cli`](https://github.com/nawodyaishan/pdf2md-tui/labels/area%2Fcli)

`pdf2md-tui watch <dir>` monitors a directory with `fsnotify` and converts new or modified PDFs on arrival. Intended for teams that auto-export PDFs from document management systems.

### Configuration file
**Status:** Planned | **Label:** [`area/config`](https://github.com/nawodyaishan/pdf2md-tui/labels/area%2Fconfig)

Support a `.pdf2md.yaml` (or `pdf2md.toml`) project config file via `github.com/spf13/viper` so per-project defaults (output dir, workers, date format, strip-noise) don't need to be re-specified on every invocation.

---

## Long-term (v2.x)

### npm binary wrapper distribution
**Status:** Research | **Label:** [`area/distribution`](https://github.com/nawodyaishan/pdf2md-tui/labels/area%2Fdistribution)

Distribute `pdf2md-tui` via npm using the `optionalDependencies` pattern pioneered by esbuild and Turbo:

- A main wrapper package (`@nawodyaishan/pdf2md-tui`) containing a JS shim.
- Per-platform optional packages (`@nawodyaishan/pdf2md-tui-darwin-arm64`, `-linux-x64`, `-win32-x64`, etc.) each shipping the pre-built Go binary.
- `package.json` `os` + `cpu` fields ensure npm installs only the relevant platform binary.
- The JS shim resolves the correct binary path and `execFileSync`s it, forwarding all arguments.

This unlocks `npx pdf2md-tui` and lets Node.js/Python toolchains add `pdf2md-tui` as a dev dependency without requiring Go or Homebrew.

**Implementation reference:** [esbuild's npm package structure](https://github.com/evanw/esbuild/tree/main/npm)

### Plugin API
**Status:** Exploratory

A Go plugin interface (`Processor`) that third parties can implement to add custom extraction backends, post-processors, or output formats — loadable via Go's `plugin` package or a subprocess protocol.

### Web UI / API server mode
**Status:** Exploratory

`pdf2md-tui serve --port 8080` — expose conversion as an HTTP endpoint for integration with document management platforms and no-code tools.

---

## Contributing to the Roadmap

- **Vote** on items with 👍 on the linked GitHub Issue.
- **Propose** new items by opening an Issue with the `roadmap` label.
- **Claim** an item by commenting on the issue and opening a PR — see [CONTRIBUTING.md](CONTRIBUTING.md).

Milestones on GitHub Issues denote the target release quarter.
