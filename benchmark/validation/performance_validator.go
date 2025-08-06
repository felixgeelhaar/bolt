// Package validation provides comprehensive performance validation and regression detection
package validation

import (
	"encoding/json"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"sort"
	"time"

	"github.com/felixgeelhaar/bolt/benchmark/framework"
)

// PerformanceThresholds defines acceptable performance limits
type PerformanceThresholds struct {
	// Latency thresholds (nanoseconds)
	MaxLatencyNs      float64 `json:"max_latency_ns"`
	MaxLatencyP95Ns   float64 `json:"max_latency_p95_ns"`
	MaxLatencyP99Ns   float64 `json:"max_latency_p99_ns"`
	
	// Allocation thresholds
	MaxAllocsPerOp    float64 `json:"max_allocs_per_op"`
	MaxBytesPerOp     float64 `json:"max_bytes_per_op"`
	
	// Throughput thresholds
	MinThroughputRPS  float64 `json:"min_throughput_rps"`
	
	// Regression thresholds (percentage)
	MaxLatencyRegression    float64 `json:"max_latency_regression"`
	MaxAllocationRegression float64 `json:"max_allocation_regression"`
	MinThroughputRegression float64 `json:"min_throughput_regression"`
	
	// Consistency thresholds
	MaxCoefficientOfVariation float64 `json:"max_coefficient_of_variation"`
	
	// Resource thresholds
	MaxMemoryGrowthMB float64 `json:"max_memory_growth_mb"`
	MaxGCPauseMs      float64 `json:"max_gc_pause_ms"`
}

// DefaultThresholds provides production-ready performance thresholds for Bolt
var DefaultThresholds = PerformanceThresholds{
	MaxLatencyNs:              100000,  // 100μs
	MaxLatencyP95Ns:           150000,  // 150μs
	MaxLatencyP99Ns:           200000,  // 200μs
	MaxAllocsPerOp:            0,       // Zero allocations
	MaxBytesPerOp:             0,       // Zero bytes allocated
	MinThroughputRPS:          10000000, // 10M ops/sec
	MaxLatencyRegression:      0.05,    // 5% regression threshold
	MaxAllocationRegression:   0.01,    // 1% allocation regression
	MinThroughputRegression:   0.95,    // 5% throughput regression
	MaxCoefficientOfVariation: 0.1,     // 10% CV
	MaxMemoryGrowthMB:         100,     // 100MB max growth
	MaxGCPauseMs:              10,      // 10ms max GC pause
}

// ValidationResult represents the outcome of performance validation
type ValidationResult struct {
	Timestamp      time.Time            `json:"timestamp"`
	TestSuite      string               `json:"test_suite"`
	OverallResult  ValidationStatus     `json:"overall_result"`
	ThresholdChecks []ThresholdCheck    `json:"threshold_checks"`
	RegressionChecks []RegressionCheck  `json:"regression_checks"`
	StatisticalAnalysis StatisticalAnalysis `json:"statistical_analysis"`
	QualityGates   QualityGateResult    `json:"quality_gates"`
	
	// Summary metrics
	TotalTests     int     `json:"total_tests"`
	PassedTests    int     `json:"passed_tests"`
	FailedTests    int     `json:"failed_tests"`
	WarningTests   int     `json:"warning_tests"`
	SuccessRate    float64 `json:"success_rate"`
	
	// Recommendations
	Recommendations []string `json:"recommendations"`
	CriticalIssues  []string `json:"critical_issues"`
	Warnings        []string `json:"warnings"`
}

type ValidationStatus string

const (
	ValidationPassed  ValidationStatus = "PASSED"
	ValidationFailed  ValidationStatus = "FAILED"
	ValidationWarning ValidationStatus = "WARNING"
)

type ThresholdCheck struct {
	Name           string           `json:"name"`
	TestScenario   string           `json:"test_scenario"`
	Library        string           `json:"library"`
	Metric         string           `json:"metric"`
	ActualValue    float64          `json:"actual_value"`
	ThresholdValue float64          `json:"threshold_value"`
	Status         ValidationStatus `json:"status"`
	Deviation      float64          `json:"deviation"`
	Message        string           `json:"message"`
}

type RegressionCheck struct {
	Name            string           `json:"name"`
	TestScenario    string           `json:"test_scenario"`
	Library         string           `json:"library"`
	Metric          string           `json:"metric"`
	CurrentValue    float64          `json:"current_value"`
	BaselineValue   float64          `json:"baseline_value"`
	ChangePercent   float64          `json:"change_percent"`
	Status          ValidationStatus `json:"status"`
	IsImprovement   bool             `json:"is_improvement"`
	Message         string           `json:"message"`
}

type StatisticalAnalysis struct {
	DataPoints              int     `json:"data_points"`
	Mean                    float64 `json:"mean"`
	Median                  float64 `json:"median"`
	StandardDeviation       float64 `json:"standard_deviation"`
	CoefficientOfVariation  float64 `json:"coefficient_of_variation"`
	Min                     float64 `json:"min"`
	Max                     float64 `json:"max"`
	P95                     float64 `json:"p95"`
	P99                     float64 `json:"p99"`
	OutlierCount            int     `json:"outlier_count"`
	IsStatisticallyStable   bool    `json:"is_statistically_stable"`
	ConfidenceLevel         float64 `json:"confidence_level"`
}

type QualityGateResult struct {
	ZeroAllocationsGate bool    `json:"zero_allocations_gate"`
	PerformanceGate     bool    `json:"performance_gate"`
	ConsistencyGate     bool    `json:"consistency_gate"`
	RegressionGate      bool    `json:"regression_gate"`
	ResourceGate        bool    `json:"resource_gate"`
	OverallGate         bool    `json:"overall_gate"`
	GateScore           float64 `json:"gate_score"`
}

// PerformanceValidator validates benchmark results against thresholds
type PerformanceValidator struct {
	thresholds PerformanceThresholds
	baselineData map[string]interface{}
}

// NewPerformanceValidator creates a new validator with default thresholds
func NewPerformanceValidator() *PerformanceValidator {
	return &PerformanceValidator{
		thresholds: DefaultThresholds,
		baselineData: make(map[string]interface{}),
	}
}

// WithThresholds configures custom thresholds
func (pv *PerformanceValidator) WithThresholds(thresholds PerformanceThresholds) *PerformanceValidator {
	pv.thresholds = thresholds
	return pv
}

// WithBaseline loads baseline data for regression detection
func (pv *PerformanceValidator) WithBaseline(baselinePath string) (*PerformanceValidator, error) {
	if baselinePath == "" || !fileExists(baselinePath) {
		return pv, nil // No baseline available
	}

	data, err := os.ReadFile(baselinePath)
	if err != nil {
		return pv, fmt.Errorf("failed to read baseline file: %w", err)
	}

	if err := json.Unmarshal(data, &pv.baselineData); err != nil {
		return pv, fmt.Errorf("failed to parse baseline data: %w", err)
	}

	return pv, nil
}

// ValidateResults performs comprehensive validation of benchmark results
func (pv *PerformanceValidator) ValidateResults(resultsPath string) (*ValidationResult, error) {
	// Load results data
	resultsData, err := pv.loadResultsData(resultsPath)
	if err != nil {
		return nil, fmt.Errorf("failed to load results: %w", err)
	}

	result := &ValidationResult{
		Timestamp:    time.Now(),
		TestSuite:    "Bolt Performance Validation",
		OverallResult: ValidationPassed,
		ThresholdChecks: make([]ThresholdCheck, 0),
		RegressionChecks: make([]RegressionCheck, 0),
		Recommendations: make([]string, 0),
		CriticalIssues:  make([]string, 0),
		Warnings:        make([]string, 0),
	}

	// Run threshold validations
	if err := pv.validateThresholds(resultsData, result); err != nil {
		return nil, fmt.Errorf("threshold validation failed: %w", err)
	}

	// Run regression detection
	if len(pv.baselineData) > 0 {
		if err := pv.detectRegressions(resultsData, result); err != nil {
			return nil, fmt.Errorf("regression detection failed: %w", err)
		}
	}

	// Perform statistical analysis
	if err := pv.performStatisticalAnalysis(resultsData, result); err != nil {
		return nil, fmt.Errorf("statistical analysis failed: %w", err)
	}

	// Evaluate quality gates
	pv.evaluateQualityGates(result)

	// Generate recommendations
	pv.generateRecommendations(result)

	// Calculate summary metrics
	pv.calculateSummaryMetrics(result)

	return result, nil
}

// loadResultsData loads and parses benchmark results
func (pv *PerformanceValidator) loadResultsData(resultsPath string) (map[string]interface{}, error) {
	var data map[string]interface{}

	// Try to load JSON data first
	jsonPath := filepath.Join(resultsPath, "performance_data.json")
	if fileExists(jsonPath) {
		fileData, err := os.ReadFile(jsonPath)
		if err != nil {
			return nil, err
		}
		if err := json.Unmarshal(fileData, &data); err != nil {
			return nil, err
		}
		return data, nil
	}

	return nil, fmt.Errorf("no performance data found at %s", resultsPath)
}

// validateThresholds checks all metrics against configured thresholds
func (pv *PerformanceValidator) validateThresholds(data map[string]interface{}, result *ValidationResult) error {
	// Extract summary performance data
	summary, ok := data["summary"].(map[string]interface{})
	if !ok {
		return fmt.Errorf("invalid summary data structure")
	}

	avgPerformance, ok := summary["avg_performance"].(map[string]interface{})
	if !ok {
		return fmt.Errorf("invalid avg_performance data structure")
	}

	// Validate each library's performance
	for library, perfData := range avgPerformance {
		libraryData, ok := perfData.(map[string]interface{})
		if !ok {
			continue
		}

		// Extract metrics
		avgNsPerOp, _ := libraryData["avg_ns_per_op"].(float64)
		avgAllocsPerOp, _ := libraryData["avg_allocs_per_op"].(float64)
		avgBytesPerOp, _ := libraryData["avg_bytes_per_op"].(float64)
		stdDevNsPerOp, _ := libraryData["stddev_ns_per_op"].(float64)

		// Validate latency thresholds
		result.ThresholdChecks = append(result.ThresholdChecks, ThresholdCheck{
			Name:           "Latency Threshold",
			TestScenario:   "Average",
			Library:        library,
			Metric:         "ns_per_op",
			ActualValue:    avgNsPerOp,
			ThresholdValue: pv.thresholds.MaxLatencyNs,
			Status:         pv.checkThreshold(avgNsPerOp, pv.thresholds.MaxLatencyNs, false),
			Deviation:      (avgNsPerOp - pv.thresholds.MaxLatencyNs) / pv.thresholds.MaxLatencyNs,
			Message:        pv.generateThresholdMessage("latency", avgNsPerOp, pv.thresholds.MaxLatencyNs, "ns/op", false),
		})

		// Validate allocation thresholds
		result.ThresholdChecks = append(result.ThresholdChecks, ThresholdCheck{
			Name:           "Zero Allocation Requirement",
			TestScenario:   "Average",
			Library:        library,
			Metric:         "allocs_per_op",
			ActualValue:    avgAllocsPerOp,
			ThresholdValue: pv.thresholds.MaxAllocsPerOp,
			Status:         pv.checkThreshold(avgAllocsPerOp, pv.thresholds.MaxAllocsPerOp, false),
			Deviation:      avgAllocsPerOp - pv.thresholds.MaxAllocsPerOp,
			Message:        pv.generateThresholdMessage("allocations", avgAllocsPerOp, pv.thresholds.MaxAllocsPerOp, "allocs/op", false),
		})

		// Validate bytes per operation
		result.ThresholdChecks = append(result.ThresholdChecks, ThresholdCheck{
			Name:           "Memory Allocation Threshold",
			TestScenario:   "Average",
			Library:        library,
			Metric:         "bytes_per_op",
			ActualValue:    avgBytesPerOp,
			ThresholdValue: pv.thresholds.MaxBytesPerOp,
			Status:         pv.checkThreshold(avgBytesPerOp, pv.thresholds.MaxBytesPerOp, false),
			Deviation:      (avgBytesPerOp - pv.thresholds.MaxBytesPerOp) / math.Max(pv.thresholds.MaxBytesPerOp, 1),
			Message:        pv.generateThresholdMessage("memory", avgBytesPerOp, pv.thresholds.MaxBytesPerOp, "bytes/op", false),
		})

		// Validate consistency (coefficient of variation)
		if avgNsPerOp > 0 {
			cv := stdDevNsPerOp / avgNsPerOp
			result.ThresholdChecks = append(result.ThresholdChecks, ThresholdCheck{
				Name:           "Performance Consistency",
				TestScenario:   "Average",
				Library:        library,
				Metric:         "coefficient_of_variation",
				ActualValue:    cv,
				ThresholdValue: pv.thresholds.MaxCoefficientOfVariation,
				Status:         pv.checkThreshold(cv, pv.thresholds.MaxCoefficientOfVariation, false),
				Deviation:      (cv - pv.thresholds.MaxCoefficientOfVariation) / pv.thresholds.MaxCoefficientOfVariation,
				Message:        pv.generateThresholdMessage("consistency", cv, pv.thresholds.MaxCoefficientOfVariation, "CV", false),
			})
		}
	}

	return nil
}

// detectRegressions compares current results with baseline to detect performance regressions
func (pv *PerformanceValidator) detectRegressions(data map[string]interface{}, result *ValidationResult) error {
	// Extract current summary data
	currentSummary, ok := data["summary"].(map[string]interface{})
	if !ok {
		return fmt.Errorf("invalid current summary data")
	}

	currentPerf, ok := currentSummary["avg_performance"].(map[string]interface{})
	if !ok {
		return fmt.Errorf("invalid current performance data")
	}

	// Extract baseline summary data
	baselineSummary, ok := pv.baselineData["summary"].(map[string]interface{})
	if !ok {
		return fmt.Errorf("invalid baseline summary data")
	}

	baselinePerf, ok := baselineSummary["avg_performance"].(map[string]interface{})
	if !ok {
		return fmt.Errorf("invalid baseline performance data")
	}

	// Compare metrics for each library
	for library := range currentPerf {
		currentLib, ok1 := currentPerf[library].(map[string]interface{})
		baselineLib, ok2 := baselinePerf[library].(map[string]interface{})
		
		if !ok1 || !ok2 {
			continue
		}

		// Compare latency
		currentLatency, _ := currentLib["avg_ns_per_op"].(float64)
		baselineLatency, _ := baselineLib["avg_ns_per_op"].(float64)
		
		if baselineLatency > 0 {
			latencyChange := (currentLatency - baselineLatency) / baselineLatency
			result.RegressionChecks = append(result.RegressionChecks, RegressionCheck{
				Name:          "Latency Regression Check",
				TestScenario:  "Average",
				Library:       library,
				Metric:        "ns_per_op",
				CurrentValue:  currentLatency,
				BaselineValue: baselineLatency,
				ChangePercent: latencyChange * 100,
				Status:        pv.checkRegression(latencyChange, pv.thresholds.MaxLatencyRegression, false),
				IsImprovement: latencyChange < 0,
				Message:       pv.generateRegressionMessage("latency", latencyChange, "ns/op"),
			})
		}

		// Compare allocations
		currentAllocs, _ := currentLib["avg_allocs_per_op"].(float64)
		baselineAllocs, _ := baselineLib["avg_allocs_per_op"].(float64)
		
		allocsChange := currentAllocs - baselineAllocs
		if baselineAllocs > 0 {
			allocsChange = allocsChange / baselineAllocs
		}
		
		result.RegressionChecks = append(result.RegressionChecks, RegressionCheck{
			Name:          "Allocation Regression Check",
			TestScenario:  "Average",
			Library:       library,
			Metric:        "allocs_per_op",
			CurrentValue:  currentAllocs,
			BaselineValue: baselineAllocs,
			ChangePercent: allocsChange * 100,
			Status:        pv.checkRegression(allocsChange, pv.thresholds.MaxAllocationRegression, false),
			IsImprovement: allocsChange < 0,
			Message:       pv.generateRegressionMessage("allocations", allocsChange, "allocs/op"),
		})
	}

	return nil
}

// performStatisticalAnalysis analyzes the statistical properties of the results
func (pv *PerformanceValidator) performStatisticalAnalysis(data map[string]interface{}, result *ValidationResult) error {
	// Extract raw results for statistical analysis
	rawResults, ok := data["raw_results"].(map[string]interface{})
	if !ok {
		result.StatisticalAnalysis = StatisticalAnalysis{
			DataPoints:            0,
			IsStatisticallyStable: false,
			ConfidenceLevel:       0,
		}
		return nil
	}

	// Analyze Bolt's performance (primary focus)
	var boltResults []interface{}
	for key, results := range rawResults {
		if contains(key, "Bolt") {
			if resultsArray, ok := results.([]interface{}); ok {
				boltResults = append(boltResults, resultsArray...)
			}
		}
	}

	if len(boltResults) == 0 {
		result.StatisticalAnalysis = StatisticalAnalysis{
			DataPoints:            0,
			IsStatisticallyStable: false,
			ConfidenceLevel:       0,
		}
		return nil
	}

	// Extract latency values for analysis
	var latencies []float64
	for _, res := range boltResults {
		if resMap, ok := res.(map[string]interface{}); ok {
			if nsPerOp, ok := resMap["ns_per_op"].(float64); ok {
				latencies = append(latencies, nsPerOp)
			}
		}
	}

	if len(latencies) < 2 {
		result.StatisticalAnalysis = StatisticalAnalysis{
			DataPoints:            len(latencies),
			IsStatisticallyStable: false,
			ConfidenceLevel:       0,
		}
		return nil
	}

	// Perform statistical calculations
	sort.Float64s(latencies)
	
	mean := calculateMean(latencies)
	median := calculatePercentile(latencies, 0.5)
	stdDev := calculateStandardDeviation(latencies, mean)
	cv := stdDev / mean
	min := latencies[0]
	max := latencies[len(latencies)-1]
	p95 := calculatePercentile(latencies, 0.95)
	p99 := calculatePercentile(latencies, 0.99)
	
	// Detect outliers using IQR method
	q1 := calculatePercentile(latencies, 0.25)
	q3 := calculatePercentile(latencies, 0.75)
	iqr := q3 - q1
	lowerBound := q1 - 1.5*iqr
	upperBound := q3 + 1.5*iqr
	
	outlierCount := 0
	for _, val := range latencies {
		if val < lowerBound || val > upperBound {
			outlierCount++
		}
	}

	// Assess statistical stability
	isStable := cv < pv.thresholds.MaxCoefficientOfVariation && 
		       outlierCount < len(latencies)/10 // Less than 10% outliers

	// Calculate confidence level (simplified)
	confidenceLevel := math.Max(0, 100*(1-cv)-float64(outlierCount)/float64(len(latencies))*10)

	result.StatisticalAnalysis = StatisticalAnalysis{
		DataPoints:             len(latencies),
		Mean:                   mean,
		Median:                 median,
		StandardDeviation:      stdDev,
		CoefficientOfVariation: cv,
		Min:                    min,
		Max:                    max,
		P95:                    p95,
		P99:                    p99,
		OutlierCount:           outlierCount,
		IsStatisticallyStable:  isStable,
		ConfidenceLevel:        confidenceLevel,
	}

	return nil
}

// evaluateQualityGates assesses overall quality gates
func (pv *PerformanceValidator) evaluateQualityGates(result *ValidationResult) {
	gates := QualityGateResult{}

	// Zero allocations gate
	gates.ZeroAllocationsGate = true
	for _, check := range result.ThresholdChecks {
		if check.Metric == "allocs_per_op" && check.Status != ValidationPassed {
			gates.ZeroAllocationsGate = false
			break
		}
	}

	// Performance gate
	gates.PerformanceGate = true
	for _, check := range result.ThresholdChecks {
		if check.Metric == "ns_per_op" && check.Status == ValidationFailed {
			gates.PerformanceGate = false
			break
		}
	}

	// Consistency gate
	gates.ConsistencyGate = result.StatisticalAnalysis.IsStatisticallyStable

	// Regression gate
	gates.RegressionGate = true
	for _, check := range result.RegressionChecks {
		if check.Status == ValidationFailed {
			gates.RegressionGate = false
			break
		}
	}

	// Resource gate (simplified)
	gates.ResourceGate = true // Would need more resource metrics

	// Overall gate
	gates.OverallGate = gates.ZeroAllocationsGate && 
		               gates.PerformanceGate && 
		               gates.ConsistencyGate && 
		               gates.RegressionGate && 
		               gates.ResourceGate

	// Gate score (0-100)
	score := 0.0
	if gates.ZeroAllocationsGate { score += 25 }
	if gates.PerformanceGate { score += 25 }
	if gates.ConsistencyGate { score += 20 }
	if gates.RegressionGate { score += 20 }
	if gates.ResourceGate { score += 10 }
	gates.GateScore = score

	result.QualityGates = gates
	
	// Update overall result based on quality gates
	if !gates.OverallGate {
		result.OverallResult = ValidationFailed
	}
}

// generateRecommendations provides actionable recommendations based on validation results
func (pv *PerformanceValidator) generateRecommendations(result *ValidationResult) {
	// Check for critical issues
	for _, check := range result.ThresholdChecks {
		if check.Status == ValidationFailed {
			if check.Metric == "allocs_per_op" && check.ActualValue > 0 {
				result.CriticalIssues = append(result.CriticalIssues, 
					fmt.Sprintf("CRITICAL: Zero-allocation requirement violated for %s (%.2f allocs/op)", 
						check.Library, check.ActualValue))
				result.Recommendations = append(result.Recommendations,
					"Review memory allocation patterns and implement object pooling")
			}
			
			if check.Metric == "ns_per_op" {
				result.CriticalIssues = append(result.CriticalIssues,
					fmt.Sprintf("CRITICAL: Performance threshold exceeded for %s (%.2f ns/op vs %.2f ns/op threshold)",
						check.Library, check.ActualValue, check.ThresholdValue))
				result.Recommendations = append(result.Recommendations,
					"Profile CPU usage and optimize hot paths")
			}
		}
	}

	// Check for regressions
	for _, check := range result.RegressionChecks {
		if check.Status == ValidationFailed && !check.IsImprovement {
			result.CriticalIssues = append(result.CriticalIssues,
				fmt.Sprintf("REGRESSION: %s performance degraded by %.1f%% (%s)",
					check.Library, check.ChangePercent, check.Metric))
			result.Recommendations = append(result.Recommendations,
				"Investigate recent code changes for performance impact")
		}
	}

	// Statistical analysis recommendations
	if !result.StatisticalAnalysis.IsStatisticallyStable {
		result.Warnings = append(result.Warnings, 
			"Performance results show high variability")
		result.Recommendations = append(result.Recommendations,
			"Increase benchmark iterations and ensure consistent test environment")
	}

	// Quality gate recommendations
	if !result.QualityGates.OverallGate {
		result.Recommendations = append(result.Recommendations,
			"Address quality gate failures before proceeding to production")
	}

	// Performance optimization recommendations
	if result.StatisticalAnalysis.CoefficientOfVariation > 0.05 {
		result.Recommendations = append(result.Recommendations,
			"Consider implementing performance optimizations to reduce latency variance")
	}
}

// calculateSummaryMetrics computes summary statistics for the validation
func (pv *PerformanceValidator) calculateSummaryMetrics(result *ValidationResult) {
	totalChecks := len(result.ThresholdChecks) + len(result.RegressionChecks)
	passedChecks := 0
	warningChecks := 0
	
	for _, check := range result.ThresholdChecks {
		if check.Status == ValidationPassed {
			passedChecks++
		} else if check.Status == ValidationWarning {
			warningChecks++
		}
	}
	
	for _, check := range result.RegressionChecks {
		if check.Status == ValidationPassed {
			passedChecks++
		} else if check.Status == ValidationWarning {
			warningChecks++
		}
	}

	result.TotalTests = totalChecks
	result.PassedTests = passedChecks
	result.WarningTests = warningChecks
	result.FailedTests = totalChecks - passedChecks - warningChecks
	
	if totalChecks > 0 {
		result.SuccessRate = float64(passedChecks) / float64(totalChecks)
	}
}

// Helper methods

func (pv *PerformanceValidator) checkThreshold(actual, threshold float64, higherIsBetter bool) ValidationStatus {
	if higherIsBetter {
		if actual >= threshold {
			return ValidationPassed
		} else if actual >= threshold*0.9 {
			return ValidationWarning
		}
		return ValidationFailed
	} else {
		if actual <= threshold {
			return ValidationPassed
		} else if actual <= threshold*1.1 {
			return ValidationWarning
		}
		return ValidationFailed
	}
}

func (pv *PerformanceValidator) checkRegression(change, threshold float64, higherIsBetter bool) ValidationStatus {
	absChange := math.Abs(change)
	
	if higherIsBetter {
		if change >= -threshold {
			return ValidationPassed
		} else if change >= -threshold*1.5 {
			return ValidationWarning
		}
		return ValidationFailed
	} else {
		if change <= threshold {
			return ValidationPassed
		} else if change <= threshold*1.5 {
			return ValidationWarning
		}
		return ValidationFailed
	}
}

func (pv *PerformanceValidator) generateThresholdMessage(metric string, actual, threshold float64, unit string, higherIsBetter bool) string {
	if higherIsBetter {
		if actual >= threshold {
			return fmt.Sprintf("✅ %s within threshold: %.2f %s (>= %.2f %s)", 
				metric, actual, unit, threshold, unit)
		} else {
			return fmt.Sprintf("❌ %s below threshold: %.2f %s (< %.2f %s)", 
				metric, actual, unit, threshold, unit)
		}
	} else {
		if actual <= threshold {
			return fmt.Sprintf("✅ %s within threshold: %.2f %s (<= %.2f %s)", 
				metric, actual, unit, threshold, unit)
		} else {
			return fmt.Sprintf("❌ %s exceeds threshold: %.2f %s (> %.2f %s)", 
				metric, actual, unit, threshold, unit)
		}
	}
}

func (pv *PerformanceValidator) generateRegressionMessage(metric string, change float64, unit string) string {
	if change < 0 {
		return fmt.Sprintf("🚀 %s improved by %.1f%% (%s)", 
			metric, math.Abs(change)*100, unit)
	} else if change > 0 {
		return fmt.Sprintf("⚠️ %s regressed by %.1f%% (%s)", 
			metric, change*100, unit)
	} else {
		return fmt.Sprintf("➡️ %s unchanged (%s)", metric, unit)
	}
}

// Utility functions

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && s[:len(substr)] == substr || 
		   len(s) >= len(substr) && s[len(s)-len(substr):] == substr
}

func calculateMean(values []float64) float64 {
	sum := 0.0
	for _, v := range values {
		sum += v
	}
	return sum / float64(len(values))
}

func calculateStandardDeviation(values []float64, mean float64) float64 {
	sumSquares := 0.0
	for _, v := range values {
		diff := v - mean
		sumSquares += diff * diff
	}
	variance := sumSquares / float64(len(values)-1)
	return math.Sqrt(variance)
}

func calculatePercentile(sortedValues []float64, p float64) float64 {
	if len(sortedValues) == 0 {
		return 0
	}
	if len(sortedValues) == 1 {
		return sortedValues[0]
	}
	
	index := p * float64(len(sortedValues)-1)
	lower := int(index)
	upper := lower + 1
	
	if upper >= len(sortedValues) {
		return sortedValues[len(sortedValues)-1]
	}
	
	weight := index - float64(lower)
	return sortedValues[lower]*(1-weight) + sortedValues[upper]*weight
}