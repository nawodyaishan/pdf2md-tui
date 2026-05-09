# Implementation Plan: Phase 4 — Structural Integrity

This plan details the implementation of a structural validator to ensure row-column consistency in extracted tabular data.

## Objectives
- Enforce semantic consistency for all extracted `BlockTypeTable` elements.
- Proactively detect and flag fragmented or misaligned tables before Markdown emission.
- Provide clear feedback in `domain.Result` when table structure is invalid.

## Tasks Breakdown

### 4.1: BlockValidator Implementation
- [ ] Implement `ValidateTableStructure(table domain.TableData) error` in `pkg/repository/pdf/tables.go`.
- [ ] Logic: Ensure all rows in a table have an equal number of columns.
- [ ] Logic: Flag empty cells or cells that consistently fail to span the row width.

### 4.2: Result Integration
- [ ] Update `domain.Result` to include `StructuralErrors []string`.
- [ ] Integrate the `BlockValidator` into the existing `buildPageBlocks` flow.
- [ ] Emit errors when a detected table fails structural validation.

### 4.3: Table Detection Thresholds
- [ ] Evaluate the findings from `razvandimescu/gopdf` and adapt our `buildPageBlocks` logic to handle varying row gaps more robustly.

## Verification
- Run existing integration tests and confirm no regressions in table extraction.
- Introduce a "poisoned" table PDF (with irregular row lengths) and verify it is correctly flagged by the `BlockValidator`.
