package zerolog

import (
	"fmt"
	"go/ast"
	"go/format"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// TransformResult represents the result of a code transformation.
type TransformResult struct {
	FilePath        string   `json:"file_path"`
	Success         bool     `json:"success"`
	Changes         int      `json:"changes"`
	Errors          []string `json:"errors"`
	OriginalCode    string   `json:"original_code,omitempty"`
	TransformedCode string   `json:"transformed_code,omitempty"`
}

// CodeTransformer performs automated code transformations from Zerolog to Bolt.
type CodeTransformer struct {
	fileSet *token.FileSet
	dryRun  bool
}

// NewCodeTransformer creates a new code transformer.
func NewCodeTransformer(dryRun bool) *CodeTransformer {
	return &CodeTransformer{
		fileSet: token.NewFileSet(),
		dryRun:  dryRun,
	}
}

// TransformFile transforms a single Go file from Zerolog to Bolt.
func (ct *CodeTransformer) TransformFile(filePath string) (*TransformResult, error) {
	result := &TransformResult{
		FilePath: filePath,
		Success:  true,
		Changes:  0,
		Errors:   []string{},
	}

	// Read original file
	originalContent, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}
	result.OriginalCode = string(originalContent)

	// Parse the file
	file, err := parser.ParseFile(ct.fileSet, filePath, originalContent, parser.ParseComments)
	if err != nil {
		result.Success = false
		result.Errors = append(result.Errors, fmt.Sprintf("parse error: %v", err))
		return result, nil
	}

	// Track if any changes were made
	changed := false

	// Transform imports
	for _, imp := range file.Imports {
		if imp.Path.Value == `"github.com/rs/zerolog"` {
			imp.Path.Value = `"github.com/felixgeelhaar/bolt"`
			changed = true
			result.Changes++
		}
		if imp.Path.Value == `"github.com/rs/zerolog/log"` {
			imp.Path.Value = `"github.com/felixgeelhaar/bolt"`
			changed = true
			result.Changes++
		}
	}

	// Transform AST nodes
	ast.Inspect(file, func(n ast.Node) bool {
		if ct.transformNode(n, result) {
			changed = true
		}
		return true
	})

	if changed {
		// Format the transformed code
		var buf strings.Builder
		if err := format.Node(&buf, ct.fileSet, file); err != nil {
			result.Success = false
			result.Errors = append(result.Errors, fmt.Sprintf("format error: %v", err))
			return result, nil
		}

		transformedCode := buf.String()
		result.TransformedCode = transformedCode

		// Write the file if not in dry-run mode
		if !ct.dryRun {
			if err := os.WriteFile(filePath, []byte(transformedCode), 0644); err != nil {
				result.Success = false
				result.Errors = append(result.Errors, fmt.Sprintf("write error: %v", err))
				return result, nil
			}
		}
	}

	return result, nil
}

// transformNode transforms individual AST nodes.
func (ct *CodeTransformer) transformNode(n ast.Node, result *TransformResult) bool {
	changed := false

	switch node := n.(type) {
	case *ast.CallExpr:
		if ct.transformCallExpr(node, result) {
			changed = true
		}
	case *ast.SelectorExpr:
		if ct.transformSelectorExpr(node, result) {
			changed = true
		}
	case *ast.Ident:
		if ct.transformIdent(node, result) {
			changed = true
		}
	}

	return changed
}

// transformCallExpr transforms function call expressions.
func (ct *CodeTransformer) transformCallExpr(call *ast.CallExpr, result *TransformResult) bool {
	changed := false

	// Transform zerolog.New() calls
	if selector, ok := call.Fun.(*ast.SelectorExpr); ok {
		if ident, ok := selector.X.(*ast.Ident); ok && ident.Name == "zerolog" {
			switch selector.Sel.Name {
			case "New":
				// Transform zerolog.New(writer) to bolt.New(bolt.NewJSONHandler(writer))
				if len(call.Args) == 1 {
					// Create new call expression: bolt.New(bolt.NewJSONHandler(writer))
					newCall := &ast.CallExpr{
						Fun: &ast.SelectorExpr{
							X:   &ast.Ident{Name: "bolt"},
							Sel: &ast.Ident{Name: "New"},
						},
						Args: []ast.Expr{
							&ast.CallExpr{
								Fun: &ast.SelectorExpr{
									X:   &ast.Ident{Name: "bolt"},
									Sel: &ast.Ident{Name: "NewJSONHandler"},
								},
								Args: call.Args, // Use the original writer argument
							},
						},
					}
					*call = *newCall
					changed = true
					result.Changes++
				}
			case "NewConsoleWriter":
				// Transform to use bolt.NewConsoleHandler
				selector.Sel.Name = "NewConsoleHandler"
				ident.Name = "bolt"
				changed = true
				result.Changes++
			}
		}
	}

	return changed
}

// transformSelectorExpr transforms selector expressions (e.g., logger.Info()).
func (ct *CodeTransformer) transformSelectorExpr(selector *ast.SelectorExpr, result *TransformResult) bool {
	changed := false

	// Transform method calls
	if _, ok := selector.X.(*ast.Ident); ok {
		// Common transformations for method names
		switch selector.Sel.Name {
		case "With":
			// zerolog With() returns Context, but bolt With() returns Event
			// This might need manual review
			result.Errors = append(result.Errors,
				"Warning: With() method behavior differs between zerolog and bolt - manual review needed")
		}
	}

	return changed
}

// transformIdent transforms identifiers.
func (ct *CodeTransformer) transformIdent(ident *ast.Ident, result *TransformResult) bool {
	changed := false

	// Transform package references
	switch ident.Name {
	case "zerolog":
		ident.Name = "bolt"
		changed = true
		result.Changes++
	}

	return changed
}

// TransformDirectory recursively transforms all Go files in a directory.
func (ct *CodeTransformer) TransformDirectory(dirPath string) ([]*TransformResult, error) {
	var results []*TransformResult

	err := filepath.Walk(dirPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip non-Go files and vendor directories
		if !strings.HasSuffix(path, ".go") || strings.Contains(path, "vendor/") {
			return nil
		}

		result, err := ct.TransformFile(path)
		if err != nil {
			return fmt.Errorf("failed to transform %s: %w", path, err)
		}

		results = append(results, result)
		return nil
	})

	return results, err
}

// TextTransformer performs regex-based text transformations for simple cases.
type TextTransformer struct {
	dryRun bool
}

// NewTextTransformer creates a new text transformer.
func NewTextTransformer(dryRun bool) *TextTransformer {
	return &TextTransformer{dryRun: dryRun}
}

// TransformationRule represents a text transformation rule.
type TransformationRule struct {
	Pattern     *regexp.Regexp
	Replacement string
	Description string
}

// GetTransformationRules returns a list of transformation rules for Zerolog to Bolt migration.
func (tt *TextTransformer) GetTransformationRules() []TransformationRule {
	return []TransformationRule{
		{
			Pattern:     regexp.MustCompile(`"github\.com/rs/zerolog"`),
			Replacement: `"github.com/felixgeelhaar/bolt"`,
			Description: "Replace zerolog import with bolt import",
		},
		{
			Pattern:     regexp.MustCompile(`"github\.com/rs/zerolog/log"`),
			Replacement: `"github.com/felixgeelhaar/bolt"`,
			Description: "Replace zerolog/log import with bolt import",
		},
		{
			Pattern:     regexp.MustCompile(`zerolog\.New\(([^)]+)\)`),
			Replacement: `bolt.New(bolt.NewJSONHandler($1))`,
			Description: "Transform zerolog.New() to bolt.New(bolt.NewJSONHandler())",
		},
		{
			Pattern:     regexp.MustCompile(`zerolog\.NewConsoleWriter\(\)`),
			Replacement: `bolt.NewConsoleHandler(os.Stdout)`,
			Description: "Transform zerolog console writer to bolt console handler",
		},
		{
			Pattern:     regexp.MustCompile(`log\.Logger\(\)`),
			Replacement: `bolt.New(bolt.NewJSONHandler(os.Stdout))`,
			Description: "Transform global logger access",
		},
		{
			Pattern:     regexp.MustCompile(`\.Level\(zerolog\.(\w+)Level\)`),
			Replacement: `.SetLevel(bolt.$1)`, // This needs manual mapping
			Description: "Transform level setting (may need manual adjustment)",
		},
	}
}

// TransformFileText performs regex-based transformations on a file.
func (tt *TextTransformer) TransformFileText(filePath string) (*TransformResult, error) {
	result := &TransformResult{
		FilePath: filePath,
		Success:  true,
		Changes:  0,
		Errors:   []string{},
	}

	// Read file
	content, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	result.OriginalCode = string(content)
	transformedContent := string(content)

	rules := tt.GetTransformationRules()
	for _, rule := range rules {
		if rule.Pattern.Match([]byte(transformedContent)) {
			newContent := rule.Pattern.ReplaceAllString(transformedContent, rule.Replacement)
			if newContent != transformedContent {
				result.Changes++
				transformedContent = newContent
			}
		}
	}

	result.TransformedCode = transformedContent

	// Write file if changes were made and not in dry-run mode
	if result.Changes > 0 && !tt.dryRun {
		if err := os.WriteFile(filePath, []byte(transformedContent), 0644); err != nil {
			result.Success = false
			result.Errors = append(result.Errors, fmt.Sprintf("write error: %v", err))
		}
	}

	return result, nil
}

// TransformDirectoryText performs regex-based transformations on all Go files in a directory.
func (tt *TextTransformer) TransformDirectoryText(dirPath string) ([]*TransformResult, error) {
	var results []*TransformResult

	err := filepath.Walk(dirPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if !strings.HasSuffix(path, ".go") || strings.Contains(path, "vendor/") {
			return nil
		}

		result, err := tt.TransformFileText(path)
		if err != nil {
			return fmt.Errorf("failed to transform %s: %w", path, err)
		}

		results = append(results, result)
		return nil
	})

	return results, err
}

// MigrationGuide generates a migration guide based on detected patterns.
type MigrationGuide struct {
	detectedPatterns []DetectedPattern
}

// DetectedPattern represents a code pattern that needs migration.
type DetectedPattern struct {
	Pattern    string `json:"pattern"`
	Suggestion string `json:"suggestion"`
	Automatic  bool   `json:"automatic"`
	FilePath   string `json:"file_path"`
	LineNumber int    `json:"line_number"`
}

// NewMigrationGuide creates a new migration guide generator.
func NewMigrationGuide() *MigrationGuide {
	return &MigrationGuide{
		detectedPatterns: []DetectedPattern{},
	}
}

// AnalyzeFile analyzes a file and detects patterns that need migration.
func (mg *MigrationGuide) AnalyzeFile(filePath string) error {
	content, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("failed to read file: %w", err)
	}

	lines := strings.Split(string(content), "\n")
	for lineNum, line := range lines {
		mg.analyzeLine(line, filePath, lineNum+1)
	}

	return nil
}

// analyzeLine analyzes a single line for migration patterns.
func (mg *MigrationGuide) analyzeLine(line, filePath string, lineNum int) {
	line = strings.TrimSpace(line)

	patterns := []struct {
		regex      *regexp.Regexp
		suggestion string
		automatic  bool
	}{
		{
			regex:      regexp.MustCompile(`zerolog\.New\(`),
			suggestion: "Replace with bolt.New(bolt.NewJSONHandler(writer))",
			automatic:  true,
		},
		{
			regex:      regexp.MustCompile(`\.With\(\)\..*\.Logger\(\)`),
			suggestion: "Zerolog's With().Logger() pattern should be replaced with bolt's With().Logger() - note that behavior may differ",
			automatic:  false,
		},
		{
			regex:      regexp.MustCompile(`\.Level\(zerolog\.\w+Level\)`),
			suggestion: "Replace zerolog levels with bolt levels (TRACE, DEBUG, INFO, WARN, ERROR, FATAL)",
			automatic:  false,
		},
		{
			regex:      regexp.MustCompile(`zerolog\.Hook`),
			suggestion: "Zerolog hooks are not directly supported - consider using bolt's error handlers or custom handlers",
			automatic:  false,
		},
		{
			regex:      regexp.MustCompile(`\.Ctx\(`),
			suggestion: "bolt automatically handles OpenTelemetry context - .Ctx() calls may be redundant",
			automatic:  false,
		},
		{
			regex:      regexp.MustCompile(`zerolog\.ConsoleWriter`),
			suggestion: "Replace with bolt.NewConsoleHandler(os.Stdout)",
			automatic:  true,
		},
	}

	for _, pattern := range patterns {
		if pattern.regex.MatchString(line) {
			mg.detectedPatterns = append(mg.detectedPatterns, DetectedPattern{
				Pattern:    pattern.regex.String(),
				Suggestion: pattern.suggestion,
				Automatic:  pattern.automatic,
				FilePath:   filePath,
				LineNumber: lineNum,
			})
		}
	}
}

// GenerateGuide generates a comprehensive migration guide.
func (mg *MigrationGuide) GenerateGuide() string {
	var guide strings.Builder

	guide.WriteString("# Zerolog to Bolt Migration Guide\n\n")
	guide.WriteString("This guide was automatically generated based on your codebase analysis.\n\n")

	if len(mg.detectedPatterns) == 0 {
		guide.WriteString("No Zerolog patterns detected in the analyzed files.\n")
		return guide.String()
	}

	// Group patterns by automatic vs manual
	automaticPatterns := []DetectedPattern{}
	manualPatterns := []DetectedPattern{}

	for _, pattern := range mg.detectedPatterns {
		if pattern.Automatic {
			automaticPatterns = append(automaticPatterns, pattern)
		} else {
			manualPatterns = append(manualPatterns, pattern)
		}
	}

	// Automatic transformations
	if len(automaticPatterns) > 0 {
		guide.WriteString("## Automatic Transformations\n\n")
		guide.WriteString("These patterns can be automatically transformed using the migration tool:\n\n")

		for _, pattern := range automaticPatterns {
			guide.WriteString(fmt.Sprintf("- **File**: %s:%d\n", pattern.FilePath, pattern.LineNumber))
			guide.WriteString(fmt.Sprintf("  **Suggestion**: %s\n\n", pattern.Suggestion))
		}
	}

	// Manual transformations
	if len(manualPatterns) > 0 {
		guide.WriteString("## Manual Transformations Required\n\n")
		guide.WriteString("These patterns require manual attention:\n\n")

		for _, pattern := range manualPatterns {
			guide.WriteString(fmt.Sprintf("- **File**: %s:%d\n", pattern.FilePath, pattern.LineNumber))
			guide.WriteString(fmt.Sprintf("  **Suggestion**: %s\n\n", pattern.Suggestion))
		}
	}

	// General migration tips
	guide.WriteString("## General Migration Tips\n\n")
	guide.WriteString("1. **Import Changes**: Replace `github.com/rs/zerolog` with `github.com/felixgeelhaar/bolt`\n")
	guide.WriteString("2. **Logger Creation**: Replace `zerolog.New(writer)` with `bolt.New(bolt.NewJSONHandler(writer))`\n")
	guide.WriteString("3. **Console Output**: Replace `zerolog.ConsoleWriter` with `bolt.NewConsoleHandler(os.Stdout)`\n")
	guide.WriteString("4. **Levels**: Map zerolog levels to bolt levels (TraceLevel -> TRACE, etc.)\n")
	guide.WriteString("5. **Context**: bolt handles OpenTelemetry context automatically\n")
	guide.WriteString("6. **Performance**: bolt offers significant performance improvements with zero allocations\n\n")

	guide.WriteString("## Performance Benefits\n\n")
	guide.WriteString("After migration, you can expect:\n")
	guide.WriteString("- 64% faster logging performance\n")
	guide.WriteString("- Zero memory allocations in hot paths\n")
	guide.WriteString("- Sub-100ns latency for logging operations\n\n")

	return guide.String()
}

// SaveGuide saves the migration guide to a file.
func (mg *MigrationGuide) SaveGuide(filePath string) error {
	guide := mg.GenerateGuide()
	return os.WriteFile(filePath, []byte(guide), 0644)
}
