# Development Guide for Bolt

This guide covers the development setup and workflow for the Bolt zero-allocation structured logging library.

## Quick Setup

To set up your development environment:

```bash
# Install all tools and setup hooks
make dev-setup

# Or manually:
make install-deps    # Install Go dependencies
make install-tools   # Install development tools (golangci-lint, goimports, etc.)
make setup-hooks     # Install Git pre-commit hooks
```

## Pre-commit Hooks

Pre-commit hooks are automatically installed when you run `make dev-setup` or `make setup-hooks`. These hooks run before each commit to ensure code quality.

### What the hooks check:

1. **Go Formatting** - `gofmt -s` formatting
2. **Import Organization** - `goimports` for proper import ordering
3. **Go Vet** - Standard Go static analysis
4. **Build Check** - Ensures code compiles
5. **Tests** - Runs all tests
6. **Linting** - Comprehensive linting with `golangci-lint`
7. **Module Tidying** - `go mod tidy` to clean dependencies
8. **Common Issues** - Checks for TODOs, debug prints, etc.
9. **Zero-allocation Compliance** - Alerts on potential allocations in core files

### Using pre-commit hooks:

```bash
# Hooks run automatically on commit
git commit -m "fix: some changes"

# Skip hooks if needed (not recommended)
git commit --no-verify -m "urgent fix"

# Run hooks manually
.git/hooks/pre-commit

# Or use the Makefile target
make pre-commit
```

## Code Quality Tools

### Formatting

```bash
# Format all Go code and organize imports
make format

# Manual formatting
gofmt -s -w .
goimports -w .
```

### Linting

```bash
# Run linter
make lint

# Run linter with auto-fixes
make lint-fix

# Run specific checks
make vet
```

### Testing

```bash
# Run all tests
make test

# Run with race detection
make test-race

# Generate coverage report
make test-cover
```

## Avoiding Common Issues

### Printf Format Issues

The project has had issues with printf formatting directive false positives. To avoid them:

```go
// ❌ Avoid printf format specifiers in documentation strings
fmt.Println("log.Printf(\"User %s logged in\", username)")

// ✅ Use placeholders or escape them
fmt.Println("log.Printf(\"User {name} logged in\", username)")
fmt.Println(`log.Printf("User %s logged in", username)`) // Raw strings work sometimes
```

### Unused Parameters

Use underscore prefix for intentionally unused parameters:

```go
// ❌ Will trigger linting error
func handler(w http.ResponseWriter, r *http.Request) {
    // Only using w, not r
}

// ✅ Mark unused parameter
func handler(w http.ResponseWriter, _r *http.Request) {
    // Using only w
}
```

### Zero-allocation Compliance

Bolt's core files (`bolt.go`, `event.go`) should maintain zero allocations:

```go
// ❌ Avoid in hot paths
data := make([]byte, size)
obj := &MyStruct{}

// ✅ Use pools or pre-allocated buffers
data := getFromPool()
obj := objectPool.Get().(*MyStruct)
```

## Configuration Files

### `.golangci.yml`

Comprehensive linting configuration that:
- Excludes false positives in migration code
- Allows necessary complexity in benchmark validation
- Configures appropriate thresholds for the project

### `.pre-commit-config.yaml`

Pre-commit framework configuration (optional, requires `pre-commit` tool):

```bash
# Install pre-commit framework (optional)
pip install pre-commit

# Install hooks
pre-commit install

# Run manually
pre-commit run --all-files
```

## Development Workflow

### Standard Development Flow

1. **Create feature branch**
   ```bash
   git checkout -b feature/new-feature
   ```

2. **Make changes with quality checks**
   ```bash
   # Make your changes
   vim bolt.go
   
   # Run quality checks during development
   make format  # Format code
   make lint    # Check for issues
   make test    # Run tests
   ```

3. **Commit with hooks**
   ```bash
   git add .
   git commit -m "feat: add new feature"
   # Hooks run automatically and will block bad commits
   ```

4. **Final quality check**
   ```bash
   make pre-commit  # Run all pre-commit checks
   ```

### CI/CD Integration

The hooks mirror what runs in CI/CD:

```bash
# Run the same checks as CI
make ci

# Or comprehensive check
make full-check
```

## Troubleshooting

### Hook Issues

If hooks are causing problems:

```bash
# Temporarily skip hooks
git commit --no-verify -m "urgent fix"

# Update hooks
make setup-hooks

# Debug hook execution
bash -x .git/hooks/pre-commit
```

### Tool Installation

If tools are missing:

```bash
# Check what's installed
make check-tools

# Install missing tools
make install-tools

# Check versions
make version
```

### Performance Issues

For zero-allocation validation:

```bash
# Run zero-allocation benchmarks
make benchmark-zero

# Check for allocations
go test -bench=BenchmarkZeroAllocation -benchmem -memprofile=mem.prof ./...
go tool pprof mem.prof
```

## Contributing

Before submitting a PR:

1. Run `make full-check` to ensure all quality checks pass
2. Add tests for new functionality
3. Update documentation as needed
4. Ensure zero-allocation compliance in core paths
5. Follow conventional commit messages

## Tool Versions

- **Go**: 1.21+ required
- **golangci-lint**: Latest version recommended
- **goimports**: Latest version from golang.org/x/tools/cmd/goimports

## Getting Help

- Run `make help` to see all available commands
- Check CI logs for detailed error messages
- Review `.golangci.yml` for linting configuration details
- Use `git commit --no-verify` only for urgent fixes (fix issues afterward)