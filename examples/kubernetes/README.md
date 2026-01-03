# Kubernetes Deployment Example

This example demonstrates deploying a Bolt-based application to Kubernetes with production-grade logging, monitoring, and observability.

## Features

- **Production Deployment**: 3 replicas with rolling updates
- **Structured Logging**: JSON output for log aggregation
- **OpenTelemetry Integration**: Distributed tracing with automatic correlation
- **Health Checks**: Liveness, readiness, and startup probes
- **Resource Management**: CPU/memory requests and limits
- **Security**: Non-root user, read-only filesystem, RBAC
- **Observability**: Prometheus metrics, Fluent Bit log aggregation
- **High Availability**: Pod anti-affinity across nodes

## Prerequisites

- Kubernetes cluster (1.24+)
- kubectl configured
- OpenTelemetry Collector deployed (optional)
- Prometheus monitoring (optional)

## Quick Start

```bash
# Create namespace
kubectl create namespace production

# Deploy all resources
kubectl apply -f serviceaccount.yaml
kubectl apply -f configmap.yaml
kubectl apply -f deployment.yaml
kubectl apply -f service.yaml

# Verify deployment
kubectl -n production get pods
kubectl -n production get svc
```

## Architecture

### Components

```
┌─────────────────────────────────────────────────────┐
│                   Kubernetes Cluster                 │
│                                                       │
│  ┌──────────────────────────────────────────────┐  │
│  │           Namespace: production               │  │
│  │                                                │  │
│  │  ┌──────────────────────────────────────┐   │  │
│  │  │        Deployment (3 replicas)        │   │  │
│  │  │                                        │   │  │
│  │  │  Pod 1          Pod 2         Pod 3  │   │  │
│  │  │  ┌────────┐    ┌────────┐   ┌────────┐   │  │
│  │  │  │ App    │    │ App    │   │ App    │   │  │
│  │  │  │ Container   │ Container  │ Container   │  │
│  │  │  └────────┘    └────────┘   └────────┘   │  │
│  │  └──────────────────────────────────────┘   │  │
│  │                                                │  │
│  │  ┌──────────────────────────────────────┐   │  │
│  │  │         Service (ClusterIP)           │   │  │
│  │  │  - HTTP: 80 → 8080                   │   │  │
│  │  │  - gRPC: 9090 → 9090                 │   │  │
│  │  │  - Metrics: 8080 → 8080              │   │  │
│  │  └──────────────────────────────────────┘   │  │
│  │                                                │  │
│  │  ┌──────────────────────────────────────┐   │  │
│  │  │         ConfigMap                     │   │  │
│  │  │  - app.yaml (config)                 │   │  │
│  │  │  - fluent-bit.conf (logging)         │   │  │
│  │  └──────────────────────────────────────┘   │  │
│  │                                                │  │
│  │  ┌──────────────────────────────────────┐   │  │
│  │  │      ServiceAccount + RBAC            │   │  │
│  │  │  - ConfigMap read access              │   │  │
│  │  │  - Pod metadata access                │   │  │
│  │  └──────────────────────────────────────┘   │  │
│  └──────────────────────────────────────────────┘  │
│                                                       │
│  External Dependencies:                              │
│  - OpenTelemetry Collector (traces)                 │
│  - Prometheus (metrics)                              │
│  - Elasticsearch/Loki (logs)                         │
└─────────────────────────────────────────────────────┘
```

## Configuration

### Environment Variables

The deployment uses environment variables for configuration:

```yaml
# Logging configuration
LOG_FORMAT: "json"
LOG_LEVEL: "info"
SERVICE_NAME: "bolt-app"
SERVICE_VERSION: "v1.3.0"

# OpenTelemetry configuration
OTEL_EXPORTER_OTLP_ENDPOINT: "http://otel-collector:4317"
OTEL_SERVICE_NAME: "bolt-app"
OTEL_RESOURCE_ATTRIBUTES: "service.namespace=production,service.version=v1.3.0"

# Kubernetes metadata (auto-injected)
POD_NAME: metadata.name
POD_NAMESPACE: metadata.namespace
POD_IP: status.podIP
NODE_NAME: spec.nodeName
```

### ConfigMap

The ConfigMap provides additional configuration:

```yaml
# Application configuration
app.yaml:
  logging:
    format: json
    level: info
    buffer_size: 4096

  telemetry:
    enabled: true
    endpoint: http://otel-collector:4317
    sampling:
      rate: 0.1  # 10% sampling

# Fluent Bit configuration for log aggregation
fluent-bit.conf:
  - Tail application logs
  - Parse JSON format
  - Add Kubernetes metadata
  - Forward to aggregator
```

## Log Output Examples

### Startup Logs

```json
{"level":"info","timestamp":"2025-01-03T10:23:15.342Z","service":"bolt-app","version":"v1.3.0","pod_name":"bolt-app-7d4b9c8f5-xyz12","pod_namespace":"production","pod_ip":"10.244.1.15","node_name":"worker-1","message":"starting application"}
{"level":"info","timestamp":"2025-01-03T10:23:15.345Z","service":"bolt-app","version":"v1.3.0","pod_name":"bolt-app-7d4b9c8f5-xyz12","config_file":"/etc/config/app.yaml","message":"loading configuration"}
{"level":"info","timestamp":"2025-01-03T10:23:15.378Z","service":"bolt-app","version":"v1.3.0","pod_name":"bolt-app-7d4b9c8f5-xyz12","port":8080,"message":"http server started"}
```

### Request Logs with Tracing

```json
{"level":"info","timestamp":"2025-01-03T10:25:42.123Z","service":"bolt-app","version":"v1.3.0","trace_id":"4bf92f3577b34da6a3ce929d0e0e4736","span_id":"00f067aa0ba902b7","method":"GET","path":"/api/users/123","message":"request started"}
{"level":"info","timestamp":"2025-01-03T10:25:42.145Z","service":"bolt-app","version":"v1.3.0","trace_id":"4bf92f3577b34da6a3ce929d0e0e4736","span_id":"00f067aa0ba902b7","method":"GET","path":"/api/users/123","status":200,"duration":"22ms","message":"request completed"}
```

### Health Check Logs

```json
{"level":"debug","timestamp":"2025-01-03T10:26:00.001Z","service":"bolt-app","version":"v1.3.0","probe":"liveness","status":"healthy","message":"health check"}
{"level":"debug","timestamp":"2025-01-03T10:26:00.002Z","service":"bolt-app","version":"v1.3.0","probe":"readiness","status":"ready","message":"readiness check"}
```

## Log Aggregation

### Fluent Bit Sidecar

Add Fluent Bit as a sidecar container for log aggregation:

```yaml
spec:
  containers:
  - name: app
    # ... main app container

  - name: fluent-bit
    image: fluent/fluent-bit:2.0
    volumeMounts:
    - name: config
      mountPath: /fluent-bit/etc/
    - name: varlog
      mountPath: /var/log/app
      readOnly: true
    resources:
      requests:
        cpu: 50m
        memory: 64Mi
      limits:
        cpu: 100m
        memory: 128Mi

  volumes:
  - name: varlog
    emptyDir: {}
```

### Log Forwarding

Configure Fluent Bit to forward logs to your aggregation backend:

```conf
# For Elasticsearch
[OUTPUT]
    Name  es
    Match *
    Host  elasticsearch.logging.svc.cluster.local
    Port  9200
    Index bolt-app
    Type  _doc

# For Loki
[OUTPUT]
    Name  loki
    Match *
    Host  loki.logging.svc.cluster.local
    Port  3100
    Labels job=bolt-app
```

## OpenTelemetry Integration

### Collector Deployment

Deploy the OpenTelemetry Collector in your cluster:

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: otel-collector
  namespace: observability
spec:
  replicas: 1
  selector:
    matchLabels:
      app: otel-collector
  template:
    spec:
      containers:
      - name: otel-collector
        image: otel/opentelemetry-collector:0.88.0
        ports:
        - containerPort: 4317  # OTLP gRPC
        - containerPort: 4318  # OTLP HTTP
        volumeMounts:
        - name: config
          mountPath: /etc/otel
      volumes:
      - name: config
        configMap:
          name: otel-collector-config
```

### Trace Propagation

Bolt automatically propagates trace context from OpenTelemetry:

```go
import (
    "github.com/felixgeelhaar/bolt/v3"
    "go.opentelemetry.io/otel"
    "go.opentelemetry.io/otel/trace"
)

func handleRequest(w http.ResponseWriter, r *http.Request) {
    ctx := r.Context()

    // Bolt automatically extracts trace_id and span_id
    logger := bolt.Default().Ctx(ctx)

    logger.Info().
        Str("method", r.Method).
        Str("path", r.URL.Path).
        Msg("handling request")

    // Logs will include trace_id and span_id automatically
}
```

## Monitoring with Prometheus

### Service Monitor

Create a ServiceMonitor for Prometheus Operator:

```yaml
apiVersion: monitoring.coreos.com/v1
kind: ServiceMonitor
metadata:
  name: bolt-app
  namespace: production
  labels:
    app: bolt-app
spec:
  selector:
    matchLabels:
      app: bolt-app
  endpoints:
  - port: metrics
    interval: 30s
    path: /metrics
```

### Application Metrics

Expose application metrics in your code:

```go
import (
    "github.com/felixgeelhaar/bolt/v3"
    "github.com/prometheus/client_golang/prometheus"
    "github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
    requestCounter = prometheus.NewCounterVec(
        prometheus.CounterOpts{
            Name: "bolt_app_requests_total",
            Help: "Total number of requests",
        },
        []string{"method", "path", "status"},
    )

    requestDuration = prometheus.NewHistogramVec(
        prometheus.HistogramOpts{
            Name: "bolt_app_request_duration_seconds",
            Help: "Request duration in seconds",
            Buckets: prometheus.DefBuckets,
        },
        []string{"method", "path"},
    )
)

func init() {
    prometheus.MustRegister(requestCounter, requestDuration)
}

func loggingMiddleware(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        start := time.Now()
        wrapped := &responseWriter{ResponseWriter: w}

        logger.Info().
            Str("method", r.Method).
            Str("path", r.URL.Path).
            Msg("request started")

        next.ServeHTTP(wrapped, r)

        duration := time.Since(start)

        // Record metrics
        requestCounter.WithLabelValues(
            r.Method,
            r.URL.Path,
            strconv.Itoa(wrapped.statusCode),
        ).Inc()

        requestDuration.WithLabelValues(
            r.Method,
            r.URL.Path,
        ).Observe(duration.Seconds())

        // Log completion
        logger.Info().
            Str("method", r.Method).
            Str("path", r.URL.Path).
            Int("status", wrapped.statusCode).
            Dur("duration", duration).
            Msg("request completed")
    })
}

// Metrics endpoint
http.Handle("/metrics", promhttp.Handler())
```

## Troubleshooting

### View Pod Logs

```bash
# View logs from a specific pod
kubectl -n production logs bolt-app-7d4b9c8f5-xyz12

# Follow logs in real-time
kubectl -n production logs -f bolt-app-7d4b9c8f5-xyz12

# View logs from all pods
kubectl -n production logs -l app=bolt-app --all-containers=true

# View previous container logs (after crash)
kubectl -n production logs bolt-app-7d4b9c8f5-xyz12 --previous
```

### Check Pod Status

```bash
# Get pod status
kubectl -n production get pods -l app=bolt-app

# Describe pod for events
kubectl -n production describe pod bolt-app-7d4b9c8f5-xyz12

# Check resource usage
kubectl -n production top pod bolt-app-7d4b9c8f5-xyz12
```

### Debug Configuration

```bash
# View ConfigMap
kubectl -n production get configmap bolt-app-config -o yaml

# View environment variables
kubectl -n production exec bolt-app-7d4b9c8f5-xyz12 -- env | grep -E "(LOG_|OTEL_|POD_)"

# Test configuration parsing
kubectl -n production exec bolt-app-7d4b9c8f5-xyz12 -- cat /etc/config/app.yaml
```

### Health Check Issues

```bash
# Test health endpoints manually
kubectl -n production exec bolt-app-7d4b9c8f5-xyz12 -- curl http://localhost:8080/health
kubectl -n production exec bolt-app-7d4b9c8f5-xyz12 -- curl http://localhost:8080/ready

# Check probe configuration
kubectl -n production get pod bolt-app-7d4b9c8f5-xyz12 -o jsonpath='{.spec.containers[0].livenessProbe}'
kubectl -n production get pod bolt-app-7d4b9c8f5-xyz12 -o jsonpath='{.spec.containers[0].readinessProbe}'
```

### Network Issues

```bash
# Test service connectivity
kubectl -n production run debug --image=curlimages/curl -it --rm -- curl http://bolt-app

# Check service endpoints
kubectl -n production get endpoints bolt-app

# View service details
kubectl -n production describe svc bolt-app
```

### Common Issues

#### Pod CrashLoopBackOff

**Symptom**: Pods continuously restart

**Solution**:
```bash
# Check logs for errors
kubectl -n production logs bolt-app-7d4b9c8f5-xyz12 --previous

# Common causes:
# 1. Missing ConfigMap
kubectl -n production get configmap bolt-app-config

# 2. Invalid configuration
kubectl -n production exec bolt-app-7d4b9c8f5-xyz12 -- cat /etc/config/app.yaml

# 3. Resource limits too low
kubectl -n production describe pod bolt-app-7d4b9c8f5-xyz12 | grep -A 5 Resources
```

#### High Memory Usage

**Symptom**: Pods being OOMKilled

**Solution**:
```bash
# Check current memory usage
kubectl -n production top pod -l app=bolt-app

# Increase memory limits
kubectl -n production patch deployment bolt-app --patch '
spec:
  template:
    spec:
      containers:
      - name: app
        resources:
          limits:
            memory: 1Gi
'
```

#### Missing Trace Context

**Symptom**: Logs missing trace_id/span_id

**Solution**:
```bash
# Verify OpenTelemetry Collector is running
kubectl -n observability get pods -l app=otel-collector

# Check collector endpoint configuration
kubectl -n production get deployment bolt-app -o jsonpath='{.spec.template.spec.containers[0].env[?(@.name=="OTEL_EXPORTER_OTLP_ENDPOINT")].value}'

# Test collector connectivity
kubectl -n production exec bolt-app-7d4b9c8f5-xyz12 -- curl -v http://otel-collector.observability:4317
```

## Production Best Practices

### Resource Limits

Always set appropriate resource requests and limits:

```yaml
resources:
  requests:
    cpu: 100m      # Guaranteed CPU
    memory: 128Mi  # Guaranteed memory
  limits:
    cpu: 500m      # Maximum CPU
    memory: 512Mi  # Maximum memory
```

### Log Level Management

Use environment variables for dynamic log level control:

```yaml
env:
- name: LOG_LEVEL
  value: "info"  # Use "debug" for troubleshooting

# Or use ConfigMap for easy updates without restarts
- name: LOG_LEVEL
  valueFrom:
    configMapKeyRef:
      name: bolt-app-config
      key: log.level
```

### Security Hardening

```yaml
securityContext:
  # Run as non-root user
  runAsNonRoot: true
  runAsUser: 1000
  fsGroup: 1000

  # Prevent privilege escalation
  allowPrivilegeEscalation: false

  # Read-only root filesystem
  readOnlyRootFilesystem: true

  # Drop all capabilities
  capabilities:
    drop:
    - ALL
```

### High Availability

```yaml
# Pod anti-affinity for distribution
affinity:
  podAntiAffinity:
    requiredDuringSchedulingIgnoredDuringExecution:
    - labelSelector:
        matchExpressions:
        - key: app
          operator: In
          values:
          - bolt-app
      topologyKey: kubernetes.io/hostname

# Pod disruption budget
apiVersion: policy/v1
kind: PodDisruptionBudget
metadata:
  name: bolt-app-pdb
spec:
  minAvailable: 2
  selector:
    matchLabels:
      app: bolt-app
```

### Graceful Shutdown

Ensure clean shutdown for log flushing:

```go
func main() {
    logger := bolt.New(bolt.NewJSONHandler(os.Stdout))
    defer logger.Close()  // Flush buffers

    // Setup signal handling
    quit := make(chan os.Signal, 1)
    signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

    // ... run server

    <-quit
    logger.Info().Msg("received shutdown signal")

    // Graceful shutdown with timeout
    ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
    defer cancel()

    if err := server.Shutdown(ctx); err != nil {
        logger.Error().
            Str("error", err.Error()).
            Msg("shutdown error")
    }

    logger.Info().Msg("server stopped")
}
```

## Scaling

### Horizontal Pod Autoscaling

Create an HPA for automatic scaling:

```yaml
apiVersion: autoscaling/v2
kind: HorizontalPodAutoscaler
metadata:
  name: bolt-app-hpa
  namespace: production
spec:
  scaleTargetRef:
    apiVersion: apps/v1
    kind: Deployment
    name: bolt-app
  minReplicas: 3
  maxReplicas: 10
  metrics:
  - type: Resource
    resource:
      name: cpu
      target:
        type: Utilization
        averageUtilization: 70
  - type: Resource
    resource:
      name: memory
      target:
        type: Utilization
        averageUtilization: 80
  behavior:
    scaleDown:
      stabilizationWindowSeconds: 300
      policies:
      - type: Percent
        value: 50
        periodSeconds: 60
    scaleUp:
      stabilizationWindowSeconds: 60
      policies:
      - type: Percent
        value: 100
        periodSeconds: 30
```

### Manual Scaling

```bash
# Scale to 5 replicas
kubectl -n production scale deployment bolt-app --replicas=5

# Verify scaling
kubectl -n production get pods -l app=bolt-app -w
```

## Cleanup

```bash
# Delete all resources
kubectl delete -f service.yaml
kubectl delete -f deployment.yaml
kubectl delete -f configmap.yaml
kubectl delete -f serviceaccount.yaml

# Or delete namespace (if dedicated)
kubectl delete namespace production
```

## See Also

- [REST API Example](../rest-api/) - HTTP middleware patterns
- [gRPC Service Example](../grpc-service/) - gRPC interceptor patterns
- [Batch Processor Example](../batch-processor/) - Worker pool patterns
- [Bolt Documentation](../../README.md) - Main library documentation
