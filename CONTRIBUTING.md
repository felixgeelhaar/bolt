# Contributing to Bolt

Thank you for your interest in contributing to Bolt! This document provides guidelines and instructions for contributing.

## 🎯 Quick Start

1. **Fork** the repository on GitHub
2. **Clone** your fork locally:
   ```bash
   git clone https://github.com/YOUR_USERNAME/bolt.git
   cd bolt
   ```
3. **Create a branch** for your feature:
   ```bash
   git checkout -b feature/your-feature-name
   ```
4. **Make changes** and test thoroughly
5. **Submit a pull request** from your fork to the main repository

## 🏗️ Development Setup

### Prerequisites

- Go 1.19 or later
- Git
- (Optional) golangci-lint for code quality checks

### Setting Up Your Environment

```bash
# Clone your fork
git clone https://github.com/YOUR_USERNAME/bolt.git
cd bolt

# Install dependencies
go mod download

# Run tests to verify setup
go test ./...

# Run benchmarks
go test -bench=. -benchmem
```

## 📝 Contribution Guidelines

### Code of Conduct

Please read and follow our [Code of Conduct](CODE_OF_CONDUCT.md). We are committed to providing a welcoming and inclusive environment for all contributors.

### What We're Looking For

We welcome contributions in the following areas:

✅ **Encouraged:**
- Bug fixes with test coverage
- Performance improvements with benchmarks
- Documentation improvements
- New examples and use cases
- Integration guides for popular frameworks
- Security enhancements
- Test coverage improvements

⚠️ **Discuss First:**
- Major architectural changes
- New features (open an issue first)
- Breaking API changes
- New dependencies

❌ **Not Accepted:**
- Changes that increase allocations in hot paths
- Features that compromise performance
- Dependencies without clear justification
- Changes without tests

### Before You Start

1. **Check existing issues** - Someone may already be working on it
2. **Open an issue** for major changes to discuss approach
3. **Review recent PRs** to understand our standards
4. **Read the architecture docs** to understand design decisions

## 🧪 Testing Requirements

### Test Coverage

All contributions must include appropriate tests:

```bash
# Run all tests
go test ./...

# Run tests with coverage
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out

# Run tests with race detector
go test -race ./...
```

**Coverage Requirements:**
- New code: Minimum 90% coverage
- Bug fixes: Include reproduction test
- Performance changes: Include benchmarks

### Benchmark Requirements

Performance-related changes must include benchmarks:

```bash
# Run benchmarks
go test -bench=. -benchmem -count=5

# Compare before/after
go test -bench=BenchmarkYourChange -benchmem -count=10 > new.txt
git checkout main
go test -bench=BenchmarkYourChange -benchmem -count=10 > old.txt
benchstat old.txt new.txt
```

**Performance Standards:**
- No new allocations in hot paths (JSON handler)
- Performance improvements should be measurable (>5%)
- Benchmark variance should be <5%

### Fuzzing (for security-critical code)

```bash
# Run fuzzing tests
go test -fuzz=FuzzJSONHandler -fuzztime=30s
go test -fuzz=FuzzInputValidation -fuzztime=30s
```

## 💻 Code Quality Standards

### Code Style

- Follow standard Go conventions (gofmt, goimports)
- Use meaningful variable and function names
- Keep functions small and focused (<50 lines)
- Comment exported functions and types
- Use godoc-style comments

```go
// Good: Clear, concise, descriptive
func (e *Event) Str(key, value string) *Event {
    // Add string field to event
    ...
}

// Bad: Vague, unclear purpose
func (e *Event) AddThing(k, v string) *Event {
    ...
}
```

### Performance Guidelines

**Zero-Allocation Hot Paths:**
```go
// ✅ Good: No allocations
func (e *Event) Int(key string, val int) *Event {
    e.buf = append(e.buf, `,"`...)
    e.buf = append(e.buf, key...)
    e.buf = append(e.buf, `":`...)
    e.buf = appendInt(e.buf, int64(val))
    return e
}

// ❌ Bad: Allocates string
func (e *Event) Int(key string, val int) *Event {
    e.fields[key] = fmt.Sprintf("%d", val)  // Allocation!
    return e
}
```

**Use Benchmarks to Verify:**
```bash
# Verify zero allocations
go test -bench=BenchmarkYourFunction -benchmem
# Should show: 0 B/op    0 allocs/op
```

### Error Handling

```go
// ✅ Good: Explicit error handling
func (h *JSONHandler) Handle(e *Event) error {
    buf, err := json.Marshal(e)
    if err != nil {
        return fmt.Errorf("marshal failed: %w", err)
    }
    _, err = h.w.Write(buf)
    return err
}

// ❌ Bad: Ignoring errors
func (h *JSONHandler) Handle(e *Event) error {
    buf, _ := json.Marshal(e)
    h.w.Write(buf)
    return nil
}
```

### Documentation Standards

All exported functions must have documentation:

```go
// Str adds a string field to the log event.
// The key should be a valid JSON key (no special characters).
// The value will be automatically JSON-escaped.
//
// Example:
//
//	logger.Info().Str("user", "alice").Msg("user logged in")
//
// Output:
//
//	{"level":"info","user":"alice","message":"user logged in"}
func (e *Event) Str(key, value string) *Event {
    ...
}
```

## 🔍 Code Review Process

### Submitting a Pull Request

1. **Create a descriptive PR title:**
   ```
   feat: add support for custom timestamp formats
   fix: resolve race condition in event pool
   docs: improve OpenTelemetry integration guide
   perf: optimize integer serialization
   ```

2. **Write a clear description:**
   - What problem does this solve?
   - How does it solve it?
   - Any breaking changes?
   - Performance impact?

3. **Include test results:**
   ```
   Benchmark results:
   BenchmarkNewFeature-8   5000000   234 ns/op   0 B/op   0 allocs/op

   Test coverage:
   coverage: 95.2% of statements
   ```

4. **Link related issues:**
   ```
   Fixes #123
   Related to #456
   ```

### Review Checklist

Before requesting review, ensure:

- [ ] Code follows Go conventions (gofmt, goimports)
- [ ] All tests pass (`go test ./...`)
- [ ] No race conditions (`go test -race ./...`)
- [ ] Benchmarks show no performance regression
- [ ] Documentation is updated
- [ ] Examples are provided (if applicable)
- [ ] CHANGELOG.md is updated (for non-trivial changes)
- [ ] No new dependencies without justification

### Review Process

1. **Automated checks** run on every PR:
   - Tests (multiple Go versions)
   - Benchmarks
   - Code coverage
   - Linting (golangci-lint)
   - Security scanning (gosec)

2. **Maintainer review:**
   - Code quality
   - Performance impact
   - API design
   - Documentation completeness

3. **Feedback and iteration:**
   - Address review comments
   - Update PR based on feedback
   - Request re-review when ready

4. **Merge:**
   - Squash commits for clean history
   - Maintainer merges when approved

## 🎨 Commit Message Guidelines

Follow conventional commit format:

```
<type>(<scope>): <subject>

<body>

<footer>
```

**Types:**
- `feat`: New feature
- `fix`: Bug fix
- `perf`: Performance improvement
- `docs`: Documentation changes
- `test`: Test additions/changes
- `refactor`: Code refactoring
- `chore`: Build process or auxiliary tool changes

**Examples:**
```
feat(handler): add support for custom timestamp formats

Adds ability to configure timestamp format in handlers.
Includes tests and documentation.

Closes #123

---

fix(event): resolve race condition in event pool

The event pool had a race condition when accessing the
buffer under high concurrency. This adds proper synchronization.

Fixes #456

---

perf(serialize): optimize integer-to-string conversion

Replaces fmt.Sprintf with custom conversion for 2x speedup.

Benchmark comparison:
BenchmarkInt-8    10000000    150 ns/op (was 320 ns/op)
```

## 🐛 Reporting Bugs

### Security Vulnerabilities

**DO NOT** open public issues for security vulnerabilities. Instead, follow our [Security Policy](SECURITY.md) for responsible disclosure.

### Bug Reports

Open a GitHub issue with:

1. **Clear title** describing the problem
2. **Environment details:**
   - Go version
   - OS/Architecture
   - Bolt version
3. **Minimal reproduction:**
   ```go
   package main

   import "github.com/felixgeelhaar/bolt"

   func main() {
       // Minimal code to reproduce
   }
   ```
4. **Expected vs actual behavior**
5. **Stack trace** (if applicable)

## 💡 Feature Requests

Before requesting a feature:

1. **Check existing issues** - May already be planned
2. **Consider impact** - Does it align with project goals?
3. **Provide use case** - Why is this needed?

Open a GitHub issue with:

- **Problem description** - What problem does this solve?
- **Proposed solution** - How would it work?
- **Alternatives considered** - Other approaches?
- **Performance impact** - Any concerns?

## 📊 Performance Contributions

Performance improvements are highly valued! When contributing performance changes:

### Benchmark Requirements

```bash
# Run baseline benchmarks
git checkout main
go test -bench=BenchmarkTargetFunction -benchmem -count=10 > old.txt

# Apply your changes
git checkout your-branch
go test -bench=BenchmarkTargetFunction -benchmem -count=10 > new.txt

# Compare results
benchstat old.txt new.txt
```

### Include in PR Description

```markdown
## Performance Impact

**Before:**
BenchmarkTargetFunction-8   5000000   320 ns/op   48 B/op   2 allocs/op

**After:**
BenchmarkTargetFunction-8   10000000  150 ns/op   0 B/op    0 allocs/op

**Improvement:** 2.1x faster, eliminated allocations
```

### Verification Checklist

- [ ] Benchmarks show measurable improvement (>5%)
- [ ] No regression in other benchmarks
- [ ] Zero-allocation guarantee maintained (if applicable)
- [ ] Performance improvement is documented

## 📚 Documentation Contributions

Documentation improvements are always welcome:

- **API documentation** - Godoc comments
- **Guides** - Integration examples, best practices
- **Examples** - Real-world use cases
- **Troubleshooting** - Common issues and solutions

### Documentation Style

- Use clear, concise language
- Include code examples
- Provide context and explanations
- Link to related documentation

```markdown
## Using Bolt with Gin

Bolt integrates seamlessly with Gin through middleware:

\`\`\`go
func BoltLogger(logger *bolt.Logger) gin.HandlerFunc {
    return func(c *gin.Context) {
        // Log request details
        logger.Info().
            Str("method", c.Request.Method).
            Str("path", c.Request.URL.Path).
            Msg("request started")

        c.Next()
    }
}
\`\`\`

See [full example](examples/gin) for production setup.
```

## 🏆 Recognition

Contributors are recognized in:

- `CONTRIBUTORS.md` file
- GitHub contributors page
- Release notes for significant contributions

## 📞 Getting Help

- **Questions:** Open a GitHub discussion
- **Bugs:** Open a GitHub issue
- **Security:** Follow [Security Policy](SECURITY.md)
- **Chat:** Join our community discussions

## 📄 License

By contributing to Bolt, you agree that your contributions will be licensed under the MIT License.

---

**Thank you for contributing to Bolt!** 🎉

Your contributions help make high-performance logging accessible to the Go community.
