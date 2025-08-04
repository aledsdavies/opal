package decorators

import (
	"fmt"
	"strings"
	"text/template"

	"github.com/aledsdavies/devcmd/core/ast"
	"github.com/aledsdavies/devcmd/runtime/execution"
)

// TemplateBuilder helps decorators generate Go code using generic patterns
type TemplateBuilder struct {
	imports   map[string]bool
	patterns  []string
	variables map[string]interface{}
}

// NewTemplateBuilder creates a new template builder
func NewTemplateBuilder() *TemplateBuilder {
	return &TemplateBuilder{
		imports:   make(map[string]bool),
		variables: make(map[string]interface{}),
	}
}

// Operation represents a code block that can be executed
type Operation struct {
	Code string
}

// WithConcurrentExecution adds concurrent execution pattern
func (tb *TemplateBuilder) WithConcurrentExecution(maxConcurrency int, operations []Operation) *TemplateBuilder {
	tb.addImports(PatternImports["ConcurrentExecutionPattern"]...)
	tb.variables["MaxConcurrency"] = maxConcurrency
	tb.variables["Operations"] = operations
	tb.patterns = append(tb.patterns, ConcurrentExecutionPattern)
	return tb
}

// WithTimeout adds timeout pattern around an operation
func (tb *TemplateBuilder) WithTimeout(duration string, operation Operation) *TemplateBuilder {
	tb.addImports(PatternImports["TimeoutPattern"]...)
	tb.variables["Duration"] = duration
	tb.variables["Operation"] = operation
	tb.patterns = append(tb.patterns, TimeoutPattern)
	return tb
}

// WithTimeoutExpr adds timeout pattern with pre-validated Go duration expression
func (tb *TemplateBuilder) WithTimeoutExpr(durationExpr string, operation Operation) *TemplateBuilder {
	tb.addImports(PatternImports["TimeoutPattern"]...)
	tb.variables["DurationExpr"] = durationExpr
	tb.variables["Operation"] = operation
	tb.patterns = append(tb.patterns, TimeoutPattern)
	return tb
}

// WithRetry adds retry pattern around an operation
func (tb *TemplateBuilder) WithRetry(maxAttempts int, delayDuration string, operation Operation) *TemplateBuilder {
	tb.addImports(PatternImports["RetryPattern"]...)
	tb.variables["MaxAttempts"] = maxAttempts
	tb.variables["DelayDuration"] = delayDuration
	tb.variables["Operation"] = operation
	tb.patterns = append(tb.patterns, RetryPattern)
	return tb
}

// WithRetryExpr adds retry pattern with pre-validated Go duration expression
func (tb *TemplateBuilder) WithRetryExpr(maxAttempts int, delayExpr string, operation Operation) *TemplateBuilder {
	tb.addImports(PatternImports["RetryPattern"]...)
	tb.variables["MaxAttempts"] = maxAttempts
	tb.variables["DelayExpr"] = delayExpr
	tb.variables["Operation"] = operation
	tb.patterns = append(tb.patterns, RetryPattern)
	return tb
}

// WithCancellation adds cancellation support around an operation
func (tb *TemplateBuilder) WithCancellation(operation Operation) *TemplateBuilder {
	tb.addImports(PatternImports["CancellableOperationPattern"]...)
	tb.variables["Operation"] = operation
	tb.patterns = append(tb.patterns, CancellableOperationPattern)
	return tb
}

// WithSequentialExecution adds sequential execution pattern
func (tb *TemplateBuilder) WithSequentialExecution(operations []Operation, stopOnError bool) *TemplateBuilder {
	tb.addImports(PatternImports["SequentialExecutionPattern"]...)
	tb.variables["Operations"] = operations
	tb.variables["StopOnError"] = stopOnError
	tb.patterns = append(tb.patterns, SequentialExecutionPattern)
	return tb
}

// WithConditionalExecution adds conditional execution pattern
func (tb *TemplateBuilder) WithConditionalExecution(condition Operation, thenOp Operation, elseOp *Operation) *TemplateBuilder {
	tb.addImports(PatternImports["ConditionalExecutionPattern"]...)
	tb.variables["Condition"] = condition
	tb.variables["ThenOperation"] = thenOp
	if elseOp != nil {
		tb.variables["ElseOperation"] = elseOp
	}
	tb.patterns = append(tb.patterns, ConditionalExecutionPattern)
	return tb
}

// WithResourceCleanup adds resource cleanup pattern
func (tb *TemplateBuilder) WithResourceCleanup(setupCode string, operation Operation, cleanupCode string) *TemplateBuilder {
	tb.addImports(PatternImports["ResourceCleanupPattern"]...)
	tb.variables["SetupCode"] = setupCode
	tb.variables["Operation"] = operation
	tb.variables["CleanupCode"] = cleanupCode
	tb.patterns = append(tb.patterns, ResourceCleanupPattern)
	return tb
}

// WithErrorCollection adds error collection pattern
func (tb *TemplateBuilder) WithErrorCollection(operations []Operation, continueOnError bool) *TemplateBuilder {
	tb.addImports(PatternImports["ErrorCollectionPattern"]...)
	tb.variables["Operations"] = operations
	tb.variables["ContinueOnError"] = continueOnError
	tb.patterns = append(tb.patterns, ErrorCollectionPattern)
	return tb
}

// WithTryCatchFinally adds try-catch-finally pattern
func (tb *TemplateBuilder) WithTryCatchFinally(mainOp Operation, catchOp *Operation, finallyOp *Operation) *TemplateBuilder {
	tb.addImports(PatternImports["TryCatchFinallyPattern"]...)
	tb.variables["MainOperation"] = mainOp
	tb.variables["HasCatch"] = catchOp != nil
	tb.variables["HasFinally"] = finallyOp != nil
	
	if catchOp != nil {
		tb.variables["CatchOperation"] = *catchOp
	}
	if finallyOp != nil {
		tb.variables["FinallyOperation"] = *finallyOp
	}
	
	tb.patterns = append(tb.patterns, TryCatchFinallyPattern)
	return tb
}

// AddImport manually adds an import
func (tb *TemplateBuilder) AddImport(pkg string) *TemplateBuilder {
	tb.imports[pkg] = true
	return tb
}

// addImports adds multiple imports
func (tb *TemplateBuilder) addImports(pkgs ...string) {
	for _, pkg := range pkgs {
		tb.imports[pkg] = true
	}
}

// BuildTemplate generates the complete Go code template
func (tb *TemplateBuilder) BuildTemplate() (string, error) {
	if len(tb.patterns) == 0 {
		return "", fmt.Errorf("no patterns added to template builder")
	}

	// Combine all patterns
	combined := strings.Join(tb.patterns, "\n\n")

	// Parse and execute template
	tmpl, err := template.New("builder").Parse(combined)
	if err != nil {
		return "", fmt.Errorf("failed to parse template: %w", err)
	}

	var result strings.Builder
	if err := tmpl.Execute(&result, tb.variables); err != nil {
		return "", fmt.Errorf("failed to execute template: %w", err)
	}

	return result.String(), nil
}

// BuildCommandResultTemplate builds a template that returns CommandResult instead of error
// This wraps the error-based patterns to work in CLI generation context
func (tb *TemplateBuilder) BuildCommandResultTemplate() (string, error) {
	innerTemplate, err := tb.BuildTemplate()
	if err != nil {
		return "", err
	}
	
	// Wrap the template so it returns CommandResult instead of error
	wrappedTemplate := fmt.Sprintf(`{
		if err := func() error %s; err != nil {
			return CommandResult{Stdout: "", Stderr: err.Error(), ExitCode: 1}
		}
		return CommandResult{Stdout: "", Stderr: "", ExitCode: 0}
	}`, innerTemplate)
	
	return wrappedTemplate, nil
}

// GetRequiredImports returns the required imports for this template
func (tb *TemplateBuilder) GetRequiredImports() ImportRequirement {
	standardLib := make([]string, 0, len(tb.imports))
	for pkg := range tb.imports {
		standardLib = append(standardLib, pkg)
	}

	return ImportRequirement{
		StandardLibrary: standardLib,
		ThirdParty:      []string{},
		GoModules:       map[string]string{},
	}
}

// ConvertCommandsToOperations converts AST commands to Operation structs using ShellCodeBuilder
func ConvertCommandsToOperations(ctx execution.GeneratorContext, commands []ast.CommandContent) ([]Operation, error) {
	operations := make([]Operation, len(commands))
	shellBuilder := execution.NewShellCodeBuilder(ctx)

	for i, cmd := range commands {
		code, err := shellBuilder.GenerateShellCodeWithReturn(cmd)
		if err != nil {
			return nil, fmt.Errorf("failed to generate code for command %d: %w", i, err)
		}
		operations[i] = Operation{Code: code}
	}

	return operations, nil
}

// WrapOperationInTemplate is a helper to wrap a single operation in a pattern
func WrapOperationInTemplate(pattern string, variables map[string]interface{}) (string, error) {
	tmpl, err := template.New("wrapper").Parse(pattern)
	if err != nil {
		return "", fmt.Errorf("failed to parse pattern template: %w", err)
	}

	var result strings.Builder
	if err := tmpl.Execute(&result, variables); err != nil {
		return "", fmt.Errorf("failed to execute pattern template: %w", err)
	}

	return result.String(), nil
}