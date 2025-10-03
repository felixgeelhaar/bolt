# Pull Request

## 📋 Description

<!-- Provide a clear and concise description of what this PR does -->

Fixes #(issue number)

## 🔄 Type of Change

Please check the relevant option:

- [ ] 🐛 Bug fix (non-breaking change that fixes an issue)
- [ ] ✨ New feature (non-breaking change that adds functionality)
- [ ] 💥 Breaking change (fix or feature that would cause existing functionality to break)
- [ ] 📚 Documentation update
- [ ] 🔧 Refactoring (no functional changes)
- [ ] ⚡ Performance improvement
- [ ] 🧪 Test improvements

## 🚀 Performance Impact

<!-- For performance-critical changes, include benchmark results -->

### Benchmark Results

**Before:**
```
BenchmarkRelevant-8   1000000    123 ns/op    64 B/op    2 allocs/op
```

**After:**
```
BenchmarkRelevant-8   2000000     87 ns/op     0 B/op    0 allocs/op
```

**Analysis:**
<!-- Run benchstat to compare: benchstat old.txt new.txt -->

### Allocation Impact

- [ ] No performance impact
- [ ] Performance improvement (include benchmarks above)
- [ ] No new allocations in hot paths (JSON handler)
- [ ] New allocations are justified and documented
- [ ] Potential performance regression (justified why)

## 🧪 Testing

### Test Coverage

- [ ] I have added tests that prove my fix is effective or that my feature works
- [ ] I have added benchmarks for performance-critical changes
- [ ] New and existing unit tests pass locally with my changes
- [ ] I have tested with race detection (`go test -race`)
- [ ] I have run fuzzing tests (if applicable)

### Test Results

```bash
# Paste test results
go test -v ./...

# Coverage
go test -cover ./...

# Race detection
go test -race ./...
```

## 📖 Documentation

- [ ] I have updated relevant documentation
- [ ] I have added/updated godoc comments for exported functions
- [ ] I have updated the README if needed
- [ ] I have added examples for new features
- [ ] I have updated CHANGELOG.md

## ✅ Code Quality Checklist

### Code Standards

- [ ] My code follows the project's style guidelines (gofmt, goimports)
- [ ] I have performed a self-review of my own code
- [ ] I have commented my code, particularly in hard-to-understand areas
- [ ] My changes maintain zero allocations for core logging paths
- [ ] My changes generate no new warnings
- [ ] golangci-lint passes without errors
- [ ] Functions are small and focused (<50 lines where possible)

### Performance Standards (if applicable)

- [ ] Zero allocations maintained in hot paths
- [ ] Benchmarks show no performance regression
- [ ] Performance improvements are measurable (>5%)
- [ ] Benchmark variance is acceptable (<5%)

## 🔗 Related Issues

<!-- Link any related issues here -->
- Closes #(issue)
- Related to #(issue)

## 📝 Changes Made

<!-- Provide a bullet-point list of changes -->

-
-
-

## 📊 Breaking Changes

<!-- If this is a breaking change, describe the impact and migration path -->

- [ ] This change is backwards compatible
- [ ] This change breaks backwards compatibility (breaking change justification required)

### Migration Guide (for breaking changes)

**Before:**
```go
// Old API usage
```

**After:**
```go
// New API usage
```

**Why this breaking change is necessary:**

## 💭 Implementation Notes

<!-- Add any other context about the pull request here -->

### Implementation Decisions
<!-- Key decisions made during implementation -->

### Alternative Approaches Considered
<!-- Other approaches considered and why this approach was chosen -->

### Areas Needing Special Review Attention
<!-- Specific areas reviewers should focus on -->

## 🚀 Deployment Notes

<!-- Any special considerations for deploying this change -->

- [ ] No deployment concerns
- [ ] Requires configuration changes
- [ ] Requires dependency updates
- [ ] Other: ___________

---

**By submitting this pull request, I confirm that my contribution is made under the terms of the MIT license.**