package zap

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
)

// ConfigMigrator handles migration of Zap configurations to Bolt.
type ConfigMigrator struct {
	dryRun bool
}

// NewConfigMigrator creates a new configuration migrator.
func NewConfigMigrator(dryRun bool) *ConfigMigrator {
	return &ConfigMigrator{dryRun: dryRun}
}

// ZapConfig represents a Zap configuration that needs to be migrated.
type ZapConfig struct {
	Level             string              `json:"level" yaml:"level"`
	Development       bool                `json:"development" yaml:"development"`
	DisableCaller     bool                `json:"disableCaller" yaml:"disableCaller"`
	DisableStacktrace bool                `json:"disableStacktrace" yaml:"disableStacktrace"`
	Sampling          *ZapSampling        `json:"sampling" yaml:"sampling"`
	Encoding          string              `json:"encoding" yaml:"encoding"`
	EncoderConfig     ZapEncoderConfig    `json:"encoderConfig" yaml:"encoderConfig"`
	OutputPaths       []string            `json:"outputPaths" yaml:"outputPaths"`
	ErrorOutputPaths  []string            `json:"errorOutputPaths" yaml:"errorOutputPaths"`
	InitialFields     map[string]interface{} `json:"initialFields" yaml:"initialFields"`
}

// ZapSampling represents Zap sampling configuration.
type ZapSampling struct {
	Initial    int `json:"initial" yaml:"initial"`
	Thereafter int `json:"thereafter" yaml:"thereafter"`
}

// ZapEncoderConfig represents Zap encoder configuration.
type ZapEncoderConfig struct {
	MessageKey       string `json:"messageKey" yaml:"messageKey"`
	LevelKey         string `json:"levelKey" yaml:"levelKey"`
	TimeKey          string `json:"timeKey" yaml:"timeKey"`
	NameKey          string `json:"nameKey" yaml:"nameKey"`
	CallerKey        string `json:"callerKey" yaml:"callerKey"`
	FunctionKey      string `json:"functionKey" yaml:"functionKey"`
	StacktraceKey    string `json:"stacktraceKey" yaml:"stacktraceKey"`
	SkipLineEnding   bool   `json:"skipLineEnding" yaml:"skipLineEnding"`
	LineEnding       string `json:"lineEnding" yaml:"lineEnding"`
	ConsoleSeparator string `json:"consoleSeparator" yaml:"consoleSeparator"`
}

// BoltConfig represents the equivalent Bolt configuration.
type BoltConfig struct {
	Level       string                 `json:"level"`
	Format      string                 `json:"format"`
	Development bool                   `json:"development"`
	Fields      map[string]interface{} `json:"initial_fields,omitempty"`
	Notes       []string               `json:"migration_notes"`
}

// MigrationResult contains the result of a configuration migration.
type MigrationResult struct {
	OriginalConfig ZapConfig   `json:"original_config"`
	BoltConfig     BoltConfig  `json:"bolt_config"`
	CodeSuggestion string      `json:"code_suggestion"`
	Warnings       []string    `json:"warnings"`
	Success        bool        `json:"success"`
}

// MigrateConfigFromJSON migrates a Zap JSON configuration to Bolt.
func (cm *ConfigMigrator) MigrateConfigFromJSON(jsonData []byte) (*MigrationResult, error) {
	var zapConfig ZapConfig
	if err := json.Unmarshal(jsonData, &zapConfig); err != nil {
		return nil, fmt.Errorf("failed to parse Zap config: %w", err)
	}

	return cm.migrateConfig(zapConfig), nil
}

// MigrateConfigFromFile migrates a Zap configuration file to Bolt.
func (cm *ConfigMigrator) MigrateConfigFromFile(filePath string) (*MigrationResult, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	return cm.MigrateConfigFromJSON(data)
}

// migrateConfig performs the actual configuration migration.
func (cm *ConfigMigrator) migrateConfig(zapConfig ZapConfig) *MigrationResult {
	result := &MigrationResult{
		OriginalConfig: zapConfig,
		Success:        true,
		Warnings:       []string{},
	}

	boltConfig := BoltConfig{
		Development: zapConfig.Development,
		Fields:      zapConfig.InitialFields,
		Notes:       []string{},
	}

	// Migrate log level
	boltConfig.Level = cm.migrateLevelString(zapConfig.Level)

	// Migrate encoding/format
	switch zapConfig.Encoding {
	case "json":
		boltConfig.Format = "json"
	case "console":
		boltConfig.Format = "console"
	default:
		boltConfig.Format = "json"
		result.Warnings = append(result.Warnings,
			fmt.Sprintf("Unknown encoding '%s', defaulting to JSON", zapConfig.Encoding))
	}

	// Handle sampling
	if zapConfig.Sampling != nil {
		result.Warnings = append(result.Warnings,
			"Zap sampling configuration detected - Bolt doesn't currently support sampling, consider implementing custom handler")
		boltConfig.Notes = append(boltConfig.Notes,
			fmt.Sprintf("Original sampling: initial=%d, thereafter=%d", 
				zapConfig.Sampling.Initial, zapConfig.Sampling.Thereafter))
	}

	// Handle output paths
	if len(zapConfig.OutputPaths) > 1 {
		result.Warnings = append(result.Warnings,
			"Multiple output paths detected - Bolt supports single output, consider custom handler for multiple outputs")
	}

	// Handle encoder config
	if zapConfig.EncoderConfig.MessageKey != "message" && zapConfig.EncoderConfig.MessageKey != "" {
		result.Warnings = append(result.Warnings,
			fmt.Sprintf("Custom message key '%s' detected - Bolt uses 'message'", zapConfig.EncoderConfig.MessageKey))
	}

	if zapConfig.EncoderConfig.LevelKey != "level" && zapConfig.EncoderConfig.LevelKey != "" {
		result.Warnings = append(result.Warnings,
			fmt.Sprintf("Custom level key '%s' detected - Bolt uses 'level'", zapConfig.EncoderConfig.LevelKey))
	}

	// Handle caller/stacktrace settings
	if !zapConfig.DisableCaller {
		boltConfig.Notes = append(boltConfig.Notes, "Consider using .Caller() method for caller information")
	}

	if !zapConfig.DisableStacktrace {
		boltConfig.Notes = append(boltConfig.Notes, "Consider using .Stack() method for stack traces")
	}

	result.BoltConfig = boltConfig
	result.CodeSuggestion = cm.generateCodeSuggestion(boltConfig)

	return result
}

// migrateLevelString converts Zap level strings to Bolt level strings.
func (cm *ConfigMigrator) migrateLevelString(zapLevel string) string {
	switch strings.ToLower(zapLevel) {
	case "debug":
		return "debug"
	case "info":
		return "info"
	case "warn":
		return "warn"
	case "error":
		return "error"
	case "fatal":
		return "fatal"
	case "panic":
		return "fatal" // Bolt doesn't have panic level, use fatal
	default:
		return "info"
	}
}

// generateCodeSuggestion generates Go code for the equivalent Bolt configuration.
func (cm *ConfigMigrator) generateCodeSuggestion(config BoltConfig) string {
	var code strings.Builder

	code.WriteString("// Migrated Bolt logger configuration\n")
	code.WriteString("package main\n\n")
	code.WriteString("import (\n")
	code.WriteString("    \"os\"\n")
	code.WriteString("    \"github.com/felixgeelhaar/bolt\"\n")
	code.WriteString(")\n\n")

	code.WriteString("func createLogger() *bolt.Logger {\n")

	// Handler selection
	if config.Format == "console" {
		code.WriteString("    handler := bolt.NewConsoleHandler(os.Stdout)\n")
	} else {
		code.WriteString("    handler := bolt.NewJSONHandler(os.Stdout)\n")
	}

	code.WriteString("    logger := bolt.New(handler)\n")

	// Set level
	boltLevel := strings.ToUpper(config.Level)
	code.WriteString(fmt.Sprintf("    logger = logger.SetLevel(bolt.%s)\n", boltLevel))

	// Add initial fields if present
	if len(config.Fields) > 0 {
		code.WriteString("\n    // Add initial context fields\n")
		code.WriteString("    logger = logger.With()")
		for key, value := range config.Fields {
			switch v := value.(type) {
			case string:
				code.WriteString(fmt.Sprintf(".\n        Str(\"%s\", \"%s\")", key, v))
			case int:
				code.WriteString(fmt.Sprintf(".\n        Int(\"%s\", %d)", key, v))
			case bool:
				code.WriteString(fmt.Sprintf(".\n        Bool(\"%s\", %t)", key, v))
			case float64:
				code.WriteString(fmt.Sprintf(".\n        Float64(\"%s\", %f)", key, v))
			default:
				code.WriteString(fmt.Sprintf(".\n        Any(\"%s\", %v)", key, v))
			}
		}
		code.WriteString(".\n        Logger()\n")
	}

	code.WriteString("\n    return logger\n")
	code.WriteString("}\n")

	return code.String()
}

// ConfigAnalyzer analyzes existing Zap configurations in code.
type ConfigAnalyzer struct{}

// NewConfigAnalyzer creates a new configuration analyzer.
func NewConfigAnalyzer() *ConfigAnalyzer {
	return &ConfigAnalyzer{}
}

// AnalysisResult contains the results of configuration analysis.
type AnalysisResult struct {
	ConfigType        string              `json:"config_type"`
	DetectedSettings  map[string]string   `json:"detected_settings"`
	MigrationSuggestions []string         `json:"migration_suggestions"`
	RequiresManualWork bool               `json:"requires_manual_work"`
}

// AnalyzeCode analyzes Go code for Zap configuration patterns.
func (ca *ConfigAnalyzer) AnalyzeCode(code string) *AnalysisResult {
	result := &AnalysisResult{
		DetectedSettings:     make(map[string]string),
		MigrationSuggestions: []string{},
		RequiresManualWork:   false,
	}

	lines := strings.Split(code, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		ca.analyzeLine(line, result)
	}

	// Generate migration suggestions based on detected patterns
	ca.generateSuggestions(result)

	return result
}

// analyzeLine analyzes a single line of code for configuration patterns.
func (ca *ConfigAnalyzer) analyzeLine(line string, result *AnalysisResult) {
	// Detect configuration method calls
	patterns := map[string]string{
		"NewProduction":   "production_config",
		"NewDevelopment":  "development_config",
		"NewExample":      "example_config",
		"Config{":         "struct_config",
		".Level(":         "level_setting",
		".Encoding(":      "encoding_setting",
		".OutputPaths(":   "output_paths",
		".ErrorOutputPaths(": "error_output_paths",
		".InitialFields(": "initial_fields",
		".Sampling(":      "sampling_config",
	}

	for pattern, configType := range patterns {
		if strings.Contains(line, pattern) {
			result.DetectedSettings[configType] = line
			if pattern == ".Sampling(" {
				result.RequiresManualWork = true
			}
		}
	}

	// Detect specific configuration values
	if strings.Contains(line, "zapcore.") {
		result.DetectedSettings["zapcore_usage"] = line
		result.RequiresManualWork = true
	}
}

// generateSuggestions generates migration suggestions based on detected patterns.
func (ca *ConfigAnalyzer) generateSuggestions(result *AnalysisResult) {
	for configType := range result.DetectedSettings {
		switch configType {
		case "production_config":
			result.MigrationSuggestions = append(result.MigrationSuggestions,
				"Replace zap.NewProduction() with bolt.New(bolt.NewJSONHandler(os.Stdout)).SetLevel(bolt.INFO)")
		case "development_config":
			result.MigrationSuggestions = append(result.MigrationSuggestions,
				"Replace zap.NewDevelopment() with bolt.New(bolt.NewConsoleHandler(os.Stdout)).SetLevel(bolt.DEBUG)")
		case "struct_config":
			result.MigrationSuggestions = append(result.MigrationSuggestions,
				"Use ConfigMigrator to convert zap.Config struct to Bolt equivalent")
		case "sampling_config":
			result.MigrationSuggestions = append(result.MigrationSuggestions,
				"Sampling not directly supported in Bolt - consider implementing custom handler")
		case "zapcore_usage":
			result.MigrationSuggestions = append(result.MigrationSuggestions,
				"zapcore usage requires manual migration - review Bolt's handler system")
		case "initial_fields":
			result.MigrationSuggestions = append(result.MigrationSuggestions,
				"Use logger.With().Fields(...).Logger() pattern for initial fields")
		}
	}
}

// EnvironmentMigrator helps migrate environment-based Zap configurations.
type EnvironmentMigrator struct{}

// NewEnvironmentMigrator creates a new environment migrator.
func NewEnvironmentMigrator() *EnvironmentMigrator {
	return &EnvironmentMigrator{}
}

// ZapEnvConfig represents Zap environment configuration.
type ZapEnvConfig struct {
	Level    string `env:"ZAP_LEVEL"`
	Encoding string `env:"ZAP_ENCODING"`
	DevMode  string `env:"ZAP_DEV_MODE"`
}

// BoltEnvConfig represents the equivalent Bolt environment configuration.
type BoltEnvConfig struct {
	Level  string `env:"BOLT_LEVEL"`
	Format string `env:"BOLT_FORMAT"`
}

// MigrateEnvironmentVars provides a mapping of Zap to Bolt environment variables.
func (em *EnvironmentMigrator) MigrateEnvironmentVars() map[string]string {
	return map[string]string{
		"ZAP_LEVEL":    "BOLT_LEVEL",
		"ZAP_ENCODING": "BOLT_FORMAT", 
		"ZAP_DEV_MODE": "BOLT_FORMAT=console (if true)",
	}
}

// GenerateEnvMigrationScript generates a shell script to migrate environment variables.
func (em *EnvironmentMigrator) GenerateEnvMigrationScript() string {
	script := `#!/bin/bash
# Zap to Bolt Environment Variable Migration Script

echo "Migrating Zap environment variables to Bolt..."

# Migrate log level
if [ ! -z "$ZAP_LEVEL" ]; then
    export BOLT_LEVEL="$ZAP_LEVEL"
    echo "Migrated ZAP_LEVEL=$ZAP_LEVEL to BOLT_LEVEL=$BOLT_LEVEL"
fi

# Migrate encoding to format
if [ ! -z "$ZAP_ENCODING" ]; then
    if [ "$ZAP_ENCODING" = "json" ]; then
        export BOLT_FORMAT="json"
    elif [ "$ZAP_ENCODING" = "console" ]; then
        export BOLT_FORMAT="console"
    else
        export BOLT_FORMAT="json"
        echo "Warning: Unknown ZAP_ENCODING=$ZAP_ENCODING, defaulting to json"
    fi
    echo "Migrated ZAP_ENCODING=$ZAP_ENCODING to BOLT_FORMAT=$BOLT_FORMAT"
fi

# Handle development mode
if [ "$ZAP_DEV_MODE" = "true" ]; then
    export BOLT_FORMAT="console"
    export BOLT_LEVEL="debug"
    echo "Development mode detected, set BOLT_FORMAT=console and BOLT_LEVEL=debug"
fi

echo "Environment variable migration completed!"
echo "Current Bolt configuration:"
echo "  BOLT_LEVEL=$BOLT_LEVEL"
echo "  BOLT_FORMAT=$BOLT_FORMAT"
`
	return script
}

// SaveEnvMigrationScript saves the environment migration script to a file.
func (em *EnvironmentMigrator) SaveEnvMigrationScript(filePath string) error {
	script := em.GenerateEnvMigrationScript()
	return os.WriteFile(filePath, []byte(script), 0755)
}

// ConfigComparator compares Zap and Bolt configurations side by side.
type ConfigComparator struct{}

// NewConfigComparator creates a new configuration comparator.
func NewConfigComparator() *ConfigComparator {
	return &ConfigComparator{}
}

// ComparisonReport contains a side-by-side comparison of configurations.
type ComparisonReport struct {
	ZapConfig   string   `json:"zap_config"`
	BoltConfig  string   `json:"bolt_config"`
	Differences []string `json:"differences"`
	Benefits    []string `json:"benefits"`
}

// Compare generates a side-by-side comparison of Zap and Bolt configurations.
func (cc *ConfigComparator) Compare(zapConfig ZapConfig, boltConfig BoltConfig) *ComparisonReport {
	report := &ComparisonReport{
		Differences: []string{},
		Benefits:    []string{},
	}

	// Generate Zap configuration representation
	zapConfigStr := cc.generateZapConfigString(zapConfig)
	report.ZapConfig = zapConfigStr

	// Generate Bolt configuration representation  
	boltConfigStr := cc.generateBoltConfigString(boltConfig)
	report.BoltConfig = boltConfigStr

	// Identify key differences
	cc.identifyDifferences(zapConfig, boltConfig, report)

	// List migration benefits
	report.Benefits = []string{
		"80% faster logging performance compared to Zap",
		"Zero memory allocations in hot paths",
		"Sub-100ns latency for logging operations",
		"Simpler configuration with sensible defaults",
		"Built-in OpenTelemetry integration",
		"No need for explicit logger synchronization",
		"Smaller binary size and dependencies",
	}

	return report
}

// generateZapConfigString creates a string representation of Zap config.
func (cc *ConfigComparator) generateZapConfigString(config ZapConfig) string {
	var str strings.Builder
	str.WriteString("Zap Configuration:\n")
	str.WriteString(fmt.Sprintf("  Level: %s\n", config.Level))
	str.WriteString(fmt.Sprintf("  Encoding: %s\n", config.Encoding))
	str.WriteString(fmt.Sprintf("  Development: %t\n", config.Development))
	if config.Sampling != nil {
		str.WriteString(fmt.Sprintf("  Sampling: initial=%d, thereafter=%d\n", 
			config.Sampling.Initial, config.Sampling.Thereafter))
	}
	str.WriteString(fmt.Sprintf("  Output Paths: %v\n", config.OutputPaths))
	if len(config.InitialFields) > 0 {
		str.WriteString(fmt.Sprintf("  Initial Fields: %v\n", config.InitialFields))
	}
	return str.String()
}

// generateBoltConfigString creates a string representation of Bolt config.
func (cc *ConfigComparator) generateBoltConfigString(config BoltConfig) string {
	var str strings.Builder
	str.WriteString("Bolt Configuration:\n")
	str.WriteString(fmt.Sprintf("  Level: %s\n", config.Level))
	str.WriteString(fmt.Sprintf("  Format: %s\n", config.Format))
	str.WriteString(fmt.Sprintf("  Development: %t\n", config.Development))
	if len(config.Fields) > 0 {
		str.WriteString(fmt.Sprintf("  Initial Fields: %v\n", config.Fields))
	}
	return str.String()
}

// identifyDifferences identifies key differences between configurations.
func (cc *ConfigComparator) identifyDifferences(zapConfig ZapConfig, boltConfig BoltConfig, report *ComparisonReport) {
	if zapConfig.Encoding != boltConfig.Format {
		report.Differences = append(report.Differences,
			fmt.Sprintf("Field name change: Zap 'encoding' -> Bolt 'format'"))
	}

	if zapConfig.Sampling != nil {
		report.Differences = append(report.Differences,
			"Zap sampling configuration not directly supported in Bolt")
	}

	if len(zapConfig.OutputPaths) > 1 {
		report.Differences = append(report.Differences,
			"Multiple output paths not directly supported in Bolt")
	}

	if zapConfig.EncoderConfig.MessageKey != "message" && zapConfig.EncoderConfig.MessageKey != "" {
		report.Differences = append(report.Differences,
			fmt.Sprintf("Custom message key '%s' -> Bolt uses 'message'", zapConfig.EncoderConfig.MessageKey))
	}
}