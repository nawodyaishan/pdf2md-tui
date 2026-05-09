# Implementation Plan: Scalable QA Infrastructure

This document outlines the strategic roadmap for evolving the `pdf2md-tui` extraction engine from reactive heuristic-chasing to a robust, scalable quality engineering discipline.

---

## 🛑 Retrospective: The "Heuristic Chase"
Our recent overhaul revealed four systemic pain points in our extraction process:
1.  **Fragile Heuristics:** Reliance on fixed constants (e.g., `0.5pt`) fails when source document styles vary (LaTeX vs. Word).
2.  **Ambiguous Ground Truth:** Marker-based testing (`grep "across"`) creates blind spots for unseen regressions.
3.  **Silent Encoding Failures:** High-entropy "garbage" text often bypasses standard sanitization.
4.  **Structural Bias:** Simple grid-based table detection breaks on document micro-misalignments.

---

## 📈 Roadmap: Scalable Quality Engineering

### Phase 1: Robust Ground Truth (Snapshot Testing)
*Target: Eliminate marker-based testing in favor of full-content verification.*

- [x] **Task 1.1:** Build `TestQA_GoldenMaster` snapshot infrastructure (`pkg/service/qa_golden_test.go`).
- [x] **Task 1.2:** Establish golden reference repository in `testdata/devops_project/golden/`.
- [x] **Task 1.3:** Implement a CLI flag `qa --update-snapshots` (Integrated in test harness).

### Phase 2: Proactive Health Checks (Encoding & Garbage Detection)
*Target: Proactively catch corrupted inputs and silent encoding failures.*

- [x] **Task 2.1:** Implement Shannon Entropy threshold monitoring in `pkg/repository/pdf/entropy.go`.
- [x] **Task 2.2:** Define a `ConfidenceScore` per `PageBlock` (Implied by block entropy filtering).
- [x] **Task 2.3:** Add an automated "Cleanliness Filter" to drop high-entropy blocks (Implemented in `converter.go`).

### Phase 3: Extraction Profiles (Source-Aware Logic)
*Target: Enable engine adaptation based on document origin (Word, LaTeX, Scans).*

- [x] **Task 3.1:** Introduce `ExtractionProfile` types in `domain/config.go` (`Default`, `Academic`, `Legacy`).
- [x] **Task 3.2:** Refactor `pkg/repository/pdf` to consume these profiles, allowing dynamic threshold adjustment.

### Phase 4: Structural Integrity
*Target: Ensure table layouts are semantically coherent for LLMs.*

- [x] **Task 4.1:** Develop a structural `BlockValidator` (`ValidateTableStructure`) to ensure row-column consistency.
- [x] **Task 4.2:** Bubble up structural validation errors into the `domain.Result` object.

---

## 5. QA Discipline: Execution Workflow
To ensure scalability, the following workflow is mandatory for all future contributions:

1.  **Snapshot Update:** Whenever logic changes, verify changes via human-reviewed diffs:
    `go test -v ./pkg/service -run TestQA_GoldenMaster -- -update`
2.  **Integrity Gate:** Run the full integration test suite:
    `make check`
3.  **Cleanliness Audit:** Check the conversion summary for entropy warnings/OCR flags.
4.  **Regression Matrix:** Update the regression matrix in `qa_test_plan.md` to reflect new edge cases.

---

## 6. Future Proofing
- **Semantic Similarity Metrics:** Integrate LLM/BERT-based scoring to ensure extraction preserves the *meaning* of the original text during noise-stripping.
- **Synthetic Mocking:** Develop a `SyntheticPDF` utility to test mathematical edge cases without the brittleness of opaque binary test files.
