# Security Policy

## Supported Versions

We actively support the following versions of Bolt with security updates:

| Version | Supported          |
| ------- | ------------------ |
| 1.0.x   | :white_check_mark: |
| < 1.0   | :x:                |

## Reporting a Vulnerability

We take the security of Bolt seriously. If you discover a security vulnerability, please report it responsibly.

### How to Report

1. **Do NOT create a public GitHub issue** for security vulnerabilities
2. **Use GitHub's private vulnerability reporting** feature:
   - Go to the [Security tab](https://github.com/felixgeelhaar/bolt/security/advisories/new)
   - Click "Report a vulnerability"
   - Fill out the advisory form with details

3. **Alternative**: Email security reports to:
   - **Email**: felix.geelhaar@gmail.com
   - **Subject**: [SECURITY] Bolt Vulnerability Report
   - **PGP Key**: Available on request

### What to Include

Please include the following information in your report:

- **Description** of the vulnerability
- **Steps to reproduce** the issue
- **Potential impact** and attack scenarios
- **Affected versions** of Bolt
- **Suggested fix** if you have one
- **Your contact information** for follow-up

### Response Process

1. **Acknowledgment**: We'll acknowledge receipt within **24 hours**
2. **Investigation**: We'll investigate and validate the report within **72 hours**
3. **Fix Development**: We'll work on a fix with appropriate urgency
4. **Disclosure**: We'll coordinate responsible disclosure with you
5. **Release**: We'll release a security update and advisory

### Security Update Process

- **Critical vulnerabilities**: Emergency release within 48 hours
- **High severity**: Release within 1 week
- **Medium/Low severity**: Include in next scheduled release

### Scope

This security policy covers:

- **Core logging functionality** that could lead to information disclosure
- **Memory safety issues** that could cause crashes or corruption
- **Injection vulnerabilities** in log formatting
- **Denial of service** vulnerabilities
- **Dependencies** with known security issues

### Out of Scope

The following are generally not considered security vulnerabilities:

- **Performance issues** (unless they enable DoS attacks)
- **Feature requests** or enhancements
- **Issues in example code** or documentation
- **Theoretical attacks** without practical exploitation
- **Issues requiring physical access** to the system

## Built-in Security Features

Bolt incorporates multiple layers of security protection:

### 🛡️ Automatic Input Validation

All inputs are automatically validated and sanitized:

```go
// JSON injection prevention - all strings are automatically escaped
userInput := `"},"admin":true,"injected":"` 
logger.Info().Str("user_data", userInput).Msg("Safe logging")
// Output: {"level":"info","user_data":"\"}\"admin\":true,\"injected\":\"","message":"Safe logging"}

// Size limits prevent resource exhaustion attacks
// - Keys: max 256 characters
// - Values: max 64KB per field  
// - Total buffer: max 1MB per log entry
```

### 🔐 Thread Safety & Concurrency Protection

All operations use atomic synchronization for complete thread safety:

```go
// Thread-safe level changes using atomic operations
var logger = bolt.New(bolt.NewJSONHandler(os.Stdout))

go func() {
    logger.SetLevel(bolt.DEBUG) // Atomic operation - no race conditions
}()

go func() {
    logger.Info().Msg("Concurrent logging") // Safe concurrent access
}()
```

### ⚡ Memory Safety

Event pooling and buffer management prevent memory corruption:

```go
// Event lifecycle automatically managed
// Pool reuse prevents memory leaks and corruption
for i := 0; i < 1000; i++ {
    go func(id int) {
        logger.Info().Int("goroutine", id).Msg("safe concurrent access")
        // Events automatically returned to pool
    }(i)
}
```

### Security Best Practices

When using Bolt in production:

#### 1. Input Sanitization (Built-in Protection)
```go
// ✅ Bolt automatically handles dangerous input
logger.Info().Str("user_input", userInput).Msg("User action")  // Always safe

// ✅ Control character filtering is automatic  
logger.Info().Str("key\x00\n\r", "value").Msg("test") // Rejected automatically

// ✅ Size limits prevent DoS attacks
oversizedInput := strings.Repeat("A", 100000)
logger.Info().Str("data", oversizedInput).Msg("test") // Safely handled
```

#### 2. Sensitive Data Protection
```go
// ✅ Use data masking for sensitive information
logger.Info().
    Str("user_id", userID).                    // Safe identifier
    Str("email", maskEmail(email)).            // Masked: u***@example.com
    Str("ip", maskIP(clientIP)).               // Masked: 192.168.***.*
    Str("session", hashSession(sessionID)).    // Hashed session ID
    Msg("User action")

// ❌ Never log credentials directly
logger.Info().Str("password", password).Msg("Login") // DON'T DO THIS

// ✅ Structured redaction for complex data
type SecureEvent struct {
    UserID   string `json:"user_id"`
    Password string `json:"-"`           // Never serialized
    Token    string `json:"token,omitempty"` // Omit if empty
}
```

#### 3. Error Handling Security
```go
// ✅ Custom error handlers prevent information disclosure
logger := bolt.New(bolt.NewJSONHandler(os.Stdout)).
    SetErrorHandler(func(err error) {
        // Log to monitoring system, don't expose to users
        monitoring.ReportSecurityEvent("logging_error", err)
    })

// ✅ Safe error serialization
logger.Info().Any("data", maliciousData).Msg("safe")
// Malicious data becomes: {"data":"!ERROR: serialization failed!","message":"safe"}
```

#### 4. Production Configuration
```go
func setupSecureLogger() *bolt.Logger {
    // Structured JSON for security monitoring
    handler := bolt.NewJSONHandler(os.Stdout)
    
    return bolt.New(handler).
        SetLevel(bolt.WARN).                    // Limit information disclosure
        SetErrorHandler(secureErrorHandler)     // Custom error handling
}

// Environment-based security configuration
export BOLT_LEVEL=warn           # Limit debug information in production
export BOLT_FORMAT=json          # Structured format for monitoring
```

#### 5. Network Security
```go
// ✅ Secure transport for remote logging
func setupRemoteLogging() *bolt.Logger {
    conn, err := tls.Dial("tcp", "logs.example.com:443", &tls.Config{
        ServerName: "logs.example.com",
        MinVersion: tls.VersionTLS12,
    })
    if err != nil {
        log.Fatal(err)
    }
    
    return bolt.New(bolt.NewJSONHandler(conn))
}

// ✅ Rate limiting to prevent DoS
type RateLimitedLogger struct {
    *bolt.Logger
    limiter *rate.Limiter
}

func (rl *RateLimitedLogger) Info() *bolt.Event {
    if !rl.limiter.Allow() {
        return &bolt.Event{} // No-op when rate limited
    }
    return rl.Logger.Info()
}
```

#### 6. Container Security
```dockerfile
# Secure container deployment
FROM golang:1.21-alpine AS builder
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags='-w -s' -o app

FROM scratch
COPY --from=builder /app /app
USER 65534:65534  # Non-root user
ENTRYPOINT ["/app"]
```

### Security Testing & Validation

#### Automated Security Testing
```go
// Fuzz testing for input validation
func FuzzStringField(f *testing.F) {
    f.Add(`"},"admin":true,"`)  // JSON injection attempt
    f.Add("\x00\x01\x02")       // Control characters
    f.Add(strings.Repeat("A", 10000)) // Large input
    
    f.Fuzz(func(t *testing.T, input string) {
        logger.Info().Str("test", input).Msg("fuzz test")
        // Should never panic or produce invalid JSON
    })
}

// Security regression tests
func TestSecurityVulnerabilities(t *testing.T) {
    injectionAttempts := []string{
        `"},"admin":true,"`,
        `\"},\"level\":\"error\",\"fake\":\"entry\"`,
        "\n{\"malicious\":\"entry\"}\n",
    }
    
    for _, attempt := range injectionAttempts {
        buf := &bytes.Buffer{}
        logger := bolt.New(bolt.NewJSONHandler(buf))
        logger.Info().Str("input", attempt).Msg("test")
        
        // Verify only one valid JSON object
        lines := strings.Split(strings.TrimSpace(buf.String()), "\n")
        assert.Equal(t, 1, len(lines), "Injection attack succeeded")
    }
}
```

#### Static Analysis Integration
```bash
# Run security-focused static analysis
gosec ./...

# Check for known vulnerabilities
govulncheck ./...

# Dependency security scanning
go mod audit
```

### Responsible Disclosure

We believe in responsible disclosure and will:

- **Acknowledge** your contribution to improving security
- **Coordinate** public disclosure timing with you
- **Credit** you in the security advisory (if desired)
- **Keep you informed** throughout the process

### Security Advisories

Security advisories will be published:

- **GitHub Security Advisories** for this repository
- **Release notes** for affected versions
- **Go vulnerability database** for broad awareness

## Contact

For security-related questions or concerns:

- **Security Reports**: Use GitHub's private vulnerability reporting
- **General Security Questions**: Create a GitHub Discussion
- **Emergency Contact**: felix.geelhaar@gmail.com

Thank you for helping keep Bolt secure! 🔒