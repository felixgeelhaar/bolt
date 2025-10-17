# Bolt Benchmarks

This directory contains performance benchmarks comparing Bolt against other popular Go logging libraries.

## Why a Separate Module?

The benchmarks are maintained as a separate Go module to avoid adding unnecessary dependencies (zerolog, zap) to the main Bolt library. This ensures that users of Bolt only download the dependencies they actually need.

## Compared Libraries

- **Bolt** - This library
- **Zerolog** - Popular zero-allocation logging library
- **Zap** - Uber's structured logging library
- **slog** - Go's standard library structured logger (Go 1.21+)

## Running Benchmarks

From this directory, run:

```bash
# Run all benchmarks
go test -bench=. -benchmem

# Run specific library benchmarks
go test -bench=BenchmarkBolt -benchmem
go test -bench=BenchmarkZerolog -benchmem
go test -bench=BenchmarkZap -benchmem
go test -bench=BenchmarkSlog -benchmem

# Run with different field counts
go test -bench=Bolt5Fields -benchmem
go test -bench=Zerolog5Fields -benchmem
go test -bench=Zap5Fields -benchmem
go test -bench=Slog5Fields -benchmem

# Test disabled logger performance
go test -bench=Disabled -benchmem
```

## Benchmark Scenarios

Each library is tested with three scenarios:

1. **Standard** - Single log entry with 2 fields (string + int)
2. **5 Fields** - Single log entry with 5 string fields
3. **Disabled** - Logger level set to disable the log level being tested (tests conditional evaluation overhead)

## Understanding Results

The benchmarks measure:
- **ns/op** - Nanoseconds per operation (lower is better)
- **B/op** - Bytes allocated per operation (lower is better)
- **allocs/op** - Number of allocations per operation (zero is ideal)

Example output:
```
BenchmarkBolt-12              11540166        105.2 ns/op       0 B/op       0 allocs/op
BenchmarkZerolog-12            7268754        172.4 ns/op       0 B/op       0 allocs/op
BenchmarkZap-12                2304831        521.6 ns/op       0 B/op       0 allocs/op
```

## Module Structure

This module:
- Uses `replace` directive to reference the parent Bolt module locally
- Maintains its own `go.mod` with benchmark-specific dependencies
- Can be safely ignored by Bolt users who don't need to run comparisons
