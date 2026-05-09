# Implementation Plan: Phase 2 — Proactive Health Checks

This plan details the implementation of entropy-based garbage detection to proactively identify and filter out corrupted document segments.

## Objectives
- Proactively identify document segments that suffer from encoding breakdown (silent failures).
- Reduce the amount of high-entropy "garbage" text passed to LLM pipelines.
- Improve system observability during conversion failures.

## Tasks Breakdown

### 2.1: Entropy Monitoring Engine
- [ ] Implement `CalculateShannonEntropy(text string) float64` in `pkg/repository/pdf/entropy.go`.
- [ ] Define a `ConfidenceScore` per `PageBlock` based on computed entropy.
- [ ] Benchmark standard clean text vs. known garbage (e.g., the output from *The DevOps Pivot*) to establish an entropy threshold (approx. 4.0 - 5.0 bits/char).

### 2.2: Automated Cleanliness Filter
- [ ] Integrate a "Cleanliness Filter" in `pkg/service/converter.go`.
- [ ] Blocks with entropy > `Threshold` should be:
    - Logged as a warning (with block content).
    - Optionally dropped if configured in `domain.Config`.
    - Flagged for fallback to a different extraction path if available.

### 2.3: Observability Integration
- [ ] Add a `ConversionReport` field in `domain.Result` to track high-entropy blocks encountered.
- [ ] Ensure these flags are surfaced in the JSON summary emitted when `--quiet` is enabled.

## Verification
- Run existing test suite and verify no regressions in content fidelity.
- Introduce a "poisoned" document (artificially corrupted text) and verify that the filter flags/drops the block.
