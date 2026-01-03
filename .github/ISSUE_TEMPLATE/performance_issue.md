---
name: Performance Issue
about: Report a performance regression or optimization opportunity
title: '[PERF] '
labels: performance
assignees: ''
---

## Performance Issue Description

A clear description of the performance problem or regression.

## Environment

- **Bolt version**: [e.g., v1.3.0]
- **Go version**: [e.g., 1.23.4]
- **OS/Architecture**: [e.g., Linux amd64]
- **CPU**: [e.g., Intel Xeon, Apple M2]
- **Handler type**: [JSON / Console]

## Benchmark Results

### Current Performance

```bash
# Run this command and paste results:
go test -bench=BenchmarkRelevant -benchmem -count=10

BenchmarkRelevant-8   X ops   Y ns/op   Z B/op   W allocs/op
```

### Expected Performance

What performance did you expect based on documentation or previous versions?

```
BenchmarkRelevant-8   X ops   Y ns/op   Z B/op   W allocs/op
```

## Performance Regression (if applicable)

If this is a regression from a previous version:

```bash
# Baseline (previous version)
git checkout v1.2.0
go test -bench=BenchmarkRelevant -benchmem -count=10 > old.txt

# Current version
git checkout main
go test -bench=BenchmarkRelevant -benchmem -count=10 > new.txt

# Compare
benchstat old.txt new.txt
```

Paste benchstat output here:

```
// benchstat results
```

## Profiling Data

### CPU Profile

```bash
go test -bench=BenchmarkRelevant -cpuprofile=cpu.prof
go tool pprof -top cpu.prof
```

```
// Top CPU consumers
```

### Memory Profile

```bash
go test -bench=BenchmarkRelevant -memprofile=mem.prof
go tool pprof -top mem.prof
```

```
// Top memory allocators
```

## Reproduction Code

Minimal code that demonstrates the performance issue:

```go
package main

import (
    "github.com/felixgeelhaar/bolt/v3"
    "testing"
)

func BenchmarkIssue(b *testing.B) {
    logger := bolt.New(bolt.NewJSONHandler(io.Discard))

    b.ResetTimer()
    b.ReportAllocs()

    for i := 0; i < b.N; i++ {
        // Code demonstrating the issue
    }
}
```

## Analysis

What is causing the performance issue? (If known)

- [ ] Unexpected allocations in hot path
- [ ] CPU-intensive operation
- [ ] Lock contention
- [ ] Inefficient algorithm
- [ ] Other: ___________

## Potential Fix (Optional)

If you have ideas for how to fix this, please share:

```go
// Proposed optimization
```

## Performance Goals

What performance characteristics should this operation have?

- **Target latency**: ___ ns/op
- **Target allocations**: 0 allocs/op (or justify why allocations are needed)
- **Target throughput**: ___ ops/sec

## Additional Context

- [ ] This affects JSON handler only
- [ ] This affects Console handler only
- [ ] This affects both handlers
- [ ] This is specific to certain field types
- [ ] This occurs under high concurrency
- [ ] This involves OpenTelemetry integration

## Checklist

Before submitting:

- [ ] I have provided benchmark results with `-benchmem`
- [ ] I have run benchmarks multiple times (`-count=10`) for reliability
- [ ] I have included profiling data when possible
- [ ] I have verified this with the latest version
- [ ] I have checked existing performance issues
