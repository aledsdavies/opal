package decorator

import (
	"testing"

	"github.com/aledsdavies/opal/core/types"
	"github.com/google/go-cmp/cmp"
)

// TestParamString_Basic tests basic string parameter creation
func TestParamString_Basic(t *testing.T) {
	desc := NewDescriptor("test").
		Summary("Test decorator").
		ParamString("name", "User name").
		Done().
		Build()

	param, exists := desc.Schema.Parameters["name"]
	if !exists {
		t.Fatal("parameter 'name' not found")
	}

	if param.Type != types.TypeString {
		t.Errorf("expected type string, got %v", param.Type)
	}
	if param.Description != "User name" {
		t.Errorf("expected description 'User name', got %q", param.Description)
	}
	if param.Required {
		t.Error("expected parameter to be optional by default")
	}
}

// TestParamString_Required tests required string parameter
func TestParamString_Required(t *testing.T) {
	desc := NewDescriptor("test").
		Summary("Test decorator").
		ParamString("name", "User name").
		Required().
		Done().
		Build()

	param := desc.Schema.Parameters["name"]
	if !param.Required {
		t.Error("expected parameter to be required")
	}
}

// TestParamString_Default tests string parameter with default value
func TestParamString_Default(t *testing.T) {
	desc := NewDescriptor("test").
		Summary("Test decorator").
		ParamString("name", "User name").
		Default("anonymous").
		Done().
		Build()

	param := desc.Schema.Parameters["name"]
	if param.Required {
		t.Error("parameter with default should be optional")
	}
	if param.Default != "anonymous" {
		t.Errorf("expected default 'anonymous', got %v", param.Default)
	}
}

// TestParamString_MinMaxLength tests string length constraints
func TestParamString_MinMaxLength(t *testing.T) {
	desc := NewDescriptor("test").
		Summary("Test decorator").
		ParamString("name", "User name").
		MinLength(3).
		MaxLength(50).
		Done().
		Build()

	param := desc.Schema.Parameters["name"]
	if param.MinLength == nil || *param.MinLength != 3 {
		t.Errorf("expected MinLength=3, got %v", param.MinLength)
	}
	if param.MaxLength == nil || *param.MaxLength != 50 {
		t.Errorf("expected MaxLength=50, got %v", param.MaxLength)
	}
}

// TestParamString_Pattern tests regex pattern constraint
func TestParamString_Pattern(t *testing.T) {
	desc := NewDescriptor("test").
		Summary("Test decorator").
		ParamString("email", "Email address").
		Pattern(`^[a-z0-9._%+-]+@[a-z0-9.-]+\.[a-z]{2,}$`).
		Done().
		Build()

	param := desc.Schema.Parameters["email"]
	if param.Pattern == nil {
		t.Fatal("expected pattern to be set")
	}
	expected := `^[a-z0-9._%+-]+@[a-z0-9.-]+\.[a-z]{2,}$`
	if *param.Pattern != expected {
		t.Errorf("expected pattern %q, got %q", expected, *param.Pattern)
	}
}

// TestParamString_Format tests typed format constraint
func TestParamString_Format(t *testing.T) {
	desc := NewDescriptor("test").
		Summary("Test decorator").
		ParamString("url", "Website URL").
		Format(types.FormatURI).
		Done().
		Build()

	param := desc.Schema.Parameters["url"]
	if param.Format == nil {
		t.Fatal("expected format to be set")
	}
	if *param.Format != types.FormatURI {
		t.Errorf("expected format URI, got %v", *param.Format)
	}
}

// TestParamString_Examples tests example values
func TestParamString_Examples(t *testing.T) {
	desc := NewDescriptor("test").
		Summary("Test decorator").
		ParamString("name", "User name").
		Examples("alice", "bob", "charlie").
		Done().
		Build()

	param := desc.Schema.Parameters["name"]
	expected := []string{"alice", "bob", "charlie"}
	if diff := cmp.Diff(expected, param.Examples); diff != "" {
		t.Errorf("examples mismatch (-want +got):\n%s", diff)
	}
}

// TestParamString_Chaining tests method chaining
func TestParamString_Chaining(t *testing.T) {
	desc := NewDescriptor("test").
		Summary("Test decorator").
		ParamString("email", "Email address").
		Required().
		Pattern(`^[a-z0-9._%+-]+@[a-z0-9.-]+\.[a-z]{2,}$`).
		MinLength(5).
		MaxLength(100).
		Examples("user@example.com").
		Done().
		Build()

	param := desc.Schema.Parameters["email"]
	if !param.Required {
		t.Error("expected parameter to be required")
	}
	if param.Pattern == nil {
		t.Error("expected pattern to be set")
	}
	if param.MinLength == nil || *param.MinLength != 5 {
		t.Error("expected MinLength=5")
	}
	if param.MaxLength == nil || *param.MaxLength != 100 {
		t.Error("expected MaxLength=100")
	}
	if len(param.Examples) != 1 || param.Examples[0] != "user@example.com" {
		t.Error("expected example 'user@example.com'")
	}
}

// TestParamInt_Basic tests basic integer parameter creation
func TestParamInt_Basic(t *testing.T) {
	desc := NewDescriptor("test").
		Summary("Test decorator").
		ParamInt("count", "Number of items").
		Done().
		Build()

	param := desc.Schema.Parameters["count"]
	if param.Type != types.TypeInt {
		t.Errorf("expected type int, got %v", param.Type)
	}
	if param.Description != "Number of items" {
		t.Errorf("expected description 'Number of items', got %q", param.Description)
	}
}

// TestParamInt_MinMax tests integer min/max constraints
func TestParamInt_MinMax(t *testing.T) {
	desc := NewDescriptor("test").
		Summary("Test decorator").
		ParamInt("port", "Port number").
		Min(1).
		Max(65535).
		Done().
		Build()

	param := desc.Schema.Parameters["port"]
	if param.Minimum == nil || *param.Minimum != 1.0 {
		t.Errorf("expected Minimum=1, got %v", param.Minimum)
	}
	if param.Maximum == nil || *param.Maximum != 65535.0 {
		t.Errorf("expected Maximum=65535, got %v", param.Maximum)
	}
}

// TestParamInt_DefaultValue tests integer with default
func TestParamInt_DefaultValue(t *testing.T) {
	desc := NewDescriptor("test").
		Summary("Test decorator").
		ParamInt("retries", "Number of retries").
		Default(3).
		Done().
		Build()

	param := desc.Schema.Parameters["retries"]
	if param.Default != 3 {
		t.Errorf("expected default 3, got %v", param.Default)
	}
	if param.Required {
		t.Error("parameter with default should be optional")
	}
}

// TestParamDuration_Basic tests basic duration parameter creation
func TestParamDuration_Basic(t *testing.T) {
	desc := NewDescriptor("test").
		Summary("Test decorator").
		ParamDuration("timeout", "Timeout duration").
		Done().
		Build()

	param := desc.Schema.Parameters["timeout"]
	if param.Type != types.TypeDuration {
		t.Errorf("expected type duration, got %v", param.Type)
	}
	if param.Description != "Timeout duration" {
		t.Errorf("expected description 'Timeout duration', got %q", param.Description)
	}
}

// TestParamDuration_Default tests duration with default value
func TestParamDuration_Default(t *testing.T) {
	desc := NewDescriptor("test").
		Summary("Test decorator").
		ParamDuration("timeout", "Timeout duration").
		Default("30s").
		Done().
		Build()

	param := desc.Schema.Parameters["timeout"]
	if param.Default != "30s" {
		t.Errorf("expected default '30s', got %v", param.Default)
	}
}

// TestParamBool_Basic tests basic boolean parameter creation
func TestParamBool_Basic(t *testing.T) {
	desc := NewDescriptor("test").
		Summary("Test decorator").
		ParamBool("verbose", "Enable verbose output").
		Done().
		Build()

	param := desc.Schema.Parameters["verbose"]
	if param.Type != types.TypeBool {
		t.Errorf("expected type bool, got %v", param.Type)
	}
	if param.Description != "Enable verbose output" {
		t.Errorf("expected description 'Enable verbose output', got %q", param.Description)
	}
}

// TestParamBool_Default tests boolean with default value
func TestParamBool_Default(t *testing.T) {
	desc := NewDescriptor("test").
		Summary("Test decorator").
		ParamBool("verbose", "Enable verbose output").
		Default(false).
		Done().
		Build()

	param := desc.Schema.Parameters["verbose"]
	if param.Default != false {
		t.Errorf("expected default false, got %v", param.Default)
	}
}

// TestMultipleParams tests multiple parameters in order
func TestMultipleParams(t *testing.T) {
	desc := NewDescriptor("test").
		Summary("Test decorator").
		ParamString("name", "Name").Required().Done().
		ParamInt("age", "Age").Min(0).Max(150).Done().
		ParamBool("active", "Is active").Default(true).Done().
		Build()

	if len(desc.Schema.Parameters) != 3 {
		t.Errorf("expected 3 parameters, got %d", len(desc.Schema.Parameters))
	}

	// Check parameter order
	expectedOrder := []string{"name", "age", "active"}
	if diff := cmp.Diff(expectedOrder, desc.Schema.ParameterOrder); diff != "" {
		t.Errorf("parameter order mismatch (-want +got):\n%s", diff)
	}
}

// TestPrimaryParam_WithBuilder tests primary parameter with builder
func TestPrimaryParam_WithBuilder(t *testing.T) {
	desc := NewDescriptor("env").
		Summary("Get environment variable").
		PrimaryParamString("name", "Variable name").
		Pattern(`^[A-Z_][A-Z0-9_]*$`).
		Examples("PATH", "HOME").
		Done().
		Build()

	if desc.Schema.PrimaryParameter != "name" {
		t.Errorf("expected primary parameter 'name', got %q", desc.Schema.PrimaryParameter)
	}

	param := desc.Schema.Parameters["name"]
	if !param.Required {
		t.Error("primary parameter should be required")
	}
	if param.Pattern == nil {
		t.Error("expected pattern to be set")
	}

	// Primary parameter should be first in order
	if len(desc.Schema.ParameterOrder) == 0 || desc.Schema.ParameterOrder[0] != "name" {
		t.Error("primary parameter should be first in order")
	}
}

// TestGuardrails_RequiredAndDefault tests that required+default is caught
func TestGuardrails_RequiredAndDefault(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("expected panic for required parameter with default value")
		}
	}()

	NewDescriptor("test").
		Summary("Test decorator").
		ParamString("name", "Name").
		Required().
		Default("value").
		Done().
		Build()
}

// TestGuardrails_InvalidPattern tests that invalid regex pattern is caught
func TestGuardrails_InvalidPattern(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("expected panic for invalid regex pattern")
		}
	}()

	NewDescriptor("test").
		Summary("Test decorator").
		ParamString("name", "Name").
		Pattern(`[invalid(`). // Invalid regex
		Done().
		Build()
}

// TestGuardrails_MinGreaterThanMax tests that min > max is caught
func TestGuardrails_MinGreaterThanMax(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("expected panic for min > max")
		}
	}()

	NewDescriptor("test").
		Summary("Test decorator").
		ParamInt("count", "Count").
		Min(100).
		Max(10). // min > max
		Done().
		Build()
}

// TestGuardrails_DuplicatePrimaryParam tests that duplicate primary parameter is caught
func TestGuardrails_DuplicatePrimaryParam(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("expected panic for duplicate primary parameter")
		}
	}()

	NewDescriptor("test").
		Summary("Test decorator").
		PrimaryParamString("first", "First").Done().
		PrimaryParamString("second", "Second").Done(). // Duplicate primary
		Build()
}

// TestBackwardCompatibility_OldParamMethod tests that old Param() method still works
func TestBackwardCompatibility_OldParamMethod(t *testing.T) {
	desc := NewDescriptor("test").
		Summary("Test decorator").
		Param("name", types.TypeString, "Name", "example1", "example2").
		Build()

	param := desc.Schema.Parameters["name"]
	if param.Type != types.TypeString {
		t.Errorf("expected type string, got %v", param.Type)
	}
	if param.Description != "Name" {
		t.Errorf("expected description 'Name', got %q", param.Description)
	}
	if len(param.Examples) != 2 {
		t.Errorf("expected 2 examples, got %d", len(param.Examples))
	}
}

// TestBackwardCompatibility_OldPrimaryParamMethod tests that old PrimaryParam() method still works
func TestBackwardCompatibility_OldPrimaryParamMethod(t *testing.T) {
	desc := NewDescriptor("test").
		Summary("Test decorator").
		PrimaryParam("name", types.TypeString, "Name", "example").
		Build()

	if desc.Schema.PrimaryParameter != "name" {
		t.Errorf("expected primary parameter 'name', got %q", desc.Schema.PrimaryParameter)
	}

	param := desc.Schema.Parameters["name"]
	if !param.Required {
		t.Error("primary parameter should be required")
	}
}
