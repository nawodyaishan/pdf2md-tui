# Implementation Plan: Phase 3 — Extraction Profiles

This plan details the implementation of source-aware extraction profiles to handle the high variance in PDF document quality (e.g., LaTeX-generated research vs. legacy scanned/Office documents).

## Objectives
- Introduce document-specific extraction profiles to adapt thresholding dynamically.
- Reduce the need for "global" constant tuning, which currently leads to heuristic fragility.
- Provide a robust way for users (or the engine) to optimize extraction based on PDF origin.

## Tasks Breakdown

### 3.1: Define Profile Domain
- [ ] Add `ExtractionProfile` enum to `pkg/domain/config.go` (`ProfileDefault`, `ProfileAcademic`, `ProfileLegacy`).
- [ ] Update `Config` struct to hold the active profile.

### 3.2: Profile-Based Thresholding
- [ ] Refactor `pkg/repository/pdf/tables.go` to accept an `ExtractionProfile`.
- [ ] Implement a factory or mapping that translates profiles to specific constants (e.g., `tableColumnFreqThresh`, `charSpaceRatio`).
- [ ] Update `processLineIntoWords` to use these profile-specific thresholds.

### 3.3: Auto-Discovery Heuristics
- [ ] Implement a pre-flight "PDF Scanner" that inspects document metadata (`/Creator`, `/Producer`) to auto-select the best-fitting `ExtractionProfile`.

## Verification
- Test `Academic` profile on dense research papers to ensure high table fidelity.
- Test `Legacy` profile on scanned PDFs to ensure aggressive noise filtering without text corruption.
- Verify that `Default` profile remains unchanged for general-purpose usage.
