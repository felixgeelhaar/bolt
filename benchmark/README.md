# Bolt Performance Benchmarking Suite

A comprehensive, enterprise-grade performance benchmarking and monitoring system for the Bolt logging library.

## 🚀 Features

### Core Capabilities
- **Competitive Analysis**: Compare Bolt against Zerolog, Zap, Logrus, and Slog
- **Enterprise Scenarios**: Real-world testing with high-frequency trading, web services, microservices
- **Regression Detection**: Automated performance regression alerts
- **Statistical Analysis**: Comprehensive statistical validation with confidence intervals
- **Professional Reporting**: Interactive HTML reports with charts and visualizations
- **Continuous Monitoring**: GitHub Actions integration with automated alerts

### Key Metrics
- **Zero-allocation validation** - Ensures Bolt maintains zero heap allocations
- **Sub-100ns latency** - Validates ultra-low latency requirements  
- **Statistical significance** - Multiple iterations with confidence intervals
- **Resource monitoring** - Memory usage, GC pressure, CPU utilization
- **Quality gates** - Pass/fail criteria for CI/CD pipelines

## 📊 Quick Start

### Command Line Interface

```bash
# Build the benchmark tool
cd benchmark/cmd/bolt-benchmark
go build -o ../../../bolt-benchmark .

# Run competitive analysis
./bolt-benchmark -type=competitive -duration=10m

# Run enterprise scenarios
./bolt-benchmark -type=enterprise -scenarios=WebAPI,Trading -duration=15m

# Generate HTML reports
./bolt-benchmark -type=competitive -html=true -output=./results
```

### Programmatic Usage

```go
package main

import (
    "context"
    "github.com/felixgeelhaar/bolt/benchmark/framework"
    "github.com/felixgeelhaar/bolt/benchmark/enterprise"
)

func main() {
    // Competitive analysis
    analyzer := framework.NewCompetitiveAnalyzer().
        WithLibraries(framework.LibraryBolt, framework.LibraryZerolog).
        WithIterations(10).
        WithOutputDir("./results")
    
    ctx := context.Background()
    err := analyzer.RunComprehensiveAnalysis(ctx)
    if err != nil {
        panic(err)
    }

    // Generate reports
    reportGen := framework.NewReportGenerator(analyzer)
    err = reportGen.GenerateHTMLReport()
    if err != nil {
        panic(err)
    }
}
```

## 🏗️ Architecture

### Package Structure

```
benchmark/
├── cmd/bolt-benchmark/     # CLI application
├── framework/              # Core benchmarking framework
│   ├── competitive_analysis.go
│   └── reporting.go
├── enterprise/             # Enterprise scenario testing
│   └── enterprise_suite.go
├── validation/             # Performance validation
│   └── performance_validator.go
├── alerting/              # Alert system
│   └── performance_alerting.go
└── README.md              # This file
```

### Core Components

#### 1. Competitive Analysis Framework (`framework/`)
- **CompetitiveAnalyzer**: Orchestrates multi-library benchmarks
- **TestScenario**: Defines realistic logging scenarios
- **BenchmarkResult**: Captures comprehensive performance metrics
- **ReportGenerator**: Creates professional HTML/JSON/CSV reports

#### 2. Enterprise Benchmarking (`enterprise/`)
- **EnterpriseBenchmarkSuite**: Production-grade scenario testing
- **EnterpriseScenario**: Real-world enterprise patterns
- **LoadProfile**: Different load generation patterns
- **Resource Monitoring**: Memory, CPU, and GC tracking

#### 3. Performance Validation (`validation/`)
- **PerformanceValidator**: Threshold validation and regression detection
- **ValidationResult**: Comprehensive validation outcomes
- **QualityGates**: Pass/fail criteria for CI/CD
- **Statistical Analysis**: Confidence intervals and outlier detection

#### 4. Alerting System (`alerting/`)
- **PerformanceAlerting**: Multi-channel alert delivery
- **Alert Channels**: Slack, Discord, Teams, Email, GitHub Issues
- **Rate Limiting**: Prevents alert spam
- **Severity Levels**: Critical, Warning, Info classifications

## 📈 Benchmark Scenarios

### Competitive Analysis Scenarios

| Scenario | Description | Fields | Concurrency |
|----------|-------------|--------|-------------|
| WebAPI | Typical web service logging | 5 | 10 |
| Microservice | High-frequency internal logging | 3 | 50 |
| HighFrequencyTrading | Ultra-low latency requirements | 2 | 1 |
| DataPipeline | Batch processing with detailed logs | 10 | 20 |
| ContainerOrchestration | Container metadata logging | 8 | 100 |

### Enterprise Scenarios

| Scenario | Target RPS | Duration | Memory Limit | Description |
|----------|-----------|----------|--------------|-------------|
| HighFrequencyTrading | 100,000 | 5m | 100MB | Sub-microsecond trading logs |
| WebServiceAPI | 25,000 | 10m | 500MB | Bursty web traffic patterns |
| MicroserviceMesh | 50,000 | 15m | 1GB | Distributed tracing logs |
| DataPipeline | 75,000 | 20m | 2GB | High-volume batch processing |
| SecurityAuditSystem | 10,000 | 30m | 256MB | Compliance logging |
| ContainerOrchestration | 40,000 | 25m | 1GB | Container runtime logs |

## 🔧 Configuration

### Environment Variables

```bash
# CI/CD Integration
export GITHUB_TOKEN="your-token"
export GITHUB_REPOSITORY="felixgeelhaar/bolt"

# Performance Thresholds
export MAX_LATENCY_NS="100000"        # 100μs
export MIN_THROUGHPUT_OPS="10000000"  # 10M ops/sec
export MAX_ALLOCATIONS="0"            # Zero allocations

# Alerting Configuration
export SLACK_WEBHOOK_URL="https://hooks.slack.com/..."
export DISCORD_WEBHOOK_URL="https://discord.com/api/webhooks/..."

# Resource Limits
export MAX_MEMORY_MB="4096"
export MAX_CPU_PERCENT="95"
```

### Configuration Files

Create `benchmark-config.json`:

```json
{
  "thresholds": {
    "max_latency_ns": 100000,
    "max_latency_p95_ns": 150000,
    "max_latency_p99_ns": 200000,
    "max_allocs_per_op": 0,
    "min_throughput_rps": 10000000,
    "max_latency_regression": 0.05,
    "max_allocation_regression": 0.01
  },
  "alerting": {
    "alert_on_regression": true,
    "alert_on_threshold": true,
    "create_github_issues": true,
    "cooldown_period": "30m",
    "max_alerts_per_hour": 10
  }
}
```

## 🚀 GitHub Actions Integration

The benchmarking suite integrates seamlessly with GitHub Actions for continuous performance monitoring:

### Features
- **PR Performance Checks**: Automatic regression detection on pull requests
- **Scheduled Monitoring**: Daily/weekly performance validation
- **Results Deployment**: Automatic deployment to GitHub Pages
- **Alert Integration**: Instant notifications on performance regressions
- **Baseline Management**: Automatic baseline updates on main branch

### Workflow Configuration

See `.github/workflows/performance-monitoring.yml` for the complete setup.

Key capabilities:
- Multi-platform testing (Linux, macOS, Windows)
- Multiple Go versions (1.21, 1.22, 1.23)
- Resource-constrained testing
- Statistical analysis with benchstat
- Professional HTML report generation

## 📊 Reporting & Visualization

### HTML Reports
Interactive reports with:
- **Performance Charts**: Bar charts, line graphs, heatmaps
- **Statistical Analysis**: Confidence intervals, outlier detection
- **Comparison Tables**: Library rankings and metrics
- **Regression Analysis**: Historical trend visualization
- **Quality Gates**: Pass/fail status indicators

### JSON Data Export
Machine-readable format for:
- External tooling integration
- Custom analysis workflows
- API consumption
- Data warehouse ingestion

### CSV Exports
Tabular data for:
- Spreadsheet analysis
- Statistical software (R, Python)
- Business intelligence tools
- Historical data analysis

## 🚨 Performance Validation

### Quality Gates

| Gate | Criteria | Impact |
|------|----------|---------|
| **Zero Allocations** | 0 allocs/op | CRITICAL |
| **Latency Threshold** | < 100ns/op | HIGH |
| **Performance Consistency** | CV < 10% | MEDIUM |
| **Regression Detection** | < 5% degradation | HIGH |
| **Resource Usage** | < limits | MEDIUM |

### Validation Process

1. **Threshold Validation**: Check against absolute performance limits
2. **Regression Detection**: Compare against historical baselines  
3. **Statistical Analysis**: Assess result reliability and consistency
4. **Quality Gate Evaluation**: Determine overall pass/fail status
5. **Recommendation Generation**: Provide actionable improvement suggestions

### Alert Triggers

- **Critical**: Zero-allocation violations, >10% performance regression
- **Warning**: Threshold violations, 5-10% performance degradation  
- **Info**: Successful validation, performance improvements

## 🔍 Advanced Features

### Statistical Analysis
- **Confidence Intervals**: 95% confidence bounds on measurements
- **Outlier Detection**: IQR-based outlier identification
- **Coefficient of Variation**: Measurement consistency assessment
- **Percentile Analysis**: P50, P95, P99 latency distribution

### Load Testing
- **Concurrent Load Generation**: Configurable worker pool sizes
- **Rate Limiting**: Precise RPS control with token bucket
- **Load Profiles**: Constant, burst, spike, ramp, random patterns
- **Resource Monitoring**: Real-time memory, CPU, GC tracking

### Enterprise Integration
- **RBAC Support**: Role-based access control for team environments
- **Audit Logging**: Comprehensive benchmark execution logs
- **Multi-tenant**: Isolated benchmarking for multiple teams
- **API Gateway**: RESTful API for programmatic access

## 🛠️ Development Guide

### Adding New Libraries

```go
// Add to LibraryType enum
const (
    LibraryBolt LibraryType = iota
    LibraryNewLib // Add here
)

// Implement logger setup
func (ca *CompetitiveAnalyzer) setupLogger(library LibraryType) (interface{}, error) {
    switch library {
    case LibraryNewLib:
        return newlib.New(newlib.Options{}), nil
    }
}

// Implement logging method
func (ca *CompetitiveAnalyzer) logMessage(library LibraryType, logger interface{}, scenario TestScenario, iteration int) {
    switch library {
    case LibraryNewLib:
        l := logger.(*newlib.Logger)
        l.Info("message", fields...)
    }
}
```

### Adding New Scenarios

```go
var CustomScenario = TestScenario{
    Name:        "CustomWorkload",
    Description: "Custom application workload pattern",
    Fields:      7,
    Concurrency: 25,
    MessageSize: "medium",
    LogLevel:    "info",
    Duration:    8 * time.Minute,
}

// Add to analyzer
analyzer := framework.NewCompetitiveAnalyzer().
    WithScenarios(CustomScenario)
```

### Custom Validation Rules

```go
validator := validation.NewPerformanceValidator().
    WithThresholds(validation.PerformanceThresholds{
        MaxLatencyNs:      50000,  // Custom 50μs limit
        MaxAllocsPerOp:    0,      // Still zero allocations
        MinThroughputRPS:  20000000, // Higher throughput requirement
    })

result, err := validator.ValidateResults("./results")
```

## 🚀 Performance Optimization Tips

### For Bolt Library Development
1. **Profile regularly**: Use built-in CPU and memory profiling
2. **Monitor allocations**: Zero-allocation requirement is critical
3. **Benchmark variations**: Test different message sizes and field counts
4. **Statistical significance**: Run multiple iterations for reliable results
5. **Resource limits**: Test under memory and CPU constraints

### For Application Usage
1. **Choose appropriate scenarios**: Match your use case patterns
2. **Set realistic thresholds**: Based on your application requirements
3. **Monitor trends**: Track performance over time, not just point measurements
4. **Validate in CI/CD**: Prevent performance regressions in production
5. **Use profiling data**: Identify optimization opportunities

## 📚 API Reference

### Command Line Interface

```bash
bolt-benchmark [OPTIONS]

OPTIONS:
  -type string
        Benchmark type: competitive, enterprise, regression, load (default "competitive")
  -output string
        Output directory for results (default "benchmark-results")
  -duration duration
        Test duration (default 5m0s)
  -libraries string
        Comma-separated list of libraries (default "Bolt,Zerolog,Zap,Logrus,Slog")
  -scenarios string
        Comma-separated list of scenarios or 'all' (default "all")
  -iterations int
        Number of benchmark iterations (default 10)
  -parallel int
        Number of parallel workers (default CPU count)
  -profiling
        Enable CPU and memory profiling (default true)
  -html
        Generate HTML reports (default true)
  -csv
        Generate CSV exports (default true)
  -max-memory int
        Maximum memory usage in MB (default 4096)
  -max-cpu float
        Maximum CPU usage percentage (default 95)
  -quiet
        Suppress non-essential output
  -version
        Show version information
```

### Programmatic API

See [GoDoc](https://pkg.go.dev/github.com/felixgeelhaar/bolt/benchmark) for complete API documentation.

## 🤝 Contributing

1. **Fork the repository**
2. **Create feature branch**: `git checkout -b feature/amazing-feature`
3. **Run benchmarks**: Ensure no performance regressions
4. **Add tests**: Include benchmark tests for new features
5. **Update documentation**: Keep README and API docs current
6. **Submit pull request**: Include performance impact analysis

### Performance Requirements for Contributions
- Maintain zero allocations in hot paths
- No more than 5% performance regression
- Statistical significance (>10 iterations)
- Cross-platform compatibility
- Memory usage within limits

## 📄 License

This project is licensed under the MIT License - see the [LICENSE](../LICENSE) file for details.

## 🙏 Acknowledgments

- Go team for excellent benchmark tooling
- Zerolog, Zap, Logrus, and Slog teams for performance comparison baselines
- GitHub Actions team for CI/CD integration capabilities
- Open source community for feedback and contributions

---

**🚀 Ready to benchmark? Start with:**
```bash
./bolt-benchmark -type=competitive -duration=5m -html=true
```

For questions, issues, or contributions, visit [GitHub Issues](https://github.com/felixgeelhaar/bolt/issues).