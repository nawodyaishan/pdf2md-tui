# Tech Spec: Automated PDF-to-Markdown Regression Framework

## 1. Executive Summary
This specification defines the architecture for a scalable automated regression framework designed to manage high-variance document ingestion in `pdf2md-tui`. By combining real-world document case studies with synthetic geometric mocks, we eliminate heuristic fragility and provide a robust gate for future feature development.

## 2. Architectural Components

### 2.1 Regression Corpus (`testdata/regression/`)
- Stores anonymized real-world PDFs for integration testing.
- Mandatory pairing: Every `X.pdf` must have a corresponding `X.golden.md` snapshot.

### 2.2 Synthetic Mock Harness (`pkg/repository/pdf/mock_test.go`)
- Enables unit-testing extraction logic against precisely defined coordinate geometries.
- Avoids reliance on binary PDF bloat for every edge case.

### 2.3 The "Golden Master" Pipeline
- `go-cmp` based comparison tool for snapshot verification.
- Enforced equality via `TestQA_GoldenMaster`.

## 3. Workflow Specification
1. **Ingest:** Identify a failing real-world PDF.
2. **Snapshot:** Generate `.golden.md` using the CLI's snapshot flag.
3. **Analyze:** Use synthetic tests to reproduce the failure geometry.
4. **Refactor:** Apply fixes to extraction heuristics.
5. **Verify:** Run full test suite; verify diffs against golden master.

## 4. Task Breakdown for Future Scenarios

### 4.1 Regression Suite Expansion
- [ ] **Task 1:** Create `testdata/regression/` organizational structure (by doc type).
- [ ] **Task 2:** Add Git LFS or storage strategy if data volume exceeds 100MB.
- [ ] **Task 3:** Implement automated snapshot-drift detection in CI.

### 4.2 Synthetic Test Harness Development
- [ ] **Task 1:** Define `MockPDFContent` struct representing text, font, and coordinate primitives.
- [ ] **Task 2:** Implement `MockPDF.ToLedongPage()` adapter for seamless engine integration.
- [ ] **Task 3:** Build parameterized test runners for common geometric failure modes (e.g., stacked text, overlapping tables).

### 4.3 Automated Quality Gates
- [ ] **Task 1:** Integrate `StructuralValidator` alerts into the existing CLI report summary.
- [ ] **Task 2:** Implement CI threshold for Entropy filtering; fail builds if new documents produce > 20% high-entropy output.
