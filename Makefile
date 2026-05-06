.PHONY: build test lint fmt vet tidy snapshot cover run clean help

APP_NAME    := pdf2md-tui
BIN_DIR     := bin
BINARY      := $(BIN_DIR)/$(APP_NAME)

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
	go test -race -coverprofile=coverage.out ./...

cover: test ## Open HTML coverage report in the browser
	go tool cover -html=coverage.out

lint: ## Run golangci-lint
	golangci-lint run ./...

fmt: ## Format all Go source files
	gofmt -w $$(find . -name '*.go' -not -path './vendor/*')

vet: ## Run go vet
	go vet ./...

tidy: ## Tidy go.mod and go.sum
	go mod tidy

check: fmt vet lint test ## Run all quality checks (fmt, vet, lint, test)

# ── Release ────────────────────────────────────────────────────────────────────

snapshot: ## Build a local GoReleaser snapshot (no publish, no tag required)
	goreleaser release --snapshot --clean

# ── Utilities ──────────────────────────────────────────────────────────────────

clean: ## Remove build artifacts
	rm -rf $(BIN_DIR)/ dist/ coverage.out coverage.html

help: ## Show this help message
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / {printf "\033[36m%-12s\033[0m %s\n", $$1, $$2}' $(MAKEFILE_LIST)

.DEFAULT_GOAL := help
