# pdf2md-tui — QA 3-Phase Implementation Tech Spec

**Date:** 2026-05-09  
**Status:** Approved for Implementation  
**Authors:** Engineering / QA  
**Replaces:** `qa_test_plan.md`, `qa_implementation_roadmap.md`, `qa_automation_workflow.md`

---

## 1. Current State Audit

### 1.1 Coverage Summary

| Package | Coverage | Test Functions | Status |
|---|---|---|---|
| `pkg/repository/pdf` (tables) | ~90% | 42 | ✅ Well-tested |
| `pkg/repository/discovery` | 100% | 6 | ✅ Complete |
| `internal/handler/cli` | ~80% | 3 | ✅ Good |
| `internal/handler/tui` | ~60% | 6 | ⚠️ Partial |
| `pkg/service` | ~68% | 5 | ⚠️ Partial |
| `pkg/repository/storage` | **0%** | 0 | ❌ Gap |
| `pkg/repository/pdf` (analyze) | **0%** | 0 | ❌ Gap |
| `cmd/pdf2md-tui` | **0%** | 0 | ❌ Gap |
| `pkg/domain` | **0%** | 0 | (pure data, lower priority) |

**Overall: 45.5%** — Target: **≥ 80%**

### 1.2 Pain Points

| ID | Area | Issue |
|---|---|---|
| P1 | Golden corpus | Only `devops_project/` domain; no multi-column, image-only, edge-case PDFs |
| P2 | Table detection | 42 unit tests but zero property-based or fuzz tests for the core heuristic |
| P3 | No property tests | `coalesceChars`, `groupIntoRows`, `applyLLMOptimizations` untested for invariants |
| P4 | No CLI E2E | Full `convert <dir>` pipeline never exercised as a subprocess |
| P5 | No fidelity metrics | Golden diff is byte-exact; structural regression (missing table, wrong heading level) passes silently |
| P6 | No benchmarks | Worker pool and `extractWithTables` hot path have no throughput regression signal |
| P7 | No coverage gate | Coverage can silently drop between PRs |
| P8 | OCR preflight | `AnalyzePDF()` path untested; regression enables conversion of image-only PDFs |
| P9 | No interface assertions | `var _ domain.PDFParser = (*Parser)(nil)` absent; drift is compile-silent |
| P10 | Synthetic test gap | `synthetic_test.go` only tests degenerate case (< 3 cols); positive table path untested |

---

## 2. Architecture of the QA System

```
QA Pyramid — pdf2md-tui
─────────────────────────────────────────────────────────────
          [E2E / testscript]          ← Phase 3
               CLI subprocess, real binary, .txtar scripts

        [Integration / Golden]        ← Phase 1 + 3
        Real PDFs, golden snapshots, corpus expansion

    [Property-Based / Fuzz]           ← Phase 2
    rapid v1.3.0 properties, go fuzz for parsing

  [Unit]                              ← Phase 1 + exists today
  t.TempDir isolated, table-driven, interface assertions

[Static / Lint]                       ← Phase 3
golangci-lint: testifylint, tparallel, thelper
─────────────────────────────────────────────────────────────
```

### Scoring Targets (OpenDataLoader Benchmark Methodology)

| Metric | Measures | Method | Target |
|---|---|---|---|
| **NID** (Reading Order) | Text extraction sequence | Normalized Indel Distance vs ground truth | ≥ 0.85 |
| **TEDS** (Table Fidelity) | Table DOM structure accuracy | Tree Edit Distance Similarity | ≥ 0.80 |
| **MHS** (Heading Hierarchy) | H1/H2/H3 nesting | Markdown Heading-level Similarity | ≥ 0.75 |

---

## 3. Phase 1 — QA Foundations

**Goal:** Eliminate 0%-coverage gaps, enforce a coverage floor, fix known test invariant errors.  
**Duration:** ~1 day  
**No new dependencies required.**

### 3.1 Compile-Time Interface Assertions

**Files:** `pkg/repository/pdf/pdf.go`, `pkg/repository/storage/storage.go`

```go
// In pkg/repository/pdf/pdf.go
var _ domain.PDFParser = (*Parser)(nil)

// In pkg/repository/storage/storage.go
var _ domain.PDFStorage = (*Storage)(nil)
```

Catches interface drift at `go build` time, before any test runs.

### 3.2 Coverage Gate — Makefile

**File:** `Makefile`

```makefile
cover-check: test
	@go tool cover -func=coverage.out \
	  | awk '/total:/{cov=$$3+0; if(cov<70){printf "Coverage %.1f%% < 70%% threshold\n",cov; exit 1} else {printf "Coverage OK: %.1f%%\n",cov}}'
```

Add `cover-check` as a dependency of `make check`. Raises threshold from 70% → 80% once Phase 2 lands.

### 3.3 Fix Synthetic Test — Positive Table Case

**File:** `pkg/repository/pdf/synthetic_test.go`

Current test only validates the degenerate path (2 columns → `BlockTypeText`). Add the positive path:

```go
func TestExtractPageBlocks_SyntheticTable_ThreeColumns(t *testing.T) {
    // 3 columns at >50pt spacing, 2 rows — should produce BlockTypeTable
    texts := []pdf.Text{
        // Row 1 (Y=100)
        {S: "Col1", X: 50,  Y: 100, W: 30, FontSize: 10},
        {S: "Col2", X: 150, Y: 100, W: 30, FontSize: 10},
        {S: "Col3", X: 250, Y: 100, W: 30, FontSize: 10},
        // Row 2 (Y=88) — within groupIntoRows tolerance
        {S: "A",    X: 50,  Y: 88,  W: 10, FontSize: 10},
        {S: "B",    X: 150, Y: 88,  W: 10, FontSize: 10},
        {S: "C",    X: 250, Y: 88,  W: 10, FontSize: 10},
    }
    page := NewMockPage(texts)
    blocks := extractPageBlocks(page)

    found := false
    for _, b := range blocks {
        if b.Type == domain.BlockTypeTable {
            found = true
            break
        }
    }
    if !found {
        t.Errorf("expected BlockTypeTable for 3-column 2-row layout, got: %+v", blocks)
    }
}
```

Reuses `NewMockPage()` and `Text()` from `pkg/repository/pdf/mock_test.go:13`.

### 3.4 Storage Layer Tests (0% → ~85%)

**File to create:** `pkg/repository/storage/storage_test.go`

```go
package storage_test

import (
    "os"
    "path/filepath"
    "testing"

    "github.com/nawodyaishan/pdf2md-tui/pkg/repository/storage"
)

func TestWriteMarkdown_CreatesFile(t *testing.T) {
    dir := t.TempDir()
    s := storage.NewStorage()
    path := filepath.Join(dir, "out.md")

    if err := s.WriteMarkdown(path, []byte("# Hello")); err != nil {
        t.Fatalf("WriteMarkdown: %v", err)
    }
    got, err := os.ReadFile(path)
    if err != nil {
        t.Fatalf("ReadFile: %v", err)
    }
    if string(got) != "# Hello" {
        t.Errorf("content mismatch: %q", got)
    }
}

func TestWriteMarkdown_ErrorOnBadPath(t *testing.T) {
    s := storage.NewStorage()
    err := s.WriteMarkdown("/nonexistent/deep/path/out.md", []byte("data"))
    if err == nil {
        t.Fatal("expected error for non-existent parent dir")
    }
}

func TestCreateImageDir_CreatesMissingDir(t *testing.T) {
    dir := t.TempDir()
    s := storage.NewStorage()
    imgDir, err := s.CreateImageDir(filepath.Join(dir, "doc"))
    if err != nil {
        t.Fatalf("CreateImageDir: %v", err)
    }
    if _, err := os.Stat(imgDir); os.IsNotExist(err) {
        t.Errorf("image dir not created: %s", imgDir)
    }
}

func TestFileExists_TrueAndFalse(t *testing.T) {
    dir := t.TempDir()
    s := storage.NewStorage()
    existing := filepath.Join(dir, "file.txt")
    _ = os.WriteFile(existing, []byte("x"), 0644)

    if !s.FileExists(existing) {
        t.Error("expected true for existing file")
    }
    if s.FileExists(filepath.Join(dir, "missing.txt")) {
        t.Error("expected false for missing file")
    }
}
```

### 3.5 OCR Preflight Tests (0% → ~75%)

**File to create:** `pkg/repository/pdf/analyze_test.go`

```go
package pdf

import (
    "testing"

    ledongpdf "github.com/ledongthuc/pdf"
)

func TestAnalyzePreFlight_TextRichPage_NotOCR(t *testing.T) {
    texts := make([]ledongpdf.Text, 50)
    for i := range texts {
        texts[i] = ledongpdf.Text{S: "word", X: float64(i * 10), Y: 100, FontSize: 12, W: 20}
    }
    page := NewMockPage(texts)
    result := analyzePageForOCR(page)
    if result.RequiresOCR {
        t.Error("text-rich page should not require OCR")
    }
}

func TestAnalyzePreFlight_EmptyPage_RequiresOCR(t *testing.T) {
    page := NewMockPage(nil)
    result := analyzePageForOCR(page)
    if !result.RequiresOCR {
        t.Error("empty page should be flagged as requiring OCR")
    }
}
```

Reuses `NewMockPage()` from `mock_test.go:13`. Adjust function names to match the actual exported/unexported API in `analyze.go`.

### Phase 1 Acceptance Criteria

- [ ] `make build` passes (interface assertions compile)
- [ ] `make test` — all tests green
- [ ] `make cover-check` — total coverage ≥ 55% (after storage + analyze tests)
- [ ] `TestExtractPageBlocks_SyntheticTable_ThreeColumns` passes

---

## 4. Phase 2 — Property-Based Testing & Fuzzing

**Goal:** Encode invariants of the core algorithms so edge cases are found automatically.  
**Duration:** ~2 days  
**New dependencies:**

```bash
go get pgregory.net/rapid@v1.3.0
```

### 4.1 Property-Based Tests — Table Extraction

**File to create:** `pkg/repository/pdf/tables_property_test.go`

```go
package pdf

import (
    "testing"

    ledongpdf "github.com/ledongthuc/pdf"
    "pgregory.net/rapid"
)

// P1: word count invariant — coalesceChars never produces more words than input chars
func TestProperty_CoalesceChars_WordCountBound(t *testing.T) {
    rapid.Check(t, func(t *rapid.T) {
        n := rapid.IntRange(0, 200).Draw(t, "n")
        texts := make([]ledongpdf.Text, n)
        for i := range texts {
            texts[i] = ledongpdf.Text{
                S:        rapid.StringMatching(`[A-Za-z]`).Draw(t, "char"),
                X:        rapid.Float64Range(0, 612).Draw(t, "x"),
                Y:        rapid.Float64Range(0, 792).Draw(t, "y"),
                FontSize: rapid.Float64Range(6, 72).Draw(t, "fs"),
                W:        rapid.Float64Range(1, 30).Draw(t, "w"),
            }
        }
        words := coalesceChars(texts)
        if len(words) > len(texts) {
            t.Fatalf("coalesceChars produced %d words from %d chars — impossible", len(words), len(texts))
        }
    })
}

// P2: groupIntoRows output is sorted Y-descending, cells X-ascending
func TestProperty_GroupIntoRows_SortOrder(t *testing.T) {
    rapid.Check(t, func(t *rapid.T) {
        n := rapid.IntRange(1, 50).Draw(t, "n")
        words := make([]word, n)
        for i := range words {
            words[i] = word{
                text: "w",
                x:    rapid.Float64Range(0, 500).Draw(t, "x"),
                y:    rapid.Float64Range(0, 700).Draw(t, "y"),
            }
        }
        rows := groupIntoRows(words)
        for i := 1; i < len(rows); i++ {
            if rows[i].y > rows[i-1].y {
                t.Fatalf("rows not sorted Y-descending at index %d: %.2f > %.2f", i, rows[i].y, rows[i-1].y)
            }
        }
        for _, r := range rows {
            for j := 1; j < len(r.cells); j++ {
                if r.cells[j].x < r.cells[j-1].x {
                    t.Fatalf("cells not sorted X-ascending within row")
                }
            }
        }
    })
}

// P3: extractPageBlocks never panics on arbitrary well-formed input
func TestProperty_ExtractPageBlocks_NoPanic(t *testing.T) {
    rapid.Check(t, func(t *rapid.T) {
        n := rapid.IntRange(0, 100).Draw(t, "n")
        texts := make([]ledongpdf.Text, n)
        for i := range texts {
            texts[i] = ledongpdf.Text{
                S:        rapid.String().Draw(t, "s"),
                X:        rapid.Float64Range(0, 612).Draw(t, "x"),
                Y:        rapid.Float64Range(0, 792).Draw(t, "y"),
                FontSize: rapid.Float64Range(0, 100).Draw(t, "fs"),
                W:        rapid.Float64Range(0, 50).Draw(t, "w"),
            }
        }
        page := NewMockPage(texts)
        _ = extractPageBlocks(page) // must not panic
    })
}
```

### 4.2 Property-Based Tests — LLM Optimizations

**File to create:** `pkg/service/converter_property_test.go`

```go
package service

import (
    "testing"
    "pgregory.net/rapid"
)

// P4: applyLLMOptimizations is idempotent — f(f(x)) == f(x)
func TestProperty_LLMOptimizations_Idempotent(t *testing.T) {
    rapid.Check(t, func(t *rapid.T) {
        input := rapid.String().Draw(t, "input")
        once := applyLLMOptimizations(input)
        twice := applyLLMOptimizations(once)
        if once != twice {
            t.Fatalf("applyLLMOptimizations not idempotent:\nonce:  %q\ntwice: %q", once, twice)
        }
    })
}

// P5: applyLLMOptimizations never increases byte length
func TestProperty_LLMOptimizations_LengthNonIncreasing(t *testing.T) {
    rapid.Check(t, func(t *rapid.T) {
        input := rapid.String().Draw(t, "input")
        output := applyLLMOptimizations(input)
        if len(output) > len(input) {
            t.Fatalf("output longer than input: %d > %d", len(output), len(input))
        }
    })
}
```

### 4.3 Fuzz: `coalesceChars`

**File to create:** `pkg/repository/pdf/fuzz_test.go`  
**Seed corpus dir:** `pkg/repository/pdf/testdata/fuzz/FuzzCoalesceChars/`

```go
package pdf

import (
    "encoding/binary"
    "math"
    "testing"

    ledongpdf "github.com/ledongthuc/pdf"
)

func FuzzCoalesceChars(f *testing.F) {
    // Seed: 2 chars close together (one word)
    f.Add([]byte{
        0x41, 0x00, 0x00, 0x00, 0x00, 0x00, 0x24, 0x40, // S="A", X=10.0
        0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x59, 0x40, // Y=100.0
        0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x1C, 0x40, // FontSize=7.0
        0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x1C, 0x40, // W=7.0
    })

    f.Fuzz(func(t *testing.T, data []byte) {
        texts := parseTextsFromBytes(data)
        result := coalesceChars(texts)
        // Invariant: must not panic, count bounded
        if len(result) > len(texts) {
            t.Fatalf("word count %d exceeds char count %d", len(result), len(texts))
        }
    })
}

func parseTextsFromBytes(data []byte) []ledongpdf.Text {
    const stride = 32 // 4 x float64
    var texts []ledongpdf.Text
    for i := 0; i+stride <= len(data); i += stride {
        chunk := data[i : i+stride]
        s := string(rune(chunk[0]))
        x := math.Float64frombits(binary.LittleEndian.Uint64(chunk[8:16]))
        y := math.Float64frombits(binary.LittleEndian.Uint64(chunk[16:24]))
        fs := math.Float64frombits(binary.LittleEndian.Uint64(chunk[24:32]))
        texts = append(texts, ledongpdf.Text{S: s, X: x, Y: y, FontSize: fs, W: 7})
    }
    return texts
}
```

### 4.4 Fuzz: `applyLLMOptimizations`

**File to create:** `pkg/service/fuzz_test.go`

```go
package service

import "testing"

func FuzzApplyLLMOptimizations(f *testing.F) {
    f.Add("Hello  World\n12\nNext Page")
    f.Add("")
    f.Add("\x00\xff")

    f.Fuzz(func(t *testing.T, input string) {
        out1 := applyLLMOptimizations(input)
        out2 := applyLLMOptimizations(out1)
        if out1 != out2 {
            t.Fatalf("not idempotent for input %q", input)
        }
    })
}
```

### Running Phase 2 Tests

```bash
# Property-based tests (runs hundreds of cases automatically)
go test -run TestProperty ./pkg/...

# Fuzzing (30 seconds in CI, longer locally)
go test -fuzz=FuzzCoalesceChars -fuzztime=30s ./pkg/repository/pdf/
go test -fuzz=FuzzApplyLLMOptimizations -fuzztime=30s ./pkg/service/

# Seed corpus only (fast, runs in regular go test)
go test ./pkg/repository/pdf/ ./pkg/service/
```

### Phase 2 Acceptance Criteria

- [ ] `go get pgregory.net/rapid@v1.3.0` succeeds, `go.mod` updated
- [ ] All `TestProperty_*` tests pass with default 100 cases each
- [ ] Fuzz seed corpus tests pass as part of `make test`
- [ ] No panics found during 30s fuzz run
- [ ] `make test` + `make cover-check` — coverage ≥ 65%

---

## 5. Phase 3 — Integration, Benchmarks & CI

**Goal:** End-to-end pipeline coverage, performance regression detection, full CI gate.  
**Duration:** ~3 days  
**New dependencies:**

```bash
go get github.com/rogpeppe/go-internal@v1.14.1
go install golang.org/x/perf/cmd/benchstat@latest
```

### 5.1 CLI End-to-End Tests via testscript

**File to create:** `internal/handler/cli/e2e_test.go`

```go
package cli_test

import (
    "os"
    "testing"

    "github.com/rogpeppe/go-internal/testscript"
    "github.com/nawodyaishan/pdf2md-tui/internal/handler/cli"
)

func TestMain(m *testing.M) {
    os.Exit(testscript.RunMain(m, map[string]func() int{
        "pdf2md-tui": func() int {
            if err := cli.Execute(); err != nil {
                return 1
            }
            return 0
        },
    }))
}

func TestCLI(t *testing.T) {
    testscript.Run(t, testscript.Params{
        Dir: "testdata/scripts",
        Setup: func(env *testscript.Env) error {
            env.Setenv("TESTDATA_DIR", "../../../testdata")
            return nil
        },
    })
}
```

**Script dir:** `internal/handler/cli/testdata/scripts/`

**`convert_single.txtar`:**
```txtar
# Convert a single PDF in quiet mode and verify JSON output
exec pdf2md-tui convert $TESTDATA_DIR/devops_project --quiet --output $WORK/out
stdout '"converted"'
stdout '"errors":0'
! stderr .

-- out/.gitkeep --
```

**`quiet_json_schema.txtar`:**
```txtar
# --quiet must emit valid JSON with required keys
exec pdf2md-tui convert $TESTDATA_DIR/devops_project --quiet --output $WORK/out
stdout '"converted":\s*[0-9]+'
stdout '"errors":\s*[0-9]+'
stdout '"skipped":\s*[0-9]+'
```

**`overwrite_requires_force.txtar`:**
```txtar
# Second run without --force must fail in non-interactive mode
exec pdf2md-tui convert $TESTDATA_DIR/devops_project --quiet --output $WORK/out
exec pdf2md-tui convert $TESTDATA_DIR/devops_project --quiet --output $WORK/out
! stdout .
stderr 'force'
```

**`recursive_flag.txtar`:**
```txtar
# --recursive discovers PDFs in nested subdirectories
exec pdf2md-tui convert $TESTDATA_DIR --recursive --quiet --output $WORK/out
stdout '"converted":\s*[1-9][0-9]*'
```

### 5.2 Benchmark Suite

**File to create:** `pkg/repository/pdf/bench_test.go`

```go
package pdf

import (
    "testing"
    ledongpdf "github.com/ledongthuc/pdf"
)

func BenchmarkCoalesceChars(b *testing.B) {
    for _, n := range []int{100, 500, 1000} {
        texts := make([]ledongpdf.Text, n)
        for i := range texts {
            texts[i] = ledongpdf.Text{S: "A", X: float64(i * 8), Y: 100, FontSize: 12, W: 7}
        }
        b.Run(fmt.Sprintf("N=%d", n), func(b *testing.B) {
            b.ReportAllocs()
            for i := 0; i < b.N; i++ {
                _ = coalesceChars(texts)
            }
        })
    }
}

func BenchmarkExtractPageBlocks_SyntheticTable(b *testing.B) {
    texts := make([]ledongpdf.Text, 60) // 3 cols × 20 rows
    for i := range texts {
        col := i % 3
        row := i / 3
        texts[i] = ledongpdf.Text{
            S: "cell", X: float64(col*150 + 50), Y: float64(100 - row*12), FontSize: 10, W: 28,
        }
    }
    page := NewMockPage(texts)
    b.ReportAllocs()
    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        _ = extractPageBlocks(page)
    }
}
```

**File to create:** `pkg/service/bench_test.go`

```go
package service

import (
    "testing"
    "github.com/nawodyaishan/pdf2md-tui/pkg/domain"
)

func BenchmarkRenderMarkdown_MixedContent(b *testing.B) {
    blocks := []domain.PageBlock{
        {Type: domain.BlockTypeText, Text: "Introduction paragraph with some content."},
        {Type: domain.BlockTypeTable, Table: domain.TableData{
            Rows: [][]string{{"A", "B", "C"}, {"1", "2", "3"}, {"4", "5", "6"}},
        }},
        {Type: domain.BlockTypeText, Text: "Conclusion paragraph."},
    }
    b.ReportAllocs()
    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        _ = renderMarkdown(blocks, false)
    }
}
```

**Makefile additions:**

```makefile
bench:
	go test -bench=. -benchmem -count=5 ./pkg/... 2>/dev/null | tee bench.txt
	@echo "Benchmark results written to bench.txt"

bench-compare: bench
	@benchstat bench.txt
```

### 5.3 Golangci-lint Test Linters

**File to create/modify:** `.golangci.yml`

```yaml
linters:
  enable:
    - testifylint    # Enforce assert.NoError not assert.Nil for errors
    - tparallel      # Catch t.Parallel() safety issues
    - thelper        # Require t.Helper() as first line in helpers

linters-settings:
  testifylint:
    disable:
      - suite-dont-use-pkg  # Not using testify suite

issues:
  exclude-rules:
    - linters: [thelper]
      path: "_test.go"
      text: "begin"  # Allow setup funcs without t.Helper()
```

### 5.4 Corpus Expansion

**Directory to create:** `testdata/corpus/`

```
testdata/corpus/
├── simple_text/           # Single-column prose — validates basic text extraction
│   └── *.pdf
├── image_only/            # Scanned PDFs — must produce StatusIgnored
│   └── *.pdf
└── edge_cases/
    └── single_page.pdf
```

**Extend `qa_golden_test.go`** to scan `testdata/corpus/` directories in addition to `testdata/devops_project/`. Image-only PDFs in `corpus/image_only/` assert `res.Status == domain.StatusIgnored` (no output file written).

### 5.5 CI Workflow — `.github/workflows/ci.yml`

Add to the **Test** job:

```yaml
- name: Coverage gate
  run: |
    go tool cover -func=coverage.out \
      | awk '/total:/{cov=$3+0; if(cov<70){printf "FAIL: %.1f%% < 70%%\n",cov; exit 1} else {printf "OK: %.1f%%\n",cov}}'

- name: Fuzz (seed corpus, CI mode)
  run: |
    go test -run=^$ -fuzz=FuzzCoalesceChars -fuzztime=30s ./pkg/repository/pdf/
    go test -run=^$ -fuzz=FuzzApplyLLMOptimizations -fuzztime=20s ./pkg/service/
```

Add a new **Bench** job (runs only on PRs targeting `main`):

```yaml
bench:
  name: Benchmark Regression
  runs-on: ubuntu-latest
  if: github.event_name == 'pull_request' && github.base_ref == 'main'
  steps:
    - uses: actions/checkout@v4
      with:
        fetch-depth: 0

    - uses: actions/setup-go@v5
      with:
        go-version-file: go.mod

    - name: Baseline (main)
      run: |
        git stash
        go test -bench=. -benchmem -count=10 -run=^$ ./pkg/... > baseline.txt
        git stash pop

    - name: PR benchmarks
      run: go test -bench=. -benchmem -count=10 -run=^$ ./pkg/... > pr.txt

    - name: benchstat comparison
      run: |
        go install golang.org/x/perf/cmd/benchstat@latest
        benchstat baseline.txt pr.txt | tee benchstat-result.txt

    - uses: actions/upload-artifact@v4
      with:
        name: benchstat-result
        path: benchstat-result.txt
```

### Phase 3 Acceptance Criteria

- [ ] `go get github.com/rogpeppe/go-internal@v1.14.1` succeeds
- [ ] `TestCLI` testscript suite passes (all 4 scenarios)
- [ ] `make bench` produces `bench.txt` without errors
- [ ] `golangci-lint run ./...` passes with `testifylint`, `tparallel`, `thelper` enabled
- [ ] CI: coverage gate ≥ 70% green
- [ ] CI: fuzz seed corpus jobs pass
- [ ] CI: benchstat job posts artifact on PRs targeting `main`
- [ ] `make test` + `make cover-check` — coverage ≥ 80%

---

## 6. File Change Summary

| File | Action | Phase |
|---|---|---|
| `docs/specs/2026-05-09-pdf2md-tui-qa-3phase-tech-spec.md` | CREATE (this file) | — |
| `pkg/repository/pdf/pdf.go` | ADD interface assertion | 1 |
| `pkg/repository/storage/storage.go` | ADD interface assertion | 1 |
| `pkg/repository/pdf/synthetic_test.go` | ADD positive table test | 1 |
| `pkg/repository/storage/storage_test.go` | CREATE — 4 tests | 1 |
| `pkg/repository/pdf/analyze_test.go` | CREATE — 2 tests | 1 |
| `Makefile` | ADD `cover-check`, `bench`, `bench-compare` | 1, 3 |
| `pkg/repository/pdf/tables_property_test.go` | CREATE — 3 property tests | 2 |
| `pkg/service/converter_property_test.go` | CREATE — 2 property tests | 2 |
| `pkg/repository/pdf/fuzz_test.go` | CREATE — FuzzCoalesceChars | 2 |
| `pkg/service/fuzz_test.go` | CREATE — FuzzApplyLLMOptimizations | 2 |
| `pkg/repository/pdf/testdata/fuzz/FuzzCoalesceChars/` | CREATE — seed corpus | 2 |
| `internal/handler/cli/e2e_test.go` | CREATE — testscript runner | 3 |
| `internal/handler/cli/testdata/scripts/*.txtar` | CREATE — 4 scenarios | 3 |
| `pkg/repository/pdf/bench_test.go` | CREATE — 2 benchmarks | 3 |
| `pkg/service/bench_test.go` | CREATE — 1 benchmark | 3 |
| `.golangci.yml` | CREATE/MODIFY — add 3 test linters | 3 |
| `.github/workflows/ci.yml` | MODIFY — coverage gate, fuzz, bench job | 3 |
| `testdata/corpus/` | CREATE — expanded fixture corpus | 3 |

---

## 7. Reusable Existing Utilities

| Utility | Location | Used By |
|---|---|---|
| `NewMockPage(texts)` | `pkg/repository/pdf/mock_test.go:13` | `analyze_test.go`, `synthetic_test.go`, property tests |
| `Text(s,x,y,w,fs,i)` | `pkg/repository/pdf/mock_test.go:27` | `tables_property_test.go` |
| `t.TempDir()` pattern | `pkg/repository/discovery/discovery_test.go:14` | `storage_test.go` |
| `domain.NewConfig()` | `pkg/domain/config.go` | converter property tests |
| `mockStorage`, `mockParser` | `pkg/service/converter_test.go:65–100` | extend for new service tests |

---

## 8. Dependency Summary

```bash
# Phase 2
go get pgregory.net/rapid@v1.3.0

# Phase 3
go get github.com/rogpeppe/go-internal@v1.14.1
go install golang.org/x/perf/cmd/benchstat@latest
```

---

## 9. Verification Runbook

```bash
# Phase 1 — run after implementation
make test             # all existing + new tests pass
make cover-check      # ≥ 55% (before Phase 2)

# Phase 2 — run after implementation
go test -run TestProperty ./pkg/...        # property tests green
go test -fuzz=FuzzCoalesceChars -fuzztime=10s ./pkg/repository/pdf/
go test -fuzz=FuzzApplyLLMOptimizations -fuzztime=10s ./pkg/service/
make cover-check      # ≥ 65%

# Phase 3 — run after implementation
go test -run TestCLI ./internal/handler/cli/   # E2E green
make bench                                      # bench.txt written
benchstat bench.txt                             # no anomalies
golangci-lint run ./...                        # testifylint/tparallel/thelper clean
make cover-check      # ≥ 80% ← final target
```

---

## 10. Coverage Trajectory

| Milestone | Expected Coverage |
|---|---|
| Baseline (today) | 45.5% |
| After Phase 1 | ~58% |
| After Phase 2 | ~68% |
| After Phase 3 | **≥ 80%** |
