# Pull Request

## 📋 Description

Provide a clear and concise description of what this PR does.

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

For performance-critical changes, include benchmark results:

```
BenchmarkBefore-8    1000000    123 ns/op    64 B/op    2 allocs/op
BenchmarkAfter-8     2000000     87 ns/op     0 B/op    0 allocs/op
```

- [ ] No performance impact
- [ ] Performance improvement (include benchmarks above)
- [ ] Potential performance regression (justified why)
- [ ] Performance impact unknown (needs benchmarking)

## 🧪 Testing

- [ ] I have added tests that prove my fix is effective or that my feature works
- [ ] I have added benchmarks for performance-critical changes
- [ ] New and existing unit tests pass locally with my changes
- [ ] I have tested with race detection (`go test -race`)

## 📖 Documentation

- [ ] I have updated relevant documentation
- [ ] I have added/updated code comments for exported functions
- [ ] I have updated the README if needed
- [ ] I have added examples for new features

## ✅ Checklist

- [ ] My code follows the project's style guidelines
- [ ] I have performed a self-review of my own code
- [ ] I have commented my code, particularly in hard-to-understand areas
- [ ] My changes maintain zero allocations for core logging paths
- [ ] I have made corresponding changes to the documentation
- [ ] My changes generate no new warnings
- [ ] Any dependent changes have been merged and published

## 🔗 Related Issues

Link any related issues here:
- Closes #(issue)
- Related to #(issue)

## 📝 Additional Notes

Add any other context about the pull request here, such as:
- Implementation decisions made
- Alternative approaches considered
- Areas that need special attention during review
- Migration guide for breaking changes

## 📊 Backwards Compatibility

- [ ] This change is backwards compatible
- [ ] This change breaks backwards compatibility (breaking change justification required)

If this is a breaking change, please describe:
1. What breaks
2. How users can migrate
3. Why this breaking change is necessary

---

By submitting this pull request, I confirm that my contribution is made under the terms of the MIT license.