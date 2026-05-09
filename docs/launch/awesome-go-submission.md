# Awesome-Go submission text

Draft entry for the
[avelino/awesome-go](https://github.com/avelino/awesome-go) Logging
section. Submitted via PR by the maintainer; this file is the
prepared copy so it's checked in alongside the project rather than
lost in a private notes app.

## Listing entry

Insert under `## Logging` in alphabetical order:

```markdown
* [bolt](https://github.com/felixgeelhaar/bolt) - Zero-allocation `slog.Handler` for Go with first-class OpenTelemetry trace/span injection. Passes `testing/slogtest.TestHandler` conformance.
```

## PR title

```
Add bolt: zero-allocation slog.Handler with first-class OpenTelemetry
```

## PR body

Copy as the awesome-go PR description:

```markdown
## What

Adding [bolt](https://github.com/felixgeelhaar/bolt) under Logging.

## Description

> Zero-allocation `slog.Handler` for Go with first-class OpenTelemetry trace/span injection. Passes `testing/slogtest.TestHandler` conformance.

## Why it belongs in awesome-go

* Fills a gap between `log/slog` (stdlib, no perf budget) and `zerolog` / `zap` (older chained or typed-constructor APIs without first-class slog conformance).
* Standard `testing/slogtest.TestHandler` conformance — slog group nesting, `WithAttrs` scoping, empty-group elision, `LogValuer` resolution all verified.
* MIT licensed, Go 1.24+, single maintainer, response-time SLA documented in [SECURITY.md](https://github.com/felixgeelhaar/bolt/blob/main/SECURITY.md).
* Supply-chain hardening: cosign-signed artefacts, SPDX SBOM, SLSA-3 provenance attestation per release.
* Migration guides from zerolog, zap, slog: [docs/how-to/](https://github.com/felixgeelhaar/bolt/tree/main/docs/how-to).

## Quality checklist

* [x] Repository has a clear README with a hero pitch in the first 30 lines
* [x] License is OSI-approved (MIT)
* [x] No third-party dependencies in the core library other than OpenTelemetry
* [x] CI badge green at submission time
* [x] At least one tagged release (v1.3.0+)
* [x] No vendored dependencies in the import path
```

## Process notes (for the maintainer doing the submission)

- Awesome-Go's contribution guidelines require alphabetical order and
  a single sentence description; the entry above honours both.
- The maintainer must run their PR template's lint script locally
  before opening — see
  [contributing.md](https://github.com/avelino/awesome-go/blob/main/CONTRIBUTING.md).
- Approval timeline historically: 1–4 weeks. Don't rebase
  aggressively; reviewers sometimes batch checks.
