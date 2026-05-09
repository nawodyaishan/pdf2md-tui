# Future Test CI/CD Stabilization Plan

Date: 2026-05-10
Scope: Prevent future CI failures caused by test fixture drift, local-only state, coverage mismatch, and test automation assumptions.

## Problem Statement

The current CI failures were not caused by production code regressions. They came from test infrastructure drift:

- Package-local `testdata/` fixtures were ignored by `.gitignore`, so GitHub Actions did not receive files that local tests used.
- Root-level large PDF fixtures existed locally but are intentionally ignored, causing local test behavior to differ from clean checkout behavior.
- E2E tests assumed fixture paths existed and failed with low-signal errors when they did not.
- Coverage claims in documentation did not match reproducible clean-checkout coverage.
- Lint, test, and coverage gates were not aligned between local hooks, Makefile targets, and GitHub Actions.

The goal is to make tests deterministic from a clean clone and make failures explain the real problem.

## Guiding Rules

1. CI must pass from a clean checkout with no untracked files.
2. Tests must not depend on ignored files unless they explicitly skip with a clear reason.
3. Required test fixtures must be either tracked, generated inside the test, or downloaded by an explicit CI step.
4. Coverage gates must use clean-checkout measurements, not local workstation results.
5. Local developer commands should match CI as closely as practical.
6. Do not add CI complexity unless it catches a real failure mode.

## Test Fixture Policy

### Tracked Fixtures

Use tracked fixtures for small, deterministic test inputs:

- `internal/**/testdata/**`
- `pkg/**/testdata/**`
- `.txtar` scripts for `testscript`
- Small synthetic PDFs or generated text files

These directories must not be globally ignored.

Recommended `.gitignore` pattern:

```gitignore
/testdata/
```

Do not use:

```gitignore
testdata/
```

The unanchored rule ignores every package-local `testdata` directory and breaks CI.

### Ignored Fixtures

Use ignored root-level fixtures only for large manual/integration corpora:

- `/testdata/devops_project`
- large real-world PDFs
- generated markdown outputs
- local logs and benchmark dumps

Tests that use ignored fixtures must call `t.Skipf(...)` when the fixture root is missing.

### Generated Fixtures

Prefer generated fixtures when possible:

- Temp directories via `t.TempDir()`.
- Tiny files created in test setup.
- Synthetic domain objects instead of real PDFs when testing CLI formatting or service behavior.

Only use real PDFs when parser behavior is the thing being tested.

## E2E Test Rules

E2E tests must clearly separate two modes:

1. Clean-checkout mode: runs in CI without root `/testdata`.
2. Corpus mode: runs locally or in a dedicated integration workflow with large PDFs available.

Recommended pattern:

```go
fixtureRoot := filepath.Join(projectRoot, "testdata")
if _, err := os.Stat(filepath.Join(fixtureRoot, "devops_project")); err != nil {
	t.Skipf("skipping corpus E2E tests; fixture root is missing: %s", fixtureRoot)
}
```

Avoid fallback paths that walk outside the repository. They can accidentally find unrelated directories and make tests non-deterministic.

## Coverage Policy

Coverage gates must be based on clean-checkout CI behavior.

Current verified baseline:

- Clean checkout without root `/testdata`: `47.9%`
- Local with large PDF corpus: about `51.7%`

Near-term gate:

- Keep CI threshold at `45%` until fixture-independent tests raise the baseline.

Future target increments:

- Raise to `50%` after CLI E2E coverage is made fixture-independent.
- Raise to `55%` after TUI/model tests are expanded without golden-file brittleness.
- Raise to `60%+` only after service/parser tests cover real missing branches.

Do not raise coverage by adding shallow tests that assert only that functions exist.

## CI Test Workflow Shape

Keep test CI simple:

```yaml
test:
  strategy:
    matrix:
      os: [ubuntu-latest, macos-latest]
  steps:
    - checkout
    - setup-go
    - go test -race -coverprofile=coverage.out -covermode=atomic ./...
    - coverage gate on ubuntu only
    - upload coverage.out on ubuntu only
```

Use Ubuntu for authoritative coverage because it is cheaper and consistent enough for the gate.

Use macOS as a portability smoke test, not a second coverage authority.

## Dedicated Integration Workflow

If the large PDF corpus is important in CI, create a separate workflow instead of mixing it into default CI.

Trigger:

```yaml
on:
  workflow_dispatch:
  schedule:
    - cron: '0 3 * * 1'
```

Responsibilities:

- Download or restore large PDF corpus.
- Run corpus E2E tests.
- Run golden-master tests.
- Upload diffs and converted outputs as artifacts.

This keeps pull request CI fast and deterministic.

## Local Developer Commands

Keep these commands as the source of truth:

```bash
go test -race -coverprofile=coverage.out -covermode=atomic ./...
go tool cover -func=coverage.out | tail -n 1
golangci-lint run ./...
go vet ./...
```

Optional corpus check:

```bash
go test ./internal/handler/cli -run TestCLI -count=1
go test ./pkg/service -run TestQA_GoldenMaster -count=1
```

Use `make check` only if it runs the same core commands.

## Testscript Guidelines

For `github.com/rogpeppe/go-internal/testscript`:

- Use `testscript.Main`, not deprecated `testscript.RunMain`.
- Keep scripts in package-local `testdata/scripts`.
- Keep scripts small and focused.
- Avoid depending on local absolute paths.
- Use environment variables for fixture roots.
- Prefer scripts that validate CLI behavior with generated temp files.

Bad pattern:

```txt
exec pdf2md-tui convert ../../../../testdata/devops_project
```

Preferred pattern:

```txt
exec pdf2md-tui convert $TESTDATA_DIR/devops_project
```

## Fuzz Test Policy

Short fuzz runs are useful in CI, but they can create new corpus files locally.

Rules:

- Do not commit generated fuzz corpus files accidentally.
- Commit fuzz seeds only when they reproduce a real bug or improve meaningful baseline coverage.
- Keep CI fuzz time short, for example `30s`.
- Run longer fuzzing manually or in scheduled jobs.

Before committing, check:

```bash
git status --short
find pkg internal -path '*/testdata/fuzz/*' -type f
```

## Documentation Rules

Coverage claims in docs must include:

- Command used.
- Whether root `/testdata` was present.
- Date measured.
- Clean-checkout result if CI is discussed.

Avoid status claims like “completed” unless they are reproducible from a clean checkout.

## Future Work Queue

1. Replace corpus-dependent CLI scripts with generated lightweight fixtures.
2. Add a small synthetic PDF fixture if parser CLI paths need non-skipped CI coverage.
3. Split `TestQA_GoldenMaster` behind an explicit integration build tag or environment variable.
4. Add a weekly corpus workflow for real PDFs and golden-master diffs.
5. Raise coverage gate from `45%` to `50%` once clean-checkout coverage exceeds `52%`.
6. Add a contributor note explaining tracked vs ignored test fixtures.

## Acceptance Criteria

The test CI/CD setup is considered stable when:

- A fresh clone can run `go test -race -coverprofile=coverage.out -covermode=atomic ./...` successfully.
- GitHub Actions no longer fails because of missing ignored fixtures.
- Coverage gate reflects clean-checkout coverage.
- E2E tests skip intentionally when optional corpora are absent.
- Required fixtures are tracked or generated during tests.
- Local `make check` and CI produce equivalent pass/fail results.
