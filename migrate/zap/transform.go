package zap

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

// ZapTransformResult represents the result of a Zap to Bolt transformation.
type ZapTransformResult struct {
	FilePath        string   `json:"file_path"`
	Success         bool     `json:"success"`
	Changes         int      `json:"changes"`
	Errors          []string `json:"errors"`
	Warnings        []string `json:"warnings"`
	OriginalCode    string   `json:"original_code,omitempty"`
	TransformedCode string   `json:"transformed_code,omitempty"`
}

// ZapCodeTransformer performs automated code transformations from Zap to Bolt.
type ZapCodeTransformer struct {
	fileSet *token.FileSet
	dryRun  bool
}

// NewZapCodeTransformer creates a new Zap code transformer.
func NewZapCodeTransformer(dryRun bool) *ZapCodeTransformer {
	return &ZapCodeTransformer{
		fileSet: token.NewFileSet(),
		dryRun:  dryRun,
	}
}

// TransformFile transforms a single Go file from Zap to Bolt.
func (zct *ZapCodeTransformer) TransformFile(filePath string) (*ZapTransformResult, error) {
	result := &ZapTransformResult{
		FilePath: filePath,
		Success:  true,
		Changes:  0,
		Errors:   []string{},
		Warnings: []string{},
	}

	// Read original file
	originalContent, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}
	result.OriginalCode = string(originalContent)

	// Parse the file
	file, err := parser.ParseFile(zct.fileSet, filePath, originalContent, parser.ParseComments)
	if err != nil {
		result.Success = false
		result.Errors = append(result.Errors, fmt.Sprintf("parse error: %v", err))
		return result, nil
	}

	// Track if any changes were made
	changed := false

	// Transform imports
	for _, imp := range file.Imports {
		if zct.transformImport(imp, result) {
			changed = true
		}
	}

	// Transform AST nodes
	ast.Inspect(file, func(n ast.Node) bool {
		if zct.transformNode(n, result) {
			changed = true
		}
		return true
	})

	if changed {
		// Format the transformed code
		var buf strings.Builder
		if err := format.Node(&buf, zct.fileSet, file); err != nil {
			result.Success = false
			result.Errors = append(result.Errors, fmt.Sprintf("format error: %v", err))
			return result, nil
		}

		transformedCode := buf.String()
		result.TransformedCode = transformedCode

		// Write the file if not in dry-run mode
		if !zct.dryRun {
			if err := os.WriteFile(filePath, []byte(transformedCode), 0644); err != nil {
				result.Success = false
				result.Errors = append(result.Errors, fmt.Sprintf("write error: %v", err))
				return result, nil
			}
		}
	}

	return result, nil
}

// transformImport transforms import statements.
func (zct *ZapCodeTransformer) transformImport(imp *ast.ImportSpec, result *ZapTransformResult) bool {
	changed := false

	switch imp.Path.Value {
	case `"go.uber.org/zap"`:
		imp.Path.Value = `"github.com/felixgeelhaar/bolt/v2"`
		changed = true
		result.Changes++
	case `"go.uber.org/zap/zapcore"`:
		// zapcore might need special handling
		result.Warnings = append(result.Warnings,
			"zapcore import detected - manual review needed for core configuration")
		imp.Path.Value = `"github.com/felixgeelhaar/bolt/v2"`
		changed = true
		result.Changes++
	}

	return changed
}

// transformNode transforms individual AST nodes.
func (zct *ZapCodeTransformer) transformNode(n ast.Node, result *ZapTransformResult) bool {
	changed := false

	switch node := n.(type) {
	case *ast.CallExpr:
		if zct.transformCallExpr(node, result) {
			changed = true
		}
	case *ast.SelectorExpr:
		if zct.transformSelectorExpr(node, result) {
			changed = true
		}
	case *ast.Ident:
		if zct.transformIdent(node, result) {
			changed = true
		}
	}

	return changed
}

// transformCallExpr transforms function call expressions.
func (zct *ZapCodeTransformer) transformCallExpr(call *ast.CallExpr, result *ZapTransformResult) bool {
	changed := false

	// Transform zap function calls
	if selector, ok := call.Fun.(*ast.SelectorExpr); ok {
		if ident, ok := selector.X.(*ast.Ident); ok && ident.Name == "zap" {
			switch selector.Sel.Name {
			case "NewProduction":
				// Transform zap.NewProduction() to bolt.New(bolt.NewJSONHandler(os.Stdout))
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
							Args: []ast.Expr{
								&ast.SelectorExpr{
									X:   &ast.Ident{Name: "os"},
									Sel: &ast.Ident{Name: "Stdout"},
								},
							},
						},
					},
				}
				*call = *newCall
				changed = true
				result.Changes++

			case "NewDevelopment":
				// Transform zap.NewDevelopment() to bolt.New(bolt.NewConsoleHandler(os.Stdout))
				newCall := &ast.CallExpr{
					Fun: &ast.SelectorExpr{
						X:   &ast.Ident{Name: "bolt"},
						Sel: &ast.Ident{Name: "New"},
					},
					Args: []ast.Expr{
						&ast.CallExpr{
							Fun: &ast.SelectorExpr{
								X:   &ast.Ident{Name: "bolt"},
								Sel: &ast.Ident{Name: "NewConsoleHandler"},
							},
							Args: []ast.Expr{
								&ast.SelectorExpr{
									X:   &ast.Ident{Name: "os"},
									Sel: &ast.Ident{Name: "Stdout"},
								},
							},
						},
					},
				}
				*call = *newCall
				changed = true
				result.Changes++

			case "NewExample":
				// Transform to development setup
				newCall := &ast.CallExpr{
					Fun: &ast.SelectorExpr{
						X:   &ast.Ident{Name: "bolt"},
						Sel: &ast.Ident{Name: "New"},
					},
					Args: []ast.Expr{
						&ast.CallExpr{
							Fun: &ast.SelectorExpr{
								X:   &ast.Ident{Name: "bolt"},
								Sel: &ast.Ident{Name: "NewConsoleHandler"},
							},
							Args: []ast.Expr{
								&ast.SelectorExpr{
									X:   &ast.Ident{Name: "os"},
									Sel: &ast.Ident{Name: "Stdout"},
								},
							},
						},
					},
				}
				*call = *newCall
				changed = true
				result.Changes++

			case "String", "Int", "Int64", "Uint", "Uint64", "Bool", "Float64", "Time", "Duration", "Error", "Any":
				// These are field constructors that need to be converted to method calls
				result.Warnings = append(result.Warnings,
					fmt.Sprintf("zap.%s field constructor detected - needs manual conversion to method call", selector.Sel.Name))
			}
		}
	}

	return changed
}

// transformSelectorExpr transforms selector expressions.
func (zct *ZapCodeTransformer) transformSelectorExpr(selector *ast.SelectorExpr, result *ZapTransformResult) bool {
	changed := false

	// Transform method calls
	if ident, ok := selector.X.(*ast.Ident); ok {
		switch selector.Sel.Name {
		case "Sugar":
			// Zap's Sugar() method doesn't have a direct equivalent in Bolt
			result.Warnings = append(result.Warnings,
				"Sugar() method detected - consider using Bolt's direct API instead")
		case "With":
			// With() behavior is similar but may need adjustment
			result.Warnings = append(result.Warnings,
				"With() method detected - verify field handling matches Bolt's With().Logger() pattern")
		case "Sync":
			// Bolt doesn't need explicit syncing
			result.Warnings = append(result.Warnings,
				"Sync() method detected - not needed in Bolt, consider removing")
		}
	}

	return changed
}

// transformIdent transforms identifiers.
func (zct *ZapCodeTransformer) transformIdent(ident *ast.Ident, result *ZapTransformResult) bool {
	changed := false

	// Transform package references
	switch ident.Name {
	case "zap":
		ident.Name = "bolt"
		changed = true
		result.Changes++
	}

	return changed
}

// ZapTextTransformer performs regex-based text transformations for simple cases.
type ZapTextTransformer struct {
	dryRun bool
}

// NewZapTextTransformer creates a new Zap text transformer.
func NewZapTextTransformer(dryRun bool) *ZapTextTransformer {
	return &ZapTextTransformer{dryRun: dryRun}
}

// ZapTransformationRule represents a text transformation rule.
type ZapTransformationRule struct {
	Pattern     *regexp.Regexp
	Replacement string
	Description string
	Warning     string
}

// GetZapTransformationRules returns transformation rules for Zap to Bolt migration.
func (ztt *ZapTextTransformer) GetZapTransformationRules() []ZapTransformationRule {
	return []ZapTransformationRule{
		{
			Pattern:     regexp.MustCompile(`"go\.uber\.org/zap"`),
			Replacement: `"github.com/felixgeelhaar/bolt/v2"`,
			Description: "Replace zap import with bolt import",
		},
		{
			Pattern:     regexp.MustCompile(`"go\.uber\.org/zap/zapcore"`),
			Replacement: `"github.com/felixgeelhaar/bolt/v2"`,
			Description: "Replace zapcore import with bolt import",
			Warning:     "zapcore usage may need manual adjustment",
		},
		{
			Pattern:     regexp.MustCompile(`zap\.NewProduction\(\)`),
			Replacement: `bolt.New(bolt.NewJSONHandler(os.Stdout)).SetLevel(bolt.INFO)`,
			Description: "Transform zap.NewProduction() to bolt equivalent",
		},
		{
			Pattern:     regexp.MustCompile(`zap\.NewDevelopment\(\)`),
			Replacement: `bolt.New(bolt.NewConsoleHandler(os.Stdout)).SetLevel(bolt.DEBUG)`,
			Description: "Transform zap.NewDevelopment() to bolt equivalent",
		},
		{
			Pattern:     regexp.MustCompile(`zap\.NewExample\(\)`),
			Replacement: `bolt.New(bolt.NewConsoleHandler(os.Stdout)).SetLevel(bolt.DEBUG)`,
			Description: "Transform zap.NewExample() to bolt equivalent",
		},
		{
			Pattern:     regexp.MustCompile(`zap\.L\(\)`),
			Replacement: `bolt.New(bolt.NewJSONHandler(os.Stdout))`,
			Description: "Transform global logger access",
		},
		{
			Pattern:     regexp.MustCompile(`zap\.S\(\)`),
			Replacement: `bolt.New(bolt.NewJSONHandler(os.Stdout))`,
			Description: "Transform sugar logger access",
			Warning:     "Sugar API methods need manual conversion",
		},
		{
			Pattern:     regexp.MustCompile(`\.Sugar\(\)`),
			Replacement: ``,
			Description: "Remove Sugar() calls",
			Warning:     "Sugar API methods need manual conversion to structured logging",
		},
		{
			Pattern:     regexp.MustCompile(`\.Sync\(\)`),
			Replacement: ``,
			Description: "Remove Sync() calls (not needed in Bolt)",
		},
	}
}

// TransformFileText performs regex-based transformations on a file.
func (ztt *ZapTextTransformer) TransformFileText(filePath string) (*ZapTransformResult, error) {
	result := &ZapTransformResult{
		FilePath: filePath,
		Success:  true,
		Changes:  0,
		Errors:   []string{},
		Warnings: []string{},
	}

	// Read file
	content, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	result.OriginalCode = string(content)
	transformedContent := string(content)

	rules := ztt.GetZapTransformationRules()
	for _, rule := range rules {
		if rule.Pattern.Match([]byte(transformedContent)) {
			newContent := rule.Pattern.ReplaceAllString(transformedContent, rule.Replacement)
			if newContent != transformedContent {
				result.Changes++
				transformedContent = newContent
				if rule.Warning != "" {
					result.Warnings = append(result.Warnings, rule.Warning)
				}
			}
		}
	}

	result.TransformedCode = transformedContent

	// Write file if changes were made and not in dry-run mode
	if result.Changes > 0 && !ztt.dryRun {
		if err := os.WriteFile(filePath, []byte(transformedContent), 0644); err != nil {
			result.Success = false
			result.Errors = append(result.Errors, fmt.Sprintf("write error: %v", err))
		}
	}

	return result, nil
}

// TransformDirectoryText performs regex-based transformations on all Go files in a directory.
func (ztt *ZapTextTransformer) TransformDirectoryText(dirPath string) ([]*ZapTransformResult, error) {
	var results []*ZapTransformResult

	err := filepath.Walk(dirPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if !strings.HasSuffix(path, ".go") || strings.Contains(path, "vendor/") {
			return nil
		}

		result, err := ztt.TransformFileText(path)
		if err != nil {
			return fmt.Errorf("failed to transform %s: %w", path, err)
		}

		results = append(results, result)
		return nil
	})

	return results, err
}

// ZapMigrationGuide generates migration guides for Zap to Bolt.
type ZapMigrationGuide struct {
	detectedPatterns []ZapDetectedPattern
}

// ZapDetectedPattern represents a Zap code pattern that needs migration.
type ZapDetectedPattern struct {
	Pattern     string `json:"pattern"`
	Suggestion  string `json:"suggestion"`
	Automatic   bool   `json:"automatic"`
	FilePath    string `json:"file_path"`
	LineNumber  int    `json:"line_number"`
	Category    string `json:"category"`
}

// NewZapMigrationGuide creates a new Zap migration guide generator.
func NewZapMigrationGuide() *ZapMigrationGuide {
	return &ZapMigrationGuide{
		detectedPatterns: []ZapDetectedPattern{},
	}
}

// AnalyzeFile analyzes a file and detects Zap patterns that need migration.
func (zmg *ZapMigrationGuide) AnalyzeFile(filePath string) error {
	content, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("failed to read file: %w", err)
	}

	lines := strings.Split(string(content), "\n")
	for lineNum, line := range lines {
		zmg.analyzeLine(line, filePath, lineNum+1)
	}

	return nil
}

// analyzeLine analyzes a single line for Zap migration patterns.
func (zmg *ZapMigrationGuide) analyzeLine(line, filePath string, lineNum int) {
	line = strings.TrimSpace(line)

	patterns := []struct {
		regex      *regexp.Regexp
		suggestion string
		automatic  bool
		category   string
	}{
		{
			regex:      regexp.MustCompile(`zap\.NewProduction\(`),
			suggestion: "Replace with bolt.New(bolt.NewJSONHandler(os.Stdout)).SetLevel(bolt.INFO)",
			automatic:  true,
			category:   "Logger Creation",
		},
		{
			regex:      regexp.MustCompile(`zap\.NewDevelopment\(`),
			suggestion: "Replace with bolt.New(bolt.NewConsoleHandler(os.Stdout)).SetLevel(bolt.DEBUG)",
			automatic:  true,
			category:   "Logger Creation",
		},
		{
			regex:      regexp.MustCompile(`\.Sugar\(\)`),
			suggestion: "Remove .Sugar() calls and convert to structured logging with Bolt's API",
			automatic:  false,
			category:   "Sugar API",
		},
		{
			regex:      regexp.MustCompile(`\.Infof\(|\.Debugf\(|\.Errorf\(|\.Warnf\(`),
			suggestion: "Replace formatted logging with bolt's .Printf() method or structured fields",
			automatic:  false,
			category:   "Sugar API",
		},
		{
			regex:      regexp.MustCompile(`\.Infow\(|\.Debugw\(|\.Errorw\(|\.Warnw\(`),
			suggestion: "Convert keyed logging to structured fields with bolt",
			automatic:  false,
			category:   "Sugar API",
		},
		{
			regex:      regexp.MustCompile(`zap\.String\(|zap\.Int\(|zap\.Bool\(`),
			suggestion: "Replace zap field constructors with bolt method calls (.Str(), .Int(), .Bool())",
			automatic:  false,
			category:   "Field Constructors",
		},
		{
			regex:      regexp.MustCompile(`\.With\(`),
			suggestion: "Zap's With() returns Logger, bolt's With() returns Event - use .Logger() to get logger",
			automatic:  false,
			category:   "Context",
		},
		{
			regex:      regexp.MustCompile(`\.Sync\(\)`),
			suggestion: "Remove .Sync() calls - not needed in Bolt",
			automatic:  true,
			category:   "Synchronization",
		},
		{
			regex:      regexp.MustCompile(`zapcore\.`),
			suggestion: "zapcore usage requires manual migration - consider Bolt's handler system",
			automatic:  false,
			category:   "Core Configuration",
		}
	}

	for _, pattern := range patterns {
		if pattern.regex.MatchString(line) {
			zmg.detectedPatterns = append(zmg.detectedPatterns, ZapDetectedPattern{
				Pattern:    pattern.regex.String(),
				Suggestion: pattern.suggestion,
				Automatic:  pattern.automatic,
				FilePath:   filePath,
				LineNumber: lineNum,
				Category:   pattern.category,
			})
		}
	}
}

// GenerateGuide generates a comprehensive Zap to Bolt migration guide.
func (zmg *ZapMigrationGuide) GenerateGuide() string {
	var guide strings.Builder

	guide.WriteString("# Zap to Bolt Migration Guide\n\n")
	guide.WriteString("This guide was automatically generated based on your codebase analysis.\n\n")

	if len(zmg.detectedPatterns) == 0 {
		guide.WriteString("No Zap patterns detected in the analyzed files.\n")
		return guide.String()
	}

	// Group patterns by category
	categoryMap := make(map[string][]ZapDetectedPattern)
	automaticPatterns := []ZapDetectedPattern{}
	manualPatterns := []ZapDetectedPattern{}

	for _, pattern := range zmg.detectedPatterns {
		categoryMap[pattern.Category] = append(categoryMap[pattern.Category], pattern)
		if pattern.Automatic {
			automaticPatterns = append(automaticPatterns, pattern)
		} else {
			manualPatterns = append(manualPatterns, pattern)
		}
	}

	// Summary
	guide.WriteString("## Migration Summary\n\n")
	guide.WriteString(fmt.Sprintf("- **Total patterns detected**: %d\n", len(zmg.detectedPatterns)))
	guide.WriteString(fmt.Sprintf("- **Automatic transformations**: %d\n", len(automaticPatterns)))
	guide.WriteString(fmt.Sprintf("- **Manual transformations**: %d\n\n", len(manualPatterns)))

	// Automatic transformations
	if len(automaticPatterns) > 0 {
		guide.WriteString("## Automatic Transformations\n\n")
		guide.WriteString("These patterns can be automatically transformed using the migration tool:\n\n")

		for _, pattern := range automaticPatterns {
			guide.WriteString(fmt.Sprintf("- **File**: %s:%d\n", pattern.FilePath, pattern.LineNumber))
			guide.WriteString(fmt.Sprintf("  **Category**: %s\n", pattern.Category))
			guide.WriteString(fmt.Sprintf("  **Suggestion**: %s\n\n", pattern.Suggestion))
		}
	}

	// Manual transformations by category
	if len(manualPatterns) > 0 {
		guide.WriteString("## Manual Transformations Required\n\n")
		
		for category, patterns := range categoryMap {
			manualInCategory := []ZapDetectedPattern{}
			for _, p := range patterns {
				if !p.Automatic {
					manualInCategory = append(manualInCategory, p)
				}
			}
			
			if len(manualInCategory) > 0 {
				guide.WriteString(fmt.Sprintf("### %s\n\n", category))
				for _, pattern := range manualInCategory {
					guide.WriteString(fmt.Sprintf("- **File**: %s:%d\n", pattern.FilePath, pattern.LineNumber))
					guide.WriteString(fmt.Sprintf("  **Suggestion**: %s\n\n", pattern.Suggestion))
				}
			}
		}
	}

	// Migration mapping reference
	guide.WriteString("## Zap to Bolt API Mapping\n\n")
	guide.WriteString("### Logger Creation\n")
	guide.WriteString("```go\n")
	guide.WriteString("// Zap\n")
	guide.WriteString("logger, _ := zap.NewProduction()\n")
	guide.WriteString("logger, _ := zap.NewDevelopment()\n\n")
	guide.WriteString("// Bolt\n")
	guide.WriteString("logger := bolt.New(bolt.NewJSONHandler(os.Stdout)).SetLevel(bolt.INFO)\n")
	guide.WriteString("logger := bolt.New(bolt.NewConsoleHandler(os.Stdout)).SetLevel(bolt.DEBUG)\n")
	guide.WriteString("```\n\n")

	guide.WriteString("### Structured Logging\n")
	guide.WriteString("```go\n")
	guide.WriteString("// Zap\n")
	guide.WriteString("logger.Info(\"message\", zap.String(\"key\", \"value\"), zap.Int(\"count\", 42))\n\n")
	guide.WriteString("// Bolt\n")
	guide.WriteString("logger.Info().Str(\"key\", \"value\").Int(\"count\", 42).Msg(\"message\")\n")
	guide.WriteString("```\n\n")

	guide.WriteString("### Sugar API Migration\n")
	guide.WriteString("```go\n")
	guide.WriteString("// Zap Sugar\n")
	guide.WriteString("sugar := logger.Sugar()\n")
	guide.WriteString("sugar.Infof(\"User %s logged in\", username)\n")
	guide.WriteString("sugar.Infow(\"User logged in\", \"username\", username, \"ip\", clientIP)\n\n")
	guide.WriteString("// Bolt (Structured approach)\n")
	guide.WriteString("logger.Info().Str(\"username\", username).Msg(\"User logged in\")\n")
	guide.WriteString("logger.Info().Str(\"username\", username).Str(\"ip\", clientIP).Msg(\"User logged in\")\n")
	guide.WriteString("```\n\n")

	guide.WriteString("### Context and Fields\n")
	guide.WriteString("```go\n")
	guide.WriteString("// Zap\n")
	guide.WriteString("contextLogger := logger.With(zap.String(\"service\", \"auth\"))\n")
	guide.WriteString("contextLogger.Info(\"Operation completed\")\n\n")
	guide.WriteString("// Bolt\n")
	guide.WriteString("contextLogger := logger.With().Str(\"service\", \"auth\").Logger()\n")
	guide.WriteString("contextLogger.Info().Msg(\"Operation completed\")\n")
	guide.WriteString("```\n\n")

	guide.WriteString("## Performance Benefits After Migration\n\n")
	guide.WriteString("- **80% faster** logging operations compared to Zap\n")
	guide.WriteString("- **Zero memory allocations** in hot paths\n")
	guide.WriteString("- **Sub-100ns latency** for logging operations\n")
	guide.WriteString("- **Better concurrent performance** under high load\n")
	guide.WriteString("- **Automatic OpenTelemetry integration**\n\n")

	guide.WriteString("## Migration Checklist\n\n")
	guide.WriteString("- [ ] Replace import statements\n")
	guide.WriteString("- [ ] Update logger creation calls\n")
	guide.WriteString("- [ ] Convert field constructors to method calls\n")
	guide.WriteString("- [ ] Migrate Sugar API usage to structured logging\n")
	guide.WriteString("- [ ] Update With() usage pattern\n")
	guide.WriteString("- [ ] Remove Sync() calls\n")
	guide.WriteString("- [ ] Test logging output format\n")
	guide.WriteString("- [ ] Verify performance improvements\n")
	guide.WriteString("- [ ] Update configuration if using zapcore\n\n")

	return guide.String()
}

// SaveGuide saves the migration guide to a file.
func (zmg *ZapMigrationGuide) SaveGuide(filePath string) error {
	guide := zmg.GenerateGuide()
	return os.WriteFile(filePath, []byte(guide), 0644)
}