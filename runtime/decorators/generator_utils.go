package decorators

import (
	"fmt"
	"hash/fnv"
	"time"

	"github.com/aledsdavies/devcmd/core/ast"
	"github.com/aledsdavies/devcmd/runtime/execution"
)

// CommandResultFunction represents a function that returns CommandResult code for generation
type CommandResultFunction func() string

// CommandResultExecutor provides utilities for CommandResult-based code generation
type CommandResultExecutor struct {
	ctx execution.GeneratorContext
}

// NewCommandResultExecutor creates a new CommandResult executor for code generation
func NewCommandResultExecutor(ctx execution.GeneratorContext) *CommandResultExecutor {
	return &CommandResultExecutor{
		ctx: ctx,
	}
}

// ================================================================================================
// COMMANDRESULT INLINE GENERATION FUNCTIONS
// ================================================================================================

// GenerateCommandResultDefinition generates the complete CommandResult struct definition and methods as Go code
func GenerateCommandResultDefinition() string {
	return `// CommandResult represents the structured output from command execution
type CommandResult struct {
	Stdout   string // Standard output as string
	Stderr   string // Standard error as string  
	ExitCode int    // Exit code (0 = success)
}

// Success returns true if the command executed successfully (exit code 0)
func (r CommandResult) Success() bool {
	return r.ExitCode == 0
}

// Failed returns true if the command failed (non-zero exit code)
func (r CommandResult) Failed() bool {
	return r.ExitCode != 0
}

// ToError returns an error representation when the command failed
func (r CommandResult) ToError() error {
	if r.Success() {
		return nil
	}
	if r.Stderr != "" {
		return fmt.Errorf("exit code %d: %s", r.ExitCode, r.Stderr)
	}
	return fmt.Errorf("exit code %d", r.ExitCode)
}`
}

// GenerateCommandResultSuccess generates Go code for a success CommandResult
func GenerateCommandResultSuccess(stdout, stderr string) string {
	return fmt.Sprintf("CommandResult{Stdout: %q, Stderr: %q, ExitCode: 0}", stdout, stderr)
}

// GenerateCommandResultError generates Go code for an error CommandResult
func GenerateCommandResultError(stderr string, exitCode int) string {
	return fmt.Sprintf("CommandResult{Stdout: \"\", Stderr: %q, ExitCode: %d}", stderr, exitCode)
}

// GenerateFailureCheck generates Go code to check if a CommandResult failed
func GenerateFailureCheck(resultVar string) string {
	return fmt.Sprintf("%s.Failed()", resultVar)
}

// GenerateSuccessCheck generates Go code to check if a CommandResult succeeded
func GenerateSuccessCheck(resultVar string) string {
	return fmt.Sprintf("!%s.Failed()", resultVar)
}

// ConvertCommandsToCommandResultOperations converts AST commands to operations that return CommandResult
func (cre *CommandResultExecutor) ConvertCommandsToCommandResultOperations(commands []ast.CommandContent) ([]Operation, error) {
	operations := make([]Operation, len(commands))
	shellBuilder := execution.NewShellCodeBuilder(cre.ctx)

	for i, cmd := range commands {
		code, err := shellBuilder.GenerateShellCodeWithReturn(cmd)
		if err != nil {
			return nil, fmt.Errorf("failed to generate code for command %d: %w", i, err)
		}
		operations[i] = Operation{Code: code}
	}

	return operations, nil
}

// GenerateSequentialExecution generates code for sequential execution with CommandResult handling
func (cre *CommandResultExecutor) GenerateSequentialExecution(operations []Operation, stopOnError bool) (string, error) {
	if len(operations) == 0 {
		return "return CommandResult{Stdout: \"\", Stderr: \"\", ExitCode: 0}", nil
	}

	if len(operations) == 1 {
		return operations[0].Code, nil
	}

	builder := NewTemplateBuilder()
	builder.WithSequentialExecution(operations, stopOnError)
	return builder.BuildTemplate()
}

// GenerateConcurrentExecution generates code for concurrent execution with CommandResult handling
func (cre *CommandResultExecutor) GenerateConcurrentExecution(operations []Operation, maxConcurrency int) (string, error) {
	if len(operations) == 0 {
		return "return CommandResult{Stdout: \"\", Stderr: \"\", ExitCode: 0}", nil
	}

	if len(operations) == 1 {
		return operations[0].Code, nil
	}

	builder := NewTemplateBuilder()
	builder.WithConcurrentExecution(maxConcurrency, operations)
	return builder.BuildTemplate()
}

// GenerateTimeoutWrapper generates code for timeout wrapper with CommandResult handling
func (cre *CommandResultExecutor) GenerateTimeoutWrapper(operation Operation, timeout time.Duration) (string, error) {
	builder := NewTemplateBuilder()
	durationExpr := DurationToGoExpr(timeout)
	builder.WithTimeoutExpr(durationExpr, operation)
	return builder.BuildTemplate()
}

// GenerateRetryWrapper generates code for retry wrapper with CommandResult handling
func (cre *CommandResultExecutor) GenerateRetryWrapper(operation Operation, maxAttempts int, delay time.Duration) (string, error) {
	builder := NewTemplateBuilder()
	delayExpr := DurationToGoExpr(delay)
	builder.WithRetryExpr(maxAttempts, delayExpr, operation)
	return builder.BuildTemplate()
}

// GenerateTryCatchFinally generates code for try-catch-finally with CommandResult handling
func (cre *CommandResultExecutor) GenerateTryCatchFinally(mainOp Operation, catchOp *Operation, finallyOp *Operation) (string, error) {
	builder := NewTemplateBuilder()
	builder.WithTryCatchFinally(mainOp, catchOp, finallyOp)
	return builder.BuildTemplate()
}

// GenerateResourceCleanup generates code for resource cleanup with CommandResult handling
func (cre *CommandResultExecutor) GenerateResourceCleanup(setupCode string, operation Operation, cleanupCode string) (string, error) {
	builder := NewTemplateBuilder()
	builder.WithResourceCleanup(setupCode, operation, cleanupCode)
	return builder.BuildTemplate()
}

// GenerateConditionalExecution generates code for conditional execution with CommandResult handling
func (cre *CommandResultExecutor) GenerateConditionalExecution(condition Operation, thenOp Operation, elseOp *Operation) (string, error) {
	builder := NewTemplateBuilder()
	builder.WithConditionalExecution(condition, thenOp, elseOp)
	return builder.BuildTemplate()
}

// WrapWithCommandResultHandler wraps error-returning code to return CommandResult
func (cre *CommandResultExecutor) WrapWithCommandResultHandler(errorCode string) string {
	return fmt.Sprintf(`{
		if err := func() error %s; err != nil {
			return CommandResult{Stdout: "", Stderr: err.Error(), ExitCode: 1}
		}
		return CommandResult{Stdout: "", Stderr: "", ExitCode: 0}
	}`, errorCode)
}

// CommandResultChainBuilder helps build command result chains with proper error handling
type CommandResultChainBuilder struct {
	operations []Operation
	ctx        execution.GeneratorContext
}

// NewCommandResultChainBuilder creates a new command result chain builder
func NewCommandResultChainBuilder(ctx execution.GeneratorContext) *CommandResultChainBuilder {
	return &CommandResultChainBuilder{
		operations: make([]Operation, 0),
		ctx:        ctx,
	}
}

// AddOperation adds an operation to the chain
func (crcb *CommandResultChainBuilder) AddOperation(code string) *CommandResultChainBuilder {
	crcb.operations = append(crcb.operations, Operation{Code: code})
	return crcb
}

// AddShellCommand adds a shell command to the chain
func (crcb *CommandResultChainBuilder) AddShellCommand(command string) *CommandResultChainBuilder {
	shellBuilder := execution.NewShellCodeBuilder(crcb.ctx)
	shellContent := &ast.ShellContent{
		Parts: []ast.ShellPart{
			&ast.TextPart{Text: command},
		},
	}

	code, err := shellBuilder.GenerateShellCodeWithReturn(shellContent)
	if err != nil {
		// Add error handling operation
		code = fmt.Sprintf(`return CommandResult{Stdout: "", Stderr: "failed to generate shell command: %s", ExitCode: 1}`, err.Error())
	}

	return crcb.AddOperation(code)
}

// BuildSequential builds a sequential execution chain
func (crcb *CommandResultChainBuilder) BuildSequential(stopOnError bool) (string, error) {
	executor := NewCommandResultExecutor(crcb.ctx)
	return executor.GenerateSequentialExecution(crcb.operations, stopOnError)
}

// BuildConcurrent builds a concurrent execution chain
func (crcb *CommandResultChainBuilder) BuildConcurrent(maxConcurrency int) (string, error) {
	executor := NewCommandResultExecutor(crcb.ctx)
	return executor.GenerateConcurrentExecution(crcb.operations, maxConcurrency)
}

// BuildWithTimeout builds a timeout-wrapped execution
func (crcb *CommandResultChainBuilder) BuildWithTimeout(timeout time.Duration) (string, error) {
	if len(crcb.operations) == 0 {
		return "return CommandResult{Stdout: \"\", Stderr: \"\", ExitCode: 0}", nil
	}

	// First build sequential execution of all operations
	sequentialCode, err := crcb.BuildSequential(true)
	if err != nil {
		return "", err
	}

	operation := Operation{Code: sequentialCode}
	executor := NewCommandResultExecutor(crcb.ctx)
	return executor.GenerateTimeoutWrapper(operation, timeout)
}

// BuildWithRetry builds a retry-wrapped execution
func (crcb *CommandResultChainBuilder) BuildWithRetry(maxAttempts int, delay time.Duration) (string, error) {
	if len(crcb.operations) == 0 {
		return "return CommandResult{Stdout: \"\", Stderr: \"\", ExitCode: 0}", nil
	}

	// First build sequential execution of all operations
	sequentialCode, err := crcb.BuildSequential(true)
	if err != nil {
		return "", err
	}

	operation := Operation{Code: sequentialCode}
	executor := NewCommandResultExecutor(crcb.ctx)
	return executor.GenerateRetryWrapper(operation, maxAttempts, delay)
}

// CommandResultPatternMatcher provides pattern matching for CommandResult operations
type CommandResultPatternMatcher struct {
	patterns map[string]func(Operation) string
}

// NewCommandResultPatternMatcher creates a new pattern matcher
func NewCommandResultPatternMatcher() *CommandResultPatternMatcher {
	return &CommandResultPatternMatcher{
		patterns: make(map[string]func(Operation) string),
	}
}

// RegisterPattern registers a pattern transformation
func (crpm *CommandResultPatternMatcher) RegisterPattern(name string, transformer func(Operation) string) {
	crpm.patterns[name] = transformer
}

// ApplyPattern applies a registered pattern to an operation
func (crpm *CommandResultPatternMatcher) ApplyPattern(name string, operation Operation) (string, error) {
	transformer, exists := crpm.patterns[name]
	if !exists {
		return "", fmt.Errorf("pattern '%s' not registered", name)
	}

	return transformer(operation), nil
}

// Common pattern transformations

// WrapWithSuccessCheck wraps operation with success check
func WrapWithSuccessCheck(operation Operation) string {
	return fmt.Sprintf(`{
		result := func() CommandResult {
			%s
		}()
		if result.Failed() {
			return result
		}
		return CommandResult{Stdout: result.Stdout, Stderr: "", ExitCode: 0}
	}`, operation.Code)
}

// WrapWithErrorPropagation wraps operation with error propagation
func WrapWithErrorPropagation(operation Operation) string {
	return fmt.Sprintf(`{
		result := func() CommandResult {
			%s
		}()
		// Propagate any error
		return result
	}`, operation.Code)
}

// WrapWithOutputCapture wraps operation with output capture
func WrapWithOutputCapture(operation Operation, captureVar string) string {
	return fmt.Sprintf(`{
		result := func() CommandResult {
			%s
		}()
		%s := result.Stdout
		return result
	}`, operation.Code, captureVar)
}

// ============================================================================
// VARIABLE NAME GENERATION UTILITIES
// ============================================================================

// GenerateUniqueVarName generates a unique variable name based on input content
// This helps avoid variable name conflicts in generated code
func GenerateUniqueVarName(prefix, content string) string {
	h := fnv.New32a()
	h.Write([]byte(content))
	return fmt.Sprintf("%s%d", prefix, h.Sum32())
}

// GenerateUniqueContextVar generates a unique context variable name for decorators
func GenerateUniqueContextVar(prefix, path, additionalContent string) string {
	return GenerateUniqueVarName(prefix+"Ctx", path+additionalContent)
}

// GenerateUniqueResultVar generates a unique result variable name for shell commands
func GenerateUniqueResultVar(prefix, command, context string) string {
	return GenerateUniqueVarName(prefix+"Result", command+context)
}
