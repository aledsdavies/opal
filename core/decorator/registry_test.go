package decorator

import (
	"io"
	"testing"
)

// TestAutoInference verifies roles are auto-inferred from interfaces
func TestAutoInference(t *testing.T) {
	r := NewRegistry()

	// Register a value decorator
	varDec := &mockValueDecorator{path: "var"}
	err := r.register("var", varDec)
	if err != nil {
		t.Fatalf("register failed: %v", err)
	}

	// Lookup and verify role was inferred
	entry, ok := r.Lookup("var")
	if !ok {
		t.Fatal("decorator not found")
	}

	if len(entry.Roles) != 1 {
		t.Fatalf("expected 1 role, got %d", len(entry.Roles))
	}

	if entry.Roles[0] != RoleProvider {
		t.Errorf("expected RoleProvider, got %v", entry.Roles[0])
	}
}

// TestMultiRoleInference verifies multi-role decorators
func TestMultiRoleInference(t *testing.T) {
	r := NewRegistry()

	// Register a decorator that implements both Value and Endpoint
	s3Dec := &mockMultiRoleDecorator{path: "aws.s3.object"}
	err := r.register("aws.s3.object", s3Dec)
	if err != nil {
		t.Fatalf("register failed: %v", err)
	}

	// Lookup and verify both roles were inferred
	entry, ok := r.Lookup("aws.s3.object")
	if !ok {
		t.Fatal("decorator not found")
	}

	if len(entry.Roles) != 2 {
		t.Fatalf("expected 2 roles, got %d", len(entry.Roles))
	}

	hasProvider := false
	hasEndpoint := false
	for _, role := range entry.Roles {
		if role == RoleProvider {
			hasProvider = true
		}
		if role == RoleEndpoint {
			hasEndpoint = true
		}
	}

	if !hasProvider {
		t.Error("missing RoleProvider")
	}
	if !hasEndpoint {
		t.Error("missing RoleEndpoint")
	}
}

// TestGlobalRegistration verifies database/sql pattern
func TestGlobalRegistration(t *testing.T) {
	// Simulate init() registration
	varDec := &mockValueDecorator{path: "test.var"}
	err := Register("test.var", varDec)
	if err != nil {
		t.Fatalf("Register failed: %v", err)
	}

	// Lookup from global registry
	entry, ok := Global().Lookup("test.var")
	if !ok {
		t.Fatal("decorator not found in global registry")
	}

	if entry.Impl.Descriptor().Path != "test.var" {
		t.Errorf("path: got %q, want %q", entry.Impl.Descriptor().Path, "test.var")
	}
}

// TestURIBasedLookup verifies path-based lookup
func TestURIBasedLookup(t *testing.T) {
	r := NewRegistry()

	// Register hierarchical paths
	r.register("env", &mockValueDecorator{path: "env"})
	r.register("aws.secret", &mockValueDecorator{path: "aws.secret"})
	r.register("aws.s3.object", &mockMultiRoleDecorator{path: "aws.s3.object"})

	tests := []struct {
		path      string
		wantFound bool
	}{
		{"env", true},
		{"aws.secret", true},
		{"aws.s3.object", true},
		{"aws", false},    // Partial path doesn't match
		{"aws.s3", false}, // Partial path doesn't match
		{"nonexistent", false},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			_, found := r.Lookup(tt.path)
			if found != tt.wantFound {
				t.Errorf("Lookup(%q): found=%v, want=%v", tt.path, found, tt.wantFound)
			}
		})
	}
}

// Mock decorators for testing

type mockValueDecorator struct {
	path string
}

func (m *mockValueDecorator) Descriptor() Descriptor {
	return Descriptor{Path: m.path}
}

func (m *mockValueDecorator) Resolve(ctx ValueEvalContext, call ValueCall) (any, error) {
	return "mock-value", nil
}

type mockMultiRoleDecorator struct {
	path string
}

func (m *mockMultiRoleDecorator) Descriptor() Descriptor {
	return Descriptor{Path: m.path}
}

func (m *mockMultiRoleDecorator) Resolve(ctx ValueEvalContext, call ValueCall) (any, error) {
	return map[string]any{"size": 1024}, nil
}

func (m *mockMultiRoleDecorator) Open(ctx ExecContext, mode IOType) (io.ReadWriteCloser, error) {
	return nil, nil // Stub
}
