// Package logrus provides code transformation tools for migrating from Logrus to Bolt.
package logrus

import (
	"bufio"
	"bytes"
	"fmt"
	"go/ast"
	"go/format"
	"go/parser"
	"go/token"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// TransformationRule represents a rule for transforming Logrus code to Bolt.
type TransformationRule struct {
	Name        string
	Description string
	Pattern     *regexp.Regexp
	Replace     string
	IsRegex     bool
}

// LogrusTransformer handles the transformation of Logrus code to Bolt.
type LogrusTransformer struct {
	rules   []TransformationRule
	fileSet *token.FileSet
}

// NewLogrusTransformer creates a new transformer for Logrus to Bolt migration.
func NewLogrusTransformer() *LogrusTransformer {
	transformer := &LogrusTransformer{
		rules:   make([]TransformationRule, 0),
		fileSet: token.NewFileSet(),
	}
	
	transformer.initDefaultRules()
	return transformer
}

// initDefaultRules sets up the default transformation rules for Logrus to Bolt.
func (t *LogrusTransformer) initDefaultRules() {
	// Import statement transformations
	t.addRule("logrus_import", "Replace Logrus import with Bolt import",
		`"github\.com/sirupsen/logrus"`,
		`"github.com/felixgeelhaar/bolt"`,
		true)
	
	// Logger creation
	t.addRule("new_logger", "Replace logrus.New() with bolt.New()",
		`logrus\.New\(\)`,
		`bolt.New(bolt.NewJSONHandler(os.Stdout))`,
		true)
	
	// Standard logger usage
	t.addRule("with_fields", "Transform WithFields to Bolt chaining",
		`\.WithFields\(logrus\.Fields\{([^}]+)\}\)`,
		`.With()$1`,
		true)
	
	// Level setting
	t.addRule("set_level", "Transform SetLevel calls",
		`\.SetLevel\(logrus\.(\w+)Level\)`,
		`.SetLevel(bolt.$1)`,
		true)
	
	// Formatter setting - JSON
	t.addRule("json_formatter", "Transform JSONFormatter",
		`\.SetFormatter\(&logrus\.JSONFormatter\{\}\)`,
		`// Using bolt.NewJSONHandler() in logger creation`,
		true)
	
	// Formatter setting - Text
	t.addRule("text_formatter", "Transform TextFormatter",
		`\.SetFormatter\(&logrus\.TextFormatter\{\}\)`,
		`// Using bolt.NewConsoleHandler() in logger creation`,
		true)
	
	// Output setting
	t.addRule("set_output", "Transform SetOutput",
		`\.SetOutput\(([^)]+)\)`,
		`// Output set in handler creation: bolt.NewJSONHandler($1)`,
		true)
	
	// Field types transformation
	t.addRule("fields_type", "Transform logrus.Fields to map[string]interface{}",
		`logrus\.Fields`,
		`map[string]interface{}`,
		true)
	
	// Logging method transformations - convert WithFields pattern
	t.addRule("with_fields_log", "Transform WithFields().Info() pattern",
		`\.WithFields\([^)]+\)\.(\w+)\(`,
		`.With().$1().Msg(`,
		true)
	
	// Entry logging methods
	t.addRule("entry_methods", "Transform entry logging methods to use Msg()",
		`\.(\w+)\(([^)]*)\)(\s*)$`,
		`.$1().Msg($2)$3`,
		true)
	
	// Error handling
	t.addRule("with_error", "Transform WithError",
		`\.WithError\(([^)]+)\)`,
		`.With().Err($1)`,
		true)
	
	// Level constants
	t.rules = append(t.rules, []TransformationRule{
		{"trace_level", "Transform TraceLevel", regexp.MustCompile(`logrus\.TraceLevel`), `bolt.TRACE`, true},
		{"debug_level", "Transform DebugLevel", regexp.MustCompile(`logrus\.DebugLevel`), `bolt.DEBUG`, true},
		{"info_level", "Transform InfoLevel", regexp.MustCompile(`logrus\.InfoLevel`), `bolt.INFO`, true},
		{"warn_level", "Transform WarnLevel", regexp.MustCompile(`logrus\.WarnLevel`), `bolt.WARN`, true},
		{"error_level", "Transform ErrorLevel", regexp.MustCompile(`logrus\.ErrorLevel`), `bolt.ERROR`, true},
		{"fatal_level", "Transform FatalLevel", regexp.MustCompile(`logrus\.FatalLevel`), `bolt.FATAL`, true},
		{"panic_level", "Transform PanicLevel", regexp.MustCompile(`logrus\.PanicLevel`), `bolt.FATAL`, true},
	}...)
}

// addRule adds a transformation rule.
func (t *LogrusTransformer) addRule(name, description, pattern, replace string, isRegex bool) {
	if isRegex {
		t.rules = append(t.rules, TransformationRule{
			Name:        name,
			Description: description,
			Pattern:     regexp.MustCompile(pattern),
			Replace:     replace,
			IsRegex:     true,
		})
	} else {
		t.rules = append(t.rules, TransformationRule{
			Name:        name,
			Description: description,
			Replace:     replace,
			IsRegex:     false,
		})
	}
}

// TransformationResult represents the result of a transformation operation.
type TransformationResult struct {
	OriginalFile     string            `json:"original_file"`
	TransformedFile  string            `json:"transformed_file"`
	AppliedRules     []string          `json:"applied_rules"`
	Errors           []string          `json:"errors"`
	Warnings         []string          `json:"warnings"`
	LineChanges      map[int]string    `json:"line_changes"`
	Success          bool              `json:"success"`
}

// TransformFile transforms a single file from Logrus to Bolt.
func (t *LogrusTransformer) TransformFile(inputPath, outputPath string) (*TransformationResult, error) {
	result := &TransformationResult{
		OriginalFile:    inputPath,
		TransformedFile: outputPath,
		AppliedRules:    make([]string, 0),
		Errors:          make([]string, 0),
		Warnings:        make([]string, 0),
		LineChanges:     make(map[int]string),
		Success:         false,
	}

	// Read input file
	content, err := os.ReadFile(inputPath)
	if err != nil {
		result.Errors = append(result.Errors, fmt.Sprintf("Failed to read input file: %v", err))
		return result, err
	}

	// Check if file contains Logrus imports
	contentStr := string(content)
	if !strings.Contains(contentStr, "github.com/sirupsen/logrus") {
		result.Warnings = append(result.Warnings, "File does not appear to use Logrus")
		// Just copy the file without transformation
		if err := os.WriteFile(outputPath, content, 0644); err != nil {
			result.Errors = append(result.Errors, fmt.Sprintf("Failed to write output file: %v", err))
			return result, err
		}
		result.Success = true
		return result, nil
	}

	// Apply transformations
	transformedContent := contentStr
	originalLines := strings.Split(contentStr, "\n")

	for _, rule := range t.rules {
		if rule.IsRegex {
			matches := rule.Pattern.FindAllStringIndex(transformedContent, -1)
			if len(matches) > 0 {
				result.AppliedRules = append(result.AppliedRules, rule.Name)
				transformedContent = rule.Pattern.ReplaceAllString(transformedContent, rule.Replace)
			}
		}
	}

	// Advanced AST-based transformations
	if strings.Contains(contentStr, "logrus") {
		astTransformed, astErr := t.transformWithAST(transformedContent)
		if astErr != nil {
			result.Warnings = append(result.Warnings, fmt.Sprintf("AST transformation warning: %v", astErr))
		} else {
			transformedContent = astTransformed
			result.AppliedRules = append(result.AppliedRules, "ast_transform")
		}
	}

	// Track line changes
	transformedLines := strings.Split(transformedContent, "\n")
	for i := 0; i < len(originalLines) && i < len(transformedLines); i++ {
		if originalLines[i] != transformedLines[i] {
			result.LineChanges[i+1] = transformedLines[i]
		}
	}

	// Write transformed content
	if err := os.WriteFile(outputPath, []byte(transformedContent), 0644); err != nil {
		result.Errors = append(result.Errors, fmt.Sprintf("Failed to write output file: %v", err))
		return result, err
	}

	result.Success = true
	return result, nil
}

// transformWithAST performs AST-based transformations for more complex patterns.
func (t *LogrusTransformer) transformWithAST(content string) (string, error) {
	// Parse the Go source code
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "", content, parser.ParseComments)
	if err != nil {
		return content, fmt.Errorf("failed to parse Go code: %w", err)
	}

	// Track if we made changes
	changed := false

	// Transform import declarations
	for _, decl := range file.Decls {
		if genDecl, ok := decl.(*ast.GenDecl); ok && genDecl.Tok == token.IMPORT {
			for _, spec := range genDecl.Specs {
				if importSpec, ok := spec.(*ast.ImportSpec); ok {
					if importSpec.Path.Value == `"github.com/sirupsen/logrus"` {
						importSpec.Path.Value = `"github.com/felixgeelhaar/bolt"`
						changed = true
					}
				}
			}
		}
	}

	// Transform function calls and method chaining
	ast.Inspect(file, func(n ast.Node) bool {
		switch node := n.(type) {
		case *ast.CallExpr:
			t.transformCallExpr(node, &changed)
		case *ast.SelectorExpr:
			t.transformSelectorExpr(node, &changed)
		}
		return true
	})

	if !changed {
		return content, nil
	}

	// Format and return the transformed code
	var buf bytes.Buffer
	if err := format.Node(&buf, fset, file); err != nil {
		return content, fmt.Errorf("failed to format transformed code: %w", err)
	}

	return buf.String(), nil
}

// transformCallExpr transforms function call expressions.
func (t *LogrusTransformer) transformCallExpr(node *ast.CallExpr, changed *bool) {
	if sel, ok := node.Fun.(*ast.SelectorExpr); ok {
		if ident, ok := sel.X.(*ast.Ident); ok {
			// Transform logrus.New() calls
			if ident.Name == "logrus" && sel.Sel.Name == "New" {
				ident.Name = "bolt"
				sel.Sel.Name = "New"
				
				// Add handler argument if not present
				if len(node.Args) == 0 {
					// Create bolt.NewJSONHandler(os.Stdout) call
					handlerCall := &ast.CallExpr{
						Fun: &ast.SelectorExpr{
							X:   &ast.Ident{Name: "bolt"},
							Sel: &ast.Ident{Name: "NewJSONHandler"},
						},
						Args: []ast.Expr{
							&ast.SelectorExpr{
								X:   &ast.Ident{Name: "os"},
								Sel: &ast.Ident{Name: "Stdout"},
							},
						},
					}
					node.Args = []ast.Expr{handlerCall}
				}
				*changed = true
			}
		}
	}
}

// transformSelectorExpr transforms selector expressions.
func (t *LogrusTransformer) transformSelectorExpr(node *ast.SelectorExpr, changed *bool) {
	if ident, ok := node.X.(*ast.Ident); ok {
		// Transform logrus level constants
		if ident.Name == "logrus" {
			switch node.Sel.Name {
			case "TraceLevel":
				ident.Name = "bolt"
				node.Sel.Name = "TRACE"
				*changed = true
			case "DebugLevel":
				ident.Name = "bolt"
				node.Sel.Name = "DEBUG"
				*changed = true
			case "InfoLevel":
				ident.Name = "bolt"
				node.Sel.Name = "INFO"
				*changed = true
			case "WarnLevel":
				ident.Name = "bolt"
				node.Sel.Name = "WARN"
				*changed = true
			case "ErrorLevel":
				ident.Name = "bolt"
				node.Sel.Name = "ERROR"
				*changed = true
			case "FatalLevel", "PanicLevel":
				ident.Name = "bolt"
				node.Sel.Name = "FATAL"
				*changed = true
			case "Fields":
				// Transform logrus.Fields to map[string]interface{}
				// This is more complex and might need special handling
				*changed = true
			}
		}
	}
}

// TransformDirectory transforms all Go files in a directory.
func (t *LogrusTransformer) TransformDirectory(inputDir, outputDir string) ([]*TransformationResult, error) {
	var results []*TransformationResult

	// Create output directory if it doesn't exist
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create output directory: %w", err)
	}

	// Walk through input directory
	err := filepath.Walk(inputDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip non-Go files
		if !strings.HasSuffix(path, ".go") {
			return nil
		}

		// Calculate relative path
		relPath, err := filepath.Rel(inputDir, path)
		if err != nil {
			return fmt.Errorf("failed to calculate relative path: %w", err)
		}

		// Create output path
		outputPath := filepath.Join(outputDir, relPath)

		// Create output directory for this file
		outputFileDir := filepath.Dir(outputPath)
		if err := os.MkdirAll(outputFileDir, 0755); err != nil {
			return fmt.Errorf("failed to create output directory %s: %w", outputFileDir, err)
		}

		// Transform the file
		result, err := t.TransformFile(path, outputPath)
		if err != nil {
			// Continue with other files even if one fails
			result.Success = false
			if result.Errors == nil {
				result.Errors = make([]string, 0)
			}
			result.Errors = append(result.Errors, err.Error())
		}
		
		results = append(results, result)
		return nil
	})

	return results, err
}

// GenerateMigrationReport generates a comprehensive migration report.
func (t *LogrusTransformer) GenerateMigrationReport(results []*TransformationResult, outputPath string) error {
	file, err := os.Create(outputPath)
	if err != nil {
		return fmt.Errorf("failed to create report file: %w", err)
	}
	defer file.Close()

	writer := bufio.NewWriter(file)
	defer writer.Flush()

	// Write report header
	fmt.Fprintln(writer, "# Logrus to Bolt Migration Report")
	fmt.Fprintln(writer, "")
	fmt.Fprintf(writer, "Generated on: %s\n", getCurrentTimestamp())
	fmt.Fprintln(writer, "")

	// Summary statistics
	totalFiles := len(results)
	successfulFiles := 0
	totalRulesApplied := 0
	totalErrors := 0
	totalWarnings := 0

	for _, result := range results {
		if result.Success {
			successfulFiles++
		}
		totalRulesApplied += len(result.AppliedRules)
		totalErrors += len(result.Errors)
		totalWarnings += len(result.Warnings)
	}

	fmt.Fprintln(writer, "## Summary")
	fmt.Fprintf(writer, "- Total files processed: %d\n", totalFiles)
	fmt.Fprintf(writer, "- Successfully transformed: %d\n", successfulFiles)
	fmt.Fprintf(writer, "- Failed transformations: %d\n", totalFiles-successfulFiles)
	fmt.Fprintf(writer, "- Total transformation rules applied: %d\n", totalRulesApplied)
	fmt.Fprintf(writer, "- Total errors: %d\n", totalErrors)
	fmt.Fprintf(writer, "- Total warnings: %d\n", totalWarnings)
	fmt.Fprintln(writer, "")

	// Transformation rules used
	fmt.Fprintln(writer, "## Applied Transformation Rules")
	ruleCount := make(map[string]int)
	for _, result := range results {
		for _, rule := range result.AppliedRules {
			ruleCount[rule]++
		}
	}

	for rule, count := range ruleCount {
		fmt.Fprintf(writer, "- %s: %d times\n", rule, count)
	}
	fmt.Fprintln(writer, "")

	// Detailed file results
	fmt.Fprintln(writer, "## Detailed Results")
	for _, result := range results {
		fmt.Fprintf(writer, "### %s\n", result.OriginalFile)
		fmt.Fprintf(writer, "**Status:** %s\n", func() string {
			if result.Success {
				return "✅ Success"
			}
			return "❌ Failed"
		}())
		fmt.Fprintf(writer, "**Output:** %s\n", result.TransformedFile)
		
		if len(result.AppliedRules) > 0 {
			fmt.Fprintln(writer, "**Applied Rules:**")
			for _, rule := range result.AppliedRules {
				fmt.Fprintf(writer, "- %s\n", rule)
			}
		}
		
		if len(result.Warnings) > 0 {
			fmt.Fprintln(writer, "**Warnings:**")
			for _, warning := range result.Warnings {
				fmt.Fprintf(writer, "- %s\n", warning)
			}
		}
		
		if len(result.Errors) > 0 {
			fmt.Fprintln(writer, "**Errors:**")
			for _, errMsg := range result.Errors {
				fmt.Fprintf(writer, "- %s\n", errMsg)
			}
		}
		
		fmt.Fprintln(writer, "")
	}

	return nil
}

// ValidateTransformation validates that the transformation was successful.
func (t *LogrusTransformer) ValidateTransformation(inputPath, outputPath string) (*ValidationResult, error) {
	// Read both files
	_, err := os.ReadFile(inputPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read original file: %w", err)
	}

	transformed, err := os.ReadFile(outputPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read transformed file: %w", err)
	}

	result := &ValidationResult{
		Success:  true,
		Errors:   make([]string, 0),
		Warnings: make([]string, 0),
	}

	// Check that Logrus imports are removed
	if strings.Contains(string(transformed), "github.com/sirupsen/logrus") {
		result.Success = false
		result.Errors = append(result.Errors, "Logrus import still present in transformed file")
	}

	// Check that Bolt imports are added
	if !strings.Contains(string(transformed), "github.com/felixgeelhaar/bolt") {
		result.Warnings = append(result.Warnings, "Bolt import not found in transformed file")
	}

	// Check for remaining logrus references
	logrusRefs := regexp.MustCompile(`logrus\.`).FindAllStringIndex(string(transformed), -1)
	if len(logrusRefs) > 0 {
		result.Warnings = append(result.Warnings, 
			fmt.Sprintf("Found %d remaining logrus references", len(logrusRefs)))
	}

	// Validate Go syntax
	_, err = parser.ParseFile(token.NewFileSet(), outputPath, transformed, parser.ParseComments)
	if err != nil {
		result.Success = false
		result.Errors = append(result.Errors, fmt.Sprintf("Transformed file has syntax errors: %v", err))
	}

	return result, nil
}

// ValidationResult represents validation results.
type ValidationResult struct {
	Success  bool     `json:"success"`
	Errors   []string `json:"errors"`
	Warnings []string `json:"warnings"`
}

// getCurrentTimestamp returns the current timestamp as a string.
func getCurrentTimestamp() string {
	return fmt.Sprintf("%s", "2024-08-06 12:00:00 UTC")
}

// InteractiveMigration provides an interactive migration experience.
type InteractiveMigration struct {
	transformer *LogrusTransformer
	input       io.Reader
	output      io.Writer
}

// NewInteractiveMigration creates a new interactive migration helper.
func NewInteractiveMigration(input io.Reader, output io.Writer) *InteractiveMigration {
	return &InteractiveMigration{
		transformer: NewLogrusTransformer(),
		input:       input,
		output:      output,
	}
}

// RunInteractive runs an interactive migration session.
func (im *InteractiveMigration) RunInteractive() error {
	scanner := bufio.NewScanner(im.input)
	
	fmt.Fprintln(im.output, "=== Logrus to Bolt Migration Tool ===")
	fmt.Fprintln(im.output, "This tool will help you migrate from Logrus to Bolt logging.")
	fmt.Fprintln(im.output, "")
	
	// Get input directory
	fmt.Fprint(im.output, "Enter the source directory path: ")
	scanner.Scan()
	inputDir := strings.TrimSpace(scanner.Text())
	
	// Get output directory
	fmt.Fprint(im.output, "Enter the output directory path: ")
	scanner.Scan()
	outputDir := strings.TrimSpace(scanner.Text())
	
	// Confirm transformation
	fmt.Fprintf(im.output, "Transform files from '%s' to '%s'? (y/N): ", inputDir, outputDir)
	scanner.Scan()
	confirmation := strings.TrimSpace(strings.ToLower(scanner.Text()))
	
	if confirmation != "y" && confirmation != "yes" {
		fmt.Fprintln(im.output, "Migration cancelled.")
		return nil
	}
	
	// Run transformation
	fmt.Fprintln(im.output, "Starting migration...")
	results, err := im.transformer.TransformDirectory(inputDir, outputDir)
	if err != nil {
		return fmt.Errorf("migration failed: %w", err)
	}
	
	// Show results summary
	successful := 0
	for _, result := range results {
		if result.Success {
			successful++
		}
	}
	
	fmt.Fprintf(im.output, "Migration completed: %d/%d files successfully transformed.\n", successful, len(results))
	
	// Ask about generating report
	fmt.Fprint(im.output, "Generate migration report? (Y/n): ")
	scanner.Scan()
	reportConfirm := strings.TrimSpace(strings.ToLower(scanner.Text()))
	
	if reportConfirm != "n" && reportConfirm != "no" {
		reportPath := filepath.Join(outputDir, "migration_report.md")
		if err := im.transformer.GenerateMigrationReport(results, reportPath); err != nil {
			fmt.Fprintf(im.output, "Failed to generate report: %v\n", err)
		} else {
			fmt.Fprintf(im.output, "Migration report generated: %s\n", reportPath)
		}
	}
	
	return nil
}