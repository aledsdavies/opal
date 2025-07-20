package engine

import (
	"context"
	"strings"
	"testing"

	"github.com/aledsdavies/devcmd/pkgs/decorators"
	"github.com/aledsdavies/devcmd/pkgs/parser"
)

// TestImportManagement_BasicDecorators tests import collection for all decorator types
func TestImportManagement_BasicDecorators(t *testing.T) {
	tests := []struct {
		name            string
		input           string
		expectedImports []string
		expectedModules map[string]string
	}{
		{
			name: "function decorators",
			input: `var USER = "admin"
test: echo "@var(USER) lives in @env(HOME)"`,
			expectedImports: []string{"context", "fmt", "os"},
			expectedModules: map[string]string{},
		},
		{
			name:            "timeout decorator",
			input:           `test: @timeout(30s) { echo "hello" }`,
			expectedImports: []string{"context", "fmt", "os", "time"},
			expectedModules: map[string]string{},
		},
		{
			name:            "parallel decorator",
			input:           `test: @parallel(concurrency=2) { echo "task1"; echo "task2" }`,
			expectedImports: []string{"context", "fmt", "os", "sync", "strings"},
			expectedModules: map[string]string{},
		},
		{
			name:            "retry decorator",
			input:           `test: @retry(attempts=3, delay=1s) { echo "might fail" }`,
			expectedImports: []string{"context", "fmt", "os", "time"},
			expectedModules: map[string]string{},
		},
		{
			name: "when pattern decorator",
			input: `test: @when(ENV) {
  prod: echo "production"
  dev: echo "development"
}`,
			expectedImports: []string{"context", "fmt", "os"},
			expectedModules: map[string]string{},
		},
		{
			name: "try pattern decorator",
			input: `test: @try {
  main: echo "try this"
  error: echo "fallback"
}`,
			expectedImports: []string{"context", "fmt", "os"},
			expectedModules: map[string]string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			program, err := parser.Parse(strings.NewReader(tt.input))
			if err != nil {
				t.Fatalf("Failed to parse program: %v", err)
			}

			ctx := decorators.NewExecutionContext(context.Background(), program)
			engine := New(GeneratorMode, ctx)

			result, err := engine.Execute(program)
			if err != nil {
				t.Fatalf("Failed to generate code: %v", err)
			}

			genResult, ok := result.(*GenerationResult)
			if !ok {
				t.Fatalf("Expected GenerationResult, got %T", result)
			}

			// Check all expected imports are present
			for _, expectedImport := range tt.expectedImports {
				if !genResult.HasStandardImport(expectedImport) {
					t.Errorf("Expected standard import %q to be collected", expectedImport)
				}
			}

			// Check expected modules are present
			for module, version := range tt.expectedModules {
				if !genResult.HasGoModule(module) {
					t.Errorf("Expected module %q to be collected", module)
				} else if genResult.GoModules[module] != version {
					t.Errorf("Expected module %q version %q, got %q", module, version, genResult.GoModules[module])
				}
			}

			// Verify generated code contains imports
			code := genResult.String()
			for _, expectedImport := range tt.expectedImports {
				expectedImportLine := `"` + expectedImport + `"`
				if !strings.Contains(code, expectedImportLine) {
					t.Errorf("Generated code missing import %q", expectedImportLine)
				}
			}
		})
	}
}

// TestImportManagement_NestedDecorators tests import collection for complex nested scenarios
func TestImportManagement_NestedDecorators(t *testing.T) {
	input := `var USER = "admin"
var TIMEOUT = 30s

deploy: {
  @timeout(30s) {
    @parallel(concurrency=2, failOnFirstError=true) {
      @retry(attempts=3, delay=1s) {
        echo "Deploying as @var(USER) to @env(HOME)"
      }
      echo "Second task"  
    }
  }
}

test: echo "Simple test with @var(USER)"`

	program, err := parser.Parse(strings.NewReader(input))
	if err != nil {
		t.Fatalf("Failed to parse program: %v", err)
	}

	ctx := decorators.NewExecutionContext(context.Background(), program)
	engine := New(GeneratorMode, ctx)

	result, err := engine.Execute(program)
	if err != nil {
		t.Fatalf("Failed to generate code: %v", err)
	}

	genResult, ok := result.(*GenerationResult)
	if !ok {
		t.Fatalf("Expected GenerationResult, got %T", result)
	}

	// All decorators are used, so we should have all their imports
	expectedImports := []string{
		"context", "fmt", "os", // base + var/env
		"time",            // timeout + retry
		"sync", "strings", // parallel
	}

	for _, expectedImport := range expectedImports {
		if !genResult.HasStandardImport(expectedImport) {
			t.Errorf("Expected standard import %q to be collected from nested decorators", expectedImport)
		}
	}

	// Verify go.mod structure
	goMod := genResult.GoModString()
	if !strings.Contains(goMod, "module devcmd-generated") {
		t.Error("go.mod should contain module declaration")
	}
	if !strings.Contains(goMod, "go 1.24") {
		t.Error("go.mod should contain Go version")
	}
}

// TestImportManagement_NoDuplicates tests that imports are properly deduplicated
func TestImportManagement_NoDuplicates(t *testing.T) {
	input := `test1: @timeout(10s) { echo "first" }
test2: @timeout(20s) { echo "second" } 
test3: @timeout(30s) { echo "third" }`

	program, err := parser.Parse(strings.NewReader(input))
	if err != nil {
		t.Fatalf("Failed to parse program: %v", err)
	}

	ctx := decorators.NewExecutionContext(context.Background(), program)
	engine := New(GeneratorMode, ctx)

	result, err := engine.Execute(program)
	if err != nil {
		t.Fatalf("Failed to generate code: %v", err)
	}

	genResult, ok := result.(*GenerationResult)
	if !ok {
		t.Fatalf("Expected GenerationResult, got %T", result)
	}

	code := genResult.String()

	// Count occurrences of import declarations - should only appear once each
	timeImportCount := strings.Count(code, `"time"`)
	if timeImportCount != 1 {
		t.Errorf("Expected 'time' import to appear exactly once, got %d", timeImportCount)
	}

	contextImportCount := strings.Count(code, `"context"`)
	if contextImportCount != 1 {
		t.Errorf("Expected 'context' import to appear exactly once, got %d", contextImportCount)
	}

	// Verify import section structure
	if !strings.Contains(code, "import (") {
		t.Error("Generated code should have proper import section")
	}
}

// TestImportManagement_EmptyProgram tests that base imports are still included for empty programs
func TestImportManagement_EmptyProgram(t *testing.T) {
	input := `var PORT = 8080`

	program, err := parser.Parse(strings.NewReader(input))
	if err != nil {
		t.Fatalf("Failed to parse program: %v", err)
	}

	ctx := decorators.NewExecutionContext(context.Background(), program)
	engine := New(GeneratorMode, ctx)

	result, err := engine.Execute(program)
	if err != nil {
		t.Fatalf("Failed to generate code: %v", err)
	}

	genResult, ok := result.(*GenerationResult)
	if !ok {
		t.Fatalf("Expected GenerationResult, got %T", result)
	}

	// Even with no decorators, base imports should be present
	baseImports := []string{"context", "fmt", "os"}
	for _, baseImport := range baseImports {
		if !genResult.HasStandardImport(baseImport) {
			t.Errorf("Expected base import %q even in empty program", baseImport)
		}
	}
}

// TestImportManagement_ThirdPartyModules tests handling of third-party dependencies
func TestImportManagement_ThirdPartyModules(t *testing.T) {
	// This test would be for future decorators that require third-party deps
	// For now, just test the structure is in place

	program, err := parser.Parse(strings.NewReader(`test: echo "hello"`))
	if err != nil {
		t.Fatalf("Failed to parse program: %v", err)
	}

	ctx := decorators.NewExecutionContext(context.Background(), program)
	engine := New(GeneratorMode, ctx)

	result, err := engine.Execute(program)
	if err != nil {
		t.Fatalf("Failed to generate code: %v", err)
	}

	genResult, ok := result.(*GenerationResult)
	if !ok {
		t.Fatalf("Expected GenerationResult, got %T", result)
	}

	// Test that the structures exist and work
	genResult.AddThirdPartyImport("github.com/pkg/errors")
	genResult.AddGoModule("github.com/pkg/errors", "v0.9.1")

	if !genResult.HasThirdPartyImport("github.com/pkg/errors") {
		t.Error("Should be able to add and check third-party imports")
	}

	if !genResult.HasGoModule("github.com/pkg/errors") {
		t.Error("Should be able to add and check go.mod dependencies")
	}
}

// TestImportManagement_CustomGoVersion tests go.mod generation with different Go versions
func TestImportManagement_CustomGoVersion(t *testing.T) {
	input := `test: @timeout(10s) { echo "hello" }`

	program, err := parser.Parse(strings.NewReader(input))
	if err != nil {
		t.Fatalf("Failed to parse program: %v", err)
	}

	versions := []string{"1.21", "1.22", "1.23", "1.24"}

	for _, version := range versions {
		t.Run("go_version_"+version, func(t *testing.T) {
			ctx := decorators.NewExecutionContext(context.Background(), program)
			engine := NewWithGoVersion(GeneratorMode, ctx, version)

			result, err := engine.Execute(program)
			if err != nil {
				t.Fatalf("Failed to generate code: %v", err)
			}

			genResult, ok := result.(*GenerationResult)
			if !ok {
				t.Fatalf("Expected GenerationResult, got %T", result)
			}

			goMod := genResult.GoModString()
			expectedGoLine := "go " + version
			if !strings.Contains(goMod, expectedGoLine) {
				t.Errorf("Expected go.mod to contain %q, got:\n%s", expectedGoLine, goMod)
			}
		})
	}
}
