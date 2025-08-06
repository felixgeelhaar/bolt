# Bolt Benchmark Suite Makefile
# Professional build system for comprehensive performance testing

# Variables
BINARY_NAME := bolt-benchmark
OUTPUT_DIR := benchmark-results
VERSION := $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
COMMIT := $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
BUILD_DATE := $(shell date -u '+%Y-%m-%d_%H:%M:%S')
GO_VERSION := $(shell go version | awk '{print $$3}')

# Build flags
LDFLAGS := -ldflags "-X main.version=$(VERSION) -X main.commit=$(COMMIT) -X main.date=$(BUILD_DATE)"
BUILD_FLAGS := -v -trimpath $(LDFLAGS)
BUILD_DIR := ./cmd/bolt-benchmark

# Go tools
GOCMD := go
GOBUILD := $(GOCMD) build
GOTEST := $(GOCMD) test
GOCLEAN := $(GOCMD) clean
GOMOD := $(GOCMD) mod
GOFMT := $(GOCMD) fmt

# Test flags
TEST_FLAGS := -v -race -coverprofile=coverage.out
BENCH_FLAGS := -bench=. -benchmem -count=5 -timeout=30m

# Colors for output
RED := \033[0;31m
GREEN := \033[0;32m
YELLOW := \033[1;33m
BLUE := \033[0;34m
NC := \033[0m # No Color

.PHONY: all build build-all test bench clean install uninstall help
.PHONY: competitive enterprise regression load validate
.PHONY: docker docker-build docker-run ci-setup
.PHONY: deps lint format check coverage
.DEFAULT_GOAL := help

## Build Commands

all: clean deps lint test build ## Run all quality checks and build

build: ## Build benchmark tool for current platform
	@echo "$(BLUE)Building $(BINARY_NAME) v$(VERSION)$(NC)"
	@mkdir -p bin/
	@cd $(BUILD_DIR) && $(GOBUILD) $(BUILD_FLAGS) -o ../../bin/$(BINARY_NAME)
	@echo "$(GREEN)✅ Built bin/$(BINARY_NAME)$(NC)"

build-all: ## Build for all supported platforms
	@echo "$(BLUE)Building $(BINARY_NAME) for all platforms$(NC)"
	@mkdir -p dist/
	@for os in linux darwin windows; do \
		for arch in amd64 arm64; do \
			if [ "$$os" = "windows" ]; then \
				ext=".exe"; \
			else \
				ext=""; \
			fi; \
			echo "Building $$os/$$arch..."; \
			cd $(BUILD_DIR) && GOOS=$$os GOARCH=$$arch $(GOBUILD) $(BUILD_FLAGS) \
				-o ../../dist/$(BINARY_NAME)-$$os-$$arch$$ext; \
		done; \
	done
	@echo "$(GREEN)✅ Built all platform binaries in dist/$(NC)"

install: build ## Install benchmark tool to system
	@echo "$(BLUE)Installing $(BINARY_NAME) to /usr/local/bin$(NC)"
	@sudo cp bin/$(BINARY_NAME) /usr/local/bin/
	@echo "$(GREEN)✅ Installed $(BINARY_NAME)$(NC)"

uninstall: ## Remove benchmark tool from system
	@echo "$(YELLOW)Uninstalling $(BINARY_NAME)$(NC)"
	@sudo rm -f /usr/local/bin/$(BINARY_NAME)
	@echo "$(GREEN)✅ Uninstalled $(BINARY_NAME)$(NC)"

## Development Commands

deps: ## Download and verify dependencies
	@echo "$(BLUE)Downloading dependencies$(NC)"
	@$(GOMOD) download
	@$(GOMOD) verify
	@$(GOMOD) tidy
	@echo "$(GREEN)✅ Dependencies ready$(NC)"

lint: ## Run linting tools
	@echo "$(BLUE)Running linters$(NC)"
	@if command -v golangci-lint >/dev/null 2>&1; then \
		golangci-lint run --timeout=5m; \
	else \
		echo "$(YELLOW)⚠️  golangci-lint not installed, skipping$(NC)"; \
	fi
	@echo "$(GREEN)✅ Linting complete$(NC)"

format: ## Format Go code
	@echo "$(BLUE)Formatting code$(NC)"
	@$(GOFMT) ./...
	@echo "$(GREEN)✅ Code formatted$(NC)"

check: deps lint format ## Run all code quality checks
	@echo "$(GREEN)✅ All checks passed$(NC)"

test: ## Run all tests
	@echo "$(BLUE)Running tests$(NC)"
	@$(GOTEST) $(TEST_FLAGS) ./...
	@echo "$(GREEN)✅ Tests passed$(NC)"

bench: ## Run Go benchmarks
	@echo "$(BLUE)Running Go benchmarks$(NC)"
	@$(GOTEST) $(BENCH_FLAGS) ./...
	@echo "$(GREEN)✅ Benchmarks complete$(NC)"

coverage: test ## Generate and open coverage report
	@echo "$(BLUE)Generating coverage report$(NC)"
	@$(GOCMD) tool cover -html=coverage.out -o coverage.html
	@echo "$(GREEN)✅ Coverage report: coverage.html$(NC)"
	@if command -v open >/dev/null 2>&1; then \
		open coverage.html; \
	elif command -v xdg-open >/dev/null 2>&1; then \
		xdg-open coverage.html; \
	fi

## Benchmark Commands

competitive: build ## Run competitive analysis benchmark
	@echo "$(BLUE)Running competitive analysis$(NC)"
	@mkdir -p $(OUTPUT_DIR)
	@./bin/$(BINARY_NAME) \
		-type=competitive \
		-duration=10m \
		-output=$(OUTPUT_DIR)/competitive \
		-html=true \
		-csv=true \
		-verbose=true
	@echo "$(GREEN)✅ Competitive analysis complete: $(OUTPUT_DIR)/competitive$(NC)"

enterprise: build ## Run enterprise scenario benchmarks
	@echo "$(BLUE)Running enterprise benchmarks$(NC)"
	@mkdir -p $(OUTPUT_DIR)
	@./bin/$(BINARY_NAME) \
		-type=enterprise \
		-duration=15m \
		-output=$(OUTPUT_DIR)/enterprise \
		-max-memory=4096 \
		-max-cpu=95 \
		-html=true \
		-csv=true \
		-verbose=true
	@echo "$(GREEN)✅ Enterprise benchmarks complete: $(OUTPUT_DIR)/enterprise$(NC)"

regression: build ## Run regression testing
	@echo "$(BLUE)Running regression tests$(NC)"
	@mkdir -p $(OUTPUT_DIR)
	@./bin/$(BINARY_NAME) \
		-type=regression \
		-duration=5m \
		-output=$(OUTPUT_DIR)/regression \
		-iterations=10 \
		-verbose=true
	@echo "$(GREEN)✅ Regression tests complete: $(OUTPUT_DIR)/regression$(NC)"

load: build ## Run load testing
	@echo "$(BLUE)Running load tests$(NC)"
	@mkdir -p $(OUTPUT_DIR)
	@./bin/$(BINARY_NAME) \
		-type=load \
		-duration=20m \
		-output=$(OUTPUT_DIR)/load \
		-parallel=16 \
		-profiling=true \
		-verbose=true
	@echo "$(GREEN)✅ Load tests complete: $(OUTPUT_DIR)/load$(NC)"

validate: build ## Validate performance against thresholds
	@echo "$(BLUE)Validating performance$(NC)"
	@if [ -d "$(OUTPUT_DIR)" ]; then \
		./bin/$(BINARY_NAME) -validate -output=$(OUTPUT_DIR); \
		echo "$(GREEN)✅ Validation complete$(NC)"; \
	else \
		echo "$(YELLOW)⚠️  No results found, run benchmarks first$(NC)"; \
		exit 1; \
	fi

## Comprehensive Test Suite

full-suite: build ## Run complete benchmark suite (takes ~1 hour)
	@echo "$(BLUE)🚀 Starting full benchmark suite$(NC)"
	@echo "$(YELLOW)⏱️  This will take approximately 60 minutes$(NC)"
	@mkdir -p $(OUTPUT_DIR)/full-suite
	
	@echo "$(BLUE)Phase 1/4: Competitive Analysis (10m)$(NC)"
	@./bin/$(BINARY_NAME) \
		-type=competitive \
		-duration=10m \
		-output=$(OUTPUT_DIR)/full-suite/competitive \
		-iterations=10 \
		-html=true -csv=true
	
	@echo "$(BLUE)Phase 2/4: Enterprise Scenarios (20m)$(NC)"
	@./bin/$(BINARY_NAME) \
		-type=enterprise \
		-duration=20m \
		-output=$(OUTPUT_DIR)/full-suite/enterprise \
		-html=true -csv=true
	
	@echo "$(BLUE)Phase 3/4: Regression Testing (10m)$(NC)"
	@./bin/$(BINARY_NAME) \
		-type=regression \
		-duration=10m \
		-output=$(OUTPUT_DIR)/full-suite/regression \
		-iterations=15
	
	@echo "$(BLUE)Phase 4/4: Load Testing (20m)$(NC)"
	@./bin/$(BINARY_NAME) \
		-type=load \
		-duration=20m \
		-output=$(OUTPUT_DIR)/full-suite/load \
		-parallel=32 \
		-profiling=true
	
	@echo "$(GREEN)🎉 Full benchmark suite complete: $(OUTPUT_DIR)/full-suite$(NC)"
	@echo "$(BLUE)📊 Summary reports available in each subdirectory$(NC)"

quick-test: build ## Run quick benchmark test (5 minutes)
	@echo "$(BLUE)Running quick benchmark test$(NC)"
	@mkdir -p $(OUTPUT_DIR)
	@./bin/$(BINARY_NAME) \
		-type=competitive \
		-duration=2m \
		-output=$(OUTPUT_DIR)/quick \
		-iterations=3 \
		-libraries=Bolt,Zerolog \
		-scenarios=WebAPI \
		-html=true
	@echo "$(GREEN)✅ Quick test complete: $(OUTPUT_DIR)/quick$(NC)"

## Docker Commands

docker-build: ## Build Docker image for benchmarking
	@echo "$(BLUE)Building Docker image$(NC)"
	@docker build -t bolt-benchmark:$(VERSION) -f Dockerfile .
	@docker tag bolt-benchmark:$(VERSION) bolt-benchmark:latest
	@echo "$(GREEN)✅ Docker image built: bolt-benchmark:$(VERSION)$(NC)"

docker-run: docker-build ## Run benchmarks in Docker container
	@echo "$(BLUE)Running benchmarks in Docker$(NC)"
	@mkdir -p $(OUTPUT_DIR)/docker
	@docker run --rm \
		-v $(PWD)/$(OUTPUT_DIR)/docker:/output \
		-e OUTPUT_DIR=/output \
		bolt-benchmark:latest \
		-type=competitive \
		-duration=5m \
		-output=/output \
		-html=true
	@echo "$(GREEN)✅ Docker benchmarks complete: $(OUTPUT_DIR)/docker$(NC)"

## CI/CD Commands

ci-setup: ## Setup CI environment
	@echo "$(BLUE)Setting up CI environment$(NC)"
	@$(GOMOD) download
	@if [ "$$CI" = "true" ]; then \
		echo "CI=true" >> $$GITHUB_ENV; \
		echo "BENCHMARK_TIMEOUT=30m" >> $$GITHUB_ENV; \
	fi
	@echo "$(GREEN)✅ CI environment ready$(NC)"

ci-test: check test bench ## Run CI test suite
	@echo "$(GREEN)✅ CI tests passed$(NC)"

ci-benchmark: build ## Run CI benchmark suite
	@echo "$(BLUE)Running CI benchmarks$(NC)"
	@mkdir -p $(OUTPUT_DIR)/ci
	@timeout 25m ./bin/$(BINARY_NAME) \
		-type=competitive \
		-duration=5m \
		-output=$(OUTPUT_DIR)/ci \
		-iterations=5 \
		-quiet=true \
		-html=true \
		-csv=true || echo "$(YELLOW)⚠️  Benchmark timeout$(NC)"
	@echo "$(GREEN)✅ CI benchmarks complete$(NC)"

## Utility Commands

clean: ## Clean build artifacts and results
	@echo "$(BLUE)Cleaning build artifacts$(NC)"
	@$(GOCLEAN)
	@rm -rf bin/ dist/ $(OUTPUT_DIR)/ coverage.out coverage.html
	@echo "$(GREEN)✅ Cleaned$(NC)"

version: ## Show version information
	@echo "$(BLUE)Bolt Benchmark Suite$(NC)"
	@echo "Version:    $(VERSION)"
	@echo "Commit:     $(COMMIT)"
	@echo "Build Date: $(BUILD_DATE)"
	@echo "Go Version: $(GO_VERSION)"

list-scenarios: build ## List available benchmark scenarios
	@echo "$(BLUE)Available benchmark scenarios:$(NC)"
	@./bin/$(BINARY_NAME) -list-scenarios

status: ## Show benchmark results status
	@echo "$(BLUE)Benchmark Results Status:$(NC)"
	@if [ -d "$(OUTPUT_DIR)" ]; then \
		echo "Results directory: $(OUTPUT_DIR)/"; \
		find $(OUTPUT_DIR) -name "*.html" -type f | head -10 | while read file; do \
			echo "  📊 $$file"; \
		done; \
		find $(OUTPUT_DIR) -name "*.json" -type f | head -10 | while read file; do \
			echo "  📋 $$file"; \
		done; \
	else \
		echo "$(YELLOW)No results found. Run 'make competitive' to get started.$(NC)"; \
	fi

open-results: ## Open latest HTML report
	@echo "$(BLUE)Opening latest results$(NC)"
	@latest=$$(find $(OUTPUT_DIR) -name "performance_report.html" -type f -exec ls -t {} + | head -1); \
	if [ -n "$$latest" ]; then \
		echo "Opening: $$latest"; \
		if command -v open >/dev/null 2>&1; then \
			open "$$latest"; \
		elif command -v xdg-open >/dev/null 2>&1; then \
			xdg-open "$$latest"; \
		else \
			echo "Please open: $$latest"; \
		fi; \
	else \
		echo "$(YELLOW)No HTML reports found. Run benchmarks first.$(NC)"; \
	fi

## Help

help: ## Show this help message
	@echo "$(BLUE)Bolt Benchmark Suite - Makefile Commands$(NC)"
	@echo ""
	@echo "$(YELLOW)Build Commands:$(NC)"
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | grep -E '^(all|build|build-all|install|uninstall):' | awk 'BEGIN {FS = ":.*?## "}; {printf "  $(GREEN)%-20s$(NC) %s\n", $$1, $$2}'
	@echo ""
	@echo "$(YELLOW)Development Commands:$(NC)"
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | grep -E '^(deps|lint|format|check|test|bench|coverage):' | awk 'BEGIN {FS = ":.*?## "}; {printf "  $(GREEN)%-20s$(NC) %s\n", $$1, $$2}'
	@echo ""
	@echo "$(YELLOW)Benchmark Commands:$(NC)"
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | grep -E '^(competitive|enterprise|regression|load|validate|full-suite|quick-test):' | awk 'BEGIN {FS = ":.*?## "}; {printf "  $(GREEN)%-20s$(NC) %s\n", $$1, $$2}'
	@echo ""
	@echo "$(YELLOW)Docker Commands:$(NC)"
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | grep -E '^(docker-build|docker-run):' | awk 'BEGIN {FS = ":.*?## "}; {printf "  $(GREEN)%-20s$(NC) %s\n", $$1, $$2}'
	@echo ""
	@echo "$(YELLOW)CI/CD Commands:$(NC)"
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | grep -E '^(ci-setup|ci-test|ci-benchmark):' | awk 'BEGIN {FS = ":.*?## "}; {printf "  $(GREEN)%-20s$(NC) %s\n", $$1, $$2}'
	@echo ""
	@echo "$(YELLOW)Utility Commands:$(NC)"
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | grep -E '^(clean|version|list-scenarios|status|open-results|help):' | awk 'BEGIN {FS = ":.*?## "}; {printf "  $(GREEN)%-20s$(NC) %s\n", $$1, $$2}'
	@echo ""
	@echo "$(BLUE)Examples:$(NC)"
	@echo "  make competitive                 # Quick competitive analysis"
	@echo "  make full-suite                  # Complete 1-hour test suite"
	@echo "  make quick-test                  # 5-minute validation test"
	@echo "  make enterprise                  # Enterprise scenarios"
	@echo ""
	@echo "$(BLUE)For more information:$(NC) https://github.com/felixgeelhaar/bolt/tree/main/benchmark"