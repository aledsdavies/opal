package parser

import (
	"strings"
	"testing"
)

// TestSchemaValidation_IntegerRange tests integer min/max validation
func TestSchemaValidation_IntegerRange(t *testing.T) {
	// @retry has times parameter with min=1, max=100
	// Test value above max
	source := `@retry(times=200) { echo "test" }`
	tree := Parse([]byte(source))

	if len(tree.Errors) == 0 {
		t.Fatal("expected validation error for value above max")
	}

	err := tree.Errors[0]
	if !strings.Contains(err.Message, "invalid value") {
		t.Errorf("unexpected message: %s", err.Message)
	}
	if !strings.Contains(err.Suggestion, "between") || !strings.Contains(err.Suggestion, "100") {
		t.Errorf("expected range suggestion, got: %s", err.Suggestion)
	}
}

// TestSchemaValidation_ValidInteger tests that valid integers pass
func TestSchemaValidation_ValidInteger(t *testing.T) {
	source := `@retry(times=3) { echo "test" }`
	tree := Parse([]byte(source))

	if len(tree.Errors) > 0 {
		t.Errorf("unexpected errors for valid value: %v", tree.Errors)
	}
}

// TestSchemaValidation_IntegerMax tests integer maximum validation
func TestSchemaValidation_IntegerMax(t *testing.T) {
	// @retry has times parameter with max=100
	source := `@retry(times=150) { echo "test" }`
	tree := Parse([]byte(source))

	if len(tree.Errors) == 0 {
		t.Fatal("expected validation error for value above maximum")
	}

	err := tree.Errors[0]
	if !strings.Contains(err.Message, "invalid value") {
		t.Errorf("unexpected message: %s", err.Message)
	}
}

// TestSchemaValidation_EnumValues tests enum validation
func TestSchemaValidation_EnumValues(t *testing.T) {
	tests := []struct {
		name      string
		source    string
		wantError bool
	}{
		{
			name:      "valid_enum",
			source:    `@retry(backoff="exponential") { echo "test" }`,
			wantError: false,
		},
		{
			name:      "invalid_enum",
			source:    `@retry(backoff="invalid") { echo "test" }`,
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tree := Parse([]byte(tt.source))
			hasError := len(tree.Errors) > 0

			if hasError != tt.wantError {
				t.Errorf("wantError=%v, got errors=%v", tt.wantError, tree.Errors)
			}
		})
	}
}

// TestSchemaValidation_ObjectLiteral tests object literal validation
func TestSchemaValidation_ObjectLiteral(t *testing.T) {
	tests := []struct {
		name      string
		source    string
		wantError bool
		errorMsg  string
	}{
		{
			name:      "valid_object",
			source:    `@config.myconfig(settings={timeout: "5m"})`,
			wantError: false,
		},
		{
			name:      "object_with_multiple_fields",
			source:    `@config.myconfig(settings={timeout: "5m", retries: 3})`,
			wantError: false,
		},
		{
			name:      "empty_object",
			source:    `@config.myconfig(settings={})`,
			wantError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tree := Parse([]byte(tt.source))
			hasError := len(tree.Errors) > 0

			if hasError != tt.wantError {
				t.Errorf("wantError=%v, got errors=%v", tt.wantError, tree.Errors)
				for _, err := range tree.Errors {
					t.Logf("  Error: %s", err.Message)
				}
			}

			if tt.wantError && len(tree.Errors) > 0 {
				found := false
				for _, err := range tree.Errors {
					if strings.Contains(err.Message, tt.errorMsg) {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("expected error containing %q, got: %v", tt.errorMsg, tree.Errors)
				}
			}
		})
	}
}

// TestSchemaValidation_ArrayLiteral tests array literal validation
func TestSchemaValidation_ArrayLiteral(t *testing.T) {
	tests := []struct {
		name      string
		source    string
		wantError bool
		errorMsg  string
	}{
		{
			name:      "valid_array",
			source:    `@deploy.production(hosts=["web1", "web2"])`,
			wantError: false,
		},
		{
			name:      "array_of_integers",
			source:    `@deploy.staging(hosts=[8080, 8081])`,
			wantError: false,
		},
		{
			name:      "empty_array",
			source:    `@deploy.test(hosts=[])`,
			wantError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tree := Parse([]byte(tt.source))
			hasError := len(tree.Errors) > 0

			if hasError != tt.wantError {
				t.Errorf("wantError=%v, got errors=%v", tt.wantError, tree.Errors)
				for _, err := range tree.Errors {
					t.Logf("  Error: %s", err.Message)
				}
			}

			if tt.wantError && len(tree.Errors) > 0 {
				found := false
				for _, err := range tree.Errors {
					if strings.Contains(err.Message, tt.errorMsg) {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("expected error containing %q, got: %v", tt.errorMsg, tree.Errors)
				}
			}
		})
	}
}
