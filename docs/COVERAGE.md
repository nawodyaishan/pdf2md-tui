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
