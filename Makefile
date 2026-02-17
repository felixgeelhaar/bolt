# Makefile for Bolt - Zero-allocation structured logging library
# This Makefile provides common development tasks and ensures consistency

.PHONY: help install-tools setup-hooks lint format test build clean benchmark security pre-commit install-deps check-tools

# Default target
help: ## Show this help message
	@echo "🚀 Bolt - Zero-allocation structured logging library"
	@echo ""
	@echo "Available targets:"
	@awk 'BEGIN {FS = ":.*##"; printf "\nUsage:\n  make \033[36m<target>\033[0m\n"} /^[a-zA-Z_-]+:.*?##/ { printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2 } /^##@/ { printf "\n\033[1m%s\033[0m\n", substr($$0, 5) } ' $(MAKEFILE_LIST)

##@ Setup and Installation

install-deps: ## Install Go dependencies
	@echo "📦 Installing Go dependencies..."
	@go mod download
	@go mod tidy
	@echo "✅ Dependencies installed"

install-tools: ## Install development tools (golangci-lint, goimports, etc.)
	@echo "🔧 Installing development tools..."
	@# Install golangci-lint
	@if ! command -v golangci-lint >/dev/null 2>&1; then \
		echo "Installing golangci-lint..."; \
		go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest; \
	else \
		echo "✅ golangci-lint already installed"; \
	fi
	@# Install goimports
	@if ! command -v goimports >/dev/null 2>&1; then \
		echo "Installing goimports..."; \
		go install golang.org/x/tools/cmd/goimports@latest; \
	else \
		echo "✅ goimports already installed"; \
	fi
	@# Install govulncheck
	@if ! command -v govulncheck >/dev/null 2>&1; then \
		echo "Installing govulncheck..."; \
		go install golang.org/x/vuln/cmd/govulncheck@latest; \
	else \
		echo "✅ govulncheck already installed"; \
	fi
	@# Install benchstat
	@if ! command -v benchstat >/dev/null 2>&1; then \
		echo "Installing benchstat..."; \
		go install golang.org/x/perf/cmd/benchstat@latest; \
	else \
		echo "✅ benchstat already installed"; \
	fi
	@echo "✅ All development tools installed"

setup-hooks: ## Set up Git hooks for automated code quality checks
	@echo "🪝 Setting up Git hooks..."
	@# Create hooks directory if it doesn't exist
	@mkdir -p .git/hooks
	@# Install pre-commit hook
	@cp .githooks/pre-commit .git/hooks/pre-commit
	@chmod +x .git/hooks/pre-commit
	@echo "✅ Git hooks installed"
	@echo "💡 Tip: You can skip hooks with 'git commit --no-verify'"

check-tools: ## Check if required tools are installed
	@echo "🔍 Checking required tools..."
	@command -v go >/dev/null 2>&1 || { echo "❌ Go is not installed"; exit 1; }
	@echo "✅ Go: $(shell go version)"
	@command -v golangci-lint >/dev/null 2>&1 || { echo "⚠️  golangci-lint not found (run 'make install-tools')"; }
	@command -v goimports >/dev/null 2>&1 || { echo "⚠️  goimports not found (run 'make install-tools')"; }
	@command -v govulncheck >/dev/null 2>&1 || { echo "⚠️  govulncheck not found (run 'make install-tools')"; }
	@command -v benchstat >/dev/null 2>&1 || { echo "⚠️  benchstat not found (run 'make install-tools')"; }

##@ Code Quality

format: ## Format Go code and organize imports
	@echo "📝 Formatting Go code..."
	@find . -name '*.go' -not -path './vendor/*' -exec gofmt -s -w {} +
	@if command -v goimports >/dev/null 2>&1; then \
		find . -name '*.go' -not -path './vendor/*' -exec goimports -w {} +; \
		echo "✅ Code formatted and imports organized"; \
	else \
		echo "⚠️  goimports not found, only gofmt applied"; \
	fi

lint: ## Run golangci-lint with project configuration
	@echo "🧹 Running golangci-lint..."
	@if command -v golangci-lint >/dev/null 2>&1; then \
		golangci-lint run --config .golangci.yml --timeout 5m; \
		echo "✅ Linting completed"; \
	else \
		echo "❌ golangci-lint not found. Install with 'make install-tools'"; \
		exit 1; \
	fi

lint-fix: ## Run golangci-lint with automatic fixes
	@echo "🧹 Running golangci-lint with auto-fix..."
	@if command -v golangci-lint >/dev/null 2>&1; then \
		golangci-lint run --config .golangci.yml --timeout 5m --fix; \
		echo "✅ Linting completed with auto-fixes"; \
	else \
		echo "❌ golangci-lint not found. Install with 'make install-tools'"; \
		exit 1; \
	fi

vet: ## Run go vet
	@echo "🔍 Running go vet..."
	@go vet ./...
	@echo "✅ go vet completed"

##@ Testing

test: ## Run all tests
	@echo "🧪 Running tests..."
	@go test ./...
	@echo "✅ All tests passed"

test-race: ## Run tests with race detection
	@echo "🧪 Running tests with race detection..."
	@go test -race ./...
	@echo "✅ Race tests passed"

test-verbose: ## Run tests with verbose output
	@echo "🧪 Running tests (verbose)..."
	@go test -v ./...

test-cover: ## Run tests with coverage report
	@echo "🧪 Running tests with coverage..."
	@go test -coverprofile=coverage.out ./...
	@go tool cover -html=coverage.out -o coverage.html
	@echo "✅ Coverage report generated: coverage.html"

##@ Build

build: ## Build the project
	@echo "🔨 Building project..."
	@go build ./...
	@echo "✅ Build completed"

build-race: ## Build with race detection
	@echo "🔨 Building with race detection..."
	@go build -race ./...
	@echo "✅ Race build completed"

##@ Performance

benchmark: ## Run performance benchmarks
	@echo "⚡ Running benchmarks..."
	@go test -bench=. -benchmem ./...

benchmark-zero: ## Run zero-allocation benchmarks specifically
	@echo "⚡ Running zero-allocation benchmarks..."
	@go test -bench=BenchmarkZeroAllocation -benchmem ./...

benchmark-compare: ## Run benchmarks and compare with previous results
	@echo "⚡ Running benchmark comparison..."
	@if [ -f benchmarks.txt ]; then \
		go test -bench=. -benchmem -count=5 ./... > benchmarks.new.txt; \
		benchstat benchmarks.txt benchmarks.new.txt; \
		mv benchmarks.new.txt benchmarks.txt; \
	else \
		go test -bench=. -benchmem -count=5 ./... > benchmarks.txt; \
		echo "✅ Baseline benchmarks saved to benchmarks.txt"; \
	fi

##@ Security

security: ## Run security checks
	@echo "🔒 Running security checks..."
	@if command -v govulncheck >/dev/null 2>&1; then \
		govulncheck ./...; \
		echo "✅ Vulnerability check completed"; \
	else \
		echo "⚠️  govulncheck not found, running go vet only"; \
		go vet ./...; \
	fi

##@ Maintenance

clean: ## Clean build artifacts and temporary files
	@echo "🧹 Cleaning build artifacts..."
	@go clean ./...
	@rm -f coverage.out coverage.html
	@rm -f benchmarks.txt benchmarks.new.txt
	@rm -rf dist/
	@echo "✅ Clean completed"

mod-tidy: ## Tidy go modules
	@echo "📋 Tidying go modules..."
	@go mod tidy
	@echo "✅ Modules tidied"

mod-verify: ## Verify go modules
	@echo "🔍 Verifying go modules..."
	@go mod verify
	@echo "✅ Modules verified"

##@ Quality Assurance

pre-commit: format vet lint test ## Run all pre-commit checks
	@echo "🚀 All pre-commit checks passed!"

ci: install-deps vet lint test-race build ## Run CI pipeline checks
	@echo "🚀 All CI checks passed!"

full-check: clean install-deps format vet lint test-race build benchmark security ## Run comprehensive quality checks
	@echo "🚀 Full quality check completed!"

##@ Information

version: ## Show Go and tool versions
	@echo "📋 Version Information:"
	@echo "Go: $(shell go version)"
	@if command -v golangci-lint >/dev/null 2>&1; then \
		echo "golangci-lint: $(shell golangci-lint --version)"; \
	fi
	@if command -v goimports >/dev/null 2>&1; then \
		echo "goimports: installed"; \
	fi
	@if command -v govulncheck >/dev/null 2>&1; then \
		echo "govulncheck: installed"; \
	fi

deps: ## Show dependency information
	@echo "📦 Dependencies:"
	@go list -m all

##@ Development

dev-setup: install-deps install-tools setup-hooks check-tools ## Complete development environment setup
	@echo "🎉 Development environment is ready!"
	@echo ""
	@echo "🚀 Quick start:"
	@echo "  make format    # Format code"
	@echo "  make lint      # Run linter"
	@echo "  make test      # Run tests"
	@echo "  make pre-commit # Run all pre-commit checks"

.DEFAULT_GOAL := help