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

### Security Best Practices

When using Bolt in production:

1. **Sanitize user input** before logging:
   ```go
   // Good: Sanitize untrusted input
   logger.Info().Str("user_input", sanitize(userInput)).Msg("User action")
   
   // Avoid: Logging raw user input directly
   logger.Info().Str("raw_input", userInput).Msg("User action")
   ```

2. **Avoid logging sensitive data**:
   ```go
   // Good: Log non-sensitive identifiers
   logger.Info().Str("user_id", userID).Msg("Login successful")
   
   // Bad: Never log passwords or tokens
   logger.Info().Str("password", password).Msg("Login attempt")
   ```

3. **Configure appropriate log levels** for production
4. **Secure log storage** and transmission
5. **Regular dependency updates** to get security patches

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