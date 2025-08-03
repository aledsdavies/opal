package decorators

import (
	"os"
	"testing"

	"github.com/aledsdavies/devcmd/core/ast"
	decoratortesting "github.com/aledsdavies/devcmd/testing"
)

func TestEnvDecorator_Basic(t *testing.T) {
	decorator := &EnvDecorator{}
	
	// Set environment variable for test
	os.Setenv("TEST_ENV_VAR", "env_value")
	defer os.Unsetenv("TEST_ENV_VAR")
	
	result := decoratortesting.NewDecoratorTest(t, decorator).
		TestValueDecorator([]ast.NamedParameter{
			decoratortesting.StringParam("key", "TEST_ENV_VAR"),
		})
	
	errors := decoratortesting.Assert(result).
		InterpreterSucceeds().
		InterpreterReturns("env_value").
		GeneratorSucceeds().
		GeneratorCodeContains("TEST_ENV_VAR").
		PlanSucceeds().
		CompletesWithin("50ms").
		Validate()
	
	if len(errors) > 0 {
		t.Errorf("EnvDecorator basic test failed:\n%s", decoratortesting.JoinErrors(errors))
	}
}

func TestEnvDecorator_WithDefault(t *testing.T) {
	decorator := &EnvDecorator{}
	
	// Test undefined env var with default
	result := decoratortesting.NewDecoratorTest(t, decorator).
		TestValueDecorator([]ast.NamedParameter{
			decoratortesting.StringParam("key", "UNDEFINED_ENV_VAR"),
			decoratortesting.StringParam("default", "default_value"),
		})
	
	errors := decoratortesting.Assert(result).
		InterpreterSucceeds().
		InterpreterReturns("default_value").
		GeneratorSucceeds().
		GeneratorCodeContains("UNDEFINED_ENV_VAR", "default_value").
		PlanSucceeds().
		Validate()
	
	if len(errors) > 0 {
		t.Errorf("EnvDecorator with default test failed:\n%s", decoratortesting.JoinErrors(errors))
	}
}

func TestEnvDecorator_UndefinedNoDefault(t *testing.T) {
	decorator := &EnvDecorator{}
	
	// Test undefined env var without default
	result := decoratortesting.NewDecoratorTest(t, decorator).
		TestValueDecorator([]ast.NamedParameter{
			decoratortesting.StringParam("key", "UNDEFINED_ENV_VAR"),
		})
	
	errors := decoratortesting.Assert(result).
		InterpreterSucceeds().
		InterpreterReturns(""). // Empty string for undefined env vars
		GeneratorSucceeds().
		PlanSucceeds().
		Validate()
	
	if len(errors) > 0 {
		t.Errorf("EnvDecorator undefined no default test failed:\n%s", decoratortesting.JoinErrors(errors))
	}
}

func TestEnvDecorator_EmptyValue_AllowEmpty(t *testing.T) {
	decorator := &EnvDecorator{}
	
	// Set empty environment variable
	os.Setenv("EMPTY_ENV_VAR", "")
	defer os.Unsetenv("EMPTY_ENV_VAR")
	
	result := decoratortesting.NewDecoratorTest(t, decorator).
		TestValueDecorator([]ast.NamedParameter{
			decoratortesting.StringParam("key", "EMPTY_ENV_VAR"),
			decoratortesting.StringParam("default", "should_not_be_used"),
			decoratortesting.BoolParam("allowEmpty", true), // Allow empty values
		})
	
	errors := decoratortesting.Assert(result).
		InterpreterSucceeds().
		InterpreterReturns(""). // Empty env var should return empty when allowEmpty=true
		GeneratorSucceeds().
		PlanSucceeds().
		Validate()
	
	if len(errors) > 0 {
		t.Errorf("EnvDecorator empty value (allowEmpty=true) test failed:\n%s", decoratortesting.JoinErrors(errors))
	}
}

func TestEnvDecorator_EmptyValue_DefaultBehavior(t *testing.T) {
	decorator := &EnvDecorator{}
	
	// Set empty environment variable
	os.Setenv("EMPTY_ENV_VAR", "")
	defer os.Unsetenv("EMPTY_ENV_VAR")
	
	result := decoratortesting.NewDecoratorTest(t, decorator).
		TestValueDecorator([]ast.NamedParameter{
			decoratortesting.StringParam("key", "EMPTY_ENV_VAR"),
			decoratortesting.StringParam("default", "default_value"),
			// No allowEmpty parameter - defaults to false
		})
	
	errors := decoratortesting.Assert(result).
		InterpreterSucceeds().
		InterpreterReturns("default_value"). // Empty env var should use default when allowEmpty=false (default)
		GeneratorSucceeds().
		PlanSucceeds().
		Validate()
	
	if len(errors) > 0 {
		t.Errorf("EnvDecorator empty value (default behavior) test failed:\n%s", decoratortesting.JoinErrors(errors))
	}
}

func TestEnvDecorator_NoParameter(t *testing.T) {
	decorator := &EnvDecorator{}
	
	// Test missing required parameter
	result := decoratortesting.NewDecoratorTest(t, decorator).
		TestValueDecorator([]ast.NamedParameter{})
	
	errors := decoratortesting.Assert(result).
		InterpreterFails("requires at least 1 parameter").
		GeneratorFails("requires at least 1 parameter").
		PlanFails("requires at least 1 parameter").
		Validate()
	
	if len(errors) > 0 {
		t.Errorf("EnvDecorator no parameter test failed:\n%s", decoratortesting.JoinErrors(errors))
	}
}

func TestEnvDecorator_GlobalTracking(t *testing.T) {
	decorator := &EnvDecorator{}
	
	// Test that env vars are tracked globally for generator mode
	os.Setenv("TRACKED_VAR", "tracked_value")
	defer os.Unsetenv("TRACKED_VAR")
	
	result := decoratortesting.NewDecoratorTest(t, decorator).
		TestValueDecorator([]ast.NamedParameter{
			decoratortesting.StringParam("key", "TRACKED_VAR"),
		})
	
	errors := decoratortesting.Assert(result).
		GeneratorSucceeds().
		// Should contain env var access patterns (using captured environment context)
		GeneratorCodeContains("envContext", "TRACKED_VAR").
		Validate()
	
	if len(errors) > 0 {
		t.Errorf("EnvDecorator global tracking test failed:\n%s", decoratortesting.JoinErrors(errors))
	}
}