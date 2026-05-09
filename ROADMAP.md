# Roadmap

This document tracks the planned evolution of `pdf2md-tui` — a high-speed data preparation layer for multimodal AI pipelines. We prioritize maintaining a zero-dependency, single-static-binary architecture optimized for downstream RAG frameworks, agentic workflows, and Vision-Language Model (VLM) pipelines.

---

## Vision & Strategy

> **North Star:** `pdf2md-tui` is **not** a general-purpose chat tool. It is a high-speed Phase 1 ingestion engine that maximizes **context quality** for downstream AI systems. Every feature must trace back to one of the goals below.

### The Strategic Shift

The project's value proposition has evolved from _"saving UI tokens"_ to **"maximizing context quality"** for downstream AI pipelines. The tool produces the highest-fidelity structured output possible, enabling the industry-standard **"Look Twice" methodology**: extract text now, enable vision reasoning later.

### Core Problems We Solve (The "Why")

**The Semantic Disconnect & The Orphaned Asset Problem**
Traditional RAG parsers treat documents as flat text. Legacy chunkers slice through tables and orphan image references from their paragraphs, rendering vector embeddings useless. `pdf2md-tui` extracts structured Markdown alongside an isolated `./images/` directory with standardized linking syntax (`![chart](images/page1_img.png)`), providing the hierarchical DOM required by modern hybrid chunkers.

**The Ingestion-Time Summarization Fallacy**
Early tools summarized images during parsing and discarded the original pixels — catastrophic information loss for org charts, scatter plots, and complex diagrams. `pdf2md-tui` rigidly preserves high-resolution source images on disk, enabling the **"Look Twice" Dual-VLM methodology**: embed text for fast search, pull raw images for deep reasoning only when retrieved.

**Edge Hardware & Token Exhaustion**
Feeding raw PDFs or base64 image arrays into local models causes OOM crashes. In cloud environments, it burns API quotas. Our text-and-folder separation keeps Markdown lightweight for cheap semantic search, decoupling heavy visual data until explicitly needed.

### Target Audiences (The "Who")

| Audience | Needs | Use Case |
|---|---|---|
| **Enterprise RAG Architects** (LlamaIndex / LangChain) | Framework compatibility, high-fidelity tables, strict file mapping | LlamaIndex's `SimpleDirectoryReader` natively ingests mixed `.md` + `.png` directories — `pdf2md-tui` provides this out-of-the-box |
| **Local Agent & Edge AI Developers** | Pure local execution, controllable asset sizes, zero cloud dependency | Route extracted `./images/` through local downsampling scripts before feeding edge agents with restricted VRAM |
| **Agentic Workflow Builders** (MCP Adopters) | Standardized protocols, autonomous tool usage, dynamic parsing | AI agents invoke `pdf2md-tui` autonomously via MCP to parse local documents into contextually rich Markdown |

### Strategic Goals (The "What")

| # | Goal | Description |
|---|---|---|
| **G1** | **"Text + Local Image" Gold Standard** | Generate pristine Markdown mapped to a localized `./images/` directory with deterministic reference syntax (`![image](images/file_p1_1.png)`) |
| **G2** | **Chunking-Safe Structures** | Emit tables as indivisible GFM pipe tables. Bind headers to their paragraphs so hybrid chunkers cannot destroy context during embedding |
| **G3** | **Enable the "Look Twice" Pipeline** | Act as Phase 1 only — extract text, isolate images, step out of the way. No "cheap" local summarization. Downstream orchestrators handle VLM reasoning |
| **G4** | **MCP Server Readiness** | Architect Go packages so `internal/converter` can be wrapped into a standalone MCP server binary for autonomous agent invocations |

---

## Near-term (v0.x) — "Reliability & Reach"

### 🛡️ Graceful OCR Detection
**Status:** In Progress | **Goals:** G2, G3 | **Label:** [`area/extraction`](https://github.com/nawodyaishan/pdf2md-tui/labels/area%2Fextraction)

Instead of outputting empty files for scanned PDFs, the tool detects image-only pages via an `AnalyzePDF()` pre-flight heuristic — comparing extracted text character yield against `XObject` image presence. Files requiring OCR are skipped gracefully and reported in the final TUI summary as `Ignored: Requires OCR`.

### 🖼️ Image Extraction Pipeline (`--extract-images`)
**Status:** Planned | **Goals:** G1, G3 | **Label:** [`area/ingestion`](https://github.com/nawodyaishan/pdf2md-tui/labels/area%2Fingestion)

Groundwork for the "Look Twice" VLM pipeline. Uses **`pdfcpu`** (pure Go, zero Cgo) to extract raw PDF images to an isolated `./images/{document}/` directory with deterministic naming (`page{N}_img{M}.png`). Injects `![image](images/doc/page1_img1.png)` references into the Markdown stream at correct positional offsets, maintaining spatial relationships for downstream chunkers.

### 🧱 Chunking-Safe Table Output
**Status:** Completed | **Goals:** G2 | **Label:** [`area/extraction`](https://github.com/nawodyaishan/pdf2md-tui/labels/area%2Fextraction)

Ensure positional extraction emits GFM pipe tables as indivisible atomic units. Headers remain permanently bound to their subordinate paragraphs. This prevents modern hybrid chunkers (Docling `HybridChunker`, LlamaIndex) from destroying table integrity during vector embedding.

### ⚡ Zero-Arg Execution
**Status:** Planned | **Goals:** All | **Label:** [`area/cli`](https://github.com/nawodyaishan/pdf2md-tui/labels/area%2Fcli)

Enhance usability by allowing `pdf2md-tui` to run without arguments. It will default to the current working directory, scan for `.pdf` files, and immediately launch the interactive wizard. If no PDFs are found, it prints a graceful message and exits cleanly.

### 🔇 CI-Friendly Quiet Mode
**Status:** Planned | **Goals:** G4 | **Label:** [`area/cli`](https://github.com/nawodyaishan/pdf2md-tui/labels/area%2Fcli)

A `--quiet` flag to suppress the TUI and progress bars, emitting only a final JSON summary to stdout: `{"converted": 12, "skipped": 3, "errors": 1, "files": [...]}`. Perfect for automation, piping into `jq`, and future MCP server invocations.

---

## Mid-term (v1.x) — "Pluggable Intelligence"

### 👁️ "Look Twice" VLM Pipeline (v1.1)
**Status:** Research | **Goals:** G1, G3 | **Label:** [`area/vlm`](https://github.com/nawodyaishan/pdf2md-tui/labels/area%2Fvlm)

Leverages the image extraction pipeline (v0.x) to enable the full **Dual-VLM "Look Twice" methodology**. Adds a `--provider [openai|anthropic|google]` flag to send detected image-only pages or extracted images to cloud VLM APIs for high-accuracy visual understanding. Returns results directly into the Markdown stream — completing the Phase 2 retrieval-time reading cycle.

### 🔌 MCP Server Prototype
**Status:** Planned | **Goals:** G4 | **Label:** [`area/distribution`](https://github.com/nawodyaishan/pdf2md-tui/labels/area%2Fdistribution)

Wrap `internal/converter` in a standalone MCP server binary (`cmd/pdf2md-mcp/`). Exposes `convert` as an MCP tool primitive, enabling autonomous AI agents to invoke PDF-to-Markdown conversion without manual human preparation. The core converter package remains TUI-free.

### 📄 Heterogeneous Ingestion (`.docx`, `.txt`)
**Status:** Planned | **Label:** [`area/ingestion`](https://github.com/nawodyaishan/pdf2md-tui/labels/area%2Fingestion)

Expanding the `discovery` package to handle Word documents and raw text files, making `pdf2md-tui` a universal document-to-LLM bridge.

### 🔄 Watch Mode
**Status:** Planned | **Label:** [`area/cli`](https://github.com/nawodyaishan/pdf2md-tui/labels/area%2Fcli)

Monitor directories via `fsnotify` to automatically convert new PDFs as they arrive in a folder.

---

## Long-term (v2.x) — "Ecosystem & Scale"

### 🧩 Pluggable Post-Processors
**Status:** Exploratory | **Label:** [`area/output`](https://github.com/nawodyaishan/pdf2md-tui/labels/area%2Foutput)

A plugin system to allow custom Markdown formatting, YAML frontmatter injection, and metadata enrichment (e.g., auto-summarization via downstream LLMs).

### 🕸️ Full MCP Server + REST API
**Status:** Exploratory | **Label:** [`area/distribution`](https://github.com/nawodyaishan/pdf2md-tui/labels/area%2Fdistribution)

Evolve the MCP prototype into a production server. Add a `serve` command to expose the conversion pipeline as a REST/gRPC API, enabling team-wide document processing microservices and dynamic agent-driven workflows.

---

## Icebox — Not Currently Planned

### ~~🔤 Native OCR Integration (Tesseract)~~

Local native OCR violates our pure-Go, single-binary philosophy. Adding Cgo-dependent OCR engines (like Tesseract) bloats the binary, breaks cross-platform portability, and introduces complex system dependency management. Visual understanding is instead delegated to external VLM APIs via the **"Look Twice" pipeline** (see Mid-term v1.1).

---

## Architectural Principles

These principles are non-negotiable constraints that govern all contributions:

| Principle | Rule |
|---|---|
| **Zero-Cgo Policy** | All PRs must maintain `CGO_ENABLED=0`. No native system dependencies. Single static binary on every platform. |
| **Output Contract** | Markdown + `./images/` directory is the canonical output format. Image references use standard `![alt](images/...)` syntax. |
| **Package Separation** | `internal/converter` must never import TUI dependencies (`bubbletea`, etc.). This ensures the core can be wrapped by both the TUI and a future MCP server binary. |
| **No Ingestion-Time Summarization** | The tool extracts and preserves — it does not summarize. Downstream orchestrators own the reasoning layer. |

---

## Contributing to the Roadmap

- **Vote** on items with 👍 on GitHub Issues.
- **Claim** an item by commenting on the issue and opening a PR.
- All contributions must adhere to the **Architectural Principles** above.
