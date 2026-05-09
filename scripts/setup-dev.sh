#!/bin/bash
# Development environment setup script for pdf2md-tui
# Run: bash scripts/setup-dev.sh

set -e

echo "🔧 Setting up pdf2md-tui development environment..."
echo ""

# Check Go version
GO_VERSION=$(go version | awk '{print $3}')
echo "✓ Go version: $GO_VERSION"

# Check/Install lefthook
if ! command -v lefthook &> /dev/null; then
    echo ""
    echo "📦 Installing lefthook..."
    if command -v brew &> /dev/null; then
        brew install lefthook
    else
        go install github.com/evilmartians/lefthook/v2@v2.1.6
    fi
    echo "✓ lefthook installed"
else
    echo "✓ lefthook already installed: $(lefthook version)"
fi

# Install git hooks
echo ""
echo "🪝 Installing git hooks..."
lefthook install
echo "✓ Git hooks installed"

# Check/Install golangci-lint (optional)
echo ""
if ! command -v golangci-lint &> /dev/null; then
    echo "⚠️  golangci-lint not found (optional for linting)"
    echo ""
    echo "To install golangci-lint:"
    echo "  brew install golangci-lint"
    echo "  # OR"
    echo "  go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest"
    echo ""
else
    echo "✓ golangci-lint installed: $(golangci-lint version)"
fi

# Run go mod tidy
echo ""
echo "📚 Running go mod tidy..."
go mod tidy
echo "✓ Dependencies tidied"

# Show next steps
echo ""
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "✅ Development environment ready!"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo ""
echo "Next steps:"
echo "  make test          # Run tests"
echo "  make build         # Build binary"
echo "  make lint          # Lint code"
echo "  make cover         # View coverage report"
echo ""
echo "Git hooks are active:"
echo "  - pre-commit: gofmt, go vet, golangci-lint"
echo "  - pre-push: golangci-lint, go test -race, go build"
echo ""
