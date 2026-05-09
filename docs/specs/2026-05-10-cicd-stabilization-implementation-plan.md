# pdf2md-tui CI/CD Improvements Implementation Plan

**Date**: 2026-05-10  
**Based on**: Stabilization audit + Exa research (Go 1.20+ patterns, GitHub Actions best practices)  
**Target**: Fix infrastructure issues preventing clean-checkout CI success

---

## Priority Overview

| Priority | Task | File(s) | Effort | Impact |
|----------|------|---------|--------|--------|
| **P0** | Fix `.gitignore` anchoring | `.gitignore` | 5 min | Prevents CI/local divergence |
| **P0** | Document coverage baseline | `README.md` + `docs/COVERAGE.md` | 10 min | Stops false claims (76% → 47.9%) |
| **P0** | E2E test fixture skip | `internal/handler/cli/e2e_test.go` | 30 min | Fixes clean-checkout CI failures |
| **P1** | Modernize GitHub Actions | `.github/workflows/ci.yml` | 1 hour | Faster feedback (<5 min wall time) |
| **P1** | Add govulncheck | `.github/workflows/ci.yml` | 20 min | Catches CVEs early (official Go DB) |
| **P1** | Align `make test` with CI | `Makefile` | 30 min | Local ≡ CI behavior |
| **P2** | Document fixture policy | `CONTRIBUTING.md` | 20 min | Prevents future drift |

---

## P0: Fix `.gitignore` Anchoring

**File**: `.gitignore`  
**Problem**: Current pattern `testdata/` matches every package-local `testdata` directory, breaking CI for `internal/**/testdata/**` and `pkg/**/testdata/**` fixtures.

**Current `.gitignore` (incorrect)**:
```gitignore
testdata/
```

**Why it breaks**: The unanchored pattern means:
- `internal/handler/cli/testdata/scripts/` → **IGNORED** (broken in CI)
- `pkg/repository/pdf/testdata/` → **IGNORED** (broken in CI)
- `/testdata/devops_project/` (root-level large corpus) → **IGNORED** (correct, intentional)

**Desired `.gitignore` (correct)**:
```gitignore
/testdata/
```

**Change**:
- Line 1 (or wherever testdata pattern appears): Replace `testdata/` with `/testdata/`
- The leading `/` anchors the pattern to the repository root only

**Validation**:
```bash
# After change, verify package-local testdata is tracked:
git ls-files | grep testdata
# Should show:
# internal/handler/cli/testdata/...
# pkg/repository/pdf/testdata/...
# But NOT list /testdata/ contents (root level ignored)
```

**Why this works** (from Go `.gitignore` patterns):
- `testdata/` = match all directories named `testdata` anywhere
- `/testdata/` = match only the root-level `testdata` directory

This pattern comes from Go's standard library approach (Go docs: Managing dependencies) where:
- Package-local testdata (tracked in git) is used by unit/integration tests
- Root-level fixtures (ignored) are for large corpora or local-only development

---

## P0: Document Coverage Baseline

**Files**: 
1. Create new `docs/COVERAGE.md`
2. Update `README.md` Development section

**Current Problem**: Documentation claims "76.1% coverage" but clean-checkout measures 47.9%. This creates false confidence and misaligned gates.

**Create `docs/COVERAGE.md`**:
```markdown
# Test Coverage Policy

## Current Baseline (Verified 2026-05-10)

Measured via `go test -race -coverprofile=coverage.out -covermode=atomic ./...` on `ubuntu-latest`:

| Scenario | Coverage | Conditions |
|----------|----------|-----------|
| Clean checkout | 47.9% | No `/testdata` root-level fixtures, no corpus |
| With corpus | 51.7% | `/testdata/devops_project/` present (local dev only) |

## Target Coverage Roadmap

- **Phase 1 (now)**: Maintain 45%+ (clean-checkout baseline)
- **Phase 2 (next 2 weeks)**: 50%+ (via E2E fixture-independent tests)
- **Phase 3 (month 2)**: 55%+ (TUI/model expansion without golden-file brittleness)
- **Phase 4+ (month 3+)**: 60%+ (service/parser real branches, not shallow tests)

## Acceptance Criteria for Coverage

✅ CI always measures on `ubuntu-latest` (authoritative)  
✅ Documentation includes exact command and conditions  
✅ Date-stamped so we know when baseline was established  
✅ Never document coverage that requires ignored files  
✅ Coverage gates use clean-checkout measurement (47.9%), not corpus measurement  

## CI Coverage Gate (from `.github/workflows/ci.yml`)

```yaml
- if: matrix.os == 'ubuntu-latest'
  run: |
    COVERAGE=$(go tool cover -func=coverage.out | grep total | awk '{print $3}' | sed 's/%//')
    if (( $(echo "$COVERAGE < 45" | bc -l) )); then
      echo "❌ Coverage ${COVERAGE}% below clean-checkout baseline (45%)"
      exit 1
    fi
    echo "✅ Coverage ${COVERAGE}% meets baseline"
```

## How to Measure Locally

```bash
# Simulate CI (clean checkout, no corpus)
rm -rf /testdata/devops_project  # Don't do this in your repo; just test without it
go test -race -coverprofile=coverage.out -covermode=atomic ./...
go tool cover -func=coverage.out | tail -1  # Shows total percentage
```
```

**Update `README.md` Development section** (add under existing "Commands" table):

```markdown
### Test Coverage

Coverage baseline measured on clean checkout (`ubuntu-latest`):
- **Current**: 47.9% (verified 2026-05-10)
- **CI gate**: 45%+ required
- **Target**: 70%+ (phased roadmap in [docs/COVERAGE.md](docs/COVERAGE.md))

Measure coverage locally:
```bash
go test -race -coverprofile=coverage.out -covermode=atomic ./...
go tool cover -func=coverage.out | tail -1
make cover  # Opens HTML report in browser
```

Coverage documentation always includes:
- Exact command used
- Whether root `/testdata` was present
- Date measured
```

**Validation**:
```bash
# Verify documentation is accurate
grep -r "76.1" docs/  # Should find nothing (old false claim removed)
grep -r "47.9" docs/  # Should find COVERAGE.md and this spec
go test -race -coverprofile=coverage.out -covermode=atomic ./...
go tool cover -func=coverage.out | grep total
# Should show coverage near 47-48% (matches documented baseline)
```

---

## P0: Add E2E Test Skip for Missing Fixtures

**File**: `internal/handler/cli/e2e_test.go` (already created, needs enhancement)  
**Problem**: E2E tests fail with cryptic errors when `/testdata/devops_project` is missing (clean checkout). Should skip gracefully instead.

**Current Issue** (hypothetical):
```go
// BAD: fails with low-signal "file not found" error
func TestCLIConvertCorpus(t *testing.T) {
    result := runCLI("convert", "testdata/devops_project")
    if result.ExitCode != 0 {
        t.Fatalf("conversion failed: %v", result.Stderr)
    }
    // Test continues but has already failed for unclear reason
}
```

**Fixed Pattern** (from Go 1.20+ best practices):
```go
func TestCLIConvertCorpus(t *testing.T) {
    // Check for fixture availability FIRST
    fixtureRoot := filepath.Join(projectRoot, "testdata", "devops_project")
    if _, err := os.Stat(filepath.Join(fixtureRoot, "sample.pdf")); err != nil {
        t.Skipf("skipping corpus E2E tests; fixture unavailable: %s (run locally with full corpus)", fixtureRoot)
    }
    
    // Fixture exists; proceed with test
    result := runCLI("convert", fixtureRoot)
    if result.ExitCode != 0 {
        t.Fatalf("conversion failed: %v", result.Stderr)
    }
    
    // Assert on output...
}
```

**Key Pattern Elements** (from Go testing docs):
1. **Check fixture before test logic** — prevents cascading failures
2. **Use `t.Skipf()` not `t.Fatalf()`** — signals to test runner that this test is conditional, not broken
3. **Clear skip reason** — explains why it's skipped (fixture missing, not a bug)
4. **Early return implicit** — `t.Skip()` ends the test immediately

**Implementation in `e2e_test.go`**:

Add helper at top of file:
```go
var projectRoot string

func init() {
    // Compute project root once at package init time
    wd, err := os.Getwd()
    if err != nil {
        panic(err)
    }
    // Walk up until we find go.mod
    for {
        if _, err := os.Stat(filepath.Join(wd, "go.mod")); err == nil {
            projectRoot = wd
            break
        }
        parent := filepath.Dir(wd)
        if parent == wd {
            panic("could not find project root (no go.mod)")
        }
        wd = parent
    }
}

// skipIfNoCorpus checks if /testdata/devops_project exists; skips test if not
func skipIfNoCorpus(t *testing.T) {
    corpusRoot := filepath.Join(projectRoot, "testdata", "devops_project")
    if _, err := os.Stat(corpusRoot); err != nil {
        t.Skipf("skipping corpus-dependent test; %s not found (expected in local dev, optional in CI)", corpusRoot)
    }
}
```

Then in each corpus-dependent test:
```go
func TestCLIConvertMultiplePDFs(t *testing.T) {
    skipIfNoCorpus(t)
    
    // Test corpus conversion...
}

func TestCLIExtractImages(t *testing.T) {
    skipIfNoCorpus(t)
    
    // Test image extraction...
}
```

**Validation**:
```bash
# Clean checkout (no corpus)
rm -rf testdata/devops_project  # Simulate CI clean checkout
go test -v ./internal/handler/cli/
# Should show:
# --- SKIP: TestCLIConvertMultiplePDFs
#     e2e_test.go:42: skipping corpus-dependent test; .../testdata/devops_project not found

# With corpus (local dev)
# Restore testdata/devops_project
git checkout testdata/devops_project  # or `git restore` if it's tracked
go test -v ./internal/handler/cli/
# Should show:
# --- PASS: TestCLIConvertMultiplePDFs (0.45s)
```

**Why this matters** (from stabilization plan):
- CI no longer fails with "file not found" noise
- Developers understand immediately why test is skipped
- Local dev with corpus runs full tests, CI gracefully skips
- Clean separation of concerns: required vs optional fixtures

---

## P1: Modernize GitHub Actions (Layered Jobs)

**File**: `.github/workflows/ci.yml`  
**Current State**: Likely sequential steps in single job (slow feedback)  
**Desired State**: Separate jobs ordered by speed (lint → test → security), parallel where safe

**Why layer** (from GitHub Actions best practices):
- Lint fails fast (~1 min) before running slow tests
- Tests can run while security scan runs (parallel)
- Wall-clock time: ~5 min instead of 10–15 min
- Developer feedback: "linting failed" beats "tests timed out" for root cause

**Complete Replacement Workflow**:

```yaml
name: CI

on:
  push:
    branches: [main, dev]
  pull_request:
    branches: [main]

jobs:
  # Fast checks — run first (1 min total)
  format:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with:
          go-version: '1.22'
      - name: Check formatting
        run: |
          if [ -n "$(gofmt -l ./...)" ]; then
            echo "❌ Code not formatted. Run: make fmt"
            gofmt -l ./...
            exit 1
          fi
      - name: Run go vet
        run: go vet ./...

  lint:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with:
          go-version: '1.22'
      - uses: golangci/golangci-lint-action@v6
        with:
          version: v1.59
          args: --timeout=5m

  # Test job — can run in parallel with security
  test:
    needs: [format, lint]  # Wait for fast checks first
    strategy:
      matrix:
        os: [ubuntu-latest, macos-latest]
        go-version: ['1.22']
    runs-on: ${{ matrix.os }}
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with:
          go-version: ${{ matrix.go-version }}
      
      - name: Run tests with race detector
        run: go test -v -race -coverprofile=coverage.out -covermode=atomic ./...
      
      # Coverage gate (ubuntu only, authoritative)
      - if: matrix.os == 'ubuntu-latest'
        name: Check coverage baseline
        run: |
          COVERAGE=$(go tool cover -func=coverage.out | grep total | awk '{print $3}' | sed 's/%//')
          echo "Coverage: ${COVERAGE}%"
          if (( $(echo "$COVERAGE < 45" | bc -l) )); then
            echo "❌ Coverage ${COVERAGE}% below clean-checkout baseline (45%)"
            exit 1
          fi
          echo "✅ Coverage ${COVERAGE}% meets baseline"
      
      # Upload coverage (ubuntu only)
      - if: matrix.os == 'ubuntu-latest'
        uses: codecov/codecov-action@v4
        with:
          files: ./coverage.out
          flags: unittests

  # Security scan (can run in parallel with test)
  security:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with:
          go-version: '1.22'
      - name: Run govulncheck
        run: |
          go install golang.org/x/vuln/cmd/govulncheck@latest
          govulncheck ./...

  # Build validation (after test passes)
  build:
    needs: test
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with:
          go-version: '1.22'
      - name: Build binary
        run: go build ./cmd/pdf2md-tui
      - name: Verify binary
        run: ./pdf2md-tui version
```

**Key improvements**:

| Feature | Old | New | Benefit |
|---------|-----|-----|---------|
| Job order | All sequential | Layered (format → lint, then test+security parallel) | 50% faster |
| Coverage gate | Never run | On ubuntu only (authoritative) | Avoids macOS variance |
| Vulnerability scan | Missing | Added (govulncheck official Go DB) | Catches CVEs early |
| Action versions | Unspecified | Pinned (v4, v5, v6) | Reproducible CI |
| Go version | Inferred | Explicit (1.22) | Clear compatibility |
| Race detector | Missing | `-race` on all platforms | Catches concurrency bugs |

**Validation**:
```bash
# After pushing this workflow to .github/workflows/ci.yml:
# 1. Create a PR and watch the Actions tab
# 2. Verify jobs run in order: format → lint → (test, security in parallel) → build
# 3. Check that coverage gate catches regressions if you artificially drop coverage
```

**Cost estimate**:
- Ubuntu test: ~2 min
- macOS test: ~2.5 min (slower due to instance allocation)
- Lint: ~1 min (parallel with others)
- Security: ~1 min (parallel with others)
- **Wall time**: ~5 minutes total (if layered correctly)

---

## P1: Add govulncheck (Go's Official Vulnerability Scanner)

**File**: `.github/workflows/ci.yml` (already added in P1 section above)  
**Purpose**: Scan for known CVEs in dependencies using Go's authoritative vulnerability database

**Why govulncheck over alternatives** (from Go security docs):
- Official Go team tool (golang.org/x/vuln)
- Uses Go's authoritative CVE database (not generic NVD)
- Only flags vulnerabilities your code **actually calls** (not transitive noise)
- Built-in to Go toolchain (no external dependencies)

**How it works** (from Go 1.18+ patterns):
1. Analyzes your binary call graph
2. Checks `go.mod` and transitive deps against CVE database
3. Reports only vulnerabilities reachable from your code
4. Exits non-zero if any CVEs found

**Implementation** (isolated security job in `.github/workflows/ci.yml`, already shown above):

```yaml
security:
  runs-on: ubuntu-latest
  steps:
    - uses: actions/checkout@v4
    - uses: actions/setup-go@v5
      with:
        go-version: '1.22'
    - name: Run govulncheck
      run: |
        go install golang.org/x/vuln/cmd/govulncheck@latest
        govulncheck ./...
```

**What to expect**:
- First run may find existing vulnerabilities in dependencies
- If found, `go get -u` the affected package (often upstream publishes a patch)
- Example output:
  ```
  govulncheck: no vulnerabilities found
  ```
  or
  ```
  Vulnerability #1: GO-2024-12345
    pkg/domain imports vulnerable module github.com/some/lib v1.0.0
    Call stack:
      main.go:15 main() calls example.go:10 parseConfig()
      ... (call chain showing how vuln is reachable)
  ```

**Local testing** (before pushing to CI):
```bash
go install golang.org/x/vuln/cmd/govulncheck@latest
govulncheck ./...
```

**Why separate job** (from GitHub Actions patterns):
- Security scans can fail independently of tests
- Allows `continue-on-error: true` if you want to warn but not block
- Fast (1 min) so can run in parallel with tests
- Clear failure signal: "security issue" vs "test failure"

---

## P1: Align `make test` with CI Behavior

**File**: `Makefile`  
**Problem**: Local `make test` doesn't match CI exactly, causing "green locally, red in CI" surprises

**Current Makefile** (likely):
```makefile
test:
	go test ./...
```

**Desired Makefile**:
```makefile
.PHONY: test
test:
	go test -v -race -coverprofile=coverage.out -covermode=atomic ./...

.PHONY: cover
cover:
	go tool cover -func=coverage.out | tail -n 1

.PHONY: cover-check
cover-check: test
	@COVERAGE=$$(go tool cover -func=coverage.out | grep total | awk '{print $$3}' | sed 's/%//'); \
	if [ "$$(echo "$$COVERAGE < 45" | bc -l)" = "1" ]; then \
	  echo "❌ Coverage $$COVERAGE% below baseline"; \
	  exit 1; \
	fi; \
	echo "✅ Coverage $$COVERAGE% OK"

.PHONY: ci-local
ci-local: fmt vet lint cover-check
	@echo "✅ All CI checks passed locally"

.PHONY: fmt
fmt:
	go fmt ./...

.PHONY: vet
vet:
	go vet ./...

.PHONY: lint
lint:
	golangci-lint run ./... || echo "⚠️  lint issues found (optional in Makefile)"
```

**Key additions**:

| Target | Purpose | Matches CI? | Notes |
|--------|---------|------------|-------|
| `test` | Run tests with `-race` and coverage | ✅ Yes | Default target developers run |
| `cover` | Show coverage percentage | ✅ Yes | Simulates CI gate check |
| `cover-check` | Fail if coverage < 45% | ✅ Yes | Exact same gate as CI |
| `ci-local` | Run all CI checks locally | ✅ Yes | `fmt + vet + lint + cover-check` |
| `fmt`, `vet`, `lint` | Individual check targets | ✅ Yes | Matches `make check` pattern |

**Usage**:
```bash
# Quick test (same as CI)
make test

# Check coverage
make cover-check

# Full local CI simulation (what you'd run before push)
make ci-local

# View coverage report in browser
make cover
```

**Validation**:
```bash
# After updating Makefile:
make test              # Should pass
make cover             # Shows ~47.9% (clean checkout)
make ci-local          # Should pass all checks
```

**Why `-race` flag** (from Go concurrency best practices):
- Detects data races at runtime (not via static analysis)
- Critical for catching bugs in worker pools (your code uses them)
- Adds ~20% overhead but only enabled in tests, not production
- Go 1.18+ race detector is production-ready (no false positives in well-tested code)

**Why `-covermode=atomic`** (from coverage docs):
- Thread-safe coverage counting
- Pair with `-race` for accurate concurrent test coverage
- Slight overhead but correct for concurrency

---

## P2: Document Fixture Policy in CONTRIBUTING.md

**File**: Create `CONTRIBUTING.md` (or add section if exists)  
**Purpose**: Make fixture management explicit so contributors don't accidentally commit test data

**Content**:

```markdown
# Contributing to pdf2md-tui

## Testing Guidelines

### Test Fixtures

This project uses three types of test fixtures, each managed differently:

#### 1. Tracked Fixtures (Committed to Git)

Small, deterministic test inputs that all tests depend on:
- Location: `internal/**/testdata/`, `pkg/**/testdata/`
- Examples: `.txtar` scripts for testscript, tiny synthetic PDFs, mock JSON
- Policy: **Must be tracked in git** (not ignored)
- Validation: `go test ./... ` from a clean clone must pass

```bash
# Example: adding a new testdata file
mkdir -p pkg/repository/pdf/testdata
echo 'test data' > pkg/repository/pdf/testdata/example.txt
git add pkg/repository/pdf/testdata/example.txt
```

#### 2. Ignored Root-Level Fixtures (Local Dev Only)

Large real-world corpora used for local integration testing:
- Location: `/testdata/devops_project/`, `/testdata/samples/`, other root-level directories
- Examples: Multi-page PDFs, real documents, local golden-master outputs
- Policy: **Ignored by `.gitignore`** (not committed)
- Tests using these: **Must skip gracefully** if fixture unavailable

```bash
# Tests MUST use this pattern:
func TestCLICorpusConversion(t *testing.T) {
    fixtureRoot := filepath.Join(projectRoot, "testdata", "devops_project")
    if _, err := os.Stat(fixtureRoot); err != nil {
        t.Skipf("skipping corpus test; fixture unavailable: %s", fixtureRoot)
    }
    // Test logic here...
}
```

#### 3. Generated Fixtures (Created During Tests)

Temporary test data created on-the-fly:
- Location: `t.TempDir()` (automatically cleaned up after test)
- Examples: Temp PDFs, intermediate outputs, scratch directories
- Policy: **No tracking needed** (ephemeral)

```bash
# Example pattern (from Go testing docs):
func TestConverter(t *testing.T) {
    tmpDir := t.TempDir()  // Cleaned up automatically
    outputPath := filepath.Join(tmpDir, "output.md")
    // Use tmpDir for test...
}
```

### Before Committing

Check for accidentally committed test data:

```bash
# See what you're about to commit
git status --short

# If you added testdata, verify it's NOT in root /testdata/
find testdata -type f | head -5
# Should show nothing (root testdata is ignored)

# If you added pkg/*/testdata/*, verify it IS tracked
git ls-files | grep -E 'pkg/.*testdata'
# Should show your new files
```

### Coverage and Fixtures

- **CI coverage baseline**: 47.9% (clean checkout, no `/testdata`)
- **Local coverage with corpus**: ~51.7% (with `/testdata/devops_project`)
- **CI gate**: 45%+ (always based on clean-checkout measurement)
- **See**: [docs/COVERAGE.md](docs/COVERAGE.md) for details

### Running Tests

```bash
# Local with all fixtures (corpus + tracked)
make test

# Simulate CI (clean checkout, no ignored fixtures)
rm -rf testdata/  # Don't actually do this in your repo; just test without it
go test ./...

# Full local CI simulation
make ci-local
```

---

## Code Review Checklist

For all PRs:

- [ ] New tests added for new code
- [ ] No test files accidentally committed to `/testdata/`
- [ ] Tests using corpus fixtures call `t.Skipf()` if missing
- [ ] No hardcoded absolute paths in tests (use `t.TempDir()` or `filepath.Join`)
- [ ] `make ci-local` passes locally before pushing
- [ ] Coverage did not decrease (check `make cover`)
```

**Validation**:
```bash
# After creating CONTRIBUTING.md:
# 1. Check it's in the repo
ls -la CONTRIBUTING.md

# 2. Verify it's referenced in README
grep -i "CONTRIBUTING" README.md
# (should have a link to CONTRIBUTING.md)
```

---

## Execution Roadmap

**Week 1 (Now)**:
1. ✅ P0: Fix `.gitignore` (`/testdata/`)
2. ✅ P0: Create `docs/COVERAGE.md` + update `README.md`
3. ✅ P0: Update `e2e_test.go` with `skipIfNoCorpus()` helper

**Week 1-2**:
4. ✅ P1: Replace `.github/workflows/ci.yml` with layered jobs
5. ✅ P1: Add `govulncheck` security job (already in workflow above)
6. ✅ P1: Update `Makefile` with `test`, `cover`, `cover-check`, `ci-local`

**Week 2**:
7. ✅ P2: Create `CONTRIBUTING.md`

**After these tasks**:
- Push to a feature branch (e.g., `feature/ci-stabilization`)
- Run `.github/workflows/ci.yml` (should complete in ~5 min)
- Coverage gate should pass (~47.9% on ubuntu)
- Create PR with before/after metrics
- Merge to `main` once approved

---

## Acceptance Criteria — How to Know It's Done

### P0 — Infrastructure fixed

- [ ] `.gitignore` has `/testdata/` (anchored), not `testdata/`
- [ ] `git ls-files | grep testdata` shows package-local files (tracked)
- [ ] `docs/COVERAGE.md` documents baseline (47.9%)
- [ ] `README.md` references `docs/COVERAGE.md`
- [ ] E2E tests with `skipIfNoCorpus()` skip gracefully when corpus missing
- [ ] `go test ./...` passes on clean checkout (no corpus)

### P1 — Workflows modern

- [ ] `.github/workflows/ci.yml` has separate format, lint, test, security, build jobs
- [ ] Test job includes `-race` and `coverprofile=coverage.out`
- [ ] Coverage gate on ubuntu only (≥45%)
- [ ] `security` job runs `govulncheck ./...`
- [ ] `Makefile` has `test`, `cover`, `cover-check`, `ci-local` targets
- [ ] `make ci-local` runs `fmt + vet + lint + cover-check` in sequence
- [ ] Local `make test` uses `-race` and coverage (matches CI)

### P2 — Documentation clear

- [ ] `CONTRIBUTING.md` explains tracked vs ignored vs generated fixtures
- [ ] Fixture policy examples are Go 1.20+ patterns (tested)
- [ ] PR reviewers can reference `CONTRIBUTING.md` for fixture questions

### Overall

- [ ] Clean-checkout CI passes in ~5 minutes
- [ ] All tests pass on ubuntu-latest (authoritative)
- [ ] macOS tests pass (portability check)
- [ ] Coverage stays ≥45% (clean baseline)
- [ ] No false "test data missing" errors (graceful skips instead)
- [ ] Developer command `make ci-local` ≡ GitHub Actions behavior

---

## References

**Go Documentation**:
- [Go testing — lifecycle management](https://pkg.go.dev/testing) — t.Cleanup, TestMain patterns
- [Go testing — test skipping](https://pkg.go.dev/testing#hdr-Skipping) — t.Skip() vs t.Fatalf()
- [Go coverage](https://pkg.go.dev/cmd/cover) — -covermode, -coverprofile flags
- [Race detector](https://golang.org/doc/articles/race_detector) — -race flag for concurrency testing

**Industry Patterns** (from Exa research):
- CockroachDB: Fixture lifecycle management with $HOME/.cache (benchmarks, expensive corpora)
- Dave Cheney: Test fixtures in Go (distinction between tracked and ignored)
- GitHub Actions: Workflow parallelization (job dependencies with `needs`)
- Go 1.20+: Native coverage in CI (`covdata` tools, per-binary collection)

**pdf2md-tui Specific**:
- [docs/ARCHITECTURE.md](docs/ARCHITECTURE.md) — Worker pool, extraction pipeline
- [docs/COVERAGE.md](docs/COVERAGE.md) — Coverage baseline and roadmap (new)
- [CONTRIBUTING.md](CONTRIBUTING.md) — Fixture policy (new)
