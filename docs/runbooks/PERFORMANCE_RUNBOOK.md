# Bolt Performance Incident Response Runbook

## Overview

This runbook provides step-by-step procedures for diagnosing and resolving performance issues in the Bolt logging library. It covers the most common performance scenarios and provides escalation paths for complex issues.

## Alert Classification

### Critical Performance Alerts (P0)
- `BoltLoggingLatencyHigh`: p95 latency > 100μs
- `BoltUnexpectedAllocations`: Memory allocations detected
- `BoltEventPoolExhaustion`: < 5 available events
- `BoltLoggingRateCritical`: > 500,000 events/sec

### High Priority Alerts (P1)
- `BoltLoggingLatencyWarning`: p95 latency > 50μs
- `BoltHandlerLatencyHigh`: Handler p99 > 1ms
- `BoltEventPoolWarning`: < 10 available events
- `BoltMemoryUsageHigh`: > 500MB RSS

## Quick Reference Dashboard URLs

```
Grafana Performance Overview: https://grafana.bolt-monitoring.local/d/bolt-performance
Prometheus Alerts: https://prometheus.bolt-monitoring.local/alerts
Jaeger Tracing: https://jaeger.bolt-monitoring.local
AlertManager: https://alertmanager.bolt-monitoring.local
```

## Incident Response Procedures

### 1. BoltLoggingLatencyHigh (P0)

#### Symptoms
- 95th percentile logging latency exceeds 100μs
- Application performance degradation
- Customer reports of slow response times

#### Immediate Actions (< 5 minutes)

1. **Acknowledge the Alert**
   ```bash
   # Check current latency
   curl -s "http://prometheus:9090/api/v1/query?query=histogram_quantile(0.95,%20rate(bolt_logging_duration_seconds_bucket[5m]))" | jq '.data.result[0].value[1]'
   ```

2. **Check System Resources**
   ```bash
   # CPU utilization
   kubectl top pods -n bolt-logging
   
   # Memory usage
   kubectl get pods -n bolt-logging -o custom-columns=NAME:.metadata.name,CPU:.status.containerStatuses[0].usage.cpu,MEMORY:.status.containerStatuses[0].usage.memory
   ```

3. **Verify Service Health**
   ```bash
   # Check pod status
   kubectl get pods -n bolt-logging -l app=bolt-app
   
   # Check recent restarts
   kubectl get events -n bolt-logging --sort-by='.lastTimestamp' | head -10
   ```

#### Investigation Steps

1. **Analyze Performance Metrics**
   ```bash
   # Query recent performance data
   curl -s "http://prometheus:9090/api/v1/query_range?query=histogram_quantile(0.95,%20rate(bolt_logging_duration_seconds_bucket[5m]))&start=$(date -d '1 hour ago' +%s)&end=$(date +%s)&step=60" | jq '.data.result[0].values'
   ```

2. **Check for Resource Contention**
   ```bash
   # Check node resources
   kubectl describe nodes | grep -A 5 "Allocated resources"
   
   # Check for CPU throttling
   kubectl exec -n bolt-logging deployment/bolt-app -- cat /sys/fs/cgroup/cpu/cpu.stat
   ```

3. **Examine Recent Changes**
   ```bash
   # Check deployment history
   kubectl rollout history deployment/bolt-app -n bolt-logging
   
   # Check recent configuration changes
   kubectl get configmap -n bolt-logging -o yaml | grep -A 5 -B 5 "last-applied"
   ```

#### Root Cause Analysis

1. **CPU Bound Issues**
   ```bash
   # Get CPU profile from running pod
   kubectl exec -n bolt-logging deployment/bolt-app -- curl -s http://localhost:6060/debug/pprof/profile?seconds=30 > cpu.prof
   
   # Analyze profile
   go tool pprof cpu.prof
   # Commands: top, list, web
   ```

2. **Memory Issues**
   ```bash
   # Get memory profile
   kubectl exec -n bolt-logging deployment/bolt-app -- curl -s http://localhost:6060/debug/pprof/heap > heap.prof
   
   # Analyze memory usage
   go tool pprof heap.prof
   ```

3. **Garbage Collection Issues**
   ```bash
   # Check GC metrics
   kubectl exec -n bolt-logging deployment/bolt-app -- curl -s http://localhost:9090/metrics | grep go_gc
   ```

#### Resolution Actions

1. **Scale Resources (Quick Fix)**
   ```bash
   # Increase CPU/memory limits
   kubectl patch deployment bolt-app -n bolt-logging -p '{"spec":{"template":{"spec":{"containers":[{"name":"bolt-app","resources":{"limits":{"cpu":"4000m","memory":"2Gi"}}}]}}}}'
   
   # Scale out replicas
   kubectl scale deployment bolt-app -n bolt-logging --replicas=6
   ```

2. **Optimize Configuration**
   ```bash
   # Update event pool size
   kubectl patch configmap bolt-app-config -n bolt-logging -p '{"data":{"config.yaml":"...\nevent_pool_size: 10000\n..."}}'
   
   # Restart to apply changes
   kubectl rollout restart deployment/bolt-app -n bolt-logging
   ```

3. **Network Optimization**
   ```bash
   # Check for network issues
   kubectl exec -n bolt-logging deployment/bolt-app -- netstat -i
   
   # Verify service endpoints
   kubectl get endpoints -n bolt-logging bolt-app-service
   ```

#### Post-Resolution

1. **Verify Resolution**
   ```bash
   # Monitor latency for 10 minutes
   watch -n 30 'curl -s "http://prometheus:9090/api/v1/query?query=histogram_quantile(0.95,%20rate(bolt_logging_duration_seconds_bucket[5m]))" | jq -r ".data.result[0].value[1]"'
   ```

2. **Document Changes**
   - Update incident log with root cause
   - Document configuration changes made
   - Update runbook if new patterns discovered

### 2. BoltUnexpectedAllocations (P0)

#### Symptoms
- Memory allocations detected in zero-allocation code path
- Potential memory leak or performance degradation
- GC pressure increase

#### Immediate Actions

1. **Identify Allocation Source**
   ```bash
   # Get detailed allocation profile
   kubectl exec -n bolt-logging deployment/bolt-app -- curl -s http://localhost:6060/debug/pprof/allocs > allocs.prof
   
   # Analyze allocation sources
   go tool pprof allocs.prof
   # Use: top, list main.*, traces
   ```

2. **Check Memory Growth**
   ```bash
   # Monitor memory usage trend
   curl -s "http://prometheus:9090/api/v1/query_range?query=process_resident_memory_bytes{job=\"bolt-app\"}&start=$(date -d '2 hours ago' +%s)&end=$(date +%s)&step=300" | jq '.data.result[0].values'
   ```

#### Investigation Steps

1. **Analyze Allocation Patterns**
   ```bash
   # Get goroutine dump for analysis
   kubectl exec -n bolt-logging deployment/bolt-app -- curl -s http://localhost:6060/debug/pprof/goroutine?debug=1 > goroutines.txt
   
   # Check for goroutine leaks
   grep -c "goroutine" goroutines.txt
   ```

2. **Review Recent Code Changes**
   ```bash
   # Check recent commits for allocation-prone changes
   git log --oneline --since="1 week ago" | grep -E "(string|slice|map|make|new)"
   ```

#### Resolution Actions

1. **Immediate Mitigation**
   ```bash
   # Force garbage collection
   kubectl exec -n bolt-logging deployment/bolt-app -- curl -X POST http://localhost:6060/debug/gc
   
   # Restart affected pods
   kubectl delete pods -n bolt-logging -l app=bolt-app
   ```

2. **Code Review**
   - Review recent changes for allocation sources
   - Use escape analysis: `go build -gcflags="-m" ./...`
   - Update code to eliminate allocations

### 3. BoltEventPoolExhaustion (P0)

#### Symptoms
- Available events < 5 in object pool
- High latency due to object creation
- Potential service degradation

#### Immediate Actions

1. **Check Pool Status**
   ```bash
   # Current pool metrics
   curl -s http://prometheus:9090/api/v1/query?query=bolt_event_pool_available_events | jq '.data.result[0].value[1]'
   curl -s http://prometheus:9090/api/v1/query?query=bolt_event_pool_total_events | jq '.data.result[0].value[1]'
   ```

2. **Analyze Pool Pressure**
   ```bash
   # Pool get/put ratio
   curl -s "http://prometheus:9090/api/v1/query?query=rate(bolt_event_pool_gets_total[5m]) / rate(bolt_event_pool_puts_total[5m])" | jq '.data.result[0].value[1]'
   ```

#### Resolution Actions

1. **Increase Pool Size**
   ```bash
   # Update configuration
   kubectl patch configmap bolt-app-config -n bolt-logging -p '{"data":{"BOLT_EVENT_POOL_SIZE":"20000"}}'
   kubectl rollout restart deployment/bolt-app -n bolt-logging
   ```

2. **Check for Leaks**
   ```bash
   # Monitor pool metrics after restart
   watch -n 10 'kubectl exec -n bolt-logging deployment/bolt-app -- curl -s http://localhost:9090/metrics | grep bolt_event_pool'
   ```

### 4. BoltMemoryUsageHigh (P1)

#### Symptoms
- RSS memory usage > 500MB
- Potential memory leak
- OOM risk

#### Investigation Steps

1. **Memory Breakdown Analysis**
   ```bash
   # Get memory stats
   kubectl exec -n bolt-logging deployment/bolt-app -- curl -s http://localhost:6060/debug/pprof/heap?debug=1 > heap.txt
   
   # Analyze heap usage
   grep -E "(HeapAlloc|HeapSys|HeapInuse)" heap.txt
   ```

2. **Check for Memory Leaks**
   ```bash
   # Take heap snapshots 10 minutes apart
   kubectl exec -n bolt-logging deployment/bolt-app -- curl -s http://localhost:6060/debug/pprof/heap > heap1.prof
   sleep 600
   kubectl exec -n bolt-logging deployment/bolt-app -- curl -s http://localhost:6060/debug/pprof/heap > heap2.prof
   
   # Compare profiles
   go tool pprof -base heap1.prof heap2.prof
   ```

#### Resolution Actions

1. **Tune GC Settings**
   ```bash
   # Adjust GC target percentage
   kubectl patch deployment bolt-app -n bolt-logging -p '{"spec":{"template":{"spec":{"containers":[{"name":"bolt-app","env":[{"name":"GOGC","value":"50"}]}]}}}}'
   ```

2. **Memory Limit Adjustment**
   ```bash
   # Increase memory limits if legitimate usage
   kubectl patch deployment bolt-app -n bolt-logging -p '{"spec":{"template":{"spec":{"containers":[{"name":"bolt-app","resources":{"limits":{"memory":"2Gi"}}}]}}}}'
   ```

## Escalation Procedures

### Level 1: On-Call Engineer (0-30 minutes)
- Follow immediate action steps
- Gather initial diagnostics
- Apply quick fixes if available
- Escalate if no resolution within 30 minutes

### Level 2: Senior Engineers (30-60 minutes)
- Deep performance analysis
- Code review for recent changes
- Advanced troubleshooting techniques
- Coordinate with development team if needed

### Level 3: Development Team (60+ minutes)
- Root cause analysis requiring code changes
- Performance optimization implementation
- Long-term architectural decisions
- Post-incident review coordination

### Emergency Escalation (Immediate)
- Multiple P0 alerts simultaneously
- Customer-impacting service outage
- Data integrity concerns
- Security implications

**Emergency Contacts:**
- Engineering Manager: +1-xxx-xxx-xxxx
- Senior Engineer: +1-xxx-xxx-xxxx
- Product Manager: +1-xxx-xxx-xxxx

## Tools and Resources

### Performance Analysis Tools

1. **Go Profiling**
   ```bash
   # CPU profile
   curl http://localhost:6060/debug/pprof/profile?seconds=30 > cpu.prof
   
   # Memory profile  
   curl http://localhost:6060/debug/pprof/heap > mem.prof
   
   # Goroutine profile
   curl http://localhost:6060/debug/pprof/goroutine > goroutine.prof
   
   # Block profile
   curl http://localhost:6060/debug/pprof/block > block.prof
   ```

2. **Kubernetes Debugging**
   ```bash
   # Resource usage
   kubectl top pods -n bolt-logging
   kubectl top nodes
   
   # Events
   kubectl get events -n bolt-logging --sort-by='.lastTimestamp'
   
   # Logs with timestamps
   kubectl logs -n bolt-logging deployment/bolt-app --timestamps=true --tail=100
   ```

3. **Performance Monitoring**
   ```bash
   # Prometheus queries
   curl "http://prometheus:9090/api/v1/query?query=METRIC_NAME"
   
   # Grafana API
   curl -H "Authorization: Bearer $GRAFANA_TOKEN" "http://grafana:3000/api/search"
   ```

### Useful Prometheus Queries

```prometheus
# Current latency p95
histogram_quantile(0.95, rate(bolt_logging_duration_seconds_bucket[5m]))

# Allocation rate
rate(bolt_allocation_count_total[5m])

# Throughput
rate(bolt_log_events_total[5m])

# Error rate
rate(bolt_logging_errors_total[5m]) / rate(bolt_log_events_total[5m])

# Memory usage trend
increase(process_resident_memory_bytes{job="bolt-app"}[1h])

# CPU usage
rate(process_cpu_seconds_total{job="bolt-app"}[5m])

# Pool utilization
(bolt_event_pool_total_events - bolt_event_pool_available_events) / bolt_event_pool_total_events
```

## Prevention and Monitoring

### Proactive Monitoring Setup

1. **Performance Dashboards**
   - Real-time latency monitoring
   - Resource utilization trends
   - Error rate tracking
   - Pool health monitoring

2. **Alerting Thresholds**
   - Latency: Warning at 50μs, Critical at 100μs
   - Allocations: Any allocation triggers alert
   - Memory: Warning at 70%, Critical at 90%
   - Pool: Warning at 20%, Critical at 10%

3. **Capacity Planning**
   - Weekly resource trend analysis
   - Growth projection modeling
   - Scaling trigger configuration
   - Resource optimization opportunities

### Performance Testing

1. **Load Testing Schedule**
   ```bash
   # Weekly performance regression tests
   ./scripts/performance-regression-check.sh
   
   # Monthly capacity testing
   k6 run scripts/load-test.js
   ```

2. **Canary Deployments**
   - Performance validation on small traffic percentage
   - Automatic rollback on SLA violation
   - Gradual traffic increase with monitoring

### Documentation Updates

1. **Incident Logs**
   - Root cause analysis
   - Resolution steps taken
   - Prevention measures implemented
   - Runbook improvements

2. **Performance Baselines**
   - Regular baseline updates
   - Performance regression tracking
   - Capacity planning data
   - Optimization opportunities

## Post-Incident Procedures

### Immediate Post-Resolution (< 2 hours)

1. **Verify Stability**
   ```bash
   # Monitor for 2 hours post-resolution
   watch -n 300 'echo "Latency: $(curl -s "http://prometheus:9090/api/v1/query?query=histogram_quantile(0.95,%20rate(bolt_logging_duration_seconds_bucket[5m]))" | jq -r ".data.result[0].value[1]")μs"'
   ```

2. **Document Actions Taken**
   - Commands executed
   - Configuration changes
   - Resource adjustments
   - Timeline of events

3. **Customer Communication**
   - Internal stakeholder update
   - Customer-facing status update if applicable
   - Expected resolution timeline
   - Prevention measures

### Post-Incident Review (< 1 week)

1. **Root Cause Analysis**
   - Technical root cause
   - Process failures
   - Monitoring gaps
   - Prevention opportunities

2. **Action Items**
   - Code improvements
   - Monitoring enhancements
   - Process updates
   - Training needs

3. **Runbook Updates**
   - New troubleshooting steps
   - Updated contact information
   - Tool improvements
   - Documentation gaps

### Continuous Improvement

1. **Monthly Performance Review**
   - SLA/SLO compliance analysis
   - Incident trend analysis
   - Performance optimization opportunities
   - Tool and process improvements

2. **Quarterly Architecture Review**
   - Performance architecture assessment
   - Scalability planning
   - Technology updates
   - Best practice adoption

## Contact Information

### On-Call Rotation
- Primary: Available 24/7 via PagerDuty
- Secondary: Backup escalation
- Manager: Business hours + emergency

### Specialized Teams
- **Performance Team**: Deep performance analysis
- **Infrastructure Team**: Kubernetes and infrastructure issues
- **Security Team**: Security-related performance issues
- **Product Team**: Business impact assessment

### External Resources
- **Cloud Provider Support**: For infrastructure-level issues
- **Monitoring Vendor**: For observability platform issues
- **Performance Consultants**: For complex optimization projects

---

**Document Version**: 2.0  
**Last Updated**: 2024-08-06  
**Next Review**: 2024-11-06  
**Owner**: Bolt Engineering Team