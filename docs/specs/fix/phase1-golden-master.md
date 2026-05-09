# Implementation Plan: Phase 1 — Golden-Master Snapshot Testing

This plan details the implementation of a snapshot-based quality gate to replace marker-based `grep` assertions, ensuring 100% content fidelity in PDF-to-Markdown conversions.

## Objectives
- Achieve full content validation for every PDF conversion.
- Enable human-reviewed diffs for all structural or formatting changes.
- Remove the dependency on fragile keyword markers (`across`, `ebooks`).

## Tasks Breakdown

### 1.1: Snapshot Harness Finalization
- [ ] Refactor `pkg/service/qa_golden_test.go` to support automatic recursive discovery of reference files.
- [ ] Implement a lightweight diff helper that provides context-aware output (using `diff` or a Go-native equivalent) when a snapshot mismatch is detected.

### 1.2: Establish Golden Repository
- [ ] Create `testdata/devops_project/golden/` with verified, high-fidelity Markdown outputs.
- [ ] Integrate this directory into the Git ignore, *except* for the committed `.golden.md` reference files.

### 1.3: CLI Integration (QA Helper)
- [ ] Add a new CLI command `pdf2md-tui qa --update-golden` to the `handler/cli` layer.
- [ ] This command must safely trigger the snapshot update logic, ensuring developers have a frictionless way to promote changes.

## Verification
- Run `go test -v ./pkg/service -run TestQA_GoldenMaster` and ensure zero failures.
- Verify that a controlled modification to `converter.go` triggers a snapshot failure and a visible diff report.
