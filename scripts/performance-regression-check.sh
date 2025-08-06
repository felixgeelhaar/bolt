#!/bin/bash
# Performance Regression Detection Script for Bolt Logging Library
# Automated performance monitoring and alerting for CI/CD pipelines

set -euo pipefail

# Configuration
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"
RESULTS_DIR="${PROJECT_ROOT}/performance-results"
BASELINE_DIR="${PROJECT_ROOT}/performance-baselines"

# Performance thresholds (configurable via environment)
MAX_LATENCY_NS="${MAX_LATENCY_NS:-100000}"        # 100μs SLA
MIN_THROUGHPUT_OPS="${MIN_THROUGHPUT_OPS:-10000000}" # 10M ops/sec
MAX_ALLOCATIONS="${MAX_ALLOCATIONS:-0}"            # Zero-allocation requirement
MAX_ERROR_RATE="${MAX_ERROR_RATE:-0.001}"          # 0.1% error rate
REGRESSION_THRESHOLD="${REGRESSION_THRESHOLD:-0.05}" # 5% regression tolerance

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Logging functions
log_info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

log_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

log_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# Create necessary directories
mkdir -p "$RESULTS_DIR" "$BASELINE_DIR"

# Performance test runner
run_performance_tests() {
    local test_name="$1"
    local output_file="$2"
    local count="${3:-10}"
    local benchtime="${4:-5s}"
    
    log_info "Running performance tests: $test_name"
    
    cd "$PROJECT_ROOT"
    
    # Run benchmarks with multiple iterations for statistical significance
    go test -bench=BenchmarkZeroAllocation \
        -benchmem \
        -count="$count" \
        -cpu=1,2,4 \
        -benchtime="$benchtime" \
        -timeout=30m \
        > "$output_file" 2>&1
    
    if [[ $? -ne 0 ]]; then
        log_error "Performance tests failed"
        cat "$output_file"
        return 1
    fi
    
    log_success "Performance tests completed: $output_file"
}

# Extract metrics from benchmark output
extract_metrics() {
    local benchmark_file="$1"
    local output_csv="$2"
    
    log_info "Extracting metrics from $benchmark_file"
    
    # Extract key metrics using benchstat for statistical analysis
    if command -v benchstat >/dev/null 2>&1; then
        benchstat "$benchmark_file" > "${benchmark_file%.txt}_stats.txt"
    fi
    
    # Parse benchmark results
    local latency_ns=0
    local allocations=0
    local memory_bytes=0
    local ops_count=0
    
    # Get the most recent benchmark result (last line with BenchmarkZeroAllocation)
    local bench_line
    bench_line=$(grep "BenchmarkZeroAllocation" "$benchmark_file" | tail -1)
    
    if [[ -n "$bench_line" ]]; then
        # Parse: BenchmarkZeroAllocation-4   15000000    62.96 ns/op    0 B/op    0 allocs/op
        ops_count=$(echo "$bench_line" | awk '{print $2}')
        latency_ns=$(echo "$bench_line" | awk '{print $3}' | sed 's/ns\/op//')
        memory_bytes=$(echo "$bench_line" | awk '{print $4}' | sed 's/B\/op//')
        allocations=$(echo "$bench_line" | awk '{print $6}' | sed 's/allocs\/op//')
        
        # Calculate throughput (ops/second)
        local throughput
        throughput=$(echo "scale=0; 1000000000 / $latency_ns" | bc -l)
        
        # Output CSV format
        echo "timestamp,latency_ns,allocations,memory_bytes,throughput_ops,ops_count" > "$output_csv"
        echo "$(date -u +%Y-%m-%dT%H:%M:%SZ),$latency_ns,$allocations,$memory_bytes,$throughput,$ops_count" >> "$output_csv"
        
        log_info "Extracted metrics:"
        log_info "  Latency: ${latency_ns}ns/op"
        log_info "  Allocations: ${allocations}allocs/op"
        log_info "  Memory: ${memory_bytes}B/op"
        log_info "  Throughput: ${throughput}ops/sec"
        log_info "  Operations: ${ops_count}"
    else
        log_error "Could not parse benchmark results from $benchmark_file"
        return 1
    fi
}

# Validate performance against SLA thresholds
validate_performance() {
    local metrics_csv="$1"
    local exit_code=0
    
    log_info "Validating performance against SLA thresholds"
    
    # Read metrics from CSV (skip header)
    local metrics
    metrics=$(tail -1 "$metrics_csv")
    
    local timestamp latency_ns allocations memory_bytes throughput_ops ops_count
    IFS=',' read -r timestamp latency_ns allocations memory_bytes throughput_ops ops_count <<< "$metrics"
    
    # Latency SLA validation
    if (( $(echo "$latency_ns > $MAX_LATENCY_NS" | bc -l) )); then
        log_error "❌ LATENCY SLA VIOLATION: ${latency_ns}ns exceeds maximum ${MAX_LATENCY_NS}ns"
        exit_code=1
    else
        log_success "✅ Latency within SLA: ${latency_ns}ns <= ${MAX_LATENCY_NS}ns"
    fi
    
    # Zero-allocation validation
    if (( $(echo "$allocations > $MAX_ALLOCATIONS" | bc -l) )); then
        log_error "❌ ZERO-ALLOCATION VIOLATION: ${allocations} allocations detected"
        exit_code=1
    else
        log_success "✅ Zero-allocation compliance: ${allocations} allocations"
    fi
    
    # Throughput validation
    if (( $(echo "$throughput_ops < $MIN_THROUGHPUT_OPS" | bc -l) )); then
        log_error "❌ THROUGHPUT SLA VIOLATION: ${throughput_ops}ops/sec below minimum ${MIN_THROUGHPUT_OPS}ops/sec"
        exit_code=1
    else
        log_success "✅ Throughput within SLA: ${throughput_ops}ops/sec >= ${MIN_THROUGHPUT_OPS}ops/sec"
    fi
    
    return $exit_code
}

# Compare against baseline performance
compare_with_baseline() {
    local current_csv="$1"
    local baseline_csv="$2"
    
    if [[ ! -f "$baseline_csv" ]]; then
        log_warning "No baseline found at $baseline_csv, creating new baseline"
        cp "$current_csv" "$baseline_csv"
        return 0
    fi
    
    log_info "Comparing performance against baseline: $baseline_csv"
    
    # Read current and baseline metrics
    local current_metrics baseline_metrics
    current_metrics=$(tail -1 "$current_csv")
    baseline_metrics=$(tail -1 "$baseline_csv")
    
    local curr_latency curr_throughput
    local base_latency base_throughput
    
    IFS=',' read -r _ curr_latency _ _ curr_throughput _ <<< "$current_metrics"
    IFS=',' read -r _ base_latency _ _ base_throughput _ <<< "$baseline_metrics"
    
    # Calculate regression percentages
    local latency_change throughput_change
    latency_change=$(echo "scale=4; ($curr_latency - $base_latency) / $base_latency * 100" | bc -l)
    throughput_change=$(echo "scale=4; ($curr_throughput - $base_throughput) / $base_throughput * 100" | bc -l)
    
    local regression_threshold_percent
    regression_threshold_percent=$(echo "$REGRESSION_THRESHOLD * 100" | bc -l)
    
    log_info "Performance comparison results:"
    log_info "  Latency change: ${latency_change}% (${curr_latency}ns vs ${base_latency}ns)"
    log_info "  Throughput change: ${throughput_change}% (${curr_throughput} vs ${base_throughput} ops/sec)"
    
    local exit_code=0
    
    # Check for significant regressions
    if (( $(echo "$latency_change > $regression_threshold_percent" | bc -l) )); then
        log_error "❌ PERFORMANCE REGRESSION: Latency increased by ${latency_change}% (threshold: ${regression_threshold_percent}%)"
        exit_code=1
    elif (( $(echo "$latency_change < -$regression_threshold_percent" | bc -l) )); then
        log_success "🚀 PERFORMANCE IMPROVEMENT: Latency decreased by ${latency_change#-}%"
    fi
    
    if (( $(echo "$throughput_change < -$regression_threshold_percent" | bc -l) )); then
        log_error "❌ PERFORMANCE REGRESSION: Throughput decreased by ${throughput_change#-}% (threshold: ${regression_threshold_percent}%)"
        exit_code=1
    elif (( $(echo "$throughput_change > $regression_threshold_percent" | bc -l) )); then
        log_success "🚀 PERFORMANCE IMPROVEMENT: Throughput increased by ${throughput_change}%"
    fi
    
    return $exit_code
}

# Generate performance report
generate_report() {
    local metrics_csv="$1"
    local baseline_csv="$2"
    local report_file="$3"
    
    log_info "Generating performance report: $report_file"
    
    cat > "$report_file" << EOF
# Bolt Logging Library Performance Report
Generated: $(date -u)
Commit: ${GITHUB_SHA:-$(git rev-parse HEAD 2>/dev/null || echo "unknown")}
Branch: ${GITHUB_REF_NAME:-$(git branch --show-current 2>/dev/null || echo "unknown")}

## Current Performance Metrics
$(cat "$metrics_csv")

## SLA Compliance
- Maximum Latency: ${MAX_LATENCY_NS}ns
- Minimum Throughput: ${MIN_THROUGHPUT_OPS}ops/sec
- Maximum Allocations: ${MAX_ALLOCATIONS}
- Maximum Error Rate: ${MAX_ERROR_RATE}%

## Performance Analysis
EOF
    
    if [[ -f "$baseline_csv" ]]; then
        local current_metrics baseline_metrics
        current_metrics=$(tail -1 "$metrics_csv")
        baseline_metrics=$(tail -1 "$baseline_csv")
        
        local curr_latency curr_throughput
        local base_latency base_throughput
        
        IFS=',' read -r _ curr_latency _ _ curr_throughput _ <<< "$current_metrics"
        IFS=',' read -r _ base_latency _ _ base_throughput _ <<< "$baseline_metrics"
        
        local latency_change throughput_change
        latency_change=$(echo "scale=2; ($curr_latency - $base_latency) / $base_latency * 100" | bc -l)
        throughput_change=$(echo "scale=2; ($curr_throughput - $base_throughput) / $base_throughput * 100" | bc -l)
        
        cat >> "$report_file" << EOF

### Baseline Comparison
- Current Latency: ${curr_latency}ns
- Baseline Latency: ${base_latency}ns
- Latency Change: ${latency_change}%

- Current Throughput: ${curr_throughput}ops/sec
- Baseline Throughput: ${base_throughput}ops/sec
- Throughput Change: ${throughput_change}%
EOF
    fi
    
    log_success "Report generated: $report_file"
}

# Memory profiling analysis
analyze_memory_profile() {
    local profile_file="$1"
    local analysis_file="$2"
    
    if [[ ! -f "$profile_file" ]]; then
        log_warning "Memory profile not found: $profile_file"
        return 0
    fi
    
    log_info "Analyzing memory profile: $profile_file"
    
    # Generate memory analysis
    go tool pprof -top "$profile_file" > "$analysis_file" 2>/dev/null
    
    # Check for unexpected allocations
    local heap_allocs
    heap_allocs=$(grep -o '[0-9.]*[KMGT]*B.*alloc_space' "$analysis_file" | head -1 | awk '{print $1}' || echo "0")
    
    if [[ "$heap_allocs" != "0" && "$heap_allocs" != "" ]]; then
        log_warning "Memory allocations detected in profile:"
        head -10 "$analysis_file"
        
        # Check if allocations exceed threshold
        local alloc_bytes
        case "$heap_allocs" in
            *KB) alloc_bytes=$(echo "${heap_allocs%KB} * 1024" | bc) ;;
            *MB) alloc_bytes=$(echo "${heap_allocs%MB} * 1024 * 1024" | bc) ;;
            *GB) alloc_bytes=$(echo "${heap_allocs%GB} * 1024 * 1024 * 1024" | bc) ;;
            *B) alloc_bytes=${heap_allocs%B} ;;
            *) alloc_bytes=0 ;;
        esac
        
        if (( alloc_bytes > 1000 )); then  # > 1KB threshold
            log_error "❌ Significant memory allocations detected: $heap_allocs"
            return 1
        fi
    else
        log_success "✅ No significant memory allocations detected"
    fi
    
    return 0
}

# CPU profiling analysis
analyze_cpu_profile() {
    local profile_file="$1"
    local analysis_file="$2"
    
    if [[ ! -f "$profile_file" ]]; then
        log_warning "CPU profile not found: $profile_file"
        return 0
    fi
    
    log_info "Analyzing CPU profile: $profile_file"
    
    # Generate CPU analysis
    go tool pprof -top "$profile_file" > "$analysis_file" 2>/dev/null
    
    # Look for hotspots
    local top_function_pct
    top_function_pct=$(head -4 "$analysis_file" | tail -1 | awk '{print $1}' | sed 's/%//' || echo "0")
    
    if (( $(echo "$top_function_pct > 50" | bc -l) )); then
        log_warning "CPU hotspot detected (${top_function_pct}% in single function):"
        head -10 "$analysis_file"
    fi
    
    log_success "CPU profile analysis completed"
    return 0
}

# Main execution function
main() {
    local test_name="${1:-bolt-performance-test}"
    local skip_baseline="${2:-false}"
    
    log_info "Starting Bolt performance regression check"
    log_info "Test: $test_name"
    log_info "Thresholds: Latency=${MAX_LATENCY_NS}ns, Throughput=${MIN_THROUGHPUT_OPS}ops/sec, Allocations=${MAX_ALLOCATIONS}"
    
    local timestamp
    timestamp=$(date +%Y%m%d_%H%M%S)
    local results_file="${RESULTS_DIR}/${test_name}_${timestamp}.txt"
    local metrics_file="${RESULTS_DIR}/${test_name}_${timestamp}.csv"
    local baseline_file="${BASELINE_DIR}/${test_name}_baseline.csv"
    local report_file="${RESULTS_DIR}/${test_name}_${timestamp}_report.md"
    
    # Run performance tests
    if ! run_performance_tests "$test_name" "$results_file" 10 "10s"; then
        log_error "Performance tests failed"
        return 1
    fi
    
    # Extract metrics
    if ! extract_metrics "$results_file" "$metrics_file"; then
        log_error "Failed to extract metrics"
        return 1
    fi
    
    # Validate against SLA
    local sla_exit_code=0
    if ! validate_performance "$metrics_file"; then
        sla_exit_code=1
    fi
    
    # Compare with baseline (unless skipped)
    local regression_exit_code=0
    if [[ "$skip_baseline" != "true" ]]; then
        if ! compare_with_baseline "$metrics_file" "$baseline_file"; then
            regression_exit_code=1
        fi
        
        # Update baseline if this is a good run and we're on main branch
        if [[ $sla_exit_code -eq 0 && $regression_exit_code -eq 0 ]]; then
            if [[ "${GITHUB_REF_NAME:-}" == "main" || "${CI:-}" != "true" ]]; then
                log_info "Updating baseline with current results"
                cp "$metrics_file" "$baseline_file"
            fi
        fi
    fi
    
    # Generate memory and CPU profiles if available
    local mem_profile="${PROJECT_ROOT}/mem.prof"
    local cpu_profile="${PROJECT_ROOT}/cpu.prof"
    local mem_analysis="${RESULTS_DIR}/${test_name}_${timestamp}_memory.txt"
    local cpu_analysis="${RESULTS_DIR}/${test_name}_${timestamp}_cpu.txt"
    
    # Run with profiling for detailed analysis
    log_info "Running profiling benchmarks..."
    cd "$PROJECT_ROOT"
    go test -bench=BenchmarkZeroAllocation -benchmem -memprofile="$mem_profile" -cpuprofile="$cpu_profile" -benchtime=5s > /dev/null 2>&1 || true
    
    analyze_memory_profile "$mem_profile" "$mem_analysis"
    analyze_cpu_profile "$cpu_profile" "$cpu_analysis"
    
    # Generate report
    generate_report "$metrics_file" "$baseline_file" "$report_file"
    
    # Summary
    log_info "Performance regression check completed"
    log_info "Results: $results_file"
    log_info "Metrics: $metrics_file"
    log_info "Report: $report_file"
    
    # Determine overall exit code
    local overall_exit_code=0
    if [[ $sla_exit_code -ne 0 ]]; then
        log_error "SLA validation failed"
        overall_exit_code=1
    fi
    if [[ $regression_exit_code -ne 0 ]]; then
        log_error "Performance regression detected"
        overall_exit_code=1
    fi
    
    if [[ $overall_exit_code -eq 0 ]]; then
        log_success "🎉 All performance checks passed!"
    else
        log_error "💥 Performance check failed!"
    fi
    
    return $overall_exit_code
}

# Script execution
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    main "$@"
fi