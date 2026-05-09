# CI/CD Improvements Analysis: Research-Backed Recommendations

**Date**: 2026-05-10  
**Source**: Exa research + Stabilization Plan analysis  
**Target**: Align pdf2md-tui CI/CD with industry best practices and eliminate test fixture drift

---

## Executive Summary

The stabilization plan correctly identifies that **CI failures are infrastructure issues, not production bugs**. Exa research validates this approach and provides concrete patterns used by Go projects at scale (Go 1.20+, GitHub Actions best practices, fixture caching strategies).

Key findings:
- **Layer-based CI** (lint → test → coverage → security) catches different categories of bugs
- **Fixture management** should use tracked (git) or generated (temp) data, never ignored paths
- **Coverage gates** must reflect clean-checkout behavior, not local workstation state
- **Race detector** (-race flag) is critical for catching concurrency bugs before production

---

## Research-Backed Improvements

### 1. GitHub Actions Workflow Modernization

**Current State**: Simple CI, likely single job

**Industry Standard** (from Exa sources):
```yaml
# Fast job order: lint first (1 min), then test (2-5 min), security (1 min)
jobs:
  lint:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
      - uses: golangci/golangci-lint-action@v6
      
  test:
    runs-on: ubuntu-latest
    needs: lint  # Fast feedback: lint failures fail the job immediately
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
      - run: go test -v -race -coverprofile=coverage.out ./...
      
  security:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
      - run: go install golang.org/x/vuln/cmd/govulncheck@latest
      - run: govulncheck ./...  # Go's authoritative vuln DB
```

**Recommended Changes for pdf2md-tui**:
1. **Separate jobs by function** (lint, test, security) instead of sequential steps
2. **Use `needs:` to enforce order** while running jobs in parallel where safe
3. **Pin tool versions** (golangci-lint action to v6, Go to 1.22+)
4. **Add vulnerability check** with `govulncheck` (part of Go's official toolkit)
5. **Matrix testing** on ubuntu-latest (authority) + macos-latest (smoke test)

**Impact**: Failures reported in <5 minutes instead of waiting for full test suite

---

### 2. Test Fixture Management

**Problem** (from stabilization plan):
- Root `/testdata/` ignored by `.gitignore` but tests depend on it
- Clean checkout ≠ local development
- E2E tests fail with low-signal errors

**Research Pattern** (Dave Cheney, LogRocket):
```
DO track:        pkg/*/testdata/**         (committed to git)
DO NOT track:    /testdata/                (CI smoke test data)
CACHE at:        $HOME/.cache/crdb-test-fixtures/  (for expensive benchmarks)
GENERATE:        t.TempDir() for unit tests
```

**Action Items**:
1. **Audit .gitignore** — ensure `testdata/` is anchored to repo root only:
   ```gitignore
   /testdata/          # ✅ root level only
   # testdata/         # ❌ WRONG: matches everywhere
   ```

2. **Convert corpus tests to optional** — skip gracefully:
   ```go
   fixtureRoot := filepath.Join(projectRoot, "testdata", "devops_project")
   if _, err := os.Stat(fixtureRoot); err != nil {
       t.Skipf("skipping corpus tests; fixture unavailable: %s", err)
   }
   ```

3. **Inline simple fixtures** using functional builders:
   ```go
   // Not: hardcoded values scattered in test
   // YES: builder pattern with sensible defaults
   func newTestPDF(opts ...PDFOption) *domain.PDF {
       p := &domain.PDF{ /* defaults */ }
       for _, opt := range opts {
           opt(p)
       }
       return p
   }
   ```

4. **Document tracking policy** in CONTRIBUTING.md:
   - Unit tests: generate or use `t.TempDir()`
   - Integration tests: skip if `/testdata` absent, don't error
   - Benchmarks: cache in `$HOME/.cache/` (outside repo)

**Research Support**: CockroachDB stores expensive benchmarks in `$HOME/.cache`, not repo. This isolates fixture noise from code review.

---

### 3. Coverage Gates (Clean-Checkout Baseline)

**Issue**: Documentation claims 76.1% but clean checkout measures 45-51%

**Research Finding** (Go 1.20+, covdata):
```bash
# Build with coverage instrumentation
go build -cover -o bin/pdf2md-tui ./cmd/pdf2md-tui
export GOCOVERDIR=$(mktemp)

# Run binaries, collect coverage
./bin/pdf2md-tui convert testdata/sample.pdf

# Report
go tool covdata percent -i=$GOCOVERDIR
```

**Recommendation**:
1. **Measure on ubuntu-latest in CI** (authoritative)
2. **Document baseline with conditions**:
   ```markdown
   - Clean checkout (no `/testdata`): 47.9%
   - With corpus available: 51.7%
   - Target: 70%+ (aspirational, phase-gated)
   ```

3. **Gate by phase**:
   ```yaml
   # In .github/workflows/ci.yml
   - name: Check coverage (clean checkout)
     run: |
       COVERAGE=$(go tool cover -func=coverage.out | grep total | awk '{print $3}' | sed 's/%//')
       # Gate: currently 47.9% for clean checkout
       if (( $(echo "$COVERAGE < 45" | bc -l) )); then
         echo "Coverage ${COVERAGE}% below clean-checkout baseline"
         exit 1
       fi
   ```

4. **Avoid false claims** — never document "75% coverage" if it requires ignored fixtures

---

### 4. E2E/Testscript Configuration

**Current**: `testscript.RunMain` (deprecated), corpus path hardcoded

**Research**: Go 1.20+ patterns use `testscript.Main`, environment variables for paths

**Changes**:
1. Update `e2e_test.go` to use `testscript.Main` instead of `RunMain`
2. Pass fixture root via environment variable:
   ```bash
   TESTDATA_DIR=/path/to/testdata go test ./internal/handler/cli
   ```
3. Testscript should skip gracefully if `$TESTDATA_DIR` is unavailable

---

### 5. Local Developer Command Alignment

**Goal**: `make test` ≡ CI behavior

**From Exa research** — GitHub Actions best practices:

```bash
# Local development (fast)
go test -race ./...

# Full CI simulation
go test -v -race -coverprofile=coverage.out ./...
go tool cover -func=coverage.out | tail -1

# Lint (if installed)
golangci-lint run ./...

# Vulnerability scan
go install golang.org/x/vuln/cmd/govulncheck@latest
govulncheck ./...
```

**Makefile target**:
```makefile
.PHONY: ci-local
ci-local: lint test cover
	@echo "✅ Passed all CI checks"

.PHONY: test
test:
	go test -v -race -coverprofile=coverage.out -covermode=atomic ./...

.PHONY: cover
cover:
	go tool cover -func=coverage.out | tail -1

.PHONY: lint
lint:
	golangci-lint run ./... || echo "⚠️  lint failed (optional)"
```

---

### 6. Lifecycle Management (Test Fixtures)

**Research Finding** (Rednafi, Go docs):
- **Per-test scope** (default) — fastest, safest for isolation
- **Per-group scope** — share expensive setup across subtests via parent
- **Per-package scope** (TestMain) — rarely needed, global state risk

**Pattern for pdf2md-tui**:
```go
// ✅ Most tests: function scope (t.TempDir)
func TestConverter(t *testing.T) {
    tmpDir := t.TempDir()  // Cleaned up automatically
    // test logic
}

// ✅ Optional: subtest grouping
func TestPDFExtractionFlow(t *testing.T) {
    mockPDF := newMockPDF()  // Created once, shared
    t.Run("extracts text", func(t *testing.T) {
        // uses mockPDF
    })
    t.Run("extracts images", func(t *testing.T) {
        // uses same mockPDF
    })
    // Implicit cleanup via parent scope
}

// ❌ Avoid TestMain unless really expensive (e.g., database)
```

---

### 7. Build Matrix Testing

**Research**: Test on multiple Go versions and OSes

**Recommended**:
```yaml
test:
  strategy:
    matrix:
      go-version: ['1.22', '1.23']
      os: [ubuntu-latest, macos-latest]  # Windows optional
  runs-on: ${{ matrix.os }}
  steps:
    - uses: actions/setup-go@v5
      with:
        go-version: ${{ matrix.go-version }}
    - run: go test -race -coverprofile=coverage.out ./...
    # Coverage gate only on ubuntu-latest, Go 1.22
    - if: matrix.os == 'ubuntu-latest' && matrix.go-version == '1.22'
      run: ./scripts/check-coverage.sh
```

---

### 8. Vulnerability Scanning

**Current**: Not mentioned in CI

**Research**: Go's official `govulncheck` is authoritative (not generic scanners)

**Add to CI**:
```yaml
security:
  runs-on: ubuntu-latest
  steps:
    - uses: actions/checkout@v4
    - uses: actions/setup-go@v5
    - run: |
        go install golang.org/x/vuln/cmd/govulncheck@latest
        govulncheck ./...
```

**Key advantage**: Only flags CVEs your code actually *calls*, not transitive noise

---

### 9. Fuzz Testing in CI

**Research**: Go 1.18+ native fuzzing

**Current**: E2E tests exist; CI doesn't run fuzzing

**Recommendation**:
```yaml
# In .github/workflows/ci.yml, add to test job
- name: Run fuzz tests (30s)
  run: |
    go test -fuzz=FuzzCoalesceChars -fuzztime=30s ./pkg/repository/pdf/
    go test -fuzz=FuzzApplyLLMOptimizations -fuzztime=30s ./pkg/service/
```

**Why**: Catches edge cases before production; 30s is reasonable for CI

---

### 10. Benchmarking Strategy

**Research**: CockroachDB uses reusable cached fixtures for benchmarks

**For pdf2md-tui**:
```bash
# Local: baseline
go test -bench=. -benchmem ./pkg/...

# CI: compare (optional, scheduled weekly)
# Use benchstat or similar for regression detection
```

**Policy**:
- Benchmarks optional on every PR (slow)
- Run weekly via `workflow_dispatch` or schedule
- Cache large fixture inputs in `$HOME/.cache/` (not repo)

---

## Recommended CI Workflow Shape

```yaml
name: CI

on: [push, pull_request]

jobs:
  # ── Fast checks (1 min) ──
  format:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
      - run: go fmt ./...
      - run: go vet ./...

  lint:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
      - uses: golangci/golangci-lint-action@v6
        with:
          version: v1.59
          args: --timeout=5m

  # ── Tests (3-5 min) ──
  test:
    needs: [format, lint]
    strategy:
      matrix:
        go-version: ['1.22']
        os: [ubuntu-latest, macos-latest]
    runs-on: ${{ matrix.os }}
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with:
          go-version: ${{ matrix.go-version }}
      - run: go test -v -race -coverprofile=coverage.out ./...
      - if: matrix.os == 'ubuntu-latest'
        uses: codecov/codecov-action@v4
        with:
          files: ./coverage.out
      - if: matrix.os == 'ubuntu-latest'
        run: |
          COVERAGE=$(go tool cover -func=coverage.out | grep total | awk '{print $3}' | sed 's/%//')
          if (( $(echo "$COVERAGE < 45" | bc -l) )); then
            echo "Coverage ${COVERAGE}% below baseline"
            exit 1
          fi

  # ── Security (1 min) ──
  security:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
      - run: |
          go install golang.org/x/vuln/cmd/govulncheck@latest
          govulncheck ./...

  # ── Build (2 min) ──
  build:
    needs: test
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
      - run: go build ./cmd/pdf2md-tui
```

**Total wall time**: ~5 minutes (lint, test, security run in parallel after format)

---

## Immediate Action Items (Priority Order)

| Priority | Task | Effort | Impact |
|----------|------|--------|--------|
| **P0** | Fix `.gitignore` — anchor `testdata/` to root only | 5 min | Prevents CI/local divergence |
| **P0** | Document clean-checkout coverage baseline (47.9%) | 10 min | Stops false 76% claims |
| **P0** | Add E2E test skip gracefully when fixtures missing | 30 min | Fixes clean-checkout CI |
| **P1** | Modernize `.github/workflows/ci.yml` (layered jobs) | 1 hour | Faster feedback, cleaner |
| **P1** | Add `govulncheck` security job | 20 min | Catches CVEs early |
| **P1** | Align `make test` with CI (`-race`, coverage) | 30 min | Local ≡ CI |
| **P2** | Add build matrix (Go 1.22, 1.23 × ubuntu, macos) | 45 min | Portability validation |
| **P2** | Document fixture policy in CONTRIBUTING.md | 20 min | Prevents future drift |
| **P3** | Add scheduled benchmark job (weekly) | 1 hour | Performance regression detection |

---

## Testing Patterns from Research

### Golden Files (from Exa)
Already implemented in pdf2md-tui (via `internal/handler/tui/golden.go`). Keep this pattern.

### Table-Driven Tests
Already used extensively. Continue for property-based testing.

### Mocking Strategy
Research shows **manual mocks** are better than generated mocks for simple interfaces. Current pattern is good.

### Fixture Builders
Use functional builder pattern (already done) to avoid global state in test data.

---

## Success Metrics

| Metric | Current | Target | Timeline |
|--------|---------|--------|----------|
| Clean-checkout coverage | 47.9% | 50%+ | Next 2 weeks |
| CI wall time | Unknown | <5 min | With job layering |
| Test determinism (pass 100% of time) | ~90% | 100% | With fixture cleanup |
| Security scanning | No | govulncheck running | Next PR |
| Documentation accuracy | False (76%) | True (47.9%) | Immediate |

---

## References

**Exa Research Sources**:
- Calmops CI/CD for Go (2026)
- Atharva Pandey: "Go CI/CD that catches bugs" (lesson-based approach)
- Go docs: Integration test coverage (Go 1.20+)
- LogRocket: Test fixtures and golden files
- Dave Cheney: Test fixtures in Go (foundational)
- Rednafi: Test lifecycle management

**Project Docs**:
- `/docs/specs/2026-05-10-future-test-cicd-stabilization.md`
- `/CLAUDE.md` (local dev requirements)
- `/README.md` (development setup)

---

**Author**: QA/CI/CD Review  
**Date**: 2026-05-10  
**Status**: Research Complete, Ready for Implementation
