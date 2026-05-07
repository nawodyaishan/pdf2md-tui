# Refactoring Plan: `internal/repository/pdf/tables.go`

This document outlines the strategy to resolve Clean Architecture violations, improve code quality, and enhance test coverage in the PDF table extraction logic.

## 1. Clean Architecture Violations
*   **Mixing Data Extraction with Presentation (Formatting):** `tables.go` resides in the `repository` layer, whose sole responsibility should be retrieving and structuring raw data from the PDF. However, functions like `renderRowsAsMarkdown` directly generate Markdown strings. This tightly couples PDF coordinate parsing with Markdown presentation logic.
*   **Infrastructure Leakage in Domain Logic:** While passing `ledongpdf.Text` around inside the repository is acceptable, the complex business logic of determining what constitutes a "word" or a "table" is heavily intertwined with the specific fields of the `ledongthuc/pdf` library.

## 2. Code Quality Improvements
*   **Duplicated Clustering Logic (DRY Violation):** Both `detectColumnPositions` and `findTableEnd` contain nearly identical logic to group X-coordinates into clusters/buckets using a `minGap` of 50.0. This must be extracted into a shared helper function.
*   **Monolithic Function (`coalesceChars`):** This function does entirely too much. It handles indexing, Y-coordinate clustering, X-coordinate sorting, median calculation, gap analysis, deduplication, and sanitization. It needs to be decomposed into single-responsibility helpers.
*   **Magic Numbers:** The file is littered with hardcoded heuristics that lack context. Values like `1.0` (Y-tolerance), `0.5` (X-noise floor), `50.0` (min column gap), `5.0` (bucket size), and `0.4` (frequency threshold) should be defined as named constants at the top of the file to improve readability and tweak-ability.
*   **Dead/Placeholder Code:** `deduplicateDoubleGlyphs` was stripped down to a placeholder returning `s` as-is. It adds unnecessary overhead to the sanitization loop and should be removed entirely if coordinate-level deduplication is deemed sufficient.

## 3. Test Improvements
*   **Missing Structural Heuristic Tests:** There are no tests covering the table detection logic (`detectColumnPositions` and `findTableEnd`). Since these rely on complex heuristics, they are highly susceptible to regressions and desperately need unit tests.
*   **Missing Rendering Tests:** `renderRowsAsMarkdown` lacks tests verifying the newly added "Table Fencing" (`\n\n` boundaries) and proper cell alignment.
*   **Incomplete Adaptive Thresholding Coverage:** The current `coalesceChars` tests use clean, well-formed data. We need edge-case tests specifically targeting the new logic:
    *   Missing widths (`W: 0`) to trigger the `FontSize * 0.4` fallback.
    *   Coordinate stacking deduplication (testing the `< 0.5pt` logic).
    *   Varying widths on a single line to ensure the `medianW` calculation works as expected.

---

## Execution Strategy

### Phase 1: Code Quality & Refactoring (Target: `tables.go`)
1.  Define named constants for all magic numbers.
2.  Extract the shared column clustering logic from `detectColumnPositions` and `findTableEnd` into a new `extractColumnClusters(cells []tableCell) []float64` helper.
3.  Decompose `coalesceChars` into smaller helpers.
4.  Remove the `deduplicateDoubleGlyphs` function and inline the remaining sanitization.

### Phase 2: Test Suite Expansion (Target: `tables_test.go`)
1.  Write unit tests for the new `extractColumnClusters` helper to verify it merges close coordinates correctly.
2.  Write unit tests for `findTableEnd` and `detectColumnPositions` using mock `tableRow` data.
3.  Write tests for `renderRowsAsMarkdown` to definitively prove the Table Fencing logic works.
4.  Add edge-case tests to `coalesceChars` targeting zero-width characters and stacked coordinates.

### Phase 3: Architectural Boundary Improvement (Target: `pdf.go` & Service Layer)
1.  *Refactor:* Define a `PageBlock` abstraction (e.g., `TextBlock`, `TableBlock`) to represent structural elements independent of Markdown.
2.  *Refactor:* Change `extractWithTables` to return `[]PageBlock` instead of a raw `string`.
3.  *Refactor:* Move `renderRowsAsMarkdown` out of the repository and into a dedicated formatting function in the `service` layer, ensuring the repository only handles extraction.