.PHONY: help test bench lint release check-version tag

# Default target
help:
	@echo "Available targets:"
	@echo "  test          - Run all tests with race detection"
	@echo "  bench         - Run benchmarks"
	@echo "  lint          - Run linters"
	@echo "  check-version - Validate module version consistency"
	@echo "  tag          - Create a new version tag (use VERSION=vX.Y.Z)"
	@echo "  release      - Create and push a new release tag"
	@echo "  release-dry  - Dry run of goreleaser"

# Run tests
test:
	go test -v -race -coverprofile=coverage.txt ./...

# Run benchmarks
bench:
	go test -bench=. -benchmem -run=^$

# Run linters
lint:
	@if command -v golangci-lint >/dev/null 2>&1; then \
		golangci-lint run; \
	else \
		echo "golangci-lint not installed, running go vet instead"; \
		go vet ./...; \
	fi
	go mod tidy
	go mod verify

# Check version consistency
check-version:
	@VERSION=$$(git describe --tags --abbrev=0 2>/dev/null || echo "v0.0.0"); \
	MAJOR=$$(echo $$VERSION | cut -d. -f1); \
	MODULE=$$(grep "^module " go.mod | cut -d' ' -f2); \
	if [ "$$MAJOR" != "v0" ] && [ "$$MAJOR" != "v1" ]; then \
		if [[ ! "$$MODULE" == *"/$$MAJOR" ]]; then \
			echo "❌ Module path must include /$$MAJOR for $$VERSION"; \
			echo "   Current: $$MODULE"; \
			echo "   Expected: github.com/felixgeelhaar/bolt/$$MAJOR"; \
			exit 1; \
		fi; \
	fi; \
	echo "✅ Version $$VERSION is consistent with module path"

# Create a new tag
tag:
	@if [ -z "$(VERSION)" ]; then \
		echo "Usage: make tag VERSION=vX.Y.Z"; \
		exit 1; \
	fi
	@echo "Creating tag $(VERSION)..."
	@MAJOR=$$(echo $(VERSION) | cut -d. -f1); \
	MODULE=$$(grep "^module " go.mod | cut -d' ' -f2); \
	if [ "$$MAJOR" != "v0" ] && [ "$$MAJOR" != "v1" ]; then \
		EXPECTED="github.com/felixgeelhaar/bolt/$$MAJOR"; \
		if [ "$$MODULE" != "$$EXPECTED" ]; then \
			echo "❌ Module path must be $$EXPECTED for $(VERSION)"; \
			echo "   Please update go.mod first"; \
			exit 1; \
		fi; \
	fi
	git tag -a $(VERSION) -m "Release $(VERSION)"
	@echo "✅ Tag $(VERSION) created. Push with: git push origin $(VERSION)"

# Create and push a release
release:
	@if [ -z "$(VERSION)" ]; then \
		echo "Usage: make release VERSION=vX.Y.Z"; \
		exit 1; \
	fi
	$(MAKE) lint
	$(MAKE) test
	$(MAKE) bench
	$(MAKE) tag VERSION=$(VERSION)
	git push origin $(VERSION)
	@echo "✅ Release $(VERSION) pushed. GitHub Actions will handle the rest."

# Dry run of goreleaser (requires goreleaser installed)
release-dry:
	@if command -v goreleaser >/dev/null 2>&1; then \
		goreleaser release --snapshot --clean --skip=publish; \
	else \
		echo "❌ goreleaser not installed"; \
		echo "Install with: go install github.com/goreleaser/goreleaser/v2@latest"; \
		exit 1; \
	fi

# Module maintenance
.PHONY: mod-update mod-clean

mod-update:
	go get -u ./...
	go mod tidy

mod-clean:
	go mod tidy
	go clean -modcache