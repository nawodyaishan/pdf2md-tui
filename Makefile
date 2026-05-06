.PHONY: build test lint snapshot clean help

APP_NAME    := pdf2md-tui
VERSION     := $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
GOVERSION   := $(shell go version | awk '{print $$3}')
LDFLAGS     := -s -w \
  -X github.com/nawodyaishan/pdf2md-tui/pkg/version.Version=$(VERSION) \
  -X github.com/nawodyaishan/pdf2md-tui/pkg/version.Commit=$(shell git rev-parse --short HEAD) \
  -X github.com/nawodyaishan/pdf2md-tui/pkg/version.Date=$(shell date -u +%Y-%m-%dT%H:%M:%SZ) \
  -X github.com/nawodyaishan/pdf2md-tui/pkg/version.GoVersion=$(GOVERSION)

help: ## Show this help message
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / {printf "\033[36m%-15s\033[0m %s\n", $$1, $$2}' $(MAKEFILE_LIST)

build: ## Build the pdf2md-tui binary
	go build -ldflags="$(LDFLAGS)" -o bin/$(APP_NAME) ./cmd/pdf2md-tui

test: ## Run the QA test suite
	go test -race -coverprofile=coverage.out ./...

lint: ## Run golangci-lint
	golangci-lint run ./...

snapshot: ## Build a local snapshot release via GoReleaser
	goreleaser release --snapshot --clean

clean: ## Remove build artifacts
	rm -rf bin/ dist/ coverage.out
