# Monitoring Integration Example

This example demonstrates integrating Bolt with popular monitoring platforms including Prometheus, Grafana, DataDog, and New Relic.

## Features

- **Prometheus Metrics**: Log metrics, request metrics, business metrics
- **Custom Dashboards**: Pre-configured Grafana dashboards
- **Alert Rules**: Production-ready alerting configurations
- **Business Metrics**: Track business KPIs alongside technical metrics
- **Cache Monitoring**: Hit rates and performance tracking
- **Queue Monitoring**: Depth tracking and alerting
- **Error Tracking**: Categorized error metrics

## Running the Example

```bash
cd examples/monitoring
go run main.go
```

The server will start on `http://localhost:8080` with metrics available at `http://localhost:8080/metrics`.

## Metrics Exposed

### Log Metrics

```
# Total log messages by level
bolt_logs_total{level="info"} 1523
bolt_logs_total{level="warn"} 42
bolt_logs_total{level="error"} 8

# Log operation latency
bolt_log_latency_seconds_bucket{level="info",le="0.000001"} 1200
bolt_log_latency_seconds_bucket{level="info",le="0.000002"} 1450
bolt_log_latency_seconds_sum{level="info"} 0.00152
bolt_log_latency_seconds_count{level="info"} 1523
```

### Application Metrics

```
# HTTP request metrics
app_requests_total{method="GET",path="/api/users",status="200"} 342
app_requests_total{method="GET",path="/api/error",status="500"} 5

# Request duration histogram
app_request_duration_seconds_bucket{method="GET",path="/api/users",le="0.05"} 320
app_request_duration_seconds_bucket{method="GET",path="/api/users",le="0.1"} 342
app_request_duration_seconds_sum{method="GET",path="/api/users"} 8.45
app_request_duration_seconds_count{method="GET",path="/api/users"} 342

# Active requests gauge
app_active_requests 3

# Error counters
app_errors_total{type="http_error",severity="error"} 5
app_errors_total{type="client_error",severity="warn"} 12
app_errors_total{type="payment_failure",severity="error"} 8
```

### Business Metrics

```
# Business event counters
app_business_events_total{event_type="order_placed",status="success"} 245
app_business_events_total{event_type="payment_processed",status="success"} 233
app_business_events_total{event_type="payment_processed",status="failed"} 12

# Queue depth gauge
app_queue_depth 23

# Cache metrics
app_cache_operations_total{result="hit"} 1456
app_cache_operations_total{result="miss"} 234
```

## Log Output Examples

### Application Startup

```json
{"level":"info","timestamp":"2025-01-03T14:30:00.123Z","service":"monitoring-example","version":"1.0.0","message":"starting monitoring integration example"}
{"level":"info","timestamp":"2025-01-03T14:30:00.145Z","addr":":8080","message":"http server started"}
```

### HTTP Request Logging

```json
{"level":"info","timestamp":"2025-01-03T14:30:15.234Z","method":"GET","path":"/api/users","remote_addr":"127.0.0.1:54321","message":"request started"}
{"level":"info","timestamp":"2025-01-03T14:30:15.256Z","method":"GET","path":"/api/users","status":200,"duration":"22ms","message":"request completed"}
```

### Business Event Logging

```json
{"level":"info","timestamp":"2025-01-03T14:30:20.345Z","event_type":"order_placed","order_id":"order_1704289820","user_id":"user_42","amount":245.67,"message":"order placed successfully"}
{"level":"info","timestamp":"2025-01-03T14:30:20.367Z","event_type":"payment_processed","payment_id":"pay_1704289820","amount":245.67,"success":true,"message":"payment processed"}
```

### Error Logging

```json
{"level":"error","timestamp":"2025-01-03T14:30:25.456Z","error":"simulated error","message":"error endpoint called"}
{"level":"error","timestamp":"2025-01-03T14:30:25.478Z","method":"GET","path":"/api/error","status":500,"duration":"22ms","message":"request completed"}
{"level":"error","timestamp":"2025-01-03T14:30:30.567Z","event_type":"payment_processed","payment_id":"pay_1704289830","amount":123.45,"success":false,"message":"payment processed"}
```

### Cache Metrics Logging

```json
{"level":"info","timestamp":"2025-01-03T14:30:45.678Z","cache_hits":456,"cache_misses":78,"hit_rate_pct":85.39,"message":"cache metrics"}
```

### Queue Monitoring

```json
{"level":"info","timestamp":"2025-01-03T14:30:50.789Z","queue_depth":15,"message":"queue status"}
{"level":"warn","timestamp":"2025-01-03T14:31:00.890Z","queue_depth":105,"message":"queue status"}
```

## Grafana Dashboard

### Dashboard JSON

Save this as `bolt-monitoring-dashboard.json`:

```json
{
  "dashboard": {
    "title": "Bolt Application Monitoring",
    "panels": [
      {
        "title": "Request Rate",
        "targets": [
          {
            "expr": "rate(app_requests_total[5m])",
            "legendFormat": "{{method}} {{path}} {{status}}"
          }
        ],
        "type": "graph"
      },
      {
        "title": "Request Duration (p95)",
        "targets": [
          {
            "expr": "histogram_quantile(0.95, rate(app_request_duration_seconds_bucket[5m]))",
            "legendFormat": "{{method}} {{path}}"
          }
        ],
        "type": "graph"
      },
      {
        "title": "Error Rate",
        "targets": [
          {
            "expr": "rate(app_errors_total[5m])",
            "legendFormat": "{{type}} ({{severity}})"
          }
        ],
        "type": "graph"
      },
      {
        "title": "Log Rate by Level",
        "targets": [
          {
            "expr": "rate(bolt_logs_total[1m])",
            "legendFormat": "{{level}}"
          }
        ],
        "type": "graph"
      },
      {
        "title": "Business Events",
        "targets": [
          {
            "expr": "rate(app_business_events_total[5m])",
            "legendFormat": "{{event_type}} ({{status}})"
          }
        ],
        "type": "graph"
      },
      {
        "title": "Cache Hit Rate",
        "targets": [
          {
            "expr": "rate(app_cache_operations_total{result=\"hit\"}[5m]) / rate(app_cache_operations_total[5m]) * 100",
            "legendFormat": "Hit Rate %"
          }
        ],
        "type": "graph"
      },
      {
        "title": "Queue Depth",
        "targets": [
          {
            "expr": "app_queue_depth",
            "legendFormat": "Queue Depth"
          }
        ],
        "type": "graph"
      },
      {
        "title": "Active Requests",
        "targets": [
          {
            "expr": "app_active_requests",
            "legendFormat": "Active Requests"
          }
        ],
        "type": "graph"
      }
    ]
  }
}
```

### Import Dashboard

```bash
# Import dashboard into Grafana
curl -X POST http://admin:admin@localhost:3000/api/dashboards/db \
  -H "Content-Type: application/json" \
  -d @bolt-monitoring-dashboard.json
```

## Prometheus Configuration

### prometheus.yml

```yaml
global:
  scrape_interval: 15s
  evaluation_interval: 15s

# Alert manager configuration
alerting:
  alertmanagers:
  - static_configs:
    - targets:
      - alertmanager:9093

# Load rules
rule_files:
  - "alerts.yml"

# Scrape configurations
scrape_configs:
  # Bolt application
  - job_name: 'bolt-app'
    static_configs:
    - targets: ['localhost:8080']
      labels:
        service: 'bolt-app'
        environment: 'production'

  # Prometheus self-monitoring
  - job_name: 'prometheus'
    static_configs:
    - targets: ['localhost:9090']
```

### alerts.yml

```yaml
groups:
- name: bolt_app_alerts
  interval: 30s
  rules:
  # High error rate
  - alert: HighErrorRate
    expr: |
      rate(app_errors_total[5m]) > 1
    for: 5m
    labels:
      severity: warning
    annotations:
      summary: "High error rate detected"
      description: "Error rate is {{ $value }} errors/sec for {{ $labels.type }}"

  # Critical error rate
  - alert: CriticalErrorRate
    expr: |
      rate(app_errors_total{severity="error"}[5m]) > 5
    for: 2m
    labels:
      severity: critical
    annotations:
      summary: "Critical error rate detected"
      description: "Critical error rate is {{ $value }} errors/sec"

  # High request latency
  - alert: HighRequestLatency
    expr: |
      histogram_quantile(0.95, rate(app_request_duration_seconds_bucket[5m])) > 1
    for: 10m
    labels:
      severity: warning
    annotations:
      summary: "High request latency (p95)"
      description: "95th percentile latency is {{ $value }}s for {{ $labels.method }} {{ $labels.path }}"

  # Payment failure rate
  - alert: HighPaymentFailureRate
    expr: |
      rate(app_business_events_total{event_type="payment_processed",status="failed"}[5m]) /
      rate(app_business_events_total{event_type="payment_processed"}[5m]) > 0.1
    for: 5m
    labels:
      severity: critical
    annotations:
      summary: "High payment failure rate"
      description: "Payment failure rate is {{ $value | humanizePercentage }}"

  # Low cache hit rate
  - alert: LowCacheHitRate
    expr: |
      rate(app_cache_operations_total{result="hit"}[5m]) /
      rate(app_cache_operations_total[5m]) < 0.5
    for: 15m
    labels:
      severity: warning
    annotations:
      summary: "Low cache hit rate"
      description: "Cache hit rate is {{ $value | humanizePercentage }}"

  # High queue depth
  - alert: HighQueueDepth
    expr: |
      app_queue_depth > 100
    for: 10m
    labels:
      severity: warning
    annotations:
      summary: "High queue depth"
      description: "Queue depth is {{ $value }} items"

  # Queue stuck
  - alert: QueueStuck
    expr: |
      delta(app_queue_depth[10m]) == 0 and app_queue_depth > 10
    for: 15m
    labels:
      severity: critical
    annotations:
      summary: "Queue appears stuck"
      description: "Queue depth unchanged at {{ $value }} for 15 minutes"

  # Service down
  - alert: ServiceDown
    expr: |
      up{job="bolt-app"} == 0
    for: 1m
    labels:
      severity: critical
    annotations:
      summary: "Bolt application is down"
      description: "Bolt application has been down for more than 1 minute"
```

## DataDog Integration

### DataDog Agent Configuration

```yaml
# datadog.yaml
logs_enabled: true
logs_config:
  container_collect_all: true
  processing_rules:
    - type: multi_line
      name: log_start_with_date
      pattern: \{"level"

# Custom metrics from logs
logs:
  - type: file
    path: /var/log/app/*.log
    service: bolt-app
    source: go
    log_processing_rules:
      - type: exclude_at_match
        name: exclude_health_checks
        pattern: /health
```

### Log-Based Metrics

Create custom metrics from logs in DataDog:

```json
{
  "name": "bolt.request.duration",
  "type": "distribution",
  "query": "@duration",
  "filter": "service:bolt-app",
  "group_by": ["@method", "@path", "@status"]
}
```

### DataDog Logger Integration

```go
import (
    "github.com/felixgeelhaar/bolt"
    "gopkg.in/DataDog/dd-trace-go.v1/ddtrace/tracer"
)

func main() {
    // Initialize DataDog tracer
    tracer.Start(
        tracer.WithService("bolt-app"),
        tracer.WithEnv("production"),
    )
    defer tracer.Stop()

    // Create logger with DataDog fields
    logger := bolt.New(bolt.NewJSONHandler(os.Stdout)).With().
        Str("dd.service", "bolt-app").
        Str("dd.env", "production").
        Str("dd.version", "1.0.0").
        Logger()

    // Logs will include DataDog trace correlation
    span, ctx := tracer.StartSpanFromContext(context.Background(), "request")
    defer span.Finish()

    // Extract trace IDs
    traceID := span.Context().TraceID()
    spanID := span.Context().SpanID()

    logger.Info().
        Uint64("dd.trace_id", traceID).
        Uint64("dd.span_id", spanID).
        Msg("processing request")
}
```

## New Relic Integration

### New Relic Configuration

```go
import (
    "github.com/felixgeelhaar/bolt"
    "github.com/newrelic/go-agent/v3/newrelic"
)

func main() {
    // Initialize New Relic
    app, err := newrelic.NewApplication(
        newrelic.ConfigAppName("Bolt App"),
        newrelic.ConfigLicense("YOUR_LICENSE_KEY"),
        newrelic.ConfigDistributedTracerEnabled(true),
    )
    if err != nil {
        panic(err)
    }

    logger := bolt.New(bolt.NewJSONHandler(os.Stdout))

    // Wrap HTTP handler with New Relic
    http.HandleFunc(newrelic.WrapHandleFunc(app, "/api/users", func(w http.ResponseWriter, r *http.Request) {
        txn := newrelic.FromContext(r.Context())

        // Log with transaction correlation
        logger.Info().
            Str("trace_id", txn.GetTraceMetadata().TraceID).
            Str("span_id", txn.GetTraceMetadata().SpanID).
            Msg("handling request")

        // ... handle request
    }))
}
```

### Custom New Relic Metrics

```go
// Record custom metrics
func recordCustomMetrics(app *newrelic.Application, metrics *MetricsCollector) {
    ticker := time.NewTicker(60 * time.Second)
    defer ticker.Stop()

    for range ticker.C {
        // Get metric values
        errorCount := // get from metrics
        requestCount := // get from metrics

        // Record in New Relic
        app.RecordCustomMetric("Custom/ErrorRate", errorCount/requestCount)
        app.RecordCustomMetric("Custom/QueueDepth", queueDepth)
        app.RecordCustomMetric("Custom/CacheHitRate", cacheHitRate)
    }
}
```

## Cloud Platform Integration

### AWS CloudWatch

```go
import (
    "github.com/aws/aws-sdk-go/aws"
    "github.com/aws/aws-sdk-go/service/cloudwatch"
)

// Send custom metrics to CloudWatch
func sendToCloudWatch(cw *cloudwatch.CloudWatch) {
    _, err := cw.PutMetricData(&cloudwatch.PutMetricDataInput{
        Namespace: aws.String("BoltApp"),
        MetricData: []*cloudwatch.MetricDatum{
            {
                MetricName: aws.String("RequestCount"),
                Value:      aws.Float64(requestCount),
                Unit:       aws.String("Count"),
                Dimensions: []*cloudwatch.Dimension{
                    {
                        Name:  aws.String("Service"),
                        Value: aws.String("bolt-app"),
                    },
                },
            },
        },
    })
}
```

### GCP Cloud Logging

```go
import (
    "cloud.google.com/go/logging"
)

// Create structured log entry for GCP
func logToGCP(client *logging.Client) {
    logger := client.Logger("bolt-app")

    logger.Log(logging.Entry{
        Severity: logging.Info,
        Payload: map[string]interface{}{
            "message":  "request completed",
            "duration": "25ms",
            "status":   200,
        },
        Labels: map[string]string{
            "service": "bolt-app",
        },
    })
}
```

### Azure Monitor

```go
import (
    "github.com/microsoft/ApplicationInsights-Go/appinsights"
)

// Track request with Application Insights
func trackRequest(client appinsights.TelemetryClient) {
    request := appinsights.NewRequestTelemetry(
        "GET",
        "/api/users",
        200,
        time.Millisecond*25,
    )

    request.Properties["service"] = "bolt-app"
    client.Track(request)
}
```

## Docker Compose for Monitoring Stack

```yaml
version: '3.8'

services:
  # Bolt application
  bolt-app:
    build: .
    ports:
      - "8080:8080"
    environment:
      - SERVICE_NAME=bolt-app
      - LOG_LEVEL=info

  # Prometheus
  prometheus:
    image: prom/prometheus:latest
    ports:
      - "9090:9090"
    volumes:
      - ./prometheus.yml:/etc/prometheus/prometheus.yml
      - ./alerts.yml:/etc/prometheus/alerts.yml
    command:
      - '--config.file=/etc/prometheus/prometheus.yml'
      - '--storage.tsdb.path=/prometheus'

  # Grafana
  grafana:
    image: grafana/grafana:latest
    ports:
      - "3000:3000"
    environment:
      - GF_SECURITY_ADMIN_PASSWORD=admin
    volumes:
      - grafana-data:/var/lib/grafana
      - ./grafana/provisioning:/etc/grafana/provisioning

  # Alert Manager
  alertmanager:
    image: prom/alertmanager:latest
    ports:
      - "9093:9093"
    volumes:
      - ./alertmanager.yml:/etc/alertmanager/alertmanager.yml

volumes:
  grafana-data:
```

## Best Practices

### 1. Metric Naming Conventions

```go
// Use consistent prefixes
const (
    MetricPrefixApp  = "app_"
    MetricPrefixBolt = "bolt_"
)

// Use descriptive names with units
"app_request_duration_seconds"  // Good
"app_req_time"                   // Bad

// Use labels for dimensions
app_requests_total{method="GET",path="/api/users",status="200"}  // Good
app_get_api_users_200_total                                       // Bad
```

### 2. Alert Design

- **Progressive severity**: Start with warnings, escalate to critical
- **Meaningful thresholds**: Base on actual production data
- **Action-oriented**: Alerts should require action
- **Reduce noise**: Use appropriate `for` durations

### 3. Dashboard Organization

- **Overview dashboard**: High-level health indicators
- **Service dashboard**: Service-specific metrics
- **Debug dashboard**: Detailed troubleshooting metrics
- **Business dashboard**: KPIs and business metrics

### 4. Log-Metric Correlation

```go
// Include correlation IDs in both logs and metrics
logger.Info().
    Str("request_id", requestID).
    Str("trace_id", traceID).
    Dur("duration", duration).
    Msg("request completed")

metrics.requestDuration.WithLabelValues(
    method,
    path,
).Observe(duration.Seconds())
```

## Testing

```bash
# Generate load
ab -n 1000 -c 10 http://localhost:8080/api/users

# Query metrics
curl http://localhost:8080/metrics

# Query Prometheus
curl 'http://localhost:9090/api/v1/query?query=rate(app_requests_total[5m])'

# Test alerts
curl 'http://localhost:9090/api/v1/alerts'
```

## See Also

- [Kubernetes Example](../kubernetes/) - K8s deployment with monitoring
- [REST API Example](../rest-api/) - HTTP middleware patterns
- [Batch Processor Example](../batch-processor/) - Background job monitoring
- [Bolt Documentation](../../README.md) - Main library documentation
