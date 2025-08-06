# Bolt Enterprise Integration Examples

This directory contains comprehensive, production-ready examples demonstrating how to integrate the Bolt logging library in enterprise environments. Each example includes complete working code, configuration files, and documentation.

## Quick Start

```bash
# Clone the repository
git clone https://github.com/felixgeelhaar/bolt.git
cd bolt/examples

# Run any example
cd microservices/http-middleware
go run main.go
```

## Example Categories

### 🏗️ Microservices Architecture
Real-world patterns for distributed systems with proper logging, tracing, and correlation.

- [`microservices/http-middleware/`](./microservices/http-middleware/) - HTTP request/response logging with correlation IDs
- [`microservices/grpc-interceptors/`](./microservices/grpc-interceptors/) - gRPC client/server logging interceptors  
- [`microservices/service-mesh/`](./microservices/service-mesh/) - Service-to-service communication logging
- [`microservices/circuit-breaker/`](./microservices/circuit-breaker/) - Circuit breaker integration with logging
- [`microservices/event-sourcing/`](./microservices/event-sourcing/) - Event-driven architecture logging patterns

### ☁️ Cloud-Native Deployment
Container orchestration, configuration management, and cloud platform integration.

- [`cloud-native/kubernetes/`](./cloud-native/kubernetes/) - Kubernetes deployment with structured logging
- [`cloud-native/docker-compose/`](./cloud-native/docker-compose/) - Multi-service local development setup
- [`cloud-native/helm-charts/`](./cloud-native/helm-charts/) - Helm chart templates with logging configuration
- [`cloud-native/health-checks/`](./cloud-native/health-checks/) - Health monitoring and readiness probes
- [`cloud-native/config-management/`](./cloud-native/config-management/) - Dynamic configuration with logging

### 📊 Monitoring & Observability
Complete observability stack integration with metrics, traces, and alerting.

- [`observability/prometheus/`](./observability/prometheus/) - Prometheus metrics integration
- [`observability/opentelemetry/`](./observability/opentelemetry/) - Distributed tracing with OpenTelemetry
- [`observability/grafana/`](./observability/grafana/) - Grafana dashboards and log visualization
- [`observability/jaeger/`](./observability/jaeger/) - Jaeger trace correlation
- [`observability/alertmanager/`](./observability/alertmanager/) - Alert rules and notification setup

### 🔄 High-Availability Patterns
Enterprise-grade reliability, failover, and disaster recovery strategies.

- [`high-availability/load-balancer/`](./high-availability/load-balancer/) - Load balancer integration
- [`high-availability/failover/`](./high-availability/failover/) - Automatic failover logging strategies
- [`high-availability/log-aggregation/`](./high-availability/log-aggregation/) - Centralized log collection
- [`high-availability/disaster-recovery/`](./high-availability/disaster-recovery/) - DR logging patterns
- [`high-availability/backup-strategies/`](./high-availability/backup-strategies/) - Log backup and retention

### 🔐 Enterprise Security
Security-focused logging with compliance, auditing, and data protection.

- [`security/audit-logging/`](./security/audit-logging/) - Compliance audit trail implementation
- [`security/pii-masking/`](./security/pii-masking/) - PII data redaction and masking
- [`security/log-encryption/`](./security/log-encryption/) - Log encryption at rest and in transit
- [`security/gdpr-compliance/`](./security/gdpr-compliance/) - GDPR compliance patterns
- [`security/access-control/`](./security/access-control/) - Role-based access logging

### 🏭 Production Deployment
Complete production deployment scenarios with real-world configurations.

- [`production/e-commerce-platform/`](./production/e-commerce-platform/) - Full e-commerce system
- [`production/financial-services/`](./production/financial-services/) - Financial compliance and audit trails
- [`production/healthcare-hipaa/`](./production/healthcare-hipaa/) - HIPAA-compliant logging
- [`production/saas-multi-tenant/`](./production/saas-multi-tenant/) - Multi-tenant SaaS logging
- [`production/gaming-platform/`](./production/gaming-platform/) - High-throughput gaming backend

## Features Demonstrated

### Performance Optimization
- **Zero-allocation logging** in hot paths
- **Structured logging** without reflection
- **High-throughput scenarios** (millions of logs/second)
- **Memory-efficient patterns**
- **CPU profiling integration**

### Enterprise Requirements
- **Audit compliance** (SOX, GDPR, HIPAA)
- **Security standards** (encryption, access control)
- **Monitoring integration** (metrics, alerts, dashboards)
- **Operational excellence** (health checks, circuit breakers)
- **Disaster recovery** (backup, replication, failover)

### Cloud-Native Features
- **Container orchestration** (Kubernetes, Docker Swarm)
- **Service mesh integration** (Istio, Linkerd)
- **Cloud provider integration** (AWS, GCP, Azure)
- **Auto-scaling patterns**
- **Multi-region deployment**

## Quick Reference

### Common Patterns
```go
// Structured logging with correlation ID
logger.Info().
    Str("correlation_id", correlationID).
    Str("service", "auth").
    Int("user_id", userID).
    Msg("User authentication successful")

// Error logging with stack trace
logger.Error().
    Err(err).
    Str("operation", "database_query").
    Str("table", "users").
    Msg("Database operation failed")

// Performance logging with metrics
logger.Info().
    Dur("duration", duration).
    Int("records_processed", count).
    Float64("throughput_per_sec", float64(count)/duration.Seconds()).
    Msg("Batch processing completed")
```

### Configuration Examples
```go
// Production JSON logging
logger := bolt.New(bolt.NewJSONHandler(os.Stdout)).
    Level(bolt.InfoLevel).
    With().
    Str("service", "user-api").
    Str("version", "v1.2.3").
    Str("environment", "production").
    Logger()

// Development console logging
if os.Getenv("ENVIRONMENT") == "development" {
    logger = bolt.New(bolt.NewConsoleHandler(os.Stdout)).
        Level(bolt.DebugLevel).
        Logger()
}
```

## Testing the Examples

Each example includes comprehensive testing:

```bash
# Run all tests
make test-examples

# Run specific example tests
cd microservices/http-middleware
go test -v ./...

# Run benchmarks
go test -bench=. -benchmem

# Run with race detection
go test -race ./...
```

## Docker Quick Start

```bash
# Start complete observability stack
cd observability/complete-stack
docker-compose up -d

# Access services
# - Application: http://localhost:8080
# - Grafana: http://localhost:3000
# - Prometheus: http://localhost:9090
# - Jaeger: http://localhost:16686
```

## Kubernetes Deployment

```bash
# Deploy to Kubernetes
cd cloud-native/kubernetes
kubectl apply -f manifests/

# Check deployment status
kubectl get pods -l app=bolt-demo

# View logs
kubectl logs -f deployment/bolt-demo
```

## Contributing

When adding new examples:

1. **Create complete working code** - All examples must be runnable
2. **Include comprehensive documentation** - README with setup instructions
3. **Add Docker/Kubernetes configs** - Container deployment examples
4. **Provide test coverage** - Unit and integration tests
5. **Follow performance standards** - Maintain zero-allocation patterns
6. **Include monitoring setup** - Metrics and observability integration

## Performance Benchmarks

All examples maintain Bolt's performance standards:
- **Sub-100ns latency** for logging operations
- **Zero allocations** in hot paths
- **High throughput** under load
- **Memory efficiency** in long-running processes

## Support

- **Documentation**: [Bolt Documentation](https://felixgeelhaar.github.io/bolt/)
- **Issues**: [GitHub Issues](https://github.com/felixgeelhaar/bolt/issues)
- **Discussions**: [GitHub Discussions](https://github.com/felixgeelhaar/bolt/discussions)
- **Security**: See [SECURITY.md](../SECURITY.md)

## License

All examples are provided under the same license as the Bolt project. See [LICENSE](../LICENSE) for details.