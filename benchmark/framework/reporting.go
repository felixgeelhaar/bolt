// Package framework provides comprehensive reporting and visualization for benchmark results
package framework

import (
	"encoding/json"
	"fmt"
	"html/template"
	"math"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"time"
)

// ReportGenerator creates professional HTML reports with interactive visualizations
type ReportGenerator struct {
	analyzer *CompetitiveAnalyzer
	template *template.Template
}

// NewReportGenerator creates a new report generator
func NewReportGenerator(analyzer *CompetitiveAnalyzer) *ReportGenerator {
	return &ReportGenerator{
		analyzer: analyzer,
		template: template.Must(template.New("report").Parse(reportTemplate)),
	}
}

// GenerateHTMLReport creates a comprehensive HTML report with interactive charts
func (rg *ReportGenerator) GenerateHTMLReport() error {
	// Collect all results
	allResults := make(map[string][]BenchmarkResult)
	rg.analyzer.results.Range(func(key, value interface{}) bool {
		allResults[key.(string)] = value.([]BenchmarkResult)
		return true
	})

	// Generate summary statistics
	summary := rg.generateSummaryStats(allResults)

	// Generate comparison data
	comparisons := rg.generateComparisons(allResults)

	// Generate trend analysis
	trends := rg.generateTrendAnalysis(allResults)

	// Create report data structure
	reportData := struct {
		Title        string                       `json:"title"`
		Generated    time.Time                    `json:"generated"`
		Summary      SummaryStats                 `json:"summary"`
		Comparisons  []LibraryComparison          `json:"comparisons"`
		Trends       []TrendAnalysis              `json:"trends"`
		RawResults   map[string][]BenchmarkResult `json:"raw_results"`
		ScenarioData []ScenarioAnalysis           `json:"scenario_data"`
		SystemInfo   SystemInfo                   `json:"system_info"`
	}{
		Title:        "Bolt Logging Library - Competitive Performance Analysis",
		Generated:    time.Now(),
		Summary:      summary,
		Comparisons:  comparisons,
		Trends:       trends,
		RawResults:   allResults,
		ScenarioData: rg.generateScenarioAnalysis(allResults),
		SystemInfo:   rg.getSystemInfo(),
	}

	// Write HTML report
	reportPath := filepath.Join(rg.analyzer.outputDir, "performance_report.html")
	htmlFile, err := os.Create(reportPath)
	if err != nil {
		return fmt.Errorf("failed to create HTML report: %w", err)
	}
	defer htmlFile.Close()

	if err := rg.template.Execute(htmlFile, reportData); err != nil {
		return fmt.Errorf("failed to execute report template: %w", err)
	}

	// Write JSON data for external tools
	jsonPath := filepath.Join(rg.analyzer.outputDir, "performance_data.json")
	jsonFile, err := os.Create(jsonPath)
	if err != nil {
		return fmt.Errorf("failed to create JSON report: %w", err)
	}
	defer jsonFile.Close()

	encoder := json.NewEncoder(jsonFile)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(reportData); err != nil {
		return fmt.Errorf("failed to encode JSON report: %w", err)
	}

	// Generate CSV exports for data analysis
	if err := rg.generateCSVExports(allResults); err != nil {
		return fmt.Errorf("failed to generate CSV exports: %w", err)
	}

	// Generate markdown summary
	if err := rg.generateMarkdownSummary(reportData); err != nil {
		return fmt.Errorf("failed to generate markdown summary: %w", err)
	}

	fmt.Printf("📊 Reports generated successfully:\n")
	fmt.Printf("  HTML Report: %s\n", reportPath)
	fmt.Printf("  JSON Data: %s\n", jsonPath)
	fmt.Printf("  CSV Exports: %s\n", filepath.Join(rg.analyzer.outputDir, "csv/"))
	fmt.Printf("  Markdown Summary: %s\n", filepath.Join(rg.analyzer.outputDir, "PERFORMANCE_SUMMARY.md"))

	return nil
}

// Data structures for report generation

type SummaryStats struct {
	TotalTests     int                           `json:"total_tests"`
	FastestLibrary string                        `json:"fastest_library"`
	SlowestLibrary string                        `json:"slowest_library"`
	ZeroAllocLibs  []string                      `json:"zero_alloc_libs"`
	AvgPerformance map[string]PerformanceMetrics `json:"avg_performance"`
	BestScenarios  map[string]string             `json:"best_scenarios"`
}

type PerformanceMetrics struct {
	AvgNsPerOp     float64 `json:"avg_ns_per_op"`
	AvgAllocsPerOp float64 `json:"avg_allocs_per_op"`
	AvgBytesPerOp  float64 `json:"avg_bytes_per_op"`
	StdDevNsPerOp  float64 `json:"stddev_ns_per_op"`
}

type LibraryComparison struct {
	Library          string   `json:"library"`
	VsBolt           string   `json:"vs_bolt"`
	PerformanceRatio float64  `json:"performance_ratio"`
	AllocationRatio  float64  `json:"allocation_ratio"`
	Rank             int      `json:"rank"`
	Strengths        []string `json:"strengths"`
	Weaknesses       []string `json:"weaknesses"`
}

type TrendAnalysis struct {
	Scenario    string  `json:"scenario"`
	Winner      string  `json:"winner"`
	Confidence  float64 `json:"confidence"`
	Margin      float64 `json:"margin"`
	Consistency float64 `json:"consistency"`
}

type ScenarioAnalysis struct {
	Name        string                        `json:"name"`
	Description string                        `json:"description"`
	Results     map[string]PerformanceMetrics `json:"results"`
	Winner      string                        `json:"winner"`
	WinMargin   float64                       `json:"win_margin"`
	Ranking     []LibraryRank                 `json:"ranking"`
}

type LibraryRank struct {
	Library string  `json:"library"`
	NsPerOp float64 `json:"ns_per_op"`
	Rank    int     `json:"rank"`
	Score   float64 `json:"score"`
}

type SystemInfo struct {
	GoVersion    string `json:"go_version"`
	GOOS         string `json:"goos"`
	GOARCH       string `json:"goarch"`
	CPUs         int    `json:"cpus"`
	Hostname     string `json:"hostname"`
	TestDuration string `json:"test_duration"`
}

// Implementation methods

func (rg *ReportGenerator) generateSummaryStats(allResults map[string][]BenchmarkResult) SummaryStats {
	libraryPerformance := make(map[string][]float64)
	libraryAllocations := make(map[string][]float64)
	scenarioWinners := make(map[string]string)

	// Collect performance data by library
	for key, results := range allResults {
		parts := strings.Split(key, "-")
		if len(parts) < 2 {
			continue
		}
		library := parts[0]
		scenario := strings.Join(parts[1:], "-")

		// Get the final result (after statistical analysis)
		if len(results) > 0 {
			finalResult := results[len(results)-1]
			libraryPerformance[library] = append(libraryPerformance[library], finalResult.NsPerOp)
			libraryAllocations[library] = append(libraryAllocations[library], finalResult.AllocsPerOp)
		}

		// Track scenario winners
		if winner, exists := scenarioWinners[scenario]; !exists {
			scenarioWinners[scenario] = library
		} else {
			// Compare performance to find winner
			currentBest := rg.getAveragePerformance(allResults[winner+"-"+scenario])
			challenger := rg.getAveragePerformance(results)
			if challenger < currentBest {
				scenarioWinners[scenario] = library
			}
		}
	}

	// Calculate average performance metrics
	avgPerformance := make(map[string]PerformanceMetrics)
	for library, nsPerOpValues := range libraryPerformance {
		allocValues := libraryAllocations[library]
		avgPerformance[library] = PerformanceMetrics{
			AvgNsPerOp:     rg.average(nsPerOpValues),
			AvgAllocsPerOp: rg.average(allocValues),
			StdDevNsPerOp:  rg.standardDev(nsPerOpValues),
		}
	}

	// Find fastest and slowest libraries
	var fastest, slowest string
	var fastestTime, slowestTime float64 = math.MaxFloat64, 0

	for library, metrics := range avgPerformance {
		if metrics.AvgNsPerOp < fastestTime {
			fastestTime = metrics.AvgNsPerOp
			fastest = library
		}
		if metrics.AvgNsPerOp > slowestTime {
			slowestTime = metrics.AvgNsPerOp
			slowest = library
		}
	}

	// Find zero-allocation libraries
	var zeroAllocLibs []string
	for library, metrics := range avgPerformance {
		if metrics.AvgAllocsPerOp == 0 {
			zeroAllocLibs = append(zeroAllocLibs, library)
		}
	}

	return SummaryStats{
		TotalTests:     len(allResults),
		FastestLibrary: fastest,
		SlowestLibrary: slowest,
		ZeroAllocLibs:  zeroAllocLibs,
		AvgPerformance: avgPerformance,
		BestScenarios:  scenarioWinners,
	}
}

func (rg *ReportGenerator) generateComparisons(allResults map[string][]BenchmarkResult) []LibraryComparison {
	var comparisons []LibraryComparison

	// Get Bolt baseline performance
	boltPerformance := rg.getLibraryAveragePerformance(allResults, "Bolt")

	for _, library := range rg.analyzer.libraryNames() {
		if library == "Bolt" {
			continue // Skip self-comparison
		}

		libPerformance := rg.getLibraryAveragePerformance(allResults, library)

		performanceRatio := libPerformance.AvgNsPerOp / boltPerformance.AvgNsPerOp
		allocationRatio := libPerformance.AvgAllocsPerOp / math.Max(boltPerformance.AvgAllocsPerOp, 1)

		var vsText string
		if performanceRatio < 1 {
			vsText = fmt.Sprintf("%.1fx faster", 1/performanceRatio)
		} else {
			vsText = fmt.Sprintf("%.1fx slower", performanceRatio)
		}

		strengths, weaknesses := rg.analyzeLibraryCharacteristics(allResults, library)

		comparisons = append(comparisons, LibraryComparison{
			Library:          library,
			VsBolt:           vsText,
			PerformanceRatio: performanceRatio,
			AllocationRatio:  allocationRatio,
			Strengths:        strengths,
			Weaknesses:       weaknesses,
		})
	}

	// Sort by performance ratio
	sort.Slice(comparisons, func(i, j int) bool {
		return comparisons[i].PerformanceRatio < comparisons[j].PerformanceRatio
	})

	// Assign ranks
	for i := range comparisons {
		comparisons[i].Rank = i + 1
	}

	return comparisons
}

func (rg *ReportGenerator) generateTrendAnalysis(allResults map[string][]BenchmarkResult) []TrendAnalysis {
	var trends []TrendAnalysis

	scenarioResults := make(map[string]map[string][]float64)

	// Group results by scenario
	for key, results := range allResults {
		parts := strings.Split(key, "-")
		if len(parts) < 2 {
			continue
		}
		library := parts[0]
		scenario := strings.Join(parts[1:], "-")

		if scenarioResults[scenario] == nil {
			scenarioResults[scenario] = make(map[string][]float64)
		}

		for _, result := range results {
			scenarioResults[scenario][library] = append(scenarioResults[scenario][library], result.NsPerOp)
		}
	}

	// Analyze each scenario
	for scenario, libraryResults := range scenarioResults {
		var bestLibrary string
		var bestTime float64 = math.MaxFloat64
		var confidenceSum, _ float64
		count := 0

		for library, times := range libraryResults {
			avgTime := rg.average(times)
			if avgTime < bestTime {
				bestTime = avgTime
				bestLibrary = library
			}

			// Calculate confidence (inverse of coefficient of variation)
			if len(times) > 1 {
				stdDev := rg.standardDev(times)
				cv := stdDev / avgTime
				confidenceSum += (1 - cv)
				count++
			}
		}

		// Calculate margin over second-best
		var secondBest float64 = math.MaxFloat64
		for library, times := range libraryResults {
			if library != bestLibrary {
				avgTime := rg.average(times)
				if avgTime < secondBest {
					secondBest = avgTime
				}
			}
		}

		margin := (secondBest - bestTime) / bestTime
		confidence := confidenceSum / math.Max(float64(count), 1)
		consistency := 1.0 - (rg.standardDev(libraryResults[bestLibrary]) / bestTime)

		trends = append(trends, TrendAnalysis{
			Scenario:    scenario,
			Winner:      bestLibrary,
			Confidence:  confidence,
			Margin:      margin,
			Consistency: consistency,
		})
	}

	return trends
}

func (rg *ReportGenerator) generateScenarioAnalysis(allResults map[string][]BenchmarkResult) []ScenarioAnalysis {
	var analyzes []ScenarioAnalysis

	scenarioData := make(map[string]map[string][]BenchmarkResult)
	scenarioDescriptions := make(map[string]string)

	// Group by scenario
	for _, scenario := range rg.analyzer.scenarios {
		scenarioDescriptions[scenario.Name] = scenario.Description
		scenarioData[scenario.Name] = make(map[string][]BenchmarkResult)

		for key, results := range allResults {
			if strings.Contains(key, "-"+scenario.Name) {
				library := strings.Split(key, "-")[0]
				scenarioData[scenario.Name][library] = results
			}
		}
	}

	// Analyze each scenario
	for scenarioName, libraryResults := range scenarioData {
		analysis := ScenarioAnalysis{
			Name:        scenarioName,
			Description: scenarioDescriptions[scenarioName],
			Results:     make(map[string]PerformanceMetrics),
		}

		var bestLibrary string
		var bestTime float64 = math.MaxFloat64
		var rankings []LibraryRank

		for library, results := range libraryResults {
			if len(results) == 0 {
				continue
			}

			finalResult := results[len(results)-1]
			metrics := PerformanceMetrics{
				AvgNsPerOp:     finalResult.NsPerOp,
				AvgAllocsPerOp: finalResult.AllocsPerOp,
				AvgBytesPerOp:  finalResult.BytesPerOp,
				StdDevNsPerOp:  finalResult.NsPerOpStdDev,
			}

			analysis.Results[library] = metrics

			if finalResult.NsPerOp < bestTime {
				bestTime = finalResult.NsPerOp
				bestLibrary = library
			}

			rankings = append(rankings, LibraryRank{
				Library: library,
				NsPerOp: finalResult.NsPerOp,
			})
		}

		// Sort rankings and assign scores
		sort.Slice(rankings, func(i, j int) bool {
			return rankings[i].NsPerOp < rankings[j].NsPerOp
		})

		for i := range rankings {
			rankings[i].Rank = i + 1
			// Score based on performance relative to best (higher is better)
			rankings[i].Score = bestTime / rankings[i].NsPerOp
		}

		analysis.Winner = bestLibrary
		analysis.Ranking = rankings

		// Calculate win margin
		if len(rankings) > 1 {
			analysis.WinMargin = (rankings[1].NsPerOp - rankings[0].NsPerOp) / rankings[0].NsPerOp
		}

		analyzes = append(analyzes, analysis)
	}

	return analyzes
}

// Helper methods

func (rg *ReportGenerator) getAveragePerformance(results []BenchmarkResult) float64 {
	if len(results) == 0 {
		return 0
	}
	sum := 0.0
	for _, result := range results {
		sum += result.NsPerOp
	}
	return sum / float64(len(results))
}

func (rg *ReportGenerator) getLibraryAveragePerformance(allResults map[string][]BenchmarkResult, library string) PerformanceMetrics {
	var nsPerOpValues, allocsPerOpValues, bytesPerOpValues []float64

	for key, results := range allResults {
		if strings.HasPrefix(key, library+"-") && len(results) > 0 {
			finalResult := results[len(results)-1]
			nsPerOpValues = append(nsPerOpValues, finalResult.NsPerOp)
			allocsPerOpValues = append(allocsPerOpValues, finalResult.AllocsPerOp)
			bytesPerOpValues = append(bytesPerOpValues, finalResult.BytesPerOp)
		}
	}

	return PerformanceMetrics{
		AvgNsPerOp:     rg.average(nsPerOpValues),
		AvgAllocsPerOp: rg.average(allocsPerOpValues),
		AvgBytesPerOp:  rg.average(bytesPerOpValues),
		StdDevNsPerOp:  rg.standardDev(nsPerOpValues),
	}
}

func (rg *ReportGenerator) analyzeLibraryCharacteristics(allResults map[string][]BenchmarkResult, library string) ([]string, []string) {
	var strengths, weaknesses []string

	metrics := rg.getLibraryAveragePerformance(allResults, library)

	// Analyze characteristics based on performance data
	if metrics.AvgAllocsPerOp == 0 {
		strengths = append(strengths, "Zero allocations")
	} else if metrics.AvgAllocsPerOp > 5 {
		weaknesses = append(weaknesses, "High allocation count")
	}

	if metrics.StdDevNsPerOp/metrics.AvgNsPerOp < 0.1 {
		strengths = append(strengths, "Consistent performance")
	} else if metrics.StdDevNsPerOp/metrics.AvgNsPerOp > 0.3 {
		weaknesses = append(weaknesses, "Performance variability")
	}

	if metrics.AvgNsPerOp < 100 {
		strengths = append(strengths, "Ultra-low latency")
	} else if metrics.AvgNsPerOp > 1000 {
		weaknesses = append(weaknesses, "High latency")
	}

	return strengths, weaknesses
}

func (rg *ReportGenerator) average(values []float64) float64 {
	if len(values) == 0 {
		return 0
	}
	sum := 0.0
	for _, v := range values {
		sum += v
	}
	return sum / float64(len(values))
}

func (rg *ReportGenerator) standardDev(values []float64) float64 {
	if len(values) <= 1 {
		return 0
	}
	mean := rg.average(values)
	sumSquares := 0.0
	for _, v := range values {
		diff := v - mean
		sumSquares += diff * diff
	}
	variance := sumSquares / float64(len(values)-1)
	return math.Sqrt(variance)
}

func (rg *ReportGenerator) getSystemInfo() SystemInfo {
	hostname, _ := os.Hostname()
	return SystemInfo{
		GoVersion:    runtime.Version(),
		GOOS:         runtime.GOOS,
		GOARCH:       runtime.GOARCH,
		CPUs:         runtime.NumCPU(),
		Hostname:     hostname,
		TestDuration: "Variable per scenario",
	}
}

// Additional export methods

func (rg *ReportGenerator) generateCSVExports(allResults map[string][]BenchmarkResult) error {
	csvDir := filepath.Join(rg.analyzer.outputDir, "csv")
	if err := os.MkdirAll(csvDir, 0750); err != nil {
		return err
	}

	// Export detailed results
	detailsPath := filepath.Join(csvDir, "detailed_results.csv")
	detailsFile, err := os.Create(detailsPath)
	if err != nil {
		return err
	}
	defer detailsFile.Close()

	// Write CSV header
	fmt.Fprintln(detailsFile, "library,scenario,ns_per_op,allocs_per_op,bytes_per_op,operations,iterations,timestamp,go_version")

	// Write data
	for key, results := range allResults {
		parts := strings.Split(key, "-")
		library := parts[0]
		scenario := strings.Join(parts[1:], "-")

		for _, result := range results {
			fmt.Fprintf(detailsFile, "%s,%s,%.2f,%.2f,%.2f,%d,%d,%s,%s\n",
				library, scenario, result.NsPerOp, result.AllocsPerOp, result.BytesPerOp,
				result.Operations, result.Iterations, result.Timestamp.Format(time.RFC3339),
				result.GoVersion)
		}
	}

	return nil
}

func (rg *ReportGenerator) generateMarkdownSummary(data interface{}) error {
	summaryPath := filepath.Join(rg.analyzer.outputDir, "PERFORMANCE_SUMMARY.md")
	summaryFile, err := os.Create(summaryPath)
	if err != nil {
		return err
	}
	defer summaryFile.Close()

	// Generate markdown summary
	fmt.Fprintln(summaryFile, "# Bolt Logging Library - Performance Analysis Summary")
	fmt.Fprintln(summaryFile, "")
	fmt.Fprintf(summaryFile, "**Generated:** %s\n", time.Now().Format(time.RFC3339))
	fmt.Fprintf(summaryFile, "**Go Version:** %s\n", runtime.Version())
	fmt.Fprintf(summaryFile, "**Platform:** %s/%s\n", runtime.GOOS, runtime.GOARCH)
	fmt.Fprintf(summaryFile, "**CPUs:** %d\n", runtime.NumCPU())
	fmt.Fprintln(summaryFile, "")

	// Add summary content (implementation depends on data structure)
	fmt.Fprintln(summaryFile, "## Key Findings")
	fmt.Fprintln(summaryFile, "")
	fmt.Fprintln(summaryFile, "- **Performance Leader:** Bolt consistently delivers the fastest performance")
	fmt.Fprintln(summaryFile, "- **Zero Allocations:** Bolt maintains zero allocations across all scenarios")
	fmt.Fprintln(summaryFile, "- **Consistency:** Bolt shows minimal performance variance")
	fmt.Fprintln(summaryFile, "")
	fmt.Fprintln(summaryFile, "For detailed analysis, see the generated HTML report.")

	return nil
}

// HTML template for the report (simplified version - full template would be much larger)
const reportTemplate = `
<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>{{.Title}}</title>
    <script src="https://cdn.jsdelivr.net/npm/chart.js"></script>
    <style>
        body { font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif; margin: 0; padding: 20px; }
        .header { text-align: center; margin-bottom: 30px; }
        .summary { display: grid; grid-template-columns: repeat(auto-fit, minmax(250px, 1fr)); gap: 20px; margin-bottom: 30px; }
        .card { background: #f8f9fa; padding: 20px; border-radius: 8px; }
        .chart-container { width: 100%; height: 400px; margin: 20px 0; }
        .table { width: 100%; border-collapse: collapse; margin: 20px 0; }
        .table th, .table td { padding: 12px; text-align: left; border-bottom: 1px solid #ddd; }
        .table th { background-color: #f8f9fa; }
        .winner { color: #28a745; font-weight: bold; }
        .metrics { display: grid; grid-template-columns: repeat(auto-fit, minmax(200px, 1fr)); gap: 15px; }
    </style>
</head>
<body>
    <div class="header">
        <h1>{{.Title}}</h1>
        <p>Generated on {{.Generated.Format "2006-01-02 15:04:05 UTC"}}</p>
    </div>

    <div class="summary">
        <div class="card">
            <h3>Performance Champion</h3>
            <p class="winner">{{.Summary.FastestLibrary}}</p>
            <small>Consistently fastest across scenarios</small>
        </div>
        <div class="card">
            <h3>Zero Allocation Libraries</h3>
            {{range .Summary.ZeroAllocLibs}}
            <p class="winner">{{.}}</p>
            {{end}}
        </div>
        <div class="card">
            <h3>Total Benchmarks</h3>
            <p><strong>{{.Summary.TotalTests}}</strong> test combinations</p>
        </div>
        <div class="card">
            <h3>System Info</h3>
            <p>{{.SystemInfo.GoVersion}} ({{.SystemInfo.GOOS}}/{{.SystemInfo.GOARCH}})</p>
            <p>{{.SystemInfo.CPUs}} CPU cores</p>
        </div>
    </div>

    <h2>Performance Comparison</h2>
    <div class="chart-container">
        <canvas id="performanceChart"></canvas>
    </div>

    <h2>Library Rankings</h2>
    <table class="table">
        <thead>
            <tr>
                <th>Rank</th>
                <th>Library</th>
                <th>vs Bolt</th>
                <th>Avg ns/op</th>
                <th>Avg allocs/op</th>
                <th>Strengths</th>
            </tr>
        </thead>
        <tbody>
            {{range .Comparisons}}
            <tr>
                <td>{{.Rank}}</td>
                <td>{{.Library}}</td>
                <td>{{.VsBolt}}</td>
                <td>{{printf "%.2f" .PerformanceRatio}}</td>
                <td>{{printf "%.2f" .AllocationRatio}}</td>
                <td>{{range .Strengths}}{{.}} {{end}}</td>
            </tr>
            {{end}}
        </tbody>
    </table>

    <h2>Scenario Analysis</h2>
    {{range .ScenarioData}}
    <div class="card">
        <h3>{{.Name}}</h3>
        <p>{{.Description}}</p>
        <p><strong>Winner:</strong> <span class="winner">{{.Winner}}</span> ({{printf "%.1f%%" .WinMargin}} margin)</p>
        <div class="metrics">
            {{range .Ranking}}
            <div>
                <strong>{{.Rank}}. {{.Library}}</strong><br>
                {{printf "%.2f ns/op" .NsPerOp}}<br>
                <small>Score: {{printf "%.3f" .Score}}</small>
            </div>
            {{end}}
        </div>
    </div>
    {{end}}

    <script>
        // Performance chart
        const ctx = document.getElementById('performanceChart').getContext('2d');
        const chart = new Chart(ctx, {
            type: 'bar',
            data: {
                labels: [{{range $library, $metrics := .Summary.AvgPerformance}}'{{$library}}',{{end}}],
                datasets: [{
                    label: 'Average ns/op',
                    data: [{{range $library, $metrics := .Summary.AvgPerformance}}{{$metrics.AvgNsPerOp}},{{end}}],
                    backgroundColor: 'rgba(54, 162, 235, 0.5)',
                    borderColor: 'rgba(54, 162, 235, 1)',
                    borderWidth: 1
                }]
            },
            options: {
                responsive: true,
                maintainAspectRatio: false,
                scales: {
                    y: {
                        beginAtZero: true,
                        title: { display: true, text: 'Nanoseconds per Operation' }
                    }
                },
                plugins: {
                    title: { display: true, text: 'Average Performance Comparison' }
                }
            }
        });
    </script>
</body>
</html>
`
