---
name: Feature Request
about: Suggest a new feature or enhancement for Bolt
title: '[FEATURE] '
labels: enhancement
assignees: ''
---

## Feature Description

A clear and concise description of the feature you'd like to see.

## Problem Statement

What problem does this feature solve? Describe the use case and why existing functionality is insufficient.

## Proposed Solution

How would you like this feature to work? Provide as much detail as possible.

```go
// Example API or usage pattern (if applicable)
logger.NewFeature().
    SomeField("value").
    Msg("example usage")
```

## Alternatives Considered

What alternative solutions or workarounds have you considered?

## Performance Impact

What is the expected performance impact of this feature?

- [ ] This feature should maintain zero allocations in hot paths
- [ ] This feature may introduce allocations (please justify)
- [ ] This feature is not performance-critical
- [ ] I have benchmark data to support this feature

## Use Case Priority

How critical is this feature for your use case?

- [ ] **Critical** - Blocking adoption of Bolt
- [ ] **Important** - Significant improvement to my workflow
- [ ] **Nice to have** - Would improve experience but has workarounds

## Implementation Considerations

Are there any specific implementation details or constraints to consider?

- [ ] This feature should be opt-in (not enabled by default)
- [ ] This feature requires new dependencies
- [ ] This feature may introduce breaking changes
- [ ] This feature affects the public API

## Additional Context

Add any other context, screenshots, or examples about the feature request here.

## Framework/Platform Relevance

Is this feature specific to a framework or platform?

- [ ] General Bolt feature
- [ ] Framework-specific (Gin, Echo, Fiber, Chi)
- [ ] Cloud platform-specific (AWS, GCP, Azure)
- [ ] OpenTelemetry integration

## Checklist

Before submitting, please verify:

- [ ] I have searched existing issues and discussions
- [ ] I have read the [CONTRIBUTING.md](../../CONTRIBUTING.md) guidelines
- [ ] I have considered the performance implications
- [ ] I have provided a clear use case and motivation
- [ ] I understand this may require discussion before implementation
