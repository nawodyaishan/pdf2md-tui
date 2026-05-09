.PHONY: build test lint fmt vet tidy snapshot cover cover-check bench run clean release tag help \
	check hooks-install hooks-run-pre-commit hooks-run-pre-push hooks-validate

APP_NAME    := pdf2md-tui
BIN_DIR     := bin
BINARY      := $(BIN_DIR)/$(APP_NAME)
COVERAGE_THRESHOLD ?= 45

VERSION     := $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
COMMIT      := $(shell git rev-parse --short HEAD 2>/dev/null || echo "none")
DATE        := $(shell date -u +%Y-%m-%dT%H:%M:%SZ)
GOVERSION   := $(shell go version | awk '{print $$3}')

PKG_VERSION := github.com/nawodyaishan/pdf2md-tui/pkg/version
LDFLAGS     := -s -w \
  -X $(PKG_VERSION).Version=$(VERSION) \
  -X $(PKG_VERSION).Commit=$(COMMIT) \
  -X $(PKG_VERSION).Date=$(DATE) \
  -X $(PKG_VERSION).GoVersion=$(GOVERSION)

# ── Build ──────────────────────────────────────────────────────────────────────

build: $(BIN_DIR) ## Build the binary to bin/pdf2md-tui
	go build -ldflags="$(LDFLAGS)" -o $(BINARY) ./cmd/pdf2md-tui

$(BIN_DIR):
	mkdir -p $(BIN_DIR)

run: build ## Build and run a quick smoke test against ./testdata
	./$(BINARY) convert ./testdata --verbose

# ── Quality ────────────────────────────────────────────────────────────────────

test: ## Run the test suite with race detection and coverage
	go test -v -race -coverprofile=coverage.out -covermode=atomic ./...

cover: test ## Open HTML coverage report in the browser
	go tool cover -html=coverage.out

cover-check: test ## Check that coverage meets the configured minimum threshold
	@go tool cover -func=coverage.out \
		| awk -v threshold="$(COVERAGE_THRESHOLD)" '/total:/{cov=$$3+0; if(cov<threshold){printf "FAIL: Coverage %.1f%% < %.1f%% threshold\n",cov,threshold; exit 1} else {printf "OK: Coverage %.1f%%\n",cov}}'

bench: ## Run benchmarks and output results
	go test -bench=. -benchmem -count=5 ./pkg/... 2>/dev/null | tee bench.txt
	@echo "Benchmark results written to bench.txt"

lint: ## Run golangci-lint
	@command -v golangci-lint >/dev/null 2>&1 || { \
		echo "ERROR: golangci-lint is required. Install it with 'brew install golangci-lint'."; \
		exit 1; \
	}
	golangci-lint run ./...

fmt: ## Format all Go source files
	gofmt -w $$(find . -name '*.go' -not -path './vendor/*')

vet: ## Run go vet
	go vet ./...

tidy: ## Tidy go.mod and go.sum
	go mod tidy

check: fmt vet lint test cover-check ## Run all quality checks (fmt, vet, lint, test, cover-check)

ci-local: fmt vet lint cover-check ## Simulate CI locally (all checks in sequence)
	@echo "✅ All CI checks passed locally"

hooks-install: ## Install Git hooks via Lefthook
	@command -v lefthook >/dev/null 2>&1 || { \
		echo "ERROR: lefthook is required. Install it with 'brew install lefthook' or 'go install github.com/evilmartians/lefthook/v2@v2.1.6'."; \
		exit 1; \
	}
	lefthook install

hooks-run-pre-commit: ## Run the configured pre-commit hook locally
	@command -v lefthook >/dev/null 2>&1 || { \
		echo "ERROR: lefthook is required. Install it with 'brew install lefthook' or 'go install github.com/evilmartians/lefthook/v2@v2.1.6'."; \
		exit 1; \
	}
	lefthook run pre-commit

hooks-run-pre-push: ## Run the configured pre-push hook locally
	@command -v lefthook >/dev/null 2>&1 || { \
		echo "ERROR: lefthook is required. Install it with 'brew install lefthook' or 'go install github.com/evilmartians/lefthook/v2@v2.1.6'."; \
		exit 1; \
	}
	lefthook run pre-push

hooks-validate: ## Validate lefthook.yml
	@command -v lefthook >/dev/null 2>&1 || { \
		echo "ERROR: lefthook is required. Install it with 'brew install lefthook' or 'go install github.com/evilmartians/lefthook/v2@v2.1.6'."; \
		exit 1; \
	}
	lefthook validate

# ── Release ────────────────────────────────────────────────────────────────────

snapshot: ## Build a local GoReleaser snapshot (no publish, no tag required)
	goreleaser release --snapshot --clean

# Usage: make release V=v1.2.0 MSG="what changed in this release"
release: ## Tag and push a new release  [V=<version> MSG=<note>]
ifndef V
	$(error V is required. Usage: make release V=v1.2.0 MSG="release note")
endif
ifndef MSG
	$(error MSG is required. Usage: make release V=v1.2.0 MSG="release note")
endif
	@if ! echo "$(V)" | grep -qE '^v[0-9]+\.[0-9]+\.[0-9]+$$'; then \
		echo "ERROR: V must be a semver tag like v1.2.0 (got '$(V)')"; exit 1; \
	fi
	@if git rev-parse "$(V)" >/dev/null 2>&1; then \
		echo "ERROR: tag $(V) already exists"; exit 1; \
	fi
	@echo "▶ Running pre-release checks (fmt, vet, test)..."
	@$(MAKE) fmt vet test
	@echo "▶ Tagging $(V)..."
	git tag -a "$(V)" -m "$(MSG)"
	@echo "▶ Pushing tag $(V) → origin (triggers GoReleaser CI)..."
	git push origin "$(V)"
	@echo "✓ Released $(V). Monitor: https://github.com/nawodyaishan/pdf2md-tui/actions"

# Usage: make tag V=v1.2.0 MSG="what changed" — tag only, no push
tag: ## Create an annotated git tag without pushing  [V=<version> MSG=<note>]
ifndef V
	$(error V is required. Usage: make tag V=v1.2.0 MSG="release note")
endif
ifndef MSG
	$(error MSG is required. Usage: make tag V=v1.2.0 MSG="release note")
endif
	@if ! echo "$(V)" | grep -qE '^v[0-9]+\.[0-9]+\.[0-9]+$$'; then \
		echo "ERROR: V must be a semver tag like v1.2.0 (got '$(V)')"; exit 1; \
	fi
	git tag -a "$(V)" -m "$(MSG)"
	@echo "✓ Tagged $(V) locally. Run 'git push origin $(V)' to trigger release."

# ── Utilities ──────────────────────────────────────────────────────────────────

clean: ## Remove build artifacts
	rm -rf $(BIN_DIR)/ dist/ coverage.out coverage.html

help: ## Show this help message
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / {printf "\033[36m%-12s\033[0m %s\n", $$1, $$2}' $(MAKEFILE_LIST)

.DEFAULT_GOAL := help
