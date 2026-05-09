# QA Test Plan: PDF to Markdown Extraction Quality (Expanded)

This test plan defines the automated and manual validation procedures for ensuring high-fidelity PDF to Markdown conversion, specifically targeting LLM-optimized RAG pipelines.

## 1. Test Objectives
- **G1: Content Fidelity:** Ensure no characters are collapsed or lost (Double-letter preservation).
- **G2: Structural Integrity:** Preserve paragraph breaks (`\n\n`) and table layouts even in noise-stripping mode.
- **G3: Semantic Coherence:** Correctly coalesce spaced/tracked headings and specialized technical symbols (arrows, bullets).
- **G4: Cleanliness:** Eliminate encoding artifacts and handle diverse document generation sources (Word, Markdown-to-PDF, LaTeX).

## 2. Expanded Test Suite (Testdata)
| Document Category | Focus Area | Key Markers to Verify |
| :--- | :--- | :--- |
| **Technical Literature** | High volume, diverse fonts | `ebooks`, `across`, `www`, `need` |
| **Career & Strategy** | Spaced headings, emphasis | `DEFINITIVE`, `MASTER ACTION PLAN` |
| **DevOps Technical** | Symbols, CLI outputs, Flow | `→` (Arrows), `•` (Bullets), `|` (Pipes) |
| **Research/Academic** | Dense tables, PUA chars | Ligatures (`fi`, `fl`), Multi-column logic |
| **Legacy/Scan-like** | Low-quality encoding | `\ufffd` removal, Garbage text filtering |

## 3. Automated Validation Assertions
The following assertions are integrated into `pkg/service/qa_validation_test.go`:

### Level 1: Character & Word Correctness
- [x] **Assertion A1 (Double-Print):** Verify `ebooks` and `across` are NOT `eboks` or `acros`.
- [x] **Assertion A2 (Spaced Headings):** Verify `D E F I N I T I V E` is coalesced to `DEFINITIVE`.
- [x] **Assertion A3 (Technical Symbols):** Verify `→` is preserved and not replaced by `?` or boxes.

### Level 2: Layout & Structure
- [x] **Assertion B1 (Paragraph Density):** Verify output has at least 5% blank line density (indicates structure preservation).
- [x] **Assertion B2 (Table Integrity):** Verify `| --- |` presence if input has tabular data.
- [x] **Assertion B3 (Bullet Points):** Verify lists are maintained (lines starting with `•` or `-`).

### Level 3: Cleanliness
- [x] **Assertion C1 (Artifact Zero-Tolerance):** `grep` for `\ufffd` must return zero results.
- [x] **Assertion C2 (Encoding Stability):** Detect and fail on "garbage strings" (sequences of high-entropy non-ASCII characters).

## 4. Regression Matrix
| Feature | Document Source | Expected Behavior | Status |
| :--- | :--- | :--- | :--- |
| **Coalescing** | Clean Code | `across` stay `across` | ✅ |
| **Tracking** | Master Plan | `M A S T E R` becomes `MASTER` | ✅ |
| **Flow Symbols** | DevOps Pivot | `A → B` preserved | ✅ |
| **Lists** | MLOps Doc | `• Item` preserved | ✅ |
| **Sanitization** | Career Review | `ﬁ` becomes `fi` | ✅ |

## 5. Execution Workflow
1. **Build:** `make build`
2. **Clean:** `rm -rf testdata/md/*`
3. **Convert:** `bin/pdf2md-tui convert ./testdata --force --quiet`
4. **Validate:** `go test ./pkg/service -v -run TestQA_ConversionQuality`
5. **Report:** Generate `qa_report.json` for CI visibility.
