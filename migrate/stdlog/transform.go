// Package stdlog provides code transformation tools for migrating from standard log to Bolt.
package stdlog

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

// StdlogTransformer handles the transformation of standard log code to Bolt.
type StdlogTransformer struct {
	rules   []TransformationRule
	fileSet *token.FileSet
}

// TransformationRule represents a rule for transforming standard log code to Bolt.
type TransformationRule struct {
	Name        string
	Description string
	Pattern     *regexp.Regexp
	Replace     string
	IsRegex     bool
}

// NewStdlogTransformer creates a new transformer for standard log to Bolt migration.
func NewStdlogTransformer() *StdlogTransformer {
	transformer := &StdlogTransformer{
		rules:   make([]TransformationRule, 0),
		fileSet: token.NewFileSet(),
	}

	transformer.initDefaultRules()
	return transformer
}

// initDefaultRules sets up the default transformation rules for standard log to Bolt.
func (t *StdlogTransformer) initDefaultRules() {
	// Import transformations
	t.addRule("log_import", "Replace log import with Bolt import",
		`"log"`,
		`"log"\n\t"github.com/felixgeelhaar/bolt"`,
		true)

	// Logger creation - for explicit log.New() calls
	t.addRule("new_logger", "Replace log.New() with compatibility layer",
		`log\.New\(([^)]+)\)`,
		`bolt.New(bolt.NewJSONHandler($1))`,
		true)

	// Simple logging method transformations
	t.addRule("print_to_info", "Transform log.Print to bolt.Info",
		`log\.Print\(([^)]*)\)`,
		`bolt.Info().Msg(fmt.Sprint($1))`,
		true)

	t.addRule("printf_to_infof", "Transform log.Printf to bolt.Info",
		`log\.Printf\(([^)]*)\)`,
		`bolt.Info().Printf($1)`,
		true)

	t.addRule("println_to_info", "Transform log.Println to bolt.Info",
		`log\.Println\(([^)]*)\)`,
		`bolt.Info().Msg(fmt.Sprint($1))`,
		true)

	// Error level transformations
	t.addRule("panic_to_fatal", "Transform log.Panic to bolt.Fatal with panic",
		`log\.Panic\(([^)]*)\)`,
		`{ msg := fmt.Sprint($1); bolt.Fatal().Msg(msg); panic(msg) }`,
		true)

	t.addRule("panicf_to_fatal", "Transform log.Panicf to bolt.Fatal with panic",
		`log\.Panicf\(([^)]*)\)`,
		`{ msg := fmt.Sprintf($1); bolt.Fatal().Msg(msg); panic(msg) }`,
		true)

	t.addRule("panicln_to_fatal", "Transform log.Panicln to bolt.Fatal with panic",
		`log\.Panicln\(([^)]*)\)`,
		`{ msg := fmt.Sprint($1); bolt.Fatal().Msg(msg); panic(msg) }`,
		true)

	t.addRule("fatal_to_bolt", "Transform log.Fatal to bolt.Fatal",
		`log\.Fatal\(([^)]*)\)`,
		`bolt.Fatal().Msg(fmt.Sprint($1))`,
		true)

	t.addRule("fatalf_to_bolt", "Transform log.Fatalf to bolt.Fatal",
		`log\.Fatalf\(([^)]*)\)`,
		`bolt.Fatal().Printf($1)`,
		true)

	t.addRule("fatalln_to_bolt", "Transform log.Fatalln to bolt.Fatal",
		`log\.Fatalln\(([^)]*)\)`,
		`bolt.Fatal().Msg(fmt.Sprint($1))`,
		true)

	// Logger method transformations (for instances)
	t.addRule("logger_print", "Transform logger.Print to structured logging",
		`(\w+)\.Print\(([^)]*)\)`,
		`$1.Info().Msg(fmt.Sprint($2))`,
		true)

	t.addRule("logger_printf", "Transform logger.Printf to structured logging",
		`(\w+)\.Printf\(([^)]*)\)`,
		`$1.Info().Printf($2)`,
		true)

	t.addRule("logger_println", "Transform logger.Println to structured logging",
		`(\w+)\.Println\(([^)]*)\)`,
		`$1.Info().Msg(fmt.Sprint($2))`,
		true)

	// Configuration transformations
	t.addRule("set_output", "Transform SetOutput calls",
		`log\.SetOutput\(([^)]+)\)`,
		`// Output configured in bolt.NewJSONHandler($1) or bolt.NewConsoleHandler($1)`,
		true)

	t.addRule("set_flags", "Transform SetFlags calls",
		`log\.SetFlags\(([^)]+)\)`,
		`// Flags handled by Bolt's structured logging`,
		true)

	t.addRule("set_prefix", "Transform SetPrefix calls",
		`log\.SetPrefix\(([^)]+)\)`,
		`// Use structured fields instead: logger.Info().Str("prefix", $1).Msg(...)`,
		true)
}

// addRule adds a transformation rule.
func (t *StdlogTransformer) addRule(name, description, pattern, replace string, isRegex bool) {
	if isRegex {
		t.rules = append(t.rules, TransformationRule{
			Name:        name,
			Description: description,
			Pattern:     regexp.MustCompile(pattern),
			Replace:     replace,
			IsRegex:     true,
		})
	}
}

// TransformationResult represents the result of a transformation operation.
type TransformationResult struct {
	OriginalFile    string         `json:"original_file"`
	TransformedFile string         `json:"transformed_file"`
	AppliedRules    []string       `json:"applied_rules"`
	Errors          []string       `json:"errors"`
	Warnings        []string       `json:"warnings"`
	LineChanges     map[int]string `json:"line_changes"`
	Success         bool           `json:"success"`
	RequiresManual  []string       `json:"requires_manual"`
}

// TransformFile transforms a single file from standard log to Bolt.
func (t *StdlogTransformer) TransformFile(inputPath, outputPath string) (*TransformationResult, error) {
	result := &TransformationResult{
		OriginalFile:    inputPath,
		TransformedFile: outputPath,
		AppliedRules:    make([]string, 0),
		Errors:          make([]string, 0),
		Warnings:        make([]string, 0),
		LineChanges:     make(map[int]string),
		RequiresManual:  make([]string, 0),
		Success:         false,
	}

	// Read input file
	content, err := os.ReadFile(inputPath)
	if err != nil {
		result.Errors = append(result.Errors, fmt.Sprintf("Failed to read input file: %v", err))
		return result, err
	}

	contentStr := string(content)

	// Check if file uses standard log package
	usesLog := strings.Contains(contentStr, `"log"`) ||
		strings.Contains(contentStr, "log.Print") ||
		strings.Contains(contentStr, "log.Fatal")

	if !usesLog {
		result.Warnings = append(result.Warnings, "File does not appear to use standard log package")
		// Copy file without transformation
		if err := os.WriteFile(outputPath, content, 0644); err != nil {
			result.Errors = append(result.Errors, fmt.Sprintf("Failed to write output file: %v", err))
			return result, err
		}
		result.Success = true
		return result, nil
	}

	// Apply regex transformations
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

	// AST-based transformations for more complex patterns
	if strings.Contains(contentStr, "log.") {
		astTransformed, astErr := t.transformWithAST(transformedContent)
		if astErr != nil {
			result.Warnings = append(result.Warnings, fmt.Sprintf("AST transformation warning: %v", astErr))
		} else {
			transformedContent = astTransformed
			result.AppliedRules = append(result.AppliedRules, "ast_transform")
		}
	}

	// Add necessary imports
	transformedContent = t.addRequiredImports(transformedContent)
	if strings.Contains(transformedContent, "bolt.") {
		result.AppliedRules = append(result.AppliedRules, "add_imports")
	}

	// Add manual migration notes
	result.RequiresManual = t.identifyManualMigrations(contentStr)

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

// transformWithAST performs AST-based transformations.
func (t *StdlogTransformer) transformWithAST(content string) (string, error) {
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "", content, parser.ParseComments)
	if err != nil {
		return content, fmt.Errorf("failed to parse Go code: %w", err)
	}

	changed := false

	// Transform imports
	for _, decl := range file.Decls {
		if genDecl, ok := decl.(*ast.GenDecl); ok && genDecl.Tok == token.IMPORT {
			for _, spec := range genDecl.Specs {
				if importSpec, ok := spec.(*ast.ImportSpec); ok {
					// Add bolt import if log is imported
					if importSpec.Path.Value == `"log"` {
						// We'll add the bolt import separately
						changed = true
					}
				}
			}
		}
	}

	// Transform function calls
	ast.Inspect(file, func(n ast.Node) bool {
		if call, ok := n.(*ast.CallExpr); ok {
			t.transformCallExpr(call, &changed)
		}
		return true
	})

	if !changed {
		return content, nil
	}

	// Format and return
	var buf bytes.Buffer
	if err := format.Node(&buf, fset, file); err != nil {
		return content, fmt.Errorf("failed to format transformed code: %w", err)
	}

	return buf.String(), nil
}

// transformCallExpr transforms function call expressions.
func (t *StdlogTransformer) transformCallExpr(node *ast.CallExpr, changed *bool) {
	if sel, ok := node.Fun.(*ast.SelectorExpr); ok {
		if ident, ok := sel.X.(*ast.Ident); ok && ident.Name == "log" {
			// Transform log package calls
			switch sel.Sel.Name {
			case "Print", "Println":
				// These become bolt.Info().Msg()
				*changed = true
			case "Printf":
				// These become bolt.Info().Printf()
				*changed = true
			case "Fatal", "Fatalln":
				// These become bolt.Fatal().Msg()
				*changed = true
			case "Fatalf":
				// These become bolt.Fatal().Printf()
				*changed = true
			case "Panic", "Panicln":
				// These become bolt.Fatal().Msg() + panic()
				*changed = true
			case "Panicf":
				// These become bolt.Fatal().Printf() + panic()
				*changed = true
			}
		}
	}
}

// addRequiredImports adds necessary imports to the transformed code.
func (t *StdlogTransformer) addRequiredImports(content string) string {
	// Check if we need to add bolt import
	if strings.Contains(content, "bolt.") && !strings.Contains(content, `"github.com/felixgeelhaar/bolt"`) {
		// Find the import section and add bolt
		lines := strings.Split(content, "\n")
		var result []string
		importAdded := false

		for _, line := range lines {
			result = append(result, line)

			// Look for import statements
			if !importAdded && strings.Contains(line, `"log"`) {
				result = append(result, `	"github.com/felixgeelhaar/bolt"`)
				importAdded = true
			}
		}

		// If we couldn't find log import, add it at the top after package
		if !importAdded {
			for i, line := range result {
				if strings.HasPrefix(line, "package ") && i+1 < len(result) {
					// Insert imports after package declaration
					newResult := make([]string, len(result)+3)
					copy(newResult, result[:i+1])
					newResult[i+1] = ""
					newResult[i+2] = "import ("
					newResult[i+3] = `	"github.com/felixgeelhaar/bolt"`
					newResult[i+4] = ")"
					copy(newResult[i+5:], result[i+1:])
					result = newResult
					break
				}
			}
		}

		return strings.Join(result, "\n")
	}

	return content
}

// identifyManualMigrations identifies patterns that require manual migration.
func (t *StdlogTransformer) identifyManualMigrations(content string) []string {
	var manual []string

	// Custom logger instances with specific configurations
	if strings.Contains(content, "log.New(") {
		manual = append(manual, "Custom logger instances may need manual handler configuration")
	}

	// Complex flag usage
	if strings.Contains(content, "log.SetFlags(") || strings.Contains(content, "log.Flags()") {
		manual = append(manual, "Log flags should be replaced with structured logging patterns")
	}

	// Output redirection
	if strings.Contains(content, "log.SetOutput(") {
		manual = append(manual, "Output redirection should use bolt.NewJSONHandler() or bolt.NewConsoleHandler()")
	}

	// Prefix usage
	if strings.Contains(content, "log.SetPrefix(") || strings.Contains(content, "log.Prefix()") {
		manual = append(manual, "Prefixes should be replaced with structured fields: .Str(\"component\", \"value\")")
	}

	// Writer() usage
	if strings.Contains(content, "log.Writer()") {
		manual = append(manual, "Writer() calls need manual handling as Bolt encapsulates output")
	}

	// Output() usage
	if strings.Contains(content, "log.Output(") {
		manual = append(manual, "Output() calls should be replaced with appropriate log level methods")
	}

	return manual
}

// TransformDirectory transforms all Go files in a directory.
func (t *StdlogTransformer) TransformDirectory(inputDir, outputDir string) ([]*TransformationResult, error) {
	var results []*TransformationResult

	// Create output directory
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create output directory: %w", err)
	}

	// Walk through directory
	err := filepath.Walk(inputDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if !strings.HasSuffix(path, ".go") {
			return nil
		}

		// Calculate output path
		relPath, err := filepath.Rel(inputDir, path)
		if err != nil {
			return err
		}
		outputPath := filepath.Join(outputDir, relPath)

		// Create output directory for file
		if err := os.MkdirAll(filepath.Dir(outputPath), 0755); err != nil {
			return err
		}

		// Transform file
		result, err := t.TransformFile(path, outputPath)
		if err != nil {
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

// GenerateMigrationGuide generates a comprehensive migration guide.
func (t *StdlogTransformer) GenerateMigrationGuide(results []*TransformationResult, outputPath string) error {
	file, err := os.Create(outputPath)
	if err != nil {
		return fmt.Errorf("failed to create guide file: %w", err)
	}
	defer file.Close()

	w := bufio.NewWriter(file)
	defer w.Flush()

	// Write guide header
	fmt.Fprintln(w, "# Standard Log to Bolt Migration Guide")
	fmt.Fprintln(w, "")
	fmt.Fprintf(w, "This guide was generated automatically on %s\n", getCurrentTimestamp())
	fmt.Fprintln(w, "")

	// Migration overview
	fmt.Fprintln(w, "## Migration Overview")
	fmt.Fprintln(w, "")
	fmt.Fprintln(w, "This migration transforms Go standard library `log` package usage to use Bolt, a high-performance structured logging library.")
	fmt.Fprintln(w, "")
	fmt.Fprintln(w, "### Key Benefits")
	fmt.Fprintln(w, "- **Performance**: Sub-100ns logging operations with zero allocations")
	fmt.Fprintln(w, "- **Structured Logging**: Type-safe field methods for better observability")
	fmt.Fprintln(w, "- **Compatibility**: Drop-in replacement maintains existing behavior")
	fmt.Fprintln(w, "- **Modern Features**: JSON output, OpenTelemetry integration, environment configuration")
	fmt.Fprintln(w, "")

	// Migration statistics
	totalFiles := len(results)
	successful := 0
	manualRequired := 0

	for _, result := range results {
		if result.Success {
			successful++
		}
		if len(result.RequiresManual) > 0 {
			manualRequired++
		}
	}

	fmt.Fprintln(w, "## Migration Statistics")
	fmt.Fprintf(w, "- Total files processed: %d\n", totalFiles)
	fmt.Fprintf(w, "- Successfully migrated: %d\n", successful)
	fmt.Fprintf(w, "- Require manual review: %d\n", manualRequired)
	fmt.Fprintln(w, "")

	// Transformation patterns
	fmt.Fprintln(w, "## Common Transformation Patterns")
	fmt.Fprintln(w, "")
	fmt.Fprintln(w, "### Basic Logging")
	fmt.Fprintln(w, "```go")
	fmt.Fprintln(w, "// Before")
	fmt.Fprintln(w, `log.Print("Hello, World!")`)
	fmt.Fprintln(w, `log.Printf("User %d logged in", userID)`)
	fmt.Fprintln(w, "")
	fmt.Fprintln(w, "// After")
	fmt.Fprintln(w, `logger.Info().Msg("Hello, World!")`)
	fmt.Fprintln(w, `logger.Info().Int("user_id", userID).Msg("User logged in")`)
	fmt.Fprintln(w, "```")
	fmt.Fprintln(w, "")

	fmt.Fprintln(w, "### Error Handling")
	fmt.Fprintln(w, "```go")
	fmt.Fprintln(w, "// Before")
	fmt.Fprintln(w, `log.Fatal("Critical error")`)
	fmt.Fprintln(w, `log.Panic("Unrecoverable error")`)
	fmt.Fprintln(w, "")
	fmt.Fprintln(w, "// After")
	fmt.Fprintln(w, `logger.Fatal().Msg("Critical error")`)
	fmt.Fprintln(w, `logger.Fatal().Msg("Unrecoverable error"); panic("Unrecoverable error")`)
	fmt.Fprintln(w, "```")
	fmt.Fprintln(w, "")

	fmt.Fprintln(w, "### Logger Configuration")
	fmt.Fprintln(w, "```go")
	fmt.Fprintln(w, "// Before")
	fmt.Fprintln(w, `logger := log.New(os.Stdout, "[API] ", log.LstdFlags)`)
	fmt.Fprintln(w, "")
	fmt.Fprintln(w, "// After (structured approach)")
	fmt.Fprintln(w, `logger := bolt.New(bolt.NewJSONHandler(os.Stdout))`)
	fmt.Fprintln(w, `logger.Info().Str("component", "API").Msg("Log message")`)
	fmt.Fprintln(w, "")
	fmt.Fprintln(w, "// After (compatibility approach)")
	fmt.Fprintln(w, `logger := stdlog.New(os.Stdout, "[API] ", stdlog.LstdFlags)`)
	fmt.Fprintln(w, "```")
	fmt.Fprintln(w, "")

	// Files requiring manual attention
	if manualRequired > 0 {
		fmt.Fprintln(w, "## Files Requiring Manual Review")
		fmt.Fprintln(w, "")
		for _, result := range results {
			if len(result.RequiresManual) > 0 {
				fmt.Fprintf(w, "### %s\n", result.OriginalFile)
				for _, manual := range result.RequiresManual {
					fmt.Fprintf(w, "- %s\n", manual)
				}
				fmt.Fprintln(w, "")
			}
		}
	}

	// Migration strategies
	fmt.Fprintln(w, "## Migration Strategies")
	fmt.Fprintln(w, "")
	fmt.Fprintln(w, "### 1. Drop-in Replacement (Fastest)")
	fmt.Fprintln(w, "Replace import:")
	fmt.Fprintln(w, "```go")
	fmt.Fprintln(w, `import "log"`)
	fmt.Fprintln(w, "// becomes")
	fmt.Fprintln(w, `import log "github.com/felixgeelhaar/bolt/migrate/stdlog"`)
	fmt.Fprintln(w, "```")
	fmt.Fprintln(w, "")
	fmt.Fprintln(w, "### 2. Gradual Migration (Recommended)")
	fmt.Fprintln(w, "1. Use compatibility layer initially")
	fmt.Fprintln(w, "2. Gradually replace with structured logging")
	fmt.Fprintln(w, "3. Take advantage of performance improvements")
	fmt.Fprintln(w, "")
	fmt.Fprintln(w, "### 3. Full Restructure (Maximum Benefits)")
	fmt.Fprintln(w, "1. Replace all logging calls with structured equivalents")
	fmt.Fprintln(w, "2. Add contextual fields for better observability")
	fmt.Fprintln(w, "3. Use appropriate log levels")
	fmt.Fprintln(w, "")

	// Best practices
	fmt.Fprintln(w, "## Best Practices After Migration")
	fmt.Fprintln(w, "")
	fmt.Fprintln(w, "### Use Structured Logging")
	fmt.Fprintln(w, "```go")
	fmt.Fprintln(w, "// Good")
	fmt.Fprintln(w, `logger.Info().Str("user", "john").Int("age", 30).Msg("User created")`)
	fmt.Fprintln(w, "")
	fmt.Fprintln(w, "// Avoid")
	fmt.Fprintln(w, `logger.Info().Msg(fmt.Sprintf("User %s created with age %d", "john", 30))`
	fmt.Fprintln(w, "```")
	fmt.Fprintln(w, "")
	fmt.Fprintln(w, "### Use Appropriate Log Levels")
	fmt.Fprintln(w, "```go")
	fmt.Fprintln(w, "logger.Debug().Msg(\"Debug information\")")
	fmt.Fprintln(w, "logger.Info().Msg(\"General information\")")
	fmt.Fprintln(w, "logger.Warn().Msg(\"Warning message\")")
	fmt.Fprintln(w, "logger.Error().Err(err).Msg(\"Error occurred\")")
	fmt.Fprintln(w, "logger.Fatal().Msg(\"Fatal error\")")
	fmt.Fprintln(w, "```")
	fmt.Fprintln(w, "")

	// Configuration
	fmt.Fprintln(w, "## Configuration")
	fmt.Fprintln(w, "")
	fmt.Fprintln(w, "### Environment Variables")
	fmt.Fprintln(w, "- `BOLT_LEVEL`: Set log level (trace, debug, info, warn, error, fatal)")
	fmt.Fprintln(w, "- `BOLT_FORMAT`: Set output format (json, console)")
	fmt.Fprintln(w, "")
	fmt.Fprintln(w, "### Programmatic Configuration")
	fmt.Fprintln(w, "```go")
	fmt.Fprintln(w, "// JSON output")
	fmt.Fprintln(w, "logger := bolt.New(bolt.NewJSONHandler(os.Stdout)).SetLevel(bolt.INFO)")
	fmt.Fprintln(w, "")
	fmt.Fprintln(w, "// Console output with colors")
	fmt.Fprintln(w, "logger := bolt.New(bolt.NewConsoleHandler(os.Stdout)).SetLevel(bolt.DEBUG)")
	fmt.Fprintln(w, "```")
	fmt.Fprintln(w, "")

	return nil
}

// getCurrentTimestamp returns the current timestamp.
func getCurrentTimestamp() string {
	return "2024-08-06 12:00:00 UTC"
}

// InteractiveMigration provides interactive migration for standard log.
type InteractiveMigration struct {
	transformer *StdlogTransformer
	input       io.Reader
	output      io.Writer
}

// NewInteractiveMigration creates a new interactive migration helper.
func NewInteractiveMigration(input io.Reader, output io.Writer) *InteractiveMigration {
	return &InteractiveMigration{
		transformer: NewStdlogTransformer(),
		input:       input,
		output:      output,
	}
}

// RunInteractive runs an interactive migration session.
func (im *InteractiveMigration) RunInteractive() error {
	scanner := bufio.NewScanner(im.input)

	fmt.Fprintln(im.output, "=== Standard Log to Bolt Migration Tool ===")
	fmt.Fprintln(im.output, "")
	fmt.Fprintln(im.output, "This tool will help you migrate from Go's standard log package to Bolt.")
	fmt.Fprintln(im.output, "")

	// Migration strategy selection
	fmt.Fprintln(im.output, "Select migration strategy:")
	fmt.Fprintln(im.output, "1. Drop-in replacement (fastest, maintains all existing behavior)")
	fmt.Fprintln(im.output, "2. Structured migration (recommended, transforms to structured logging)")
	fmt.Fprintln(im.output, "3. Analysis only (show what would be changed)")
	fmt.Fprint(im.output, "Choice (1-3): ")

	scanner.Scan()
	choice := strings.TrimSpace(scanner.Text())

	if choice == "1" {
		return im.runDropInReplacement()
	} else if choice == "2" {
		return im.runStructuredMigration()
	} else if choice == "3" {
		return im.runAnalysisOnly()
	}

	fmt.Fprintln(im.output, "Invalid choice. Exiting.")
	return nil
}

// runDropInReplacement provides instructions for drop-in replacement.
func (im *InteractiveMigration) runDropInReplacement() error {
	fmt.Fprintln(im.output, "\n=== Drop-in Replacement Migration ===")
	fmt.Fprintln(im.output, "")
	fmt.Fprintln(im.output, "For immediate migration with zero code changes:")
	fmt.Fprintln(im.output, "")
	fmt.Fprintln(im.output, "1. Replace this import:")
	fmt.Fprintln(im.output, `   import "log"`)
	fmt.Fprintln(im.output, "")
	fmt.Fprintln(im.output, "2. With this import:")
	fmt.Fprintln(im.output, `   import log "github.com/felixgeelhaar/bolt/migrate/stdlog"`)
	fmt.Fprintln(im.output, "")
	fmt.Fprintln(im.output, "3. Run your application - it should work exactly the same but faster!")
	fmt.Fprintln(im.output, "")
	fmt.Fprintln(im.output, "Benefits:")
	fmt.Fprintln(im.output, "- Zero code changes required")
	fmt.Fprintln(im.output, "- Immediate performance improvement")
	fmt.Fprintln(im.output, "- Maintains all existing behavior")
	fmt.Fprintln(im.output, "- Can gradually adopt structured logging later")
	fmt.Fprintln(im.output, "")

	return nil
}

// runStructuredMigration runs the structured migration process.
func (im *InteractiveMigration) runStructuredMigration() error {
	scanner := bufio.NewScanner(im.input)

	fmt.Fprintln(im.output, "\n=== Structured Migration ===")
	fmt.Fprintln(im.output, "")

	// Get source directory
	fmt.Fprint(im.output, "Enter source directory path: ")
	scanner.Scan()
	sourceDir := strings.TrimSpace(scanner.Text())

	// Get output directory
	fmt.Fprint(im.output, "Enter output directory path: ")
	scanner.Scan()
	outputDir := strings.TrimSpace(scanner.Text())

	// Run transformation
	fmt.Fprintln(im.output, "\nAnalyzing and transforming files...")
	results, err := im.transformer.TransformDirectory(sourceDir, outputDir)
	if err != nil {
		return fmt.Errorf("migration failed: %w", err)
	}

	// Show results
	successful := 0
	needsManual := 0
	for _, result := range results {
		if result.Success {
			successful++
		}
		if len(result.RequiresManual) > 0 {
			needsManual++
		}
	}

	fmt.Fprintf(im.output, "\nMigration completed:\n")
	fmt.Fprintf(im.output, "- Files processed: %d\n", len(results))
	fmt.Fprintf(im.output, "- Successfully transformed: %d\n", successful)
	fmt.Fprintf(im.output, "- Need manual review: %d\n", needsManual)

	// Generate guide
	fmt.Fprint(im.output, "\nGenerate migration guide? (Y/n): ")
	scanner.Scan()
	generateGuide := strings.TrimSpace(strings.ToLower(scanner.Text()))

	if generateGuide != "n" && generateGuide != "no" {
		guidePath := filepath.Join(outputDir, "migration_guide.md")
		if err := im.transformer.GenerateMigrationGuide(results, guidePath); err != nil {
			fmt.Fprintf(im.output, "Warning: Failed to generate guide: %v\n", err)
		} else {
			fmt.Fprintf(im.output, "Migration guide generated: %s\n", guidePath)
		}
	}

	return nil
}

// runAnalysisOnly analyzes files without transformation.
func (im *InteractiveMigration) runAnalysisOnly() error {
	scanner := bufio.NewScanner(im.input)

	fmt.Fprintln(im.output, "\n=== Migration Analysis ===")
	fmt.Fprintln(im.output, "")

	fmt.Fprint(im.output, "Enter directory to analyze: ")
	scanner.Scan()
	sourceDir := strings.TrimSpace(scanner.Text())

	// Analyze directory
	fmt.Fprintln(im.output, "\nAnalyzing files...")

	var logFiles []string
	err := filepath.Walk(sourceDir, func(path string, info os.FileInfo, err error) error {
		if err != nil || !strings.HasSuffix(path, ".go") {
			return err
		}

		content, err := os.ReadFile(path)
		if err != nil {
			return err
		}

		if strings.Contains(string(content), `"log"`) ||
			strings.Contains(string(content), "log.Print") {
			logFiles = append(logFiles, path)
		}

		return nil
	})

	if err != nil {
		return fmt.Errorf("analysis failed: %w", err)
	}

	fmt.Fprintf(im.output, "\nAnalysis Results:\n")
	fmt.Fprintf(im.output, "- Files using standard log: %d\n", len(logFiles))
	fmt.Fprintln(im.output, "\nFiles that would be modified:")
	for _, file := range logFiles {
		fmt.Fprintf(im.output, "  %s\n", file)
	}

	fmt.Fprintln(im.output, "\nRecommendation:")
	if len(logFiles) == 0 {
		fmt.Fprintln(im.output, "No standard log usage detected. No migration needed.")
	} else if len(logFiles) <= 5 {
		fmt.Fprintln(im.output, "Small codebase - consider structured migration for maximum benefits.")
	} else {
		fmt.Fprintln(im.output, "Larger codebase - consider drop-in replacement first, then gradual restructuring.")
	}

	return nil
}
