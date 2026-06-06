# Bolt documentation

This directory follows the [Diataxis](https://diataxis.fr/) framework.
Pages live in one of four quadrants based on what the reader is trying
to do:

| Quadrant | When to read | When to write |
|---|---|---|
| **[Tutorial](./tutorial/)** | I am new and want a guided first build. | "Let me walk you through building a thing." |
| **[How-to](./how-to/)** | I have a task and need a recipe. | "How do I migrate from zerolog?" / "How do I redact PII?" |
| **[Reference](./reference/)** | I know what I'm doing and need lookup. | API surface, field types, env vars, levels. |
| **[Explanation](./explanation/)** | I want to understand why bolt is the way it is. | Design rationale: zero-alloc, OTel correlation, Hook v2. |

If you're contributing docs, pick one quadrant per page. A page that
mixes "let me teach you" and "look up this method" usually serves
neither audience well — split it.

## Living index

### Tutorial
- [Five-minute first log](./tutorial/quickstart.md)

### How-to
- [Migrate from `log/slog`](./how-to/migrate-from-slog.md)
- [Migrate from zerolog](./how-to/migrate-from-zerolog.md)
- [Migrate from zap](./how-to/migrate-from-zap.md)

### Reference
- [API reference (pkg.go.dev)](https://pkg.go.dev/go.klarlabs.de/bolt) — generated GoDoc; canonical
- [Field types](./reference/field-types.md)
- [Levels and configuration](./reference/levels-and-config.md)

### Explanation
- [Why zero allocations](./explanation/zero-alloc-rationale.md)
- [Hook v2 — field-aware interception design](./explanation/hook-v2.md)

## What is intentionally NOT here

- **Marketing copy.** Belongs in the top-level [README](../README.md).
- **Roadmap.** Belongs in [ROADMAP.md](../ROADMAP.md).
- **Changelog.** Belongs in [CHANGELOG.md](../CHANGELOG.md).
- **Examples that are full Go modules.** Those live under
  [examples/](../examples/) so they can be `go run`'d.

If a doc is none of those four quadrants, it probably doesn't belong
in `docs/` at all.
