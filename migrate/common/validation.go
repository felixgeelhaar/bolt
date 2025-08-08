package common

import (
	"bufio"
	"encoding/json"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// MigrationValidator provides tools to validate migration success.
type MigrationValidator struct {
	SourceDir string
	fileSet   *token.FileSet
}

// NewMigrationValidator creates a new migration validator.
func NewMigrationValidator(sourceDir string) *MigrationValidator {
	return &MigrationValidator{
		SourceDir: sourceDir,
		fileSet:   token.NewFileSet(),
	}
}

// ValidationResult represents the result of a migration validation.
type ValidationResult struct {
	Success    bool     `json:"success"`
	Errors     []string `json:"errors"`
	Warnings   []string `json:"warnings"`
	FilesCount int      `json:"files_count"`
	Summary    string   `json:"summary"`
}

// ValidateZerologMigration validates that Zerolog code has been properly migrated to Bolt.
func (mv *MigrationValidator) ValidateZerologMigration() *ValidationResult {
	result := &ValidationResult{
		Success:    true,
		Errors:     []string{},
		Warnings:   []string{},
		FilesCount: 0,
	}

	// Patterns to detect unmigrated Zerolog usage
	zerologPatterns := []*regexp.Regexp{
		regexp.MustCompile(`github\.com/rs/zerolog`),
		regexp.MustCompile(`zerolog\.New\(`),
		regexp.MustCompile(`zerolog\.Logger`),
		regexp.MustCompile(`log\.Logger\(\)\.`), // Common Zerolog pattern
	}

	err := filepath.Walk(mv.SourceDir, func(filePath string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if !strings.HasSuffix(filePath, ".go") || strings.Contains(filePath, "vendor/") {
			return nil
		}

		result.FilesCount++
		content, err := os.ReadFile(filePath) // #nosec G304 - Migration tool needs to read user-specified files // #nosec G304 - Migration tool needs to read user-specified files
		if err != nil {
			result.Errors = append(result.Errors, fmt.Sprintf("Failed to read %s: %v", filePath, err))
			return nil
		}

		for _, pattern := range zerologPatterns {
			if matches := pattern.FindAllStringIndex(string(content), -1); len(matches) > 0 {
				result.Success = false
				result.Errors = append(result.Errors,
					fmt.Sprintf("Found unmigrated Zerolog code in %s (pattern: %s)",
						filePath, pattern.String()))
			}
		}

		return nil
	})

	if err != nil {
		result.Success = false
		result.Errors = append(result.Errors, fmt.Sprintf("Walk error: %v", err))
	}

	mv.generateSummary(result, "Zerolog")
	return result
}

// ValidateZapMigration validates that Zap code has been properly migrated to Bolt.
func (mv *MigrationValidator) ValidateZapMigration() *ValidationResult {
	result := &ValidationResult{
		Success:    true,
		Errors:     []string{},
		Warnings:   []string{},
		FilesCount: 0,
	}

	zapPatterns := []*regexp.Regexp{
		regexp.MustCompile(`go\.uber\.org/zap`),
		regexp.MustCompile(`zap\.New\w*\(`),
		regexp.MustCompile(`zap\.Logger`),
		regexp.MustCompile(`\.Sugar\(\)`),
	}

	err := filepath.Walk(mv.SourceDir, func(filePath string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if !strings.HasSuffix(filePath, ".go") || strings.Contains(filePath, "vendor/") {
			return nil
		}

		result.FilesCount++
		content, err := os.ReadFile(filePath) // #nosec G304 - Migration tool needs to read user-specified files // #nosec G304 - Migration tool needs to read user-specified files
		if err != nil {
			result.Errors = append(result.Errors, fmt.Sprintf("Failed to read %s: %v", filePath, err))
			return nil
		}

		for _, pattern := range zapPatterns {
			if matches := pattern.FindAllStringIndex(string(content), -1); len(matches) > 0 {
				result.Success = false
				result.Errors = append(result.Errors,
					fmt.Sprintf("Found unmigrated Zap code in %s (pattern: %s)",
						filePath, pattern.String()))
			}
		}

		return nil
	})

	if err != nil {
		result.Success = false
		result.Errors = append(result.Errors, fmt.Sprintf("Walk error: %v", err))
	}

	mv.generateSummary(result, "Zap")
	return result
}

// ValidateLogrusM igration validates that Logrus code has been properly migrated to Bolt.
func (mv *MigrationValidator) ValidateLogrusMigration() *ValidationResult {
	result := &ValidationResult{
		Success:    true,
		Errors:     []string{},
		Warnings:   []string{},
		FilesCount: 0,
	}

	logrusPatterns := []*regexp.Regexp{
		regexp.MustCompile(`github\.com/sirupsen/logrus`),
		regexp.MustCompile(`logrus\.New\(`),
		regexp.MustCompile(`logrus\.Logger`),
		regexp.MustCompile(`\.WithFields\(`),
		regexp.MustCompile(`logrus\.Fields`),
	}

	err := filepath.Walk(mv.SourceDir, func(filePath string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if !strings.HasSuffix(filePath, ".go") || strings.Contains(filePath, "vendor/") {
			return nil
		}

		result.FilesCount++
		content, err := os.ReadFile(filePath) // #nosec G304 - Migration tool needs to read user-specified files // #nosec G304 - Migration tool needs to read user-specified files
		if err != nil {
			result.Errors = append(result.Errors, fmt.Sprintf("Failed to read %s: %v", filePath, err))
			return nil
		}

		for _, pattern := range logrusPatterns {
			if matches := pattern.FindAllStringIndex(string(content), -1); len(matches) > 0 {
				result.Success = false
				result.Errors = append(result.Errors,
					fmt.Sprintf("Found unmigrated Logrus code in %s (pattern: %s)",
						filePath, pattern.String()))
			}
		}

		return nil
	})

	if err != nil {
		result.Success = false
		result.Errors = append(result.Errors, fmt.Sprintf("Walk error: %v", err))
	}

	mv.generateSummary(result, "Logrus")
	return result
}

// ValidateBoltUsage checks that Bolt is being used correctly after migration.
func (mv *MigrationValidator) ValidateBoltUsage() *ValidationResult {
	result := &ValidationResult{
		Success:    true,
		Errors:     []string{},
		Warnings:   []string{},
		FilesCount: 0,
	}

	err := filepath.Walk(mv.SourceDir, func(filePath string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if !strings.HasSuffix(filePath, ".go") || strings.Contains(filePath, "vendor/") {
			return nil
		}

		result.FilesCount++
		content, err := os.ReadFile(filePath) // #nosec G304 - Migration tool needs to read user-specified files // #nosec G304 - Migration tool needs to read user-specified files
		if err != nil {
			result.Errors = append(result.Errors, fmt.Sprintf("Failed to read %s: %v", filePath, err))
			return nil
		}

		// Check for common Bolt usage patterns
		if strings.Contains(string(content), "github.com/felixgeelhaar/bolt") {
			mv.validateBoltPatterns(filePath, string(content), result)
		}

		return nil
	})

	if err != nil {
		result.Success = false
		result.Errors = append(result.Errors, fmt.Sprintf("Walk error: %v", err))
	}

	mv.generateSummary(result, "Bolt Usage")
	return result
}

// validateBoltPatterns checks for proper Bolt usage patterns.
func (mv *MigrationValidator) validateBoltPatterns(filePath, content string, result *ValidationResult) {
	lines := strings.Split(content, "\n")

	for lineNum, line := range lines {
		line = strings.TrimSpace(line)

		// Check for proper message termination
		if strings.Contains(line, "bolt.") && (strings.Contains(line, ".Str(") ||
			strings.Contains(line, ".Int(") || strings.Contains(line, ".Bool(")) {
			if !strings.Contains(line, ".Msg(") && !strings.Contains(line, ".Send()") &&
				!strings.Contains(line, ".Printf(") {
				result.Warnings = append(result.Warnings,
					fmt.Sprintf("%s:%d: Bolt event chain should end with .Msg(), .Send(), or .Printf()",
						filePath, lineNum+1))
			}
		}

		// Check for proper logger initialization
		if strings.Contains(line, "bolt.New(") && !strings.Contains(line, "bolt.NewJSONHandler") &&
			!strings.Contains(line, "bolt.NewConsoleHandler") {
			result.Warnings = append(result.Warnings,
				fmt.Sprintf("%s:%d: Consider using bolt.NewJSONHandler() or bolt.NewConsoleHandler()",
					filePath, lineNum+1))
		}
	}
}

// LogOutputValidator validates that log output format is correct.
type LogOutputValidator struct {
	expectedFields []string
}

// NewLogOutputValidator creates a new log output validator.
func NewLogOutputValidator(expectedFields []string) *LogOutputValidator {
	return &LogOutputValidator{
		expectedFields: expectedFields,
	}
}

// ValidateJSONOutput validates that JSON log output contains expected fields.
func (lov *LogOutputValidator) ValidateJSONOutput(output io.Reader) *ValidationResult {
	result := &ValidationResult{
		Success: true,
		Errors:  []string{},
	}

	scanner := bufio.NewScanner(output)
	lineNum := 0

	for scanner.Scan() {
		lineNum++
		line := scanner.Text()

		if strings.TrimSpace(line) == "" {
			continue
		}

		var logEntry map[string]interface{}
		if err := json.Unmarshal([]byte(line), &logEntry); err != nil {
			result.Success = false
			result.Errors = append(result.Errors,
				fmt.Sprintf("Line %d: Invalid JSON: %v", lineNum, err))
			continue
		}

		// Check for required fields
		for _, field := range lov.expectedFields {
			if _, exists := logEntry[field]; !exists {
				result.Warnings = append(result.Warnings,
					fmt.Sprintf("Line %d: Missing expected field '%s'", lineNum, field))
			}
		}

		// Check for standard fields
		if level, exists := logEntry["level"]; !exists {
			result.Warnings = append(result.Warnings,
				fmt.Sprintf("Line %d: Missing 'level' field", lineNum))
		} else if levelStr, ok := level.(string); !ok {
			result.Errors = append(result.Errors,
				fmt.Sprintf("Line %d: 'level' field should be string", lineNum))
		} else if !isValidLogLevel(levelStr) {
			result.Warnings = append(result.Warnings,
				fmt.Sprintf("Line %d: Unknown log level '%s'", lineNum, levelStr))
		}

		if _, exists := logEntry["message"]; !exists {
			result.Warnings = append(result.Warnings,
				fmt.Sprintf("Line %d: Missing 'message' field", lineNum))
		}
	}

	if err := scanner.Err(); err != nil {
		result.Success = false
		result.Errors = append(result.Errors, fmt.Sprintf("Scanner error: %v", err))
	}

	lov.generateOutputSummary(result, lineNum)
	return result
}

// isValidLogLevel checks if a log level is valid.
func isValidLogLevel(level string) bool {
	validLevels := []string{"trace", "debug", "info", "warn", "error", "fatal"}
	for _, validLevel := range validLevels {
		if level == validLevel {
			return true
		}
	}
	return false
}

// generateSummary generates a summary for migration validation results.
func (mv *MigrationValidator) generateSummary(result *ValidationResult, migrationType string) {
	if result.Success {
		result.Summary = fmt.Sprintf("%s migration validation passed. Scanned %d files.",
			migrationType, result.FilesCount)
	} else {
		result.Summary = fmt.Sprintf("%s migration validation failed with %d errors and %d warnings. Scanned %d files.",
			migrationType, len(result.Errors), len(result.Warnings), result.FilesCount)
	}
}

// generateOutputSummary generates a summary for output validation results.
func (lov *LogOutputValidator) generateOutputSummary(result *ValidationResult, linesProcessed int) {
	if result.Success && len(result.Warnings) == 0 {
		result.Summary = fmt.Sprintf("Log output validation passed. Processed %d lines.", linesProcessed)
	} else if result.Success {
		result.Summary = fmt.Sprintf("Log output validation passed with %d warnings. Processed %d lines.",
			len(result.Warnings), linesProcessed)
	} else {
		result.Summary = fmt.Sprintf("Log output validation failed with %d errors and %d warnings. Processed %d lines.",
			len(result.Errors), len(result.Warnings), linesProcessed)
	}
}

// CodeAnalyzer performs static analysis on Go code to find migration opportunities.
type CodeAnalyzer struct {
	fileSet *token.FileSet
}

// NewCodeAnalyzer creates a new code analyzer.
func NewCodeAnalyzer() *CodeAnalyzer {
	return &CodeAnalyzer{
		fileSet: token.NewFileSet(),
	}
}

// AnalysisResult contains the results of code analysis.
type AnalysisResult struct {
	FilePath             string            `json:"file_path"`
	LoggingCalls         []LoggingCall     `json:"logging_calls"`
	ImportStatements     []ImportStatement `json:"import_statements"`
	MigrationSuggestions []string          `json:"migration_suggestions"`
}

// LoggingCall represents a logging function call found in code.
type LoggingCall struct {
	Line     int    `json:"line"`
	Column   int    `json:"column"`
	Function string `json:"function"`
	Library  string `json:"library"`
	Code     string `json:"code"`
}

// ImportStatement represents an import statement.
type ImportStatement struct {
	Line int    `json:"line"`
	Path string `json:"path"`
	Name string `json:"name"`
}

// AnalyzeFile analyzes a Go file for logging usage.
func (ca *CodeAnalyzer) AnalyzeFile(filePath string) (*AnalysisResult, error) {
	content, err := os.ReadFile(filePath) // #nosec G304 - Migration tool needs to read user-specified files
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	file, err := parser.ParseFile(ca.fileSet, filePath, content, parser.ParseComments)
	if err != nil {
		return nil, fmt.Errorf("failed to parse file: %w", err)
	}

	result := &AnalysisResult{
		FilePath:             filePath,
		LoggingCalls:         []LoggingCall{},
		ImportStatements:     []ImportStatement{},
		MigrationSuggestions: []string{},
	}

	// Analyze imports
	for _, imp := range file.Imports {
		importPath := strings.Trim(imp.Path.Value, `"`)
		position := ca.fileSet.Position(imp.Pos())

		importStmt := ImportStatement{
			Line: position.Line,
			Path: importPath,
		}

		if imp.Name != nil {
			importStmt.Name = imp.Name.Name
		}

		result.ImportStatements = append(result.ImportStatements, importStmt)

		// Generate migration suggestions based on imports
		ca.generateImportSuggestions(importPath, result)
	}

	// Walk the AST to find logging calls
	ast.Inspect(file, func(n ast.Node) bool {
		ca.inspectNode(n, result, content)
		return true
	})

	return result, nil
}

// inspectNode inspects an AST node for logging patterns.
func (ca *CodeAnalyzer) inspectNode(n ast.Node, result *AnalysisResult, content []byte) {
	call, ok := n.(*ast.CallExpr)
	if !ok {
		return
	}

	position := ca.fileSet.Position(call.Pos())

	// Extract the source code for this call
	start := ca.fileSet.Position(call.Pos()).Offset
	end := ca.fileSet.Position(call.End()).Offset
	if start >= 0 && end <= len(content) && end > start {
		code := string(content[start:end])

		// Detect logging library calls
		if selector, ok := call.Fun.(*ast.SelectorExpr); ok {
			if ident, ok := selector.X.(*ast.Ident); ok {
				library := ca.identifyLibrary(ident.Name, selector.Sel.Name)
				if library != "" {
					result.LoggingCalls = append(result.LoggingCalls, LoggingCall{
						Line:     position.Line,
						Column:   position.Column,
						Function: selector.Sel.Name,
						Library:  library,
						Code:     code,
					})
				}
			}
		}
	}
}

// identifyLibrary identifies which logging library a call belongs to.
func (ca *CodeAnalyzer) identifyLibrary(receiver, method string) string {
	// Common patterns for different libraries
	zapPatterns := []string{"Info", "Debug", "Error", "Warn", "Fatal", "Panic"}
	zerologPatterns := []string{"Info", "Debug", "Error", "Warn", "Fatal", "Trace"}
	logrusPatterns := []string{"Info", "Debug", "Error", "Warn", "Fatal", "WithFields"}

	for _, pattern := range zapPatterns {
		if method == pattern && (receiver == "logger" || strings.Contains(receiver, "zap")) {
			return "zap"
		}
	}

	for _, pattern := range zerologPatterns {
		if method == pattern && (receiver == "logger" || strings.Contains(receiver, "log")) {
			return "zerolog"
		}
	}

	for _, pattern := range logrusPatterns {
		if method == pattern && (receiver == "logger" || receiver == "log") {
			return "logrus"
		}
	}

	return ""
}

// generateImportSuggestions generates migration suggestions based on import statements.
func (ca *CodeAnalyzer) generateImportSuggestions(importPath string, result *AnalysisResult) {
	switch {
	case strings.Contains(importPath, "github.com/rs/zerolog"):
		result.MigrationSuggestions = append(result.MigrationSuggestions,
			"Replace 'github.com/rs/zerolog' with 'github.com/felixgeelhaar/bolt'")
		result.MigrationSuggestions = append(result.MigrationSuggestions,
			"Use bolt.New(bolt.NewJSONHandler(os.Stdout)) instead of zerolog.New(os.Stdout)")
	case strings.Contains(importPath, "go.uber.org/zap"):
		result.MigrationSuggestions = append(result.MigrationSuggestions,
			"Replace 'go.uber.org/zap' with 'github.com/felixgeelhaar/bolt'")
		result.MigrationSuggestions = append(result.MigrationSuggestions,
			"Use bolt.New(bolt.NewJSONHandler(os.Stdout)) instead of zap.NewProduction()")
	case strings.Contains(importPath, "github.com/sirupsen/logrus"):
		result.MigrationSuggestions = append(result.MigrationSuggestions,
			"Replace 'github.com/sirupsen/logrus' with 'github.com/felixgeelhaar/bolt'")
		result.MigrationSuggestions = append(result.MigrationSuggestions,
			"Use bolt.New(bolt.NewJSONHandler(os.Stdout)) instead of logrus.New()")
	}
}
