# Changelog

All notable changes to Bolt will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [1.2.2] - 2025-10-03

### 🔧 Fixed

#### Critical Race Condition in SetLevel
- **Fixed race condition in `SetLevel()`** where validation occurred after atomic load, creating a corruption window
- Added defensive level clamping to INFO for invalid values (previously could cause undefined behavior)
- Enhanced thread safety with proper atomic validation order
- Added comprehensive race detection tests with 100 concurrent goroutines
- **Impact**: Prevents potential security bypass and inconsistent filtering behavior in multi-threaded environments

#### ConsoleHandler Performance Enhancement
- **Rewrote ConsoleHandler with streaming JSON parser** to reduce allocations
- Replaced heavy `json.Unmarshal` with zero-allocation field extraction using `extractJSONField()`
- Improved from heavy unmarshaling to lighter parsing (~10 allocs/op at 477ns/op)
- Added comprehensive benchmarks for ConsoleHandler allocation tracking
- **Note**: ~10 allocations remain due to string→bytes conversions; full zero-allocation requires architectural changes planned for v2.0

### 📚 Documented

#### Float64 Precision Limitation
- **Added comprehensive Float64 documentation** explaining 6-decimal precision limitation
- Documented special value handling: NaN, +Inf, -Inf (negative zero not preserved)
- Provided clear guidance on Float64 (zero-alloc, 6 decimals) vs Any (allocates, full precision) tradeoff
- Added precision examples showing rounding behavior (99.99 → "99.989999")
- Created `float64_test.go` with comprehensive test coverage:
  - Precision validation tests for various float values
  - Float64 vs Any comparison tests
  - Scientific notation formatting tests
  - Performance benchmarks showing 3x speedup (60ns vs 180ns)

### ✅ Added

#### Comprehensive Default Logger Tests
- **Achieved 100% coverage on package-level API** (previously 0%)
- Added tests for all logging functions: Info(), Error(), Debug(), Warn(), Trace(), Fatal()
- Environment variable configuration tests (BOLT_LEVEL, BOLT_FORMAT)
- Concurrent access safety validation with ThreadSafeBuffer
- Context-based logging tests with persistent fields
- Dynamic level change tests
- All field types validation
- Error handling and panic behavior tests
- Zero-allocation benchmarks for default logger (82.6ns/op, 0 allocs)
- Concurrent benchmark with RunParallel (163.2ns/op, 0 allocs)

### 📈 Performance

- **Simple Log**: 70ns/op (improved from 77ns/op in v1.2.1) ⚡
- **Float64**: 66ns/op (maintained) with 6-decimal precision
- **Complex Event**: 258ns/op (improved from 259ns/op)
- **Default Logger**: 82.6ns/op with zero allocations
- **Float64 vs Any**: 3x faster (60ns vs 180ns) with precision tradeoff

### 🔒 Security

- **Enhanced thread safety** with proper atomic operation ordering in SetLevel
- **Defensive programming** with invalid level clamping prevents undefined behavior
- **Race detector clean** across all concurrent operations
- Comprehensive race detection test coverage added

### 📝 Documentation Updates

- Updated README.md with v1.2.2 performance numbers
- Updated performance badge: 70ns/op | 0 allocs
- Added ConsoleHandler allocation note with v2.0 roadmap reference
- Updated benchmark sample results with current measurements
- Documented Float64 precision limitation in code and README

---

## [1.2.1] - 2025-07-13

### Added
- Custom Float64 formatter with 6-decimal precision for zero allocations
- Enhanced OpenTelemetry integration
- Security improvements with input validation

### Fixed
- Performance optimizations for common logging paths
- Buffer management improvements

---

## [1.2.0] - 2025-07-01

### Added
- ConsoleHandler with colorized output
- OpenTelemetry trace/span ID injection
- Environment-based configuration (BOLT_LEVEL, BOLT_FORMAT)

### Changed
- Improved event pooling efficiency
- Enhanced JSON serialization performance

---

## [1.1.0] - 2025-06-15

### Added
- Structured logging with rich field types
- Context-aware logging
- Custom handlers support

---

## [1.0.0] - 2025-06-01

### Added
- Initial release with zero-allocation logging
- JSON and Console handlers
- Basic field types (Str, Int, Bool, Float64)
- Event pooling for zero allocations

---

## Upgrade Guide

### v1.2.1 → v1.2.2

**Breaking Changes**: None

**Improvements**:
1. **Thread Safety**: SetLevel now properly validates before atomic operations - no code changes needed
2. **Float64 Precision**: Now documented - users requiring full precision should migrate to Any():
   ```go
   // Before (6 decimal precision, zero-alloc)
   logger.Float64("value", 99.99) // → "99.989999"

   // After (full precision, allocates)
   logger.Any("value", 99.99) // → "99.99"
   ```
3. **ConsoleHandler**: Performance improved but still ~10 allocs/op - consider JSONHandler for production
4. **Default Logger**: New comprehensive test coverage ensures reliability

**Migration Steps**:
1. Update dependency: `go get -u github.com/felixgeelhaar/bolt@v1.2.2`
2. Review Float64 usage if precision >6 decimals is required
3. Run tests to ensure thread-safe SetLevel behavior
4. Optional: Add default logger tests to your codebase using our examples

**Recommended Actions**:
- Review logs using Float64 for financial/scientific data - consider Any() if precision matters
- Verify concurrent SetLevel calls work as expected (now race-free)
- Benchmark ConsoleHandler if used in production (JSONHandler recommended for performance)

---

## Future Roadmap

See [ROADMAP.md](ROADMAP.md) for detailed plans:

- **v1.3.0**: Quality release with 95% test coverage, fuzzing, production examples
- **v2.0.0**: Go idiomatic redesign with functional options, proper interfaces
- **v2.1.0**: Enterprise features - sampling, metrics, observability

---

[1.2.2]: https://github.com/felixgeelhaar/bolt/compare/v1.2.1...v1.2.2
[1.2.1]: https://github.com/felixgeelhaar/bolt/compare/v1.2.0...v1.2.1
[1.2.0]: https://github.com/felixgeelhaar/bolt/compare/v1.1.0...v1.2.0
[1.1.0]: https://github.com/felixgeelhaar/bolt/compare/v1.0.0...v1.1.0
[1.0.0]: https://github.com/felixgeelhaar/bolt/releases/tag/v1.0.0
