# Contributing to pdf2md-tui

Thank you for your interest in contributing! This guide covers testing, fixtures, and development workflow.

---

## Development Setup

See [README.md — Development](README.md#development) for prerequisites and commands.

Key requirement: **Install `lefthook` before cloning**. This enforces code quality on commit/push.

```bash
brew install lefthook
make hooks-install
```

---

## Testing Guidelines

### Test Fixtures

This project uses three types of test fixtures, each managed differently:

#### 1. Tracked Fixtures (Committed to Git)

Small, deterministic test inputs that all tests depend on:

- **Location**: `internal/**/testdata/`, `pkg/**/testdata/`
- **Examples**: `.txtar` scripts for testscript, tiny synthetic PDFs, mock JSON, golden files
- **Policy**: **Must be tracked in git** (not ignored)
- **Validation**: `go test ./...` from a clean clone must pass

```bash
# Example: adding a new testdata file
mkdir -p pkg/repository/pdf/testdata
echo 'test data' > pkg/repository/pdf/testdata/example.txt
git add pkg/repository/pdf/testdata/example.txt
```

#### 2. Ignored Root-Level Fixtures (Local Dev Only)

Large real-world corpora used for local integration testing:

- **Location**: `/testdata/devops_project/`, `/testdata/samples/`, other root-level directories
- **Examples**: Multi-page PDFs, real documents, local golden-master outputs
- **Policy**: **Ignored by `.gitignore`** (not committed)
- **Tests using these**: **Must skip gracefully** if fixture unavailable

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

- **Location**: `t.TempDir()` (automatically cleaned up after test)
- **Examples**: Temp PDFs, intermediate outputs, scratch directories
- **Policy**: **No tracking needed** (ephemeral)

```bash
# Example pattern (from Go testing docs):
func TestConverter(t *testing.T) {
    tmpDir := t.TempDir()  // Cleaned up automatically
    outputPath := filepath.Join(tmpDir, "output.md")
    // Use tmpDir for test...
}
```

---

### Before Committing

Check for accidentally committed test data:

```bash
# See what you're about to commit
git status --short

# If you added testdata, verify it's NOT in root /testdata/
find testdata -type f | head -5
# Should show nothing (root testdata is ignored)

# If you added pkg/*/testdata/*, verify it IS tracked
git ls-files | grep -E '(pkg|internal)/.*testdata'
# Should show your new files
```

---

### Coverage and Fixtures

Coverage measurements reflect test conditions:

- **CI clean-checkout baseline**: 47.9% (no `/testdata`)
- **Local with corpus**: ~51.7% (with `/testdata/devops_project`)
- **CI gate**: 45%+ (always based on clean-checkout measurement)

See [docs/COVERAGE.md](docs/COVERAGE.md) for complete coverage policy and roadmap.

---

### Running Tests Locally

```bash
# Run full test suite with coverage (same as CI)
make test

# Check coverage percentage
make cover

# Full local CI simulation (recommended before push)
make ci-local

# View coverage report in browser
make cover

# Run specific test or package
go test -v -race ./internal/service/
go test -run TestConverter ./pkg/service/
```

---

## Code Review Checklist

For all pull requests:

- [ ] New code has corresponding tests
- [ ] No test files accidentally committed to `/testdata/`
- [ ] Tests using corpus fixtures call `t.Skipf()` if missing
- [ ] No hardcoded absolute paths in tests (use `t.TempDir()` or `filepath.Join()`)
- [ ] `make ci-local` passes locally before pushing
- [ ] Coverage did not decrease (check `make cover`)
- [ ] All commits follow Conventional Commits format (e.g., `feat:`, `fix:`, `test:`)
- [ ] Lefthook hooks installed and passing (`make hooks-install`)

---

## Conventional Commits

Commit messages follow [Conventional Commits](https://www.conventionalcommits.org/):

```
type(scope): subject

body (optional)

footer (optional)
```

**Types**:
- `feat`: New feature
- `fix`: Bug fix
- `test`: Test additions/updates
- `refactor`: Code restructuring (no behavior change)
- `perf`: Performance improvement
- `docs`: Documentation changes
- `chore`: Tooling, dependencies, build (no production code change)
- `ci`: CI/CD workflow changes

**Examples**:
```bash
git commit -m "feat(handler): add --strip-noise flag for LLM ingestion"
git commit -m "fix(pdf): handle image-only PDFs gracefully"
git commit -m "test(tables): add property-based tests for column detection"
```

---

## Architecture Overview

The project follows a strict layered architecture:

```
cmd/pdf2md-tui/          # Minimal main.go
├── internal/handler/     # CLI + TUI orchestration
│   ├── cli/             # Cobra commands
│   └── tui/             # Bubble Tea dashboard
├── pkg/                 # Core business logic (importable)
│   ├── domain/          # Interfaces + types
│   ├── repository/      # PDF parsing, storage, discovery
│   └── service/         # Conversion engine, markdown rendering
```

Key design patterns:

- **Two-path extraction** (`pkg/repository/pdf/tables.go`): Positional extraction → fallback to plain-text
- **Worker pool** (`internal/handler/cli/convert.go`): Concurrent conversion with goroutines + channels
- **Clean Architecture**: Domain interfaces, zero external deps in core
- **Golden files** (`internal/handler/tui/golden.go`): Deterministic rendering tests

See [docs/ARCHITECTURE.md](docs/ARCHITECTURE.md) for details.

---

## Pre-Commit Hooks

**Lefthook** automatically enforces:

- **pre-commit**: `gofmt -w`, `go vet ./...` on staged files
- **pre-push**: `golangci-lint`, `go test -race ./...`, `go build ./cmd/pdf2md-tui`

If hooks fail, fix the issues locally and try again:

```bash
# Run hooks manually to debug
make hooks-run-pre-commit
make hooks-run-pre-push

# Or fix common issues
make fmt              # Auto-format code
make lint             # Check linting issues
make test             # Run tests
```

---

## Reporting Issues

Use the issue template. Include:

- **Minimal reproduction** (exact CLI command or test case)
- **Expected vs actual behavior**
- **Environment** (OS, Go version, Go output from `pdf2md-tui version`)
- **PDF file** (if applicable, use a small example)

---

## Questions?

- **README.md** — Overview and usage
- **docs/ARCHITECTURE.md** — Design decisions
- **docs/COVERAGE.md** — Coverage policy
- **CLAUDE.md** — Project-local development guidelines (git hooks, tools)
- **GitHub Issues** — Bug reports and feature requests
