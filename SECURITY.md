# Security Policy

## 🔒 Reporting Security Vulnerabilities

**DO NOT** open public GitHub issues for security vulnerabilities.

Instead, please report security issues responsibly through one of these channels:

### Preferred: GitHub Security Advisories

1. Go to the [Security Advisories page](https://github.com/felixgeelhaar/bolt/security/advisories/new)
2. Click "Report a vulnerability"
3. Fill in the details of the vulnerability
4. Submit the report

This is the **preferred method** as it allows for private disclosure and coordinated response.

### Alternative: Private Email

If you cannot use GitHub Security Advisories, email security reports to:

**security@bolt.dev** (for urgent security issues)

Please include:
- Description of the vulnerability
- Steps to reproduce
- Potential impact
- Any suggested fixes (if available)

## 🛡️ Security Update Policy

### Supported Versions

We support security updates for the following versions:

| Version | Supported          |
| ------- | ------------------ |
| 1.3.x   | ✅ Active support  |
| 1.2.x   | ✅ Critical fixes  |
| 1.1.x   | ❌ No longer supported |
| < 1.1   | ❌ No longer supported |

### Security Update Timeline

- **Critical vulnerabilities**: Patched within 48 hours
- **High severity**: Patched within 7 days
- **Medium severity**: Patched in next minor release
- **Low severity**: Patched in next release

## 🔍 Vulnerability Disclosure Process

### Our Process

1. **Acknowledgment**: We acknowledge receipt within 24 hours
2. **Investigation**: We investigate and validate the issue within 3 business days
3. **Fix Development**: We develop and test a fix
4. **Coordinated Disclosure**: We coordinate disclosure with the reporter
5. **Public Disclosure**: We publish a security advisory and release a patch
6. **Recognition**: We credit the reporter (unless they prefer to remain anonymous)

### Your Process (Responsible Disclosure)

We ask that you:

1. **Report privately** using the channels above
2. **Provide details** to help us understand and reproduce the issue
3. **Allow time** for us to develop and release a fix (typically 90 days)
4. **Coordinate disclosure** with us before public announcement
5. **Do not exploit** the vulnerability maliciously

## 🎯 Security Scope

### In Scope

The following are within our security scope:

✅ **Critical Security Issues:**
- Memory safety vulnerabilities (buffer overflows, use-after-free)
- Input validation bypasses leading to crashes or exploits
- Log injection vulnerabilities
- Denial of service vulnerabilities
- Information disclosure through logging

✅ **High Priority:**
- Race conditions leading to security issues
- Resource exhaustion attacks
- Unsafe deserialization
- Command injection through log fields

### Out of Scope

The following are **NOT** considered security vulnerabilities:

❌ **Expected Behavior:**
- Logging sensitive data (user responsibility to sanitize inputs)
- Performance degradation from large log volumes (rate limiting is user's responsibility)
- Logs being written to insecure destinations (configuration responsibility)

❌ **Low Impact:**
- Theoretical vulnerabilities without practical exploit
- Issues in example code or documentation
- Dependencies' vulnerabilities (report to respective projects)

❌ **Security Scanning Exclusions:**

The following directories are excluded from security scanning as they contain non-production code:

- `/examples/` - Demonstration code showing usage patterns
- `/benchmark/` - Performance testing code
- `/migrate/` - Migration tools for other logging libraries
- `/infrastructure/` - Terraform and Kubernetes configurations
- `/test-*/` - Test directories

**Why These Exclusions?**
1. **Examples** - Use simplified patterns for clarity, including non-cryptographic random numbers
2. **Benchmarks** - Require deterministic random for reproducible performance tests
3. **Migration Tools** - Need file system access to read and transform user code
4. **Infrastructure** - Managed separately with infrastructure-specific security tools

## 🔐 Security Best Practices

### For Users

When using Bolt in production:

1. **Sanitize Sensitive Data**
   ```go
   // DO NOT log sensitive data directly
   logger.Info().Str("password", password).Msg("user login") // ❌ Bad

   // Sanitize or redact sensitive fields
   logger.Info().Str("user", username).Msg("user login") // ✅ Good
   ```

2. **Validate External Input**
   ```go
   // Validate/sanitize user input before logging
   if isValid(userInput) {
       logger.Info().Str("input", sanitize(userInput)).Msg("processing")
   }
   ```

3. **Secure Log Destinations**
   ```go
   // Use secure file permissions
   file, _ := os.OpenFile("app.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0600)
   logger := bolt.New(bolt.NewJSONHandler(file))
   ```

4. **Rate Limiting**
   ```go
   // Implement rate limiting for untrusted sources
   if rateLimiter.Allow() {
       logger.Warn().Msg("potential attack detected")
   }
   ```

5. **Log Rotation & Retention**
   - Implement log rotation to prevent disk exhaustion
   - Set appropriate retention policies
   - Ensure logs are backed up securely

### For Contributors

When contributing code:

1. **Input Validation**
   - Validate all external inputs (field keys, values, messages)
   - Check for buffer overflow conditions
   - Handle edge cases (empty strings, maximum values)

2. **Memory Safety**
   - Avoid unsafe pointer operations
   - Use bounds checking for slice/array access
   - Prevent buffer overflows in append operations

3. **Concurrency Safety**
   - Use proper synchronization (mutexes, atomic operations)
   - Avoid race conditions in shared state
   - Test with `go test -race`

4. **Error Handling**
   - Handle all error conditions
   - Avoid panics in production code paths
   - Fail securely (don't expose internal details)

## 🚨 Security Advisories

Published security advisories can be found at:
https://github.com/felixgeelhaar/bolt/security/advisories

Subscribe to security notifications:
1. Go to the repository
2. Click "Watch" → "Custom" → "Security alerts"

## 📜 Security Audit History

### Audits Completed

- **v1.0.0** (2024-07): Initial security review
- **v1.2.0** (2024-10): Fuzzing integration and vulnerability fixes
- **v1.3.0** (2025-01): Comprehensive security hardening

### Known Security Enhancements

- **v1.2.0**: Added input validation for field keys
- **v1.2.0**: Fixed UTF-8 handling vulnerabilities
- **v1.2.0**: Resolved integer overflow issues
- **v1.3.0**: Enhanced fuzzing test coverage
- **v1.3.0**: Expanded edge case testing

## 🔧 Production Code Security

The core Bolt library (in the root directory) maintains:
- **Zero security vulnerabilities** in production code
- **Zero allocations** for performance
- **Type safety** throughout the API
- **Secure defaults** for all configurations

### Security Features

- **No external dependencies** in core library (except OpenTelemetry)
- **Read-only operations** by default
- **Structured logging** prevents injection attacks
- **Type-safe field methods** prevent type confusion
- **Atomic operations** for thread safety
- **Input validation** for all user-provided data

## 🔍 Regular Security Audits

We run automated security scanning on every commit using:
- `gosec` for Go security analysis
- GitHub Security Advisories
- Dependency vulnerability scanning
- CodeQL analysis
- Fuzzing tests for input validation

Production code maintains a clean security report with zero vulnerabilities.

## 🔗 Additional Resources

- [OWASP Logging Cheat Sheet](https://cheatsheetseries.owasp.org/cheatsheets/Logging_Cheat_Sheet.html)
- [CWE-117: Improper Output Neutralization for Logs](https://cwe.mitre.org/data/definitions/117.html)
- [Go Security Best Practices](https://go.dev/security/best-practices/)

## 📧 Security Contact

For urgent security issues requiring immediate attention:

**Security Team**: security@bolt.dev

---

**Thank you for helping keep Bolt and its users safe!** 🔐