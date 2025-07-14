# Contributing to Bolt

Thank you for your interest in contributing to Bolt! This document provides guidelines and instructions for contributing to the project.

## 🚀 Getting Started

### Fork and Clone

1. **Fork** the repository on GitHub
2. **Clone** your fork locally:
   ```bash
   git clone https://github.com/YOUR_USERNAME/bolt.git
   cd bolt
   ```
3. **Add** the original repository as upstream:
   ```bash
   git remote add upstream https://github.com/felixgeelhaar/bolt.git
   ```

### Development Setup

1. **Install Go** 1.21 or later
2. **Install dependencies**:
   ```bash
   go mod download
   ```
3. **Run tests**:
   ```bash
   go test ./...
   ```
4. **Run benchmarks** (optional):
   ```bash
   go test -bench=. -benchmem -tags=bench
   ```

## 📋 Development Workflow

### Before You Start

1. **Check existing issues** and pull requests to avoid duplication
2. **Create an issue** to discuss significant changes before implementation
3. **Keep changes focused** - one feature or fix per pull request

### Making Changes

1. **Create a feature branch** from main:
   ```bash
   git checkout -b feature/your-feature-name
   ```

2. **Make your changes** following our coding standards:
   - Follow Go conventions and use `gofmt`
   - Write comprehensive tests for new functionality
   - Maintain zero allocations for core logging paths
   - Add benchmarks for performance-critical code
   - Update documentation as needed

3. **Test your changes**:
   ```bash
   # Run all tests
   go test ./...
   
   # Run with race detection
   go test -race ./...
   
   # Run benchmarks
   go test -bench=. -benchmem -tags=bench
   
   # Check for memory leaks
   go test -v -count=1000 ./...
   ```

4. **Commit your changes** with descriptive messages:
   ```bash
   git commit -m "feat: add new field type for IP addresses"
   git commit -m "fix: resolve buffer overflow in large messages"
   git commit -m "perf: optimize timestamp formatting"
   ```

### Submitting Changes

1. **Update your fork**:
   ```bash
   git fetch upstream
   git rebase upstream/main
   ```

2. **Push to your fork**:
   ```bash
   git push origin feature/your-feature-name
   ```

3. **Create a Pull Request** with:
   - Clear title describing the change
   - Detailed description of what and why
   - Reference to related issues
   - Benchmark results for performance changes

## 🎯 Contribution Guidelines

### Code Quality

- **Zero allocations** must be maintained for core logging operations
- **Performance** is critical - include benchmarks with changes
- **Memory safety** - careful buffer management and bounds checking
- **Thread safety** - all public APIs must be goroutine-safe
- **Backward compatibility** - avoid breaking changes to public APIs

### Code Style

- Follow standard Go conventions
- Use meaningful variable and function names
- Add comments for exported functions and complex logic
- Keep functions focused and reasonably sized
- Use Go modules for dependency management

### Testing

- **Unit tests** for all new functionality
- **Benchmark tests** for performance-critical code
- **Integration tests** for complex workflows
- **Example tests** for public APIs
- Maintain **100% test coverage** for core functionality

### Performance Requirements

All contributions must meet these performance standards:

- **Zero allocations** for disabled log levels
- **Minimal allocations** for enabled logging (prefer object pooling)
- **Competitive performance** with other high-performance loggers
- **Memory efficiency** in buffer management
- **CPU efficiency** in hot paths

## 🐛 Bug Reports

When reporting bugs, please include:

- **Go version** and operating system
- **Minimal reproduction case**
- **Expected vs actual behavior**
- **Stack traces** if applicable
- **Performance impact** if relevant

Use our bug report template when creating issues.

## ✨ Feature Requests

For new features:

- **Check existing issues** to avoid duplication
- **Describe the use case** and motivation
- **Consider performance impact**
- **Propose API design** if applicable
- **Offer to implement** the feature

## 📊 Performance Contributions

Performance improvements are especially welcome:

- **Include benchmarks** showing the improvement
- **Profile before and after** with `go tool pprof`
- **Measure allocations** and CPU usage
- **Test edge cases** and high-load scenarios
- **Document the optimization** technique used

## 🔍 Code Review Process

All contributions go through code review:

1. **Automated checks** must pass (tests, linting, benchmarks)
2. **Maintainer review** for code quality and design
3. **Performance validation** for critical paths
4. **Documentation review** for clarity and completeness
5. **Final approval** and merge

## 📚 Documentation

Help improve our documentation:

- **README improvements** for clarity and examples
- **API documentation** with godoc comments
- **Performance guides** and best practices
- **Migration guides** for major changes
- **Blog posts** about advanced usage

## 🏆 Recognition

Contributors are recognized in:

- **CONTRIBUTORS.md** file
- **Release notes** for significant contributions
- **GitHub repository** insights
- **Social media** acknowledgments

## 💬 Getting Help

- **Create an issue** for bugs and features
- **Start a discussion** for questions and ideas
- **Check existing issues** for similar problems
- **Review documentation** and examples first

## 📜 Code of Conduct

This project adheres to a [Code of Conduct](CODE_OF_CONDUCT.md). By participating, you agree to uphold this code.

## 📄 License

By contributing to Bolt, you agree that your contributions will be licensed under the [MIT License](LICENSE).

---

Thank you for contributing to Bolt! 🚀