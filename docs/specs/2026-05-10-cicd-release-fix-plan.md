# CI/CD and Release Stabilization Plan

Date: 2026-05-10
Scope: Fix the current GitHub Actions failures, repair clean-checkout test reliability, align Phase 4 QA claims with real coverage, and harden release automation.

## Source Inputs

- Local repo inspection of `.github/workflows/ci.yml`, `.github/workflows/release.yml`, `.goreleaser.yml`, `Makefile`, `.gitignore`, `lefthook.yml`, and failing test files.
- GitHub Actions run #75 screenshot and lint annotations supplied by the maintainer.
- Exa MCP lookups for current upstream docs:
  - `github.com/rogpeppe/go-internal/testscript` package docs and deprecation context.
  - `golangci/golangci-lint-action` compatibility and configuration docs.
  - `golang/govulncheck-action` usage docs.
  - `actions/dependency-review-action` usage docs.
  - `goreleaser/goreleaser-action` usage and token guidance.
- Context7 MCP was requested but is not available as a callable tool in this session. Use primary docs found through Exa as the replacement source of truth.

## Current Failure Summary

The current CI run fails in three areas:

1. `Lint` fails on staticcheck issues:
   - `internal/handler/cli/e2e_test.go:14`: `SA1019`, `testscript.RunMain` is deprecated.
   - `pkg/repository/pdf/pdf_test.go:19`: `S1021`, split declaration and assignment.
   - `pkg/repository/sysinfo/sysinfo_test.go:25`: `SA4003`, unsigned integer is never less than zero.
   - `pkg/repository/sysinfo/sysinfo_test.go:28`: `SA4003`, unsigned integer is never less than zero.

2. `Test (ubuntu-latest)` and `Test (macos-latest)` fail because CI does not receive all local-only test fixtures:
   - `internal/handler/cli/e2e_test.go` expects `testdata/scripts`.
   - The actual scripts live at `internal/handler/cli/testdata/scripts`.
   - `internal/handler/cli/testdata/` is ignored because `.gitignore` ignores `testdata/` globally.

3. The Phase 4 QA status and the CI coverage gate are inconsistent:
   - The Phase 4 doc claims 76.1% coverage.
   - Local coverage with ignored fixtures present was 51.7%.
   - A clean tracked checkout produced about 48.0% before failing.
   - `.github/workflows/ci.yml` enforces 70%, so the test job will remain red even after staticcheck is fixed unless coverage or the gate is corrected.

## Fix Plan

### Phase 0: Stop the Bleeding

Goal: make the current `dev` branch CI run pass without weakening important checks silently.

1. Fix staticcheck annotations from run #75.
   - Replace `testscript.RunMain(...)` with `testscript.Main(...)` in `internal/handler/cli/e2e_test.go`.
   - Merge declaration and assignment in `pkg/repository/pdf/pdf_test.go`:
     ```go
     p := NewParser()
     ```
   - Remove impossible `< 0` checks on `uint64` fields in `pkg/repository/sysinfo/sysinfo_test.go`.
   - Keep the `MemoryPct` range check because it is a floating-point percentage and still meaningful.

2. Fix ignored test fixtures.
   - Change `.gitignore` rule `testdata/` to `/testdata/` so package-local testdata directories are not ignored.
   - Verify and commit:
     - `internal/handler/cli/testdata/scripts/convert_basic.txtar`
     - `internal/handler/cli/testdata/scripts/recursive_discovery.txtar`
     - `internal/handler/cli/testdata/scripts/overwrite_requires_force.txtar`
     - `internal/handler/cli/testdata/scripts/quiet_json.txtar`

3. Fix ignored debug command tests.
   - Change `.gitignore` rule `debug-pdf` to `/debug-pdf` so it ignores only the root binary.
   - Commit `cmd/debug-pdf/main_test.go`.

4. Correct the testscript fixture path if needed.
   - Preferred: keep `Dir: "testdata/scripts"` because it is package-relative and works once `internal/handler/cli/testdata/` is tracked.
   - Add a guard that fails with a clear message if scripts are missing, or skip only when running explicitly without integration fixtures.

5. Re-run locally from a clean tracked checkout before pushing.
   - `git worktree add --detach /private/tmp/pdf2md-tui-ci-check HEAD`
   - `go test -race -coverprofile=/private/tmp/pdf2md-tui-coverage.out -covermode=atomic ./...`
   - `go tool cover -func=/private/tmp/pdf2md-tui-coverage.out | tail -n 1`
   - `golangci-lint run ./...`

Acceptance criteria:

- `Lint` job passes.
- Both OS test jobs complete test execution.
- No CI failure caused by missing local-only fixtures.

### Phase 1: Resolve Coverage Gate Truthfully

Goal: align CI coverage policy with actual coverage and the Phase 4 plan.

1. Decide the real short-term gate.
   - Option A: keep 70% and add enough real tests before merging.
   - Option B: lower the gate temporarily to the measured baseline plus small buffer, for example 50%, and create a tracked issue to raise it.
   - Recommended: use Option B only if immediate CI recovery is required; do not leave the Phase 4 doc claiming 76.1%.

2. Update `docs/specs/2026-05-09-pdf2md-tui-qa-phase4-plan.md`.
   - Mark the coverage result as disputed or superseded.
   - Replace claimed 76.1% with the current measured clean-checkout value after fixture fixes.
   - Add a section explaining that ignored fixtures caused local/CI divergence.

3. Improve coverage accounting.
   - Generate coverage only after the full suite passes.
   - Upload `coverage.out` as an artifact on every test run, not only to Codecov.
   - Keep Codecov non-blocking if desired, but make the local `go tool cover` gate authoritative.

Acceptance criteria:

- Coverage number in docs equals a clean-checkout command result.
- CI coverage threshold is realistic and explicit.
- Raising the threshold is tracked as planned work, not implied as complete.

### Phase 2: Repair Local Developer Automation

Goal: make local checks match CI.

1. Add a `.golangci.yml` or `.golangci.yaml`.
   - Pin enabled linters intentionally instead of relying on action defaults.
   - Include a timeout, for example `timeout: 5m`.
   - Keep `staticcheck`, `govet`, `errcheck`, `ineffassign`, and `unused`.

2. Align GitHub and local lint versions.
   - CI currently uses `golangci/golangci-lint-action@v7` with `version: v2.9.0`.
   - Local installed version observed: `v2.12.2`.
   - Choose one version and make it explicit via `.golangci-lint-version` or Makefile documentation.

3. Fix `lefthook.yml`.
   - Avoid `golangci-lint run {staged_files}` for Go packages. Golangci-lint is package-oriented and staged file lists are fragile.
   - Use `golangci-lint run ./...` for pre-push.
   - For pre-commit, either run `golangci-lint run ./...` or keep pre-commit to `gofmt` plus `go vet ./...`.

4. Fix `Makefile` quality target behavior.
   - `make lint` should fail if golangci-lint is installed and reports issues.
   - Consider failing when golangci-lint is missing in CI-like mode, for example `CI=1 make lint`.

Acceptance criteria:

- `make check` gives the same pass/fail outcome as CI.
- `lefthook run pre-push` catches the same issues before pushing.
- No known local-only golangci-lint loader failure remains.

### Phase 3: Harden CI Pipeline

Goal: make CI reliable, fast enough, and useful for release decisions.

1. Add explicit workflow concurrency.
   - Cancel stale runs on the same branch:
     ```yaml
     concurrency:
       group: ${{ github.workflow }}-${{ github.ref }}
       cancel-in-progress: true
     ```

2. Split CI jobs by purpose.
   - `lint`: golangci-lint and actionlint.
   - `test`: race tests and coverage gate on Ubuntu; race smoke on macOS.
   - `fuzz-seed`: short fuzz run on Ubuntu only.
   - `build`: CGO-disabled build for release target parity.
   - `security`: govulncheck and dependency review where event type supports it.

3. Add security checks.
   - Use `golang/govulncheck-action@v1` with `go-version-file: go.mod` and `go-package: ./...`.
   - Add `actions/dependency-review-action@v4` for pull requests.
   - Add `actionlint` for workflow syntax. Use a pinned install method or a maintained action.

4. Improve build verification.
   - Build with GoReleaser snapshot in CI, or at least run `goreleaser check` once GoReleaser is installed.
   - Keep `go build -o ./bin/pdf2md-tui ./cmd/pdf2md-tui` as the fast smoke build.

5. Upload diagnostics consistently.
   - Upload `coverage.out`.
   - Upload benchmark outputs.
   - Upload GoReleaser snapshot artifacts only on manual or release validation workflows, not every push if artifact size is high.

Acceptance criteria:

- CI clearly identifies whether failures are lint, unit test, coverage, fuzz, security, build, or release-config failures.
- A clean checkout and CI behave the same.
- Security checks run without requiring release secrets.

### Phase 4: Strengthen Release Workflow

Goal: prevent publishing artifacts unless the same quality gates passed.

1. Restrict tag trigger.
   - Replace broad `v*` with a stricter semver pattern as much as GitHub Actions allows:
     ```yaml
     tags:
       - 'v[0-9]+.[0-9]+.[0-9]+'
     ```
   - Add an explicit shell validation step because workflow glob patterns are not full regex.

2. Reuse CI-equivalent gates in release.
   - Release workflow should run:
     - lint
     - race tests
     - coverage gate
     - fuzz seed jobs or at least unit fuzz corpus execution
     - `goreleaser check`
   - Only then run `goreleaser release --clean`.

3. Validate release secrets.
   - `GITHUB_TOKEN` is enough for releases in the current repo.
   - `HOMEBREW_TAP_TOKEN` is required for pushing to `nawodyaishan/homebrew-tap`.
   - Add a preflight step that fails early with a clear message when `HOMEBREW_TAP_TOKEN` is absent.

4. Add a manual release dry-run workflow.
   - `workflow_dispatch` input: tag or version.
   - Run `goreleaser release --snapshot --clean`.
   - Do not publish.

5. Keep Homebrew verification.
   - The tap currently contains `Formula/pdf2md-tui.rb` for `v1.2.9`.
   - After release, verify formula update and run:
     - `brew tap nawodyaishan/tap`
     - `brew install pdf2md-tui`
     - `pdf2md-tui version`

Acceptance criteria:

- A malformed tag cannot publish.
- Release cannot run if tests or coverage fail.
- GoReleaser config is validated before publish.
- Homebrew tap update failure is visible as a release failure, not discovered manually later.

### Phase 5: Clean Repository Hygiene

Goal: remove drift that created the current CI/local mismatch.

1. Review `.gitignore` for over-broad directory rules.
   - Use root-anchored ignores for root binaries and root fixture dumps:
     - `/debug-pdf`
     - `/pdf2md-tui`
     - `/testdata/`
   - Do not ignore package-local `testdata/` folders.

2. Remove generated artifacts from working tree when not needed.
   - `coverage.out`
   - `bench.txt`
   - root binaries
   - generated logs

3. Add a clean-checkout validation note to contributor docs.
   - Required command:
     ```bash
     git clean -ndX
     go test -race -coverprofile=coverage.out -covermode=atomic ./...
     ```
   - Do not require contributors to delete ignored local sample PDFs, but CI fixtures must be tracked or generated.

Acceptance criteria:

- A clone from GitHub has every file needed for CI.
- Ignored files are only local artifacts, secrets, or large sample data not required by default tests.

## Proposed Workflow Changes

### `.github/workflows/ci.yml`

Keep existing triggers:

```yaml
on:
  push:
    branches: [main, dev, staging]
  pull_request:
```

Add or revise jobs:

- `lint`: checkout, setup-go, golangci-lint, actionlint.
- `test`: matrix on Ubuntu and macOS; run race tests; coverage gate only on Ubuntu.
- `fuzz`: Ubuntu only; short fuzz time.
- `build`: depends on lint and test; build binary and run `version`.
- `security`: govulncheck on push and PR; dependency review on PR.
- `benchmarks`: keep PR-only, but avoid checking out `origin/main` in a way that destroys current artifacts unexpectedly.

### `.github/workflows/release.yml`

Keep tag-triggered publishing, but add:

- semver validation
- coverage gate
- `goreleaser check`
- Homebrew token preflight
- optional environment protection for release publishing

## Required Secrets and Permissions

- `GITHUB_TOKEN`: built-in, requires `contents: write` for release publishing.
- `HOMEBREW_TAP_TOKEN`: PAT with permission to push to `nawodyaishan/homebrew-tap`.
- `CODECOV_TOKEN`: optional if Codecov is used; keep upload non-blocking unless Codecov is part of branch protection.

Recommended workflow permissions:

- CI default: `contents: read`.
- Dependency review PR job: `contents: read`; add `pull-requests: write` only if PR comments are enabled.
- Release: `contents: write`, `id-token: write` only if artifact signing or provenance uses it.

## Verification Matrix

Run before pushing fixes:

```bash
gofmt -w internal/handler/cli/e2e_test.go pkg/repository/pdf/pdf_test.go pkg/repository/sysinfo/sysinfo_test.go
go test -race -coverprofile=coverage.out -covermode=atomic ./...
go tool cover -func=coverage.out | tail -n 1
golangci-lint run ./...
go test -fuzz=FuzzCoalesceChars -fuzztime=30s ./pkg/repository/pdf/
go test -fuzz=FuzzApplyLLMOptimizations -fuzztime=30s ./pkg/service/
go build -o ./bin/pdf2md-tui ./cmd/pdf2md-tui
./bin/pdf2md-tui version
```

Run after CI config changes:

```bash
goreleaser check
goreleaser release --snapshot --clean
```

If `goreleaser` or `actionlint` is not installed locally, install them or rely on CI jobs that install pinned versions.

## Rollback Plan

If the CI repair causes unexpected failures:

1. Revert only the workflow or test-fixture change that introduced the regression.
2. Keep staticcheck fixes because they are safe and directly correct.
3. Temporarily set the coverage gate to the last verified clean-checkout baseline.
4. Do not publish a new release tag until `dev` and `main` both pass CI.

## Implementation Order

1. Commit staticcheck fixes and `.gitignore` corrections.
2. Commit package-local testscript fixtures and `cmd/debug-pdf/main_test.go`.
3. Re-run clean-checkout tests and record actual coverage.
4. Update the Phase 4 QA doc with measured coverage.
5. Adjust CI coverage gate to match policy.
6. Add local lint config and lefthook fixes.
7. Add security and release hardening jobs.
8. Add release dry-run workflow.

## Open Decisions

- Should the coverage gate remain at 70% and block until more tests are added, or should it be reset to the measured clean-checkout baseline and raised incrementally?
- Should large root-level `testdata/` PDFs stay ignored and be used only for manual/integration tests, or should a minimal generated fixture corpus be committed?
- Should release publishing require a protected GitHub environment approval?
- Should Codecov failure remain non-blocking, or should branch protection require the local coverage gate only?
