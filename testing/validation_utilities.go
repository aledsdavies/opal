package testing

import (
	"context"
	"fmt"
	"go/parser"
	"go/token"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"time"

	coreast "github.com/aledsdavies/devcmd/core/ast"
	"github.com/aledsdavies/devcmd/core/plan"
	"github.com/aledsdavies/devcmd/runtime/decorators"
)

// ValidationAssertions provides fluent validation utilities for decorator testing
type ValidationAssertions struct {
	result *ValidationResult
	errors []string
}

// Assert creates a new validation assertion helper
func Assert(result ValidationResult) *ValidationAssertions {
	return &ValidationAssertions{
		result: &result,
		errors: []string{},
	}
}

// === INTERPRETER MODE VALIDATIONS ===

// InterpreterSucceeds validates that interpreter mode executed successfully
func (v *ValidationAssertions) InterpreterSucceeds() *ValidationAssertions {
	if !v.result.InterpreterResult.Success {
		v.errors = append(v.errors, fmt.Sprintf("Interpreter mode failed: %v", v.result.InterpreterResult.Error))
	}
	return v
}

// InterpreterFails validates that interpreter mode failed with expected error
func (v *ValidationAssertions) InterpreterFails(expectedErrorContains string) *ValidationAssertions {
	if v.result.InterpreterResult.Success {
		v.errors = append(v.errors, "Expected interpreter mode to fail, but it succeeded")
	} else if expectedErrorContains != "" {
		if !strings.Contains(v.result.InterpreterResult.Error.Error(), expectedErrorContains) {
			v.errors = append(v.errors, fmt.Sprintf("Interpreter error should contain %q, got: %v",
				expectedErrorContains, v.result.InterpreterResult.Error))
		}
	}
	return v
}

// InterpreterReturns validates the interpreter mode return value
func (v *ValidationAssertions) InterpreterReturns(expected interface{}) *ValidationAssertions {
	if !reflect.DeepEqual(v.result.InterpreterResult.Data, expected) {
		v.errors = append(v.errors, fmt.Sprintf("Interpreter returned %v, expected %v",
			v.result.InterpreterResult.Data, expected))
	}
	return v
}

// === GENERATOR MODE VALIDATIONS ===

// GeneratorSucceeds validates that generator mode executed successfully
func (v *ValidationAssertions) GeneratorSucceeds() *ValidationAssertions {
	if !v.result.GeneratorResult.Success {
		v.errors = append(v.errors, fmt.Sprintf("Generator mode failed: %v", v.result.GeneratorResult.Error))
	}
	return v
}

// GeneratorFails validates that generator mode failed with expected error
func (v *ValidationAssertions) GeneratorFails(expectedErrorContains string) *ValidationAssertions {
	if v.result.GeneratorResult.Success {
		v.errors = append(v.errors, "Expected generator mode to fail, but it succeeded")
	} else if expectedErrorContains != "" {
		if !strings.Contains(v.result.GeneratorResult.Error.Error(), expectedErrorContains) {
			v.errors = append(v.errors, fmt.Sprintf("Generator error should contain %q, got: %v",
				expectedErrorContains, v.result.GeneratorResult.Error))
		}
	}
	return v
}

// GeneratorProducesValidGo validates that generator mode produces syntactically correct Go code
func (v *ValidationAssertions) GeneratorProducesValidGo() *ValidationAssertions {
	if !v.result.GeneratorResult.Success {
		return v // Skip if generation failed
	}

	code, ok := v.result.GeneratorResult.Data.(string)
	if !ok {
		v.errors = append(v.errors, fmt.Sprintf("Generator should return string, got %T", v.result.GeneratorResult.Data))
		return v
	}

	// Wrap code in a function to make it parseable
	wrappedCode := fmt.Sprintf("package main\nfunc testFunc() {\n%s\n}", code)

	fset := token.NewFileSet()
	_, err := parser.ParseFile(fset, "", wrappedCode, parser.ParseComments)
	if err != nil {
		v.errors = append(v.errors, fmt.Sprintf("Generated Go code is invalid: %v\nCode:\n%s", err, code))
	}

	return v
}

// GeneratorExecutesCorrectly validates that generated code compiles and executes without hanging
func (v *ValidationAssertions) GeneratorExecutesCorrectly() *ValidationAssertions {
	if !v.result.GeneratorResult.Success {
		return v // Skip if generation failed
	}

	code, ok := v.result.GeneratorResult.Data.(string)
	if !ok {
		v.errors = append(v.errors, "Generator should return string for execution validation")
		return v
	}

	// Create a complete CLI program with the generated code
	fullProgram := fmt.Sprintf(`package main

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"time"
)

type CommandResult struct {
	Stdout   string
	Stderr   string  
	ExitCode int
}

func (cr CommandResult) Success() bool {
	return cr.ExitCode == 0
}

func (cr CommandResult) Failed() bool {
	return cr.ExitCode != 0
}

type ExecutionContext struct {
	context.Context
	variables  map[string]string
	WorkingDir string
}

func NewExecutionContext(ctx context.Context) *ExecutionContext {
	workingDir, _ := os.Getwd()
	return &ExecutionContext{
		Context:    ctx,
		variables:  make(map[string]string),
		WorkingDir: workingDir,
	}
}

func (ctx *ExecutionContext) GetVariable(name string) (string, bool) {
	value, exists := ctx.variables[name]
	return value, exists
}

func (ctx *ExecutionContext) SetVariable(name, value string) {
	ctx.variables[name] = value
}

func executeShellCommand(ctx context.Context, command string) CommandResult {
	cmd := exec.CommandContext(ctx, "sh", "-c", command)
	stdout, err := cmd.Output()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			return CommandResult{
				Stdout:   string(stdout),
				Stderr:   string(exitErr.Stderr),
				ExitCode: exitErr.ExitCode(),
			}
		}
		return CommandResult{
			Stdout:   string(stdout),
			Stderr:   err.Error(),
			ExitCode: 1,
		}
	}
	return CommandResult{
		Stdout:   string(stdout),
		Stderr:   "",
		ExitCode: 0,
	}
}

func main() {
	baseCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	
	ctx := NewExecutionContext(baseCtx)
	
	// Execute the generated code wrapped in a function
	result := executeGeneratedCode(ctx)
	
	// Check result (optional - depends on what we want to validate)
	if result.Failed() {
		fmt.Printf("Generated code failed: %%s\n", result.Stderr)
		os.Exit(result.ExitCode)
	}
	
	fmt.Println("Execution completed successfully")
}

func executeGeneratedCode(ctx *ExecutionContext) CommandResult {
	// Execute the generated code
	%s
	
	// If we reach here without returning, return success
	return CommandResult{Stdout: "", Stderr: "", ExitCode: 0}
}`, code)

	// Try to compile and execute this program with a timeout
	if err := v.compileAndExecuteGenerated(fullProgram); err != nil {
		v.errors = append(v.errors, fmt.Sprintf("Generated code execution failed: %v\nGenerated code:\n%s", err, code))
	}

	return v
}

// GeneratorCodeContains validates that generated code contains expected strings
func (v *ValidationAssertions) GeneratorCodeContains(expectedStrings ...string) *ValidationAssertions {
	return v.GeneratorCodeContainsWithContext("", expectedStrings...)
}

// GeneratorCodeContainsWithContext validates that generated code contains expected strings with optional context message
func (v *ValidationAssertions) GeneratorCodeContainsWithContext(context string, expectedStrings ...string) *ValidationAssertions {
	if !v.result.GeneratorResult.Success {
		return v
	}

	code, ok := v.result.GeneratorResult.Data.(string)
	if !ok {
		v.errors = append(v.errors, "Generator should return string")
		return v
	}

	for _, expected := range expectedStrings {
		if !strings.Contains(code, expected) {
			errorMsg := fmt.Sprintf("Generated code should contain %q", expected)
			if context != "" {
				errorMsg = fmt.Sprintf("%s (%s)", errorMsg, context)
			}
			errorMsg += fmt.Sprintf("\nCode:\n%s", code)
			v.errors = append(v.errors, errorMsg)
		}
	}

	return v
}

// GeneratorCodeContainsf validates that generated code contains formatted strings
func (v *ValidationAssertions) GeneratorCodeContainsf(format string, args ...interface{}) *ValidationAssertions {
	expected := fmt.Sprintf(format, args...)
	return v.GeneratorCodeContains(expected)
}

// GeneratorCodeContainsfWithContext validates that generated code contains formatted strings with context
func (v *ValidationAssertions) GeneratorCodeContainsfWithContext(context string, format string, args ...interface{}) *ValidationAssertions {
	expected := fmt.Sprintf(format, args...)
	return v.GeneratorCodeContainsWithContext(context, expected)
}

// GeneratorImplementsPattern validates that generated code implements a semantic pattern
func (v *ValidationAssertions) GeneratorImplementsPattern(pattern CodePattern) *ValidationAssertions {
	if !v.result.GeneratorResult.Success {
		return v
	}

	code, ok := v.result.GeneratorResult.Data.(string)
	if !ok {
		v.errors = append(v.errors, "Generator should return string")
		return v
	}

	if !pattern.Matches(code) {
		v.errors = append(v.errors, fmt.Sprintf("Generated code should implement %s\nCode:\n%s", pattern.Description(), code))
	}

	return v
}

// GeneratorTracksEnvVars validates that generator mode properly tracks environment variables
func (v *ValidationAssertions) GeneratorTracksEnvVars(expectedVars ...string) *ValidationAssertions {
	if !v.result.GeneratorResult.Success {
		return v
	}

	// This would require integration with the context to check tracked env vars
	// For now, we validate that the generated code contains environment variable access
	code, ok := v.result.GeneratorResult.Data.(string)
	if !ok {
		return v
	}

	for _, envVar := range expectedVars {
		if !strings.Contains(code, envVar) {
			v.errors = append(v.errors, fmt.Sprintf("Generated code should reference environment variable %q", envVar))
		}
	}

	return v
}

// === PLAN MODE VALIDATIONS ===

// PlanSucceeds validates that plan mode executed successfully
func (v *ValidationAssertions) PlanSucceeds() *ValidationAssertions {
	if !v.result.PlanResult.Success {
		v.errors = append(v.errors, fmt.Sprintf("Plan mode failed: %v", v.result.PlanResult.Error))
	}
	return v
}

// PlanFails validates that plan mode failed with expected error
func (v *ValidationAssertions) PlanFails(expectedErrorContains string) *ValidationAssertions {
	if v.result.PlanResult.Success {
		v.errors = append(v.errors, "Expected plan mode to fail, but it succeeded")
	} else if expectedErrorContains != "" {
		if !strings.Contains(v.result.PlanResult.Error.Error(), expectedErrorContains) {
			v.errors = append(v.errors, fmt.Sprintf("Plan error should contain %q, got: %v",
				expectedErrorContains, v.result.PlanResult.Error))
		}
	}
	return v
}

// PlanReturnsElement validates that plan mode returns a valid plan element
func (v *ValidationAssertions) PlanReturnsElement(elementType string) *ValidationAssertions {
	if !v.result.PlanResult.Success {
		return v
	}

	if v.result.PlanResult.Data == nil {
		v.errors = append(v.errors, "Plan mode should return plan element, got nil")
		return v
	}

	// Check if it's a plan element (this would depend on actual plan types)
	switch element := v.result.PlanResult.Data.(type) {
	case *plan.ExecutionStep:
		if elementType != "" && string(element.Type) != elementType {
			v.errors = append(v.errors, fmt.Sprintf("Plan element type should be %q, got %q", elementType, string(element.Type)))
		}
	default:
		// For now, just ensure it's not nil - we'd add more specific validation based on plan types
		if element == nil {
			v.errors = append(v.errors, "Plan element should not be nil")
		}
	}

	return v
}

// === PERFORMANCE VALIDATIONS ===

// CompletesWithin validates that all modes complete within expected time
func (v *ValidationAssertions) CompletesWithin(maxDuration ...string) *ValidationAssertions {
	// Default to 1 second if not specified
	defaultMax := "1s"
	if len(maxDuration) > 0 {
		defaultMax = maxDuration[0]
	}

	// This is a simplified version - would parse duration string in real implementation
	// For now, check that nothing takes absurdly long (> 5 seconds)
	maxNanos := int64(5 * 1000 * 1000 * 1000) // 5 seconds in nanoseconds

	if v.result.InterpreterResult.Duration.Nanoseconds() > maxNanos {
		v.errors = append(v.errors, fmt.Sprintf("Interpreter mode took too long: %v", v.result.InterpreterResult.Duration))
	}

	if v.result.GeneratorResult.Duration.Nanoseconds() > maxNanos {
		v.errors = append(v.errors, fmt.Sprintf("Generator mode took too long: %v", v.result.GeneratorResult.Duration))
	}

	if v.result.PlanResult.Duration.Nanoseconds() > maxNanos {
		v.errors = append(v.errors, fmt.Sprintf("Plan mode took too long: %v", v.result.PlanResult.Duration))
	}

	_ = defaultMax // Use it when we implement proper duration parsing
	return v
}

// === CROSS-MODE VALIDATIONS ===

// ModesAreConsistent validates that all modes handle parameters consistently
func (v *ValidationAssertions) ModesAreConsistent() *ValidationAssertions {
	// Check for cross-mode consistency issues
	// For example, if one mode fails due to invalid parameters, others should too

	interpreterFailed := !v.result.InterpreterResult.Success
	generatorFailed := !v.result.GeneratorResult.Success
	planFailed := !v.result.PlanResult.Success

	// Parameter validation errors should be consistent across modes
	if interpreterFailed && generatorFailed && planFailed {
		// All failed - check if they failed for parameter reasons
		interpErr := v.result.InterpreterResult.Error.Error()
		genErr := v.result.GeneratorResult.Error.Error()
		planErr := v.result.PlanResult.Error.Error()

		if strings.Contains(interpErr, "parameter") || strings.Contains(interpErr, "required") {
			if !strings.Contains(genErr, "parameter") && !strings.Contains(genErr, "required") {
				v.errors = append(v.errors, "Parameter validation inconsistent between interpreter and generator modes")
			}
			if !strings.Contains(planErr, "parameter") && !strings.Contains(planErr, "required") {
				v.errors = append(v.errors, "Parameter validation inconsistent between interpreter and plan modes")
			}
		}
	}

	return v
}

// === STRUCTURAL VALIDATIONS ===

// ValidatesParameters validates that the decorator properly validates its parameter schema
func (v *ValidationAssertions) ValidatesParameters(decorator decorators.Decorator, invalidParams []coreast.NamedParameter) *ValidationAssertions {
	// This would test parameter validation by trying invalid parameters
	// For now, just check that the decorator has a parameter schema
	schema := decorator.ParameterSchema()

	// Check that required parameters are marked as required
	for _, param := range schema {
		if param.Required && param.Name == "" {
			v.errors = append(v.errors, "Required parameter has empty name in schema")
		}
	}

	return v
}

// SupportsDevcmdChaining validates that the decorator works properly in command chaining scenarios
func (v *ValidationAssertions) SupportsDevcmdChaining() *ValidationAssertions {
	// This would test that the decorator can be chained with other decorators
	// For now, just verify that outputs are compatible with chaining

	// Generator mode should produce chainable code
	if v.result.GeneratorResult.Success {
		if code, ok := v.result.GeneratorResult.Data.(string); ok {
			// Code should be standalone or return appropriate values for chaining
			if strings.TrimSpace(code) == "" {
				v.errors = append(v.errors, "Generated code is empty - not suitable for chaining")
			}
		}
	}

	return v
}

// SupportsNesting validates that the decorator works properly when nested inside other decorators
func (v *ValidationAssertions) SupportsNesting() *ValidationAssertions {
	// This would test nesting scenarios
	// For now, just verify basic structural requirements

	// Plan mode should produce elements that can be nested
	if v.result.PlanResult.Success && v.result.PlanResult.Data == nil {
		v.errors = append(v.errors, "Plan mode returns nil - not suitable for nesting")
	}

	return v
}

// === FINALIZATION ===

// Validate completes the validation and reports any errors
func (v *ValidationAssertions) Validate() []string {
	allErrors := append(v.errors, v.result.ValidationErrors...)
	return allErrors
}

// === UTILITY FUNCTIONS ===

// Helper function to create simple shell content for testing
func Shell(text string) coreast.CommandContent {
	return &coreast.ShellContent{
		Parts: []coreast.ShellPart{
			&coreast.TextPart{Text: text},
		},
	}
}

// Helper function to create string parameters
func StringParam(name, value string) coreast.NamedParameter {
	return coreast.NamedParameter{
		Name:  name,
		Value: &coreast.StringLiteral{Value: value},
	}
}

// Helper function to create boolean parameters
func BoolParam(name string, value bool) coreast.NamedParameter {
	return coreast.NamedParameter{
		Name:  name,
		Value: &coreast.BooleanLiteral{Value: value},
	}
}

// Helper function to create identifier parameters
func IdentifierParam(name, value string) coreast.NamedParameter {
	return coreast.NamedParameter{
		Name:  name,
		Value: &coreast.Identifier{Name: value},
	}
}

// Helper function to create pattern branches for testing
func PatternBranch(pattern string, commands ...string) coreast.PatternBranch {
	var patternNode coreast.Pattern
	if pattern == "*" || pattern == "default" {
		patternNode = &coreast.WildcardPattern{}
	} else {
		patternNode = &coreast.IdentifierPattern{Name: pattern}
	}

	content := make([]coreast.CommandContent, len(commands))
	for i, cmd := range commands {
		content[i] = Shell(cmd)
	}

	return coreast.PatternBranch{
		Pattern:  patternNode,
		Commands: content,
	}
}

// === COMMON TEST UTILITIES ===

// JoinErrors combines multiple error messages into a single formatted string
func JoinErrors(errors []string) string {
	result := ""
	for i, err := range errors {
		result += err
		if i < len(errors)-1 {
			result += "\n"
		}
	}
	return result
}

// ContainsExecutionEvidence checks if generated code contains evidence of actual execution
// This is a security-critical check to ensure generator mode never executes commands
func ContainsExecutionEvidence(code string) bool {
	dangerousPatterns := []string{
		"EXECUTION_DETECTED",
		"Command was executed",
		"DANGER:",
		"rm -rf",
		"exit 1", // If this appears in output, command was executed
		// Add more patterns that would indicate actual execution
	}

	for _, pattern := range dangerousPatterns {
		if ContainsString(code, pattern) {
			return true
		}
	}
	return false
}

// ContainsString checks if text contains substr (simple implementation)
func ContainsString(text, substr string) bool {
	return len(text) > 0 && len(substr) > 0 &&
		len(text) >= len(substr) &&
		findSubstring(text, substr)
}

// findSubstring is a helper for ContainsString
func findSubstring(text, substr string) bool {
	for i := 0; i <= len(text)-len(substr); i++ {
		if text[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// ContainsAll checks if text contains all of the provided substrings
func ContainsAll(text string, substrings []string) bool {
	for _, substr := range substrings {
		if !ContainsString(text, substr) {
			return false
		}
	}
	return true
}

// RandomString generates a random string of specified length for test isolation
func RandomString(length int) string {
	const charset = "abcdefghijklmnopqrstuvwxyz0123456789"
	b := make([]byte, length)
	for i := range b {
		// Use a simple deterministic approach for tests to avoid importing crypto/rand
		b[i] = charset[(i*7+length*3)%len(charset)]
	}
	return string(b)
}

// compileAndExecuteGenerated compiles and executes generated Go code with timeout protection
func (v *ValidationAssertions) compileAndExecuteGenerated(program string) error {
	// Create a temporary directory for the test program
	tmpDir, err := ioutil.TempDir("", "devcmd_validation_test_")
	if err != nil {
		return fmt.Errorf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Write the program to a Go file
	mainFile := filepath.Join(tmpDir, "main.go")
	if err := ioutil.WriteFile(mainFile, []byte(program), 0o644); err != nil {
		return fmt.Errorf("failed to write test program: %v", err)
	}

	// Initialize go module
	goModContent := `module testvalidation

go 1.21
`
	if err := ioutil.WriteFile(filepath.Join(tmpDir, "go.mod"), []byte(goModContent), 0o644); err != nil {
		return fmt.Errorf("failed to write go.mod: %v", err)
	}

	// Compile the program with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	buildCmd := exec.CommandContext(ctx, "go", "build", "-o", "testprogram", "main.go")
	buildCmd.Dir = tmpDir
	buildOutput, err := buildCmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("compilation failed: %v\nOutput: %s\nProgram:\n%s", err, string(buildOutput), program)
	}

	// Execute the compiled program with timeout
	execCtx, execCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer execCancel()

	execCmd := exec.CommandContext(execCtx, "./testprogram")
	execCmd.Dir = tmpDir
	execOutput, err := execCmd.CombinedOutput()
	if err != nil {
		// Check if it was a timeout
		if execCtx.Err() == context.DeadlineExceeded {
			return fmt.Errorf("execution timeout (10s) - generated code likely hangs\nOutput: %s", string(execOutput))
		}
		return fmt.Errorf("execution failed: %v\nOutput: %s", err, string(execOutput))
	}

	// Check that execution completed (should contain our success message)
	if !strings.Contains(string(execOutput), "Execution completed successfully") {
		return fmt.Errorf("execution did not complete successfully\nOutput: %s", string(execOutput))
	}

	return nil
}
