# 🚨 Deprecation Notice: Bolt v1

## v1 is Deprecated - Please Upgrade to v2

**Bolt v1 is no longer maintained and contains critical security vulnerabilities.**

### 🔒 Security Issues in v1:
- **JSON injection vulnerability** - Unescaped string values can break log parsing
- **Race conditions** - SetLevel is not thread-safe
- **Silent failures** - Handler errors are ignored

### ⚡ Performance Improvements in v2:
- **27% faster** - 98ns/op vs 127ns/op in v1
- **Zero allocations maintained** - Still 0 allocs/op
- **Better concurrency** - Thread-safe under high load

### 🏢 Enterprise Features in v2:
- Complete security fixes and input validation
- Comprehensive migration tools from other libraries
- Production deployment examples and monitoring
- Enterprise compliance documentation (SOX, HIPAA, GDPR)

## 🚀 Easy Migration

### Install v2:
```bash
go get github.com/felixgeelhaar/bolt/v2
```

### Update imports:
```go
// Old (v1)
import "github.com/felixgeelhaar/bolt"

// New (v2) 
import "github.com/felixgeelhaar/bolt/v2"
```

### API Remains the Same:
```go
// Same API - just import change needed
logger := bolt.New(bolt.NewJSONHandler(os.Stdout))
logger.Info().Str("key", "value").Msg("Hello World")
```

## 📚 Migration Resources

- **[Complete Migration Guide](./migrate/README.md)** - Step-by-step instructions
- **[Security Improvements](./SECURITY.md)** - What's been fixed
- **[Performance Benchmarks](./PERFORMANCE.md)** - Speed improvements
- **[Enterprise Features](./ENTERPRISE.md)** - New capabilities

## ⏰ Timeline

- **Now**: v1 deprecated, v2 is stable and production-ready
- **3 months**: v1 support ends
- **6 months**: v1 documentation removed

## 🆘 Need Help?

- [GitHub Issues](https://github.com/felixgeelhaar/bolt/issues) - Bug reports and questions
- [GitHub Discussions](https://github.com/felixgeelhaar/bolt/discussions) - Community support
- [Migration Guide](./migrate/README.md) - Detailed upgrade instructions

**Please upgrade to v2 immediately for security and performance benefits.**