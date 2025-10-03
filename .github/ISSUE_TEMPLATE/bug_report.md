---
name: Bug Report
about: Report a bug or unexpected behavior in Bolt
title: '[BUG] '
labels: bug
assignees: ''
---

## Bug Description

A clear and concise description of what the bug is.

## Environment

- **Bolt version**: [e.g., v1.3.0]
- **Go version**: [e.g., 1.23.4]
- **OS/Architecture**: [e.g., Linux amd64, macOS arm64]
- **Framework (if applicable)**: [e.g., Gin v1.10.0, Echo v4.12.0]

## Minimal Reproduction

Please provide a minimal code example that reproduces the issue:

```go
package main

import (
    "github.com/felixgeelhaar/bolt"
)

func main() {
    // Minimal code to reproduce the bug
}
```

## Expected Behavior

What did you expect to happen?

## Actual Behavior

What actually happened?

## Stack Trace (if applicable)

```
// Paste stack trace or error output here
```

## Performance Impact (if applicable)

If this is a performance-related bug:

```bash
# Benchmark results showing the issue
go test -bench=BenchmarkRelevant -benchmem
```

## Additional Context

Add any other context about the problem here:

- [ ] This issue occurs with the JSON handler
- [ ] This issue occurs with the Console handler
- [ ] This issue involves OpenTelemetry integration
- [ ] This issue is a memory leak or allocation increase
- [ ] This issue is a race condition (run with `go test -race`)

## Checklist

Before submitting, please verify:

- [ ] I have searched existing issues to avoid duplicates
- [ ] I have provided a minimal reproduction case
- [ ] I have included environment details
- [ ] I have checked this is not a security vulnerability (see [SECURITY.md](../../SECURITY.md))
