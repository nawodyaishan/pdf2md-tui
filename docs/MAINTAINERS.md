# Maintainers Guide

Internal reference for the core team covering launch strategy, community management, and contributor onboarding.

---

## Core Pitch

> **PDFs waste LLM tokens. `pdf2md-tui` batch-extracts semantic Markdown to reduce RAG token usage by ~80%.**

One-liner variants:
- "Convert your PDF archive to LLM-ready Markdown in seconds — Go CLI, worker pool, TUI included."
- "Stop feeding raw PDFs to your RAG pipeline. Every token matters."
- "pdf2md-tui: the missing pre-processor for AI document pipelines."

---

## Pre-Launch Checklist

### Repository hygiene
- [ ] All issues labelled; `good first issue` and `help wanted` labels seeded with 3–5 realistic items (e.g., "Add shell completions", "Write --quiet flag", "Add .txt ingestion")
- [ ] `CONTRIBUTING.md` written (PR workflow, code style, test requirements)
- [ ] `SECURITY.md` with responsible disclosure instructions
- [ ] `LICENSE` committed (MIT)
- [ ] GitHub Topics set: `pdf`, `markdown`, `llm`, `cli`, `golang`, `rag`, `ai`, `tui`
- [ ] GitHub Social Preview image (1280×640) designed and uploaded
- [ ] Pinned README badges: CI status, Go version, license, latest release

### Demo assets
- [ ] Record terminal GIF demo using [vhs](https://github.com/charmbracelet/vhs) — target 30–45 seconds showing: `make build` → `pdf2md-tui convert ./docs --recursive --strip-noise` → summary table with token savings
- [ ] Embed GIF in README above the Installation section
- [ ] Optional: `asciinema` recording for interactive playback

### Distribution verification
- [ ] `brew tap nawodyaishan/tap && brew install pdf2md-tui` works end-to-end
- [ ] `go install github.com/nawodyaishan/pdf2md-tui/cmd/pdf2md-tui@latest` works
- [ ] GitHub Release page has all expected artifacts (Linux/macOS/Windows × amd64/arm64 + .deb/.rpm)
- [ ] `pdf2md-tui version` prints correct version, commit, and date

### Messaging prep
- [ ] Show HN post title drafted (≤80 chars): `Show HN: pdf2md-tui – batch-convert PDFs to LLM-optimized Markdown (Go CLI)`
- [ ] Opening HN comment written (problem → solution → honest limitations → invite feedback)
- [ ] Reddit post body drafted (plain, technical, no marketing superlatives)
- [ ] Short pitch for Twitter/X: two sentences + GIF + install command

---

## Launch Channels and Sequence

Launch on a **Tuesday–Thursday**, between **07:00–10:00 PT** for maximum HN visibility.

### Day 0 — Warm-up
1. Ensure the GitHub repo is public with README GIF and all checklist items above complete.
2. Post to any personal network / Discord servers with a soft "just shipped" note — no ask, just sharing.

### Launch Day
1. **Hacker News** — "Show HN" post. Stay in comments for 24–48 hours; respond to every technical question. Do not ask others to upvote.
2. **r/LocalLLaMA** — Primary target audience. Frame the post around RAG token efficiency, not the CLI itself. Title: "I built a Go CLI to reduce PDF→RAG token waste by ~80% — batch conversion with a TUI"
3. **r/golang** — Technical audience. Emphasize the worker pool, positional text extraction, and table detection algorithm.
4. **r/commandline** — Broader CLI enthusiast audience. Focus on the TUI and ease of use.
5. **Twitter/X and Bluesky** — Short pitch + terminal GIF + `brew install` one-liner. Tag relevant accounts in the LLM/RAG space.

### Week 2
- Post a follow-up HN comment or new thread: "Two weeks in — what we learned from the pdf2md-tui launch" (community loves retrospectives).
- Submit to relevant "awesome" lists: `awesome-go`, `awesome-llm-tools`, `awesome-cli-apps`.
- Pitch to developer newsletters: Go Weekly, TLDR DevTools, Console.dev.

---

## Community Engagement Guidelines

### PR review SLA
| PR type | Target first response | Target merge/close |
|---------|----------------------|-------------------|
| Bug fix with tests | 48 hours | 5 business days |
| Feature aligned with roadmap | 72 hours | 2 weeks |
| Feature not on roadmap | 1 week | Discuss in issue first |
| Documentation | 24 hours | 3 business days |

Label every open PR within 24 hours of opening: `needs-review`, `approved`, `needs-changes`, or `on-hold`.

### Contributor onboarding
1. Welcome comment on first-time contributor PRs linking to `CONTRIBUTING.md`.
2. Assign `good first issue` items to contributors who ask — don't let them go unclaimed.
3. Add contributors to `CONTRIBUTORS.md` (or GitHub's built-in contributors graph) once their first PR merges.
4. Monthly: scan open issues for stale PRs (>30 days no activity) and either revive or close with explanation.

### Issue triage
- Respond to all bug reports within 48 hours with a reproduction request or acknowledgement.
- Close feature requests that conflict with the project's scope (LLM token efficiency, batch CLI) with a polite explanation and pointer to the ROADMAP.
- Add `help wanted` to any bug you've reproduced but won't fix in the next sprint.

---

## Token Savings Methodology

The ~80% figure used in pitches is derived from:

- **PDF token estimate**: raw PDF binary encoded as base64 or passed as bytes → ~3–5x the token count of the same semantic content as plain text, due to binary headers, embedded fonts, and layout metadata.
- **Markdown token estimate**: clean extracted text at ~4 chars/token (GPT-4 tokenizer heuristic for English).
- **Measured savings**: internal test on a 50-page technical PDF: 3.1M tokens (raw PDF) → 450K tokens (clean Markdown) = 85% reduction.

Always present this as an estimate with a qualifier ("~80% reduction in our testing") rather than an absolute claim.
