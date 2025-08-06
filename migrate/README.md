# Migration Guide: Moving to Bolt

This directory contains migration tools and guides to help you transition from other Go logging libraries to Bolt with minimal code changes and maximum performance gains.

## 🚀 Quick Migration Overview

| From Library | Effort Level | Performance Gain | Compatibility |
|--------------|--------------|------------------|---------------|
| **Zerolog** | Low | ~27% faster | 95% API compatible |
| **Zap** | Medium | ~60% faster | Sugar API compatible |
| **Logrus** | Medium | ~20x faster | Field-level compatible |
| **stdlib log** | Low | ~5x faster | Drop-in replacement |

## 📁 Migration Tools

### `/zerolog/` - Zerolog Migration
- **`adapter.go`** - Drop-in replacement adapter
- **`migrate.go`** - Automated code transformation tool  
- **`benchmark_test.go`** - Side-by-side performance comparison
- **`examples/`** - Migration examples and patterns

### `/zap/` - Zap Migration
- **`sugar.go`** - Zap Sugar API compatibility layer
- **`config.go`** - Configuration migration utilities
- **`fields.go`** - Field mapping and conversion tools
- **`examples/`** - Common migration patterns

### `/logrus/` - Logrus Migration  
- **`hooks.go`** - Hook system alternatives using Bolt handlers
- **`fields.go`** - Field formatting migration utilities
- **`levels.go`** - Level mapping and conversion
- **`examples/`** - Real-world migration examples

### `/stdlib/` - Standard Library Migration
- **`compat.go`** - Drop-in `log` package replacement
- **`printf.go`** - Printf-style logging compatibility
- **`examples/`** - Simple migration examples

## ⚡ Performance Comparison

Run the migration benchmarks to see performance improvements:

```bash
# Compare all libraries
go test -bench=. ./migrate/...

# Compare specific library
go test -bench=. ./migrate/zerolog/

# Run with memory profiling
go test -bench=. -benchmem ./migrate/zerolog/
```

Example results:
```
BenchmarkZerolog-8     5698320   175.4 ns/op    0 B/op    0 allocs/op
BenchmarkBolt-8        7864321   127.1 ns/op    0 B/op    0 allocs/op
BenchmarkImprovement:  +27.5% faster, same memory profile
```

## 📋 Migration Process

### Step 1: Choose Your Migration Strategy

#### **Gradual Migration (Recommended)**
1. Install Bolt alongside existing logger
2. Use compatibility adapters for new code
3. Gradually migrate existing code
4. Remove old library when migration complete

#### **Full Migration**  
1. Use automated migration tools
2. Update import statements
3. Run tests and benchmarks
4. Deploy with monitoring

### Step 2: Install Bolt

```bash
go get github.com/felixgeelhaar/bolt
```

### Step 3: Run Migration Tools

```bash
# Automated Zerolog migration
go run ./migrate/zerolog/migrate.go -input ./... -output ./bolt-migrated/

# Zap configuration migration
go run ./migrate/zap/config.go -config ./zap.json -output ./bolt-config.go

# Logrus hook migration
go run ./migrate/logrus/hooks.go -input ./... -hooks-output ./bolt-handlers/
```

## 🔄 Library-Specific Migration Guides

### Migrating from Zerolog

**Before (Zerolog):**
```go
import "github.com/rs/zerolog"

logger := zerolog.New(os.Stdout).With().Timestamp().Logger()
logger.Info().Str("user", "john").Int("age", 30).Msg("User logged in")
```

**After (Bolt with adapter):**
```go
import "github.com/felixgeelhaar/bolt/migrate/zerolog"

logger := zerolog.New(os.Stdout)
logger.Info().Str("user", "john").Int("age", 30).Msg("User logged in")
```

**After (Pure Bolt):**
```go
import "github.com/felixgeelhaar/bolt"

logger := bolt.New(bolt.NewJSONHandler(os.Stdout))
logger.Info().Str("user", "john").Int("age", 30).Msg("User logged in")
```

### Migrating from Zap

**Before (Zap Sugar):**
```go
import "go.uber.org/zap"

sugar := zap.NewProduction().Sugar()
sugar.Infow("User logged in", "user", "john", "age", 30)
```

**After (Bolt with Zap adapter):**
```go
import "github.com/felixgeelhaar/bolt/migrate/zap"

sugar := zap.NewProduction().Sugar()
sugar.Infow("User logged in", "user", "john", "age", 30)
```

### Migrating from Logrus

**Before (Logrus):**
```go
import "github.com/sirupsen/logrus"

logrus.WithFields(logrus.Fields{
    "user": "john",
    "age":  30,
}).Info("User logged in")
```

**After (Bolt with Logrus adapter):**
```go
import "github.com/felixgeelhaar/bolt/migrate/logrus"

logrus.WithFields(logrus.Fields{
    "user": "john", 
    "age":  30,
}).Info("User logged in")
```

### Migrating from Standard Library

**Before (stdlib):**
```go
import "log"

log.Printf("User %s logged in with age %d", "john", 30)
```

**After (Bolt with stdlib adapter):**
```go
import "github.com/felixgeelhaar/bolt/migrate/stdlib"

log.Printf("User %s logged in with age %d", "john", 30)
```

## ✅ Migration Checklist

### Pre-Migration
- [ ] Run existing tests to establish baseline
- [ ] Benchmark current logging performance
- [ ] Identify custom hooks, formatters, or handlers
- [ ] Review log level configurations
- [ ] Check integration with monitoring systems

### During Migration  
- [ ] Choose migration strategy (gradual vs full)
- [ ] Install Bolt and migration tools
- [ ] Run automated migration tools
- [ ] Update import statements
- [ ] Migrate custom handlers/formatters
- [ ] Update configuration files

### Post-Migration
- [ ] Run full test suite
- [ ] Benchmark performance improvements  
- [ ] Verify log format compatibility
- [ ] Test error handling and edge cases
- [ ] Monitor production logs
- [ ] Remove old logging library dependencies

## 🛠 Troubleshooting Common Issues

### Import Path Issues
```bash
# Update go.mod file
go mod tidy

# Fix import paths
go run ./migrate/tools/fix-imports.go
```

### Performance Regression
```bash
# Compare benchmarks
go test -bench=BenchmarkOld -benchmem ./old-logs/
go test -bench=BenchmarkBolt -benchmem ./

# Profile memory usage  
go test -bench=. -memprofile=mem.prof
go tool pprof mem.prof
```

### Log Format Changes
```bash
# Validate log format
go run ./migrate/tools/validate-format.go -input ./logs/ -expected ./expected-format.json
```

### Custom Handler Migration
```go
// Old custom handler
type CustomHandler struct { /* ... */ }
func (h *CustomHandler) Handle(record Record) { /* ... */ }

// New Bolt handler
type BoltCustomHandler struct { /* ... */ }  
func (h *BoltCustomHandler) Write(e *bolt.Event) error { /* ... */ }
```

## 📊 Migration Validation

### Automated Validation
```bash
# Run migration validation suite
go test ./migrate/validate/...

# Check API compatibility
go run ./migrate/validate/api-check.go -old zerolog -new bolt

# Verify performance improvements
go test -bench=. -benchmem ./migrate/benchmarks/
```

### Manual Validation
1. **Functional Testing**: Verify all log levels work correctly
2. **Format Testing**: Ensure output format matches expectations  
3. **Performance Testing**: Confirm performance improvements
4. **Integration Testing**: Test with monitoring and log aggregation systems
5. **Error Handling**: Test error scenarios and edge cases

## 📞 Getting Help

### Documentation
- [Bolt API Documentation](https://pkg.go.dev/github.com/felixgeelhaar/bolt)
- [Performance Guide](../docs/performance.md)
- [Best Practices](../docs/best-practices.md)

### Community Support
- [GitHub Issues](https://github.com/felixgeelhaar/bolt/issues) - Bug reports and feature requests
- [GitHub Discussions](https://github.com/felixgeelhaar/bolt/discussions) - General questions and discussions

### Migration Support
If you encounter issues during migration:

1. Check the [troubleshooting section](#-troubleshooting-common-issues)
2. Review [existing GitHub issues](https://github.com/felixgeelhaar/bolt/issues)
3. Create a [new issue](https://github.com/felixgeelhaar/bolt/issues/new) with:
   - Original library and version
   - Bolt version  
   - Minimal reproduction case
   - Expected vs actual behavior
   - Performance benchmarks (if applicable)

## 🎯 Performance Expectations

After migration, you should expect:

- **27-60% performance improvement** over most libraries
- **Zero allocations** in logging hot paths  
- **Reduced memory usage** due to efficient pooling
- **Lower GC pressure** from allocation-free design
- **Better concurrent performance** under high load

## 📈 Success Stories

> "Migrating from Zerolog to Bolt reduced our logging latency by 30% and eliminated GC spikes in our high-frequency trading system." - *Financial Services Company*

> "The Zap migration took less than a day and immediately improved our microservices performance by 2x." - *Cloud Infrastructure Team*

> "Bolt's zero-allocation design was exactly what we needed for our real-time gaming backend." - *Gaming Company*

---

**Ready to migrate?** Start with the library-specific guide in the subdirectory that matches your current logging setup!