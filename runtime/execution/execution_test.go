package execution

import (
	"context"
	"fmt"
	"testing"

	"github.com/aledsdavies/devcmd/core/ast"
)

func TestExecutionResult_Helpers(t *testing.T) {
	// Test NewSuccessResult
	result := NewSuccessResult("test data")
	if result.Error != nil {
		t.Errorf("Expected no error, got %v", result.Error)
	}
	if result.Data != "test data" {
		t.Errorf("Expected 'test data', got %v", result.Data)
	}

	// Test NewErrorResult
	testErr := fmt.Errorf("test error")
	result = NewErrorResult(testErr)
	if result.Error == nil {
		t.Error("Expected error, got nil")
	}
	if result.Data != nil {
		t.Errorf("Expected nil data, got %v", result.Data)
	}

	// Test NewFormattedErrorResult
	result = NewFormattedErrorResult("error with value: %s", "test")
	if result.Error == nil {
		t.Error("Expected error, got nil")
	}
	if result.Error.Error() != "error with value: test" {
		t.Errorf("Expected 'error with value: test', got %v", result.Error.Error())
	}
}

func TestInterpreterContext_Creation(t *testing.T) {
	program := &ast.Program{
		Variables: []ast.VariableDecl{
			{Name: "TEST_VAR", Value: &ast.StringLiteral{Value: "test_value"}},
		},
	}

	ctx := NewInterpreterContext(context.Background(), program)

	// Test that it implements InterpreterContext interface
	var _ InterpreterContext = ctx

	// Test basic properties
	if ctx.GetProgram() != program {
		t.Error("Expected program to be set correctly")
	}
}

func TestGeneratorContext_Creation(t *testing.T) {
	program := &ast.Program{
		Variables: []ast.VariableDecl{
			{Name: "TEST_VAR", Value: &ast.StringLiteral{Value: "test_value"}},
		},
	}

	ctx := NewGeneratorContext(context.Background(), program)

	// Test that it implements GeneratorContext interface
	var _ GeneratorContext = ctx

	// Test basic properties
	if ctx.GetProgram() != program {
		t.Error("Expected program to be set correctly")
	}
}

func TestPlanContext_Creation(t *testing.T) {
	program := &ast.Program{
		Variables: []ast.VariableDecl{
			{Name: "TEST_VAR", Value: &ast.StringLiteral{Value: "test_value"}},
		},
	}

	ctx := NewPlanContext(context.Background(), program)

	// Test that it implements PlanContext interface
	var _ PlanContext = ctx

	// Test basic properties
	if ctx.GetProgram() != program {
		t.Error("Expected program to be set correctly")
	}
}

func TestContextVariables(t *testing.T) {
	program := &ast.Program{
		Variables: []ast.VariableDecl{
			{Name: "TEST_VAR", Value: &ast.StringLiteral{Value: "test_value"}},
			{Name: "NUM_VAR", Value: &ast.NumberLiteral{Value: "42"}},
		},
	}

	// Test with InterpreterContext
	ctx := NewInterpreterContext(context.Background(), program)

	// Initialize variables from program
	err := ctx.InitializeVariables()
	if err != nil {
		t.Errorf("Failed to initialize variables: %v", err)
		return
	}

	// Test variable retrieval
	value, exists := ctx.GetVariable("TEST_VAR")
	if !exists {
		t.Error("Expected TEST_VAR to exist")
	}
	if value != "test_value" {
		t.Errorf("Expected 'test_value', got %s", value)
	}

	// Test non-existent variable
	_, exists = ctx.GetVariable("NON_EXISTENT")
	if exists {
		t.Error("Expected NON_EXISTENT to not exist")
	}

	// Test setting variable
	ctx.SetVariable("NEW_VAR", "new_value")
	value, exists = ctx.GetVariable("NEW_VAR")
	if !exists {
		t.Error("Expected NEW_VAR to exist after setting")
	}
	if value != "new_value" {
		t.Errorf("Expected 'new_value', got %s", value)
	}
}

func TestShellCodeBuilder_Creation(t *testing.T) {
	program := &ast.Program{}
	ctx := NewGeneratorContext(context.Background(), program)

	builder := NewShellCodeBuilder(ctx)
	if builder == nil {
		t.Error("Expected ShellCodeBuilder to be created")
	}
}

func TestChainElement_Creation(t *testing.T) {
	// Test creating different types of chain elements
	textElement := ChainElement{
		Type: "text",
		Text: "echo hello",
	}
	if textElement.Type != "text" || textElement.Text != "echo hello" {
		t.Error("Text chain element not created correctly")
	}

	operatorElement := ChainElement{
		Type:     "operator",
		Operator: "&&",
	}
	if operatorElement.Type != "operator" || operatorElement.Operator != "&&" {
		t.Error("Operator chain element not created correctly")
	}

	actionElement := ChainElement{
		Type:       "action",
		ActionName: "cmd",
		ActionArgs: []ast.NamedParameter{
			{Name: "command", Value: &ast.StringLiteral{Value: "test"}},
		},
	}
	if actionElement.Type != "action" || actionElement.ActionName != "cmd" {
		t.Error("Action chain element not created correctly")
	}
}

func TestCommandResult_Creation(t *testing.T) {
	result := CommandResult{
		Stdout:   "output text",
		Stderr:   "error text",
		ExitCode: 0,
	}

	if result.Stdout != "output text" {
		t.Errorf("Expected 'output text', got %s", result.Stdout)
	}
	if result.ExitCode != 0 {
		t.Errorf("Expected 0, got %d", result.ExitCode)
	}
	if !result.Success() {
		t.Error("Expected success to be true")
	}

	// Test failure case
	failResult := CommandResult{
		Stdout:   "",
		Stderr:   "error",
		ExitCode: 1,
	}
	if failResult.Success() {
		t.Error("Expected success to be false for non-zero exit code")
	}
}
