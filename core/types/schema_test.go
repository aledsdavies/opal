package types

import "testing"

func TestSchemaBuilderBasic(t *testing.T) {
	schema := NewSchema("env", KindValue).
		Description("Access environment variables").
		PrimaryParam("property", TypeString, "Env var name").
		Build()

	if schema.Path != "env" {
		t.Errorf("expected path 'env', got %q", schema.Path)
	}
	if schema.Kind != "value" {
		t.Errorf("expected kind 'value', got %q", schema.Kind)
	}
	if schema.Description != "Access environment variables" {
		t.Errorf("expected description, got %q", schema.Description)
	}
	if schema.PrimaryParameter != "property" {
		t.Errorf("expected primary parameter 'property', got %q", schema.PrimaryParameter)
	}
	if len(schema.Parameters) != 1 {
		t.Errorf("expected 1 parameter, got %d", len(schema.Parameters))
	}

	// Check primary param exists in parameters
	param, exists := schema.Parameters["property"]
	if !exists {
		t.Fatal("primary parameter 'property' not found in parameters map")
	}
	if param.Type != TypeString {
		t.Errorf("expected type 'string', got %q", param.Type)
	}
	if !param.Required {
		t.Error("primary parameter should be required")
	}
}

func TestSchemaWithMultipleParams(t *testing.T) {
	schema := NewSchema("retry", KindExecution).
		Description("Retry with backoff").
		Param("times", TypeInt).
		Description("Number of retries").
		Default(3).
		Done().
		Param("delay", TypeDuration).
		Description("Delay between retries").
		Default("1s").
		Examples("1s", "5s", "30s").
		Done().
		AcceptsBlock().
		Build()

	if schema.Path != "retry" {
		t.Errorf("expected path 'retry', got %q", schema.Path)
	}
	if !schema.AcceptsBlock {
		t.Error("expected AcceptsBlock to be true")
	}
	if len(schema.Parameters) != 2 {
		t.Errorf("expected 2 parameters, got %d", len(schema.Parameters))
	}

	// Check times parameter
	times, exists := schema.Parameters["times"]
	if !exists {
		t.Fatal("parameter 'times' not found")
	}
	if times.Type != TypeInt {
		t.Errorf("expected type 'int', got %q", times.Type)
	}
	if times.Default != 3 {
		t.Errorf("expected default 3, got %v", times.Default)
	}
	if times.Required {
		t.Error("parameter with default should not be required")
	}

	// Check delay parameter
	delay, exists := schema.Parameters["delay"]
	if !exists {
		t.Fatal("parameter 'delay' not found")
	}
	if len(delay.Examples) != 3 {
		t.Errorf("expected 3 examples, got %d", len(delay.Examples))
	}
}

func TestSchemaWithReturns(t *testing.T) {
	schema := NewSchema("env", KindValue).
		PrimaryParam("property", TypeString, "Env var name").
		Returns(TypeString, "Environment variable value").
		Build()

	if schema.Returns == nil {
		t.Fatal("expected Returns to be set")
	}
	if schema.Returns.Type != "string" {
		t.Errorf("expected return type 'string', got %q", schema.Returns.Type)
	}
	if schema.Returns.Description != "Environment variable value" {
		t.Errorf("unexpected return description: %q", schema.Returns.Description)
	}
}

func TestValidateSchemaSuccess(t *testing.T) {
	schema := NewSchema("test", KindValue).
		PrimaryParam("prop", TypeString, "Test property").
		Build()

	err := ValidateSchema(schema)
	if err != nil {
		t.Errorf("expected valid schema, got error: %v", err)
	}
}

func TestValidateSchemaEmptyPath(t *testing.T) {
	schema := DecoratorSchema{
		Path: "",
		Kind: "value",
	}

	err := ValidateSchema(schema)
	if err == nil {
		t.Error("expected error for empty path")
	}
}

func TestValidateSchemaInvalidKind(t *testing.T) {
	schema := DecoratorSchema{
		Path: "test",
		Kind: "invalid",
	}

	err := ValidateSchema(schema)
	if err == nil {
		t.Error("expected error for invalid kind")
	}
}

func TestValidateSchemaPrimaryNotInParams(t *testing.T) {
	schema := DecoratorSchema{
		Path:             "test",
		Kind:             KindValue,
		PrimaryParameter: "missing",
		Parameters:       make(map[string]ParamSchema),
	}

	err := ValidateSchema(schema)
	if err == nil {
		t.Error("expected error for primary parameter not in parameters map")
	}
}

func TestRegisterWithSchema(t *testing.T) {
	r := NewRegistry()

	schema := NewSchema("test", "value").
		PrimaryParam("prop", "string", "Test property").
		Param("default", "string").
		Optional().
		Done().
		Build()

	handler := func(ctx Context, args Args) (Value, error) {
		return "test", nil
	}

	err := r.RegisterValueWithSchema(schema, handler)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Retrieve schema
	retrieved, exists := r.GetSchema("test")
	if !exists {
		t.Fatal("schema not found after registration")
	}
	if retrieved.PrimaryParameter != "prop" {
		t.Errorf("expected primary parameter 'prop', got %q", retrieved.PrimaryParameter)
	}
	if len(retrieved.Parameters) != 2 {
		t.Errorf("expected 2 parameters, got %d", len(retrieved.Parameters))
	}
}

func TestRegisterWithSchemaWrongKind(t *testing.T) {
	r := NewRegistry()

	schema := NewSchema("test", KindExecution). // Wrong kind
							Build()

	handler := func(ctx Context, args Args) (Value, error) {
		return nil, nil
	}

	err := r.RegisterValueWithSchema(schema, handler)
	if err == nil {
		t.Error("expected error when registering execution schema with RegisterValueWithSchema")
	}
}
