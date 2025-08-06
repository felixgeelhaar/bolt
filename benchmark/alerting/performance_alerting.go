// Package alerting provides comprehensive alerting system for performance monitoring
package alerting

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"os"
	"time"

	"github.com/felixgeelhaar/bolt/benchmark/validation"
)

// AlertingConfig configures the alerting system
type AlertingConfig struct {
	// Alert channels
	SlackWebhookURL    string `json:"slack_webhook_url"`
	DiscordWebhookURL  string `json:"discord_webhook_url"`
	TeamsWebhookURL    string `json:"teams_webhook_url"`
	EmailSMTPServer    string `json:"email_smtp_server"`
	EmailSMTPPort      int    `json:"email_smtp_port"`
	EmailUsername      string `json:"email_username"`
	EmailPassword      string `json:"email_password"`
	EmailRecipients    []string `json:"email_recipients"`
	
	// GitHub integration
	GitHubToken        string `json:"github_token"`
	GitHubOwner        string `json:"github_owner"`
	GitHubRepo         string `json:"github_repo"`
	CreateIssues       bool   `json:"create_issues"`
	
	// Alert thresholds
	AlertOnRegression  bool   `json:"alert_on_regression"`
	AlertOnThreshold   bool   `json:"alert_on_threshold"`
	AlertOnFailure     bool   `json:"alert_on_failure"`
	
	// Rate limiting
	CooldownPeriod     time.Duration `json:"cooldown_period"`
	MaxAlertsPerHour   int    `json:"max_alerts_per_hour"`
	
	// Alert severity levels
	CriticalThreshold  float64 `json:"critical_threshold"`  // Performance degradation %
	WarningThreshold   float64 `json:"warning_threshold"`   // Performance degradation %
}

// DefaultAlertingConfig provides sensible defaults
var DefaultAlertingConfig = AlertingConfig{
	AlertOnRegression:  true,
	AlertOnThreshold:   true,
	AlertOnFailure:     true,
	CooldownPeriod:     30 * time.Minute,
	MaxAlertsPerHour:   10,
	CriticalThreshold:  0.10, // 10% degradation
	WarningThreshold:   0.05, // 5% degradation
	CreateIssues:       true,
}

// AlertSeverity represents the severity of an alert
type AlertSeverity string

const (
	SeverityCritical AlertSeverity = "CRITICAL"
	SeverityWarning  AlertSeverity = "WARNING"
	SeverityInfo     AlertSeverity = "INFO"
)

// PerformanceAlert represents a performance alert
type PerformanceAlert struct {
	ID          string                     `json:"id"`
	Timestamp   time.Time                  `json:"timestamp"`
	Severity    AlertSeverity              `json:"severity"`
	Title       string                     `json:"title"`
	Description string                     `json:"description"`
	
	// Context information
	Repository  string                     `json:"repository"`
	Branch      string                     `json:"branch"`
	Commit      string                     `json:"commit"`
	TestSuite   string                     `json:"test_suite"`
	
	// Performance metrics
	RegressionDetails []RegressionAlert    `json:"regression_details"`
	ThresholdViolations []ThresholdAlert   `json:"threshold_violations"`
	
	// Validation results
	ValidationResult *validation.ValidationResult `json:"validation_result"`
	
	// Alert metadata
	Actions     []AlertAction              `json:"actions"`
	Links       []AlertLink               `json:"links"`
	Tags        []string                  `json:"tags"`
}

type RegressionAlert struct {
	Library       string  `json:"library"`
	Metric        string  `json:"metric"`
	CurrentValue  float64 `json:"current_value"`
	BaselineValue float64 `json:"baseline_value"`
	ChangePercent float64 `json:"change_percent"`
	Impact        string  `json:"impact"`
}

type ThresholdAlert struct {
	Library     string  `json:"library"`
	Metric      string  `json:"metric"`
	ActualValue float64 `json:"actual_value"`
	Threshold   float64 `json:"threshold"`
	Deviation   float64 `json:"deviation"`
	Impact      string  `json:"impact"`
}

type AlertAction struct {
	Type        string `json:"type"`
	Label       string `json:"label"`
	URL         string `json:"url"`
	Description string `json:"description"`
}

type AlertLink struct {
	Label string `json:"label"`
	URL   string `json:"url"`
}

// PerformanceAlerting manages performance alerts
type PerformanceAlerting struct {
	config      AlertingConfig
	httpClient  *http.Client
	alertHistory map[string]time.Time // For rate limiting
}

// NewPerformanceAlerting creates a new alerting system
func NewPerformanceAlerting(config AlertingConfig) *PerformanceAlerting {
	return &PerformanceAlerting{
		config:      config,
		httpClient:  &http.Client{Timeout: 30 * time.Second},
		alertHistory: make(map[string]time.Time),
	}
}

// ProcessValidationResults analyzes validation results and triggers alerts if needed
func (pa *PerformanceAlerting) ProcessValidationResults(ctx context.Context, result *validation.ValidationResult) error {
	if result == nil {
		return fmt.Errorf("validation result is nil")
	}

	// Determine if alerting is needed
	needsAlert, severity := pa.shouldAlert(result)
	if !needsAlert {
		return nil // No alert needed
	}

	// Create performance alert
	alert := pa.createPerformanceAlert(result, severity)

	// Send alerts through all configured channels
	return pa.sendAlert(ctx, alert)
}

// shouldAlert determines if an alert should be sent based on validation results
func (pa *PerformanceAlerting) shouldAlert(result *validation.ValidationResult) (bool, AlertSeverity) {
	// Check for failures
	if pa.config.AlertOnFailure && result.OverallResult == validation.ValidationFailed {
		return true, SeverityCritical
	}

	// Check for critical regressions
	if pa.config.AlertOnRegression {
		for _, check := range result.RegressionChecks {
			if check.Status == validation.ValidationFailed && !check.IsImprovement {
				degradation := math.Abs(check.ChangePercent) / 100
				if degradation >= pa.config.CriticalThreshold {
					return true, SeverityCritical
				} else if degradation >= pa.config.WarningThreshold {
					return true, SeverityWarning
				}
			}
		}
	}

	// Check for threshold violations
	if pa.config.AlertOnThreshold {
		for _, check := range result.ThresholdChecks {
			if check.Status == validation.ValidationFailed {
				// Critical metrics that should always trigger alerts
				if check.Metric == "allocs_per_op" && check.ActualValue > 0 {
					return true, SeverityCritical
				}
				if check.Metric == "ns_per_op" && check.Deviation > pa.config.CriticalThreshold {
					return true, SeverityCritical
				}
			}
		}
	}

	// Check quality gates
	if !result.QualityGates.OverallGate {
		gateScore := result.QualityGates.GateScore
		if gateScore < 70 {
			return true, SeverityCritical
		} else if gateScore < 85 {
			return true, SeverityWarning
		}
	}

	return false, SeverityInfo
}

// createPerformanceAlert creates a structured alert from validation results
func (pa *PerformanceAlerting) createPerformanceAlert(result *validation.ValidationResult, severity AlertSeverity) *PerformanceAlert {
	alert := &PerformanceAlert{
		ID:               fmt.Sprintf("perf-alert-%d", time.Now().Unix()),
		Timestamp:        time.Now(),
		Severity:         severity,
		Repository:       pa.getEnvOrDefault("GITHUB_REPOSITORY", "bolt"),
		Branch:           pa.getEnvOrDefault("GITHUB_REF_NAME", "main"),
		Commit:           pa.getEnvOrDefault("GITHUB_SHA", "unknown"),
		TestSuite:        result.TestSuite,
		ValidationResult: result,
		Tags:             []string{"performance", "benchmark", "bolt"},
	}

	// Generate title and description based on severity
	switch severity {
	case SeverityCritical:
		alert.Title = "🚨 CRITICAL: Performance Regression Detected"
		alert.Description = pa.generateCriticalDescription(result)
		alert.Tags = append(alert.Tags, "critical", "regression")
	case SeverityWarning:
		alert.Title = "⚠️ WARNING: Performance Degradation Detected"
		alert.Description = pa.generateWarningDescription(result)
		alert.Tags = append(alert.Tags, "warning", "degradation")
	default:
		alert.Title = "ℹ️ INFO: Performance Check Completed"
		alert.Description = pa.generateInfoDescription(result)
		alert.Tags = append(alert.Tags, "info")
	}

	// Extract regression details
	for _, check := range result.RegressionChecks {
		if check.Status == validation.ValidationFailed && !check.IsImprovement {
			alert.RegressionDetails = append(alert.RegressionDetails, RegressionAlert{
				Library:       check.Library,
				Metric:        check.Metric,
				CurrentValue:  check.CurrentValue,
				BaselineValue: check.BaselineValue,
				ChangePercent: check.ChangePercent,
				Impact:        pa.assessImpact(check.ChangePercent),
			})
		}
	}

	// Extract threshold violations
	for _, check := range result.ThresholdChecks {
		if check.Status == validation.ValidationFailed {
			alert.ThresholdViolations = append(alert.ThresholdViolations, ThresholdAlert{
				Library:     check.Library,
				Metric:      check.Metric,
				ActualValue: check.ActualValue,
				Threshold:   check.ThresholdValue,
				Deviation:   check.Deviation,
				Impact:      pa.assessThresholdImpact(check.Metric, check.Deviation),
			})
		}
	}

	// Add action items
	alert.Actions = pa.generateAlertActions(result)

	// Add useful links
	alert.Links = pa.generateAlertLinks(result)

	return alert
}

// sendAlert sends the alert through all configured channels
func (pa *PerformanceAlerting) sendAlert(ctx context.Context, alert *PerformanceAlert) error {
	var errors []error

	// Check rate limiting
	if !pa.shouldSendAlert(alert) {
		return nil
	}

	// Send Slack alert
	if pa.config.SlackWebhookURL != "" {
		if err := pa.sendSlackAlert(ctx, alert); err != nil {
			errors = append(errors, fmt.Errorf("slack alert failed: %w", err))
		}
	}

	// Send Discord alert
	if pa.config.DiscordWebhookURL != "" {
		if err := pa.sendDiscordAlert(ctx, alert); err != nil {
			errors = append(errors, fmt.Errorf("discord alert failed: %w", err))
		}
	}

	// Send Teams alert
	if pa.config.TeamsWebhookURL != "" {
		if err := pa.sendTeamsAlert(ctx, alert); err != nil {
			errors = append(errors, fmt.Errorf("teams alert failed: %w", err))
		}
	}

	// Send email alert
	if len(pa.config.EmailRecipients) > 0 && pa.config.EmailSMTPServer != "" {
		if err := pa.sendEmailAlert(ctx, alert); err != nil {
			errors = append(errors, fmt.Errorf("email alert failed: %w", err))
		}
	}

	// Create GitHub issue
	if pa.config.CreateIssues && pa.config.GitHubToken != "" {
		if err := pa.createGitHubIssue(ctx, alert); err != nil {
			errors = append(errors, fmt.Errorf("github issue creation failed: %w", err))
		}
	}

	// Update rate limiting
	pa.recordAlert(alert)

	if len(errors) > 0 {
		return fmt.Errorf("alert sending failed: %v", errors)
	}

	return nil
}

// sendSlackAlert sends alert to Slack
func (pa *PerformanceAlerting) sendSlackAlert(ctx context.Context, alert *PerformanceAlert) error {
	color := "#ff0000" // red
	if alert.Severity == SeverityWarning {
		color = "#ffaa00" // orange
	} else if alert.Severity == SeverityInfo {
		color = "#00ff00" // green
	}

	payload := map[string]interface{}{
		"text": alert.Title,
		"attachments": []map[string]interface{}{
			{
				"color": color,
				"title": alert.Title,
				"text":  alert.Description,
				"fields": []map[string]interface{}{
					{
						"title": "Repository",
						"value": alert.Repository,
						"short": true,
					},
					{
						"title": "Branch",
						"value": alert.Branch,
						"short": true,
					},
					{
						"title": "Success Rate",
						"value": fmt.Sprintf("%.1f%%", alert.ValidationResult.SuccessRate*100),
						"short": true,
					},
					{
						"title": "Quality Score",
						"value": fmt.Sprintf("%.0f/100", alert.ValidationResult.QualityGates.GateScore),
						"short": true,
					},
				},
				"actions": pa.generateSlackActions(alert),
				"ts": alert.Timestamp.Unix(),
			},
		},
	}

	return pa.sendWebhook(ctx, pa.config.SlackWebhookURL, payload)
}

// sendDiscordAlert sends alert to Discord
func (pa *PerformanceAlerting) sendDiscordAlert(ctx context.Context, alert *PerformanceAlert) error {
	color := 0xff0000 // red
	if alert.Severity == SeverityWarning {
		color = 0xffaa00 // orange
	} else if alert.Severity == SeverityInfo {
		color = 0x00ff00 // green
	}

	payload := map[string]interface{}{
		"embeds": []map[string]interface{}{
			{
				"title":       alert.Title,
				"description": alert.Description,
				"color":       color,
				"timestamp":   alert.Timestamp.Format(time.RFC3339),
				"fields": []map[string]interface{}{
					{
						"name":   "Repository",
						"value":  alert.Repository,
						"inline": true,
					},
					{
						"name":   "Branch",
						"value":  alert.Branch,
						"inline": true,
					},
					{
						"name":   "Commit",
						"value":  alert.Commit[:8],
						"inline": true,
					},
					{
						"name":   "Success Rate",
						"value":  fmt.Sprintf("%.1f%%", alert.ValidationResult.SuccessRate*100),
						"inline": true,
					},
				},
			},
		},
	}

	return pa.sendWebhook(ctx, pa.config.DiscordWebhookURL, payload)
}

// sendTeamsAlert sends alert to Microsoft Teams
func (pa *PerformanceAlerting) sendTeamsAlert(ctx context.Context, alert *PerformanceAlert) error {
	themeColor := "ff0000" // red
	if alert.Severity == SeverityWarning {
		themeColor = "ffaa00" // orange
	} else if alert.Severity == SeverityInfo {
		themeColor = "00ff00" // green
	}

	payload := map[string]interface{}{
		"@type":      "MessageCard",
		"@context":   "https://schema.org/extensions",
		"summary":    alert.Title,
		"themeColor": themeColor,
		"title":      alert.Title,
		"text":       alert.Description,
		"sections": []map[string]interface{}{
			{
				"facts": []map[string]interface{}{
					{
						"name":  "Repository",
						"value": alert.Repository,
					},
					{
						"name":  "Branch",
						"value": alert.Branch,
					},
					{
						"name":  "Success Rate",
						"value": fmt.Sprintf("%.1f%%", alert.ValidationResult.SuccessRate*100),
					},
					{
						"name":  "Quality Score",
						"value": fmt.Sprintf("%.0f/100", alert.ValidationResult.QualityGates.GateScore),
					},
				},
			},
		},
		"potentialAction": pa.generateTeamsActions(alert),
	}

	return pa.sendWebhook(ctx, pa.config.TeamsWebhookURL, payload)
}

// sendEmailAlert sends alert via email
func (pa *PerformanceAlerting) sendEmailAlert(ctx context.Context, alert *PerformanceAlert) error {
	// Simplified email implementation - would need proper SMTP setup
	fmt.Printf("EMAIL ALERT: %s\n%s\n", alert.Title, alert.Description)
	return nil
}

// createGitHubIssue creates a GitHub issue for the alert
func (pa *PerformanceAlerting) createGitHubIssue(ctx context.Context, alert *PerformanceAlert) error {
	if pa.config.GitHubToken == "" || pa.config.GitHubOwner == "" || pa.config.GitHubRepo == "" {
		return fmt.Errorf("github configuration incomplete")
	}

	issueBody := pa.generateGitHubIssueBody(alert)
	
	payload := map[string]interface{}{
		"title": alert.Title,
		"body":  issueBody,
		"labels": alert.Tags,
	}

	url := fmt.Sprintf("https://api.github.com/repos/%s/%s/issues", 
		pa.config.GitHubOwner, pa.config.GitHubRepo)

	req, err := http.NewRequestWithContext(ctx, "POST", url, 
		bytes.NewBuffer([]byte(fmt.Sprintf("%v", payload))))
	if err != nil {
		return err
	}

	req.Header.Set("Authorization", "token "+pa.config.GitHubToken)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/vnd.github.v3+json")

	resp, err := pa.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return fmt.Errorf("github api error: %d", resp.StatusCode)
	}

	return nil
}

// Helper methods

func (pa *PerformanceAlerting) sendWebhook(ctx context.Context, webhookURL string, payload interface{}) error {
	jsonData, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal payload: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", webhookURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := pa.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send webhook: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return fmt.Errorf("webhook returned error: %d", resp.StatusCode)
	}

	return nil
}

func (pa *PerformanceAlerting) shouldSendAlert(alert *PerformanceAlert) bool {
	// Check cooldown period
	key := fmt.Sprintf("%s-%s", alert.Severity, alert.Repository)
	if lastAlert, exists := pa.alertHistory[key]; exists {
		if time.Since(lastAlert) < pa.config.CooldownPeriod {
			return false // Still in cooldown
		}
	}

	// Check rate limiting (simplified)
	hourAgo := time.Now().Add(-time.Hour)
	recentAlerts := 0
	for _, timestamp := range pa.alertHistory {
		if timestamp.After(hourAgo) {
			recentAlerts++
		}
	}

	return recentAlerts < pa.config.MaxAlertsPerHour
}

func (pa *PerformanceAlerting) recordAlert(alert *PerformanceAlert) {
	key := fmt.Sprintf("%s-%s", alert.Severity, alert.Repository)
	pa.alertHistory[key] = alert.Timestamp
}

func (pa *PerformanceAlerting) getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func (pa *PerformanceAlerting) generateCriticalDescription(result *validation.ValidationResult) string {
	return fmt.Sprintf(`Critical performance issues detected in Bolt benchmarks:

• Success Rate: %.1f%%
• Quality Score: %.0f/100
• Failed Tests: %d
• Critical Issues: %d

Immediate attention required!`, 
		result.SuccessRate*100,
		result.QualityGates.GateScore,
		result.FailedTests,
		len(result.CriticalIssues))
}

func (pa *PerformanceAlerting) generateWarningDescription(result *validation.ValidationResult) string {
	return fmt.Sprintf(`Performance degradation detected in Bolt benchmarks:

• Success Rate: %.1f%%
• Quality Score: %.0f/100
• Warning Tests: %d
• Issues: %d

Review recommended.`, 
		result.SuccessRate*100,
		result.QualityGates.GateScore,
		result.WarningTests,
		len(result.Warnings))
}

func (pa *PerformanceAlerting) generateInfoDescription(result *validation.ValidationResult) string {
	return fmt.Sprintf(`Performance validation completed successfully:

• Success Rate: %.1f%%
• Quality Score: %.0f/100
• All Tests: %d

No action required.`, 
		result.SuccessRate*100,
		result.QualityGates.GateScore,
		result.TotalTests)
}

func (pa *PerformanceAlerting) assessImpact(changePercent float64) string {
	absChange := math.Abs(changePercent)
	if absChange > 20 {
		return "HIGH"
	} else if absChange > 10 {
		return "MEDIUM"
	} else if absChange > 5 {
		return "LOW"
	}
	return "MINIMAL"
}

func (pa *PerformanceAlerting) assessThresholdImpact(metric string, deviation float64) string {
	if metric == "allocs_per_op" {
		return "CRITICAL" // Zero allocation requirement
	}
	absDeviation := math.Abs(deviation)
	if absDeviation > 0.5 {
		return "HIGH"
	} else if absDeviation > 0.2 {
		return "MEDIUM"
	}
	return "LOW"
}

func (pa *PerformanceAlerting) generateAlertActions(result *validation.ValidationResult) []AlertAction {
	actions := []AlertAction{
		{
			Type:        "link",
			Label:       "View Full Report",
			URL:         fmt.Sprintf("https://%s.github.io/bolt/performance/", pa.config.GitHubOwner),
			Description: "Open detailed performance report",
		},
	}

	if len(result.CriticalIssues) > 0 {
		actions = append(actions, AlertAction{
			Type:        "link",
			Label:       "Debug Performance",
			URL:         "https://github.com/felixgeelhaar/bolt/blob/main/TROUBLESHOOTING.md#performance",
			Description: "Performance troubleshooting guide",
		})
	}

	return actions
}

func (pa *PerformanceAlerting) generateAlertLinks(result *validation.ValidationResult) []AlertLink {
	return []AlertLink{
		{
			Label: "Performance Dashboard",
			URL:   fmt.Sprintf("https://%s.github.io/bolt/performance/", pa.config.GitHubOwner),
		},
		{
			Label: "Repository",
			URL:   fmt.Sprintf("https://github.com/%s/%s", pa.config.GitHubOwner, pa.config.GitHubRepo),
		},
	}
}

func (pa *PerformanceAlerting) generateSlackActions(alert *PerformanceAlert) []map[string]interface{} {
	return []map[string]interface{}{
		{
			"type": "button",
			"text": "View Report",
			"url":  fmt.Sprintf("https://%s.github.io/bolt/performance/", pa.config.GitHubOwner),
		},
	}
}

func (pa *PerformanceAlerting) generateTeamsActions(alert *PerformanceAlert) []map[string]interface{} {
	return []map[string]interface{}{
		{
			"@type": "OpenUri",
			"name":  "View Report",
			"targets": []map[string]string{
				{
					"os":  "default",
					"uri": fmt.Sprintf("https://%s.github.io/bolt/performance/", pa.config.GitHubOwner),
				},
			},
		},
	}
}

func (pa *PerformanceAlerting) generateGitHubIssueBody(alert *PerformanceAlert) string {
	body := fmt.Sprintf(`## %s

%s

### Performance Metrics

- **Success Rate:** %.1f%%
- **Quality Score:** %.0f/100
- **Total Tests:** %d
- **Failed Tests:** %d

### Context

- **Repository:** %s
- **Branch:** %s
- **Commit:** %s
- **Timestamp:** %s

`, alert.Title, alert.Description, 
		alert.ValidationResult.SuccessRate*100,
		alert.ValidationResult.QualityGates.GateScore,
		alert.ValidationResult.TotalTests,
		alert.ValidationResult.FailedTests,
		alert.Repository,
		alert.Branch,
		alert.Commit,
		alert.Timestamp.Format(time.RFC3339))

	if len(alert.RegressionDetails) > 0 {
		body += "\n### Performance Regressions\n\n"
		for _, reg := range alert.RegressionDetails {
			body += fmt.Sprintf("- **%s %s:** %.2f → %.2f (%.1f%% change, %s impact)\n",
				reg.Library, reg.Metric, reg.BaselineValue, reg.CurrentValue, reg.ChangePercent, reg.Impact)
		}
	}

	if len(alert.ThresholdViolations) > 0 {
		body += "\n### Threshold Violations\n\n"
		for _, violation := range alert.ThresholdViolations {
			body += fmt.Sprintf("- **%s %s:** %.2f exceeds threshold %.2f (%s impact)\n",
				violation.Library, violation.Metric, violation.ActualValue, violation.Threshold, violation.Impact)
		}
	}

	if len(alert.ValidationResult.Recommendations) > 0 {
		body += "\n### Recommendations\n\n"
		for _, rec := range alert.ValidationResult.Recommendations {
			body += fmt.Sprintf("- %s\n", rec)
		}
	}

	body += "\n### Links\n\n"
	for _, link := range alert.Links {
		body += fmt.Sprintf("- [%s](%s)\n", link.Label, link.URL)
	}

	return body
}

