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

---

## 6. QA Retrospective: Pain Points & Scalability
*Self-reflection on the v1.2.7 extraction overhaul session.*

### 🛑 Critical Pain Points
1.  **The "Heuristic Chase":** Statistical coalescing (Mean/StdDev) is inherently fragile across different PDF generators. Changes in source document fonts (LaTeX vs. MS Word) necessitated multiple iterative calibration loops.
2.  **Ambiguous Ground Truth:** Current validation relies on `grep` for specific markers (`across`, `ebooks`). This creates "blind spots" for regressions in words not explicitly tracked in the test suite.
3.  **Encoding Breakdown Blindspots:** Certain PDF encodings fail silently, producing high-entropy "garbage text." Detecting these via standard assertions is difficult without specialized entropy checks.
4.  **Visual vs. Logical Layout:** Table layouts often "look" correct to humans but contain micro-misalignments that break downstream AI parsers. Simple string matching is insufficient for structural validation.

### 📈 Scalability Improvements (Future QA Discipline)
1.  **Golden-Master Snapshot Testing:** Transition to using "Golden Markdown" reference files. Any change in conversion output should trigger a mandatory human-reviewed diff. This ensures 100% content coverage instead of marker-based checks.
2.  **Source-Aware Tolerance Profiles:** Implement document-specific extraction profiles. A "LaTeX Profile" might use stricter character bounds, while an "Office Profile" handles more aggressive tracking.
3.  **Shannon Entropy Monitoring:** Integrate automated garbage detection using character entropy. If a block's entropy exceeds a threshold, it should be flagged for OCR rather than emitting corrupted text.
4.  **Semantic Similarity Metrics:** For critical documents, use a small LLM or BERT-based scorer to verify that extraction hasn't destroyed the semantic meaning (e.g., merging "not" with another word).
5.  **Synthetic PDF Mocking:** Develop a test utility to generate PDFs with precisely controlled character coordinates. This allows for unit-testing the extraction engine against mathematical edge cases without opaque binary files.
