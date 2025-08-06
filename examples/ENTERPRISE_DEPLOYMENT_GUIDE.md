# Bolt Enterprise Deployment Guide

This comprehensive guide provides enterprise-ready deployment patterns, configurations, and best practices for integrating the Bolt logging library in production environments.

## Table of Contents

- [Quick Start](#quick-start)
- [Architecture Patterns](#architecture-patterns)
- [Deployment Strategies](#deployment-strategies)
- [Security Considerations](#security-considerations)
- [Monitoring & Observability](#monitoring--observability)
- [High Availability](#high-availability)
- [Performance Optimization](#performance-optimization)
- [Compliance & Auditing](#compliance--auditing)
- [Troubleshooting](#troubleshooting)

## Quick Start

### Prerequisites

- Go 1.23 or later
- Docker 20.10+ and Docker Compose
- Kubernetes 1.25+ (for K8s deployments)
- 8GB+ RAM for full observability stack

### 5-Minute Demo

```bash
# Clone and setup
git clone https://github.com/felixgeelhaar/bolt.git
cd bolt/examples

# Install dependencies
make deps

# Run complete observability demo
make demo-all

# Access services
# - Bolt Demo: http://localhost:8080
# - Prometheus: http://localhost:9090
# - Grafana: http://localhost:3000
# - Jaeger: http://localhost:16686
```

### Production Deployment

```bash
# Deploy full monitoring stack
make monitoring-setup

# Deploy to Kubernetes
make k8s-deploy

# Run performance tests
make perf-test
```

## Architecture Patterns

### 1. Microservices Architecture

```
┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐
│   API Gateway   │────│  User Service   │────│   Database      │
│  (Load Balancer)│    │   (HTTP/gRPC)   │    │   (PostgreSQL)  │
└─────────────────┘    └─────────────────┘    └─────────────────┘
         │                        │                        │
         ▼                        ▼                        ▼
┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐
│   Bolt Logging  │    │   OpenTelemetry │    │    Prometheus   │
│   (Structured)  │    │   (Tracing)     │    │    (Metrics)    │
└─────────────────┘    └─────────────────┘    └─────────────────┘
```

**Key Components:**

- **HTTP Middleware**: Request/response logging with correlation IDs
- **gRPC Interceptors**: Service-to-service communication logging
- **Circuit Breakers**: Fault tolerance with logging integration
- **Service Mesh**: Distributed tracing across service boundaries

### 2. Cloud-Native Stack

```
┌─────────────────────────────────────────────────────────────┐
│                        Kubernetes                            │
├─────────────────┬─────────────────┬─────────────────────────┤
│   Application   │    Monitoring   │      Storage            │
│     Pods        │      Stack      │       Layer             │
├─────────────────┼─────────────────┼─────────────────────────┤
│ • Bolt Apps     │ • Prometheus    │ • PostgreSQL            │
│ • Health Checks │ • Grafana       │ • Redis Cache           │
│ • Auto-scaling  │ • Jaeger        │ • Elasticsearch         │
│ • Config Maps   │ • Alert Manager │ • Persistent Volumes    │
└─────────────────┴─────────────────┴─────────────────────────┘
```

**Features:**

- **Kubernetes Integration**: Native pod logging with metadata
- **Auto-scaling**: Resource-based scaling with logging correlation
- **Health Checks**: Comprehensive liveness/readiness probes
- **Configuration Management**: Dynamic config with audit logging

### 3. Observability Architecture

```
Application Layer
┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐
│      Logs       │    │     Metrics     │    │     Traces      │
│   (Bolt JSON)   │    │  (Prometheus)   │    │ (OpenTelemetry) │
└─────────────────┘    └─────────────────┘    └─────────────────┘
         │                        │                        │
         ▼                        ▼                        ▼
Collection Layer
┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐
│    Fluentd/     │    │   Prometheus    │    │     Jaeger      │
│   Fluent Bit    │    │    Server       │    │   Collector     │
└─────────────────┘    └─────────────────┘    └─────────────────┘
         │                        │                        │
         ▼                        ▼                        ▼
Storage Layer
┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐
│ Elasticsearch/  │    │   Prometheus    │    │     Jaeger      │
│     Loki        │    │     TSDB        │    │    Storage      │
└─────────────────┘    └─────────────────┘    └─────────────────┘
         │                        │                        │
         ▼                        ▼                        ▼
Visualization Layer
┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐
│   Kibana/       │    │     Grafana     │    │   Jaeger UI     │
│   Grafana       │    │   Dashboards    │    │   Trace View    │
└─────────────────┘    └─────────────────┘    └─────────────────┘
```

## Deployment Strategies

### 1. Docker Compose (Development/Testing)

```yaml
# examples/cloud-native/docker-compose/docker-compose.yml
version: '3.8'

services:
  bolt-app:
    image: bolt-examples/app:latest
    environment:
      - ENVIRONMENT=staging
      - LOG_LEVEL=info
      - OTEL_EXPORTER_JAEGER_ENDPOINT=http://jaeger:14268/api/traces
    depends_on:
      - postgres
      - redis
      - jaeger
    
  prometheus:
    image: prom/prometheus:latest
    volumes:
      - ./monitoring/prometheus.yml:/etc/prometheus/prometheus.yml
    
  grafana:
    image: grafana/grafana:latest
    environment:
      - GF_SECURITY_ADMIN_PASSWORD=admin
```

**Commands:**
```bash
# Start full stack
make compose-up

# Monitor logs
make compose-logs

# Stop services
make compose-down
```

### 2. Kubernetes (Production)

```yaml
# examples/cloud-native/kubernetes/manifests/deployment.yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: bolt-demo-app
spec:
  replicas: 3
  selector:
    matchLabels:
      app: bolt-demo-app
  template:
    spec:
      containers:
      - name: bolt-demo-app
        image: bolt-examples/app:latest
        env:
        - name: ENVIRONMENT
          value: "production"
        - name: LOG_LEVEL
          value: "info"
        resources:
          requests:
            memory: "128Mi"
            cpu: "100m"
          limits:
            memory: "256Mi"
            cpu: "200m"
        livenessProbe:
          httpGet:
            path: /health/live
            port: 8080
        readinessProbe:
          httpGet:
            path: /health/ready
            port: 8080
```

**Commands:**
```bash
# Deploy to Kubernetes
make k8s-deploy

# Check status
make k8s-status

# View logs
make k8s-logs

# Cleanup
make k8s-clean
```

### 3. High Availability Setup

```yaml
# Load Balancer Configuration
apiVersion: v1
kind: Service
metadata:
  name: bolt-app-lb
spec:
  type: LoadBalancer
  ports:
  - port: 80
    targetPort: 8080
  selector:
    app: bolt-demo-app

---
# Horizontal Pod Autoscaler
apiVersion: autoscaling/v2
kind: HorizontalPodAutoscaler
metadata:
  name: bolt-app-hpa
spec:
  scaleTargetRef:
    apiVersion: apps/v1
    kind: Deployment
    name: bolt-demo-app
  minReplicas: 3
  maxReplicas: 50
  metrics:
  - type: Resource
    resource:
      name: cpu
      target:
        type: Utilization
        averageUtilization: 70
```

## Security Considerations

### 1. Authentication & Authorization

```go
// Audit logging with user context
logger.Info().
    Str("user_id", userID).
    Str("action", "data_access").
    Str("resource", resourceID).
    Str("result", "authorized").
    Msg("User action audited")
```

### 2. PII Data Protection

```go
// Automatic PII masking
maskedEmail := piiMasker.MaskString("john.doe@example.com")
// Results in: "j***@example.com"

logger.Info().
    Str("user_email", maskedEmail).
    Msg("User data accessed with PII protection")
```

### 3. Encryption & Secrets

```yaml
# Kubernetes Secret for sensitive data
apiVersion: v1
kind: Secret
metadata:
  name: bolt-demo-secrets
type: Opaque
data:
  database-password: <base64-encoded-password>
  api-key: <base64-encoded-key>
```

### 4. Network Security

```yaml
# Network Policy for service isolation
apiVersion: networking.k8s.io/v1
kind: NetworkPolicy
metadata:
  name: bolt-app-netpol
spec:
  podSelector:
    matchLabels:
      app: bolt-demo-app
  policyTypes:
  - Ingress
  - Egress
  ingress:
  - from:
    - podSelector:
        matchLabels:
          app: load-balancer
    ports:
    - protocol: TCP
      port: 8080
```

## Monitoring & Observability

### 1. Metrics Collection

```go
// Custom metrics with Prometheus
requestCounter := prometheus.NewCounterVec(
    prometheus.CounterOpts{
        Name: "http_requests_total",
        Help: "Total HTTP requests",
    },
    []string{"method", "endpoint", "status"},
)

// Log with correlation to metrics
logger.Info().
    Str("metric_type", "http_request").
    Str("endpoint", endpoint).
    Int("status_code", status).
    Dur("duration", duration).
    Msg("Request metrics recorded")
```

### 2. Distributed Tracing

```go
// OpenTelemetry integration
ctx, span := tracer.Start(ctx, "database_query")
defer span.End()

// Correlation between logs and traces
logger.Info().
    Str("trace_id", span.SpanContext().TraceID().String()).
    Str("span_id", span.SpanContext().SpanID().String()).
    Str("operation", "user_lookup").
    Msg("Database operation traced")
```

### 3. Alerting Rules

```yaml
# Prometheus alerting rules
groups:
- name: bolt-app-alerts
  rules:
  - alert: HighErrorRate
    expr: rate(http_requests_total{status=~"5.."}[5m]) > 0.1
    for: 5m
    labels:
      severity: critical
    annotations:
      summary: "High error rate detected"
      
  - alert: ResponseTimeHigh
    expr: histogram_quantile(0.95, rate(http_request_duration_seconds_bucket[5m])) > 0.5
    for: 2m
    labels:
      severity: warning
```

### 4. Dashboards

```json
{
  "dashboard": {
    "title": "Bolt Application Metrics",
    "panels": [
      {
        "title": "Request Rate",
        "type": "graph",
        "targets": [
          {
            "expr": "rate(http_requests_total[5m])",
            "legendFormat": "{{method}} {{endpoint}}"
          }
        ]
      },
      {
        "title": "Error Rate",
        "type": "singlestat",
        "targets": [
          {
            "expr": "rate(http_requests_total{status=~\"5..\"}[5m]) / rate(http_requests_total[5m])"
          }
        ]
      }
    ]
  }
}
```

## High Availability

### 1. Load Balancing

```go
// Health-aware load balancer
type LoadBalancer struct {
    backends []Backend
    healthChecker *HealthChecker
}

func (lb *LoadBalancer) SelectBackend() *Backend {
    for _, backend := range lb.backends {
        if backend.IsHealthy() {
            logger.Debug().
                Str("backend_id", backend.ID).
                Str("backend_url", backend.URL).
                Msg("Selected healthy backend")
            return &backend
        }
    }
    return nil // No healthy backends
}
```

### 2. Circuit Breaker Pattern

```go
// Circuit breaker with logging
type CircuitBreaker struct {
    failures    int
    threshold   int
    state       State
    logger      bolt.Logger
}

func (cb *CircuitBreaker) Call(fn func() error) error {
    if cb.state == Open {
        cb.logger.Warn().
            Int("failures", cb.failures).
            Msg("Circuit breaker open - request blocked")
        return ErrCircuitOpen
    }
    
    err := fn()
    if err != nil {
        cb.failures++
        cb.logger.Error().
            Err(err).
            Int("failures", cb.failures).
            Msg("Circuit breaker recorded failure")
    }
    
    return err
}
```

### 3. Disaster Recovery

```yaml
# Multi-region deployment
apiVersion: v1
kind: Service
metadata:
  name: bolt-app-global
  annotations:
    external-dns.alpha.kubernetes.io/hostname: app.example.com
spec:
  type: ExternalName
  externalName: region1.example.com

---
# Backup configuration
apiVersion: batch/v1
kind: CronJob
metadata:
  name: database-backup
spec:
  schedule: "0 2 * * *"
  jobTemplate:
    spec:
      template:
        spec:
          containers:
          - name: backup
            image: postgres:15
            command:
            - pg_dump
            - $(DATABASE_URL)
            - --file=/backup/$(date +%Y%m%d_%H%M%S).sql
```

## Performance Optimization

### 1. Zero-Allocation Logging

```go
// Optimized logging patterns
logger.Info().
    Str("user_id", userID).        // No allocation
    Int("request_count", count).   // No allocation
    Dur("duration", duration).     // No allocation
    Msg("Request processed")       // Pre-allocated string
```

### 2. Resource Management

```yaml
# Kubernetes resource optimization
resources:
  requests:
    memory: "64Mi"    # Minimum needed
    cpu: "50m"        # 0.05 CPU cores
  limits:
    memory: "128Mi"   # Maximum allowed
    cpu: "100m"       # 0.1 CPU cores
```

### 3. Caching Strategies

```go
// Log with cache correlation
logger.Info().
    Str("cache_key", key).
    Str("cache_result", result).
    Dur("lookup_time", duration).
    Bool("cache_hit", hit).
    Msg("Cache operation completed")
```

### 4. Performance Monitoring

```bash
# Performance testing
make perf-test

# Benchmark specific components
go test -bench=BenchmarkZeroAllocation -benchmem

# Continuous profiling
go tool pprof http://localhost:8080/debug/pprof/profile
```

## Compliance & Auditing

### 1. Regulatory Compliance

```go
// GDPR/CCPA compliant logging
auditLogger.LogEvent(AuditEvent{
    EventType:     "DATA_ACCESS",
    UserID:        userID,
    Resource:      "PERSONAL_DATA",
    Action:        "READ",
    LegalBasis:    "CONSENT",
    DataSubject:   subjectID,
    ComplianceTag: "GDPR_ARTICLE_6",
})
```

### 2. SOX Compliance

```go
// Financial audit trail
auditLogger.LogEvent(AuditEvent{
    EventType:     "FINANCIAL_TRANSACTION",
    UserID:        userID,
    TransactionID: txnID,
    Amount:        amount,
    BeforeHash:    beforeHash,
    AfterHash:     afterHash,
    ComplianceTag: "SOX_CONTROLS",
})
```

### 3. HIPAA Compliance

```go
// Healthcare data access
auditLogger.LogEvent(AuditEvent{
    EventType:     "PHI_ACCESS",
    UserID:        providerID,
    PatientID:     patientID,
    DataType:      "MEDICAL_RECORD",
    Purpose:       "TREATMENT",
    ComplianceTag: "HIPAA_164_308",
})
```

## Troubleshooting

### 1. Common Issues

**High Memory Usage:**
```bash
# Check memory allocation patterns
go tool pprof heap http://localhost:8080/debug/pprof/heap

# Optimize logging frequency
export LOG_LEVEL=warn  # Reduce log volume
```

**Performance Degradation:**
```bash
# Profile CPU usage
go tool pprof http://localhost:8080/debug/pprof/profile

# Check allocation patterns
go test -bench=. -benchmem -cpuprofile=cpu.prof
```

**Lost Correlation IDs:**
```go
// Ensure propagation across services
ctx = context.WithValue(ctx, "correlation_id", correlationID)
req.Header.Set("X-Correlation-ID", correlationID)
```

### 2. Debug Commands

```bash
# Show system status
make status

# Run health checks
curl http://localhost:8080/health

# View live logs
make compose-logs

# Check Kubernetes pods
kubectl get pods -n bolt-examples

# Debug network issues
kubectl describe pod <pod-name> -n bolt-examples
```

### 3. Performance Tuning

```bash
# Optimize for high throughput
export GOMAXPROCS=4
export GOGC=100

# Tune logging levels
export LOG_LEVEL=warn      # Production
export LOG_LEVEL=debug     # Development

# Configure buffer sizes
export LOG_BUFFER_SIZE=8192
```

## Production Checklist

### Pre-Deployment

- [ ] Load testing completed (`make perf-test`)
- [ ] Security scanning passed
- [ ] Resource limits configured
- [ ] Health checks implemented
- [ ] Monitoring dashboards created
- [ ] Alerting rules configured
- [ ] Backup procedures tested
- [ ] Disaster recovery plan verified

### Post-Deployment

- [ ] Metrics collection verified
- [ ] Log aggregation working
- [ ] Distributed tracing active
- [ ] Alerting functional
- [ ] Performance within SLA
- [ ] Security policies enforced
- [ ] Compliance requirements met
- [ ] Documentation updated

## Support & Resources

- **Documentation**: [Bolt Documentation](https://felixgeelhaar.github.io/bolt/)
- **Examples Repository**: [GitHub Examples](https://github.com/felixgeelhaar/bolt/tree/main/examples)
- **Issues & Support**: [GitHub Issues](https://github.com/felixgeelhaar/bolt/issues)
- **Community Discussions**: [GitHub Discussions](https://github.com/felixgeelhaar/bolt/discussions)
- **Security Issues**: [Security Policy](https://github.com/felixgeelhaar/bolt/security/policy)

## License

All examples are provided under the same license as the Bolt project. See [LICENSE](../LICENSE) for details.