# Security Policy

## Supported Versions

| Version | Supported          |
| ------- | ------------------ |
| 1.2.x   | :white_check_mark: |
| < 1.2   | :x:                |

## Reporting a Vulnerability

If you discover a security vulnerability in Bolt, please report it by emailing security@bolt.dev. 

Please include:
- Description of the vulnerability
- Steps to reproduce
- Potential impact
- Suggested fix (if available)

We will acknowledge receipt within 48 hours and provide a detailed response within 7 days.

## Security Scanning Scope

Our security scanning focuses on **production code only**. The following directories are excluded from security scanning as they contain non-production code:

### Excluded from Security Scanning

- `/examples/` - Demonstration code showing usage patterns
- `/benchmark/` - Performance testing code
- `/migrate/` - Migration tools for other logging libraries
- `/infrastructure/` - Terraform and Kubernetes configurations
- `/test-*/` - Test directories

### Why These Exclusions?

1. **Examples** - Use simplified patterns for clarity, including non-cryptographic random numbers
2. **Benchmarks** - Require deterministic random for reproducible performance tests
3. **Migration Tools** - Need file system access to read and transform user code
4. **Infrastructure** - Managed separately with infrastructure-specific security tools

## Production Code Security

The core Bolt library (in the root directory) maintains:
- **Zero security vulnerabilities** in production code
- **Zero allocations** for performance
- **Type safety** throughout the API
- **Secure defaults** for all configurations

## Security Features

- **No external dependencies** in core library
- **Read-only operations** by default
- **Structured logging** prevents injection attacks
- **Type-safe field methods** prevent type confusion
- **Atomic operations** for thread safety

## Regular Security Audits

We run automated security scanning on every commit using:
- `gosec` for Go security analysis
- GitHub Security Advisories
- Dependency vulnerability scanning

Production code maintains a clean security report with zero vulnerabilities.